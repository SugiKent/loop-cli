package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/model"
)

// findFetched はコマンドを実行し、束なら要素も実行して fetchedMsg を探す。
// 見つけた時点で止めるので、1 秒待つ tick のコマンドは実行しない。
func findFetched(cmd tea.Cmd) (fetchedMsg, bool) {
	if cmd == nil {
		return fetchedMsg{}, false
	}
	switch msg := cmd().(type) {
	case fetchedMsg:
		return msg, true
	case tea.BatchMsg:
		for _, c := range msg {
			if fetched, ok := findFetched(c); ok {
				return fetched, true
			}
		}
	}
	return fetchedMsg{}, false
}

// tickModel は 1 秒間隔の自動更新を持ち、example の Result を返す Model を作る。
func tickModel(t *testing.T) (Model, *int) {
	t.Helper()
	calls := new(int)
	return newModelOpts(countingFetcher(exampleResult(t), calls), Options{RefreshInterval: time.Second}), calls
}

func TestTickStartsFetch(t *testing.T) {
	m, calls := tickModel(t)
	m, _ = send(m, fetchedMsg{res: exampleResult(t), at: at})
	if *calls != 0 {
		t.Fatalf("tick の前の呼び出し回数 = %d, want 0", *calls)
	}

	m, cmd := send(m, refreshTickMsg{})

	if !m.fetching {
		t.Error("tick の後に fetching が false")
	}
	if !strings.Contains(plainText(m), "取得中") {
		t.Errorf("tick の後のフッタに 取得中 が無い:\n%s", plainText(m))
	}
	if _, ok := findFetched(cmd); !ok {
		t.Fatal("返ったコマンドから fetchedMsg が得られない")
	}
	if *calls != 1 {
		t.Errorf("Fetcher の呼び出し回数 = %d, want 1", *calls)
	}
}

func TestTickDuringFetchIsSkipped(t *testing.T) {
	m, calls := tickModel(t)

	_, cmd := send(m, refreshTickMsg{})

	if cmd == nil {
		t.Fatal("次の tick のコマンドが返っていない")
	}
	if *calls != 0 {
		t.Errorf("取得中の tick で Fetcher が %d 回呼ばれた, want 0", *calls)
	}
}

func TestTickDuringWriteIsSkipped(t *testing.T) {
	m, calls := tickModel(t)
	m, _ = send(m, fetchedMsg{res: exampleResult(t), at: at},
		runeKey('2'), runeKey('t')) // バックログの issue 140 に stage:todo を与える
	if !m.writing {
		t.Fatal("t の後に writing が false")
	}

	m, _ = send(m, refreshTickMsg{})

	if m.fetching {
		t.Error("書き込み中の tick で取得が始まった")
	}
	text := plainText(m)
	if !strings.Contains(text, "切り替え中") {
		t.Errorf("フッタに 切り替え中 が残っていない:\n%s", text)
	}
	if strings.Contains(text, "取得中") {
		t.Errorf("フッタに 取得中 が出ている:\n%s", text)
	}
	if *calls != 0 {
		t.Errorf("書き込み中の tick で Fetcher が %d 回呼ばれた, want 0", *calls)
	}
}

func TestTickDoesNotReplaceOpenDetail(t *testing.T) {
	m, _ := tickModel(t)
	m, _ = send(m, fetchedMsg{res: exampleResult(t), at: at}, codeKey(tea.KeyEnter))
	if m.screen != screenCard {
		t.Fatalf("カード詳細に移っていない: screen = %d", m.screen)
	}

	m, _ = send(m, refreshTickMsg{},
		fetchedMsg{res: &fetch.Result{Cards: []model.Card{nowCard("org/web", 88, 1, at)}}, at: at.Add(time.Minute)})

	if m.screen != screenCard {
		t.Fatalf("取得完了で画面が変わった: screen = %d", m.screen)
	}
	if n := m.detail.card.Issue.Number; n != 108 {
		t.Errorf("詳細の対象 = issue %d, want 108", n)
	}

	m, _ = send(m, codeKey(tea.KeyEscape))
	rows := m.rows[model.TabNow]
	if len(rows) != 1 || rows[0].number != 88 {
		t.Errorf("Esc の後の今やるタブが新しい Result になっていない: %+v", rows)
	}
}

func TestInitStartsFetchWithAndWithoutInterval(t *testing.T) {
	for name, opts := range map[string]Options{
		"自動更新あり": {RefreshInterval: time.Second},
		"自動更新なし": {},
	} {
		t.Run(name, func(t *testing.T) {
			m := newModelOpts(countingFetcher(exampleResult(t), new(int)), opts)

			if m.Init() == nil {
				t.Error("Init がコマンドを返していない")
			}
			if !m.fetching {
				t.Error("Init 後に fetching が false")
			}
		})
	}
}

// TestDetailRefreshDoesNotRefetchGitHub は詳細画面の R が GitHub を取り直さず、
// セッションだけを取り直すことを検証する（s31 manual-refresh の MODIFIED）。
func TestDetailRefreshDoesNotRefetchGitHub(t *testing.T) {
	calls := 0
	stub := &sessionStub{entries: runningEntries()}
	card := sessionIssueCard([]model.Comment{{Body: "<!-- routine -->\nsession: " + testSessionID, AI: true}})
	m, _ := send(
		newModelOpts(countingFetcher(&fetch.Result{Cards: []model.Card{card}}, &calls),
			Options{ClaudeConfigDirs: appDirs(), SessionLog: stub.log}),
		tea.WindowSizeMsg{Width: 120, Height: 40},
		fetchedMsg{res: &fetch.Result{Cards: []model.Card{card}}, at: at})

	m, cmd := send(m, enterKey)
	if cmd == nil {
		t.Fatal("カード詳細を開いても取得のコマンドが返らない")
	}
	m, _ = send(m, cmd())

	m, cmd = send(m, runeKey('R'))
	if cmd == nil {
		t.Fatal("詳細の R でコマンドが返らない")
	}
	m, _ = send(m, cmd())

	if calls != 0 {
		t.Errorf("Fetcher の呼び出し回数 = %d, want 0", calls)
	}
	if m.fetching {
		t.Error("fetching が true になった")
	}
	if !m.at.Equal(at) {
		t.Errorf("最終更新時刻が変わった: %v", m.at)
	}
	if m.screen != screenCard {
		t.Errorf("画面 = %d, want カード詳細", m.screen)
	}
	if stub.calls != 2 {
		t.Errorf("claude の呼び出し回数 = %d, want 2", stub.calls)
	}
}

// TestRefreshDoesNothingOnNarrowDetailAndOverlays は右ペインを出さない幅の詳細画面と
// 確認画面 / ヘルプ画面で R が何もしないことを検証する。
func TestRefreshDoesNothingOnNarrowDetailAndOverlays(t *testing.T) {
	calls := 0
	stub := &sessionStub{entries: runningEntries()}
	res := exampleResult(t)
	m, _ := send(
		newModelOpts(countingFetcher(res, &calls), Options{ClaudeConfigDirs: appDirs(), SessionLog: stub.log}),
		tea.WindowSizeMsg{Width: 80, Height: 40}, fetchedMsg{res: res, at: at})

	card, _ := send(m, enterKey)
	pr, _ := send(card, enterKey)
	help, _ := send(m, runeKey('?'))
	confirm, _ := confirmModel(t)

	for name, target := range map[string]Model{"カード詳細": card, "PR 詳細": pr, "ヘルプ": help, "確認": confirm} {
		next, cmd := send(target, runeKey('R'))
		if cmd != nil {
			t.Errorf("%s で R がコマンドを返した: %T", name, cmd())
		}
		if next.screen != target.screen {
			t.Errorf("%s で R が画面を変えた: %d -> %d", name, target.screen, next.screen)
		}
		if next.fetching {
			t.Errorf("%s で fetching が true になった", name)
		}
	}
	if calls != 0 {
		t.Errorf("Fetcher の呼び出し回数 = %d, want 0", calls)
	}
	if stub.calls != 0 {
		t.Errorf("claude の呼び出し回数 = %d, want 0", stub.calls)
	}
}
