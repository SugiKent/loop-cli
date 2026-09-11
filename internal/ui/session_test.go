package ui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/claude"
	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/model"
)

const (
	testSessionID  = "session_01N9YWTYcwFdwhLD38CdASgA"
	testConfigDir  = "/home/alice/.claude-personal"
	otherConfigDir = "/home/alice/.claude-max"
)

// sessionStub は claude の起動を数えるスタブ。
type sessionStub struct {
	calls   int
	dirs    []string
	ids     []string
	entries []claude.Entry
	err     error
}

func (s *sessionStub) log(_ context.Context, dir, id string) ([]claude.Entry, error) {
	s.calls++
	s.dirs = append(s.dirs, dir)
	s.ids = append(s.ids, id)
	return s.entries, s.err
}

// sessionModel は claude_config_dir を設定した Model を幅 w・高さ h で返す。
// stub が nil なら SessionLog を渡さない（claude を起動する手段が無い状態）。
func sessionModel(w, h int, stub *sessionStub, dirs map[string]string, cards []model.Card) Model {
	opts := Options{ClaudeConfigDirs: dirs}
	if stub != nil {
		opts.SessionLog = stub.log
	}
	m, _ := send(newModelOpts(nil, opts), tea.WindowSizeMsg{Width: w, Height: h},
		fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})
	return m
}

// appDirs は org/app だけに claude_config_dir を設定した対応表。
func appDirs() map[string]string { return map[string]string{"org/app": testConfigDir} }

// sessionPRCard は本文にセッションの URL を持つ PR 単独の Card。
func sessionPRCard() model.Card {
	pr := prOf(131, "OPEN", []string{model.LabelApply})
	pr.Body = "未確定の判断: 0 件\n\n" + sessionURL(testSessionID) + " で進めています。"
	pr.Result = model.Result{Situation: model.SituationD, Priority: 2, Tab: model.TabNow, Summary: "確認する"}
	return model.Card{PRs: []model.PR{pr}, Result: pr.Result}
}

// sessionIssueCard は routine コメントに session: 行を持つ issue の Card。
func sessionIssueCard(comments []model.Comment) model.Card {
	issue := &model.Issue{Number: 108, Title: "手書き", UpdatedAt: at, Comments: comments}
	return issueCard(issue, nil, "回答する")
}

// logAt は at と同じタイムゾーンの時刻。
func logAt(hour, minute int) time.Time {
	return time.Date(2026, 9, 5, hour, minute, 0, 0, at.Location())
}

// runningEntries は実行中のセッションのログ（新しい順）。
func runningEntries() []claude.Entry {
	return []claude.Entry{
		{At: logAt(12, 3), Kind: "tool_use", Tool: "Bash", Text: "git push -u origin HEAD"},
		{At: logAt(12, 2), Kind: "assistant", Text: "変更を push します"},
		{At: logAt(12, 0), Kind: "env[info]", Text: "環境を起動しました"},
	}
}

// fetched は取得の完了を Model に渡す。at は取得時刻。
func fetched(m Model, entries []claude.Entry, err error) Model {
	m, _ = send(m, sessionFetchedMsg{
		id: testSessionID, configDir: testConfigDir, entries: entries, err: err, at: at})
	return m
}

// rightPane は 2 ペインの右側だけを連結して返す。
func rightPane(m Model) string {
	var out []string
	for _, l := range plain(m) {
		if _, right, ok := strings.Cut(l, "│"); ok {
			out = append(out, right)
		}
	}
	return strings.Join(out, "\n")
}

// openPRDetail は PR 単独の Card の詳細を開く。
func openPRDetail(m Model) (Model, tea.Cmd) { return send(m, enterKey) }

func TestSessionPaneShowsRunningSession(t *testing.T) {
	stub := &sessionStub{entries: runningEntries()}
	m, _ := openPRDetail(sessionModel(120, 40, stub, appDirs(), []model.Card{sessionPRCard()}))
	m = fetched(m, runningEntries(), nil)

	wantOrder(t, []string{rightPane(m)},
		"Claude セッション", "状態: 実行中", "最終更新: 12:03  1m", "直近の動き", "Bash", "取得: 12:04")
}

func TestSessionPaneShowsFinalAnswer(t *testing.T) {
	entries := []claude.Entry{
		{At: logAt(12, 3), Kind: "result", Text: "success is_error=false turns=110 duration=1168s — CI が緑になりました"},
		{At: logAt(12, 0), Kind: "env[info]", Text: "環境を起動しました"},
	}
	stub := &sessionStub{entries: entries}
	m, _ := openPRDetail(sessionModel(120, 40, stub, appDirs(), []model.Card{sessionPRCard()}))
	m = fetched(m, entries, nil)

	pane := rightPane(m)
	for _, want := range []string{"状態: 終了", "最終回答: CI が緑になりました"} {
		if !strings.Contains(pane, want) {
			t.Errorf("右ペインに %q が無い:\n%s", want, pane)
		}
	}
}

func TestSessionPaneShowsAtMost12LogLines(t *testing.T) {
	entries := make([]claude.Entry, 200)
	for i := range entries {
		entries[i] = claude.Entry{At: logAt(12, 3), Kind: "tool_use", Tool: fmt.Sprintf("T%03d", i)}
	}
	stub := &sessionStub{entries: entries}
	m, _ := openPRDetail(sessionModel(120, 40, stub, appDirs(), []model.Card{sessionPRCard()}))
	m = fetched(m, entries, nil)

	lines := strings.Split(rightPane(m), "\n")
	start := -1
	for i, l := range lines {
		if strings.Contains(l, "直近の動き") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("直近の動きの見出しが無い:\n%s", rightPane(m))
	}
	got := 0
	for _, l := range lines[start+1:] {
		if !strings.Contains(l, "tool_use") {
			break
		}
		got++
	}
	if got != sessionLogLines {
		t.Fatalf("ログの行数 = %d, want %d:\n%s", got, sessionLogLines, rightPane(m))
	}
	// 最も新しい 12 件（T000 から T011）が出る。
	for i := range sessionLogLines {
		if !strings.Contains(lines[start+1+i], fmt.Sprintf("T%03d", i)) {
			t.Errorf("%d 行目 = %q, want T%03d", i+1, lines[start+1+i], i)
		}
	}
}

func TestSessionPaneWithoutResultShowsPrompt(t *testing.T) {
	// SessionLog が無いので開いても取得は始まらない（取得の状態は「まだ 1 度も取得していない」）。
	m, _ := openPRDetail(sessionModel(120, 40, nil, appDirs(), []model.Card{sessionPRCard()}))

	pane := rightPane(m)
	if !strings.Contains(pane, "R で取得") {
		t.Errorf("右ペインに `R で取得` が無い:\n%s", pane)
	}
	if strings.Contains(pane, "状態:") {
		t.Errorf("取得していないのに状態の行がある:\n%s", pane)
	}
}

func TestSessionPaneWithoutConfigDir(t *testing.T) {
	stub := &sessionStub{entries: runningEntries()}
	m, _ := openPRDetail(sessionModel(120, 40, stub, nil, []model.Card{sessionPRCard()}))

	if !strings.Contains(rightPane(m), "claude_config_dir が未設定です") {
		t.Errorf("右ペインに未設定の理由が無い:\n%s", rightPane(m))
	}
	if stub.calls != 0 {
		t.Errorf("claude の呼び出し回数 = %d, want 0", stub.calls)
	}
	// 左ペインとフッタは今までどおり出る。
	lines := plain(m)
	if _, ok := lineWith(lines, "org/app PR#131"); !ok {
		t.Errorf("左ペインの PR のタイトル行が無い:\n%s", strings.Join(lines, "\n"))
	}
	if !strings.Contains(lines[len(lines)-1], "Esc 戻る") {
		t.Errorf("フッタが出ていない: %q", lines[len(lines)-1])
	}
}

func TestSessionPaneNotFoundSessionID(t *testing.T) {
	card := sessionIssueCard([]model.Comment{{Body: "session: session_01CCC と書いてあるだけの人のコメント"}})
	m, _ := send(sessionModel(120, 40, &sessionStub{}, appDirs(), []model.Card{card}), enterKey)

	if !strings.Contains(rightPane(m), "セッション ID が見つかりません") {
		t.Errorf("右ペインに見つからない旨が無い:\n%s", rightPane(m))
	}
}

func TestSessionPaneCardDetailUsesLatestRoutineComment(t *testing.T) {
	card := sessionIssueCard([]model.Comment{
		{Body: "<!-- routine -->\nsession: session_01AAA", AI: true},
		{Body: "&lt;!-- routine --&gt;\nsession: session_01BBB", AI: true},
	})
	m, _ := send(sessionModel(120, 40, &sessionStub{}, appDirs(), []model.Card{card}), enterKey)

	if got, ok := m.sessionID(); !ok || got != "session_01BBB" {
		t.Errorf("セッション ID = %q, %v; want session_01BBB, true", got, ok)
	}
}

func TestSessionPanePRDetailUsesBodyURL(t *testing.T) {
	m, _ := openPRDetail(sessionModel(120, 40, &sessionStub{}, appDirs(), []model.Card{sessionPRCard()}))

	if got, ok := m.sessionID(); !ok || got != testSessionID {
		t.Errorf("セッション ID = %q, %v; want %s, true", got, ok, testSessionID)
	}
}

func TestSessionPaneShows404AsProfileMismatch(t *testing.T) {
	stub := &sessionStub{err: &claude.HTTPError{Status: 404, Body: `{"error":{"type":"not_found_error"}}`}}
	m, _ := openPRDetail(sessionModel(120, 40, stub, appDirs(), []model.Card{sessionPRCard()}))
	m = fetched(m, nil, stub.err)

	pane := rightPane(m)
	for _, want := range []string{"HTTP 404", "プロファイルが違う可能性があります"} {
		if !strings.Contains(pane, want) {
			t.Errorf("右ペインに %q が無い:\n%s", want, pane)
		}
	}
}

func TestSessionPaneLimitKeepsPreviousResult(t *testing.T) {
	stub := &sessionStub{entries: runningEntries()}
	m, _ := openPRDetail(sessionModel(120, 40, stub, appDirs(), []model.Card{sessionPRCard()}))
	m = fetched(m, runningEntries(), nil)

	m, cmd := send(m, runeKey('R'))
	if cmd == nil {
		t.Fatal("R で取得のコマンドが返らない")
	}
	m = fetched(m, nil, &claude.LimitError{Resets: "Sep 14 at 2am (Asia/Tokyo)"})

	pane := rightPane(m)
	for _, want := range []string{"状態: 実行中", "最終更新: 12:03", "制限中", "Sep 14 at 2am (Asia/Tokyo)"} {
		if !strings.Contains(pane, want) {
			t.Errorf("右ペインに %q が無い:\n%s", want, pane)
		}
	}
}

func TestSessionFetchFailureKeepsOtherKeysWorking(t *testing.T) {
	stub := &sessionStub{err: errors.New("起動できません")}
	m := sessionModel(120, 40, stub, appDirs(),
		[]model.Card{sessionPRCard(), nowCard("org/app", 2, 1, at.Add(-time.Hour))})
	before := m.cards
	m, _ = send(m, enterKey)
	m = fetched(m, nil, stub.err)

	m, _ = send(m, escKey)
	m, _ = send(m, runeKey('j'))
	if m.screen != screenQueue || m.cursor != 1 {
		t.Errorf("screen = %d cursor = %d, want キュー / 1", m.screen, m.cursor)
	}
	if len(m.cards) != len(before) {
		t.Errorf("Cards が変わった: %d 枚, want %d 枚", len(m.cards), len(before))
	}
}

func TestSessionFetchOncePerOpen(t *testing.T) {
	stub := &sessionStub{entries: runningEntries()}
	m, cmd := openPRDetail(sessionModel(120, 40, stub, appDirs(), []model.Card{sessionPRCard()}))
	if cmd == nil {
		t.Fatal("詳細を開いても取得のコマンドが返らない")
	}
	m, _ = send(m, cmd())
	if stub.calls != 1 {
		t.Fatalf("claude の呼び出し回数 = %d, want 1", stub.calls)
	}
	firstAt := m.sessions[testSessionID].at

	m, _ = send(m, escKey)
	m, cmd = send(m, enterKey)
	if cmd != nil {
		t.Errorf("2 回目に開いたときにコマンドが返った: %T", cmd())
	}
	if stub.calls != 1 {
		t.Errorf("claude の呼び出し回数 = %d, want 1", stub.calls)
	}
	if got := m.sessions[testSessionID].at; !got.Equal(firstAt) {
		t.Errorf("取得時刻が変わった: %v, want %v", got, firstAt)
	}
}

func TestSessionRefreshKeyRefetches(t *testing.T) {
	stub := &sessionStub{entries: runningEntries()}
	m, cmd := openPRDetail(sessionModel(120, 40, stub, appDirs(), []model.Card{sessionPRCard()}))
	m, _ = send(m, cmd())

	m, cmd = send(m, runeKey('R'))
	if cmd == nil {
		t.Fatal("詳細の R でコマンドが返らない")
	}
	// 2 回目は別のログを返させ、右ペインが取り直した結果に変わることを確かめる。
	stub.entries = []claude.Entry{{At: logAt(12, 10), Kind: "result", Text: "success — 完了しました"}}
	m, _ = send(m, cmd())
	if stub.calls != 2 {
		t.Errorf("claude の呼び出し回数 = %d, want 2", stub.calls)
	}
	if !strings.Contains(rightPane(m), "最終回答: 完了しました") {
		t.Errorf("取り直した結果が右ペインに出ていない:\n%s", rightPane(m))
	}
}

func TestQueueRefreshAndTickDoNotStartClaude(t *testing.T) {
	stub := &sessionStub{entries: runningEntries()}
	m := sessionModel(120, 40, stub, appDirs(), []model.Card{sessionPRCard()})
	m.refreshInterval = time.Minute

	m, cmd := send(m, runeKey('R'))
	if cmd != nil {
		_ = cmd()
	}
	m, _ = send(m, fetchedMsg{res: &fetch.Result{Cards: []model.Card{sessionPRCard()}}, at: at})
	if _, cmd = send(m, refreshTickMsg{}); cmd != nil {
		_ = cmd()
	}
	if stub.calls != 0 {
		t.Errorf("claude の呼び出し回数 = %d, want 0", stub.calls)
	}
}

func TestNarrowTerminalDoesNotFetchSession(t *testing.T) {
	stub := &sessionStub{entries: runningEntries()}
	m, cmd := openPRDetail(sessionModel(80, 40, stub, appDirs(), []model.Card{sessionPRCard()}))
	if cmd != nil {
		t.Fatalf("幅 80 でコマンドが返った: %T", cmd())
	}
	if stub.calls != 0 {
		t.Errorf("claude の呼び出し回数 = %d, want 0", stub.calls)
	}
	for _, l := range plain(m) {
		if strings.Contains(l, "│") {
			t.Fatalf("幅 80 で縦の区切り線がある: %q", l)
		}
	}
}

func TestSessionRefreshDuringFetchIsIgnored(t *testing.T) {
	stub := &sessionStub{entries: runningEntries()}
	m, cmd := openPRDetail(sessionModel(120, 40, stub, appDirs(), []model.Card{sessionPRCard()}))
	if cmd == nil {
		t.Fatal("詳細を開いても取得のコマンドが返らない")
	}
	// 結果を受け取る前に R を押す。
	if _, second := send(m, runeKey('R')); second != nil {
		t.Errorf("取得中の R でコマンドが返った: %T", second())
	}
	if _, err := cmd().(sessionFetchedMsg); !err {
		t.Fatal("取得のコマンドが sessionFetchedMsg を返さない")
	}
	if stub.calls != 1 {
		t.Errorf("claude の呼び出し回数 = %d, want 1", stub.calls)
	}
}

func TestLimitedProfileDoesNotStart(t *testing.T) {
	stub := &sessionStub{entries: runningEntries()}
	m, _ := openPRDetail(sessionModel(120, 40, stub, appDirs(), []model.Card{sessionPRCard()}))
	m = fetched(m, nil, &claude.LimitError{Resets: "Sep 14 at 2am (Asia/Tokyo)"})
	calls := stub.calls

	m, cmd := send(m, runeKey('R'))
	if cmd != nil {
		t.Errorf("制限中の R でコマンドが返った: %T", cmd())
	}
	m, _ = send(m, escKey)
	m, cmd = send(m, enterKey)
	if cmd != nil {
		t.Errorf("制限中に開き直してコマンドが返った: %T", cmd())
	}
	if stub.calls != calls {
		t.Errorf("claude の呼び出し回数 = %d, want %d", stub.calls, calls)
	}
	if !strings.Contains(rightPane(m), "制限中") {
		t.Errorf("右ペインに制限中が出ていない:\n%s", rightPane(m))
	}
}

func TestOtherProfileStillFetchesWhileLimited(t *testing.T) {
	stub := &sessionStub{entries: runningEntries()}
	other := sessionPRCard()
	other.PRs[0].Repo = "org/web"
	other.PRs[0].Number = 132
	other.PRs[0].Body = sessionURL("session_01OTHER") + " で進めています。"
	dirs := map[string]string{"org/app": testConfigDir, "org/web": otherConfigDir}
	m := sessionModel(120, 40, stub, dirs, []model.Card{sessionPRCard(), other})

	m, _ = send(m, enterKey)
	m = fetched(m, nil, &claude.LimitError{Resets: "Sep 14 at 2am (Asia/Tokyo)"})
	calls := stub.calls

	m, _ = send(m, escKey)
	m, _ = send(m, runeKey('j'))
	m, cmd := send(m, enterKey)
	if cmd == nil {
		t.Fatal("別のプロファイルで取得のコマンドが返らない")
	}
	if _, _ = send(m, cmd()); stub.calls != calls+1 {
		t.Errorf("claude の呼び出し回数 = %d, want %d", stub.calls, calls+1)
	}
	if stub.dirs[len(stub.dirs)-1] != otherConfigDir {
		t.Errorf("使ったプロファイル = %q, want %q", stub.dirs[len(stub.dirs)-1], otherConfigDir)
	}
}

// TestSessionReasonCoversEveryCase は取得できない理由の 1 行が場合ごとに変わることを検証する。
func TestSessionReasonCoversEveryCase(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"claude が無い", fmt.Errorf("claude が見つかりません: %w", exec.ErrNotFound), "claude が見つかりません"},
		{"未ログイン", claude.ErrNotLoggedIn, "未ログインです"},
		{"404", &claude.HTTPError{Status: 404}, "HTTP 404 プロファイルが違う可能性があります"},
		{"利用上限", &claude.LimitError{Resets: "Sep 14 at 2am (Asia/Tokyo)"}, "制限中 Sep 14 at 2am (Asia/Tokyo)"},
		{"打ち切り", fmt.Errorf("claude: %w", context.DeadlineExceeded), "30 秒で打ち切りました"},
		{"その他", errors.New("起動できません"), "起動できません"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionReason(tt.err); got != tt.want {
				t.Errorf("sessionReason = %q, want %q", got, tt.want)
			}
		})
	}
}
