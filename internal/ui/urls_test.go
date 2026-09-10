package ui

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

var uKey = runeKey('u')

// urlModel は Card 群を取得完了として渡した Model と、URL を開く先の Fake を返す。
func urlModel(cards []model.Card, width, height int) (Model, *gh.Fake) {
	fake := gh.NewFake(fixtureDir)
	ed := &stubEditor{msg: editedMsg{err: errors.New("使わない")}}
	m, _ := send(New(nil, fake, ed.Editor, Options{}),
		tea.WindowSizeMsg{Width: width, Height: height},
		fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})
	return m, fake
}

// urlPRCard は本文・コメント・review thread を指定した open PR 1 件だけの Card。
func urlPRCard(body string, comments []model.Comment, threads []gh.ReviewThread) model.Card {
	result := model.Result{Situation: model.SituationA, Priority: 1, Tab: model.TabNow, Summary: "回答する"}
	pr := model.PR{
		Repo: "org/app", Number: 131, Title: "手書き PR", State: "OPEN", UpdatedAt: at,
		Body: body, Comments: comments, ReviewThreads: threads, Result: result,
	}
	return model.Card{PRs: []model.PR{pr}, Result: result}
}

// prDetailWithURLs は PR 詳細を開いてから u を押した Model を返す。
func prDetailWithURLs(t *testing.T, card model.Card, width, height int) (Model, *gh.Fake) {
	t.Helper()
	m, fake := urlModel([]model.Card{card}, width, height)
	m, _ = send(m, codeKey(tea.KeyEnter))
	if m.screen != screenPR {
		t.Fatalf("PR 詳細に移っていない: screen = %d", m.screen)
	}
	m, cmd := send(m, uKey)
	if cmd != nil {
		t.Fatal("u でコマンドが返っている（gh を呼ばないはず）")
	}
	if m.screen != screenURL {
		t.Fatalf("URL 一覧に移っていない: screen = %d", m.screen)
	}
	return m, fake
}

// wantURLs は一覧の中身を出典 / テキスト / URL で確かめる。
func wantURLs(t *testing.T, got []urlItem, want []urlItem) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("一覧 = %+v, want %+v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%d 件目 = %+v, want %+v", i, got[i], want[i])
		}
	}
}

const designLink = "設計は [設計メモ](https://example.com/design) を見る"

// TestURLKeyOpensListForScreenTarget は u の対象が画面の見せているものに決まることを検証する。
func TestURLKeyOpensListForScreenTarget(t *testing.T) {
	t.Run("PR 詳細は選択中の PR", func(t *testing.T) {
		m, _ := prDetailWithURLs(t, urlPRCard(designLink, nil, nil), 80, 24)

		wantURLs(t, m.urls.items, []urlItem{{source: "本文", text: "設計メモ", url: "https://example.com/design"}})
		if m.urls.cursor != 0 {
			t.Errorf("選択位置 = %d, want 0", m.urls.cursor)
		}
	})

	t.Run("キュー画面は選択行の主体", func(t *testing.T) {
		m, _ := urlModel([]model.Card{urlPRCard(designLink, nil, nil)}, 80, 24)

		m, cmd := send(m, uKey)

		if cmd != nil {
			t.Error("u でコマンドが返っている")
		}
		if m.screen != screenURL {
			t.Fatalf("画面 = %d, want URL 一覧", m.screen)
		}
		wantURLs(t, m.urls.items, []urlItem{{source: "本文", text: "設計メモ", url: "https://example.com/design"}})
	})

	t.Run("カード詳細は Issue から集める", func(t *testing.T) {
		card := urlPRCard("https://example.com/pr", nil, nil)
		card.Issue = &model.Issue{
			Repo: "org/app", Number: 108, Title: "手書き", UpdatedAt: at,
			Body: "https://example.com/issue", Result: card.Result,
		}
		m, _ := urlModel([]model.Card{card}, 80, 24)
		m, _ = send(m, codeKey(tea.KeyEnter))
		if m.screen != screenCard {
			t.Fatalf("カード詳細に移っていない: screen = %d", m.screen)
		}

		m, _ = send(m, uKey)

		wantURLs(t, m.urls.items, []urlItem{{source: "本文", url: "https://example.com/issue"}})
	})
}

// TestURLKeyWithoutURLsShowsStatus は URL が 0 件のときに画面を変えずステータスを出すことを検証する。
func TestURLKeyWithoutURLsShowsStatus(t *testing.T) {
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40},
		fetchedMsg{res: exampleResult(t), at: at})

	m, cmd := send(m, uKey)

	if cmd != nil {
		t.Error("u でコマンドが返っている")
	}
	if m.screen != screenQueue {
		t.Errorf("画面 = %d, want キュー", m.screen)
	}
	if text := plainText(m); !strings.Contains(text, "URL がありません") {
		t.Errorf("`URL がありません` が出ていない: %q", footerOf(text))
	}
	if m.writeStatusErr {
		t.Error("`URL がありません` を赤で出している")
	}
}

// TestURLKeyDoesNothingOnOtherScreens は 0 行のタブ・確認画面・ヘルプ画面で u が何もしないことを検証する。
func TestURLKeyDoesNothingOnOtherScreens(t *testing.T) {
	check := func(t *testing.T, m Model, want screen) {
		t.Helper()
		m, cmd := send(m, uKey)
		if cmd != nil {
			t.Error("u でコマンドが返っている")
		}
		if m.screen != want {
			t.Errorf("画面 = %d, want %d", m.screen, want)
		}
		if text := plainText(m); strings.Contains(text, "URL がありません") {
			t.Errorf("`URL がありません` が出ている: %q", footerOf(text))
		}
	}

	t.Run("0 行のタブ", func(t *testing.T) {
		m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40},
			fetchedMsg{res: exampleResult(t), at: at}, runeKey('4'))
		if len(m.rows[m.tab]) != 0 {
			t.Fatalf("異常タブの行数 = %d, want 0", len(m.rows[m.tab]))
		}
		check(t, m, screenQueue)
	})

	t.Run("確認画面", func(t *testing.T) {
		m, _ := confirmModel(t)
		check(t, m, screenConfirm)
	})

	t.Run("ヘルプ画面", func(t *testing.T) {
		m, _, _ := helpModel(t)
		m, _ = send(m, runeKey('?'))
		if m.screen != screenHelp {
			t.Fatalf("ヘルプ画面に移っていない: screen = %d", m.screen)
		}
		check(t, m, screenHelp)
	})
}

// TestCollectURLsOrderAndDedup は収集元の順序・出典・重複の除去・取得失敗の扱いを検証する。
func TestCollectURLsOrderAndDedup(t *testing.T) {
	t.Run("本文・コメント・thread の順に並び出典が付く", func(t *testing.T) {
		card := urlPRCard(
			"https://example.com/body",
			[]model.Comment{{Body: "[CI](https://ci.example.com/build/42)"}},
			[]gh.ReviewThread{{Comments: []gh.ReviewComment{{Body: "https://example.com/thread"}}}},
		)
		m, _ := prDetailWithURLs(t, card, 80, 24)

		wantURLs(t, m.urls.items, []urlItem{
			{source: "本文", url: "https://example.com/body"},
			{source: "コメント", text: "CI", url: "https://ci.example.com/build/42"},
			{source: "thread", url: "https://example.com/thread"},
		})
	})

	t.Run("重複した URL は初出だけ残る", func(t *testing.T) {
		card := urlPRCard(
			"[設計メモ](https://example.com/design)",
			[]model.Comment{{Body: "https://example.com/design も参照"}},
			nil,
		)
		m, _ := prDetailWithURLs(t, card, 80, 24)

		wantURLs(t, m.urls.items, []urlItem{{source: "本文", text: "設計メモ", url: "https://example.com/design"}})
	})

	t.Run("取得に失敗したコメントは飛ばす", func(t *testing.T) {
		m, _ := prDetailWithURLs(t, urlPRCard("https://example.com/body", nil, nil), 80, 24)

		wantURLs(t, m.urls.items, []urlItem{{source: "本文", url: "https://example.com/body"}})
	})
}

// twoURLCard は本文とコメントに URL を 1 件ずつ持つ Card。
func twoURLCard() model.Card {
	return urlPRCard(
		"[設計メモ](https://example.com/design)",
		[]model.Comment{{Body: "https://ci.example.com/build/42"}},
		nil,
	)
}

// TestURLListSelection は j / k で選択が動き、端では止まることを検証する。
func TestURLListSelection(t *testing.T) {
	card := urlPRCard("https://example.com/0 https://example.com/1 https://example.com/2", nil, nil)
	m, _ := prDetailWithURLs(t, card, 80, 24)

	want := []int{1, 2, 2, 1}
	for i, key := range []tea.KeyPressMsg{runeKey('j'), runeKey('j'), runeKey('j'), runeKey('k')} {
		var cmd tea.Cmd
		m, cmd = send(m, key)
		if cmd != nil {
			t.Errorf("%d 回目でコマンドが返っている", i+1)
		}
		if m.urls.cursor != want[i] {
			t.Errorf("%d 回目の選択位置 = %d, want %d", i+1, m.urls.cursor, want[i])
		}
	}
}

// TestURLListEnterOpensSelected は Enter が選択中の URL を開き、一覧に留まることを検証する。
func TestURLListEnterOpensSelected(t *testing.T) {
	card := urlPRCard("https://example.com/0 https://example.com/1 https://example.com/2", nil, nil)
	m, fake := prDetailWithURLs(t, card, 80, 24)
	m, _ = send(m, runeKey('j'))

	m, cmd := send(m, codeKey(tea.KeyEnter))
	if len(fake.Calls) != 0 {
		t.Errorf("コマンドの実行前に開いている: %+v", fake.Calls)
	}
	m, _ = runCmd(t, m, cmd)

	want := gh.Call{Method: "OpenURL", URL: "https://example.com/1"}
	if len(fake.Calls) != 1 || !reflect.DeepEqual(fake.Calls[0], want) {
		t.Fatalf("呼び出し = %+v, want [%+v]", fake.Calls, want)
	}
	if m.screen != screenURL {
		t.Errorf("画面 = %d, want URL 一覧", m.screen)
	}
	if m.urls.cursor != 1 {
		t.Errorf("選択位置 = %d, want 1", m.urls.cursor)
	}
	if text := plainText(m); strings.Contains(text, "開けません") {
		t.Errorf("成功で何か出ている: %q", footerOf(text))
	}
}

// TestURLListEscReturns は Esc が戻り先の画面と対象を保つことを検証する。
func TestURLListEscReturns(t *testing.T) {
	card := twoURLCard()
	card.Issue = &model.Issue{Repo: "org/app", Number: 108, Title: "手書き", UpdatedAt: at, Result: card.Result}
	m, _ := urlModel([]model.Card{card}, 80, 24)
	m, _ = send(m, codeKey(tea.KeyEnter), codeKey(tea.KeyEnter))
	if m.screen != screenPR {
		t.Fatalf("PR 詳細に移っていない: screen = %d", m.screen)
	}
	m, _ = send(m, uKey)

	m, cmd := send(m, codeKey(tea.KeyEscape))

	if cmd != nil {
		t.Error("Esc でコマンドが返っている")
	}
	if m.screen != screenPR {
		t.Fatalf("画面 = %d, want PR 詳細", m.screen)
	}
	if m.currentPR().Number != 131 {
		t.Errorf("対象 = PR#%d, want PR#131", m.currentPR().Number)
	}
}

// TestURLListIgnoresOtherKeys は一覧画面で他のキーが何もしないことを検証する。
func TestURLListIgnoresOtherKeys(t *testing.T) {
	fake := gh.NewFake(fixtureDir)
	ed := &stubEditor{msg: editedMsg{err: errors.New("使わない")}}
	m, _ := send(New(nil, fake, ed.Editor, Options{}),
		tea.WindowSizeMsg{Width: 80, Height: 24},
		fetchedMsg{res: &fetch.Result{Cards: []model.Card{twoURLCard()}}, at: at},
		codeKey(tea.KeyEnter), uKey)
	if m.screen != screenURL {
		t.Fatalf("URL 一覧に移っていない: screen = %d", m.screen)
	}
	before := m.urls

	for _, r := range []rune{'u', 'a', 't', 'n', 'o', '?', 'R', 'x', 'g', 'p', '2'} {
		var cmd tea.Cmd
		m, cmd = send(m, runeKey(r))
		if cmd != nil {
			t.Errorf("%c でコマンドが返っている", r)
		}
		if m.screen != screenURL {
			t.Errorf("%c で画面が %d に変わった", r, m.screen)
		}
		if m.urls.cursor != before.cursor || m.urls.from != before.from {
			t.Errorf("%c で選択位置か戻り先が変わった: %+v", r, m.urls)
		}
	}
	if len(fake.Calls) != 0 {
		t.Errorf("gh を呼んでいる: %+v", fake.Calls)
	}
	if ed.calls != 0 {
		t.Errorf("エディタを %d 回呼んでいる, want 0", ed.calls)
	}
}

// TestURLListQuits は一覧画面でも q が終了することを検証する。
func TestURLListQuits(t *testing.T) {
	m, _ := prDetailWithURLs(t, twoURLCard(), 80, 24)

	_, cmd := send(m, runeKey('q'))

	if cmd == nil {
		t.Fatal("q でコマンドが返っていない")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("q のメッセージ = %T, want tea.QuitMsg", cmd())
	}
}

// TestURLListView は見出し・行の形・選択の印・フッタを検証する。
func TestURLListView(t *testing.T) {
	m, _ := prDetailWithURLs(t, twoURLCard(), 80, 24)

	lines := plain(m)
	if lines[0] != "URL を開く" {
		t.Errorf("1 行目 = %q, want URL を開く", lines[0])
	}
	design := indexOfLine(t, lines, "https://example.com/design")
	ci := indexOfLine(t, lines, "https://ci.example.com/build/42")
	if design >= ci {
		t.Errorf("本文の行 (%d) がコメントの行 (%d) より後ろにある", design, ci)
	}
	if want := "▶ [本文] 設計メモ  https://example.com/design"; !strings.HasPrefix(lines[design], want) {
		t.Errorf("本文の行 = %q, want %q で始まる", lines[design], want)
	}
	if want := "  [コメント]  https://ci.example.com/build/42"; !strings.HasPrefix(lines[ci], want) {
		t.Errorf("コメントの行 = %q, want %q で始まる", lines[ci], want)
	}

	footer := lines[len(lines)-1]
	for _, want := range []string{"j/k 選択", "Enter 開く", "Esc 戻る", "q 終了"} {
		if !strings.Contains(footer, want) {
			t.Errorf("フッタ %q に %q が含まれない", footer, want)
		}
	}

	// 選択が下に動くと印も動く。
	m, _ = send(m, runeKey('j'))
	lines = plain(m)
	if l := lines[indexOfLine(t, lines, "https://ci.example.com/build/42")]; !strings.HasPrefix(l, "▶ ") {
		t.Errorf("選択行 = %q, want ▶ で始まる", l)
	}
	if l := lines[indexOfLine(t, lines, "https://example.com/design")]; strings.Contains(l, "▶") {
		t.Errorf("非選択行 = %q, want ▶ を含まない", l)
	}
}

// TestURLListViewFollowsSelection は件数が高さを超えるとき表示が選択に追従することを検証する。
func TestURLListViewFollowsSelection(t *testing.T) {
	var urls []string
	for i := range 30 {
		urls = append(urls, "https://example.com/"+strconv.Itoa(i))
	}
	m, _ := prDetailWithURLs(t, urlPRCard(strings.Join(urls, "\n"), nil, nil), 80, 10)
	if len(m.urls.items) != 30 {
		t.Fatalf("一覧の件数 = %d, want 30", len(m.urls.items))
	}

	text := plainText(m)
	if !strings.Contains(text, "https://example.com/0\n") {
		t.Error("先頭が出ていない")
	}
	if strings.Contains(text, "https://example.com/7") {
		t.Error("表示できる 7 行を超えて出ている")
	}

	for range 10 {
		m, _ = send(m, runeKey('j'))
	}

	lines := plain(m)
	if l := lines[indexOfLine(t, lines, "https://example.com/10")]; !strings.HasPrefix(l, "▶ ") {
		t.Errorf("選択行 = %q, want ▶ で始まる", l)
	}
	if strings.Contains(plainText(m), "https://example.com/0\n") {
		t.Error("表示が選択に追従していない（先頭が残っている）")
	}
}

// failingOpen は OpenURL だけが失敗する GHClient。
type failingOpen struct {
	*gh.Fake
	err error
}

func (f failingOpen) OpenURL(context.Context, string) error { return f.err }

// TestOpenURLFailureShowsStatus は開けなかったときにフッタへ赤で出ることを検証する。
func TestOpenURLFailureShowsStatus(t *testing.T) {
	client := failingOpen{
		Fake: gh.NewFake(fixtureDir),
		err:  errors.New("open https://example.com/design: exit 1: no browser"),
	}
	ed := &stubEditor{msg: editedMsg{err: errors.New("使わない")}}
	m, _ := send(New(nil, client, ed.Editor, Options{}),
		tea.WindowSizeMsg{Width: 120, Height: 40},
		fetchedMsg{res: &fetch.Result{Cards: []model.Card{twoURLCard()}}, at: at},
		codeKey(tea.KeyEnter), uKey)
	before := m.urls

	m, cmd := send(m, codeKey(tea.KeyEnter))
	m, _ = runCmd(t, m, cmd)

	text := plainText(m)
	for _, want := range []string{"https://example.com/design を開けません:", "no browser"} {
		if !strings.Contains(text, want) {
			t.Errorf("フッタ %q に %q が含まれない", footerOf(text), want)
		}
	}
	if !m.writeStatusErr {
		t.Error("失敗を赤で出していない")
	}
	if m.screen != screenURL {
		t.Errorf("画面 = %d, want URL 一覧", m.screen)
	}
	if m.urls.cursor != before.cursor || len(m.urls.items) != len(before.items) {
		t.Errorf("一覧が変わった: %+v", m.urls)
	}
}

// indexOfLine は sub を含む最初の行の添字を返す。
func indexOfLine(t *testing.T, lines []string, sub string) int {
	t.Helper()
	for i, l := range lines {
		if strings.Contains(l, sub) {
			return i
		}
	}
	t.Fatalf("%q を含む行が無い:\n%s", sub, strings.Join(lines, "\n"))
	return -1
}
