package ui

import (
	"testing"
	"time"

	"github.com/SugiKent/loop-cli/internal/model"
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
	rows := buildRows(exampleResult(t).Cards, nil)

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

	rows := buildRows(cards, nil)[model.TabNow]

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

// 進行中タブは優先度が全行同じなので、リポジトリ → 段階 → 新しい順 で並べる。
func TestBuildRowsSortsInProgressByRepoThenStage(t *testing.T) {
	now := at
	cards := []model.Card{
		inProgressCard("org/web", 1, true, []string{"apply"}, now.Add(-1*time.Hour)),
		inProgressCard("org/app", 2, false, []string{"stage:archive"}, now.Add(-48*time.Hour)),
		inProgressCard("org/app", 3, true, []string{"propose"}, now.Add(-3*time.Hour)),
		inProgressCard("org/app", 4, true, []string{"propose"}, now.Add(-12*time.Minute)),
		inProgressCard("org/app", 5, false, nil, now.Add(-5*time.Minute)),
	}

	rows := buildRows(cards, nil)[model.TabInProgress]

	var got [][2]string
	for _, r := range rows {
		got = append(got, [2]string{r.repo, Elapsed(now, r.updatedAt)})
	}
	want := [][2]string{
		{"org/app", "12m"}, // propose
		{"org/app", "3h"},  // propose
		{"org/app", "2d"},  // archive
		{"org/app", "5m"},  // 段階なしは同じリポジトリの最後
		{"org/web", "1h"},  // apply
	}
	if len(got) != len(want) {
		t.Fatalf("行数 = %d, want %d（%v）", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("並びが %v, want %v", got, want)
		}
	}
}

// 種別の列に出す語は、主体の種別と運用方式で決まる（`stage:` は列でだけ落とす）。
func TestRowStageWordByModeAndSubject(t *testing.T) {
	modes := map[string]model.Mode{"org/app": model.ModeSDD, "org/kanban": model.ModeLabel}
	tests := []struct {
		name      string
		repo      string
		isPR      bool
		labels    []string
		stage     string
		stageWord string
	}{
		{"sdd の issue", "org/app", false, []string{"stage:archive", "wip"}, "stage:archive", "archive"},
		{"sdd の PR", "org/app", true, []string{"propose", "question"}, "propose", "propose"},
		{"label の issue", "org/kanban", false, []string{"In Progress"}, "In Progress", "In Progress"},
		{"label の PR", "org/kanban", true, []string{"In Progress"}, "", "-"},
		{"段階なし", "org/app", false, []string{"question"}, "", "-"},
		{"docs だけの PR", "org/app", true, []string{"docs"}, "", "-"},
		{"段階 2 つ", "org/app", false, []string{"stage:propose", "stage:apply", "question"}, "stage:propose", "propose"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := inProgressCard(tt.repo, 1, tt.isPR, tt.labels, at)

			r := newRow(card, modes)

			if r.stage != tt.stage {
				t.Errorf("色を引く段階ラベル名 = %q, want %q", r.stage, tt.stage)
			}
			if r.stageWord != tt.stageWord {
				t.Errorf("列に出す語 = %q, want %q", r.stageWord, tt.stageWord)
			}
		})
	}
}
