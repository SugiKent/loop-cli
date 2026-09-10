package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

var nKey = runeKey('n')

// newDraft は確認画面まで進むときに使う下書き（1 行目がタイトル、空行を挟んで本文）。
const newDraft = "キュー画面の色を見直す\n\n種別の色が背景色と競合している。"

// failingCreateIssue は CreateIssue だけが失敗する client。
type failingCreateIssue struct{ *gh.Fake }

func (c failingCreateIssue) CreateIssue(context.Context, string, string, string) (string, error) {
	return "", errors.New("gh issue create -R org/app: exit 1: HTTP 403")
}

// newIssueConfirm は n を押して編集完了まで進め、作成の確認画面に移った Model を返す。
func newIssueConfirm(t *testing.T, m Model) Model {
	t.Helper()
	m, cmd := send(m, nKey)
	m, _ = runCmd(t, m, cmd)
	if m.screen != screenNewConfirm {
		t.Fatalf("作成の確認画面に移っていない: screen = %d, フッタ = %q", m.screen, footerOf(plainText(m)))
	}
	return m
}

// TestNewIssueRepoIsWhatTheScreenShows は n の作成先が画面の対象のリポジトリに決まることを検証する。
func TestNewIssueRepoIsWhatTheScreenShows(t *testing.T) {
	t.Run("キュー画面は選択行の主体", func(t *testing.T) {
		ed := &stubEditor{}
		m, fake := answerModel(exampleResult(t).Cards, ed)

		m, cmd := send(m, nKey)
		if cmd == nil {
			t.Fatal("n でエディタが起動していない")
		}
		cmd()
		if ed.initial != "" {
			t.Errorf("エディタに渡した初期テキスト = %q, want 空", ed.initial)
		}
		if m.newIssue.repo != "org/app" {
			t.Errorf("作成先 = %q, want org/app", m.newIssue.repo)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("編集前に GitHub を呼んでいる: %+v", fake.Calls)
		}
	})

	t.Run("カード詳細と PR 詳細も同じリポジトリ", func(t *testing.T) {
		for name, keys := range map[string][]tea.Msg{
			"カード詳細": {enterKey},
			"PR 詳細": {enterKey, enterKey},
		} {
			t.Run(name, func(t *testing.T) {
				ed := &stubEditor{}
				m, _ := answerModel(exampleResult(t).Cards, ed)
				m, _ = send(m, keys...)

				m, cmd := send(m, nKey)
				if cmd == nil {
					t.Fatal("n でエディタが起動していない")
				}
				cmd()
				if m.newIssue.repo != "org/app" {
					t.Errorf("作成先 = %q, want org/app", m.newIssue.repo)
				}
				if ed.initial != "" {
					t.Errorf("エディタに渡した初期テキスト = %q, want 空", ed.initial)
				}
			})
		}
	})

	t.Run("0 行のタブでは対象が無いと出す", func(t *testing.T) {
		ed := &stubEditor{}
		m, _ := answerModel(exampleResult(t).Cards, ed)
		m, _ = send(m, runeKey('4'))

		m, cmd := send(m, nKey)
		if cmd != nil {
			t.Errorf("行が無いのにコマンドが返った: %T", cmd())
		}
		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー", m.screen)
		}
		if text := plainText(m); !strings.Contains(text, "新規作成の対象がありません") {
			t.Errorf("フッタに 新規作成の対象がありません が無い: %q", footerOf(text))
		}
		if ed.calls != 0 {
			t.Errorf("エディタが %d 回起動した, want 0", ed.calls)
		}
	})
}

// TestNewIssueChecksEditedDraft は編集結果の検査と、確認画面へ移る条件を検証する。
func TestNewIssueChecksEditedDraft(t *testing.T) {
	t.Run("タイトルと本文がそろえば確認画面に移る", func(t *testing.T) {
		m, fake := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: newDraft}})
		m = newIssueConfirm(t, m)

		text := plainText(m)
		for _, want := range []string{
			"新規 issue の確認: org/app",
			"タイトル: キュー画面の色を見直す",
			"種別の色が背景色と競合している。",
			"y 作成", "e 編集に戻る", "Esc 中止",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("確認画面に %q が無い:\n%s", want, text)
			}
		}
		if len(fake.Calls) != 0 {
			t.Errorf("確認前に作成している: %+v", fake.Calls)
		}
	})

	for name, tc := range map[string]struct {
		edited editedMsg
		want   string
	}{
		"タイトルが空":  {edited: editedMsg{text: "\n本文だけ書いた"}, want: "作成を中止しました（タイトルが空）"},
		"本文が空":    {edited: editedMsg{text: "タイトルだけ書いた"}, want: "作成を中止しました（本文が空）"},
		"エディタが失敗": {edited: editedMsg{err: errors.New("exit status 1")}, want: "エディタ: exit status 1"},
	} {
		t.Run(name, func(t *testing.T) {
			m, fake := answerModel(exampleResult(t).Cards, &stubEditor{msg: tc.edited})

			m, cmd := send(m, nKey)
			m, cmd = runCmd(t, m, cmd)
			if cmd != nil {
				t.Errorf("作成コマンドが返った: %T", cmd())
			}
			if m.screen != screenQueue {
				t.Errorf("画面 = %d, want キュー", m.screen)
			}
			if len(fake.Calls) != 0 {
				t.Errorf("作成している: %+v", fake.Calls)
			}
			if text := plainText(m); !strings.Contains(text, tc.want) {
				t.Errorf("フッタに %q が無い: %q", tc.want, footerOf(text))
			}
		})
	}

	t.Run("マーカーを含む下書きは作成の選択肢が出ない", func(t *testing.T) {
		m, fake := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: "タイトル\n<!-- routine -->\n本文"}})
		m = newIssueConfirm(t, m)

		text := plainText(m)
		if !strings.Contains(text, "<!-- routine --> を含む下書きでは作成できません") {
			t.Errorf("マーカーの警告が無い:\n%s", text)
		}
		for _, want := range []string{"e 編集に戻る", "Esc 中止"} {
			if !strings.Contains(text, want) {
				t.Errorf("確認画面に %q が無い:\n%s", want, text)
			}
		}
		if strings.Contains(text, "y 作成") {
			t.Errorf("マーカーなのに作成の道がある:\n%s", text)
		}

		m, cmd := send(m, runeKey('y'))
		if cmd != nil {
			t.Errorf("y で作成コマンドが返った: %T", cmd())
		}
		if m.screen != screenNewConfirm || len(fake.Calls) != 0 {
			t.Errorf("y で確認画面を抜けた: screen = %d, calls = %+v", m.screen, fake.Calls)
		}
	})
}

// TestNewConfirmKeys は作成の確認画面の y / e / Esc と、他のキーが効かないことを検証する。
func TestNewConfirmKeys(t *testing.T) {
	t.Run("y で issue が作られる", func(t *testing.T) {
		m, fake := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: newDraft}})
		m = newIssueConfirm(t, m)

		m, cmd := send(m, runeKey('y'))
		m, _ = runCmd(t, m, cmd)

		want := gh.Call{
			Method: "CreateIssue", Repo: "org/app",
			Title: "キュー画面の色を見直す", Body: "種別の色が背景色と競合している。",
		}
		if len(fake.Calls) != 1 || fake.Calls[0] != want {
			t.Fatalf("呼び出し = %+v, want 1 件の %+v", fake.Calls, want)
		}
		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー", m.screen)
		}
	})

	t.Run("e で下書き全体を入れたエディタが開く", func(t *testing.T) {
		ed := &stubEditor{msg: editedMsg{text: newDraft}}
		m, fake := answerModel(exampleResult(t).Cards, ed)
		m = newIssueConfirm(t, m)

		m, cmd := send(m, runeKey('e'))
		if cmd == nil {
			t.Fatal("e でエディタが起動していない")
		}
		cmd()
		if ed.initial != newDraft {
			t.Errorf("エディタに渡した下書き = %q, want %q", ed.initial, newDraft)
		}
		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー（n を押した画面）", m.screen)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("作成している: %+v", fake.Calls)
		}
	})

	// 経路の記録を作成・中止で消すと、ここで下書きが直前の回答対象へコメントとして投稿される。
	t.Run("e から戻った編集完了は再び作成の確認画面になる", func(t *testing.T) {
		m, fake := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: newDraft}})
		m = newIssueConfirm(t, m)

		m, cmd := send(m, runeKey('e'))
		m, _ = runCmd(t, m, cmd)

		if m.screen != screenNewConfirm {
			t.Errorf("画面 = %d, want 作成の確認", m.screen)
		}
		for _, c := range fake.Calls {
			if c.Method == "CommentPR" || c.Method == "CommentIssue" {
				t.Errorf("下書きがコメントとして投稿されている: %+v", c)
			}
		}
	})

	t.Run("Esc で中止する", func(t *testing.T) {
		m, fake := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: newDraft}})
		m = newIssueConfirm(t, m)

		m, _ = send(m, escKey)

		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー", m.screen)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("作成している: %+v", fake.Calls)
		}
		if text := plainText(m); !strings.Contains(text, "作成を中止しました") {
			t.Errorf("中止が出ていない: %q", footerOf(text))
		}
	})

	t.Run("詳細画面から入った確認画面は詳細画面に戻る", func(t *testing.T) {
		m, _ := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: newDraft}})
		m, _ = send(m, enterKey)
		m = newIssueConfirm(t, m)

		m, _ = send(m, escKey)

		if m.screen != screenCard {
			t.Fatalf("画面 = %d, want カード詳細", m.screen)
		}
		if m.detail.card.Issue.Number != 108 {
			t.Errorf("戻り先のカードが変わっている: #%d", m.detail.card.Issue.Number)
		}
	})

	t.Run("確認中に取得が完了しても作成先は変わらない", func(t *testing.T) {
		m, fake := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: newDraft}})
		m = newIssueConfirm(t, m)

		// issue 108 の Card を含まない Result（主体は別リポジトリ）が届いても作成先は変わらない。
		m, _ = send(m, fetchedMsg{res: &fetch.Result{Cards: []model.Card{nowCard("other/app", 9, 1, at)}}, at: at})
		if m.screen != screenNewConfirm {
			t.Fatalf("取得完了で画面が変わった: screen = %d", m.screen)
		}

		m, cmd := send(m, runeKey('y'))
		m, _ = runCmd(t, m, cmd)

		if len(fake.Calls) != 1 || fake.Calls[0].Repo != "org/app" {
			t.Errorf("呼び出し = %+v, want org/app への CreateIssue 1 件", fake.Calls)
		}
	})

	t.Run("他のキーは何もしない", func(t *testing.T) {
		base, fake := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: newDraft}})
		base = newIssueConfirm(t, base)

		for _, k := range []rune{'a', 't', 'm', 'o', '?', 'u', 'n'} {
			t.Run(string(k), func(t *testing.T) {
				got, cmd := send(base, runeKey(k))
				if cmd != nil {
					t.Errorf("%c でコマンドが返った: %T", k, cmd())
				}
				if got.screen != screenNewConfirm {
					t.Errorf("%c で画面が変わった: screen = %d", k, got.screen)
				}
			})
		}
		if len(fake.Calls) != 0 {
			t.Errorf("確認画面から GitHub を呼んでいる: %+v", fake.Calls)
		}
	})

	t.Run("q で終了する", func(t *testing.T) {
		m, _ := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: newDraft}})
		m = newIssueConfirm(t, m)

		_, cmd := send(m, runeKey('q'))
		if cmd == nil {
			t.Fatal("q で終了コマンドが返らなかった")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("返ったコマンドが終了メッセージを生まない: %T", cmd())
		}
	})
}

// TestNewIssueResultShowsURL は作成の結果がフッタに出て、表を書き換えないことを検証する。
func TestNewIssueResultShowsURL(t *testing.T) {
	t.Run("成功すると URL が出る", func(t *testing.T) {
		m, fake := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: newDraft}})
		before := len(m.cards)
		m = newIssueConfirm(t, m)

		m, cmd := send(m, runeKey('y'))
		m, _ = runCmd(t, m, cmd)

		text := plainText(m)
		if !strings.Contains(text, "issue を作成しました: https://github.com/org/app/issues/0") {
			t.Errorf("成功と URL が出ていない: %q", footerOf(text))
		}
		if len(m.cards) != before {
			t.Errorf("Cards が変わった: %d 件, want %d 件", len(m.cards), before)
		}
		if !strings.Contains(text, "#108") {
			t.Error("作成後に再取得して行が消えている（1 件再取得は s18 の担当）")
		}
		for _, c := range fake.Calls {
			if c.Method == "AddLabel" || c.Method == "RemoveLabel" {
				t.Errorf("作成でラベルを書いている: %+v", c)
			}
		}
	})

	// 文言に作成先を入れると幅 80 で URL の末尾が省略記号なしに切れて使えなくなる。
	t.Run("長いリポジトリ名でも幅 80 で URL が切れない", func(t *testing.T) {
		m, _ := answerModel([]model.Card{nowCard("SugiKent/loop-cli", 1, 1, at)}, &stubEditor{msg: editedMsg{text: newDraft}})
		m, _ = send(m, tea.WindowSizeMsg{Width: 80, Height: 24})
		m = newIssueConfirm(t, m)

		m, cmd := send(m, runeKey('y'))
		m, _ = runCmd(t, m, cmd)

		if got := footerOf(plainText(m)); !strings.Contains(got, "issue を作成しました: https://github.com/SugiKent/loop-cli/issues/0") {
			t.Errorf("幅 80 のフッタで URL が切れている: %q", got)
		}
	})

	t.Run("作成中は作成中が出て a と t と m と n が効かない", func(t *testing.T) {
		m, fake := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: newDraft}})
		m = newIssueConfirm(t, m)

		m, cmd := send(m, runeKey('y'))
		if cmd == nil {
			t.Fatal("y で作成コマンドが返っていない")
		}
		if text := plainText(m); !strings.Contains(text, "org/app に issue を作成中") {
			t.Errorf("作成中が出ていない: %q", footerOf(text))
		}
		for _, k := range []rune{'a', 't', 'm', 'n'} {
			if _, ignored := send(m, runeKey(k)); ignored != nil {
				t.Errorf("作成中の %c が無視されていない: %T", k, ignored())
			}
		}
		if len(fake.Calls) != 0 {
			t.Errorf("呼び出し = %+v, want 空（コマンドをまだ実行していない）", fake.Calls)
		}
	})

	t.Run("作成の結果は次の n で消える", func(t *testing.T) {
		m, _ := answerModel(exampleResult(t).Cards, &stubEditor{msg: editedMsg{text: newDraft}})
		m = newIssueConfirm(t, m)

		m, cmd := send(m, runeKey('y'))
		m, _ = runCmd(t, m, cmd)
		if !strings.Contains(plainText(m), "issue を作成しました") {
			t.Fatalf("作成の成功が出ていない: %q", footerOf(plainText(m)))
		}

		m, cmd = send(m, nKey)
		if cmd == nil {
			t.Fatal("作成完了後の n でエディタが起動していない")
		}
		if text := plainText(m); strings.Contains(text, "issue を作成しました") {
			t.Errorf("作成のステータスが残っている: %q", footerOf(text))
		}
	})

	t.Run("失敗は赤で出て表を変えない", func(t *testing.T) {
		client := failingCreateIssue{gh.NewFake(fixtureDir)}
		cards := exampleResult(t).Cards
		m, _ := send(New(nil, client, (&stubEditor{msg: editedMsg{text: newDraft}}).Editor, Options{}),
			tea.WindowSizeMsg{Width: 120, Height: 40},
			fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})
		m = newIssueConfirm(t, m)

		m, cmd := send(m, runeKey('y'))
		m, _ = runCmd(t, m, cmd)

		text := plainText(m)
		for _, want := range []string{"org/app の issue 作成に失敗:", "HTTP 403"} {
			if !strings.Contains(text, want) {
				t.Errorf("フッタに %q が無い: %q", want, footerOf(text))
			}
		}
		if len(m.cards) != len(cards) {
			t.Errorf("失敗で表が変わった: %d 件, want %d 件", len(m.cards), len(cards))
		}
		if !m.writeStatusErr {
			t.Error("失敗が赤で出ていない")
		}
	})
}
