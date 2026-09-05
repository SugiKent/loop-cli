package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/SugiKent/sugi-loop/internal/classify"
	"github.com/SugiKent/sugi-loop/internal/gh"
	"github.com/SugiKent/sugi-loop/internal/model"
)

// row は出力の 1 行と、節内の並び順のキー。
type row struct {
	tab      model.Tab
	priority int
	repo     string
	isPR     bool
	number   int
	kind     string
	title    string
	updated  time.Time
}

func classifyFixture(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("classify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	alias := fs.String("fixture", "", "分類する fixture のディレクトリ名")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *alias == "" {
		return fmt.Errorf("--fixture を指定してください")
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("余分な引数があります: %q", fs.Arg(0))
	}
	if _, err := os.Stat(fixturesDir); err != nil {
		return fmt.Errorf("%s がありません。リポジトリのルートで実行してください: %w", fixturesDir, err)
	}
	dir := filepath.Join(fixturesDir, *alias)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("fixture がありません: %w", err)
	}

	rows, err := classifyRows(context.Background(), dir)
	if err != nil {
		return err
	}
	return writeQueue(stdout, rows, time.Now())
}

// classifyRows は fixture の全 open issue / PR を分類して行にする。Card 化はしない（s07 の担当）。
func classifyRows(ctx context.Context, dir string) ([]row, error) {
	f := gh.NewFake(dir)
	var rows []row

	issues, err := f.SearchIssues(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, si := range issues {
		is := model.IssueFromSearch(si)
		detail, err := f.ViewIssue(ctx, is.Repo, is.Number)
		if err != nil {
			return nil, err
		}
		is.Comments = comments(detail.Comments)
		rows = append(rows, newRow(classify.Issue(is), is.Repo, false, is.Number, is.Title, is.UpdatedAt))
	}

	prs, err := f.SearchPRs(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, sp := range prs {
		pr := model.PRFromSearch(sp)
		detail, err := f.ViewPR(ctx, pr.Repo, pr.Number)
		if err != nil {
			return nil, err
		}
		pr.Comments = comments(detail.Comments)
		if pr.MergeState, err = f.ViewPRMergeState(ctx, pr.Repo, pr.Number); err != nil {
			return nil, err
		}
		if pr.ReviewThreads, err = f.ReviewThreads(ctx, pr.Repo, pr.Number); err != nil {
			return nil, err
		}
		rows = append(rows, newRow(classify.PR(pr), pr.Repo, true, pr.Number, pr.Title, pr.UpdatedAt))
	}
	return rows, nil
}

func newRow(r model.Result, repo string, isPR bool, number int, title string, updated time.Time) row {
	return row{
		tab:      r.Tab,
		priority: r.Priority,
		repo:     repo,
		isPR:     isPR,
		number:   number,
		kind:     r.Situation.Kind(),
		title:    title,
		updated:  updated,
	}
}

func comments(cs []gh.Comment) []model.Comment {
	out := make([]model.Comment, 0, len(cs))
	for _, c := range cs {
		out = append(out, model.CommentFrom(c))
	}
	return out
}

var tabs = []struct {
	label string
	tab   model.Tab
}{
	{"[1]今やる", model.TabNow},
	{"[2]バックログ", model.TabBacklog},
	{"[3]進行中", model.TabInProgress},
	{"[4]異常", model.TabAbnormal},
}

// writeQueue は 4 タブ分の節を書く。全件分を組み立ててから 1 度で書く。
func writeQueue(w io.Writer, rows []row, now time.Time) error {
	slices.SortFunc(rows, func(a, b row) int {
		return cmp.Or(
			cmp.Compare(a.priority, b.priority),
			cmp.Compare(a.repo, b.repo),
			cmp.Compare(boolToInt(a.isPR), boolToInt(b.isPR)),
			cmp.Compare(a.number, b.number),
		)
	})

	var sb strings.Builder
	for _, t := range tabs {
		var in []row
		for _, r := range rows {
			if r.tab == t.tab {
				in = append(in, r)
			}
		}
		fmt.Fprintf(&sb, "%s %d\n", t.label, len(in))
		for _, r := range in {
			fmt.Fprintf(&sb, "%d\t%s\t%s\t%s\t%s\t%s\n",
				r.priority, r.kind, r.repo, number(r), r.title, elapsed(now, r.updated))
		}
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

func number(r row) string {
	if r.isPR {
		return fmt.Sprintf("PR%d", r.number)
	}
	return fmt.Sprintf("#%d", r.number)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// elapsed は now と t の差を 1 時間未満 <m>m / 24 時間未満 <h>h / それ以上 <d>d で書く。
func elapsed(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < 0:
		return "0m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
