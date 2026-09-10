package ui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

var mKey = runeKey('m')

// failingMergePR は MergePR だけが失敗する client。
type failingMergePR struct{ *gh.Fake }

func (f failingMergePR) MergePR(_ context.Context, _ string, _ int, _ string) error {
	return errors.New("gh pr merge 131 -R org/app --squash: exit 1: Pull request is not mergeable")
}

// mergeModel は m のテスト用の Model と、取り直し / merge 先の Fake を返す。
// Fake は Result を作ったものとは別に作る（s07 の Fetch が ViewIssue を Calls に残すため）。
func mergeModel(t *testing.T, dir string, opts Options) (Model, *gh.Fake) {
	t.Helper()
	fake := gh.NewFake(dir)
	return mergeModelClient(t, fake, opts), fake
}

// mergeModelClient は client を差し替えられる mergeModel。
func mergeModelClient(t *testing.T, client gh.GHClient, opts Options) Model {
	t.Helper()
	m, _ := send(New(nil, client, (&stubEditor{}).Editor, opts),
		tea.WindowSizeMsg{Width: 120, Height: 40},
		fetchedMsg{res: exampleResult(t), at: at})
	return m
}

// confirmed は m を押して取り直しまで進め、merge の確認画面に移った Model を返す。
func confirmed(t *testing.T, m Model) Model {
	t.Helper()
	m, cmd := send(m, mKey)
	m, _ = runCmd(t, m, cmd)
	if m.screen != screenMergeConfirm {
		t.Fatalf("確認画面に移っていない: screen = %d, フッタ = %q", m.screen, footerOf(plainText(m)))
	}
	return m
}

// TestMergeTargetIsThePROfTheScreen は m の対象が画面の PR に決まることを検証する。
func TestMergeTargetIsThePROfTheScreen(t *testing.T) {
	t.Run("キュー画面は主体の PR", func(t *testing.T) {
		m, _ := mergeModel(t, "testdata/merge", Options{})

		m, cmd := send(m, mKey)
		if cmd == nil {
			t.Fatal("m でコマンドが返っていない")
		}
		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー", m.screen)
		}
		if text := plainText(m); !strings.Contains(text, "org/app PR#131 の状態を取得中") {
			t.Errorf("取得中が出ていない: %q", footerOf(text))
		}
	})

	t.Run("主体が issue の行は何もしない", func(t *testing.T) {
		m, fake := mergeModel(t, "testdata/merge", Options{})
		m, _ = send(m, runeKey('2'))
		if rows := m.rows[m.tab]; len(rows) != 1 || rows[0].isPR {
			t.Fatalf("バックログの行 = %+v, want issue 1 行", rows)
		}

		got, cmd := send(m, mKey)
		if cmd != nil {
			t.Errorf("コマンドが返った: %T", cmd())
		}
		if got.screen != screenQueue || len(fake.Calls) != 0 {
			t.Errorf("画面 = %d / 呼び出し = %+v", got.screen, fake.Calls)
		}
		if text := plainText(got); strings.Contains(text, "の状態を取得中") {
			t.Errorf("取得中が出ている: %q", footerOf(text))
		}
	})

	t.Run("カード詳細と PR 詳細はその PR", func(t *testing.T) {
		for name, keys := range map[string][]tea.Msg{
			"カード詳細": {enterKey},
			"PR 詳細": {enterKey, enterKey},
		} {
			t.Run(name, func(t *testing.T) {
				m, _ := mergeModel(t, "testdata/merge", Options{})
				m, _ = send(m, keys...)

				m = confirmed(t, m)
				if m.merge.repo != "org/app" || m.merge.number != 131 {
					t.Errorf("対象 = %s #%d, want org/app 131", m.merge.repo, m.merge.number)
				}
			})
		}
	})

	t.Run("0 行のタブとヘルプ画面は何もしない", func(t *testing.T) {
		for name, keys := range map[string][]tea.Msg{
			"異常タブ":  {runeKey('4')},
			"ヘルプ画面": {runeKey('?')},
		} {
			t.Run(name, func(t *testing.T) {
				m, fake := mergeModel(t, "testdata/merge", Options{})
				m, _ = send(m, keys...)
				before := m.screen

				got, cmd := send(m, mKey)
				if cmd != nil {
					t.Errorf("コマンドが返った: %T", cmd())
				}
				if got.screen != before || len(fake.Calls) != 0 {
					t.Errorf("画面 = %d, want %d / 呼び出し = %+v", got.screen, before, fake.Calls)
				}
			})
		}
	})
}

// TestMergeFetchBlocksOtherWrites は取り直し中の t / a / m が効かないことを検証する。
func TestMergeFetchBlocksOtherWrites(t *testing.T) {
	m, fake := mergeModel(t, "testdata/merge", Options{})

	m, cmd := send(m, mKey)
	if cmd == nil {
		t.Fatal("m でコマンドが返っていない")
	}
	for _, key := range []tea.KeyPressMsg{tKey, aKey, mKey} {
		got, again := send(m, key)
		if again != nil {
			t.Errorf("取り直し中の %v が無視されていない: %T", key, again())
		}
		if got.screen != screenQueue {
			t.Errorf("取り直し中の %v で画面が変わった: %d", key, got.screen)
		}
	}
	if len(fake.Calls) != 0 {
		t.Errorf("呼び出し = %+v, want 空（コマンドをまだ実行していない）", fake.Calls)
	}
}

// TestMergeConfirmAfterRefetch は取り直した値で確認画面が組み立てられることを検証する。
// 画面の Card は question ラベルと PENDING の checks を持つが、取り直した値には無いので警告が出ない。
func TestMergeConfirmAfterRefetch(t *testing.T) {
	m, _ := mergeModel(t, "testdata/merge", Options{})
	m = confirmed(t, m)

	text := plainText(m)
	if !strings.Contains(text, "merge の確認: org/app PR#131") {
		t.Errorf("見出しが無い:\n%s", text)
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "注意:") {
			t.Errorf("取り直した値に警告が出ている: %q", line)
		}
	}
}

// TestMergeRefetchFailureIsRed は取り直しの失敗が赤で出て確認画面に移らないことを検証する。
func TestMergeRefetchFailureIsRed(t *testing.T) {
	m, fake := mergeModel(t, "testdata/empty", Options{})
	before := m.cards

	m, cmd := send(m, mKey)
	m, _ = runCmd(t, m, cmd)

	if m.screen != screenQueue {
		t.Fatalf("画面 = %d, want キュー", m.screen)
	}
	text := plainText(m)
	for _, sub := range []string{"org/app PR#131 の状態を取得できません:", "pr-131.json"} {
		if !strings.Contains(text, sub) {
			t.Errorf("失敗に %q が無い: %q", sub, footerOf(text))
		}
	}
	if !m.writeStatusErr {
		t.Error("失敗が赤（エラー扱い）で出ていない")
	}
	assertNoMergePR(t, fake.Calls)
	if len(m.cards) != len(before) {
		t.Errorf("Cards が変わっている: %d 件, want %d 件", len(m.cards), len(before))
	}
}

// TestMergeConfirmShowsWarnings は判断材料と警告が順に並び、フッタに merge の道が出ることを検証する。
func TestMergeConfirmShowsWarnings(t *testing.T) {
	m, _ := mergeModel(t, fixtureDir, Options{})
	m = confirmed(t, m)

	text := plainText(m)
	order(t, text,
		"merge の確認: org/app PR#131",
		"方式: squash",
		"labels: propose question",
		"mergeable: UNKNOWN BLOCKED",
		"checks: 緑以外",
		"注意: question ラベルが付いています",
		"注意: checks が緑ではありません",
		"issue #108 の提案",
	)
	footer := footerOf(text)
	for _, want := range []string{"y merge", "Esc 中止", "q 終了"} {
		if !strings.Contains(footer, want) {
			t.Errorf("フッタに %q が無い: %q", want, footer)
		}
	}
}

// TestMergeConfirmRejectsDraft は draft の PR で y が出ず、押しても何も起きないことを検証する。
func TestMergeConfirmRejectsDraft(t *testing.T) {
	m, fake := mergeModel(t, "testdata/merge-draft", Options{})
	m = confirmed(t, m)

	text := plainText(m)
	if !strings.Contains(text, "merge できません: draft の PR です") {
		t.Errorf("拒否の理由が出ていない:\n%s", text)
	}
	footer := footerOf(text)
	for _, want := range []string{"Esc 中止", "q 終了"} {
		if !strings.Contains(footer, want) {
			t.Errorf("フッタに %q が無い: %q", want, footer)
		}
	}
	if strings.Contains(footer, "y merge") {
		t.Errorf("draft のフッタに merge の道が出ている: %q", footer)
	}

	got, cmd := send(m, runeKey('y'))
	if cmd != nil {
		t.Errorf("draft の y でコマンドが返った: %T", cmd())
	}
	if got.screen != screenMergeConfirm {
		t.Errorf("画面 = %d, want 確認画面", got.screen)
	}
	assertNoMergePR(t, fake.Calls)
}

// TestMergeConfirmEscAborts は Esc が merge せずに戻り先へ戻ることを検証する。
func TestMergeConfirmEscAborts(t *testing.T) {
	t.Run("キュー画面へ戻る", func(t *testing.T) {
		m, fake := mergeModel(t, fixtureDir, Options{})
		m = confirmed(t, m)

		m, _ = send(m, escKey)
		if m.screen != screenQueue {
			t.Fatalf("画面 = %d, want キュー", m.screen)
		}
		if text := plainText(m); !strings.Contains(text, "merge を中止しました") {
			t.Errorf("中止が出ていない: %q", footerOf(text))
		}
		assertNoMergePR(t, fake.Calls)
	})

	t.Run("カード詳細へ戻る", func(t *testing.T) {
		m, _ := mergeModel(t, fixtureDir, Options{})
		m, _ = send(m, enterKey)
		m = confirmed(t, m)

		m, _ = send(m, escKey)
		if m.screen != screenCard {
			t.Fatalf("画面 = %d, want カード詳細", m.screen)
		}
		if m.detail.card.Issue == nil || m.detail.card.Issue.Number != 108 {
			t.Errorf("詳細の対象 = %+v, want issue 108", m.detail.card.Issue)
		}
	})
}

// TestMergeConfirmKeepsTargetOnFetched は確認画面での取得完了が対象を変えないことを検証する。
func TestMergeConfirmKeepsTargetOnFetched(t *testing.T) {
	m, _ := mergeModel(t, fixtureDir, Options{})
	m = confirmed(t, m)

	m, _ = send(m, fetchedMsg{res: &fetch.Result{Cards: []model.Card{cardOf(t, exampleResult(t), 108)}}, at: at})

	if m.screen != screenMergeConfirm {
		t.Fatalf("取得完了で画面が変わった: screen = %d", m.screen)
	}
	text := plainText(m)
	order(t, text, "merge の確認: org/app PR#131", "注意: question ラベルが付いています")
}

// TestMergeSucceeds は y で MergePR が 1 回呼ばれ、成功がフッタに出ることを検証する。
func TestMergeSucceeds(t *testing.T) {
	m, fake := mergeModel(t, "testdata/merge", Options{})
	before := len(m.rows[model.TabNow])
	m = confirmed(t, m)

	m, cmd := send(m, runeKey('y'))
	if m.screen != screenQueue {
		t.Fatalf("y の直後の画面 = %d, want キュー", m.screen)
	}
	if text := plainText(m); !strings.Contains(text, "org/app PR#131 を merge 中") {
		t.Errorf("merge 中が出ていない: %q", footerOf(text))
	}
	for _, key := range []tea.KeyPressMsg{tKey, aKey, mKey} {
		if _, again := send(m, key); again != nil {
			t.Errorf("merge 中の %v が無視されていない: %T", key, again())
		}
	}

	m, _ = runCmd(t, m, cmd)

	want := gh.Call{Method: "MergePR", Repo: "org/app", Number: 131, MergeMethod: "squash"}
	if len(fake.Calls) != 1 || !reflect.DeepEqual(fake.Calls[0], want) {
		t.Fatalf("呼び出し = %+v, want 1 件の %+v", fake.Calls, want)
	}
	if text := plainText(m); !strings.Contains(text, "org/app PR#131 を merge しました") {
		t.Errorf("成功が出ていない: %q", footerOf(text))
	}
	rows := m.rows[model.TabNow]
	if len(rows) != before {
		t.Errorf("今やるタブの行数 = %d, want %d（Cards は変えない）", len(rows), before)
	}
	assertNoLabelOrComment(t, fake.Calls)
}

// TestMergeFailureIsRed は merge の失敗が赤で出て Cards を変えないことを検証する。
func TestMergeFailureIsRed(t *testing.T) {
	fake := gh.NewFake("testdata/merge")
	m := mergeModelClient(t, failingMergePR{fake}, Options{})
	before := len(m.cards)
	m = confirmed(t, m)

	m, cmd := send(m, runeKey('y'))
	m, _ = runCmd(t, m, cmd)

	text := plainText(m)
	for _, sub := range []string{"org/app PR#131 の merge に失敗:", "Pull request is not mergeable"} {
		if !strings.Contains(text, sub) {
			t.Errorf("失敗に %q が無い: %q", sub, footerOf(text))
		}
	}
	if !m.writeStatusErr {
		t.Error("失敗が赤（エラー扱い）で出ていない")
	}
	if len(m.cards) != before {
		t.Errorf("Cards が変わっている: %d 件, want %d 件", len(m.cards), before)
	}
	assertNoLabelOrComment(t, fake.Calls)
}

// TestMergeMethodComesFromOptions は merge 方式が対応表から引かれることを検証する。
func TestMergeMethodComesFromOptions(t *testing.T) {
	for name, tc := range map[string]struct {
		methods map[string]string
		want    string
	}{
		"リポジトリ別の方式": {map[string]string{"org/app": "rebase"}, "rebase"},
		"表に無いリポジトリ": {map[string]string{}, "squash"},
	} {
		t.Run(name, func(t *testing.T) {
			m, fake := mergeModel(t, "testdata/merge", Options{MergeMethods: tc.methods})
			m = confirmed(t, m)

			if text := plainText(m); !strings.Contains(text, "方式: "+tc.want) {
				t.Errorf("確認画面に方式 %q が無い:\n%s", tc.want, text)
			}

			m, cmd := send(m, runeKey('y'))
			_, _ = runCmd(t, m, cmd)

			if len(fake.Calls) != 1 || fake.Calls[0].MergeMethod != tc.want {
				t.Fatalf("呼び出し = %+v, want MergeMethod %q", fake.Calls, tc.want)
			}
		})
	}
}

// assertNoMergePR は MergePR が呼ばれていないことを確かめる。
func assertNoMergePR(t *testing.T, calls []gh.Call) {
	t.Helper()
	for _, c := range calls {
		if c.Method == "MergePR" {
			t.Errorf("MergePR が呼ばれている: %+v", c)
		}
	}
}

// assertNoLabelOrComment は merge がラベルもコメントも書かないことを確かめる（不変条件 2）。
func assertNoLabelOrComment(t *testing.T, calls []gh.Call) {
	t.Helper()
	for _, c := range calls {
		switch c.Method {
		case "AddLabel", "RemoveLabel", "CommentIssue", "CommentPR":
			t.Errorf("merge が %s を呼んでいる: %+v", c.Method, c)
		}
	}
}

// TestMergeFetchedOnOtherScreenAborts は取り直しの結果が別の画面に届いたとき、
// 確認画面に移らずに中止することを検証する。戻り先は `m` を押した画面に固定されているので、
// 画面が変わったまま確認画面を開くと、抜けた先が今の詳細と食い違う。
func TestMergeFetchedOnOtherScreenAborts(t *testing.T) {
	t.Run("ヘルプ画面を確認画面で踏み潰さない", func(t *testing.T) {
		m, _ := mergeModel(t, fixtureDir, Options{})

		m, cmd := send(m, mKey)
		m, _ = send(m, runeKey('?'))
		m, _ = runCmd(t, m, cmd)

		if m.screen != screenHelp {
			t.Fatalf("画面 = %d, want ヘルプ", m.screen)
		}
		text := plainText(m)
		if !strings.Contains(text, "org/app PR#131 の merge を中止しました（画面が変わりました）") {
			t.Errorf("中止が出ていない: %q", footerOf(text))
		}
		if m.writeStatusErr {
			t.Error("画面が変わっただけなのに赤で出ている")
		}
	})

	t.Run("PR 詳細を離れてから届いても確認画面に移らない", func(t *testing.T) {
		m, _ := mergeModel(t, fixtureDir, Options{})
		m, _ = send(m, enterKey, enterKey)
		if m.screen != screenPR {
			t.Fatalf("画面 = %d, want PR 詳細", m.screen)
		}

		m, cmd := send(m, mKey)
		// 取り直し中も Esc / 数字 / Enter は動く。PR を持たない issue 140 のカード詳細へ移る。
		m, _ = send(m, escKey, escKey, runeKey('2'), enterKey)
		if m.screen != screenCard || len(m.detail.card.PRs) != 0 {
			t.Fatalf("画面 = %d / PR = %d 件, want PR 0 件のカード詳細", m.screen, len(m.detail.card.PRs))
		}

		m, _ = runCmd(t, m, cmd)

		if m.screen != screenCard {
			t.Fatalf("画面 = %d, want カード詳細のまま", m.screen)
		}
		text := plainText(m)
		if !strings.Contains(text, "org/app PR#131 の merge を中止しました（画面が変わりました）") {
			t.Errorf("中止が出ていない: %q", footerOf(text))
		}
	})
}
