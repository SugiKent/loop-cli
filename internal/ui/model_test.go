package ui

import (
	"context"
	"errors"
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
		"t":   runeKey('t'), "o": runeKey('o'), "R": runeKey('R'),
		"?": runeKey('?'), "v": runeKey('v'), "m": runeKey('m'), "n": runeKey('n'),
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
