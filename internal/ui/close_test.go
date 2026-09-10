package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

var cKey = runeKey('c')

// rejectedNote は propose / apply の PR を close する前に出す注意（action.CheckClose）。
const rejectedNote = "merge せずに close した PR は却下として扱われ、issue に blocked-by: human が書き戻されます"

// failingClosePR は ClosePR だけが失敗する client。
type failingClosePR struct{ *gh.Fake }

func (f failingClosePR) ClosePR(_ context.Context, _ string, _ int) error {
	return errors.New("gh pr close 131 -R org/app: exit 1: could not close pull request")
}

// closeModel は c のテスト用の Model と close 先の Fake を返す。
// Fake は Result を作ったものとは別に作る（s07 の Fetch が ViewIssue を Calls に残すため）。
func closeModel(t *testing.T) (Model, *gh.Fake) {
	t.Helper()
	fake := gh.NewFake(fixtureDir)
	return closeModelClient(t, fake), fake
}

// closeModelClient は client を差し替えられる closeModel。
func closeModelClient(t *testing.T, client gh.GHClient) Model {
	t.Helper()
	m, _ := send(New(nil, client, (&stubEditor{}).Editor, Options{}),
		tea.WindowSizeMsg{Width: 120, Height: 40},
		fetchedMsg{res: exampleResult(t), at: at})
	return m
}

// closeConfirmed は c を押して確認画面に移った Model を返す。
func closeConfirmed(t *testing.T, m Model) Model {
	t.Helper()
	m, cmd := send(m, cKey)
	if cmd != nil {
		t.Fatalf("c で gh を呼ぶコマンドが返った: %T", cmd())
	}
	if m.screen != screenCloseConfirm {
		t.Fatalf("確認画面に移っていない: screen = %d, フッタ = %q", m.screen, footerOf(plainText(m)))
	}
	return m
}

// TestCloseTargetIsWhatTheScreenShows は c の対象が画面の見せているものに決まることを検証する。
func TestCloseTargetIsWhatTheScreenShows(t *testing.T) {
	tests := []struct {
		name string
		keys []tea.Msg
		want string
	}{
		{name: "キュー画面は主体の PR", want: "close の確認: org/app PR#131"},
		{name: "主体が issue の行でも効く", keys: []tea.Msg{runeKey('2')}, want: "close の確認: org/app #140"},
		{name: "カード詳細は Issue", keys: []tea.Msg{enterKey}, want: "close の確認: org/app #108"},
		{name: "PR 詳細はその PR", keys: []tea.Msg{enterKey, enterKey}, want: "close の確認: org/app PR#131"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, fake := closeModel(t)
			m, _ = send(m, tt.keys...)

			m = closeConfirmed(t, m)
			if text := plainText(m); !strings.Contains(text, tt.want) {
				t.Errorf("確認画面に %q が無い:\n%s", tt.want, text)
			}
			if len(fake.Calls) != 0 {
				t.Errorf("c の押下で gh を呼んでいる: %+v", fake.Calls)
			}
		})
	}
}

// TestCloseDoesNothingWithoutTarget は対象の無い画面で c が何もしないことを検証する。
func TestCloseDoesNothingWithoutTarget(t *testing.T) {
	for name, keys := range map[string][]tea.Msg{
		"異常タブ（0 行）": {runeKey('4')},
		"ヘルプ画面":     {questionKey},
	} {
		t.Run(name, func(t *testing.T) {
			m, fake := closeModel(t)
			m, _ = send(m, keys...)
			before := m.screen

			got, cmd := send(m, cKey)
			if cmd != nil {
				t.Errorf("コマンドが返った: %T", cmd())
			}
			if got.screen != before {
				t.Errorf("画面 = %d, want %d", got.screen, before)
			}
			if len(fake.Calls) != 0 {
				t.Errorf("gh を呼んでいる: %+v", fake.Calls)
			}
		})
	}
}

// TestCloseKeyClearsPreviousStatus は c の押下で直前の書き込みの赤字が消えることを検証する。
func TestCloseKeyClearsPreviousStatus(t *testing.T) {
	m := closeModelClient(t, failingClosePR{gh.NewFake(fixtureDir)})
	m = closeConfirmed(t, m)
	m, cmd := send(m, runeKey('y'))
	m, _ = runCmd(t, m, cmd)
	if !strings.Contains(plainText(m), "の close に失敗:") {
		t.Fatalf("失敗の赤字が出ていない: %q", footerOf(plainText(m)))
	}

	m = closeConfirmed(t, m)

	if text := plainText(m); strings.Contains(text, "の close に失敗:") {
		t.Errorf("確認画面に直前の赤字が残っている: %q", footerOf(text))
	}
}

// TestCloseConfirmKeepsTargetWhileFetching は確認中の取得完了で対象が入れ替わらないことを検証する。
func TestCloseConfirmKeepsTargetWhileFetching(t *testing.T) {
	m, _ := closeModel(t)
	m = closeConfirmed(t, m)

	m, _ = send(m, fetchedMsg{res: &fetch.Result{}, at: at})

	if m.screen != screenCloseConfirm {
		t.Fatalf("取得完了で画面が変わった: screen = %d", m.screen)
	}
	if text := plainText(m); !strings.Contains(text, "close の確認: org/app PR#131") {
		t.Errorf("確認中の対象が変わった:\n%s", text)
	}
}

// TestCloseConfirmScreen は確認画面が対象・種別・ラベル・注意をこの順で出すことを検証する。
func TestCloseConfirmScreen(t *testing.T) {
	t.Run("issue には注意が出ない", func(t *testing.T) {
		m, _ := closeModel(t)
		m, _ = send(m, runeKey('2'))
		m = closeConfirmed(t, m)

		text := plainText(m)
		order(t, text, "close の確認: org/app #140", "種別: issue", "labels: なし")
		if strings.Contains(text, "注意:") {
			t.Errorf("issue に注意が出ている:\n%s", text)
		}
		order(t, footerOf(text), "y close", "Esc 中止", "q 終了")
	})

	t.Run("propose の PR には却下の注意が出る", func(t *testing.T) {
		m, _ := closeModel(t)
		m = closeConfirmed(t, m)

		text := plainText(m)
		order(t, text, "close の確認: org/app PR#131", "種別: PR", "labels: propose question", "注意: "+rejectedNote)
		if !strings.Contains(footerOf(text), "y close") {
			t.Errorf("フッタに y close が無い: %q", footerOf(text))
		}
	})
}

// TestCloseConfirmIgnoresOtherKeys は確認画面の裏の対象が触れないことを検証する。
func TestCloseConfirmIgnoresOtherKeys(t *testing.T) {
	for _, key := range []tea.Msg{aKey, runeKey('t'), mKey, cKey, runeKey('o'), questionKey, runeKey('u'), runeKey('L'), nKey} {
		m, fake := closeModel(t)
		m = closeConfirmed(t, m)

		got, cmd := send(m, key)
		if cmd != nil {
			t.Errorf("%v でコマンドが返った: %T", key, cmd())
		}
		if got.screen != screenCloseConfirm {
			t.Errorf("%v で画面が変わった: screen = %d", key, got.screen)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("%v で gh を呼んでいる: %+v", key, fake.Calls)
		}
	}
}

// TestCloseEscCancels は Esc が close せずに戻り先へ戻ることを検証する。
func TestCloseEscCancels(t *testing.T) {
	t.Run("キュー画面に戻る", func(t *testing.T) {
		m, fake := closeModel(t)
		m = closeConfirmed(t, m)

		m, _ = send(m, escKey)

		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー", m.screen)
		}
		if !strings.Contains(footerOf(plainText(m)), "close を中止しました") {
			t.Errorf("中止がフッタに無い: %q", footerOf(plainText(m)))
		}
		for _, c := range fake.Calls {
			if c.Method == "CloseIssue" || c.Method == "ClosePR" {
				t.Errorf("中止したのに close している: %+v", c)
			}
		}
	})

	t.Run("詳細画面に戻る", func(t *testing.T) {
		m, _ := closeModel(t)
		m, _ = send(m, enterKey)
		m = closeConfirmed(t, m)

		m, _ = send(m, escKey)

		if m.screen != screenCard {
			t.Errorf("画面 = %d, want カード詳細", m.screen)
		}
		if m.detail.card.Issue.Number != 108 {
			t.Errorf("詳細の対象 = #%d, want 108", m.detail.card.Issue.Number)
		}
	})
}

// TestCloseConfirmResizeRebuildsBody は確認中のリサイズが戻り先の本文領域に反映されることを検証する。
func TestCloseConfirmResizeRebuildsBody(t *testing.T) {
	m, _ := send(detailModel(100, 20, []model.Card{longBodyCard()}), enterKey)
	short := plainText(m)
	m = closeConfirmed(t, m)

	m, _ = send(m, tea.WindowSizeMsg{Width: 100, Height: 40}, escKey)

	tall := plainText(m)
	if strings.Contains(short, "行10") {
		t.Fatalf("高さ 20 で 行10 まで出ている（この検証は成り立たない）:\n%s", short)
	}
	if !strings.Contains(tall, "行10") {
		t.Errorf("高さ 40 に作り直されていない:\n%s", tall)
	}
}

// TestCloseWritesToTheKindOfTheTarget は y の close が種別どおりの書き先へ 1 回だけ行くことを検証する。
func TestCloseWritesToTheKindOfTheTarget(t *testing.T) {
	tests := []struct {
		name       string
		keys       []tea.Msg
		wantMethod string
		wantNumber int
		wantStatus string
	}{
		{name: "PR は ClosePR", wantMethod: "ClosePR", wantNumber: 131, wantStatus: "org/app PR#131 を close しました"},
		{
			name: "issue は CloseIssue", keys: []tea.Msg{runeKey('2')},
			wantMethod: "CloseIssue", wantNumber: 140, wantStatus: "org/app #140 を close しました",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, fake := closeModel(t)
			m, _ = send(m, tt.keys...)
			before := m.cards
			m = closeConfirmed(t, m)

			m, cmd := send(m, runeKey('y'))
			m, _ = runCmd(t, m, cmd)

			var closes []gh.Call
			for _, c := range fake.Calls {
				switch c.Method {
				case "CloseIssue", "ClosePR":
					closes = append(closes, c)
				case "AddLabel", "RemoveLabel", "EditIssueLabels", "EditPRLabels", "CommentIssue", "CommentPR":
					t.Errorf("ラベルかコメントを書いている: %+v", c)
				}
			}
			if len(closes) != 1 {
				t.Fatalf("close の呼び出し = %+v, want 1 件", closes)
			}
			if closes[0].Method != tt.wantMethod || closes[0].Repo != "org/app" || closes[0].Number != tt.wantNumber {
				t.Errorf("close の呼び出し = %+v, want %s org/app %d", closes[0], tt.wantMethod, tt.wantNumber)
			}
			if !strings.Contains(footerOf(plainText(m)), tt.wantStatus) {
				t.Errorf("フッタ = %q, want %q を含む", footerOf(plainText(m)), tt.wantStatus)
			}
			if m.screen != screenQueue {
				t.Errorf("画面 = %d, want キュー", m.screen)
			}
			if len(m.cards) != len(before) {
				t.Errorf("Cards の件数が変わった: %d, want %d", len(m.cards), len(before))
			}
		})
	}
}

// TestCloseFailureShowsError は close の失敗がフッタに出て Cards が変わらないことを検証する。
func TestCloseFailureShowsError(t *testing.T) {
	fake := gh.NewFake(fixtureDir)
	m := closeModelClient(t, failingClosePR{fake})
	before := len(m.cards)
	m = closeConfirmed(t, m)

	m, cmd := send(m, runeKey('y'))
	m, _ = runCmd(t, m, cmd)

	footer := footerOf(plainText(m))
	for _, want := range []string{"org/app PR#131 の close に失敗:", "could not close pull request"} {
		if !strings.Contains(footer, want) {
			t.Errorf("フッタ = %q, want %q を含む", footer, want)
		}
	}
	if len(m.cards) != before {
		t.Errorf("Cards の件数が変わった: %d, want %d", len(m.cards), before)
	}
	for _, c := range fake.Calls {
		switch c.Method {
		case "AddLabel", "RemoveLabel", "EditIssueLabels", "EditPRLabels", "CommentIssue", "CommentPR":
			t.Errorf("ラベルかコメントを書いている: %+v", c)
		}
	}
}

// TestClosingBlocksWritesButNotBrowse は close 中の書き込みキーだけが止まることを検証する。
func TestClosingBlocksWritesButNotBrowse(t *testing.T) {
	m, _ := closeModel(t)
	m, _ = send(m, runeKey('2'))
	m = closeConfirmed(t, m)
	m, cmd := send(m, runeKey('y'))
	if cmd == nil {
		t.Fatal("y で close のコマンドが返っていない")
	}
	if !strings.Contains(footerOf(plainText(m)), "org/app #140 を close 中") {
		t.Fatalf("close 中が出ていない: %q", footerOf(plainText(m)))
	}

	for _, key := range []tea.Msg{runeKey('t'), aKey, mKey, cKey} {
		if _, got := send(m, key); got != nil {
			t.Errorf("close 中の %v でコマンドが返った: %T", key, got())
		}
	}
	if _, got := send(m, runeKey('o')); got == nil {
		t.Error("close 中の o でコマンドが返らない")
	}
}
