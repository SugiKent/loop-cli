package ui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/action"
	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

var (
	lKey      = runeKey('L')
	lowerLKey = runeKey('l')
	spaceKey  = codeKey(tea.KeySpace)
)

// ラベルの一覧だけを差し替えた fixture ディレクトリ（件数で描画と判定が変わる経路のため）。
const (
	labelsDir      = "../action/testdata/labels"
	labelsEmptyDir = "../action/testdata/labels-empty"
	labelsThreeDir = "../action/testdata/labels-three"
	labelsManyDir  = "../action/testdata/labels-many"
)

// errListLabelsFake は ListLabels だけが失敗する GHClient。
type errListLabelsFake struct {
	*gh.Fake
	err error
}

func (f *errListLabelsFake) ListLabels(context.Context, string) ([]gh.RepoLabel, error) {
	return nil, f.err
}

// errEditLabelsFake は EditIssueLabels だけが失敗する GHClient。
type errEditLabelsFake struct {
	*gh.Fake
	err error
}

func (f *errEditLabelsFake) EditIssueLabels(_ context.Context, _ string, _ int, _, _ []string) error {
	return f.err
}

// labelModel は Card 群を取得完了として渡した Model と、渡した client を返す。
func labelModel(cards []model.Card, client gh.GHClient, width, height int) Model {
	ed := &stubEditor{msg: editedMsg{err: errors.New("使わない")}}
	m, _ := send(New(nil, client, ed.Editor, Options{}),
		tea.WindowSizeMsg{Width: width, Height: height},
		fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})
	return m
}

// exampleLabelModel は example の Result を渡した Model と、その fixture を読む Fake を返す。
func exampleLabelModel(t *testing.T, width, height int) (Model, *gh.Fake) {
	t.Helper()
	fake := gh.NewFake(fixtureDir)
	ed := &stubEditor{msg: editedMsg{err: errors.New("使わない")}}
	m, _ := send(New(nil, fake, ed.Editor, Options{}),
		tea.WindowSizeMsg{Width: width, Height: height},
		fetchedMsg{res: exampleResult(t), at: at})
	return m, fake
}

// labelIssueCard は Labels を指定した issue 1 件だけの Card（今やるタブ）。
func labelIssueCard(number int, labels []string) model.Card {
	result := model.Result{Situation: model.SituationA, Priority: 1, Tab: model.TabNow, Summary: "回答する"}
	issue := &model.Issue{
		Repo: "org/app", Number: number, Title: "手書き", UpdatedAt: at,
		Labels: labels, Result: result,
	}
	return model.Card{Issue: issue, Result: result}
}

// openLabels は L を押し、返った取得のコマンドを実行してラベル一覧を開いた Model を返す。
func openLabels(t *testing.T, m Model) Model {
	t.Helper()
	m, cmd := send(m, lKey)
	if cmd == nil {
		t.Fatal("L でコマンドが返らない（ラベルを取りに行くはず）")
	}
	m, _ = send(m, cmd())
	if m.screen != screenLabels {
		t.Fatalf("ラベル一覧に移っていない: screen = %d, ステータス = %q", m.screen, m.writeStatus)
	}
	return m
}

// labelPickerOn は issue 1 件の Card から一覧を開いた Model と Fake を返す。
func labelPickerOn(t *testing.T, dir string, number int, labels []string, width, height int) (Model, *gh.Fake) {
	t.Helper()
	fake := gh.NewFake(dir)
	m := labelModel([]model.Card{labelIssueCard(number, labels)}, fake, width, height)
	return openLabels(t, m), fake
}

// selectLabel は名前の行までカーソルを下に動かす（添字ではなく名前でテストの意図を書くため）。
func selectLabel(t *testing.T, m Model, name string) Model {
	t.Helper()
	for i, l := range m.labelPicker.labels {
		if l.Name != name {
			continue
		}
		for range i - m.labelPicker.cursor {
			m, _ = send(m, runeKey('j'))
		}
		return m
	}
	t.Fatalf("%q の行が無い: %+v", name, m.labelPicker.labels)
	return m
}

// callsOf は Method が一致する呼び出しだけを返す。
func callsOf(fake *gh.Fake, method string) []gh.Call {
	var out []gh.Call
	for _, c := range fake.Calls {
		if c.Method == method {
			out = append(out, c)
		}
	}
	return out
}

// TestLabelKeyTargetPerScreen は L の対象が画面の見せているものに決まることを検証する。
func TestLabelKeyTargetPerScreen(t *testing.T) {
	t.Run("キュー画面はラベルを取りに行く", func(t *testing.T) {
		m, fake := exampleLabelModel(t, 100, 24)

		m, cmd := send(m, lKey)

		if cmd == nil {
			t.Fatal("コマンドが返らない")
		}
		if m.screen != screenQueue {
			t.Errorf("取得中に画面が変わった: screen = %d", m.screen)
		}
		if footer := footerOf(plainText(m)); !strings.Contains(footer, "org/app のラベル一覧を取得中") {
			t.Errorf("フッタ = %q", footer)
		}

		m, _ = send(m, cmd())

		if m.screen != screenLabels {
			t.Fatalf("取得後の画面 = %d, want ラベル一覧", m.screen)
		}
		if m.labelPicker.cursor != 0 {
			t.Errorf("選択位置 = %d, want 0", m.labelPicker.cursor)
		}
		want := []gh.Call{{Method: "ListLabels", Repo: "org/app"}}
		if got := callsOf(fake, "ListLabels"); !reflect.DeepEqual(got, want) {
			t.Errorf("ListLabels = %+v, want %+v", got, want)
		}
	})

	t.Run("2 回目は gh を呼ばない", func(t *testing.T) {
		m, fake := exampleLabelModel(t, 100, 24)
		m = openLabels(t, m)
		m, _ = send(m, escKey)

		m, cmd := send(m, lKey)

		if cmd != nil {
			t.Error("キャッシュにあるのにコマンドが返っている")
		}
		if m.screen != screenLabels {
			t.Fatalf("画面 = %d, want ラベル一覧", m.screen)
		}
		if got := callsOf(fake, "ListLabels"); len(got) != 1 {
			t.Errorf("ListLabels = %+v, want 1 件", got)
		}
	})

	t.Run("PR 詳細は選択中の PR", func(t *testing.T) {
		m, _ := exampleLabelModel(t, 100, 24)
		m, _ = send(m, enterKey, enterKey)
		if m.screen != screenPR {
			t.Fatalf("PR 詳細に移っていない: screen = %d", m.screen)
		}

		m = openLabels(t, m)

		want := action.Target{Repo: "org/app", Number: 131, IsPR: true}
		if m.labelPicker.target != want {
			t.Errorf("書き先 = %+v, want %+v", m.labelPicker.target, want)
		}
		if origin := labelSet([]string{"propose", "question"}); !reflect.DeepEqual(m.labelPicker.origin, origin) {
			t.Errorf("元の集合 = %v, want %v", m.labelPicker.origin, origin)
		}
	})

	t.Run("カード詳細は Issue", func(t *testing.T) {
		m, _ := exampleLabelModel(t, 100, 24)
		m, _ = send(m, runeKey('2'), enterKey)
		if m.screen != screenCard {
			t.Fatalf("カード詳細に移っていない: screen = %d", m.screen)
		}

		m = openLabels(t, m)

		want := action.Target{Repo: "org/app", Number: 140}
		if m.labelPicker.target != want {
			t.Errorf("書き先 = %+v, want %+v", m.labelPicker.target, want)
		}
		if len(m.labelPicker.origin) != 0 {
			t.Errorf("元の集合 = %v, want 空", m.labelPicker.origin)
		}
	})
}

// TestLabelKeyWithoutLabelsShowsStatus は 0 件のときに画面を変えずステータスを出すことを検証する。
// キャッシュ経路でも同じ判定を通るので、2 回目も一覧を開かない。
func TestLabelKeyWithoutLabelsShowsStatus(t *testing.T) {
	fake := gh.NewFake(labelsEmptyDir)
	m := labelModel([]model.Card{labelIssueCard(150, []string{"wip"})}, fake, 80, 24)

	m, cmd := send(m, lKey)
	if cmd == nil {
		t.Fatal("コマンドが返らない")
	}
	m, _ = send(m, cmd())

	if m.screen != screenQueue {
		t.Fatalf("画面 = %d, want キュー", m.screen)
	}
	if footer := footerOf(plainText(m)); !strings.Contains(footer, "ラベルがありません") {
		t.Errorf("フッタ = %q", footer)
	}

	m, cmd = send(m, lKey)

	if cmd != nil {
		t.Error("2 回目でコマンドが返っている（キャッシュを使うはず）")
	}
	if m.screen != screenQueue {
		t.Errorf("2 回目の画面 = %d, want キュー", m.screen)
	}
	if footer := footerOf(plainText(m)); !strings.Contains(footer, "ラベルがありません") {
		t.Errorf("2 回目のフッタ = %q", footer)
	}
	if got := callsOf(fake, "ListLabels"); len(got) != 1 {
		t.Errorf("ListLabels = %+v, want 1 件", got)
	}
}

// TestLabelFetchFailureShowsError は取得の失敗を赤でフッタに出すことを検証する。
func TestLabelFetchFailureShowsError(t *testing.T) {
	client := &errListLabelsFake{
		Fake: gh.NewFake(labelsDir),
		err:  errors.New("gh label list -R org/app: exit 1: HTTP 403"),
	}
	m := labelModel([]model.Card{labelIssueCard(150, []string{"wip"})}, client, 120, 24)

	m, cmd := send(m, lKey)
	m, _ = send(m, cmd())

	if m.screen != screenQueue {
		t.Fatalf("画面 = %d, want キュー", m.screen)
	}
	footer := footerOf(plainText(m))
	for _, want := range []string{"org/app のラベル一覧を取得できません:", "HTTP 403"} {
		if !strings.Contains(footer, want) {
			t.Errorf("フッタに %q が無い: %q", want, footer)
		}
	}
	if !m.writeStatusErr {
		t.Error("エラーとして出ていない（赤にならない）")
	}
}

// TestLabelFetchAbortsWhenScreenChanged は取得中に画面が変わったら一覧を開かないことを検証する。
// 取得中も ? / u / Esc は動くので、開いた画面をあとから届いた一覧が踏み潰さないようにする。
func TestLabelFetchAbortsWhenScreenChanged(t *testing.T) {
	m, _ := exampleLabelModel(t, 120, 24)

	m, cmd := send(m, lKey)
	m, _ = send(m, questionKey)
	if m.screen != screenHelp {
		t.Fatalf("ヘルプ画面に移っていない: screen = %d", m.screen)
	}

	m, _ = send(m, cmd())

	if m.screen != screenHelp {
		t.Fatalf("画面 = %d, want ヘルプ", m.screen)
	}
	footer := footerOf(plainText(m))
	if !strings.Contains(footer, "org/app のラベル一覧の表示を中止しました（画面が変わりました）") {
		t.Errorf("フッタ = %q", footer)
	}
}

// TestLabelKeyIgnoredWhileWriting は書き込み中の L が何もしないことを検証する。
func TestLabelKeyIgnoredWhileWriting(t *testing.T) {
	m, _ := exampleLabelModel(t, 100, 24)
	m, cmd := send(m, runeKey('t'))
	if cmd == nil {
		t.Fatal("t でコマンドが返らない")
	}

	m, cmd = send(m, lKey)

	if cmd != nil {
		t.Error("書き込み中の L でコマンドが返っている")
	}
	if m.screen != screenQueue {
		t.Errorf("画面 = %d, want キュー", m.screen)
	}
}

// TestLabelKeyDoesNothingOnOtherScreens は対象の無い画面と、先に分岐する画面で L が
// 何もしないことを検証する。
func TestLabelKeyDoesNothingOnOtherScreens(t *testing.T) {
	t.Run("0 行のタブ", func(t *testing.T) {
		m, fake := exampleLabelModel(t, 100, 24)
		m, _ = send(m, runeKey('4'))

		m, cmd := send(m, lKey)

		if cmd != nil {
			t.Error("コマンドが返っている")
		}
		if m.screen != screenQueue {
			t.Errorf("画面 = %d, want キュー", m.screen)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("Calls = %+v, want 空", fake.Calls)
		}
	})

	t.Run("ヘルプ画面", func(t *testing.T) {
		m, fake := exampleLabelModel(t, 100, 24)
		m, _ = send(m, questionKey)

		m, cmd := send(m, lKey)

		if cmd != nil {
			t.Error("コマンドが返っている")
		}
		if m.screen != screenHelp {
			t.Errorf("画面 = %d, want ヘルプ", m.screen)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("Calls = %+v, want 空", fake.Calls)
		}
	})

	t.Run("URL 一覧画面", func(t *testing.T) {
		m, fake := urlModel([]model.Card{urlPRCard(designLink, nil, nil)}, 100, 24)
		m, _ = send(m, uKey)
		if m.screen != screenURL {
			t.Fatalf("URL 一覧に移っていない: screen = %d", m.screen)
		}

		m, cmd := send(m, lKey)

		if cmd != nil {
			t.Error("コマンドが返っている")
		}
		if m.screen != screenURL {
			t.Errorf("画面 = %d, want URL 一覧", m.screen)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("Calls = %+v, want 空", fake.Calls)
		}
	})

	t.Run("回答の確認画面", func(t *testing.T) {
		m, fake := confirmModel(t)

		m, cmd := send(m, lKey)

		if cmd != nil {
			t.Error("コマンドが返っている")
		}
		if m.screen != screenConfirm {
			t.Errorf("画面 = %d, want 確認", m.screen)
		}
		if len(fake.Calls) != 0 {
			t.Errorf("Calls = %+v, want 空", fake.Calls)
		}
	})
}

// TestLowerLDoesNothing は小文字の l が何もしないことを検証する
// （s17 のカンバンの列移動の予約のままである）。
func TestLowerLDoesNothing(t *testing.T) {
	m, fake := exampleLabelModel(t, 100, 24)

	queue, cmd := send(m, lowerLKey)

	if cmd != nil {
		t.Error("キュー画面の l でコマンドが返っている")
	}
	if queue.screen != screenQueue {
		t.Errorf("キュー画面の l で画面 = %d", queue.screen)
	}

	detail, _ := send(m, enterKey)
	detail, cmd = send(detail, lowerLKey)

	if cmd != nil {
		t.Error("カード詳細の l でコマンドが返っている")
	}
	if detail.screen != screenCard {
		t.Errorf("カード詳細の l で画面 = %d", detail.screen)
	}
	if len(fake.Calls) != 0 {
		t.Errorf("Calls = %+v, want 空", fake.Calls)
	}
}

// TestLabelListCursorMoves は j / k で選択が動き、端で止まることを検証する。
func TestLabelListCursorMoves(t *testing.T) {
	m, _ := labelPickerOn(t, labelsThreeDir, 150, []string{"wip"}, 80, 24)

	want := []int{1, 2, 2, 1}
	for i, k := range []tea.Msg{runeKey('j'), runeKey('j'), runeKey('j'), runeKey('k')} {
		m, _ = send(m, k)
		if m.labelPicker.cursor != want[i] {
			t.Fatalf("%d 回目の入力後の選択位置 = %d, want %d", i+1, m.labelPicker.cursor, want[i])
		}
	}
}

// TestSpaceOnlyTogglesMark は Space が印を切り替えるだけで gh を呼ばないことを検証する。
func TestSpaceOnlyTogglesMark(t *testing.T) {
	m, fake := labelPickerOn(t, labelsDir, 150, []string{"wip"}, 80, 24)

	m, cmd := send(m, spaceKey)

	if cmd != nil {
		t.Error("Space でコマンドが返っている")
	}
	if m.screen != screenLabels {
		t.Errorf("画面 = %d, want ラベル一覧", m.screen)
	}
	if !m.labelPicker.want["docs"] {
		t.Errorf("変更予定 = %v, want docs を含む", m.labelPicker.want)
	}
	if got := callsOf(fake, "ListLabels"); len(fake.Calls) != len(got) {
		t.Errorf("Calls = %+v, want ListLabels だけ", fake.Calls)
	}
}

// TestEscDiscardsPendingLabels は Esc が変更予定を捨てることを検証する。
func TestEscDiscardsPendingLabels(t *testing.T) {
	m, fake := labelPickerOn(t, labelsDir, 150, []string{"wip"}, 80, 24)
	m, _ = send(m, spaceKey)

	m, _ = send(m, escKey)

	if m.screen != screenQueue {
		t.Fatalf("Esc の後の画面 = %d, want キュー", m.screen)
	}
	for _, c := range fake.Calls {
		if strings.HasPrefix(c.Method, "Edit") {
			t.Errorf("Esc で書き込んでいる: %+v", fake.Calls)
		}
	}

	m, _ = send(m, lKey)

	if m.labelPicker.want["docs"] {
		t.Errorf("開き直した変更予定 = %v, want 元の集合", m.labelPicker.want)
	}
	if !m.labelPicker.want["wip"] {
		t.Errorf("開き直した変更予定 = %v, want wip を含む", m.labelPicker.want)
	}
}

// TestLabelListIgnoresOtherKeys は一覧画面で表に無いキーが何もしないことを検証する。
// L も含めるので、一覧を出したまま L を押しても戻り先は上書きされない。
func TestLabelListIgnoresOtherKeys(t *testing.T) {
	fake := gh.NewFake(labelsDir)
	ed := &stubEditor{msg: editedMsg{err: errors.New("使わない")}}
	base, _ := send(New(nil, fake, ed.Editor, Options{}),
		tea.WindowSizeMsg{Width: 100, Height: 24},
		fetchedMsg{res: &fetch.Result{Cards: []model.Card{labelIssueCard(150, []string{"wip"})}}, at: at})
	m := openLabels(t, base)
	m = selectLabel(t, m, "wip")
	m, _ = send(m, spaceKey)
	before := m.labelPicker

	keys := []tea.Msg{
		lKey, lowerLKey, runeKey('a'), runeKey('t'), runeKey('m'), runeKey('o'),
		questionKey, uKey, runeKey('R'), runeKey('x'), runeKey('g'), runeKey('2'),
	}
	for _, k := range keys {
		next, cmd := send(m, k)
		if cmd != nil {
			t.Errorf("%v でコマンドが返っている", k)
		}
		if next.screen != screenLabels {
			t.Errorf("%v で画面 = %d, want ラベル一覧", k, next.screen)
		}
		if next.labelPicker.cursor != before.cursor || next.labelPicker.from != before.from {
			t.Errorf("%v で選択位置か戻り先が動いた: %+v", k, next.labelPicker)
		}
		if !reflect.DeepEqual(next.labelPicker.want, before.want) {
			t.Errorf("%v で変更予定が動いた: %v", k, next.labelPicker.want)
		}
	}
	if got := callsOf(fake, "ListLabels"); len(fake.Calls) != len(got) {
		t.Errorf("Calls = %+v, want ListLabels だけ", fake.Calls)
	}
	if ed.calls != 0 {
		t.Errorf("エディタが %d 回開かれた", ed.calls)
	}
}

// TestLabelListQuits は一覧画面でも q が終了することを検証する。
func TestLabelListQuits(t *testing.T) {
	m, _ := labelPickerOn(t, labelsDir, 150, []string{"wip"}, 80, 24)

	_, cmd := send(m, runeKey('q'))

	if cmd == nil {
		t.Fatal("コマンドが返らない")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("コマンド = %T, want QuitMsg", cmd())
	}
}

// TestLabelPickerKeepsSnapshotOnFetch は一覧を出している間の取得完了で、印も書き先も
// 動かないことを検証する（自動更新は画面の状態を見ずに走るため）。
func TestLabelPickerKeepsSnapshotOnFetch(t *testing.T) {
	m, fake := labelPickerOn(t, labelsDir, 150, []string{"wip"}, 80, 24)
	m = selectLabel(t, m, "wip")
	m, _ = send(m, spaceKey)

	// 同じ issue 150 のラベルを空にし、選択行 0 を別の issue に差し替えた Result が届く。
	newer := labelIssueCard(151, nil)
	newer.Issue.UpdatedAt = at.Add(time.Hour)
	m, _ = send(m, fetchedMsg{res: &fetch.Result{Cards: []model.Card{newer, labelIssueCard(150, nil)}}, at: at})
	if got := m.rows[m.tab][m.cursor].number; got != 151 {
		t.Fatalf("選択行が差し替わっていない: number = %d, want 151", got)
	}

	if m.screen != screenLabels {
		t.Fatalf("取得完了で画面が変わった: screen = %d", m.screen)
	}
	if !m.labelPicker.origin["wip"] || m.labelPicker.want["wip"] {
		t.Errorf("印が動いた: 元 = %v / 今 = %v", m.labelPicker.origin, m.labelPicker.want)
	}

	m, cmd := send(m, enterKey)
	if cmd == nil {
		t.Fatal("Enter でコマンドが返らない")
	}
	m, _ = send(m, cmd())

	want := []gh.Call{{Method: "EditIssueLabels", Repo: "org/app", Number: 150, RemoveLabels: []string{"wip"}}}
	if got := callsOf(fake, "EditIssueLabels"); !reflect.DeepEqual(got, want) {
		t.Errorf("書き込み = %+v, want %+v", got, want)
	}
}

// TestLabelListView は見出し・ラベルの行・フッタが設計どおりに出ることを検証する。
func TestLabelListView(t *testing.T) {
	m, _ := labelPickerOn(t, labelsDir, 150, []string{"wip"}, 80, 24)

	lines := plain(m)
	if lines[0] != "ラベルを付け外し（変更 0 件）" {
		t.Errorf("1 行目 = %q", lines[0])
	}
	wantOrder(t, lines, "▶ [ ] docs", "[x] wip")
	wipLine, ok := lineWith(lines, "[x] wip")
	if !ok || strings.Contains(wipLine, "▶") {
		t.Errorf("選択していない行に ▶ がある: %q", wipLine)
	}
	footer := lines[len(lines)-1]
	for _, want := range []string{"j/k 選択", "Space 切替", "Enter 送信", "Esc 戻る", "q 終了"} {
		if !strings.Contains(footer, want) {
			t.Errorf("フッタに %q が無い: %q", want, footer)
		}
	}
}

// TestLabelMarksFollowSpace は印と見出しの変更件数が Space で動くことを検証する。
func TestLabelMarksFollowSpace(t *testing.T) {
	base, _ := labelPickerOn(t, labelsDir, 150, []string{"wip"}, 80, 24)

	one, _ := send(base, spaceKey)
	lines := plain(one)
	if lines[0] != "ラベルを付け外し（変更 1 件）" {
		t.Errorf("1 回目の 1 行目 = %q", lines[0])
	}
	if _, ok := lineWith(lines, "[+] docs"); !ok {
		t.Errorf("[+] docs が無い:\n%s", strings.Join(lines, "\n"))
	}

	two, _ := send(one, runeKey('j'), spaceKey)
	lines = plain(two)
	if lines[0] != "ラベルを付け外し（変更 2 件）" {
		t.Errorf("2 回目の 1 行目 = %q", lines[0])
	}
	wantOrder(t, lines, "[+] docs", "[-] wip")

	back, _ := send(one, spaceKey)
	lines = plain(back)
	if lines[0] != "ラベルを付け外し（変更 0 件）" {
		t.Errorf("同じ行で 2 回押した後の 1 行目 = %q", lines[0])
	}
	if _, ok := lineWith(lines, "[ ] docs"); !ok {
		t.Errorf("[ ] docs が無い:\n%s", strings.Join(lines, "\n"))
	}
	if _, ok := lineWith(lines, "[+] docs"); ok {
		t.Errorf("[+] docs が残っている:\n%s", strings.Join(lines, "\n"))
	}
}

// TestLabelListFollowsCursorWhenTaller は件数が表示できる行数を超えるとき、
// 選択に追従して表示の開始位置がずれることを検証する。
func TestLabelListFollowsCursorWhenTaller(t *testing.T) {
	// 高さ 10 なら表示できる行数は 7（見出し・空行・フッタを引く）。
	m, _ := labelPickerOn(t, labelsManyDir, 150, nil, 80, 10)

	lines := plain(m)
	if _, ok := lineWith(lines, "label-00"); !ok {
		t.Errorf("label-00 が無い:\n%s", strings.Join(lines, "\n"))
	}
	if _, ok := lineWith(lines, "label-07"); ok {
		t.Errorf("表示できる行数を超えた label-07 が出ている:\n%s", strings.Join(lines, "\n"))
	}

	for range 10 {
		m, _ = send(m, runeKey('j'))
	}
	lines = plain(m)

	selected, ok := lineWith(lines, "label-10")
	if !ok || !strings.Contains(selected, "▶") {
		t.Errorf("label-10 が選択行として出ていない: %q\n%s", selected, strings.Join(lines, "\n"))
	}
	if _, ok := lineWith(lines, "label-00"); ok {
		t.Errorf("表示がずれていない（label-00 が残っている）:\n%s", strings.Join(lines, "\n"))
	}
}

// TestSubmitLabelsWritesOnce は Enter が 1 回の書き込みにまとめ、先に一覧を閉じることを検証する。
func TestSubmitLabelsWritesOnce(t *testing.T) {
	m, fake := labelPickerOn(t, labelsDir, 150, []string{"wip"}, 100, 24)
	m = selectLabel(t, m, "docs")
	m, _ = send(m, spaceKey)
	m = selectLabel(t, m, "wip")
	m, _ = send(m, spaceKey)

	m, cmd := send(m, enterKey)

	if cmd == nil {
		t.Fatal("Enter でコマンドが返らない")
	}
	if m.screen != screenQueue {
		t.Errorf("送信の前に一覧が閉じていない: screen = %d", m.screen)
	}
	if footer := footerOf(plainText(m)); !strings.Contains(footer, "org/app #150 のラベルを更新中") {
		t.Errorf("フッタ = %q", footer)
	}

	msg := cmd()

	want := []gh.Call{
		{Method: "ListLabels", Repo: "org/app"},
		{Method: "ViewIssue", Repo: "org/app", Number: 150},
		{Method: "EditIssueLabels", Repo: "org/app", Number: 150,
			AddLabels: []string{"docs"}, RemoveLabels: []string{"wip"}},
	}
	if !reflect.DeepEqual(fake.Calls, want) {
		t.Fatalf("Calls = %+v, want %+v", fake.Calls, want)
	}

	m, _ = send(m, msg)

	if footer := footerOf(plainText(m)); !strings.Contains(footer, "org/app #150 のラベルを更新しました（+docs -wip）") {
		t.Errorf("結果のフッタ = %q", footer)
	}
	// Cards は書き換えない（反映は R / 自動更新に任せる）。
	if got := m.rows[m.tab][0].card.Issue.Labels; !reflect.DeepEqual(got, []string{"wip"}) {
		t.Errorf("Card の Labels = %v, want [wip]", got)
	}

	// 開き直した印は Cards のラベルのまま。
	m, _ = send(m, lKey)
	if m.labelPicker.want["docs"] || !m.labelPicker.want["wip"] {
		t.Errorf("開き直した印 = %v, want wip だけ", m.labelPicker.want)
	}
}

// TestSubmitLabelsOnPRUsesPRPath は対象が PR なら EditPRLabels を呼ぶことを検証する。
func TestSubmitLabelsOnPRUsesPRPath(t *testing.T) {
	m, fake := exampleLabelModel(t, 120, 24)
	m, _ = send(m, enterKey, enterKey)
	m = openLabels(t, m)
	m = selectLabel(t, m, "docs")
	m, _ = send(m, spaceKey)

	m, cmd := send(m, enterKey)
	if cmd == nil {
		t.Fatal("Enter でコマンドが返らない")
	}
	m, _ = send(m, cmd())

	want := []gh.Call{{Method: "EditPRLabels", Repo: "org/app", Number: 131, AddLabels: []string{"docs"}}}
	if got := callsOf(fake, "EditPRLabels"); !reflect.DeepEqual(got, want) {
		t.Errorf("EditPRLabels = %+v, want %+v", got, want)
	}
	if got := callsOf(fake, "EditIssueLabels"); len(got) != 0 {
		t.Errorf("EditIssueLabels = %+v, want 空", got)
	}
}

// TestSubmitLabelsWithoutChangesSkipsGh は変更 0 件なら gh を 1 度も呼ばないことを検証する。
func TestSubmitLabelsWithoutChangesSkipsGh(t *testing.T) {
	m, fake := labelPickerOn(t, labelsDir, 150, []string{"wip"}, 100, 24)

	m, cmd := send(m, enterKey)

	if cmd != nil {
		t.Error("変更 0 件でコマンドが返っている")
	}
	if m.screen != screenQueue {
		t.Errorf("画面 = %d, want キュー", m.screen)
	}
	if footer := footerOf(plainText(m)); !strings.Contains(footer, "org/app #150 のラベルに変更はありません") {
		t.Errorf("フッタ = %q", footer)
	}
	for _, c := range fake.Calls {
		if c.Method != "ListLabels" {
			t.Errorf("gh を呼んでいる: %+v", fake.Calls)
		}
	}
}

// TestSubmitLabelsMatchingAfterReread は読み直した現在のラベルと一致していれば
// 書き込まず、その旨をフッタに出すことを検証する。
func TestSubmitLabelsMatchingAfterReread(t *testing.T) {
	// 画面の Card のラベルは空だが、issue-151.json は docs が付いている。
	m, fake := labelPickerOn(t, labelsDir, 151, nil, 100, 24)
	m = selectLabel(t, m, "docs")
	m, _ = send(m, spaceKey)

	m, cmd := send(m, enterKey)
	if cmd == nil {
		t.Fatal("Enter でコマンドが返らない")
	}
	m, _ = send(m, cmd())

	if got := callsOf(fake, "ViewIssue"); len(got) != 1 {
		t.Errorf("ViewIssue = %+v, want 1 件", got)
	}
	for _, c := range fake.Calls {
		if strings.HasPrefix(c.Method, "Edit") {
			t.Errorf("書き込んでいる: %+v", fake.Calls)
		}
	}
	if footer := footerOf(plainText(m)); !strings.Contains(footer, "org/app #151 のラベルに変更はありません") {
		t.Errorf("フッタ = %q", footer)
	}
}

// TestSubmitLabelsFailureShowsError は書き込みの失敗を赤で出し、Cards を変えないことを検証する。
func TestSubmitLabelsFailureShowsError(t *testing.T) {
	fake := gh.NewFake(labelsDir)
	client := &errEditLabelsFake{
		Fake: fake,
		err:  errors.New("gh issue edit 150 -R org/app --add-label docs: exit 1: HTTP 403"),
	}
	m := labelModel([]model.Card{labelIssueCard(150, []string{"wip"})}, client, 120, 24)
	m = openLabels(t, m)
	m = selectLabel(t, m, "docs")
	m, _ = send(m, spaceKey)

	m, cmd := send(m, enterKey)
	if cmd == nil {
		t.Fatal("Enter でコマンドが返らない")
	}
	m, _ = send(m, cmd())

	footer := footerOf(plainText(m))
	for _, want := range []string{"のラベルを更新できません:", "HTTP 403"} {
		if !strings.Contains(footer, want) {
			t.Errorf("フッタに %q が無い: %q", want, footer)
		}
	}
	if !m.writeStatusErr {
		t.Error("エラーとして出ていない（赤にならない）")
	}
	if got := m.rows[m.tab][0].card.Issue.Labels; !reflect.DeepEqual(got, []string{"wip"}) {
		t.Errorf("Card の Labels = %v, want [wip]", got)
	}
}

// TestSubmitResultDoesNotStealScreen は結果が届いたときに画面を奪わないことを検証する。
func TestSubmitResultDoesNotStealScreen(t *testing.T) {
	m, _ := labelPickerOn(t, labelsDir, 150, []string{"wip"}, 120, 24)
	m = selectLabel(t, m, "docs")
	m, _ = send(m, spaceKey)
	m = selectLabel(t, m, "wip")
	m, _ = send(m, spaceKey)

	m, cmd := send(m, enterKey)
	if cmd == nil {
		t.Fatal("Enter でコマンドが返らない")
	}
	m, _ = send(m, questionKey)
	if m.screen != screenHelp {
		t.Fatalf("ヘルプ画面に移っていない: screen = %d", m.screen)
	}

	m, _ = send(m, cmd())

	if m.screen != screenHelp {
		t.Fatalf("画面 = %d, want ヘルプ", m.screen)
	}
	if footer := footerOf(plainText(m)); !strings.Contains(footer, "のラベルを更新しました（+docs -wip）") {
		t.Errorf("フッタ = %q", footer)
	}
}
