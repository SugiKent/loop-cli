package ui

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/loop-cli/internal/classify"
	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// screen は画面の状態。詳細は Model の中の状態で持ち、別 Model に委譲しない。
type screen int

const (
	screenQueue screen = iota
	screenCard
	screenPR
	screenConfirm
	screenHelp
	screenURL
	screenMergeConfirm
	screenNewConfirm
	screenLabels
	screenCloseConfirm
)

// detailState は開いている詳細。Card は開いた時点のコピーで、取得完了では差し替えない。
type detailState struct {
	card       model.Card
	prIdx      int
	expanded   bool
	fromDetail bool // PR 詳細にカード詳細から入ったか
	vp         viewport.Model
}

// dependsOnRe は mvp.md の `depends on #m`。書式が定まっていないので行頭に限定しない。
var dependsOnRe = regexp.MustCompile(`(?i)depends on #(\d+)`)

// dependsOn は本文から depends on の issue 番号を出現順に返す。
func dependsOn(body string) []int {
	var out []int
	for _, m := range dependsOnRe.FindAllStringSubmatch(body, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

// prStageOrder は PR 一覧に常に出す 3 段階。
var prStageOrder = []string{model.LabelPropose, model.LabelApply, model.LabelArchive}

// openDetail は選択行の Card の詳細を開く。Issue が無ければ PR 詳細を直接開く。
func (m Model) openDetail() Model {
	rows := m.rows[m.tab]
	if m.cursor >= len(rows) {
		return m
	}
	card := rows[m.cursor].card
	if card.Issue == nil && len(card.PRs) == 0 {
		return m
	}
	m.detail = detailState{card: card, vp: viewport.New()}
	if card.Issue == nil {
		m.screen = screenPR
	} else {
		m.screen = screenCard
	}
	m.refreshDetail()
	m.detail.vp.GotoTop()
	return m
}

// updateDetailKey は詳細画面のキーを扱う（q / Ctrl+C は Update が先に処理する）。
// 詳細から詳細へ移る enter / g だけが、移った先のセッションの取得のコマンドを返す。
func (m Model) updateDetailKey(key string) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch key {
	case "j", "down":
		m.detail.vp.ScrollDown(1)
	case "k", "up":
		m.detail.vp.ScrollUp(1)
	case "pgdown":
		m.detail.vp.PageDown()
	case "pgup":
		m.detail.vp.PageUp()
	case "x":
		m.detail.expanded = !m.detail.expanded
		m.refreshDetail()
	case "esc":
		if m.screen == screenPR && m.detail.fromDetail {
			m.screen = screenCard
			m.refreshDetail()
			m.detail.vp.GotoTop()
			break
		}
		m.screen = screenQueue
	case "tab":
		if m.screen == screenCard && len(m.detail.card.PRs) > 0 {
			m.detail.prIdx = (m.detail.prIdx + 1) % len(m.detail.card.PRs)
		}
	case "enter", "g":
		if m.screen == screenCard {
			if len(m.detail.card.PRs) == 0 {
				break
			}
			m.screen = screenPR
			m.detail.fromDetail = true
		} else {
			// PR 詳細では g だけがカード詳細へ戻る。Enter は何もしない。
			if key != "g" || m.detail.card.Issue == nil {
				break
			}
			m.screen = screenCard
		}
		m.refreshDetail()
		m.detail.vp.GotoTop()
		cmd = m.sessionOnOpenCmd()
	}
	return m, cmd
}

// refreshDetail は本文領域の内容と大きさを作り直す。View では作らない（Glamour を毎フレーム呼ばない）。
func (m *Model) refreshDetail() {
	_, bodyH := m.detailHeader()
	m.detail.vp.SetWidth(m.leftWidth())
	m.detail.vp.SetHeight(bodyH)
	if m.screen == screenPR {
		m.detail.vp.SetContentLines(m.prBodyLines())
		return
	}
	m.detail.vp.SetContentLines(m.cardBodyLines())
}

// detailHeader はヘッダ領域の行と本文領域の高さを返す。
// 本文 1 行を確保できないときはカード詳細の PR 一覧を末尾から落とし、それでも足りなければ
// 折り返したタイトル行を末尾から落とす（design.md D5）。
func (m Model) detailHeader() ([]string, int) {
	var fixed, prs []string
	var titleLines int
	if m.screen == screenPR {
		fixed, titleLines = m.prHeaderLines()
	} else {
		fixed, titleLines = m.cardHeaderLines()
		prs = m.prListLines()
	}
	keep := min(len(prs), max(m.height-2-len(fixed)-1, 0))
	header := append(fixed, prs[:keep]...)

	// 3 は区切り線 1 行 + フッタ 1 行 + 本文 1 行（戻り値の高さの下限と対応する）。
	// タイトルの 1 行目は落とさない（どの Issue / PR を見ているのか分からなくなる）。
	if over := len(header) + 3 - m.height; over > 0 && titleLines > 1 {
		drop := min(over, titleLines-1)
		header = append(header[:titleLines-drop], header[titleLines:]...)
		// 折り返し済みの行は幅以下なので Truncate は … を付けない。1 列空けて明示的に付ける。
		last := titleLines - drop - 1
		header[last] = ansi.Truncate(header[last], max(m.leftWidth()-1, 0), "") + "…"
	}
	return header, max(m.height-2-len(header), 1)
}

// cardHeaderLines はカード詳細のヘッダ行（PR 一覧を除く）を mvp.md の順で作り、
// 先頭の何行がタイトル行かを添えて返す。
func (m Model) cardHeaderLines() ([]string, int) {
	card := m.detail.card
	issue := card.Issue
	title := wrapTitle(fmt.Sprintf("%s #%d  ", issue.Repo, issue.Number), issue.Title, m.leftWidth())
	lines := title
	if card.Result.Summary != "" {
		lines = append(lines, card.Result.Summary)
	}

	mode, _ := m.repoMode(issue.Repo)
	stage := "段階なし"
	if st := model.IssueStages(mode, issue.Labels); len(st) > 0 {
		stage = "段階: " + strings.Join(m.labelNames(issue.Repo, st), " ")
	}
	// label に wip ラベルは無く、作業中は段階ラベル In Progress が示す。
	badges := []string{model.LabelBlocked, model.LabelWip, model.LabelQuestion}
	if mode == model.ModeLabel {
		badges = []string{model.LabelBlocked, model.LabelQuestion}
	}
	for _, badge := range badges {
		if model.HasLabel(issue.Labels, badge) {
			// 角括弧は画面の構造を示す記号なので塗らず、中のラベル名だけを塗る。
			stage += " [" + m.labelName(issue.Repo, badge) + "]"
		}
	}
	lines = append(lines, stage)

	if ns := dependsOn(issue.Body); len(ns) > 0 {
		refs := make([]string, len(ns))
		for i, n := range ns {
			refs[i] = fmt.Sprintf("#%d", n)
		}
		lines = append(lines, "depends on: "+strings.Join(refs, " "))
	}
	return lines, len(title)
}

// prListLines は紐づく PR を段階順に 1 行ずつ並べる。無い段階は `なし`、段階ラベルの無い PR は末尾。
// label には PR の段階ラベルが無いので、段階の見出しを持たず並び順のまま `[-]` で並べる。
func (m Model) prListLines() []string {
	prs := m.detail.card.PRs
	mode := m.detailMode()
	var lines []string
	if mode != model.ModeLabel {
		for _, stage := range prStageOrder {
			found := false
			for i, pr := range prs {
				if st := model.PRStages(mode, pr.Labels); len(st) == 0 || st[0] != stage {
					continue
				}
				found = true
				lines = append(lines, m.prListRow(i, stage, pr))
			}
			if !found {
				// `なし` はラベル名ではないので塗らず、角括弧の中の段階ラベル名だけを塗る。
				lines = append(lines, "  ["+m.labelName(m.detailRepo(), stage)+"] なし")
			}
		}
	}
	for i, pr := range prs {
		if len(model.PRStages(mode, pr.Labels)) == 0 {
			lines = append(lines, m.prListRow(i, "-", pr))
		}
	}
	return lines
}

// detailMode は詳細の対象カードのリポジトリの方式。Card は 1 リポジトリ分しか持たない。
func (m Model) detailMode() model.Mode {
	mode, _ := m.repoMode(m.detailRepo())
	return mode
}

// prListRow は PR 一覧の 1 行。選択中の PR には ▶ を付ける。
func (m Model) prListRow(i int, stage string, pr model.PR) string {
	mark := "  "
	if i == m.detail.prIdx {
		mark = "▶ "
	}
	// 段階ラベルの無い PR の `-` はラベル名ではないので、表に無い名前として色が付かない。
	line := fmt.Sprintf("%s[%s] PR#%d %s", mark, m.labelName(pr.Repo, stage), pr.Number, m.stateWord(prState(pr.State)))
	if pr.Canonical {
		line += "（最新・正本）"
	}
	if n, ok := model.ParseUndecided(pr.Body); ok {
		line += fmt.Sprintf(" 未確定 %d 件", n)
	} else {
		line += " 1 行目なし"
	}
	line += " labels: " + strings.Join(m.labelNames(pr.Repo, pr.Labels), " ")
	switch {
	case pr.MergeState == nil:
		line += " checks " + m.stateWord("取得失敗") + " mergeable " + m.stateWord("取得失敗")
	case classify.ChecksGreen(pr.MergeState):
		line += " checks " + m.stateWord("緑") + " mergeable " + m.stateWord(pr.MergeState.Mergeable)
	default:
		line += " checks " + m.stateWord("緑以外") + " mergeable " + m.stateWord(pr.MergeState.Mergeable)
	}
	return line
}

// prState は PR の状態の表記。
func prState(state string) string {
	switch state {
	case "OPEN":
		return "open"
	case "MERGED":
		return "merged"
	case "CLOSED":
		return "closed"
	}
	return strings.ToLower(state)
}

// cardBodyLines は Issue 本文・最新 blocked-by の要約・コメント時系列を並べる。
func (m Model) cardBodyLines() []string {
	issue := m.detail.card.Issue
	lines := renderMarkdown(issue.Body, m.leftWidth())
	if c, value, ok := model.LatestBlockedBy(issue.Comments); ok {
		lines = append(lines, "blocked-by: "+value)
		if value == "human" {
			if when, ok := model.UnblockWhen(c.Body); ok {
				lines = append(lines, "unblock-when: "+when)
			}
			lines = append(lines, questionLines(c.Body)...)
		}
	}
	return append(lines, m.commentSection(issue.Comments)...)
}

// questionLines は blocked-by: human のコメントから質問と選択肢を作る。
// 質問が 1 件も無ければ、マーカー行と blocked-by: 行と unblock-when: 行を除いた本文をそのまま出す
// （blocked-by: と unblock-when: は cardBodyLines が既に 1 行ずつ出している）。
func questionLines(body string) []string {
	qs := model.ParseQuestions(body)
	if len(qs) == 0 {
		var rest []string
		for _, l := range strings.Split(stripMarkers(body), "\n") {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, "blocked-by:") || strings.HasPrefix(t, "unblock-when:") {
				continue
			}
			rest = append(rest, l)
		}
		return rest
	}
	var lines []string
	for _, q := range qs {
		lines = append(lines, fmt.Sprintf("Q%d. %s", q.Number, q.Title))
		for _, o := range q.Options {
			if o.Recommended {
				lines = append(lines, "  "+o.Letter+"（推奨）: "+o.Text)
			} else {
				lines = append(lines, "  "+o.Letter+": "+o.Text)
			}
		}
	}
	return lines
}

// commentSection はコメント時系列。nil は取得失敗、長さ 0 はなし（s05 が表示に委ねた区別）。
func (m Model) commentSection(comments []model.Comment) []string {
	if comments == nil {
		return []string{"コメント: " + m.stateWord("取得失敗")}
	}
	if len(comments) == 0 {
		return []string{"コメント: なし"}
	}
	var lines []string
	for _, c := range comments {
		lines = append(lines, commentBlock(c.Author, c.Body, c.CreatedAt, c.AI, m.location(), m.leftWidth(), m.detail.expanded)...)
	}
	return lines
}

// currentPR は詳細で選択中の PR。
func (m Model) currentPR() model.PR { return m.detail.card.PRs[m.detail.prIdx] }

// prHeaderLines は PR 詳細のヘッダ（タイトル行と labels 行）と、
// 先頭の何行がタイトル行かを返す。
func (m Model) prHeaderLines() ([]string, int) {
	pr := m.currentPR()
	stage := "-"
	prMode, _ := m.repoMode(pr.Repo)
	if st := model.PRStages(prMode, pr.Labels); len(st) > 0 {
		stage = st[0]
	}
	title := wrapTitle(fmt.Sprintf("%s PR#%d  ", pr.Repo, pr.Number), pr.Title, m.leftWidth())
	labels := fmt.Sprintf("[%s] %s  labels: %s",
		m.labelName(pr.Repo, stage), m.stateWord(prState(pr.State)), strings.Join(m.labelNames(pr.Repo, pr.Labels), " "))
	return append(title, labels), len(title)
}

// prBodyLines は 1 行目判定・紐づけ・checks・本文・会話・review thread を並べる。
func (m Model) prBodyLines() []string {
	pr := m.currentPR()
	var lines []string
	if n, ok := model.ParseUndecided(pr.Body); ok {
		lines = append(lines, fmt.Sprintf("未確定の判断: %d 件", n))
	} else {
		lines = append(lines, "1 行目に未確定の判断が無い")
	}
	if n, ok := fetch.LinkedIssue(pr.Title, pr.Body); ok {
		lines = append(lines, fmt.Sprintf("紐づく issue: #%d", n))
	} else {
		lines = append(lines, "紐づく issue: なし")
	}
	lines = append(lines, m.checkLines(pr.MergeState)...)
	lines = append(lines, renderMarkdown(pr.Body, m.leftWidth())...)
	lines = append(lines, m.commentSection(pr.Comments)...)
	return append(lines, m.reviewThreadLines(pr.ReviewThreads)...)
}

// checkLines は merge 状態と checks。s20 は全 PR に取りに行くので nil は取得失敗を意味する。
// 塗るのは値の語だけで、見出し（checks: / mergeable:）とチェック名は塗らない。
func (m Model) checkLines(ms *gh.PRMergeState) []string {
	if ms == nil {
		return []string{"checks: " + m.stateWord("取得失敗")}
	}
	lines := []string{strings.TrimSpace("mergeable: " + m.stateWord(ms.Mergeable) + " " + m.stateWord(ms.MergeStateStatus))}
	if len(ms.StatusCheckRollup) == 0 {
		return append(lines, "  checks: なし")
	}
	for _, c := range ms.StatusCheckRollup {
		switch c.Typename {
		case "CheckRun":
			state := c.Conclusion
			if state == "" {
				state = c.Status
			}
			lines = append(lines, "  "+c.Name+": "+m.stateWord(state))
		case "StatusContext":
			lines = append(lines, "  "+c.Context+": "+m.stateWord(c.State))
		}
	}
	return lines
}

// reviewThreadLines は review thread を未 resolve 先頭で並べる。thread のコメントは畳まない。
func (m Model) reviewThreadLines(threads []gh.ReviewThread) []string {
	if threads == nil {
		return []string{"review thread: " + m.stateWord("取得失敗")}
	}
	if len(threads) == 0 {
		return []string{"review thread: なし"}
	}
	sorted := slices.Clone(threads)
	sort.SliceStable(sorted, func(i, j int) bool { return !sorted[i].IsResolved && sorted[j].IsResolved })

	var lines []string
	for _, th := range sorted {
		head := "thread 未 resolve"
		if th.IsResolved {
			head = "thread resolved"
		}
		lines = append(lines, head)
		for _, c := range th.Comments {
			lines = append(lines, commentBlock(c.Author.Login, c.Body, c.CreatedAt, model.IsAI(c.Body), m.location(), m.leftWidth(), true)...)
		}
	}
	return lines
}

// detailHint は詳細画面のフッタ左。動くキーだけを出す。
func (m Model) detailHint() string {
	if m.screen == screenPR {
		return "Esc 戻る  x 展開  g issue へ  ? ヘルプ  u URL  a 回答  L ラベル  m merge  c close  n 新規  o ブラウザ  q 終了"
	}
	// n と c は PR の有無によらず動くので、PR のキーと違って省かない
	// （c の対象はカード詳細では Issue で、これは常にある）。
	if len(m.detail.card.PRs) == 0 {
		return "Esc 戻る  x 展開  ? ヘルプ  u URL  a 回答  t todo  L ラベル  c close  n 新規  o ブラウザ  q 終了"
	}
	return "Esc 戻る  Tab PR 選択  Enter PR を開く  x 展開  g PR へ  ? ヘルプ  u URL  a 回答  t todo  L ラベル  m merge  c close  n 新規  o ブラウザ  q 終了"
}
