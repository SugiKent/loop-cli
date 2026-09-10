package ui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/loop-cli/internal/action"
	"github.com/SugiKent/loop-cli/internal/gh"
)

// newIssueState は `n` を押してから作成が終わるまでの状態。
// repo は `n` を押した時点の作成先で、確認中に取得が完了しても変えない。
type newIssueState struct {
	repo   string
	draft  string // 編集後のテキストそのもの。`e` はこれをエディタに返す
	title  string
	body   string
	marker bool   // 下書きに routine のマーカーがあるか（あれば作成の道を出さない）
	from   screen // `n` を押した画面。確認画面から戻る先
}

// createdMsg は 1 回の issue 作成の完了。
type createdMsg struct {
	repo string
	url  string
	err  error
}

// newKey は `n`（新しい issue を作る）を扱う。作成先は画面が見せている対象のリポジトリで、
// 人には選ばせない（`a` の answerTarget と同じ決め方）。
func (m Model) newKey() (tea.Model, tea.Cmd) {
	if m.writing {
		return m, nil
	}
	repo, ok := m.newIssueRepo()
	if !ok {
		// `a` / `m` は黙って何もしないが、`n` は「作りたい」という意思に応答が無いと理由が分からない。
		m.writeStatus, m.writeStatusErr = "新規作成の対象がありません", false
		return m, nil
	}
	m.newIssue = newIssueState{repo: repo, from: m.screen}
	m.writeStatus, m.writeStatusErr = "", false
	// 書式の案内を初期テキストに入れると、それがそのままタイトルになる（書式は確認画面が見せる）。
	cmd := m.openEditor(routeNewIssue, "")
	return m, cmd
}

// newIssueRepo は今の画面の作成先のリポジトリを返す。主体が issue か PR かでは変わらない。
func (m Model) newIssueRepo() (string, bool) {
	switch m.screen {
	case screenCard:
		return m.detail.card.Issue.Repo, true
	case screenPR:
		return m.currentPR().Repo, true
	}
	rows := m.rows[m.tab]
	if m.cursor >= len(rows) {
		return "", false
	}
	return rows[m.cursor].repo, true
}

// updateNewEdited は `n` で始めた編集の完了を扱う。分割と検査は action の関数を借り、
// internal/ui に別の判定を持たない。作れない下書きは確認画面へ進めず、理由をフッタに出す。
func (m Model) updateNewEdited(msg editedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.writeStatus, m.writeStatusErr = "エディタ: "+msg.err.Error(), true
		return m, nil
	}
	title, body := action.SplitNewIssue(msg.text)
	switch {
	case title == "":
		m.writeStatus, m.writeStatusErr = "作成を中止しました（タイトルが空）", false
	case body == "":
		m.writeStatus, m.writeStatusErr = "作成を中止しました（本文が空）", false
	default:
		m.newIssue.draft, m.newIssue.title, m.newIssue.body = msg.text, title, body
		m.newIssue.marker = action.HasRoutineMarker(msg.text)
		m.screen = screenNewConfirm
	}
	return m, nil
}

// updateNewConfirmKey は作成の確認画面のキーを扱う（q / Ctrl+C は Update が先に処理する）。
func (m Model) updateNewConfirmKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y":
		// マーカーは不変条件 7 が「書かない」なので、作成する道を出さない。
		if m.newIssue.marker {
			return m, nil
		}
		m.screen = m.newIssue.from
		m.writing = true
		m.writeStatus, m.writeStatusErr = m.newIssue.repo+" に issue を作成中", false
		return m, createIssueCmd(m.client, m.newIssue.repo, m.newIssue.title, m.newIssue.body)
	case "e":
		m.screen = m.newIssue.from
		cmd := m.openEditor(routeNewIssue, m.newIssue.draft)
		return m, cmd
	case "esc":
		m.screen = m.newIssue.from
		m.writeStatus, m.writeStatusErr = "作成を中止しました", false
	}
	return m, nil
}

// createIssueCmd は action.CreateIssue を別ゴルーチンで実行し、結果を createdMsg で返す。
func createIssueCmd(client gh.GHClient, repo, title, body string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
		defer cancel()
		url, err := action.CreateIssue(ctx, client, repo, title, body)
		return createdMsg{repo: repo, url: url, err: err}
	}
}

// updateCreated は作成の完了を扱う。結果はフッタに出すだけで、再取得はしない（1 件再取得は s18）。
// 成功の文言に作成先を入れないのは、幅 80 の端末で URL の末尾が切れないため（URL がリポジトリ名を含む）。
func (m Model) updateCreated(msg createdMsg) Model {
	m.writing = false
	if msg.err != nil {
		m.writeStatus, m.writeStatusErr = msg.repo+" の issue 作成に失敗: "+msg.err.Error(), true
		return m
	}
	m.writeStatus, m.writeStatusErr = "issue を作成しました: "+msg.url, false
	return m
}

// renderNewConfirm は作成前の確認を全画面で描く。作成先とタイトルの分かれ方を、
// `y` を押す前に必ず目に入る位置に置く。スクロールは持たない（s10 / s14 の確認画面と同じ）。
func (m Model) renderNewConfirm() string {
	s := m.newIssue
	lines := []string{"新規 issue の確認: " + s.repo, "タイトル: " + s.title}
	if s.marker {
		lines = append(lines, "<!-- routine --> を含む下書きでは作成できません（TUI からの書き込みは人の発言でなければなりません）")
	}
	lines = append(lines, strings.Repeat("─", max(m.width, 0)))
	lines = append(lines, strings.Split(s.body, "\n")...)

	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "…")
	}
	lines = cut(lines, max(m.height-1, 0))
	return strings.Join(append(lines, m.footer(m.newConfirmHint())), "\n")
}

// newConfirmHint は作成の確認画面のフッタ左。マーカーのときは作成の道を出さない。
func (m Model) newConfirmHint() string {
	if m.newIssue.marker {
		return "e 編集に戻る  Esc 中止  q 終了"
	}
	return "y 作成  e 編集に戻る  Esc 中止  q 終了"
}
