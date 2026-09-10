package action

import (
	"reflect"
	"strings"
	"testing"

	"github.com/SugiKent/loop-cli/internal/gh"
)

const labelsFixtureDir = "testdata/labels"

// TestSetLabelsWritesOnceForAddAndRemove は付けると外すが 1 回の呼び出しにまとまることを検証する。
func TestSetLabelsWritesOnceForAddAndRemove(t *testing.T) {
	client := gh.NewFake(labelsFixtureDir)
	target := Target{Repo: "org/app", Number: 150}

	// issue 150 は wip が付いている。docs を足して wip を外す。
	add, remove, err := SetLabels(t.Context(), client, target, []string{"docs"})
	if err != nil {
		t.Fatalf("SetLabels: %v", err)
	}

	if !reflect.DeepEqual(add, []string{"docs"}) || !reflect.DeepEqual(remove, []string{"wip"}) {
		t.Errorf("add = %v / remove = %v, want [docs] / [wip]", add, remove)
	}
	want := []gh.Call{
		{Method: "ViewIssue", Repo: "org/app", Number: 150},
		{Method: "EditIssueLabels", Repo: "org/app", Number: 150,
			AddLabels: []string{"docs"}, RemoveLabels: []string{"wip"}},
	}
	if !reflect.DeepEqual(client.Calls, want) {
		t.Errorf("Calls = %+v, want %+v", client.Calls, want)
	}
}

// TestSetLabelsOnPRUsesEditPRLabels は書き先が PR なら pr の経路を使うことを検証する（不変条件 4）。
func TestSetLabelsOnPRUsesEditPRLabels(t *testing.T) {
	client := gh.NewFake("../gh/testdata/fixtures/example")
	target := Target{Repo: "org/app", Number: 131, IsPR: true}

	// PR 131 は propose と question が付いている。docs を足すだけの送信。
	add, remove, err := SetLabels(t.Context(), client, target, []string{"propose", "question", "docs"})
	if err != nil {
		t.Fatalf("SetLabels: %v", err)
	}

	if !reflect.DeepEqual(add, []string{"docs"}) || len(remove) != 0 {
		t.Errorf("add = %v / remove = %v, want [docs] / []", add, remove)
	}
	want := []gh.Call{{Method: "EditPRLabels", Repo: "org/app", Number: 131, AddLabels: []string{"docs"}}}
	if !reflect.DeepEqual(client.Calls, want) {
		t.Errorf("Calls = %+v, want %+v", client.Calls, want)
	}
}

// TestSetLabelsSkipsWriteWhenAlreadyMatching は読み直した現在のラベルと一致していれば
// 書き込まないことを検証する（画面のラベルではなく読み直しが差分の基準）。
func TestSetLabelsSkipsWriteWhenAlreadyMatching(t *testing.T) {
	client := gh.NewFake(labelsFixtureDir)
	target := Target{Repo: "org/app", Number: 151}

	// issue 151 は docs が付いている。送信後の集合も docs だけ。
	add, remove, err := SetLabels(t.Context(), client, target, []string{"docs"})
	if err != nil {
		t.Fatalf("SetLabels: %v", err)
	}

	if len(add) != 0 || len(remove) != 0 {
		t.Errorf("add = %v / remove = %v, want どちらも空", add, remove)
	}
	want := []gh.Call{{Method: "ViewIssue", Repo: "org/app", Number: 151}}
	if !reflect.DeepEqual(client.Calls, want) {
		t.Errorf("Calls = %+v, want ViewIssue だけ: %+v", client.Calls, want)
	}
}

// TestSetLabelsDoesNotRejectStageLabels は段階ラベルを拒否しないことを検証する。
// 人が一覧から明示的に選んだラベルはそのまま書く（ToggleTodo の拒否はここには無い）。
func TestSetLabelsDoesNotRejectStageLabels(t *testing.T) {
	client := gh.NewFake("../gh/testdata/fixtures/example")
	target := Target{Repo: "org/app", Number: 108}

	// issue 108 は stage:propose と question。stage:apply を足すと段階ラベルが 2 つになる。
	add, _, err := SetLabels(t.Context(), client, target,
		[]string{"stage:propose", "question", "stage:apply"})
	if err != nil {
		t.Fatalf("SetLabels: %v", err)
	}

	if !reflect.DeepEqual(add, []string{"stage:apply"}) {
		t.Errorf("add = %v, want [stage:apply]", add)
	}
	if len(client.Calls) != 2 || client.Calls[1].Method != "EditIssueLabels" {
		t.Fatalf("Calls = %+v, want ViewIssue → EditIssueLabels", client.Calls)
	}
}

// TestSetLabelsSortsFlagsByName は gh に渡す並びが名前の昇順になることを検証する
// （引数を決定的にするため）。
func TestSetLabelsSortsFlagsByName(t *testing.T) {
	client := gh.NewFake(labelsFixtureDir)
	target := Target{Repo: "org/app", Number: 150}

	add, remove, err := SetLabels(t.Context(), client, target, []string{"wip", "docs", "apply"})
	if err != nil {
		t.Fatalf("SetLabels: %v", err)
	}

	if !reflect.DeepEqual(add, []string{"apply", "docs"}) {
		t.Errorf("add = %v, want [apply docs]", add)
	}
	if len(remove) != 0 {
		t.Errorf("remove = %v, want 空", remove)
	}
}

// TestSetLabelsReturnsReadError は読み直しの失敗をそのまま返し、書き込みに進まないことを検証する。
func TestSetLabelsReturnsReadError(t *testing.T) {
	client := gh.NewFake(labelsFixtureDir)
	target := Target{Repo: "org/app", Number: 999}

	_, _, err := SetLabels(t.Context(), client, target, []string{"docs"})
	if err == nil {
		t.Fatal("エラーが返らない")
	}
	if !strings.Contains(err.Error(), "issue-999.json") {
		t.Errorf("エラー文字列に読めなかったパスが無い: %v", err)
	}
	for _, c := range client.Calls {
		if strings.HasPrefix(c.Method, "Edit") {
			t.Errorf("読み直しに失敗したのに書き込んでいる: %+v", client.Calls)
		}
	}
}
