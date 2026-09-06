package model

import (
	"slices"
	"testing"

	"github.com/SugiKent/loop-cli/internal/gh"
)

func TestIssueStagesReturnsStageLabelsInStageOrder(t *testing.T) {
	got := IssueStages([]string{"question", "stage:apply", "blocked", "stage:propose"})
	want := []string{"stage:propose", "stage:apply"}
	if !slices.Equal(got, want) {
		t.Errorf("IssueStages = %v, want %v", got, want)
	}
}

func TestPRStagesReturnsOnlyPRStageLabels(t *testing.T) {
	got := PRStages([]string{"question", "archive", "docs"})
	want := []string{"archive"}
	if !slices.Equal(got, want) {
		t.Errorf("PRStages = %v, want %v", got, want)
	}
}

func TestIssueFromSearch(t *testing.T) {
	is := IssueFromSearch(gh.SearchIssue{
		Repository: gh.Repository{Name: "app", NameWithOwner: "org/app"},
		Number:     108,
		Title:      "ログインのセッション仕様を決める",
		Labels:     []gh.Label{{Name: "stage:propose"}, {Name: "question"}},
		URL:        "https://github.com/org/app/issues/108",
	})

	if is.Repo != "org/app" || is.Number != 108 {
		t.Errorf("Repo/Number = %q/%d, want org/app/108", is.Repo, is.Number)
	}
	if !slices.Equal(is.Labels, []string{"stage:propose", "question"}) {
		t.Errorf("Labels = %v", is.Labels)
	}
	if is.Comments != nil {
		t.Errorf("Comments = %v, want nil（詳細は未取得）", is.Comments)
	}
}

func TestPRFromSearchIsOpenWithoutDetails(t *testing.T) {
	pr := PRFromSearch(gh.SearchPR{
		Repository: gh.Repository{NameWithOwner: "org/app"},
		Number:     131,
		Labels:     []gh.Label{{Name: "propose"}},
		IsDraft:    true,
	})

	if pr.State != "OPEN" {
		t.Errorf("State = %q, want OPEN", pr.State)
	}
	if !pr.IsDraft {
		t.Error("IsDraft = false, want true")
	}
	if pr.Comments != nil || pr.MergeState != nil || pr.ReviewThreads != nil {
		t.Error("詳細は nil のままであるべき")
	}
}

func TestCommentFromMarksAIByBody(t *testing.T) {
	c := CommentFrom(gh.Comment{
		Author: gh.Author{Login: "user-1"},
		Body:   "<!-- routine -->\nQ1: セッションの寿命は何日にしますか。",
	})

	if c.Author != "user-1" {
		t.Errorf("Author = %q", c.Author)
	}
	if !c.AI {
		t.Error("AI = false, want true（本文がマーカーで始まる）")
	}
}

func TestSituationPriorityTabKind(t *testing.T) {
	tests := []struct {
		situation Situation
		priority  int
		tab       Tab
		kind      string
	}{
		{SituationF, 0, TabAbnormal, "異常"},
		{SituationA, 1, TabNow, "質問"},
		{SituationD, 1, TabNow, "質問"},
		{SituationB, 2, TabNow, "方針"},
		{SituationC, 3, TabNow, "merge"},
		{SituationE, 4, TabBacklog, "todo 候補"},
		{SituationG, 5, TabNow, "merge"},
		{SituationOther, 6, TabNow, "その他"},
		{SituationInProgress, 7, TabInProgress, "進行中"},
		{Situation(""), 8, Tab(""), ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.situation), func(t *testing.T) {
			if got := tt.situation.Priority(); got != tt.priority {
				t.Errorf("Priority() = %d, want %d", got, tt.priority)
			}
			if got := tt.situation.Tab(); got != tt.tab {
				t.Errorf("Tab() = %q, want %q", got, tt.tab)
			}
			if got := tt.situation.Kind(); got != tt.kind {
				t.Errorf("Kind() = %q, want %q", got, tt.kind)
			}
		})
	}
}
