package ui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/sugi-loop/internal/gh"
)

// browsedMsg は 1 回の `gh browse` の完了。成功は画面に出さない（ブラウザが開くこと自体が結果）。
type browsedMsg struct {
	label string
	err   error
}

// browseKey は `o`（ブラウザで開く）を扱う。対象は画面が見せているものに決まる（`a` と同じ規則）。
// 書き込みではないので、書き込み中フラグでも止めず、ステータスも消さない。
func (m Model) browseKey() (tea.Model, tea.Cmd) {
	label, repo, number, ok := m.browseTarget()
	if !ok {
		return m, nil
	}
	return m, browseCmd(m.client, label, repo, number)
}

// browseTarget は今の画面のブラウザで開く対象の表示名・リポジトリ・番号を返す。
func (m Model) browseTarget() (string, string, int, bool) {
	switch m.screen {
	case screenCard:
		issue := m.detail.card.Issue
		return issueLabel(issue), issue.Repo, issue.Number, true
	case screenPR:
		pr := m.currentPR()
		return prLabel(pr), pr.Repo, pr.Number, true
	}
	rows := m.rows[m.tab]
	if m.cursor >= len(rows) {
		return "", "", 0, false
	}
	r := rows[m.cursor]
	label := fmt.Sprintf("%s #%d", r.repo, r.number)
	if r.isPR {
		label = fmt.Sprintf("%s PR#%d", r.repo, r.number)
	}
	return label, r.repo, r.number, true
}

// browseCmd は gh browse を別ゴルーチンで実行し、結果を browsedMsg で返す。
func browseCmd(client gh.GHClient, label, repo string, number int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
		defer cancel()
		return browsedMsg{label: label, err: client.Browse(ctx, repo, number)}
	}
}

// updateBrowsed はブラウザで開いた結果を扱う。失敗だけをフッタに赤で出す。
func (m Model) updateBrowsed(msg browsedMsg) Model {
	if msg.err != nil {
		m.writeStatus, m.writeStatusErr = msg.label+" をブラウザで開けません: "+msg.err.Error(), true
	}
	return m
}
