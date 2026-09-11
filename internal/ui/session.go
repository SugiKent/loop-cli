package ui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/loop-cli/internal/claude"
	"github.com/SugiKent/loop-cli/internal/model"
)

// 右ペインの寸法と取得の上限（s31 design.md D3 / 未決事項 2・7）。
const (
	sessionPaneWidth = 40 // 右ペインの表示幅
	twoPaneMin       = 120
	sessionLogLines  = 12               // 「直近の動き」に出すログの行数
	sessionTimeout   = 30 * time.Second // 1 回の claude 実行の打ち切り
	sessionURLPrefix = "https://claude.ai/code/"
)

// SessionLogger は 1 セッションのログの取得。cmd/loop-cli が claude.Client を閉じて渡す。
type SessionLogger func(ctx context.Context, configDir, sessionID string) ([]claude.Entry, error)

// sessionResult は 1 セッションの取得結果。プロセスの中だけに持ち、スナップショットには書かない。
// at がゼロ値なら 1 度も取得に成功していない。reason は取得できない理由（空なら成功）。
type sessionResult struct {
	entries []claude.Entry
	at      time.Time
	reason  string
}

// sessionFetchedMsg は 1 セッションの取得の完了。
type sessionFetchedMsg struct {
	id        string
	configDir string
	entries   []claude.Entry
	err       error
	at        time.Time
}

// twoPane は詳細画面を左右 2 ペインで描くか。
func (m Model) twoPane() bool { return m.width >= twoPaneMin }

// leftWidth は左ペインの幅。1 ペインのときは端末の幅そのもの。
func (m Model) leftWidth() int {
	if !m.twoPane() {
		return max(m.width, 0)
	}
	return m.width - sessionPaneWidth - 1
}

// detailRepo は詳細の対象のリポジトリ。
func (m Model) detailRepo() string {
	if m.screen == screenPR {
		return m.currentPR().Repo
	}
	if issue := m.detail.card.Issue; issue != nil {
		return issue.Repo
	}
	return ""
}

// sessionID は詳細の対象に紐づく Claude Code セッションの ID。
// PR 詳細は PR 本文の URL から、カード詳細は issue の routine コメントの最新の session: 行から採る。
func (m Model) sessionID() (string, bool) {
	switch m.screen {
	case screenPR:
		return model.SessionURLID(m.currentPR().Body)
	case screenCard:
		if issue := m.detail.card.Issue; issue != nil {
			return model.LatestSessionID(issue.Comments)
		}
	}
	return "", false
}

// sessionOnOpenCmd は詳細に移ったときの取得。その ID の結果をまだ持っていなければ 1 回だけ始める。
func (m *Model) sessionOnOpenCmd() tea.Cmd {
	id, ok := m.sessionID()
	if !ok {
		return nil
	}
	if _, done := m.sessions[id]; done {
		return nil
	}
	return m.startSession(id)
}

// sessionRefreshKey は詳細画面の R。結果を持っていても取り直す。
func (m Model) sessionRefreshKey() (tea.Model, tea.Cmd) {
	id, ok := m.sessionID()
	if !ok {
		return m, nil
	}
	return m, m.startSession(id)
}

// startSession は claude の起動を抑える条件を見てから取得のコマンドを返す。
// 右ペインを出さない幅・claude_config_dir が無い・同じプロファイルが取得中・
// そのプロファイルが制限中、のいずれかなら起動しない。
func (m *Model) startSession(id string) tea.Cmd {
	if m.sessionLog == nil || !m.twoPane() {
		return nil
	}
	dir := m.claudeDirs[m.detailRepo()]
	if dir == "" || m.sessionBusy[dir] {
		return nil
	}
	if _, limited := m.sessionLimited[dir]; limited {
		return nil
	}
	m.sessionBusy[dir] = true
	return sessionCmd(m.sessionLog, dir, id)
}

// sessionCmd はログの取得を別ゴルーチンで実行し、完了を sessionFetchedMsg で返す。
func sessionCmd(log SessionLogger, dir, id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
		defer cancel()
		entries, err := log(ctx, dir, id)
		return sessionFetchedMsg{id: id, configDir: dir, entries: entries, err: err, at: time.Now()}
	}
}

// updateSessionFetched は取得の完了を扱う。失敗しても前回の内容は残し、理由の行だけを足す。
// Cards・最終更新時刻・詳細の対象は変えない。
func (m Model) updateSessionFetched(msg sessionFetchedMsg) Model {
	delete(m.sessionBusy, msg.configDir)
	if msg.err == nil {
		m.sessions[msg.id] = sessionResult{entries: msg.entries, at: msg.at}
		return m
	}
	// 週の利用上限に当たったプロファイルは、プロセスが動いている間は起動しない。
	var limit *claude.LimitError
	if errors.As(msg.err, &limit) {
		m.sessionLimited[msg.configDir] = limit.Resets
	}
	res := m.sessions[msg.id]
	res.reason = sessionReason(msg.err)
	m.sessions[msg.id] = res
	return m
}

// sessionReason は取得の失敗を右ペインに出す 1 行にする。
func sessionReason(err error) string {
	var httpErr *claude.HTTPError
	var limit *claude.LimitError
	switch {
	case errors.Is(err, exec.ErrNotFound):
		return "claude が見つかりません"
	case errors.Is(err, claude.ErrNotLoggedIn):
		return "未ログインです"
	case errors.As(err, &limit):
		return "制限中 " + limit.Resets
	case errors.As(err, &httpErr) && httpErr.Status == 404:
		return "HTTP 404 プロファイルが違う可能性があります"
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("%.0f 秒で打ち切りました", sessionTimeout.Seconds())
	}
	return err.Error()
}

// sessionPaneLines は右ペインの行を作る。各行は右ペインの幅で切り詰め、
// h 行に収まらない分は末尾から落とす（右ペインは独立したスクロールを持たない）。
func (m Model) sessionPaneLines(h int) []string {
	lines := []string{"Claude セッション"}
	switch id, ok := m.sessionID(); {
	case m.claudeDirs[m.detailRepo()] == "":
		lines = append(lines, wrapToWidth("claude_config_dir が未設定です", sessionPaneWidth)...)
	case !ok:
		lines = append(lines, "セッション ID が見つかりません")
	default:
		lines = append(lines, m.sessionLines(id)...)
	}
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, sessionPaneWidth, "…")
	}
	return cut(lines, h)
}

// sessionLines は取得結果と取得の状態の行。取得に成功したことがなければ取得の状態だけを出す。
func (m Model) sessionLines(id string) []string {
	res := m.sessions[id]
	var lines []string
	if !res.at.IsZero() {
		lines = append(lines, "状態: "+sessionStatus(res.entries))
		if len(res.entries) > 0 {
			last := res.entries[0]
			lines = append(lines, fmt.Sprintf("最終更新: %s  %s",
				last.At.In(m.location()).Format("15:04"), Elapsed(res.at, last.At)))
		}
		lines = append(lines, id, "", "直近の動き")
		for _, e := range res.entries[:min(len(res.entries), sessionLogLines)] {
			lines = append(lines, sessionLogLine(e, m.location()))
		}
		if answer, ok := claude.FinalAnswer(res.entries); ok {
			lines = append(lines, "最終回答: "+answer)
		}
		lines = append(lines, "取得: "+res.at.In(m.location()).Format("15:04"))
	} else if res.reason == "" {
		if m.sessionBusy[m.claudeDirs[m.detailRepo()]] {
			lines = append(lines, "取得: 取得中")
		} else {
			lines = append(lines, "R で取得")
		}
	}
	// 理由だけは切り詰めず折り返す。40 列に切ると、何をすればよいかの部分が消えるため。
	if res.reason != "" {
		lines = append(lines, wrapToWidth(res.reason, sessionPaneWidth)...)
	}
	return lines
}

// sessionStatus はログの最新の行の種類から決める状態の語（s31 design.md D1）。
func sessionStatus(entries []claude.Entry) string {
	if len(entries) == 0 {
		return "不明"
	}
	switch entries[0].Kind {
	case "result":
		return "終了"
	case "env[info]", "init":
		return "起動中"
	}
	return "実行中"
}

// sessionLogLine は「直近の動き」の 1 行。tool_use はツール名と本文の先頭 1 行、
// assistant は本文の先頭 1 行を足し、他の種類は種類だけを出す。
func sessionLogLine(e claude.Entry, loc *time.Location) string {
	line := e.At.In(loc).Format("15:04") + " " + e.Kind
	switch e.Kind {
	case "tool_use":
		return line + " " + e.Tool + ": " + firstLine(e.Text)
	case "assistant":
		return line + " " + firstLine(e.Text)
	}
	return line
}

// sessionURL はセッション ID から claude.ai の URL を作る。
func sessionURL(id string) string { return sessionURLPrefix + id }
