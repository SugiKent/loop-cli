package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

var questionKey = runeKey('?')

// helpModel はヘルプのテスト用の Model と、押されない前提の Fake / Editor を返す。
func helpModel(t *testing.T) (Model, *gh.Fake, *stubEditor) {
	t.Helper()
	fake := gh.NewFake(fixtureDir)
	ed := &stubEditor{}
	return todoModel(fake, exampleResult(t).Cards, ed), fake, ed
}

// helpBody はヘルプ画面の View からキーの行（見出しと空行の次から、次の空行かフッタの手前まで）を返す。
// キーの行が端末の高さを埋め切ると空行が 1 つも残らないので、最終行のフッタを先に外す。
func helpBody(t *testing.T, m Model) []string {
	t.Helper()
	lines := plain(m)
	if lines[0] != "キーバインド" {
		t.Fatalf("1 行目 = %q, want キーバインド", lines[0])
	}
	var body []string
	for _, l := range lines[2 : len(lines)-1] {
		if strings.TrimSpace(l) == "" {
			break
		}
		body = append(body, l)
	}
	return body
}

// TestHelpOpensAndClosesFromQueue はキューから開いて ? で戻ることを検証する。
func TestHelpOpensAndClosesFromQueue(t *testing.T) {
	m, _, _ := helpModel(t)
	m, _ = send(m, runeKey('2'))

	m, cmd := send(m, questionKey)
	if cmd != nil {
		t.Errorf("? でコマンドが返った: %T", cmd())
	}
	if m.screen != screenHelp {
		t.Fatalf("画面 = %d, want ヘルプ", m.screen)
	}

	m, _ = send(m, questionKey)

	if m.screen != screenQueue {
		t.Errorf("画面 = %d, want キュー", m.screen)
	}
	if m.tab != model.TabBacklog || m.cursor != 0 {
		t.Errorf("戻ったときの tab = %q cursor = %d, want バックログ / 0", m.tab, m.cursor)
	}
}

// TestHelpReturnsToCardDetail はカード詳細へ戻り、コメントの全文が出たままであることを検証する。
func TestHelpReturnsToCardDetail(t *testing.T) {
	m, _, _ := helpModel(t)
	// 幅 161 は AI コメントの 2 行目が折り返されずに 1 行に収まる幅（detail_test.go と同じ）。
	m, _ = send(m, tea.WindowSizeMsg{Width: 161, Height: 40}, enterKey)

	m, _ = send(m, questionKey, codeKey(tea.KeyEscape))

	if m.screen != screenCard {
		t.Fatalf("画面 = %d, want カード詳細", m.screen)
	}
	if m.detail.card.Issue.Number != 108 {
		t.Errorf("詳細の対象 = #%d, want #108", m.detail.card.Issue.Number)
	}
	line, ok := lineWith(plain(m), "Q2: 失効時はログイン画面へ戻しますか。")
	if !ok {
		t.Fatalf("戻った後に AI コメントの 2 行目が無い:\n%s", plainText(m))
	}
	if !strings.HasPrefix(line, "▌") {
		t.Errorf("戻った後の AI コメントの行が ▌ で始まらない: %q", line)
	}
}

// TestHelpReturnsToPRDetail は PR 詳細へ戻ることを検証する。
func TestHelpReturnsToPRDetail(t *testing.T) {
	m, _, _ := helpModel(t)
	m, _ = send(m, enterKey, enterKey)

	m, _ = send(m, questionKey, codeKey(tea.KeyEscape))

	if m.screen != screenPR {
		t.Fatalf("画面 = %d, want PR 詳細", m.screen)
	}
	if m.currentPR().Number != 131 {
		t.Errorf("対象 = PR#%d, want PR#131", m.currentPR().Number)
	}
}

// TestHelpDoesNotOpenFromConfirm は確認画面から ? が効かないことを検証する。
func TestHelpDoesNotOpenFromConfirm(t *testing.T) {
	m, _ := confirmModel(t)

	got, cmd := send(m, questionKey)

	if cmd != nil {
		t.Errorf("? でコマンドが返った: %T", cmd())
	}
	if got.screen != screenConfirm {
		t.Errorf("画面 = %d, want 確認", got.screen)
	}
}

// TestHelpIgnoresOtherKeys はヘルプ画面が他のキーで何もしないことを検証する。
func TestHelpIgnoresOtherKeys(t *testing.T) {
	base, fake, ed := helpModel(t)
	base, _ = send(base, questionKey)

	for _, k := range []rune{'j', '2', 'a', 't', 'n', 'o', 'R', 'x', 'g', 'p'} {
		t.Run(string(k), func(t *testing.T) {
			got, cmd := send(base, runeKey(k))
			if cmd != nil {
				t.Errorf("%c でコマンドが返った: %T", k, cmd())
			}
			if got.screen != screenHelp {
				t.Errorf("%c で画面が変わった: screen = %d", k, got.screen)
			}
			if got.tab != base.tab || got.cursor != base.cursor {
				t.Errorf("%c で tab / cursor が変わった: %q / %d", k, got.tab, got.cursor)
			}
		})
	}
	t.Run("enter", func(t *testing.T) {
		got, cmd := send(base, enterKey)
		if cmd != nil {
			t.Errorf("Enter でコマンドが返った: %T", cmd())
		}
		if got.screen != screenHelp {
			t.Errorf("Enter で画面が変わった: screen = %d", got.screen)
		}
	})
	if len(fake.Calls) != 0 {
		t.Errorf("ヘルプ画面から gh を呼んでいる: %+v", fake.Calls)
	}
	if ed.calls != 0 {
		t.Errorf("ヘルプ画面からエディタを起動している: %d 回", ed.calls)
	}
}

// TestHelpQuits はヘルプ画面でも q で終了することを検証する。
func TestHelpQuits(t *testing.T) {
	m, _, _ := helpModel(t)
	m, _ = send(m, questionKey)

	_, cmd := send(m, runeKey('q'))

	if cmd == nil {
		t.Fatal("q で終了コマンドが返らなかった")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("返ったコマンドが終了メッセージを生まない: %T", cmd())
	}
}

// TestFetchedWhileHelpOpen はヘルプ中の取得完了が画面を変えずに反映されることを検証する。
func TestFetchedWhileHelpOpen(t *testing.T) {
	m, _, _ := helpModel(t)
	m, _ = send(m, questionKey)

	m, _ = send(m, fetchedMsg{res: &fetch.Result{Cards: threeNowCards()}, at: at})

	if m.screen != screenHelp {
		t.Fatalf("取得完了で画面が変わった: screen = %d", m.screen)
	}

	m, _ = send(m, questionKey)

	text := plainText(m)
	if !strings.Contains(text, "手書き") {
		t.Errorf("新しい Result の行が出ていない:\n%s", text)
	}
	if strings.Contains(text, "PR131") {
		t.Errorf("古い Result の行が残っている:\n%s", text)
	}
}

// TestHelpListsImplementedKeys はヘルプが実装済みのキーだけを順に一覧することを検証する。
func TestHelpListsImplementedKeys(t *testing.T) {
	m, _, _ := helpModel(t)
	m, _ = send(m, tea.WindowSizeMsg{Width: 80, Height: 24}, questionKey)

	body := helpBody(t, m)
	if len(body) != 20 {
		t.Fatalf("キーの行数 = %d, want 20:\n%s", len(body), strings.Join(body, "\n"))
	}
	for _, l := range body {
		if strings.HasPrefix(l, "x") {
			t.Errorf("折りたたみを廃止したのに x の行がある: %q", l)
		}
	}
	if !strings.Contains(body[0], "j / k / ↑ / ↓") || !strings.Contains(body[0], "行移動（キュー）/ スクロール（詳細）") {
		t.Errorf("1 行目 = %q", body[0])
	}

	wants := []struct{ prefix, desc string }{
		{"t", "stage:todo / To Do を付ける / 外す"},
		{"L", "ラベルを一覧から付け外し"},
		{"m", "PR を merge する（確認あり）"},
		{"c", "issue / PR を close する（確認あり）"},
		{"n", "選択中の repo に issue を作る（確認あり）"},
		{"o", "ブラウザで開く"},
		{"u", "URL 一覧を開く（セッション URL を含む）"},
		{"R", "全件再取得（キュー）/ セッション取得（詳細）"},
		{"?", "ヘルプを開く / 閉じる"},
		{"PgUp / PgDn", "ページ単位のスクロール（詳細）"},
		{"G / End", "本文の末尾へ飛ぶ（詳細）"},
		{"Home", "本文の先頭へ飛ぶ（詳細）"},
	}
	i := 1
	for _, want := range wants {
		found := -1
		for j := i; j < len(body); j++ {
			if strings.HasPrefix(body[j], want.prefix) && strings.Contains(body[j], want.desc) {
				found = j
				break
			}
		}
		if found < 0 {
			t.Fatalf("%q で始まり %q を含む行が %d 行目以降に無い:\n%s", want.prefix, want.desc, i, strings.Join(body, "\n"))
		}
		i = found + 1
	}

	text := plainText(m)
	for _, ng := range []string{"即着手", "カンバン", "絞り込み", "review thread", "表とプレビューの切替"} {
		if strings.Contains(text, ng) {
			t.Errorf("未実装のキーの %q が出ている:\n%s", ng, text)
		}
	}

	footer := footerOf(text)
	for _, want := range []string{"? / Esc 閉じる", "q 終了"} {
		if !strings.Contains(footer, want) {
			t.Errorf("フッタに %q が無い: %q", want, footer)
		}
	}
	for _, ng := range []string{"Enter 開く", "1-4/Tab タブ"} {
		if strings.Contains(footer, ng) {
			t.Errorf("フッタにキュー画面のヒント %q がある: %q", ng, footer)
		}
	}
}

// TestHelpCutsTailOnShortTerminal は低い端末で末尾の行を切ることを検証する。
func TestHelpCutsTailOnShortTerminal(t *testing.T) {
	m, _, _ := helpModel(t)
	m, _ = send(m, questionKey, tea.WindowSizeMsg{Width: 80, Height: 10})

	lines := plain(m)
	if len(lines) != 10 {
		t.Fatalf("行数 = %d, want 10:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if lines[0] != "キーバインド" {
		t.Errorf("1 行目 = %q, want キーバインド", lines[0])
	}
	if !strings.Contains(lines[9], "? / Esc 閉じる") {
		t.Errorf("最終行 = %q", lines[9])
	}
	for _, l := range lines {
		if strings.HasPrefix(l, "PgUp / PgDn") {
			t.Errorf("切られるはずの PgUp / PgDn の行が出ている: %q", l)
		}
	}
}

// TestHelpShowsSpinnerWhileFetching はヘルプ画面でも取得中のスピナーが出ることを検証する。
func TestHelpShowsSpinnerWhileFetching(t *testing.T) {
	m, _ := send(newModel(nil), questionKey)

	lines := plain(m)
	if lines[0] != "キーバインド" {
		t.Errorf("1 行目 = %q, want キーバインド", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "取得中") {
		t.Errorf("最終行に 取得中 が無い: %q", lines[len(lines)-1])
	}
}
