package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// 表の固定列幅（design.md の未決事項の既定値）。合計 48 列で、残りをタイトルに充てる。
const (
	colMark    = 2
	colPrio    = 4
	colKind    = 10
	colRepo    = 20
	colNumber  = 7
	colElapsed = 5
	colFixed   = colMark + colPrio + colKind + colRepo + colNumber + colElapsed
)

// prioMark は優先度の記号。mvp.md にある 1 / 2 / 3 以外は design.md の未決事項の既定値。
var prioMark = map[int]string{0: "!!!", 1: "!!", 2: "!", 3: "●", 5: "●"}

// kindStyle は行の色。種別で固定する（mvp.md「行の色は種別で固定」）。
var kindStyle = map[string]lipgloss.Style{
	"質問":      lipgloss.NewStyle().Foreground(lipgloss.Color("#D876E3")),
	"方針":      lipgloss.NewStyle().Foreground(lipgloss.Color("#B60205")),
	"merge":   lipgloss.NewStyle().Foreground(lipgloss.Color("#0E8A16")),
	"todo 候補": lipgloss.NewStyle().Foreground(lipgloss.Cyan),
	"異常":      lipgloss.NewStyle().Background(lipgloss.Yellow).Foreground(lipgloss.Black),
}

var (
	boldStyle  = lipgloss.NewStyle().Bold(true)
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#B60205"))
)

func (m Model) View() tea.View {
	// v2 では alt screen を tea.View で要求する。インライン描画のままだと
	// 端末高と同じ高さのフレームが再描画のたびに積み上がる。
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	switch m.screen {
	case screenConfirm:
		return m.renderConfirm()
	case screenMergeConfirm:
		return m.renderMergeConfirm()
	case screenCard, screenPR:
		return m.renderDetail()
	case screenHelp:
		return m.renderHelp()
	case screenURL:
		return m.renderURLs()
	}
	lines := []string{m.header()}
	rest := max(m.height-3, 0)
	tableH := (rest + 1) / 2
	lines = append(lines, m.tableLines(tableH)...)
	lines = append(lines, strings.Repeat("─", max(m.width, 0)))
	lines = append(lines, m.previewLines(rest-tableH)...)
	return strings.Join(append(lines, m.footer(m.queueHint())), "\n")
}

// renderDetail は詳細画面を ヘッダ領域 / 区切り線 / 本文領域 / フッタ の順に描く。
func (m Model) renderDetail() string {
	header, _ := m.detailHeader()
	lines := make([]string, 0, len(header)+2)
	for _, l := range header {
		lines = append(lines, ansi.Truncate(l, m.width, "…"))
	}
	lines = append(lines, strings.Repeat("─", max(m.width, 0)))
	lines = append(lines, strings.Split(m.detail.vp.View(), "\n")...)
	return strings.Join(append(lines, m.footer(m.detailHint())), "\n")
}

// queueHint はキュー画面のフッタ左。
func (m Model) queueHint() string {
	// 移動系のキー（j / k / 1–4 / Tab）は出さず `?` のヘルプに委ねる（既定幅 80 に収めるため）。
	return "Enter 開く  a 回答  t todo  m merge  o ブラウザ  R 更新  ? ヘルプ  u URL  q 終了"
}

// header はアプリ名・4 タブの件数・最終更新時刻を 1 行で書く。
// 端末幅を超えるときは [4] → [3] → [2] の順にタブ名を落とし、次に時刻を省き、最後に幅で切る。
func (m Model) header() string {
	clock := "↻ --:--"
	if !m.at.IsZero() {
		clock = "↻ " + m.at.Format("15:04")
	}
	// 新しい版があるときだけ時刻の左に印を出す（s23 self-update）。打つコマンドは README にある。
	right := clock
	if m.updateAvailable {
		right = "↑ update  " + clock
	}

	fits := func(shortened int, right string) bool {
		return ansi.StringWidth(m.headerLeft(shortened, false))+1+ansi.StringWidth(right) <= m.width
	}
	shortened := 0
	for ; shortened < len(tabOrder)-1; shortened++ {
		if fits(shortened, right) {
			break
		}
	}
	// 全部短縮しても収まらなければ 更新の印 → 時刻 の順に落とす。
	if !fits(shortened, right) {
		right = clock
	}
	if !fits(shortened, right) {
		right = ""
	}

	left := m.headerLeft(shortened, true)
	plain := ansi.StringWidth(m.headerLeft(shortened, false))
	if right == "" {
		return ansi.Truncate(left, m.width, "")
	}
	return left + strings.Repeat(" ", m.width-plain-ansi.StringWidth(right)) + right
}

// headerLeft は右から shortened 個のタブをタブ名なしにしたヘッダ左側を返す。
func (m Model) headerLeft(shortened int, styled bool) string {
	parts := make([]string, 0, len(tabOrder))
	for i, tab := range tabOrder {
		var label string
		if i >= len(tabOrder)-shortened {
			label = fmt.Sprintf("[%d] %d", i+1, len(m.rows[tab]))
		} else {
			label = fmt.Sprintf("[%d]%s %d", i+1, tab, len(m.rows[tab]))
		}
		if styled && tab == m.tab {
			label = boldStyle.Render(label)
		}
		parts = append(parts, label)
	}
	return "loop-cli  " + strings.Join(parts, "  ")
}

// emptyHint は取得成功で 0 件のときに表の領域へ出す 2 行（mvp.md「初回起動（onboarding）」）。
var emptyHint = []string{
	"stage:* ラベルの無いリポジトリは何も出ません。",
	"issue-driven-sdd の routines-setup を回したリポジトリを設定してください",
}

// tableLines は現在タブの行を h 行ぶん返す。取得成功で 0 件ならヒントを縦横中央に出す。
func (m Model) tableLines(h int) []string {
	if len(m.cards) == 0 && !m.fetching && m.errText == "" {
		return cut(m.emptyHintLines(h), h)
	}
	rows := m.rows[m.tab]
	lines := make([]string, 0, len(rows))
	for i, r := range rows {
		lines = append(lines, m.tableRow(r, i == m.cursor))
	}
	return cut(lines, h)
}

// emptyHintLines はヒントの 2 行を上に空行を置いて縦中央にし、各行を横中央に置く。
// 右側には空白を足さない（Lip Gloss の配置関数は右詰めの空白を足すので使わない）。
func (m Model) emptyHintLines(h int) []string {
	lines := make([]string, 0, h)
	for range max((h-len(emptyHint))/2, 0) {
		lines = append(lines, "")
	}
	for _, hint := range emptyHint {
		line := strings.Repeat(" ", max((m.width-ansi.StringWidth(hint))/2, 0)) + hint
		lines = append(lines, ansi.Truncate(line, m.width, ""))
	}
	return lines
}

// tableRow は 1 行を 優先 / 種別 / リポジトリ / # / タイトル / 経過 の順に組み、種別の色を付ける。
func (m Model) tableRow(r row, selected bool) string {
	mark := "  "
	if selected {
		mark = "▶ "
	}
	prio, ok := prioMark[r.card.Result.Priority]
	if !ok {
		prio = "-"
	}
	number := fmt.Sprintf("#%d", r.number)
	if r.isPR {
		number = fmt.Sprintf("PR%d", r.number)
	}
	kind := r.card.Result.Situation.Kind()

	line := pad(mark, colMark) + pad(prio, colPrio) + pad(kind, colKind) +
		pad(r.repo, colRepo) + pad(number, colNumber)
	if titleW := m.width - colFixed; titleW > 0 {
		line += pad(r.title, titleW)
	}
	line += pad(Elapsed(m.at, r.updatedAt), colElapsed)
	line = ansi.Truncate(line, m.width, "")

	if style, ok := kindStyle[kind]; ok {
		return style.Render(line)
	}
	return line
}

// footer は左にキーヒント、右に取得状態を出す。両方が入らなければ状態を優先する。
func (m Model) footer(hint string) string {
	var status string
	switch {
	case m.fetching:
		status = m.spinner.View() + " 取得中"
	case m.writeStatus != "":
		status = m.writeStatus
		if m.writeStatusErr {
			status = errorStyle.Render(status)
		}
	case m.errText != "":
		status = errorStyle.Render(m.errText)
	case m.partial != "":
		status = errorStyle.Render(m.partial)
	}
	if status == "" {
		return ansi.Truncate(hint, m.width, "")
	}

	statusW := ansi.StringWidth(status)
	hintW := ansi.StringWidth(hint)
	if hintW+1+statusW > m.width {
		return ansi.Truncate(status, m.width, "")
	}
	return hint + strings.Repeat(" ", m.width-hintW-statusW) + status
}

// pad は表示幅 w に切り詰めて埋める。
func pad(s string, w int) string {
	s = ansi.Truncate(s, w, "…")
	if d := w - ansi.StringWidth(s); d > 0 {
		s += strings.Repeat(" ", d)
	}
	return s
}

// cut は行を h 行に切る。スクロールは持たない（mvp.md にキーが無い）。
func cut(lines []string, h int) []string {
	if h < 0 {
		h = 0
	}
	if len(lines) > h {
		return lines[:h]
	}
	return lines
}
