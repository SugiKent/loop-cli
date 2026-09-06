package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SugiKent/loop-cli/internal/gh"
)

var oKey = runeKey('o')

// browseModel は o のテスト用の Model。client は Result を作った Fake とは別に渡す。
func browseModel(t *testing.T) (Model, *gh.Fake) {
	t.Helper()
	return exampleTodoModel(t)
}

// wantBrowse は Calls が Browse 1 件であることを確かめる。
func wantBrowse(t *testing.T, fake *gh.Fake, number int) {
	t.Helper()
	want := gh.Call{Method: "Browse", Repo: "org/app", Number: number}
	if len(fake.Calls) != 1 || fake.Calls[0] != want {
		t.Fatalf("呼び出し = %+v, want [%+v]", fake.Calls, want)
	}
}

// TestBrowseOpensSubjectOfSelectedRow はキュー画面の o が選択行の主体を開くことを検証する。
func TestBrowseOpensSubjectOfSelectedRow(t *testing.T) {
	t.Run("主体が PR の行", func(t *testing.T) {
		m, fake := browseModel(t)

		m, cmd := send(m, oKey)
		if len(fake.Calls) != 0 {
			t.Errorf("コマンドの実行前に gh を呼んでいる: %+v", fake.Calls)
		}

		m, _ = runCmd(t, m, cmd)

		wantBrowse(t, fake, 131)
		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー", m.screen)
		}
	})

	t.Run("主体が issue の行", func(t *testing.T) {
		m, fake := browseModel(t)
		m, _ = send(m, runeKey('2'))

		m, cmd := send(m, oKey)
		m, _ = runCmd(t, m, cmd)

		wantBrowse(t, fake, 140)
		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー", m.screen)
		}
	})
}

// TestBrowseTargetInDetail はカード詳細が Issue を、PR 詳細がその PR を開くことを検証する。
func TestBrowseTargetInDetail(t *testing.T) {
	t.Run("カード詳細は Issue", func(t *testing.T) {
		m, fake := browseModel(t)
		m, _ = send(m, enterKey)

		m, cmd := send(m, oKey)
		m, _ = runCmd(t, m, cmd)

		wantBrowse(t, fake, 108)
		if m.screen != screenCard {
			t.Errorf("画面 = %d, want カード詳細", m.screen)
		}
	})

	t.Run("PR 詳細はその PR", func(t *testing.T) {
		m, fake := browseModel(t)
		m, _ = send(m, enterKey, enterKey)

		m, cmd := send(m, oKey)
		m, _ = runCmd(t, m, cmd)

		wantBrowse(t, fake, 131)
		if m.screen != screenPR {
			t.Errorf("画面 = %d, want PR 詳細", m.screen)
		}
	})
}

// TestBrowseDoesNothingWithoutTarget は対象の無い画面で o が何もしないことを検証する。
func TestBrowseDoesNothingWithoutTarget(t *testing.T) {
	abnormal, abnormalFake := browseModel(t)
	abnormal, _ = send(abnormal, runeKey('4'))
	help, helpFake := browseModel(t)
	help, _ = send(help, questionKey)
	confirm, confirmFake := confirmModel(t)

	cases := map[string]struct {
		m    Model
		fake *gh.Fake
	}{
		"異常タブ（0 行）": {abnormal, abnormalFake},
		"ヘルプ画面":     {help, helpFake},
		"確認画面":      {confirm, confirmFake},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, cmd := send(tc.m, oKey)
			if cmd != nil {
				t.Errorf("o でコマンドが返った: %T", cmd())
			}
			if len(tc.fake.Calls) != 0 {
				t.Errorf("gh を呼んでいる: %+v", tc.fake.Calls)
			}
			if got.screen != tc.m.screen {
				t.Errorf("画面が変わった: screen = %d", got.screen)
			}
		})
	}
}

// TestBrowseWorksWhileWriting は o が書き込み中フラグで止まらないことを検証する（読み取りだけのため）。
func TestBrowseWorksWhileWriting(t *testing.T) {
	m, fake := browseModel(t)
	m, _ = send(m, runeKey('2'))

	m, toggle := send(m, tKey)
	if toggle == nil {
		t.Fatal("t でコマンドが返っていない")
	}
	if !m.writing {
		t.Fatal("前提が崩れている: 書き込み中になっていない")
	}

	m, cmd := send(m, oKey)
	m, _ = runCmd(t, m, cmd)

	wantBrowse(t, fake, 140)
	if !strings.Contains(plainText(m), "切り替え中") {
		t.Errorf("o で書き込み中の表示が消えた: %q", footerOf(plainText(m)))
	}
}

// failingBrowse は Browse だけが失敗する client。
type failingBrowse struct{ *gh.Fake }

func (f failingBrowse) Browse(_ context.Context, _ string, _ int) error {
	return errors.New("gh browse 131 -R org/app: exit 1: no browser")
}

// TestBrowseFailureShowsStatus は開けなかったときにフッタへ赤で出ることを検証する。
func TestBrowseFailureShowsStatus(t *testing.T) {
	res := exampleResult(t)
	m := todoModel(failingBrowse{gh.NewFake(fixtureDir)}, res.Cards, &stubEditor{})
	before := len(m.cards)

	m, cmd := send(m, oKey)
	m, _ = runCmd(t, m, cmd)

	text := plainText(m)
	for _, want := range []string{"org/app PR#131 をブラウザで開けません:", "no browser"} {
		if !strings.Contains(text, want) {
			t.Errorf("フッタに %q が無い: %q", want, footerOf(text))
		}
	}
	if len(m.cards) != before {
		t.Errorf("失敗で Cards が変わった: %d 件, want %d 件", len(m.cards), before)
	}
	if m.screen != screenQueue {
		t.Errorf("画面 = %d, want キュー", m.screen)
	}

	m, _ = send(m, rKey)

	text = plainText(m)
	if strings.Contains(text, "開けません") {
		t.Errorf("R で失敗のステータスが消えていない: %q", footerOf(text))
	}
	if !strings.Contains(text, "取得中") {
		t.Errorf("R で取得中が出ていない: %q", footerOf(text))
	}
}

// TestBrowseSuccessShowsNothing は成功時にフッタへ何も出さないことを検証する。
func TestBrowseSuccessShowsNothing(t *testing.T) {
	m, _ := browseModel(t)

	m, cmd := send(m, oKey)
	m, _ = runCmd(t, m, cmd)

	if footer := footerOf(plainText(m)); strings.Contains(footer, "開けません") {
		t.Errorf("成功なのにフッタにエラーがある: %q", footer)
	}
}
