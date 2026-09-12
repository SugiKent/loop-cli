package ui

import (
	"errors"
	"image/color"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/model"
	"github.com/SugiKent/loop-cli/internal/snapshot"
)

// 色が付いたことは「色を指定するエスケープ + 語 + リセット」がこの順で連続して現れることで
// 確かめる。エスケープが画面のどこかにあることだけを見ると、幅で切り詰められて語が 1 文字も
// 描かれていない場合に偽の合格が出る（切り詰めは切り捨てた範囲のエスケープを行末に残す）。
// どの語がどの色になるかは color_test.go が 16 進値で固定しているので、ここでは画面に届くことを見る。

// coloredState は暗い端末（既定）の状態語。
func coloredState(word string) string { return renderStateWord(word, true) }

// sgrEnd は色を指定するエスケープ。語の直前がこれで終わっていれば、その語は塗られている。
var sgrEnd = regexp.MustCompile(`\x1b\[[0-9;]*m$`)

// wantIn は画面に want が現れることを確かめる。
func wantIn(t *testing.T, view, want, what string) {
	t.Helper()
	if !strings.Contains(view, want) {
		t.Errorf("%s が色付きで現れない: want %q", what, want)
	}
}

// notIn は画面に ng が現れないことを確かめる。
func notIn(t *testing.T, view, ng, what string) {
	t.Helper()
	if strings.Contains(view, ng) {
		t.Errorf("%s: %q が現れている", what, ng)
	}
}

// wantPlain は語が色を伴わずに現れることを確かめる。語が現れるすべての位置について、
// 直前が色を指定するエスケープで終わっていないことを見る。
func wantPlain(t *testing.T, view, word, what string) {
	t.Helper()
	if !strings.Contains(view, word) {
		t.Fatalf("%s が画面に無い: %q", what, word)
	}
	for at := 0; ; {
		i := strings.Index(view[at:], word)
		if i < 0 {
			return
		}
		i += at
		if sgrEnd.MatchString(view[max(i-32, 0):i]) {
			t.Errorf("%s に色が付いている: %q", what, word)
			return
		}
		at = i + len(word)
	}
}

// exampleModel は example の Result を渡した幅 w・高さ h の Model。
func exampleModel(t *testing.T, w, h int) Model {
	t.Helper()
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: w, Height: h}, fetchedMsg{res: exampleResult(t), at: at})
	return m
}

// colorCard はプレビューの 1 行目（本文の 1 行目 + labels）が出る手書きの Card。
// プレビューは本文の 1 行目が空だとラベルの行ごと出さないので、Body を持たせる。
func colorCard(labels []string) model.Card {
	result := model.Result{Situation: model.SituationA, Priority: 1, Tab: model.TabNow, Summary: "回答する"}
	issue := &model.Issue{
		Repo: "org/app", Number: 150, Title: "手書き", Body: "本文の 1 行目", UpdatedAt: at,
		Labels: labels, Result: result,
	}
	return model.Card{Issue: issue, Result: result}
}

// proposeColors は org/app の stage:propose を color として持つラベル色の表。
func proposeColors(color string) map[string]map[string]string {
	return map[string]map[string]string{"org/app": {"stage:propose": color}}
}

// --- queue-screen: プレビューの 1 行目 ---

func TestPreviewLabelsAreColoredAndTableRowKeepsKindColor(t *testing.T) {
	// 今やるタブの Card（issue 108）の主体は PR 131 なので、プレビューは PR のラベルを出す。
	m := exampleModel(t, 100, 24)
	view := m.View().Content

	wantIn(t, view, renderLabelName("propose", "0e8a16"), "プレビューの propose")
	wantIn(t, view, renderLabelName("question", "d876e3"), "プレビューの question")

	if line, ok := lineWith(plain(m), "labels: "); !ok || !strings.Contains(line, "labels: propose question") {
		t.Errorf("ANSI を除いた 1 行目が変わった: %q", line)
	}
	// 表の行は種別の色のままで、ラベルの背景色は入らない（design.md D6）。
	row := strings.Split(view, "\n")[1]
	notIn(t, row, "48;2;", "表の選択行")
	wantIn(t, row, "38;2;216;118;227", "表の選択行の種別（質問）の色")
}

func TestLabelOutsideTheColorTableIsDrawnPlain(t *testing.T) {
	// 取得の後に GitHub 側で増えたラベルは色の表に無い。
	cards := []model.Card{colorCard([]string{"stage:propose", "手書きラベル"})}
	m, _ := send(loaded(cards), fetchedMsg{res: &fetch.Result{Cards: cards, LabelColors: proposeColors("0e8a16")}, at: at})

	view := m.View().Content
	wantIn(t, view, renderLabelName("stage:propose", "0e8a16"), "表にあるラベル")
	wantPlain(t, view, "手書きラベル", "色の表に無いラベル")
}

// 異常の Card は行全体を黄の背景で包む。その内側に色を置くと、内側のリセットが外側の背景を
// 落として行末まで背景が消える（design.md D6）。
func TestAbnormalRowKeepsItsBackgroundToTheRightEdge(t *testing.T) {
	result := model.Result{Situation: model.SituationF, Priority: 0, Tab: model.TabAbnormal, Summary: "壊れている"}
	// タイトルを折り返させて継続行も作る（背景は継続行でも途切れてはならない）。
	issue := &model.Issue{Repo: "org/app", Number: 9, UpdatedAt: at,
		Title:  "段階ラベルが 2 つ付いていて、どちらの段階なのか決められない issue",
		Labels: []string{"stage:propose", "stage:apply"}, Result: result}
	// 表の行がラベル色を引けるようにしておく（引けない状態では D6 の退行を検出できない）。
	cards := []model.Card{{Issue: issue, Result: result}}
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 80, Height: 24},
		fetchedMsg{res: &fetch.Result{Cards: cards, LabelColors: proposeColors("0e8a16")}, at: at})
	m, _ = send(m, runeKey('4')) // 異常タブ

	rows := strings.Split(m.View().Content, "\n")[1:3]
	for i, row := range rows {
		if n := strings.Count(row, "\x1b[m"); n != 1 {
			t.Errorf("異常の %d 行目のリセットが %d 個（末尾の 1 個だけであるべき）: %q", i+1, n, row)
		}
		if w := ansi.StringWidth(row); w != 80 {
			t.Errorf("異常の %d 行目の表示幅 = %d, want 80: %q", i+1, w, row)
		}
	}
}

// --- queue-screen: Model のラベル色の表 ---

func TestLabelColorsAreReplacedOnEachFetch(t *testing.T) {
	cards := []model.Card{colorCard([]string{"stage:propose"})}
	m := loaded(cards)

	m, _ = send(m, fetchedMsg{res: &fetch.Result{Cards: cards, LabelColors: proposeColors("0e8a16")}, at: at})
	wantIn(t, m.View().Content, renderLabelName("stage:propose", "0e8a16"), "1 回目の取得の色")

	m, _ = send(m, fetchedMsg{res: &fetch.Result{Cards: cards, LabelColors: proposeColors("1d76db")}, at: at})
	wantIn(t, m.View().Content, renderLabelName("stage:propose", "1d76db"), "2 回目の取得の色")
}

func TestLabelColorsSurviveFetchFailure(t *testing.T) {
	cards := []model.Card{colorCard([]string{"stage:propose"})}
	m, _ := send(loaded(cards), fetchedMsg{res: &fetch.Result{Cards: cards, LabelColors: proposeColors("0e8a16")}, at: at})

	m, _ = send(m, fetchedMsg{err: errors.New("gh search: exit 1"), at: at})

	wantIn(t, m.View().Content, renderLabelName("stage:propose", "0e8a16"), "取得失敗の後の色")
}

func TestSnapshotOnlyStartHasNoLabelColors(t *testing.T) {
	cards := []model.Card{colorCard([]string{"stage:propose"})}
	m := newModelOpts(nil, Options{Snapshot: &snapshot.Snapshot{Cards: cards, At: at}})

	wantPlain(t, m.View().Content, "stage:propose", "スナップショットだけの起動直後のラベル名")
}

// --- queue-screen: 状態語と端末の背景 ---

func TestStateColorsFollowTerminalBackgroundMsg(t *testing.T) {
	m, _ := send(exampleModel(t, 100, 24), enterKey)
	wantIn(t, m.View().Content, coloredState("open"), "背景色を答えない端末の open（暗い側）")

	light, _ := send(m, tea.BackgroundColorMsg{Color: color.White})
	wantIn(t, light.View().Content, renderStateWord("open", false), "明るい端末の open")
	notIn(t, light.View().Content, coloredState("open"), "明るい端末で暗い側の色")
}

// --- card-detail ---

func TestPRListRowLabelsAndStatesAreColored(t *testing.T) {
	m, _ := send(exampleModel(t, 100, 24), enterKey)
	view := m.View().Content

	wantIn(t, view, renderLabelName("propose", "0e8a16"), "PR 一覧行の段階ラベル")
	wantIn(t, view, renderLabelName("question", "d876e3"), "PR 一覧行の question")
	wantIn(t, view, coloredState("open"), "PR の状態")
	wantIn(t, view, coloredState("UNKNOWN"), "mergeable の値")
	wantIn(t, view, coloredState("緑以外"), "checks の値")

	row, ok := lineWith(plain(m), "PR#131")
	if !ok {
		t.Fatal("PR 一覧行が無い")
	}
	for _, want := range []string{"[propose] PR#131 open", "labels: propose question", "checks 緑以外 mergeable UNKNOWN"} {
		if !strings.Contains(row, want) {
			t.Errorf("ANSI を除いた PR 一覧行に %q が無い: %q", want, row)
		}
	}
	// 見出しは塗らない（SGR は必ず `m` で閉じるので、`m` の直後に見出しが来ない）。
	notIn(t, view, "mchecks", "checks の見出し")
	notIn(t, view, "mmergeable", "mergeable の見出し")
	notIn(t, view, "mlabels:", "labels: の見出し")
}

func TestCardHeaderStageAndBadgesAreColored(t *testing.T) {
	m, _ := send(exampleModel(t, 100, 24), enterKey)
	view := m.View().Content

	wantIn(t, view, "段階: "+renderLabelName("stage:propose", "0e8a16"), "ヘッダの段階ラベル")
	wantIn(t, view, "["+renderLabelName("question", "d876e3")+"]", "ヘッダの question の badge")
}

func TestPRDetailHeaderAndCheckRowsAreColored(t *testing.T) {
	m, _ := send(exampleModel(t, 100, 24), enterKey, enterKey) // カード詳細 -> PR 詳細
	view := m.View().Content

	// ヘッダ: [<段階>] <状態>  labels: <各ラベル>
	wantIn(t, view, "["+renderLabelName("propose", "0e8a16")+"] "+coloredState("open"), "ヘッダの段階ラベルと PR の状態")
	wantIn(t, view, "labels: "+renderLabelName("propose", "0e8a16")+" "+renderLabelName("question", "d876e3"),
		"ヘッダの labels")
	// 本文: mergeable の 2 値と checks の各行。チェック名と見出しは塗らない。
	wantIn(t, view, "mergeable: "+coloredState("UNKNOWN")+" "+coloredState("BLOCKED"), "mergeable の 2 つの値")
	wantIn(t, view, "test: "+coloredState("SUCCESS"), "CheckRun の結果")
	wantIn(t, view, "ci/legacy: "+coloredState("PENDING"), "StatusContext の状態")
	wantPlain(t, view, "test:", "チェック名")
	wantPlain(t, view, "mergeable:", "mergeable の見出し")
}

func TestFetchFailureWordsAreColored(t *testing.T) {
	issue := &model.Issue{Number: 108, Title: "手書き", UpdatedAt: at}
	pr := prOf(131, "OPEN", []string{"propose"}) // MergeState / Comments / ReviewThreads は nil（取得失敗）
	m := detailModel(100, 24, []model.Card{issueCard(issue, []model.PR{pr}, "回答する")})
	m, _ = send(m, enterKey)

	view := m.View().Content
	wantIn(t, view, "checks "+coloredState("取得失敗"), "PR 一覧行の checks 取得失敗")
	wantIn(t, view, "mergeable "+coloredState("取得失敗"), "PR 一覧行の mergeable 取得失敗")

	prDetail, _ := send(m, enterKey)
	detailView := prDetail.View().Content
	wantIn(t, detailView, "checks: "+coloredState("取得失敗"), "本文の checks 取得失敗")
	wantIn(t, detailView, "コメント: "+coloredState("取得失敗"), "コメントの取得失敗")
	wantIn(t, detailView, "review thread: "+coloredState("取得失敗"), "review thread の取得失敗")
}

// 色を引ける状態と引けない状態で、ANSI を除いた画面が一致する（design.md D2）。
func TestColorsDoNotChangeTheDisplayedText(t *testing.T) {
	cards := []model.Card{colorCard([]string{"stage:propose"})}
	colored, _ := send(loaded(cards), fetchedMsg{res: &fetch.Result{Cards: cards, LabelColors: proposeColors("0e8a16")}, at: at})
	plainOne, _ := send(loaded(cards), fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})

	if got, want := plainText(colored), plainText(plainOne); got != want {
		t.Errorf("ANSI を除いた画面が色の有無で変わった:\n色あり:\n%s\n色なし:\n%s", got, want)
	}
}

// --- merge-pr ---

func TestMergeConfirmLabelsAndStatesAreColored(t *testing.T) {
	m, _ := mergeModel(t, "testdata/merge", Options{})
	view := confirmed(t, m).View().Content

	wantIn(t, view, renderLabelName("propose", "0e8a16"), "確認画面のラベル")
	wantIn(t, view, "mergeable: "+coloredState("MERGEABLE")+" "+coloredState("CLEAN"), "mergeable の 2 つの値")
	wantIn(t, view, "checks: "+coloredState("緑"), "checks の値")
}

func TestMergeConfirmBadStatesAreRed(t *testing.T) {
	m, _ := mergeModel(t, "testdata/merge-conflict", Options{})
	view := confirmed(t, m).View().Content

	wantIn(t, view, "mergeable: "+coloredState("CONFLICTING")+" "+coloredState("DIRTY"), "悪い merge 状態")
	wantIn(t, view, "checks: "+coloredState("緑以外"), "緑以外の checks")
}

// --- close-issue-pr ---

func TestCloseConfirmLabelsAreColored(t *testing.T) {
	m, _ := closeModel(t)
	view := closeConfirmed(t, m).View().Content

	// キュー画面の c の対象は選択行の主体（example では PR 131）。
	wantIn(t, view, renderLabelName("propose", "0e8a16"), "close の確認画面のラベル")
	wantIn(t, view, renderLabelName("question", "d876e3"), "close の確認画面の question")
	if line, ok := lineWith(plain(closeConfirmed(t, m)), "labels: "); !ok ||
		!strings.Contains(line, "labels: propose question") {
		t.Errorf("ANSI を除いた labels の行が変わった: %q", line)
	}
}

func TestCloseConfirmNashiIsNotColored(t *testing.T) {
	issue := &model.Issue{Number: 140, Title: "ラベル無し", UpdatedAt: at}
	m := detailModel(100, 24, []model.Card{issueCard(issue, nil, "todo にする")})
	m, _ = send(m, cKey)

	line, ok := lineWith(strings.Split(m.View().Content, "\n"), "labels: なし")
	if !ok {
		t.Fatal("labels: なし の行が無い")
	}
	if strings.Contains(line, "\x1b[") {
		t.Errorf("labels: なし の行に色が付いた: %q", line)
	}
}

// --- label-picker ---

func TestLabelListNamesUseTheirOwnColor(t *testing.T) {
	m, _ := labelPickerOn(t, labelsDir, 150, []string{"wip"}, 80, 24)
	view := m.View().Content

	wantIn(t, view, renderLabelName("docs", "0075ca"), "一覧の docs")
	wantIn(t, view, renderLabelName("wip", "fbca04"), "一覧の wip")
	for _, want := range []string{"▶ [ ] docs", "[x] wip"} {
		if _, ok := lineWith(plain(m), want); !ok {
			t.Errorf("ANSI を除いた一覧に %q が無い", want)
		}
	}
}

func TestLabelListWithoutColorIsDrawnPlain(t *testing.T) {
	m, _ := labelPickerOn(t, "testdata/labels-nocolor", 150, nil, 80, 24)

	line, ok := lineWith(strings.Split(m.View().Content, "\n"), "色なし")
	if !ok {
		t.Fatal("一覧に 色なし の行が無い")
	}
	if strings.Contains(line, "\x1b[") {
		t.Errorf("色を持たないラベルの行に色が付いた: %q", line)
	}
}
