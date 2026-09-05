package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/sugi-loop/internal/fetch"
	"github.com/SugiKent/sugi-loop/internal/model"
)

// plain は View から ANSI エスケープを除いた行を返す。
func plain(m Model) []string {
	return strings.Split(ansi.Strip(m.View().Content), "\n")
}

func plainText(m Model) string { return ansi.Strip(m.View().Content) }

// lineWith は sub を含む最初の行を返す。
func lineWith(lines []string, sub string) (string, bool) {
	for _, l := range lines {
		if strings.Contains(l, sub) {
			return l, true
		}
	}
	return "", false
}

// order は s の中で subs がこの順で現れるかを見る。
func order(t *testing.T, s string, subs ...string) {
	t.Helper()
	i := 0
	for _, sub := range subs {
		j := strings.Index(s[i:], sub)
		if j < 0 {
			t.Fatalf("%q に %q がこの順で現れない: %q", s, sub, subs)
		}
		i += j + len(sub)
	}
}

func TestHeaderCountsAndTime(t *testing.T) {
	m, _ := send(New(nil), fetchedMsg{res: exampleResult(t), at: at})

	header := plain(m)[0]

	for _, want := range []string{"[1]今やる 1", "[2]バックログ 1", "[3]進行中 0", "[4]異常 0", "↻ 12:04"} {
		if !strings.Contains(header, want) {
			t.Errorf("ヘッダに %q が無い: %q", want, header)
		}
	}
}

func TestHeaderShortensTabNamesFromRight(t *testing.T) {
	m, _ := send(New(nil), fetchedMsg{res: exampleResult(t), at: at}, tea.WindowSizeMsg{Width: 60, Height: 40})

	header := plain(m)[0]

	for _, want := range []string{"[1]今やる 1", "[2] 1", "[3] 0", "[4] 0", "↻ 12:04"} {
		if !strings.Contains(header, want) {
			t.Errorf("ヘッダに %q が無い: %q", want, header)
		}
	}
	for _, ng := range []string{"バックログ", "進行中", "異常"} {
		if strings.Contains(header, ng) {
			t.Errorf("短縮後のヘッダに %q が残っている: %q", ng, header)
		}
	}
}

func TestHeaderBeforeFirstFetch(t *testing.T) {
	header := plain(New(nil))[0]

	if !strings.Contains(header, "sugi-loop") {
		t.Errorf("ヘッダにアプリ名が無い: %q", header)
	}
	if !strings.Contains(header, "[1]今やる 0") || !strings.Contains(header, "↻ --:--") {
		t.Errorf("初回取得前のヘッダが想定と違う: %q", header)
	}
}

func TestFooterShowsOnlyImplementedKeys(t *testing.T) {
	lines := plain(New(nil))
	footer := lines[len(lines)-1]

	for _, want := range []string{"j/k 移動", "1-4/Tab タブ", "q 終了"} {
		if !strings.Contains(footer, want) {
			t.Errorf("フッタに %q が無い: %q", want, footer)
		}
	}
	for _, ng := range []string{"a 回答", "m merge", "Enter 開く"} {
		if strings.Contains(footer, ng) {
			t.Errorf("フッタに未実装のキー %q がある: %q", ng, footer)
		}
	}
}

func TestRowOfQuestionCard(t *testing.T) {
	res := exampleResult(t)
	m, _ := send(New(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: res, at: at})

	line, ok := lineWith(plain(m), "PR131")
	if !ok {
		t.Fatalf("PR131 の行が無い: %q", plainText(m))
	}
	order(t, line, "!!", "質問", "org/app", "PR131", cardOf(t, res, 108).PRs[0].Title)
}

func TestRowOfBacklogCard(t *testing.T) {
	m, _ := send(New(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: exampleResult(t), at: at}, runeKey('2'))

	line, ok := lineWith(plain(m), "#140")
	if !ok {
		t.Fatalf("#140 の行が無い: %q", plainText(m))
	}
	order(t, line, "-", "todo 候補", "org/app", "#140", "設定ファイル未作成でクラッシュする")
}

func TestLongTitleIsTruncated(t *testing.T) {
	card := nowCard("org/app", 1, 1, at)
	card.Issue.Title = strings.Repeat("長いタイトル", 40)
	m, _ := send(New(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: &fetch.Result{Cards: []model.Card{card}}, at: at})

	line, ok := lineWith(plain(m), "長いタイトル")
	if !ok {
		t.Fatal("タイトルの行が無い")
	}
	if ansi.StringWidth(line) > 120 {
		t.Errorf("行が端末幅を超えた: %d 列", ansi.StringWidth(line))
	}
	if !strings.Contains(line, "…") {
		t.Errorf("切り詰めの … が無い: %q", line)
	}
}

func TestSelectedRowHasMarker(t *testing.T) {
	cards := threeNowCards()[:2]
	m, _ := send(New(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})

	lines := plain(m)
	first, second := lines[1], lines[2]

	if !strings.HasPrefix(first, "▶") {
		t.Errorf("選択行に ▶ が無い: %q", first)
	}
	if strings.HasPrefix(second, "▶") {
		t.Errorf("非選択行に ▶ がある: %q", second)
	}
}

func TestElapsedColumnUsesFetchTime(t *testing.T) {
	fetchedAt := time.Date(2026, 9, 5, 12, 4, 0, 0, time.UTC)
	card := nowCard("org/app", 7, 1, time.Date(2026, 9, 5, 9, 0, 0, 0, time.UTC))
	m, _ := send(New(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: &fetch.Result{Cards: []model.Card{card}}, at: fetchedAt})

	line, ok := lineWith(plain(m), "#7")
	if !ok {
		t.Fatal("#7 の行が無い")
	}
	if !strings.Contains(line, "3h") {
		t.Errorf("経過が 3h でない: %q", line)
	}
}

func TestFooterShowsSpinnerBeforeFirstFetch(t *testing.T) {
	m := New(nil)
	m.Init()

	lines := plain(m)

	if got := len(lines); got != 4 {
		t.Errorf("初回取得前の行数 = %d, want 4（ヘッダ / 区切り / プレビュー / フッタ）: %q", got, lines)
	}
	if !strings.Contains(lines[len(lines)-1], "取得中") {
		t.Errorf("フッタに 取得中 が無い: %q", lines[len(lines)-1])
	}
}

func TestFooterAfterSuccessHasNoSpinner(t *testing.T) {
	m, _ := send(New(nil), fetchedMsg{res: exampleResult(t), at: at})

	if strings.Contains(plainText(m), "取得中") {
		t.Errorf("取得完了後に 取得中 が残っている: %q", plainText(m))
	}
}

func TestFooterShowsFetchError(t *testing.T) {
	m, _ := send(New(nil),
		fetchedMsg{res: exampleResult(t), at: at},
		fetchedMsg{err: errors.New("search issues: gh search issues: exit 1: rate limited"), at: at.Add(time.Hour)})

	lines := plain(m)
	footer := lines[len(lines)-1]

	if _, ok := lineWith(lines, "PR131"); !ok {
		t.Error("失敗後に前回の行が消えた")
	}
	if !strings.Contains(lines[0], "↻ 12:04") {
		t.Errorf("失敗後にヘッダの時刻が変わった: %q", lines[0])
	}
	if !strings.Contains(footer, "rate limited") {
		t.Errorf("フッタにエラーが無い: %q", footer)
	}
	if strings.Contains(footer, "取得中") {
		t.Errorf("失敗後もスピナーが出ている: %q", footer)
	}
}

func TestFooterShowsPartialFailure(t *testing.T) {
	res := &fetch.Result{
		Cards: []model.Card{nowCard("org/app", 1, 1, at)},
		Errors: []error{
			errors.New("ViewPR org/app#131: open pr-131.json: no such file"),
			errors.New("ViewIssue org/app#108: open issue-108.json: no such file"),
		},
	}
	m, _ := send(New(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: res, at: at})

	lines := plain(m)
	if _, ok := lineWith(lines, "#1"); !ok {
		t.Error("部分失敗で Card の行が消えた")
	}
	if !strings.Contains(lines[len(lines)-1], "詳細取得の失敗 2 件: ViewPR org/app#131") {
		t.Errorf("フッタに部分失敗の件数が無い: %q", lines[len(lines)-1])
	}
}

func TestTwoPaneShowsTableAndPreview(t *testing.T) {
	m, _ := send(New(nil), fetchedMsg{res: exampleResult(t), at: at}, tea.WindowSizeMsg{Width: 120, Height: 40})

	text := plainText(m)

	if !strings.Contains(text, "PR131") || !strings.Contains(text, "issue #108 の提案") {
		t.Errorf("2 ペインに表とプレビューの両方が出ていない: %q", text)
	}
}

func TestNarrowTerminalTogglesWithP(t *testing.T) {
	m, _ := send(New(nil), fetchedMsg{res: exampleResult(t), at: at}, tea.WindowSizeMsg{Width: 60, Height: 40})

	text := plainText(m)
	if !strings.Contains(text, "PR131") || strings.Contains(text, "issue #108 の提案") {
		t.Errorf("狭い端末で表だけになっていない: %q", text)
	}
	if !strings.Contains(text, "p プレビュー") {
		t.Errorf("フッタに p プレビュー が無い: %q", text)
	}

	m, _ = send(m, runeKey('p'))

	text = plainText(m)
	if strings.Contains(text, "PR131") || !strings.Contains(text, "issue #108 の提案") {
		t.Errorf("p でプレビューに切り替わっていない: %q", text)
	}
	if !strings.Contains(text, "p 一覧") {
		t.Errorf("フッタに p 一覧 が無い: %q", text)
	}
}

func TestShortTerminalIsOnePane(t *testing.T) {
	m, _ := send(New(nil), fetchedMsg{res: exampleResult(t), at: at}, tea.WindowSizeMsg{Width: 120, Height: 15})

	text := plainText(m)

	if !strings.Contains(text, "PR131") || strings.Contains(text, "issue #108 の提案") {
		t.Errorf("低い端末で 1 ペインになっていない: %q", text)
	}
}

func TestPDoesNothingInTwoPane(t *testing.T) {
	m, _ := send(New(nil), fetchedMsg{res: exampleResult(t), at: at}, tea.WindowSizeMsg{Width: 120, Height: 40})
	before := plainText(m)

	after, cmd := send(m, runeKey('p'))

	if cmd != nil {
		t.Errorf("2 ペインの p でコマンドが返った: %T", cmd())
	}
	if plainText(after) != before {
		t.Errorf("2 ペインの p で View が変わった:\n%q\n%q", before, plainText(after))
	}
}
