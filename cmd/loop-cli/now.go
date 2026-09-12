package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/SugiKent/loop-cli/internal/classify"
	"github.com/SugiKent/loop-cli/internal/config"
	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
	"github.com/SugiKent/loop-cli/internal/ui"
)

// nowOutput は loop-cli now が標準出力に書く JSON の最上位。
// Items と Errors は空でも [] を出す（design.md D3）。
type nowOutput struct {
	FetchedAt time.Time `json:"fetched_at"`
	Count     int       `json:"count"`
	Items     []nowItem `json:"items"`
	Errors    []string  `json:"errors"`
}

// nowItem は「今やる」の 1 枚のカード。
type nowItem struct {
	Situation string     `json:"situation"`
	Kind      string     `json:"kind"`
	Priority  int        `json:"priority"`
	Summary   string     `json:"summary"`
	Repo      string     `json:"repo"`
	Subject   nowSubject `json:"subject"`
	Issue     *nowIssue  `json:"issue"`
	PRs       []nowPR    `json:"prs"`
}

// nowSubject は局面を出した要素を指す。中身は issue / prs の該当要素から引く（design.md D2）。
type nowSubject struct {
	Type   string `json:"type"`
	Number int    `json:"number"`
}

type nowIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Labels    []string  `json:"labels"`
	UpdatedAt time.Time `json:"updated_at"`
}

// nowPR は 1 本の PR。Undecided から UnresolvedThreads までは詳細の取得に失敗した PR で null になる
// （値が空であることと、取れていないことを区別する。design.md D2b）。
type nowPR struct {
	Number            int        `json:"number"`
	Title             string     `json:"title"`
	URL               string     `json:"url"`
	Labels            []string   `json:"labels"`
	Draft             bool       `json:"draft"`
	UpdatedAt         time.Time  `json:"updated_at"`
	Undecided         *int       `json:"undecided"`
	Mergeable         *string    `json:"mergeable"`
	MergeStateStatus  *string    `json:"merge_state_status"`
	ReviewDecision    *string    `json:"review_decision"`
	ChecksGreen       *bool      `json:"checks_green"`
	Checks            []nowCheck `json:"checks"`
	UnresolvedThreads *int       `json:"unresolved_threads"`
}

type nowCheck struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

// buildNow は取得結果から「今やる」のカードだけを JSON へ写す形に組み立てる。
// 並び順は buildRows と同じ（優先度昇順 → 主体の更新が新しい順 → リポジトリ名昇順 → 番号昇順）。
func buildNow(res *fetch.Result, fetchedAt time.Time) nowOutput {
	type subject struct {
		card      model.Card
		repo      string
		number    int
		isPR      bool
		updatedAt time.Time
	}
	var subjects []subject
	for _, c := range res.Cards {
		if c.Result.Tab != model.TabNow {
			continue
		}
		repo, number, isPR, _, updatedAt := ui.Subject(c)
		subjects = append(subjects, subject{card: c, repo: repo, number: number, isPR: isPR, updatedAt: updatedAt})
	}
	sort.SliceStable(subjects, func(i, j int) bool {
		a, b := subjects[i], subjects[j]
		if a.card.Result.Priority != b.card.Result.Priority {
			return a.card.Result.Priority < b.card.Result.Priority
		}
		if !a.updatedAt.Equal(b.updatedAt) {
			return a.updatedAt.After(b.updatedAt)
		}
		if a.repo != b.repo {
			return a.repo < b.repo
		}
		return a.number < b.number
	})

	items := make([]nowItem, 0, len(subjects))
	for _, s := range subjects {
		items = append(items, nowItem{
			Situation: string(s.card.Result.Situation),
			Kind:      s.card.Result.Situation.Kind(),
			Priority:  s.card.Result.Priority,
			Summary:   s.card.Result.Summary,
			Repo:      s.repo,
			Subject:   nowSubject{Type: subjectType(s.isPR), Number: s.number},
			Issue:     newNowIssue(s.card.Issue),
			PRs:       newNowPRs(s.card.PRs),
		})
	}

	errs := make([]string, 0, len(res.Errors))
	for _, err := range res.Errors {
		errs = append(errs, err.Error())
	}
	return nowOutput{FetchedAt: fetchedAt, Count: len(items), Items: items, Errors: errs}
}

func subjectType(isPR bool) string {
	if isPR {
		return "pr"
	}
	return "issue"
}

func newNowIssue(issue *model.Issue) *nowIssue {
	if issue == nil {
		return nil
	}
	return &nowIssue{
		Number:    issue.Number,
		Title:     issue.Title,
		URL:       issue.URL,
		Labels:    labels(issue.Labels),
		UpdatedAt: issue.UpdatedAt,
	}
}

func newNowPRs(prs []model.PR) []nowPR {
	out := make([]nowPR, 0, len(prs))
	for _, pr := range prs {
		out = append(out, newNowPR(pr))
	}
	return out
}

func newNowPR(pr model.PR) nowPR {
	out := nowPR{
		Number:    pr.Number,
		Title:     pr.Title,
		URL:       pr.URL,
		Labels:    labels(pr.Labels),
		Draft:     pr.IsDraft,
		UpdatedAt: pr.UpdatedAt,
	}
	if n, ok := model.ParseUndecided(pr.Body); ok {
		out.Undecided = &n
	}
	if pr.MergeState != nil {
		ms := pr.MergeState
		green := classify.ChecksGreen(ms)
		out.Mergeable = &ms.Mergeable
		out.MergeStateStatus = &ms.MergeStateStatus
		out.ReviewDecision = &ms.ReviewDecision
		out.ChecksGreen = &green
		out.Checks = newNowChecks(ms.StatusCheckRollup)
	}
	if pr.ReviewThreads != nil {
		unresolved := 0
		for _, th := range pr.ReviewThreads {
			if !th.IsResolved {
				unresolved++
			}
		}
		out.UnresolvedThreads = &unresolved
	}
	return out
}

// newNowChecks は PR 詳細画面（internal/ui の checkLines）と同じ規則で 1 件 1 要素にする。
func newNowChecks(rollup []gh.StatusCheck) []nowCheck {
	out := make([]nowCheck, 0, len(rollup))
	for _, c := range rollup {
		switch c.Typename {
		case "CheckRun":
			state := c.Conclusion
			if state == "" {
				state = c.Status
			}
			out = append(out, nowCheck{Name: c.Name, State: state})
		case "StatusContext":
			out = append(out, nowCheck{Name: c.Context, State: c.State})
		}
	}
	return out
}

// labels は nil のラベルも [] として出す（agent 側に null との分岐を書かせない）。
func labels(ls []string) []string {
	if ls == nil {
		return []string{}
	}
	return ls
}

// nowDeps は runNow が外から受け取る依存（design.md D7）。
// テストはここにスタブを渡し、gh もネットワークも使わずに全経路を通す。
type nowDeps struct {
	configPath func() (string, error)
	check      func(context.Context) error
	fetch      func(ctx context.Context, repos []string, now time.Time, grace time.Duration) (*fetch.Result, error)
}

// runNow は設定のリポジトリを取得し、「今やる」のカードを JSON で標準出力に書く。
// 失敗は標準エラーに書いて 1 を返し、標準出力には何も書かない。
func runNow(ctx context.Context, deps nowDeps, stdout, stderr io.Writer) int {
	path, err := deps.configPath()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	// 端末でない前提で呼ぶので onboarding のフォームには入らない（form は呼ばれない）。
	if err := ensureConfig(path, false, nil); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	cfg, err := config.Load(path)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if err := deps.check(ctx); err != nil {
		_, _ = fmt.Fprintln(stderr, checkError(err))
		return 1
	}

	repos := make([]string, len(cfg.Repos))
	for i, r := range cfg.Repos {
		repos[i] = r.Name
	}
	// 分類の基準時刻と出力の fetched_at は同じ値にする（design.md D4）。
	fetchedAt := time.Now()
	res, err := deps.fetch(ctx, repos, fetchedAt, time.Duration(cfg.OtherGraceMin)*time.Minute)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	// Encode は 2 スペース整形と末尾の改行を出す（design.md D9）。
	// HTML のエスケープは切る。JSON は端末と jq に渡るだけで、< や & を < にすると読めなくなる。
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(buildNow(res, fetchedAt)); err != nil {
		// 途中まで書けている可能性があるので、成功として終わらせない。
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
