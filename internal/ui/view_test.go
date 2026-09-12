package ui

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
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

	for _, want := range []string{"[1]今やる 1", "[2]バックログ 1", "[3] 0", "[4] 0", "↻ 12:04"} {
		if !strings.Contains(header, want) {
			t.Errorf("ヘッダに %q が無い: %q", want, header)
		}
	}
	for _, ng := range []string{"進行中", "異常"} {
		if strings.Contains(header, ng) {
			t.Errorf("短縮後のヘッダに %q が残っている: %q", ng, header)
		}
	}
}

func TestHeaderBeforeFirstFetch(t *testing.T) {
	header := plain(newModel(nil))[0]

	if !strings.Contains(header, "loop-cli") {
		t.Errorf("ヘッダにアプリ名が無い: %q", header)
	}
	if !strings.Contains(header, "[1]今やる 0") || !strings.Contains(header, "↻ --:--") {
		t.Errorf("初回取得前のヘッダが想定と違う: %q", header)
	}
}

func TestFooterShowsOnlyImplementedKeys(t *testing.T) {
	// ヒントは 99 列ちょうどなので、取得中のステータスと並ぶ幅で読む。
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 108, Height: 24})
	lines := plain(m)
	footer := lines[len(lines)-1]

	order(t, footer, "Enter 開く", "a 回答", "t todo", "L ラベル", "m merge", "c close", "o ブラウザ", "R 更新", "? ヘルプ", "u URL", "q 終了")
	// n はキュー画面で動くがヒントには出さない（足すと 88 列になり、幅 80 で `q 終了` が常に切れる）。
	for _, ng := range []string{"j/k 移動", "1-4/Tab タブ", "Esc 戻る", "n 新規"} {
		if strings.Contains(footer, ng) {
			t.Errorf("フッタに出さないキー %q がある: %q", ng, footer)
		}
	}
}

// TestFooterFitsHintAndSpinnerAtWidth90 は、キューのヒント（99 列）が取得中のステータスと
// 並ばない幅ではステータスが優先され、両方が入る幅では両方出ることを検証する（s08 のフッタの規則）。
// 名前の 90 は s14 時点のヒントの幅で、両方が入る最小の幅は 108 列になった。
func TestFooterFitsHintAndSpinnerAtWidth90(t *testing.T) {
	narrow := plain(newModel(nil))
	footer := narrow[len(narrow)-1]
	if !strings.Contains(footer, "取得中") {
		t.Errorf("幅 80 の取得中のフッタにステータスが無い: %q", footer)
	}
	if strings.Contains(footer, "Enter 開く") {
		t.Errorf("幅 80 でヒントとステータスが両方出ている: %q", footer)
	}

	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 108, Height: 24})
	wide := plain(m)
	footer = wide[len(wide)-1]
	for _, want := range []string{"Enter 開く", "L ラベル", "m merge", "c close", "q 終了", "取得中"} {
		if !strings.Contains(footer, want) {
			t.Errorf("幅 108 の取得中のフッタに %q が無い: %q", want, footer)
		}
	}

	// 1 列足りなければヒントは丸ごと消える（s08 のフッタはステータスを優先する）。
	m, _ = send(newModel(nil), tea.WindowSizeMsg{Width: 107, Height: 24})
	narrower := plain(m)
	footer = narrower[len(narrower)-1]
	if !strings.Contains(footer, "取得中") {
		t.Errorf("幅 107 の取得中のフッタにステータスが無い: %q", footer)
	}
	for _, ng := range []string{"Enter 開く", "c close"} {
		if strings.Contains(footer, ng) {
			t.Errorf("幅 107 でヒントとステータスが両方出ている: %q", footer)
		}
	}
}

// rowLines は Card 1 枚ぶんの表の行を ANSI エスケープを除いて返す。
func rowLines(m Model, tab model.Tab, i int) []string {
	rows := m.rows[tab]
	out := make([]string, 0, 2)
	for _, l := range m.tableRow(rows[i], i == m.cursor) {
		out = append(out, ansi.Strip(l))
	}
	return out
}

// titleColStart は spec が定めたタイトル列の開始位置。実装の定数ではなくリテラルで持つ
// （実装と一緒にずれて通るテストにしない）。
const titleColStart = 43

// titleColumns は各行のタイトル列（先頭から 43 列目以降、経過の列より左）を
// 末尾の空白を除いて連結する。折り返しても全文が出ていることの検証に使う。
func titleColumns(lines []string, width int) string {
	var s string
	for _, l := range lines {
		s += strings.TrimRight(ansi.Cut(l, titleColStart, width-colElapsed), " ")
	}
	return s
}

func TestRowOfQuestionCard(t *testing.T) {
	res := exampleResult(t)
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: res, at: at})

	lines := rowLines(m, model.TabNow, 0)

	order(t, lines[0], "!!", "質問", "org/app", "PR131")
	title := cardOf(t, res, 108).PRs[0].Title
	if got := titleColumns(lines, 120); got != title {
		t.Errorf("タイトル列の連結 = %q, want %q", got, title)
	}
}

func TestRowOfBacklogCard(t *testing.T) {
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: exampleResult(t), at: at}, runeKey('2'))

	lines := rowLines(m, model.TabBacklog, 0)

	order(t, lines[0], "-", "todo 候補", "org/app", "#140")
	if got := titleColumns(lines, 120); got != "設定ファイル未作成でクラッシュする" {
		t.Errorf("タイトル列の連結 = %q, want %q", got, "設定ファイル未作成でクラッシュする")
	}
}

// TestLongTitleIsTruncated は端末幅が 49 以下（タイトル列に全角 1 文字が入らない幅）のとき、
// タイトルを出さず 1 行だけを端末幅で切ることを検証する。
func TestLongTitleIsTruncated(t *testing.T) {
	for _, width := range []int{48, 49} {
		card := nowCard("org/app", 1, 1, at)
		card.Issue.Title = strings.Repeat("長いタイトル", 40)
		m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: width, Height: 40}, fetchedMsg{res: &fetch.Result{Cards: []model.Card{card}}, at: at})

		lines := rowLines(m, model.TabNow, 0)

		if len(lines) != 1 {
			t.Fatalf("幅 %d の行数 = %d, want 1: %q", width, len(lines), lines)
		}
		if strings.Contains(lines[0], "長い") {
			t.Errorf("幅 %d でタイトルが出ている: %q", width, lines[0])
		}
		if w := ansi.StringWidth(lines[0]); w > width {
			t.Errorf("幅 %d の行の表示幅 = %d: %q", width, w, lines[0])
		}
	}
}

// TestLongTitleWrapsToFullText はタイトル列に収まらないタイトルが折り返して全文出ることを検証する。
func TestLongTitleWrapsToFullText(t *testing.T) {
	card := nowCard("org/app", 1, 1, at)
	card.Issue.Title = strings.Repeat("あ", 40) // 表示幅 80 > タイトル列 32
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 80, Height: 40}, fetchedMsg{res: &fetch.Result{Cards: []model.Card{card}}, at: at})

	lines := rowLines(m, model.TabNow, 0)

	if len(lines) < 2 {
		t.Fatalf("行数 = %d, want 2 以上: %q", len(lines), lines)
	}
	for i, l := range lines {
		if w := ansi.StringWidth(l); w != 80 {
			t.Errorf("%d 行目の表示幅 = %d, want 80: %q", i+1, w, l)
		}
		if strings.Contains(l, "…") {
			t.Errorf("%d 行目に切り詰めの … がある: %q", i+1, l)
		}
	}
	if got := titleColumns(lines, 80); got != card.Issue.Title {
		t.Errorf("タイトル列の連結 = %q, want %q", got, card.Issue.Title)
	}
}

// TestWrappedRowAlignsAndKeepsElapsedOnFirstLine は継続行がタイトルの開始位置に揃い、
// 経過が 1 行目にだけ出ることを検証する。
func TestWrappedRowAlignsAndKeepsElapsedOnFirstLine(t *testing.T) {
	card := nowCard("org/app", 1, 1, at.Add(-12*time.Minute))
	card.Issue.Title = strings.Repeat("あ", 30) // 表示幅 60 > タイトル列 32
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 80, Height: 40}, fetchedMsg{res: &fetch.Result{Cards: []model.Card{card}}, at: at})

	lines := rowLines(m, model.TabNow, 0)

	if len(lines) != 2 {
		t.Fatalf("行数 = %d, want 2: %q", len(lines), lines)
	}
	if !strings.Contains(lines[0], "12m") {
		t.Errorf("1 行目に経過が無い: %q", lines[0])
	}
	if strings.Contains(lines[1], "12m") {
		t.Errorf("継続行に経過がある: %q", lines[1])
	}
	if head := ansi.Cut(lines[1], 0, titleColStart); strings.TrimSpace(head) != "" {
		t.Errorf("継続行の先頭 43 列が空白でない: %q", head)
	}
	if got := ansi.Cut(lines[1], titleColStart, titleColStart+2); got != "あ" {
		t.Errorf("継続行の 44 列目からタイトルの続きが始まっていない: %q", got)
	}
}

// TestSelectedRowHasMarker は View 全体を読み、折り返した継続行が表の領域に載ること
// （tableLines が行を平らに並べること）と、▶ が 1 枚目の 1 行目にだけ付くことを検証する。
func TestSelectedRowHasMarker(t *testing.T) {
	cards := threeNowCards()[:2]
	cards[0].Issue.Title = strings.Repeat("あ", 30) // 幅 80 で 2 行になる
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 80, Height: 40}, fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})

	lines := plain(m)
	// 1 行目はヘッダ。表は 2 行目から始まり、1 枚目が 2 行・2 枚目が 1 行を占める。
	first, cont, second := lines[1], lines[2], lines[3]

	if !strings.HasPrefix(first, "▶") {
		t.Errorf("選択行の 1 行目に ▶ が無い: %q", first)
	}
	if strings.HasPrefix(cont, "▶") {
		t.Errorf("選択行の継続行に ▶ がある: %q", cont)
	}
	if strings.HasPrefix(second, "▶") {
		t.Errorf("非選択行に ▶ がある: %q", second)
	}
	// 継続行が表の領域に載っており、1 枚目のタイトルが 2 行で全文出ている。
	if got := titleColumns([]string{first, cont}, 80); got != cards[0].Issue.Title {
		t.Errorf("表に出た 1 枚目のタイトル = %q, want %q", got, cards[0].Issue.Title)
	}
	sep := strings.Repeat("─", 80)
	if i := slices.Index(lines, sep); i < 4 {
		t.Errorf("区切り線の位置 = %d, want 4 以上（継続行が表の領域に無い）:\n%s", i, strings.Join(lines, "\n"))
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

// hintLine1 / hintLine2 は空キューのヒントの 2 行。既定幅 80 には収まらないので幅 120 で確かめる。
const hintLine1 = "stage:* / To Do ラベルの無いリポジトリは何も出ません。"

const hintLine2 = "issue-driven-sdd の routines-setup を回すか、issue-label-driven の To Do ラベルを作ってください"

func TestEmptyQueueShowsHint(t *testing.T) {
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: &fetch.Result{}, at: at})

	text := plainText(m)
	for _, want := range []string{hintLine1, hintLine2} {
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
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: res, at: at})

	text := plainText(m)
	for _, want := range []string{hintLine1, hintLine2} {
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
	if hint >= sep || sep >= empty {
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

// TestHintWidths はフッタのヒントが設計どおりの表示幅であることを検証する。
// キューは c close を足して 99 列、カード詳細は 144 列だが
// `? ヘルプ` が 65 列目、`u URL` が 72 列目で終わるのでヘルプと URL 一覧の入口は幅 80 でも見える。
func TestHintWidths(t *testing.T) {
	card := Model{screen: screenCard, detail: detailState{card: model.Card{PRs: []model.PR{{Number: 131}}}}}
	cases := map[string]struct {
		hint string
		want int
	}{
		"キュー":          {newModel(nil).queueHint(), 99},
		"PR 詳細":        {Model{screen: screenPR}.detailHint(), 109},
		"カード詳細":        {card.detailHint(), 144},
		"カード詳細（PR 無し）": {Model{screen: screenCard}.detailHint(), 96},
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
	if end := ansi.StringWidth(hint[:strings.Index(hint, "u URL")]) + ansi.StringWidth("u URL"); end != 72 {
		t.Errorf("カード詳細の `u URL` が %d 列目で終わる, want 72: %q", end, hint)
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

// --- queue-screen: 進行中タブの種別の列（段階ラベル名） ---

// kindColStart / kindColWidth は spec の列順が定める種別の列の位置と幅。実装の定数ではなく
// リテラルで持つ（実装と一緒にずれて通るテストにしない）。
const (
	kindColStart = 6
	kindColWidth = 10
)

// kindColumn は表の 1 行から種別の列を末尾の空白を除いて返す。
func kindColumn(line string) string {
	return strings.TrimRight(ansi.Cut(line, kindColStart, kindColStart+kindColWidth), " ")
}

// tableArea は表の領域の行（ヘッダの次から、表とプレビューの区切り線の手前まで）を返す。
func tableArea(m Model) []string {
	lines := plain(m)
	divider := strings.Repeat("─", m.width)
	for i, l := range lines[1:] {
		if l == divider {
			return lines[1 : 1+i]
		}
	}
	return lines[1:]
}

func TestInProgressKindColumnShowsStageLabel(t *testing.T) {
	cards := []model.Card{
		inProgressCard("org/app", 101, false, []string{"stage:archive", "wip"}, at.Add(-2*time.Hour)),
		inProgressCard("org/app", 102, true, []string{"propose"}, at.Add(-1*time.Hour)),
	}
	m := inProgressModel(t, 80, 24, &fetch.Result{Cards: cards})

	lines := tableArea(m)
	issueLine, ok := lineWith(lines, "#101")
	if !ok {
		t.Fatalf("issue の行が無い: %q", lines)
	}
	prLine, ok := lineWith(lines, "PR102")
	if !ok {
		t.Fatalf("PR の行が無い: %q", lines)
	}
	if got := kindColumn(issueLine); got != "archive" {
		t.Errorf("issue の種別の列 = %q, want %q", got, "archive")
	}
	if got := kindColumn(prLine); got != "propose" {
		t.Errorf("PR の種別の列 = %q, want %q", got, "propose")
	}
	for _, l := range []string{issueLine, prLine} {
		if strings.Contains(l, "進行中") {
			t.Errorf("進行中タブの行に種別 `進行中` が残っている: %q", l)
		}
	}
}

func TestInProgressKindColumnIsDashWithoutStageLabel(t *testing.T) {
	cards := []model.Card{inProgressCard("org/app", 101, false, []string{"question"}, at)}
	m := inProgressModel(t, 80, 24, &fetch.Result{Cards: cards})

	line, ok := lineWith(tableArea(m), "#101")
	if !ok {
		t.Fatal("行が無い")
	}
	if got := kindColumn(line); got != "-" {
		t.Errorf("段階ラベルの無い主体の種別の列 = %q, want %q", got, "-")
	}
}

func TestInProgressKindColumnShowsFirstOfTwoStages(t *testing.T) {
	cards := []model.Card{
		inProgressCard("org/app", 101, false, []string{"stage:propose", "stage:apply", "question"}, at),
	}
	m := inProgressModel(t, 80, 24, &fetch.Result{Cards: cards})

	line, ok := lineWith(tableArea(m), "#101")
	if !ok {
		t.Fatal("行が無い")
	}
	if got := kindColumn(line); got != "propose" {
		t.Errorf("段階が 2 つある主体の種別の列 = %q, want %q（段階順で先のもの）", got, "propose")
	}
}

func TestInProgressKindColumnTruncatesLabelModeStage(t *testing.T) {
	cards := []model.Card{inProgressCard("org/kanban", 101, false, []string{"In Progress"}, at)}
	res := &fetch.Result{Cards: cards, Modes: map[string]model.Mode{"org/kanban": model.ModeLabel}}
	m := inProgressModel(t, 80, 24, res)

	line, ok := lineWith(tableArea(m), "#101")
	if !ok {
		t.Fatal("行が無い")
	}
	col := ansi.Cut(line, kindColStart, kindColStart+kindColWidth)
	if ansi.StringWidth(col) != kindColWidth {
		t.Errorf("種別の列の表示幅 = %d, want %d: %q", ansi.StringWidth(col), kindColWidth, col)
	}
	if !strings.HasPrefix(col, "In Progr") || !strings.HasSuffix(col, "…") {
		t.Errorf("種別の列 = %q, want `In Progr` で始まり `…` で終わる", col)
	}
}

func TestOtherTabsKindColumnIsUnchanged(t *testing.T) {
	m := exampleModel(t, 80, 24)

	now, ok := lineWith(tableArea(m), "PR131")
	if !ok {
		t.Fatal("今やるタブの行が無い")
	}
	if got := kindColumn(now); got != "質問" {
		t.Errorf("今やるタブの種別の列 = %q, want %q", got, "質問")
	}

	m, _ = send(m, runeKey('2'))
	backlog, ok := lineWith(tableArea(m), "#140")
	if !ok {
		t.Fatal("バックログタブの行が無い")
	}
	if got := kindColumn(backlog); got != "todo 候補" {
		t.Errorf("バックログタブの種別の列 = %q, want %q", got, "todo 候補")
	}
}

// --- queue-screen: 進行中タブの段階ラベル名の色 ---

// stageColors は org/app の段階ラベルの色の表。
func stageColors() map[string]map[string]string {
	return map[string]map[string]string{"org/app": {"stage:propose": "0e8a16", "archive": "5319e7"}}
}

func TestInProgressStageLabelIsColored(t *testing.T) {
	cards := []model.Card{
		inProgressCard("org/app", 101, false, []string{"stage:propose", "wip"}, at.Add(-1*time.Hour)),
		inProgressCard("org/app", 102, true, []string{"archive"}, at.Add(-2*time.Hour)),
	}
	m := inProgressModel(t, 80, 24, &fetch.Result{Cards: cards, LabelColors: stageColors()})

	view := m.View().Content
	// 色は接頭辞を落とす前のラベル名で引き、列に出すのは落とした後の語。
	wantIn(t, view, renderLabelName("propose", "0e8a16"), "issue の段階ラベル名")
	wantIn(t, view, renderLabelName("archive", "5319e7"), "PR の段階ラベル名")

	lines := tableArea(m)
	issueLine, _ := lineWith(lines, "#101")
	prLine, _ := lineWith(lines, "PR102")
	if got := kindColumn(issueLine); got != "propose" {
		t.Errorf("色を付けても ANSI を除いた種別の列は変わらない: %q, want %q", got, "propose")
	}
	if got := kindColumn(prLine); got != "archive" {
		t.Errorf("色を付けても ANSI を除いた種別の列は変わらない: %q, want %q", got, "archive")
	}
}

func TestInProgressStageLabelWithoutColorTableIsPlain(t *testing.T) {
	cards := []model.Card{inProgressCard("org/app", 101, false, []string{"stage:propose", "wip"}, at)}
	m := inProgressModel(t, 80, 24, &fetch.Result{Cards: cards})

	wantPlain(t, m.View().Content, "propose", "ラベル色の表が無いときの段階ラベル名")
}

// rawRow は ANSI を残したまま、ANSI を除くと sub を含む最初の行を返す。
func rawRow(t *testing.T, m Model, sub string) string {
	t.Helper()
	for _, l := range strings.Split(m.View().Content, "\n") {
		if strings.Contains(ansi.Strip(l), sub) {
			return l
		}
	}
	t.Fatalf("%q を含む行が無い", sub)
	return ""
}

// 進行中の行は行全体の色を持たないので、段階ラベルが無い行には色のエスケープが 1 つも無い。
func TestInProgressRowWithoutStageHasNoColor(t *testing.T) {
	cards := []model.Card{inProgressCard("org/app", 101, false, []string{"question"}, at)}
	m := inProgressModel(t, 80, 24, &fetch.Result{Cards: cards, LabelColors: stageColors()})

	row := rawRow(t, m, "#101")
	if strings.Contains(row, "\x1b[") {
		t.Errorf("段階ラベルの無い進行中の行に色のエスケープがある: %q", row)
	}
}

// 進行中の行は行全体の色を持たず、色が付くのは種別の列の段階ラベル名の範囲だけ。
func TestInProgressRowHasNoRowColor(t *testing.T) {
	cards := []model.Card{inProgressCard("org/app", 101, false, []string{"stage:propose", "wip"}, at)}
	m := inProgressModel(t, 80, 24, &fetch.Result{Cards: cards, LabelColors: stageColors()})

	row := rawRow(t, m, "#101")
	if strings.HasPrefix(strings.TrimPrefix(row, "▶ "), "\x1b[") {
		t.Errorf("進行中の行が行全体の色で始まっている: %q", row)
	}
	// 色の指定は段階ラベル名の 1 か所だけ（種別の列）。
	if got := strings.Count(row, "\x1b[m"); got != 1 {
		t.Errorf("行の色のリセットが %d 個, want 1（段階ラベル名の分だけ）: %q", got, row)
	}
	// 色が始まるのは種別の列の先頭で、それより左（印と優先記号）には色が無い。
	i := strings.Index(row, "\x1b[")
	if i < 0 {
		t.Fatalf("段階ラベル名に色が付いていない: %q", row)
	}
	if w := ansi.StringWidth(row[:i]); w != kindColStart {
		t.Errorf("色が始まる位置 = %d 列, want %d（種別の列の先頭）: %q", w, kindColStart, row)
	}
}

// --- queue-screen: 進行中タブのリポジトリの見出し行 ---

func TestRepoHeaderLineFillsWidth(t *testing.T) {
	got := repoHeaderLine("org/app", 2, 80)

	if !strings.HasPrefix(got, "── org/app ── 2 件 ") {
		t.Errorf("見出し行の書き出しが違う: %q", got)
	}
	if w := ansi.StringWidth(got); w != 80 {
		t.Errorf("表示幅 = %d, want 80: %q", w, got)
	}
	if !strings.HasSuffix(got, "─") {
		t.Errorf("端末の幅まで `─` で埋まっていない: %q", got)
	}
}

func TestRepoHeaderLineTruncatesWithoutEllipsis(t *testing.T) {
	got := repoHeaderLine("org/very-long-repository-name", 12, 20)

	if w := ansi.StringWidth(got); w != 20 {
		t.Errorf("表示幅 = %d, want 20: %q", w, got)
	}
	if strings.Contains(got, "…") {
		t.Errorf("切った跡に … が付いている: %q", got)
	}
}

func TestRepoHeaderLineIsEmptyAtZeroWidth(t *testing.T) {
	if got := repoHeaderLine("org/app", 1, 0); got != "" {
		t.Errorf("幅 0 の見出し行 = %q, want 空文字列", got)
	}
}

func TestInProgressTableSeparatesRepos(t *testing.T) {
	cards := []model.Card{
		inProgressCard("org/app", 101, false, []string{"stage:propose", "wip"}, at.Add(-1*time.Hour)),
		inProgressCard("org/app", 102, false, []string{"stage:apply", "wip"}, at.Add(-2*time.Hour)),
		inProgressCard("org/web", 103, false, []string{"stage:apply", "wip"}, at.Add(-3*time.Hour)),
	}
	m := inProgressModel(t, 80, 40, &fetch.Result{Cards: cards})

	lines := tableArea(m)
	want := []string{"── org/app ── 2 件 ", "#101", "#102", "── org/web ── 1 件 ", "#103"}
	got := strings.Join(lines, "\n")
	order(t, got, want...)
	for _, l := range lines {
		if !strings.HasPrefix(l, "──") {
			continue
		}
		trimmed := strings.TrimRight(l, " ")
		if w := ansi.StringWidth(trimmed); w != 80 {
			t.Errorf("見出し行の表示幅 = %d, want 80: %q", w, l)
		}
		if !strings.HasSuffix(trimmed, "─") {
			t.Errorf("見出し行が `─` で終わっていない: %q", l)
		}
	}
}

func TestInProgressTableShowsHeaderForSingleRepo(t *testing.T) {
	cards := []model.Card{
		inProgressCard("org/app", 101, false, []string{"stage:propose", "wip"}, at.Add(-1*time.Hour)),
		inProgressCard("org/app", 102, false, []string{"stage:apply", "wip"}, at.Add(-2*time.Hour)),
		inProgressCard("org/app", 103, false, []string{"stage:archive", "wip"}, at.Add(-3*time.Hour)),
	}
	m := inProgressModel(t, 80, 40, &fetch.Result{Cards: cards})

	lines := tableArea(m)
	if !strings.HasPrefix(lines[0], "── org/app ── 3 件 ") {
		t.Fatalf("表の 1 行目が見出し行でない: %q", lines[0])
	}
	order(t, strings.Join(lines, "\n"), "#101", "#102", "#103")
}

func TestRepoHeaderIsNotSelectable(t *testing.T) {
	cards := []model.Card{
		inProgressCard("org/app", 101, false, []string{"stage:propose", "wip"}, at.Add(-1*time.Hour)),
		inProgressCard("org/web", 102, false, []string{"stage:propose", "wip"}, at.Add(-2*time.Hour)),
	}
	m := inProgressModel(t, 80, 40, &fetch.Result{Cards: cards})

	m, _ = send(m, runeKey('j')) // j 1 回で 2 枚目の Card に移る（見出し行は飛ばさない）

	lines := tableArea(m)
	marked, ok := lineWith(lines, "▶")
	if !ok {
		t.Fatalf("選択行の印が無い: %q", lines)
	}
	if !strings.Contains(marked, "#102") {
		t.Errorf("j 1 回で 2 枚目の Card に移っていない: %q", marked)
	}
	for _, l := range lines {
		if strings.HasPrefix(l, "──") && strings.Contains(l, "▶") {
			t.Errorf("見出し行に選択の印が付いている: %q", l)
		}
	}
}

func TestOtherTabsHaveNoRepoHeader(t *testing.T) {
	cards := []model.Card{
		nowCard("org/app", 101, 1, at.Add(-1*time.Hour)),
		nowCard("org/web", 102, 1, at.Add(-2*time.Hour)),
	}
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 80, Height: 40}, fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})

	for _, l := range tableArea(m) {
		if strings.HasPrefix(l, "──") {
			t.Errorf("今やるタブに見出し行がある: %q", l)
		}
	}
}

func TestRepoHeaderIsCutAtNarrowWidth(t *testing.T) {
	cards := []model.Card{
		inProgressCard("org/app", 101, false, []string{"stage:propose", "wip"}, at.Add(-1*time.Hour)),
		inProgressCard("org/web", 102, false, []string{"stage:propose", "wip"}, at.Add(-2*time.Hour)),
	}
	m := inProgressModel(t, 20, 40, &fetch.Result{Cards: cards})

	var headers int
	for _, l := range tableArea(m) {
		if !strings.HasPrefix(l, "──") {
			continue
		}
		headers++
		if w := ansi.StringWidth(l); w != 20 {
			t.Errorf("見出し行の表示幅 = %d, want 20: %q", w, l)
		}
		if strings.Contains(l, "…") {
			t.Errorf("見出し行に … がある: %q", l)
		}
	}
	if headers != 2 {
		t.Errorf("見出し行が %d 本, want 2", headers)
	}
}

// inProgressFixture は testdata/in-progress の 2 リポジトリ分の Card を s07 の Fetch で作る。
func inProgressFixture(t *testing.T) *fetch.Result {
	t.Helper()
	res, err := fetch.Fetch(context.Background(), gh.NewFake("testdata/in-progress"), []string{"org/app", "org/web"}, at, 0)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("詳細取得の失敗: %v", res.Errors)
	}
	return res
}

// numColStart / numColWidth は spec の列順が定める番号の列の位置と幅。
const (
	numColStart = 36
	numColWidth = 7
)

// tableSummary は表の各行を「見出し行の文言」または「種別の列 + 番号の列」に畳む。
// タイトルの折り返しで生まれた継続行（番号の列が空）は落とす。
func tableSummary(lines []string) []string {
	var out []string
	for _, l := range lines {
		if strings.HasPrefix(l, "──") {
			out = append(out, strings.TrimRight(l, "─"))
			continue
		}
		number := strings.TrimSpace(ansi.Cut(l, numColStart, numColStart+numColWidth))
		if number == "" {
			continue
		}
		out = append(out, kindColumn(l)+" "+number)
	}
	return out
}

// TestInProgressTabFullScreen は進行中タブの画面全体を 1 か所で固定する。
// 見出し行 2 本・段階の語・行の並びが揃った状態を、fixture から実際に取得した Result で描く。
func TestInProgressTabFullScreen(t *testing.T) {
	m := inProgressModel(t, 80, 40, inProgressFixture(t))

	got := tableSummary(tableArea(m))
	want := []string{
		"── org/app ── 4 件 ",
		"propose #101",
		"apply PR201",
		"archive #102",
		"- PR202",
		"── org/web ── 1 件 ",
		"apply #103",
	}
	if !slices.Equal(got, want) {
		t.Errorf("進行中タブの表 =\n%q\nwant\n%q", got, want)
	}

	// 段階ラベル名には fixture のラベル色が付く（色は接頭辞を落とす前の名前で引く）。
	view := m.View().Content
	wantIn(t, view, renderLabelName("propose", "0e8a16"), "issue #101 の段階ラベル名")
	wantIn(t, view, renderLabelName("apply", "1d76db"), "PR 201 の段階ラベル名")
}
