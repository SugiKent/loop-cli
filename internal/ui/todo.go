package ui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/sugi-loop/internal/action"
	"github.com/SugiKent/sugi-loop/internal/gh"
	"github.com/SugiKent/sugi-loop/internal/model"
)

// toggledMsg は 1 回の stage:todo 切り替えの完了。
type toggledMsg struct {
	label string
	added bool
	err   error
}

// todoKey は `t`（stage:todo の切り替え）を扱う。対象は「画面に出ている Card の Issue」の 1 規則。
// 確認は出さない（取り消しはもう一度 `t`）。付けるか外すかは action.ToggleTodo が読み直して決める。
func (m Model) todoKey() (tea.Model, tea.Cmd) {
	if m.writing {
		return m, nil
	}
	issue, ok := m.todoTarget()
	if !ok {
		return m, nil
	}
	label := issueLabel(issue)
	m.writing = true
	m.writeStatus, m.writeStatusErr = label+" の stage:todo を切り替え中", false
	return m, toggleCmd(m.client, issue.Repo, issue.Number, label)
}

// todoTarget は今の画面の切り替え対象の Issue を返す。PR 詳細と PR 単独の Card には対象が無い。
func (m Model) todoTarget() (*model.Issue, bool) {
	if m.screen == screenPR {
		return nil, false
	}
	if m.screen == screenCard {
		return m.detail.card.Issue, m.detail.card.Issue != nil
	}
	rows := m.rows[m.tab]
	if m.cursor >= len(rows) {
		return nil, false
	}
	issue := rows[m.cursor].card.Issue
	return issue, issue != nil
}

// toggleCmd は action.ToggleTodo を別ゴルーチンで実行し、結果を toggledMsg で返す。
func toggleCmd(client gh.GHClient, repo string, number int, label string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
		defer cancel()
		added, err := action.ToggleTodo(ctx, client, repo, number)
		return toggledMsg{label: label, added: added, err: err}
	}
}

// updateToggled は切り替えの完了を扱う。結果はフッタに出すだけで、Cards もラベルも書き換えない
// （反映は R / 自動更新 / 1 件再取得。ToggleTodo が毎回読み直すので取り消しは画面に依存しない）。
func (m Model) updateToggled(msg toggledMsg) Model {
	m.writing = false
	switch {
	case msg.err != nil:
		m.writeStatus, m.writeStatusErr = msg.label+" の stage:todo を切り替えられません: "+msg.err.Error(), true
	case msg.added:
		m.writeStatus, m.writeStatusErr = msg.label+" に stage:todo を付けました", false
	default:
		m.writeStatus, m.writeStatusErr = msg.label+" から stage:todo を外しました", false
	}
	return m
}
