package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// refreshTickMsg は自動更新の 1 回分の合図。Model は tick の時刻を持たず、
// 周期はコマンドの遅延で作る（壁時計を読まない）。
type refreshTickMsg struct{}

// tickCmd は interval 後に refreshTickMsg を返すコマンド。
func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg { return refreshTickMsg{} })
}

// updateTick は自動更新の tick を扱う。取得中と書き込み中は取得を始めず次の tick を待つ。
// どちらの場合も次の tick を返すので、スキップしても自動更新は止まらない。
func (m Model) updateTick() (tea.Model, tea.Cmd) {
	next := tickCmd(m.refreshInterval)
	if m.fetching || m.writing {
		return m, next
	}
	return m, tea.Batch(m.startFetch(), next)
}
