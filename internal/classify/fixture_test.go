package classify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// fixtureExpectation は 1 つの fixture の運用方式と、その全 issue / PR の期待値。
type fixtureExpectation struct {
	mode       model.Mode
	situations map[string]model.Situation
}

// expected は fixture ごとの期待値表（V-1）。
// 値は human-turn-signals.md の判定表を fixture の JSON に手で当てて書く（分類器の出力を写さない）。
// s04 で採取した <alias> を足したら、その alias の全 issue / PR をここに書くまでテストは通らない。
var expected = map[string]fixtureExpectation{
	"example": {
		mode: model.ModeSDD,
		situations: map[string]model.Situation{
			// stage:propose + question（blocked 無し）で最新コメントが人 → 進行中の規則 4。
			"issue-108": model.SituationInProgress,
			// ラベル無し・blocked 無し → 行 E。
			"issue-140": model.SituationE,
			// propose + question で唯一のコメントが <!-- routine --> 始まり → 行 A。
			"pr-131": model.SituationA,
		},
	},
	"board": {
		mode: model.ModeLabel,
		situations: map[string]model.Situation{
			// In Progress の 1 件だけで question 無し → 進行中の規則 1。
			"issue-61": model.SituationInProgress,
			// label の段階ラベルが 1 つも無く blocked も無い → 行 E。
			"issue-62": model.SituationE,
			// To Do と In Progress の 2 件 → 行 F。
			"issue-63": model.SituationF,
			// blocked + question で最新コメントが AI → 行 B。
			"issue-64": model.SituationB,
			// Closes #61 を持ち question 無し・MERGEABLE・checks 緑、未 resolve thread 無し → 行 C。
			"pr-71": model.SituationC,
			// question で最新コメントが AI → 行 A。
			"pr-72": model.SituationA,
		},
	},
}

func TestFixturesMatchExpectedSituations(t *testing.T) {
	root := filepath.Join("..", "gh", "testdata", "fixtures")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("fixtures を読めません: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		alias := e.Name()
		t.Run(alias, func(t *testing.T) {
			exp, ok := expected[alias]
			if !ok {
				t.Fatalf("fixture %q の期待値が expected にありません。判定表を当てて追加してください", alias)
			}
			want := exp.situations
			got := classifyFixture(t, filepath.Join(root, alias), exp.mode)

			for key, situation := range got {
				w, ok := want[key]
				if !ok {
					t.Errorf("%s/%s の期待値が expected にありません（分類結果は %q）", alias, key, situation)
					continue
				}
				if situation != w {
					t.Errorf("%s/%s = %q, want %q", alias, key, situation, w)
				}
			}
			for key := range want {
				if _, ok := got[key]; !ok {
					t.Errorf("%s/%s が expected にありますが fixture にありません", alias, key)
				}
			}
		})
	}
}

// classifyFixture は fixture の全 issue / PR に詳細を入れて分類し、issue-<n> / pr-<n> → 局面を返す。
func classifyFixture(t *testing.T, dir string, mode model.Mode) map[string]model.Situation {
	t.Helper()
	ctx := context.Background()
	f := gh.NewFake(dir)
	out := map[string]model.Situation{}

	issues, err := f.SearchIssues(ctx, nil)
	if err != nil {
		t.Fatalf("SearchIssues: %v", err)
	}
	for _, si := range issues {
		is := model.IssueFromSearch(si)
		detail, err := f.ViewIssue(ctx, is.Repo, is.Number)
		if err != nil {
			t.Fatalf("ViewIssue(%d): %v", is.Number, err)
		}
		is.Comments = comments(detail.Comments)
		out[fmt.Sprintf("issue-%d", is.Number)] = Issue(is, mode).Situation
	}

	prs, err := f.SearchPRs(ctx, nil)
	if err != nil {
		t.Fatalf("SearchPRs: %v", err)
	}
	for _, sp := range prs {
		pr := model.PRFromSearch(sp)
		detail, err := f.ViewPR(ctx, pr.Repo, pr.Number)
		if err != nil {
			t.Fatalf("ViewPR(%d): %v", pr.Number, err)
		}
		pr.Comments = comments(detail.Comments)
		if pr.MergeState, err = f.ViewPRMergeState(ctx, pr.Repo, pr.Number); err != nil {
			t.Fatalf("ViewPRMergeState(%d): %v", pr.Number, err)
		}
		if pr.ReviewThreads, err = f.ReviewThreads(ctx, pr.Repo, pr.Number); err != nil {
			t.Fatalf("ReviewThreads(%d): %v", pr.Number, err)
		}
		out[fmt.Sprintf("pr-%d", pr.Number)] = PR(pr, mode, pr.UpdatedAt).Situation
	}
	return out
}

func comments(cs []gh.Comment) []model.Comment {
	out := make([]model.Comment, 0, len(cs))
	for _, c := range cs {
		out = append(out, model.CommentFrom(c))
	}
	return out
}
