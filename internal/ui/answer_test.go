package ui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/action"
	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

var aKey = runeKey('a')

// answerModel は回答のテスト用の Model と、投稿先の Fake を返す。
// Fake は Result を作ったものとは別に作る（s07 の Fetch が ViewIssue を Calls に残すため）。
func answerModel(cards []model.Card, ed *stubEditor) (Model, *gh.Fake) {
	fake := gh.NewFake(fixtureDir)
	m, _ := send(New(nil, fake, ed.Editor, Options{}),
		tea.WindowSizeMsg{Width: 120, Height: 40},
		fetchedMsg{res: &fetch.Result{Cards: cards, Modes: sddModes}, at: at})
	return m, fake
}

// prCard は主体が PR 131 の手書き Card。
func prCard(comments []model.Comment) model.Card {
	result := model.Result{Situation: model.SituationA, Priority: 1, Tab: model.TabNow, Summary: "回答する"}
	pr := model.PR{
		Repo: "org/app", Number: 131, Title: "手書き PR",
		State: "OPEN", UpdatedAt: at, Comments: comments, Result: result,
	}
	return model.Card{PRs: []model.PR{pr}, Result: result}
}

// runCmd はコマンドを実行してメッセージを Model に渡す。
func runCmd(t *testing.T, m Model, cmd tea.Cmd) (Model, tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("コマンドが返っていない")
	}
	return send(m, cmd())
}

// answer は a を押してエディタを起動し、スタブの編集結果まで処理した Model を返す。
func answer(t *testing.T, m Model) (Model, tea.Cmd) {
	t.Helper()
	m, cmd := send(m, aKey)
	return runCmd(t, m, cmd)
}

// TestAnswerTargetIsWhatTheScreenShows は `a` の対象が画面の見せているものに決まることを検証する。
// 局面 A は PR、局面 B は issue が主体なので、キュー画面の `a` はそのまま正しい書き先になる。
func TestAnswerTargetIsWhatTheScreenShows(t *testing.T) {
	question := "<!-- routine -->\n### Q1. 分けるか\n- **選択肢 A（推奨）**: 分ける\n- **選択肢 B**: 分けない"

	t.Run("キュー画面は選択行の主体", func(t *testing.T) {
		ed := &stubEditor{}
		m, _ := answerModel([]model.Card{prCard([]model.Comment{{Body: question, AI: true}})}, ed)

		m, cmd := send(m, aKey)
		if cmd == nil {
			t.Fatal("a でエディタが起動していない")
		}
		if want := (action.Target{Repo: "org/app", Number: 131, IsPR: true}); m.answer.target != want {
			t.Errorf("対象 = %+v, want %+v", m.answer.target, want)
		}
	})

	// 上流の書式の質問コメントが、引用付きでエディタに届くことを見る（issue #38 の動機そのもの）。
	t.Run("エディタには質問の引用と回答行が入る", func(t *testing.T) {
		ed := &stubEditor{}
		m, _ := answerModel([]model.Card{prCard([]model.Comment{{Body: question, AI: true}})}, ed)

		_, cmd := send(m, aKey)
		if cmd == nil {
			t.Fatal("a でエディタが起動していない")
		}
		cmd()
		want := "> ### Q1. 分けるか\n> - **選択肢 A（推奨）**: 分ける\n> - **選択肢 B**: 分けない\n\nQ1: A"
		if ed.initial != want {
			t.Errorf("テンプレート = %q, want %q", ed.initial, want)
		}
	})

	t.Run("カード詳細は Issue", func(t *testing.T) {
		ed := &stubEditor{}
		m, _ := answerModel(exampleResult(t).Cards, ed)
		m, _ = send(m, enterKey)

		m, cmd := send(m, aKey)
		if cmd == nil {
			t.Fatal("a でエディタが起動していない")
		}
		if want := (action.Target{Repo: "org/app", Number: 108}); m.answer.target != want {
			t.Errorf("対象 = %+v, want %+v", m.answer.target, want)
		}
		cmd()
		if ed.initial != "" {
			t.Errorf("見出しの無い質問からテンプレートを作っている: %q", ed.initial)
		}
	})

	t.Run("PR 詳細はその PR", func(t *testing.T) {
		ed := &stubEditor{}
		m, _ := answerModel(exampleResult(t).Cards, ed)
		m, _ = send(m, enterKey, enterKey)
		if m.screen != screenPR {
			t.Fatalf("前提が崩れている: screen = %d", m.screen)
		}

		m, cmd := send(m, aKey)
		if cmd == nil {
			t.Fatal("a でエディタが起動していない")
		}
		if want := (action.Target{Repo: "org/app", Number: 131, IsPR: true}); m.answer.target != want {
			t.Errorf("対象 = %+v, want %+v", m.answer.target, want)
		}
		cmd()
		if ed.initial != "" {
			t.Errorf("見出しの無い質問からテンプレートを作っている: %q", ed.initial)
		}
	})

	t.Run("question の無いバックログの issue にも書ける", func(t *testing.T) {
		ed := &stubEditor{}
		m, _ := answerModel(exampleResult(t).Cards, ed)
		m, _ = send(m, runeKey('2'))

		m, cmd := send(m, aKey)
		if cmd == nil {
			t.Fatal("a でエディタが起動していない")
		}
		if want := (action.Target{Repo: "org/app", Number: 140}); m.answer.target != want {
			t.Errorf("対象 = %+v, want %+v", m.answer.target, want)
		}
		cmd()
		if ed.initial != "" {
			t.Errorf("問いの無い issue にテンプレートが入っている: %q", ed.initial)
		}
	})

	t.Run("行が無いタブでは何も起きない", func(t *testing.T) {
		ed := &stubEditor{}
		m, _ := answerModel(exampleResult(t).Cards, ed)
		m, _ = send(m, runeKey('4'))

		m, cmd := send(m, aKey)
		if cmd != nil {
			t.Errorf("行が無いのにコマンドが返った: %T", cmd())
		}
		if ed.calls != 0 {
			t.Errorf("エディタが %d 回起動した, want 0", ed.calls)
		}
		if m.screen != screenQueue {
			t.Errorf("画面が変わった: screen = %d", m.screen)
		}
	})
}

// TestAnswerPostsEditedBody は編集結果がそのまま書き先へ届き、ラベルには触れないことを検証する。
func TestAnswerPostsEditedBody(t *testing.T) {
	ed := &stubEditor{msg: editedMsg{text: "Q1: A"}}
	m, fake := answerModel([]model.Card{prCard(nil)}, ed)

	m, cmd := answer(t, m)
	if text := plainText(m); !strings.Contains(text, "org/app PR#131 にコメントを投稿中") {
		t.Errorf("投稿中が出ていない: %q", footerOf(text))
	}
	if _, ignored := send(m, aKey); ignored != nil {
		t.Error("投稿中の a が無視されていない（二重投稿の余地がある）")
	}

	m, _ = runCmd(t, m, cmd)

	want := gh.Call{Method: "CommentPR", Repo: "org/app", Number: 131, Body: "Q1: A"}
	if len(fake.Calls) != 1 || !reflect.DeepEqual(fake.Calls[0], want) {
		t.Fatalf("呼び出し = %+v, want 1 件の %+v", fake.Calls, want)
	}
	text := plainText(m)
	if !strings.Contains(text, "org/app PR#131 にコメントしました") {
		t.Errorf("成功が出ていない: %q", footerOf(text))
	}
	if m.screen != screenQueue {
		t.Errorf("画面 = %d, want キュー", m.screen)
	}
	if !strings.Contains(text, "PR131") {
		t.Error("投稿後に再取得して行が消えている（1 件再取得は s18 の担当）")
	}
}

// TestAnswerPostsQuoteWithAnswer は、引用を残したまま書き足した本文がそのまま投稿されることを検証する。
// TUI は人が書いた行を落とさないので、引用は投稿されるコメントの一部になる。
func TestAnswerPostsQuoteWithAnswer(t *testing.T) {
	body := "> 認可の方針をどこに書きますか。\n\nA でお願いします"
	ed := &stubEditor{msg: editedMsg{text: body}}
	m, fake := answerModel([]model.Card{prCard(nil)}, ed)

	m, cmd := answer(t, m)
	runCmd(t, m, cmd)

	want := gh.Call{Method: "CommentPR", Repo: "org/app", Number: 131, Body: body}
	if len(fake.Calls) != 1 || !reflect.DeepEqual(fake.Calls[0], want) {
		t.Errorf("呼び出し = %+v, want 1 件の %+v", fake.Calls, want)
	}
}

// TestAnswerAbortsWithoutPosting は投稿に至らない編集結果を検証する。
func TestAnswerAbortsWithoutPosting(t *testing.T) {
	for name, tc := range map[string]struct {
		edited editedMsg
		want   string
	}{
		"本文が空": {edited: editedMsg{text: " \n"}, want: "回答を中止しました（本文が空）"},
		// 引用だけの下書きを投稿すると、dispatcher が「人が答えた」とみなして worker が推奨案で進む。
		"引用だけ": {
			edited: editedMsg{text: "> 認可の方針をどこに書きますか。\n>\n> - 選択肢 A（推奨）: docs/policy.md"},
			want:   "回答を中止しました（引用だけです）",
		},
		"エディタが失敗": {edited: editedMsg{err: errors.New("exit status 1")}, want: "エディタ: exit status 1"},
	} {
		t.Run(name, func(t *testing.T) {
			ed := &stubEditor{msg: tc.edited}
			m, fake := answerModel([]model.Card{prCard(nil)}, ed)

			m, cmd := answer(t, m)
			if cmd != nil {
				t.Errorf("投稿コマンドが返った: %T", cmd())
			}
			if len(fake.Calls) != 0 {
				t.Errorf("投稿している: %+v", fake.Calls)
			}
			if text := plainText(m); !strings.Contains(text, tc.want) {
				t.Errorf("フッタに %q が無い: %q", tc.want, footerOf(text))
			}
			if m.screen != screenQueue {
				t.Errorf("画面 = %d, want キュー", m.screen)
			}
		})
	}
}

// TestConfirmScreenWarnsBeforePosting は不変条件 7 / 8 の検出が確認画面になることを検証する。
func TestConfirmScreenWarnsBeforePosting(t *testing.T) {
	t.Run("blocked-by は警告のうえで投稿できる", func(t *testing.T) {
		ed := &stubEditor{msg: editedMsg{text: "Q1: A\n  blocked-by: human"}}
		m, fake := answerModel([]model.Card{prCard(nil)}, ed)

		m, cmd := answer(t, m)
		if cmd != nil {
			t.Errorf("確認せずに投稿コマンドが返った: %T", cmd())
		}
		if m.screen != screenConfirm {
			t.Fatalf("画面 = %d, want 確認", m.screen)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("確認前に投稿している: %+v", fake.Calls)
		}
		text := plainText(m)
		for _, want := range []string{
			"回答の確認: org/app PR#131",
			"blocked-by: で始まる行があります",
			"blocked-by: human",
			"Q1: A",
			"y 投稿", "e 編集に戻る", "Esc 中止",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("確認画面に %q が無い:\n%s", want, text)
			}
		}
	})

	t.Run("マーカーは投稿の道を出さない", func(t *testing.T) {
		ed := &stubEditor{msg: editedMsg{text: "<!-- routine -->\nQ1: A"}}
		m, fake := answerModel([]model.Card{prCard(nil)}, ed)

		m, _ = answer(t, m)
		if m.screen != screenConfirm {
			t.Fatalf("画面 = %d, want 確認", m.screen)
		}
		text := plainText(m)
		if !strings.Contains(text, "<!-- routine --> を含む本文は投稿できません") {
			t.Errorf("マーカーの警告が無い:\n%s", text)
		}
		for _, want := range []string{"e 編集に戻る", "Esc 中止"} {
			if !strings.Contains(text, want) {
				t.Errorf("確認画面に %q が無い:\n%s", want, text)
			}
		}
		if strings.Contains(text, "y 投稿") {
			t.Errorf("マーカーなのに投稿の道がある:\n%s", text)
		}

		m, cmd := send(m, runeKey('y'))
		if cmd != nil {
			t.Errorf("y で投稿コマンドが返った: %T", cmd())
		}
		if m.screen != screenConfirm || len(fake.Calls) != 0 {
			t.Errorf("y で確認画面を抜けた: screen = %d, calls = %+v", m.screen, fake.Calls)
		}
	})
}

// TestConfirmKeys は確認画面の y / e / Esc を検証する。
func TestConfirmKeys(t *testing.T) {
	const draft = "Q1: A\n  blocked-by: human"

	t.Run("y で下書きのまま投稿する", func(t *testing.T) {
		ed := &stubEditor{msg: editedMsg{text: draft}}
		m, fake := answerModel([]model.Card{prCard(nil)}, ed)
		m, _ = answer(t, m)

		m, cmd := send(m, runeKey('y'))
		m, _ = runCmd(t, m, cmd)

		want := gh.Call{Method: "CommentPR", Repo: "org/app", Number: 131, Body: draft}
		if len(fake.Calls) != 1 || !reflect.DeepEqual(fake.Calls[0], want) {
			t.Errorf("呼び出し = %+v, want 1 件の %+v", fake.Calls, want)
		}
		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー", m.screen)
		}
	})

	t.Run("e で下書きを持ってエディタに戻る", func(t *testing.T) {
		ed := &stubEditor{msg: editedMsg{text: draft}}
		m, fake := answerModel([]model.Card{prCard(nil)}, ed)
		m, _ = answer(t, m)

		m, cmd := send(m, runeKey('e'))
		if cmd == nil {
			t.Fatal("e でエディタが起動していない")
		}
		cmd()
		if ed.initial != draft {
			t.Errorf("エディタに渡した下書き = %q, want %q", ed.initial, draft)
		}
		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー（a を押した画面）", m.screen)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("投稿している: %+v", fake.Calls)
		}
	})

	t.Run("Esc で中止して元の画面に戻る", func(t *testing.T) {
		ed := &stubEditor{msg: editedMsg{text: "Q1: A\nblocked-by: human"}}
		m, fake := answerModel(exampleResult(t).Cards, ed)
		m, _ = send(m, enterKey)
		m, _ = answer(t, m)
		if m.screen != screenConfirm {
			t.Fatalf("画面 = %d, want 確認", m.screen)
		}

		m, _ = send(m, escKey)

		if m.screen != screenCard {
			t.Errorf("画面 = %d, want カード詳細", m.screen)
		}
		if m.detail.card.Issue.Number != 108 {
			t.Errorf("戻り先のカードが変わっている: #%d", m.detail.card.Issue.Number)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("投稿している: %+v", fake.Calls)
		}
		if text := plainText(m); !strings.Contains(text, "回答を中止しました") {
			t.Errorf("中止が出ていない: %q", footerOf(text))
		}
	})
}

// TestEditRouteSeparatesAnswerFromNewIssue は編集完了が始めたキーの経路で扱われることを検証する。
// 経路を取り違えると、issue の下書きが直前の回答対象へコメントとして投稿される。
func TestEditRouteSeparatesAnswerFromNewIssue(t *testing.T) {
	t.Run("n で始めた編集は回答として扱わない", func(t *testing.T) {
		ed := &stubEditor{msg: editedMsg{text: newDraft}}
		m, fake := answerModel([]model.Card{prCard(nil)}, ed)

		m, cmd := send(m, nKey)
		m, _ = runCmd(t, m, cmd)

		if m.screen != screenNewConfirm {
			t.Errorf("画面 = %d, want 作成の確認", m.screen)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("回答として投稿している: %+v", fake.Calls)
		}
		if text := plainText(m); strings.Contains(text, "回答の確認:") {
			t.Errorf("回答の確認画面になっている:\n%s", text)
		}
	})

	t.Run("回答の確認画面の e から戻った編集完了は回答として投稿される", func(t *testing.T) {
		const draft = "Q1: A\n  blocked-by: human"
		ed := &stubEditor{msg: editedMsg{text: draft}}
		m, fake := answerModel([]model.Card{prCard(nil)}, ed)
		m, _ = answer(t, m)
		if m.screen != screenConfirm {
			t.Fatalf("確認画面に移っていない: screen = %d", m.screen)
		}

		ed.msg = editedMsg{text: "Q1: A"}
		m, cmd := send(m, runeKey('e'))
		m, cmd = runCmd(t, m, cmd)
		if ed.initial != draft {
			t.Errorf("エディタに渡した下書き = %q, want %q", ed.initial, draft)
		}
		m, _ = runCmd(t, m, cmd)

		want := gh.Call{Method: "CommentPR", Repo: "org/app", Number: 131, Body: "Q1: A"}
		if len(fake.Calls) != 1 || !reflect.DeepEqual(fake.Calls[0], want) {
			t.Fatalf("呼び出し = %+v, want 1 件の %+v", fake.Calls, want)
		}
		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー", m.screen)
		}
	})
}

// failingClient は投稿だけが失敗する client。
type failingClient struct {
	*gh.Fake
	err error
}

func (c *failingClient) CommentPR(context.Context, string, int, string) error { return c.err }

// TestAnswerPostFailureKeepsCards は投稿の失敗をフッタに出し、表を壊さないことを検証する。
func TestAnswerPostFailureKeepsCards(t *testing.T) {
	ed := &stubEditor{msg: editedMsg{text: "Q1: A"}}
	client := &failingClient{
		Fake: gh.NewFake(fixtureDir),
		err:  errors.New("gh pr comment 131 -R org/app --body-file -: exit 1: HTTP 403"),
	}
	cards := []model.Card{prCard(nil)}
	m, _ := send(New(nil, client, ed.Editor, Options{}),
		tea.WindowSizeMsg{Width: 120, Height: 40},
		fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})

	m, cmd := answer(t, m)
	m, _ = runCmd(t, m, cmd)

	text := plainText(m)
	for _, want := range []string{"org/app PR#131 へのコメントに失敗:", "HTTP 403"} {
		if !strings.Contains(text, want) {
			t.Errorf("フッタに %q が無い: %q", want, footerOf(text))
		}
	}
	if len(m.cards) != len(cards) {
		t.Errorf("失敗で表が変わった: %d 件, want %d 件", len(m.cards), len(cards))
	}
}

// TestAnswerFromQueueDoesNotWriteBeforeEditing は a を押しただけでは何も書かないことを検証する。
func TestAnswerFromQueueDoesNotWriteBeforeEditing(t *testing.T) {
	ed := &stubEditor{}
	m, fake := answerModel(exampleResult(t).Cards, ed)

	if _, cmd := send(m, aKey); cmd == nil {
		t.Fatal("a でエディタが起動していない")
	} else {
		cmd()
	}

	if ed.calls != 1 {
		t.Errorf("エディタの起動 = %d 回, want 1", ed.calls)
	}
	if len(fake.Calls) != 0 {
		t.Errorf("編集前に GitHub を呼んでいる: %+v", fake.Calls)
	}
}

// footerOf はエラーメッセージ用にフッタ行だけを取り出す。
func footerOf(text string) string {
	lines := strings.Split(text, "\n")
	return lines[len(lines)-1]
}
