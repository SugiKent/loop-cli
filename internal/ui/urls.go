package ui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// urlItem は一覧の 1 件。source は出典の語（session / 本文 / コメント / thread）。
type urlItem struct {
	source string
	text   string
	url    string
}

// urlListState は URL 一覧画面の状態。スクロール位置は持たず、選択位置から表示位置を決める。
type urlListState struct {
	items  []urlItem
	cursor int
	from   screen // u を押した画面。Esc で戻る先
}

// openedURLMsg は 1 回の OpenURL の完了。成功は画面に出さない（o と同じ扱い）。
type openedURLMsg struct {
	url string
	err error
}

// urlKey は `u`（URL 一覧を開く）を扱う。対象は画面が見せているものに決まる（`o` と同じ規則）。
// gh を呼ばないので、URL の収集は画面の状態を変えるのと同じ処理の中で行う。
func (m Model) urlKey() (tea.Model, tea.Cmd) {
	issue, pr, ok := m.urlTarget()
	if !ok {
		return m, nil
	}
	// 右ペインのセッションが決まっていれば先頭に置く（キュー画面では sessionID が決まらない）。
	sessionID, _ := m.sessionID()
	items := collectURLs(sessionID, issue, pr)
	if len(items) == 0 {
		m.writeStatus, m.writeStatusErr = "URL がありません", false
		return m, nil
	}
	m.urls = urlListState{items: items, from: m.screen}
	m.screen = screenURL
	return m, nil
}

// urlTarget は今の画面の URL の収集元を返す。返るのは Issue か PR の片方だけ。
func (m Model) urlTarget() (*model.Issue, *model.PR, bool) {
	switch m.screen {
	case screenCard:
		return m.detail.card.Issue, nil, true
	case screenPR:
		pr := m.currentPR()
		return nil, &pr, true
	}
	rows := m.rows[m.tab]
	if m.cursor >= len(rows) {
		return nil, nil, false
	}
	issue, pr := subjectOf(rows[m.cursor].card)
	return issue, pr, true
}

// collectURLs はセッションの URL（あれば）を先頭に置き、続けて対象から
// 本文 → コメント → review thread の順に URL を集め、重複を初出だけ残す。
// セッションを先頭に置くので、PR 本文に同じ URL があっても出典は session になる。
// Comments / ReviewThreads が nil（詳細の取得失敗）のときは、その収集元を飛ばす。
func collectURLs(sessionID string, issue *model.Issue, pr *model.PR) []urlItem {
	var items []urlItem
	seen := map[string]bool{}
	addOne := func(source, text, url string) {
		if seen[url] {
			return
		}
		seen[url] = true
		items = append(items, urlItem{source: source, text: text, url: url})
	}
	add := func(source, body string) {
		for _, l := range model.ParseLinks(body) {
			addOne(source, l.Text, l.URL)
		}
	}
	if sessionID != "" {
		addOne("session", "", sessionURL(sessionID))
	}
	switch {
	case issue != nil:
		add("本文", issue.Body)
		for _, c := range issue.Comments {
			add("コメント", c.Body)
		}
	case pr != nil:
		add("本文", pr.Body)
		for _, c := range pr.Comments {
			add("コメント", c.Body)
		}
		for _, th := range pr.ReviewThreads {
			for _, c := range th.Comments {
				add("thread", c.Body)
			}
		}
	}
	return items
}

// updateURLKey は URL 一覧画面のキーを扱う（q / Ctrl+C は Update が先に処理する）。
// 表に無いキー（u / a / t / o / ? を含む）は何もしない。
func (m Model) updateURLKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.urls.cursor < len(m.urls.items)-1 {
			m.urls.cursor++
		}
	case "k", "up":
		if m.urls.cursor > 0 {
			m.urls.cursor--
		}
	case "enter":
		return m, openURLCmd(m.client, m.urls.items[m.urls.cursor].url)
	case "esc":
		m.screen = m.urls.from
		// 一覧中の WindowSizeMsg は詳細の寸法を作り直さないので、戻るときに作り直す。
		if m.screen == screenCard || m.screen == screenPR {
			m.refreshDetail()
		}
	}
	return m, nil
}

// openURLCmd は OpenURL を別ゴルーチンで実行し、結果を openedURLMsg で返す。
func openURLCmd(client gh.GHClient, url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
		defer cancel()
		return openedURLMsg{url: url, err: client.OpenURL(ctx, url)}
	}
}

// updateOpenedURL は URL を開いた結果を扱う。失敗だけをフッタに赤で出す。
func (m Model) updateOpenedURL(msg openedURLMsg) Model {
	if msg.err != nil {
		m.writeStatus, m.writeStatusErr = msg.url+" を開けません: "+msg.err.Error(), true
	}
	return m
}

// renderURLs は URL 一覧画面を 見出し / 空行 / URL の行 / 空行 / フッタ の順に描く。
func (m Model) renderURLs() string {
	// 表示できる行数は 高さ − 見出し − 空行 − フッタ。スクロール位置は持たず選択位置から決める。
	h := max(m.height-3, 0)
	start := max(m.urls.cursor-h+1, 0)
	lines := []string{"URL を開く", ""}
	for i := start; i < len(m.urls.items) && i < start+h; i++ {
		lines = append(lines, m.urlLine(m.urls.items[i], i == m.urls.cursor))
	}
	for len(lines) < m.height-1 {
		lines = append(lines, "")
	}
	lines = cut(lines, max(m.height-1, 0))
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "…")
	}
	return strings.Join(append(lines, m.footer("j/k 選択  Enter 開く  Esc 戻る  q 終了")), "\n")
}

// urlLine は一覧の 1 行 `<印>[<出典>] <リンクテキスト>  <URL>`。テキストが空なら間は空白 2 列。
func (m Model) urlLine(it urlItem, selected bool) string {
	mark := "  "
	if selected {
		mark = "▶ "
	}
	line := mark + "[" + it.source + "]"
	if it.text != "" {
		line += " " + it.text
	}
	return line + "  " + it.url
}
