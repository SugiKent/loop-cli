package ui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/loop-cli/internal/action"
	"github.com/SugiKent/loop-cli/internal/gh"
)

// closeState は `c` を押してから確認画面を抜けるまでの状態。
// 対象は押した時点の値を凍結する（確認中に取得が完了して Cards が入れ替わっても差し替えない）。
type closeState struct {
	target action.Target
	label  string
	title  string
	labels []string
	notes  []string
	from   screen // `c` を押した画面。確認画面から戻る先
}

// closedMsg は 1 回の close の完了。
type closedMsg struct {
	label string
	err   error
}

// closeKey は `c`（close）を扱う。対象は画面が見せているものに決まる。
// 押した時点では gh を呼ばない（画面に出ている issue と PR は取得時点で必ず open。design.md）。
func (m Model) closeKey() (tea.Model, tea.Cmd) {
	if m.writing {
		return m, nil
	}
	s, ok := m.closeTarget()
	if !ok {
		return m, nil
	}
	s.from = m.screen
	s.notes = action.CheckClose(s.target, s.labels)
	m.close = s
	m.screen = screenCloseConfirm
	// 直前の書き込みの赤字を確認画面と戻り先に残さない（`a` と同じ）。
	m.writeStatus, m.writeStatusErr = "", false
	return m, nil
}

// closeTarget は今の画面の close 対象を返す。カード詳細は Issue、PR 詳細はその PR、
// キュー画面は選択行の主体（issue でも PR でもよい）。
func (m Model) closeTarget() (closeState, bool) {
	switch m.screen {
	case screenCard:
		issue := m.detail.card.Issue
		return closeState{
			target: targetOfIssue(issue), label: issueLabel(issue), title: issue.Title, labels: issue.Labels,
		}, true
	case screenPR:
		pr := m.currentPR()
		return closeState{
			target: targetOfPR(pr), label: prLabel(pr), title: pr.Title, labels: pr.Labels,
		}, true
	}
	rows := m.rows[m.tab]
	if m.cursor >= len(rows) {
		return closeState{}, false
	}
	r := rows[m.cursor]
	label := fmt.Sprintf("%s #%d", r.repo, r.number)
	if r.isPR {
		label = fmt.Sprintf("%s PR#%d", r.repo, r.number)
	}
	return closeState{
		target: action.Target{Repo: r.repo, Number: r.number, IsPR: r.isPR},
		label:  label, title: r.title, labels: r.labels,
	}, true
}

// updateCloseConfirmKey は close の確認画面のキーを扱う（q / Ctrl+C は Update が先に処理する）。
func (m Model) updateCloseConfirmKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y":
		m.backFromClose()
		m.writing = true
		m.writeStatus, m.writeStatusErr = m.close.label+" を close 中", false
		return m, closeCmd(m.client, m.close.target, m.close.label)
	case "esc":
		m.backFromClose()
		m.writeStatus, m.writeStatusErr = "close を中止しました", false
	}
	return m, nil
}

// backFromClose は確認画面から戻り先へ戻る。確認画面の間に端末の大きさが変わっていることが
// あるので、戻り先が詳細画面なら本文領域を作り直す（ヘルプ画面から戻るときと同じ扱い）。
func (m *Model) backFromClose() {
	m.screen = m.close.from
	if m.screen == screenCard || m.screen == screenPR {
		m.refreshDetail()
	}
}

// closeCmd は action.Close を別ゴルーチンで実行し、結果を closedMsg で返す。
func closeCmd(client gh.GHClient, target action.Target, label string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
		defer cancel()
		return closedMsg{label: label, err: action.Close(ctx, client, target)}
	}
}

// updateClosed は close の完了を扱う。結果はフッタに出すだけで、再取得はしない（1 件再取得は s18）。
func (m Model) updateClosed(msg closedMsg) Model {
	m.writing = false
	if msg.err != nil {
		m.writeStatus, m.writeStatusErr = msg.label+" の close に失敗: "+msg.err.Error(), true
		return m
	}
	m.writeStatus, m.writeStatusErr = msg.label+" を close しました", false
	return m
}

// renderCloseConfirm は close 前の確認を全画面で描く。本文は出さず、対象と種別と
// ラベルと注意だけを出す（何を閉じようとしているかが `y` の前に読めればよい）。
func (m Model) renderCloseConfirm() string {
	s := m.close
	kind := "issue"
	if s.target.IsPR {
		kind = "PR"
	}
	labels := "なし"
	if len(s.labels) > 0 {
		labels = strings.Join(s.labels, " ")
	}
	lines := []string{
		"close の確認: " + s.label + "  " + s.title,
		"種別: " + kind,
		"labels: " + labels,
	}
	for _, n := range s.notes {
		lines = append(lines, "注意: "+n)
	}

	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "…")
	}
	lines = cut(lines, max(m.height-1, 0))
	return strings.Join(append(lines, m.footer("y close  Esc 中止  q 終了")), "\n")
}
