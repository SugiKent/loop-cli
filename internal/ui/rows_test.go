package ui

import (
	"testing"
	"time"

	"github.com/SugiKent/sugi-loop/internal/model"
)

func TestSubjectIsPRWhenPRDecidedSituation(t *testing.T) {
	res := exampleResult(t)
	card := cardOf(t, res, 108)

	repo, number, isPR, title, updatedAt := Subject(card)

	if repo != "org/app" || number != 131 || !isPR {
		t.Errorf("主体が PR 131 でない: repo=%q number=%d isPR=%v", repo, number, isPR)
	}
	if title != card.PRs[0].Title {
		t.Errorf("title = %q, want %q", title, card.PRs[0].Title)
	}
	if !updatedAt.Equal(card.PRs[0].UpdatedAt) {
		t.Errorf("updatedAt = %v, want %v", updatedAt, card.PRs[0].UpdatedAt)
	}
}

func TestSubjectIsIssueWhenIssueDecidedSituation(t *testing.T) {
	res := exampleResult(t)

	repo, number, isPR, _, _ := Subject(cardOf(t, res, 140))

	if repo != "org/app" || number != 140 || isPR {
		t.Errorf("主体が issue 140 でない: repo=%q number=%d isPR=%v", repo, number, isPR)
	}
}

func TestSubjectFallsBackToFirstOpenPR(t *testing.T) {
	issueResult := model.Result{Situation: model.SituationInProgress, Priority: 7, Tab: model.TabInProgress, Summary: "#5 は進行中"}
	prResult := model.Result{Situation: model.SituationInProgress, Priority: 7, Tab: model.TabInProgress, Summary: "PR #9 は auto-fix が受け取り中"}
	card := model.Card{
		Issue:  &model.Issue{Repo: "org/app", Number: 5, Result: issueResult},
		PRs:    []model.PR{{Repo: "org/app", Number: 9, Result: prResult}},
		Result: prResult,
	}

	_, number, isPR, _, _ := Subject(card)

	if number != 9 || !isPR {
		t.Errorf("主体が PR 9 でない: number=%d isPR=%v", number, isPR)
	}
}

func TestBuildRowsSplitsExampleIntoTabs(t *testing.T) {
	rows := buildRows(exampleResult(t).Cards)

	want := map[model.Tab]int{model.TabNow: 1, model.TabBacklog: 1, model.TabInProgress: 0, model.TabAbnormal: 0}
	for tab, n := range want {
		if got := len(rows[tab]); got != n {
			t.Errorf("%s タブの行数 = %d, want %d", tab, got, n)
		}
	}
	if rows[model.TabNow][0].number != 131 {
		t.Errorf("今やるタブの行が issue 108 の Card でない: number=%d", rows[model.TabNow][0].number)
	}
	if rows[model.TabBacklog][0].number != 140 {
		t.Errorf("バックログタブの行が issue 140 の Card でない: number=%d", rows[model.TabBacklog][0].number)
	}
}

func TestBuildRowsSortsByPriorityThenNewest(t *testing.T) {
	now := at
	cards := []model.Card{
		nowCard("org/app", 1, 3, now.Add(-5*time.Hour)),
		nowCard("org/app", 2, 3, now.Add(-9*time.Hour)),
		nowCard("org/app", 3, 3, now.Add(-24*time.Hour)),
		nowCard("org/app", 4, 1, now.Add(-1*time.Hour)),
		nowCard("org/app", 5, 1, now.Add(-12*time.Minute)),
	}

	rows := buildRows(cards)[model.TabNow]

	var got []string
	for _, r := range rows {
		got = append(got, Elapsed(now, r.updatedAt))
	}
	want := []string{"12m", "1h", "5h", "9h", "1d"}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("並びが %v, want %v", got, want)
		}
	}
}
