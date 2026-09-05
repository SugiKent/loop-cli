package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/sugi-loop/internal/fetch"
	"github.com/SugiKent/sugi-loop/internal/model"
)

func key(k tea.Key) tea.KeyPressMsg { return tea.KeyPressMsg(k) }

func runeKey(r rune) tea.KeyPressMsg { return key(tea.Key{Code: r, Text: string(r)}) }

func codeKey(code rune) tea.KeyPressMsg { return key(tea.Key{Code: code}) }

// send は msg を順に与えて Model と最後のコマンドを返す。
func send(m Model, msgs ...tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	for _, msg := range msgs {
		var next tea.Model
		next, cmd = m.Update(msg)
		m = next.(Model)
	}
	return m, cmd
}

// loaded は Card 群を取得完了として渡した Model を返す。
func loaded(cards []model.Card) Model {
	m, _ := send(newModel(nil), fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})
	return m
}

func threeNowCards() []model.Card {
	return []model.Card{
		nowCard("org/app", 1, 1, at.Add(-1*time.Hour)),
		nowCard("org/app", 2, 1, at.Add(-2*time.Hour)),
		nowCard("org/app", 3, 1, at.Add(-3*time.Hour)),
	}
}

func TestCursorMovesWithJKAndArrows(t *testing.T) {
	m := loaded(threeNowCards())

	want := []int{1, 2, 1, 2, 1}
	keys := []tea.Msg{runeKey('j'), runeKey('j'), runeKey('k'), codeKey(tea.KeyDown), codeKey(tea.KeyUp)}
	for i, k := range keys {
		m, _ = send(m, k)
		if m.cursor != want[i] {
			t.Fatalf("%d 回目の入力後の cursor = %d, want %d", i+1, m.cursor, want[i])
		}
	}
}

func TestCursorStopsAtEnds(t *testing.T) {
	m := loaded(threeNowCards()[:2])

	m, _ = send(m, runeKey('j'), runeKey('j'), runeKey('j'))
	if m.cursor != 1 {
		t.Errorf("末尾で止まっていない: cursor = %d, want 1", m.cursor)
	}
	m, _ = send(m, runeKey('k'), runeKey('k'), runeKey('k'))
	if m.cursor != 0 {
		t.Errorf("先頭で止まっていない: cursor = %d, want 0", m.cursor)
	}
}

func TestTabSwitching(t *testing.T) {
	m := newModel(nil)

	keys := []tea.Msg{runeKey('2'), runeKey('4'), codeKey(tea.KeyTab), runeKey('1'), runeKey('3')}
	want := []model.Tab{model.TabBacklog, model.TabAbnormal, model.TabNow, model.TabNow, model.TabInProgress}
	for i, k := range keys {
		m, _ = send(m, k)
		if m.tab != want[i] {
			t.Fatalf("%d 回目の入力後のタブ = %q, want %q", i+1, m.tab, want[i])
		}
	}
}

func TestUnimplementedKeysDoNothing(t *testing.T) {
	base := loaded(threeNowCards()[:2])
	base, _ = send(base, runeKey('j'))

	keys := map[string]tea.Msg{
		"esc": codeKey(tea.KeyEscape),
		"v":   runeKey('v'), "m": runeKey('m'), "n": runeKey('n'),
		"s": runeKey('s'), "A": runeKey('A'), "g": runeKey('g'), "x": runeKey('x'),
		"/": runeKey('/'),
	}
	for name, k := range keys {
		t.Run(name, func(t *testing.T) {
			got, cmd := send(base, k)
			if cmd != nil {
				t.Errorf("%s でコマンドが返った: %T", name, cmd())
			}
			if got.screen != screenQueue {
				t.Errorf("%s で画面が変わった: screen=%d", name, got.screen)
			}
			if got.tab != base.tab || got.cursor != base.cursor || len(got.cards) != len(base.cards) {
				t.Errorf("%s で状態が変わった: tab=%q cursor=%d cards=%d", name, got.tab, got.cursor, len(got.cards))
			}
		})
	}
}

func TestQuitKeys(t *testing.T) {
	for name, k := range map[string]tea.Msg{
		"q":      runeKey('q'),
		"ctrl+c": key(tea.Key{Code: 'c', Mod: tea.ModCtrl}),
	} {
		t.Run(name, func(t *testing.T) {
			_, cmd := send(newModel(nil), k)
			if cmd == nil {
				t.Fatalf("%s で終了コマンドが返らなかった", name)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Errorf("%s で返ったコマンドが終了メッセージを生まない: %T", name, cmd())
			}
		})
	}
}

func TestCursorClampsWhenCardsShrink(t *testing.T) {
	m := loaded(threeNowCards())
	m, _ = send(m, runeKey('j'), runeKey('j'))
	if m.cursor != 2 {
		t.Fatalf("前提が崩れている: cursor = %d", m.cursor)
	}

	m, _ = send(m, fetchedMsg{res: &fetch.Result{Cards: threeNowCards()[:1]}, at: at})

	if m.cursor != 0 {
		t.Errorf("行数が減ったときに丸められていない: cursor = %d, want 0", m.cursor)
	}
}

func TestFetchCmdRunsFetcherOnce(t *testing.T) {
	res := exampleResult(t)
	calls := 0
	fetcher := func(context.Context) (*fetch.Result, error) {
		calls++
		return res, nil
	}

	msg := fetchCmd(fetcher)()

	if calls != 1 {
		t.Errorf("Fetcher の呼び出し回数 = %d, want 1", calls)
	}
	fetched, ok := msg.(fetchedMsg)
	if !ok {
		t.Fatalf("返ったメッセージが fetchedMsg でない: %T", msg)
	}
	m, _ := send(newModel(fetcher), fetched)
	if len(m.cards) != len(res.Cards) {
		t.Errorf("Cards が入っていない: %d 件, want %d 件", len(m.cards), len(res.Cards))
	}
	if m.fetching {
		t.Error("取得完了後も fetching が true のまま")
	}
}

func TestFetchErrorKeepsPreviousCards(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at})
	before := len(m.cards)

	m, _ = send(m, fetchedMsg{err: errors.New("search issues: gh search issues: exit 1: rate limited"), at: at.Add(time.Hour)})

	if len(m.cards) != before {
		t.Errorf("失敗で Cards が変わった: %d 件, want %d 件", len(m.cards), before)
	}
	if !m.at.Equal(at) {
		t.Errorf("失敗で最終更新時刻が変わった: %v", m.at)
	}
	if m.errText == "" || !contains(m.errText, "rate limited") {
		t.Errorf("errText にエラーが入っていない: %q", m.errText)
	}
}

func TestFirstFetchError(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{err: errors.New("boom"), at: at})

	if len(m.cards) != 0 {
		t.Errorf("初回失敗で Cards が入った: %d 件", len(m.cards))
	}
	if !m.at.IsZero() {
		t.Errorf("初回失敗で最終更新時刻が入った: %v", m.at)
	}
	if m.errText != "boom" {
		t.Errorf("errText = %q, want %q", m.errText, "boom")
	}
}

// countingFetcher は呼ばれた回数を数えて res を返す Fetcher を作る。
func countingFetcher(res *fetch.Result, n *int) Fetcher {
	return func(context.Context) (*fetch.Result, error) {
		*n++
		return res, nil
	}
}

// runBatch はコマンドを実行し、複数コマンドの束ならその各コマンドも実行してメッセージを返す。
func runBatch(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("コマンドが返っていない")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var msgs []tea.Msg
	for _, c := range batch {
		msgs = append(msgs, c())
	}
	return msgs
}

// fetchedOf は束のメッセージから取得完了を 1 つ取り出す。
func fetchedOf(t *testing.T, msgs []tea.Msg) fetchedMsg {
	t.Helper()
	for _, msg := range msgs {
		if f, ok := msg.(fetchedMsg); ok {
			return f
		}
	}
	t.Fatalf("束に fetchedMsg が無い: %T", msgs)
	return fetchedMsg{}
}

var rKey = runeKey('R')

// TestRefreshFetchesAgain は R で Fetcher がもう一度呼ばれ、結果が反映されることを検証する。
func TestRefreshFetchesAgain(t *testing.T) {
	res := exampleResult(t)
	n := 0
	m, _ := send(newModel(countingFetcher(res, &n)), fetchedMsg{res: &fetch.Result{}, at: at})
	if n != 0 {
		t.Fatalf("前提が崩れている: Fetcher 呼び出し = %d", n)
	}

	m, cmd := send(m, rKey)
	if cmd == nil {
		t.Fatal("R でコマンドが返っていない")
	}
	if !m.fetching {
		t.Error("R の後に fetching が false")
	}
	if !strings.Contains(plainText(m), "取得中") {
		t.Errorf("取得中が出ていない: %q", footerOf(plainText(m)))
	}

	m, _ = send(m, fetchedOf(t, runBatch(t, cmd)))

	if n != 1 {
		t.Errorf("Fetcher 呼び出し = %d, want 1", n)
	}
	if len(m.cards) != len(res.Cards) {
		t.Errorf("Cards = %d 件, want %d 件", len(m.cards), len(res.Cards))
	}
	if m.fetching {
		t.Error("取得完了後も fetching が true")
	}
}

// TestRefreshIsIgnoredWhileFetching は取得中の R が多重発行しないことを検証する。
func TestRefreshIsIgnoredWhileFetching(t *testing.T) {
	n := 0
	fetcher := countingFetcher(exampleResult(t), &n)

	if _, cmd := send(newModel(fetcher), rKey); cmd != nil {
		t.Errorf("初回取得中の R でコマンドが返った: %T", cmd())
	}

	m, _ := send(newModel(fetcher), fetchedMsg{res: &fetch.Result{}, at: at})
	m, cmd := send(m, rKey)
	if cmd == nil {
		t.Fatal("1 回目の R でコマンドが返っていない")
	}
	if _, second := send(m, rKey); second != nil {
		t.Errorf("取得中の R でコマンドが返った: %T", second())
	}
	if n != 0 {
		t.Errorf("コマンドを実行していないのに Fetcher が呼ばれた: %d 回", n)
	}
}

// TestRefreshClearsErrorAndWriteStatus は R が前回のエラーと書き込みステータスを消し、
// 前回結果は取得の完了まで残すことを検証する。
func TestRefreshClearsErrorAndWriteStatus(t *testing.T) {
	m, _ := exampleTodoModel(t)
	m, _ = send(m, fetchedMsg{err: errors.New("search issues: gh search issues: exit 1: rate limited"), at: at})
	m, _ = send(m, runeKey('2'))
	m, cmd := send(m, tKey)
	m, _ = runCmd(t, m, cmd)
	if !strings.Contains(plainText(m), "付けました") {
		t.Fatalf("前提が崩れている: %q", footerOf(plainText(m)))
	}

	m, _ = send(m, runeKey('1'), rKey)

	text := plainText(m)
	if !strings.Contains(text, "取得中") {
		t.Errorf("取得中が出ていない: %q", footerOf(text))
	}
	for _, ng := range []string{"rate limited", "付けました"} {
		if strings.Contains(text, ng) {
			t.Errorf("R の後に %q が残っている: %q", ng, footerOf(text))
		}
	}
	for _, want := range []string{"PR131", "↻ 12:04"} {
		if !strings.Contains(text, want) {
			t.Errorf("前回結果の %q が消えた", want)
		}
	}
}

// TestRefreshFailureKeepsPreviousResult は再取得の失敗が前回結果を残すことを検証する。
func TestRefreshFailureKeepsPreviousResult(t *testing.T) {
	m, _ := exampleTodoModel(t)
	m, _ = send(m, runeKey('2'))
	m, cmd := send(m, tKey)
	m, _ = runCmd(t, m, cmd)

	m, _ = send(m, runeKey('1'), rKey,
		fetchedMsg{err: errors.New("search issues: gh search issues: exit 1: rate limited"), at: at.Add(time.Hour)})

	text := plainText(m)
	for _, want := range []string{"PR131", "↻ 12:04", "rate limited"} {
		if !strings.Contains(text, want) {
			t.Errorf("%q が無い: %q", want, footerOf(text))
		}
	}
	for _, ng := range []string{"付けました", "取得中"} {
		if strings.Contains(text, ng) {
			t.Errorf("失敗の後に %q が残っている: %q", ng, footerOf(text))
		}
	}
}

// TestRefreshKeepsCursor は再取得で選択行が先頭に戻らないことを検証する。
func TestRefreshKeepsCursor(t *testing.T) {
	cards := threeNowCards()
	m := loaded(cards)
	m, _ = send(m, runeKey('j'), runeKey('j'))
	if m.cursor != 2 {
		t.Fatalf("前提が崩れている: cursor = %d", m.cursor)
	}

	m, _ = send(m, rKey, fetchedMsg{res: &fetch.Result{Cards: cards}, at: at.Add(time.Hour)})

	if m.cursor != 2 {
		t.Errorf("再取得で選択行が動いた: cursor = %d, want 2", m.cursor)
	}
}

// TestRefreshDoesNothingOutsideQueue はキュー画面以外の R が何もしないことを検証する。
func TestRefreshDoesNothingOutsideQueue(t *testing.T) {
	res := exampleResult(t)
	n := 0
	base, _ := send(newModel(countingFetcher(res, &n)),
		tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: res, at: at})

	card, _ := send(base, enterKey)
	pr, _ := send(card, enterKey)
	help, _ := send(base, questionKey)
	confirm, _ := confirmModel(t)

	for name, m := range map[string]Model{"カード詳細": card, "PR 詳細": pr, "ヘルプ": help, "確認": confirm} {
		t.Run(name, func(t *testing.T) {
			got, cmd := send(m, rKey)
			if cmd != nil {
				t.Errorf("R でコマンドが返った: %T", cmd())
			}
			if got.screen != m.screen {
				t.Errorf("R で画面が変わった: screen = %d", got.screen)
			}
			if got.fetching {
				t.Error("R で fetching が true になった")
			}
		})
	}
	if n != 0 {
		t.Errorf("Fetcher が呼ばれた: %d 回", n)
	}
}
