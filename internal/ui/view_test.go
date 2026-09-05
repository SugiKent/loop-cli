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
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at})

	header := plain(m)[0]

	for _, want := range []string{"[1]今やる 1", "[2]バックログ 1", "[3]進行中 0", "[4]異常 0", "↻ 12:04"} {
		if !strings.Contains(header, want) {
			t.Errorf("ヘッダに %q が無い: %q", want, header)
		}
	}
}

func TestHeaderShortensTabNamesFromRight(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at}, tea.WindowSizeMsg{Width: 60, Height: 40})

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
	header := plain(newModel(nil))[0]

	if !strings.Contains(header, "sugi-loop") {
		t.Errorf("ヘッダにアプリ名が無い: %q", header)
	}
	if !strings.Contains(header, "[1]今やる 0") || !strings.Contains(header, "↻ --:--") {
		t.Errorf("初回取得前のヘッダが想定と違う: %q", header)
	}
}

func TestFooterShowsOnlyImplementedKeys(t *testing.T) {
	lines := plain(newModel(nil))
	footer := lines[len(lines)-1]

	order(t, footer, "Enter 開く", "a 回答", "t todo", "o ブラウザ", "R 更新", "? ヘルプ", "q 終了")
	for _, ng := range []string{"j/k 移動", "1-4/Tab タブ", "m merge", "Esc 戻る"} {
		if strings.Contains(footer, ng) {
			t.Errorf("フッタに出さないキー %q がある: %q", ng, footer)
		}
	}
}

// TestFooterFitsHintAndSpinnerAtWidth80 は既定幅 80 の初回取得中でもヒントが消えないことを検証する。
// s01 tui-entrypoint の「初期フレームに q 終了」が取得中でも成り立つ幅の担保である。
func TestFooterFitsHintAndSpinnerAtWidth80(t *testing.T) {
	lines := plain(newModel(nil))
	footer := lines[len(lines)-1]

	for _, want := range []string{"Enter 開く", "q 終了", "取得中"} {
		if !strings.Contains(footer, want) {
			t.Errorf("幅 80 の取得中のフッタに %q が無い: %q", want, footer)
		}
	}
}

func TestRowOfQuestionCard(t *testing.T) {
	res := exampleResult(t)
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: res, at: at})

	line, ok := lineWith(plain(m), "PR131")
	if !ok {
		t.Fatalf("PR131 の行が無い: %q", plainText(m))
	}
	order(t, line, "!!", "質問", "org/app", "PR131", cardOf(t, res, 108).PRs[0].Title)
}

func TestRowOfBacklogCard(t *testing.T) {
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: exampleResult(t), at: at}, runeKey('2'))

	line, ok := lineWith(plain(m), "#140")
	if !ok {
		t.Fatalf("#140 の行が無い: %q", plainText(m))
	}
	order(t, line, "-", "todo 候補", "org/app", "#140", "設定ファイル未作成でクラッシュする")
}

func TestLongTitleIsTruncated(t *testing.T) {
	card := nowCard("org/app", 1, 1, at)
	card.Issue.Title = strings.Repeat("長いタイトル", 40)
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: &fetch.Result{Cards: []model.Card{card}}, at: at})

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
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})

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
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: &fetch.Result{Cards: []model.Card{card}}, at: fetchedAt})

	line, ok := lineWith(plain(m), "#7")
	if !ok {
		t.Fatal("#7 の行が無い")
	}
	if !strings.Contains(line, "3h") {
		t.Errorf("経過が 3h でない: %q", line)
	}
}

func TestFooterShowsSpinnerBeforeFirstFetch(t *testing.T) {
	m := newModel(nil)
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
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at})

	if strings.Contains(plainText(m), "取得中") {
		t.Errorf("取得完了後に 取得中 が残っている: %q", plainText(m))
	}
}

func TestFooterShowsFetchError(t *testing.T) {
	m, _ := send(newModel(nil),
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
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: res, at: at})

	lines := plain(m)
	if _, ok := lineWith(lines, "#1"); !ok {
		t.Error("部分失敗で Card の行が消えた")
	}
	if !strings.Contains(lines[len(lines)-1], "詳細取得の失敗 2 件: ViewPR org/app#131") {
		t.Errorf("フッタに部分失敗の件数が無い: %q", lines[len(lines)-1])
	}
}

func TestTwoPaneShowsTableAndPreview(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at}, tea.WindowSizeMsg{Width: 120, Height: 40})

	text := plainText(m)

	if !strings.Contains(text, "PR131") || !strings.Contains(text, "issue #108 の提案") {
		t.Errorf("2 ペインに表とプレビューの両方が出ていない: %q", text)
	}
}

func TestNarrowTerminalStillShowsBothPanes(t *testing.T) {
	// 幅 79 は以前の 2 ペインの下限（80）を下回るが、常に 2 ペインで描く。
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at}, tea.WindowSizeMsg{Width: 79, Height: 40})

	text := plainText(m)
	if !strings.Contains(text, "PR131") || !strings.Contains(text, "issue #108 の提案") {
		t.Errorf("狭い端末で表とプレビューの両方が出ていない: %q", text)
	}
	if strings.Contains(text, "p プレビュー") || strings.Contains(text, "p 一覧") {
		t.Errorf("フッタに p のヒントが残っている: %q", text)
	}
}

func TestShortTerminalStillShowsBothPanes(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at}, tea.WindowSizeMsg{Width: 120, Height: 15})

	text := plainText(m)

	if !strings.Contains(text, "PR131") || !strings.Contains(text, "issue #108 の提案") {
		t.Errorf("低い端末で表とプレビューの両方が出ていない: %q", text)
	}
}

func TestPDoesNothing(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 79, Height: 40}} {
		m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at}, size)
		before := plainText(m)

		after, cmd := send(m, runeKey('p'))

		if cmd != nil {
			t.Errorf("幅 %d の p でコマンドが返った: %T", size.Width, cmd())
		}
		if plainText(after) != before {
			t.Errorf("幅 %d の p で View が変わった:\n%q\n%q", size.Width, before, plainText(after))
		}
	}
}

// hintLine2 は空キューのヒントの 2 行目。
const hintLine2 = "issue-driven-sdd の routines-setup を回したリポジトリを設定してください"

func TestEmptyQueueShowsHint(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: &fetch.Result{}, at: at})

	text := plainText(m)
	for _, want := range []string{"stage:* ラベルの無いリポジトリは何も出ません。", hintLine2} {
		if !strings.Contains(text, want) {
			t.Errorf("画面に %q が無い:\n%s", want, text)
		}
	}
	if !strings.Contains(plain(m)[0], "[1]今やる 0") {
		t.Errorf("ヘッダが従来どおりでない: %q", plain(m)[0])
	}
	if !strings.Contains(text, "q 終了") {
		t.Error("フッタのキーヒントが無い")
	}
}

func TestFetchingShowsNoHint(t *testing.T) {
	text := plainText(newModel(nil))

	if strings.Contains(text, "routines-setup") {
		t.Errorf("取得中にヒントが出ている:\n%s", text)
	}
	if !strings.Contains(text, "取得中") {
		t.Error("フッタに 取得中 が無い")
	}
}

func TestFetchErrorShowsNoHint(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{err: errors.New("search issues: gh search issues: exit 1: rate limited")})

	text := plainText(m)
	if strings.Contains(text, "routines-setup") {
		t.Errorf("取得失敗でヒントが出ている:\n%s", text)
	}
	if !strings.Contains(text, "rate limited") {
		t.Error("フッタに取得失敗が無い")
	}
}

func TestCardsPresentShowNoHint(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at})

	text := plainText(m)
	if strings.Contains(text, "routines-setup") {
		t.Errorf("Card があるのにヒントが出ている:\n%s", text)
	}
	if _, ok := lineWith(plain(m), "PR131"); !ok {
		t.Error("表に PR131 の行が無い")
	}
}

func TestPartialFailureWithNoCardsShowsHint(t *testing.T) {
	res := &fetch.Result{Errors: []error{errors.New("ViewPR org/app#131: open pr-131.json: no such file")}}
	m, _ := send(newModel(nil), fetchedMsg{res: res, at: at})

	text := plainText(m)
	for _, want := range []string{"stage:* ラベルの無いリポジトリは何も出ません。", hintLine2} {
		if !strings.Contains(text, want) {
			t.Errorf("画面に %q が無い:\n%s", want, text)
		}
	}
	if !strings.Contains(text, "詳細取得の失敗 1 件") {
		t.Error("フッタに部分失敗が無い")
	}
}

func TestHintIsHorizontallyCentered(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: &fetch.Result{}, at: at}, tea.WindowSizeMsg{Width: 120, Height: 40})

	line, ok := lineWith(plain(m), "routines-setup")
	if !ok {
		t.Fatal("ヒントの行が無い")
	}
	want := (120 - ansi.StringWidth(strings.TrimSpace(line))) / 2
	if got := len(line) - len(strings.TrimLeft(line, " ")); got != want {
		t.Errorf("左の空白 = %d, want %d: %q", got, want, line)
	}
	if strings.HasSuffix(line, " ") {
		t.Errorf("行末に空白がある: %q", line)
	}
}

// TestEmptyTabShowsHintAndEmptyPreview は空タブで、表の領域にヒントが出て、
// 同時にプレビュー領域の空文言も出ることを検証する。
func TestEmptyTabShowsHintAndEmptyPreview(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: &fetch.Result{}, at: at},
		tea.WindowSizeMsg{Width: 60, Height: 40})

	lines := plain(m)
	hint, ok := indexOf(lines, "routines-setup")
	if !ok {
		t.Fatalf("ヒントの行が無い:\n%s", strings.Join(lines, "\n"))
	}
	sep, ok := indexOf(lines, "──")
	if !ok {
		t.Fatalf("区切り線が無い:\n%s", strings.Join(lines, "\n"))
	}
	empty, ok := indexOf(lines, "（このタブにはカードがありません）")
	if !ok {
		t.Fatalf("プレビュー領域の空文言が無い:\n%s", strings.Join(lines, "\n"))
	}
	if !(hint < sep && sep < empty) {
		t.Errorf("ヒント(%d) / 区切り線(%d) / 空文言(%d) の順序が違う:\n%s", hint, sep, empty, strings.Join(lines, "\n"))
	}
}

// indexOf は s を含む最初の行の添字を返す。
func indexOf(lines []string, s string) (int, bool) {
	for i, l := range lines {
		if strings.Contains(l, s) {
			return i, true
		}
	}
	return 0, false
}

// TestHintWidths はフッタのヒントが既定幅 80 に収まる設計どおりの表示幅であることを検証する。
// キューは 64 列（取得中でも 64 + 1 + 8 = 73 で両方出る）、カード詳細は 101 列だが
// `? ヘルプ` が 65 列目で終わるのでヘルプの入口は幅 80 でも見える。
func TestHintWidths(t *testing.T) {
	card := Model{screen: screenCard, detail: detailState{card: model.Card{PRs: []model.PR{{Number: 131}}}}}
	cases := map[string]struct {
		hint string
		want int
	}{
		"キュー":   {newModel(nil).queueHint(), 64},
		"PR 詳細": {Model{screen: screenPR}.detailHint(), 66},
		"カード詳細": {card.detailHint(), 101},
	}
	for name, tc := range cases {
		if got := ansi.StringWidth(tc.hint); got != tc.want {
			t.Errorf("%s のヒントの表示幅 = %d, want %d: %q", name, got, tc.want, tc.hint)
		}
	}

	hint := card.detailHint()
	if end := ansi.StringWidth(hint[:strings.Index(hint, "? ヘルプ")]) + ansi.StringWidth("? ヘルプ"); end != 65 {
		t.Errorf("カード詳細の `? ヘルプ` が %d 列目で終わる, want 65: %q", end, hint)
	}
}

// TestViewRequestsAltScreen は再描画でフレームが積み上がらないよう、
// View が alt screen を要求することを検証する。
func TestViewRequestsAltScreen(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at})

	if !m.View().AltScreen {
		t.Error("キュー画面の View が alt screen を要求していない")
	}

	m, _ = send(m, questionKey)
	if m.screen != screenHelp {
		t.Fatalf("画面 = %d, want ヘルプ", m.screen)
	}
	if !m.View().AltScreen {
		t.Error("ヘルプ画面の View が alt screen を要求していない")
	}
}
