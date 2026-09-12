package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/loop-cli/internal/model"
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
	// colTitleStart はタイトル列が始まる位置（43 列目）。折り返した継続行の字下げ幅でもある。
	colTitleStart = colFixed - colElapsed
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
	case screenNewConfirm:
		return m.renderNewConfirm()
	case screenCloseConfirm:
		return m.renderCloseConfirm()
	case screenCard, screenPR:
		return m.renderDetail()
	case screenHelp:
		return m.renderHelp()
	case screenURL:
		return m.renderURLs()
	case screenLabels:
		return m.renderLabels()
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
// 端末幅が 120 以上なら左右 2 ペインにし、右にセッションの稼働状況を出す（s31 card-detail）。
func (m Model) renderDetail() string {
	leftW := m.leftWidth()
	header, _ := m.detailHeader()
	left := make([]string, 0, len(header)+2)
	for _, l := range header {
		left = append(left, ansi.Truncate(l, leftW, "…"))
	}
	left = append(left, strings.Repeat("─", max(leftW, 0)))
	left = append(left, strings.Split(m.detail.vp.View(), "\n")...)

	lines := left
	if m.twoPane() {
		lines = joinPanes(left, m.sessionPaneLines(len(left)), leftW)
	}
	// フッタは 2 ペインでも 1 行で端末の幅の全体を使う（ヒントとステータスは画面に 1 組しかない）。
	return strings.Join(append(lines, m.footer(m.detailHint())), "\n")
}

// joinPanes は左ペインの各行の右に縦の区切り線と右ペインの行を並べる。
// 右ペインの行が尽きた後は左ペインの行だけを出す（横の区切り線がそこに当たれば、
// その行は左ペインの幅でちょうど閉じる）。
func joinPanes(left, right []string, leftW int) []string {
	out := make([]string, len(left))
	for i, l := range left {
		if i >= len(right) {
			out[i] = l
			continue
		}
		out[i] = pad(l, leftW) + "│" + right[i]
	}
	return out
}

// queueHint はキュー画面のフッタ左。
func (m Model) queueHint() string {
	// 移動系のキー（j / k / 1–4 / Tab）は出さず `?` のヘルプに委ねる（既定幅 80 に収めるため）。
	return "Enter 開く  a 回答  t todo  L ラベル  m merge  c close  o ブラウザ  R 更新  ? ヘルプ  u URL  q 終了"
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
	"stage:* / To Do ラベルの無いリポジトリは何も出ません。",
	"issue-driven-sdd の routines-setup を回すか、issue-label-driven の To Do ラベルを作ってください",
}

// tableLines は現在タブの行を h 行ぶん返す。取得成功で 0 件ならヒントを縦横中央に出す。
func (m Model) tableLines(h int) []string {
	if len(m.cards) == 0 && !m.fetching && m.errText == "" {
		return cut(m.emptyHintLines(h), h)
	}
	rows := m.rows[m.tab]
	lines := make([]string, 0, len(rows))
	for i, r := range rows {
		// 進行中タブはリポジトリの区切りに見出し行を挟む。行があるリポジトリが 1 つでも出す
		// （design.md D7。データで分岐させると自動更新のたびに表が 1 行ずれる）。
		// 挟むのは描画のときだけで rows の要素は増やさないので、カーソルと選択の意味は変わらない。
		if m.tab == model.TabInProgress && (i == 0 || rows[i-1].repo != r.repo) {
			lines = append(lines, repoHeaderLine(r.repo, countRepoRows(rows[i:]), m.width))
		}
		lines = append(lines, m.tableRow(r, i == m.cursor)...)
	}
	return cut(lines, h)
}

// countRepoRows は先頭と同じリポジトリの行が先頭から何枚続くかを返す（見出し行の件数）。
func countRepoRows(rows []row) int {
	for i, r := range rows {
		if r.repo != rows[0].repo {
			return i
		}
	}
	return len(rows)
}

// repoHeaderLine は進行中タブのリポジトリの見出し行 `── <owner/name> ── <n> 件 ────…` を返す。
// 端末の幅に達するまで `─` で埋め、収まらなければ幅で切る（`…` は付けない。切った跡が
// 線の一部に見えるようにするため。design.md D8）。幅が 0 以下なら空文字列。
func repoHeaderLine(repo string, n, width int) string {
	if width <= 0 {
		return ""
	}
	head := ansi.Truncate(fmt.Sprintf("── %s ── %d 件 ", repo, n), width, "")
	return head + strings.Repeat("─", max(width-ansi.StringWidth(head), 0))
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

// tableRow は 1 枚の Card の行を 優先 / 種別 / リポジトリ / # / タイトル / 経過 の順に組み、
// 種別の色を付ける。タイトルが列幅に収まらなければ折り返して複数行を返す（design.md D3）。
// 継続行はタイトル列の開始位置に縦を揃え、▶ と経過は 1 行目にだけ出す。
func (m Model) tableRow(r row, selected bool) []string {
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
	kindCol := pad(kind, colKind)
	if m.tab == model.TabInProgress {
		// 進行中タブでは全行同じ `進行中` の代わりに段階ラベル名を出し、s33-colorful-labels の
		// 規則で色を付ける（design.md D1 / D5）。色は接頭辞を落とす前のラベル名で引くので、
		// 段階を持たない行（stage が空）の `-` には色が付かない。
		kindCol = pad(renderLabelName(r.stageWord, m.labelColors[r.repo][r.stage]), colKind)
	}

	fixed := pad(mark, colMark) + pad(prio, colPrio) + kindCol +
		pad(r.repo, colRepo) + pad(number, colNumber)
	elapsed := pad(Elapsed(m.at, r.updatedAt), colElapsed)

	var lines []string
	titleW := m.width - colFixed
	if titleW < 2 {
		// 端末幅 49 以下（タイトル列に全角 1 文字が入らない幅）ではタイトルを出さず、
		// 1 行だけを端末幅で切る。この幅では折り返しが幅を守れない（wrapToWidth）。
		lines = []string{ansi.Truncate(fixed+elapsed, m.width, "")}
	} else {
		// 継続行も端末幅ちょうどまで空白で埋める（異常の Card の背景色が行ごとに途切れないように）。
		indent := strings.Repeat(" ", colTitleStart)
		for i, frag := range wrapToWidth(r.title, titleW) {
			if i == 0 {
				lines = append(lines, fixed+pad(frag, titleW)+elapsed)
				continue
			}
			lines = append(lines, indent+pad(frag, titleW)+strings.Repeat(" ", colElapsed))
		}
	}

	if style, ok := kindStyle[kind]; ok {
		for i, l := range lines {
			lines[i] = style.Render(l)
		}
	}
	return lines
}

// footer は左にキーヒント、右に取得状態を出す。両方が入らなければ状態を優先する。
func (m Model) footer(hint string) string {
	var status string
	switch {
	// 書き込みのステータスは押されたキーへの直接の返事なので、取得中のスピナーより先に出す
	// （初回取得の最中に t を拒否したときも、押したキーが黙って効かないように見えない）。
	case m.writeStatus != "":
		status = m.writeStatus
		if m.writeStatusErr {
			status = errorStyle.Render(status)
		}
	case m.fetching:
		status = m.spinner.View() + " 取得中"
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
