package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/sugi-loop/internal/fetch"
	"github.com/SugiKent/sugi-loop/internal/gh"
	"github.com/SugiKent/sugi-loop/internal/model"
)

var tKey = runeKey('t')

// todoModel は t のテスト用の Model。client は Result を作った Fake とは別に渡す。
func todoModel(client gh.GHClient, cards []model.Card, ed *stubEditor) Model {
	m, _ := send(New(nil, client, ed.Editor, Options{}),
		tea.WindowSizeMsg{Width: 120, Height: 40},
		fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})
	return m
}

// exampleTodoModel は example の Card を持ち、切り替え先が example fixture の Model。
func exampleTodoModel(t *testing.T) (Model, *gh.Fake) {
	t.Helper()
	fake := gh.NewFake(fixtureDir)
	return todoModel(fake, exampleResult(t).Cards, &stubEditor{}), fake
}

// todoCard は stage:todo が付いた進行中の issue 150 の Card 1 枚。
func todoCard() model.Card {
	result := model.Result{Situation: model.SituationE, Priority: 4, Tab: model.TabInProgress, Summary: "待つ"}
	issue := &model.Issue{
		Repo: "org/app", Number: 150, Title: "fixture 150",
		Labels: []string{model.LabelStageTodo}, UpdatedAt: at, Result: result,
	}
	return model.Card{Issue: issue, Result: result}
}

// failingAddLabel は AddLabel だけが失敗する client。
type failingAddLabel struct{ *gh.Fake }

func (f failingAddLabel) AddLabel(_ context.Context, _ string, _ int, _ string) error {
	return errors.New("gh issue edit 140 -R org/app --add-label stage:todo: exit 1: HTTP 403")
}

// TestTodoAddsStageTodo はバックログの issue に t で stage:todo が付くことを検証する。
func TestTodoAddsStageTodo(t *testing.T) {
	m, fake := exampleTodoModel(t)
	m, _ = send(m, runeKey('2'))

	m, cmd := send(m, tKey)
	if cmd == nil {
		t.Fatal("t でコマンドが返っていない")
	}
	if text := plainText(m); !strings.Contains(text, "org/app #140 の stage:todo を切り替え中") {
		t.Errorf("切り替え中が出ていない: %q", footerOf(text))
	}

	m, _ = runCmd(t, m, cmd)

	want := []gh.Call{
		{Method: "ViewIssue", Repo: "org/app", Number: 140},
		{Method: "AddLabel", Repo: "org/app", Number: 140, Label: model.LabelStageTodo},
	}
	if len(fake.Calls) != len(want) {
		t.Fatalf("呼び出し = %+v, want %+v", fake.Calls, want)
	}
	for i, w := range want {
		if fake.Calls[i] != w {
			t.Errorf("%d 件目 = %+v, want %+v", i+1, fake.Calls[i], w)
		}
	}
	if m.screen != screenQueue {
		t.Errorf("画面 = %d, want キュー", m.screen)
	}
	if text := plainText(m); !strings.Contains(text, "org/app #140 に stage:todo を付けました") {
		t.Errorf("成功が出ていない: %q", footerOf(text))
	}

	rows := m.rows[model.TabBacklog]
	if len(rows) != 1 || rows[0].number != 140 {
		t.Fatalf("バックログの行 = %+v, want issue 140 の 1 行", rows)
	}
	if labels := rows[0].card.Issue.Labels; len(labels) != 0 {
		t.Errorf("画面の Card のラベルが書き換わっている: %q", labels)
	}
}

// TestTodoTargetsIssueOfPRCard は主体が PR の Card でも対象が Issue になることを検証する。
// issue 108 は stage:propose 付きなので ToggleTodo が拒否し、書き込みは起きない。
func TestTodoTargetsIssueOfPRCard(t *testing.T) {
	m, fake := exampleTodoModel(t)
	before := m.cards

	m, cmd := send(m, tKey)
	m, _ = runCmd(t, m, cmd)

	want := gh.Call{Method: "ViewIssue", Repo: "org/app", Number: 108}
	if len(fake.Calls) != 1 || fake.Calls[0] != want {
		t.Fatalf("呼び出し = %+v, want 1 件の %+v", fake.Calls, want)
	}
	text := plainText(m)
	for _, sub := range []string{"org/app #108 の stage:todo を切り替えられません:", "stage:propose"} {
		if !strings.Contains(text, sub) {
			t.Errorf("拒否に %q が無い: %q", sub, footerOf(text))
		}
	}
	if len(m.cards) != len(before) {
		t.Errorf("Cards が変わっている: %d 件, want %d 件", len(m.cards), len(before))
	}
}

// TestTodoRemovesStageTodo は stage:todo が付いた issue で t を押すと外れることを検証する。
func TestTodoRemovesStageTodo(t *testing.T) {
	fake := gh.NewFake("../action/testdata/todo")
	m := todoModel(fake, []model.Card{todoCard()}, &stubEditor{})
	m, _ = send(m, runeKey('3'))

	m, cmd := send(m, tKey)
	m, _ = runCmd(t, m, cmd)

	want := gh.Call{Method: "RemoveLabel", Repo: "org/app", Number: 150, Label: model.LabelStageTodo}
	if len(fake.Calls) != 2 || fake.Calls[1] != want {
		t.Fatalf("呼び出し = %+v, want 2 件目が %+v", fake.Calls, want)
	}
	if text := plainText(m); !strings.Contains(text, "org/app #150 から stage:todo を外しました") {
		t.Errorf("外した結果が出ていない: %q", footerOf(text))
	}
}

// TestTodoWorksOnCardDetail はカード詳細でも t が効き、画面が変わらないことを検証する。
func TestTodoWorksOnCardDetail(t *testing.T) {
	m, fake := exampleTodoModel(t)
	m, _ = send(m, runeKey('2'), enterKey)

	m, cmd := send(m, tKey)
	m, _ = runCmd(t, m, cmd)

	want := gh.Call{Method: "AddLabel", Repo: "org/app", Number: 140, Label: model.LabelStageTodo}
	if len(fake.Calls) != 2 || fake.Calls[1] != want {
		t.Fatalf("呼び出し = %+v, want 2 件目が %+v", fake.Calls, want)
	}
	if m.screen != screenCard || m.detail.card.Issue.Number != 140 {
		t.Errorf("画面 = %d / 対象 = %+v, want カード詳細 / issue 140", m.screen, m.detail.card.Issue)
	}
}

// TestTodoDoesNothing は対象の無い画面と行で t が何もしないことを検証する。
func TestTodoDoesNothing(t *testing.T) {
	t.Run("PR 単独の Card", func(t *testing.T) {
		pr := prOf(60, "OPEN", []string{model.LabelDocs})
		pr.Result = model.Result{Situation: model.SituationD, Priority: 2, Tab: model.TabNow, Summary: "確認する"}
		fake := gh.NewFake(fixtureDir)
		m := todoModel(fake, []model.Card{{PRs: []model.PR{pr}, Result: pr.Result}}, &stubEditor{})

		got, cmd := send(m, tKey)
		if cmd != nil {
			t.Errorf("コマンドが返った: %T", cmd())
		}
		if len(fake.Calls) != 0 || got.screen != screenQueue {
			t.Errorf("呼び出し = %+v / 画面 = %d", fake.Calls, got.screen)
		}
	})

	t.Run("PR 詳細", func(t *testing.T) {
		m, fake := exampleTodoModel(t)
		m, _ = send(m, enterKey, enterKey)
		if m.screen != screenPR {
			t.Fatalf("画面 = %d, want PR 詳細", m.screen)
		}
		fake.Calls = nil

		if _, cmd := send(m, tKey); cmd != nil {
			t.Errorf("コマンドが返った: %T", cmd())
		}
		if len(fake.Calls) != 0 {
			t.Errorf("呼び出し = %+v, want 空", fake.Calls)
		}
	})

	t.Run("0 行のタブ", func(t *testing.T) {
		m, fake := exampleTodoModel(t)
		m, _ = send(m, runeKey('4'))
		if len(m.rows[m.tab]) != 0 {
			t.Fatalf("異常タブが 0 行でない: %d 行", len(m.rows[m.tab]))
		}
		fake.Calls = nil

		if _, cmd := send(m, tKey); cmd != nil {
			t.Errorf("コマンドが返った: %T", cmd())
		}
		if len(fake.Calls) != 0 {
			t.Errorf("呼び出し = %+v, want 空", fake.Calls)
		}
	})
}

// TestTodoShowsGHFailure は gh の失敗が赤で（拒否と同じ形で）出ることを検証する。
func TestTodoShowsGHFailure(t *testing.T) {
	fake := gh.NewFake(fixtureDir)
	m := todoModel(failingAddLabel{fake}, exampleResult(t).Cards, &stubEditor{})
	m, _ = send(m, runeKey('2'))
	before := m.cards

	m, cmd := send(m, tKey)
	m, _ = runCmd(t, m, cmd)

	text := plainText(m)
	for _, sub := range []string{"org/app #140 の stage:todo を切り替えられません:", "HTTP 403"} {
		if !strings.Contains(text, sub) {
			t.Errorf("失敗に %q が無い: %q", sub, footerOf(text))
		}
	}
	if !m.writeStatusErr {
		t.Error("失敗が赤（エラー扱い）で出ていない")
	}
	if len(m.cards) != len(before) {
		t.Errorf("Cards が変わっている: %d 件, want %d 件", len(m.cards), len(before))
	}
}

// TestTodoStatusIsReplacedByNextToggle はステータスが次の t で消えることを検証する。
func TestTodoStatusIsReplacedByNextToggle(t *testing.T) {
	m, _ := exampleTodoModel(t)
	m, _ = send(m, runeKey('2'))
	m, cmd := send(m, tKey)
	m, _ = runCmd(t, m, cmd)

	m, cmd = send(m, tKey)
	if cmd == nil {
		t.Fatal("結果を受け取った後の t でコマンドが返らない")
	}
	text := plainText(m)
	if strings.Contains(text, "付けました") {
		t.Errorf("前のステータスが残っている: %q", footerOf(text))
	}
	if !strings.Contains(text, "org/app #140 の stage:todo を切り替え中") {
		t.Errorf("切り替え中が出ていない: %q", footerOf(text))
	}
}

// TestWritingBlocksTodoAndAnswer は書き込み中の t / a がどちらも効かないことを検証する
// （s10 answer-question「投稿中は a を無視」をラベル切り替え中にも広げたもの）。
func TestWritingBlocksTodoAndAnswer(t *testing.T) {
	t.Run("切り替え中の t と a", func(t *testing.T) {
		fake := gh.NewFake(fixtureDir)
		ed := &stubEditor{msg: editedMsg{text: "Q1: A"}}
		m := todoModel(fake, exampleResult(t).Cards, ed)
		m, _ = send(m, runeKey('2'))

		m, cmd := send(m, tKey)
		if cmd == nil {
			t.Fatal("最初の t でコマンドが返っていない")
		}
		if _, again := send(m, tKey); again != nil {
			t.Error("切り替え中の t が無視されていない（二重書き込みの余地がある）")
		}
		if _, answered := send(m, aKey); answered != nil {
			t.Error("切り替え中の a が無視されていない")
		}
		if ed.calls != 0 {
			t.Errorf("エディタが起動している: %d 回", ed.calls)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("呼び出し = %+v, want 空（コマンドをまだ実行していない）", fake.Calls)
		}
	})

	t.Run("投稿中の t", func(t *testing.T) {
		ed := &stubEditor{msg: editedMsg{text: "Q1: A"}}
		m, fake := answerModel([]model.Card{prCard(nil)}, ed)

		m, _ = answer(t, m)
		if _, cmd := send(m, tKey); cmd != nil {
			t.Errorf("投稿中の t が無視されていない: %T", cmd())
		}
		if len(fake.Calls) != 0 {
			t.Errorf("呼び出し = %+v, want 空", fake.Calls)
		}
	})

	t.Run("投稿の結果は次の t で消える", func(t *testing.T) {
		ed := &stubEditor{msg: editedMsg{text: "Q1: A"}}
		card := prCard(nil)
		card.Issue = &model.Issue{Repo: "org/app", Number: 140, Title: "手書き", Result: card.Result}
		m, _ := answerModel([]model.Card{card}, ed)

		m, cmd := answer(t, m)
		m, _ = runCmd(t, m, cmd)
		if !strings.Contains(plainText(m), "コメントしました") {
			t.Fatalf("投稿の成功が出ていない: %q", footerOf(plainText(m)))
		}

		m, cmd = send(m, tKey)
		if cmd == nil {
			t.Fatal("投稿完了後の t でコマンドが返らない")
		}
		if text := plainText(m); strings.Contains(text, "コメントしました") {
			t.Errorf("投稿のステータスが残っている: %q", footerOf(text))
		}
	})
}
