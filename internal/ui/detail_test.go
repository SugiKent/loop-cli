package ui

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/sugi-loop/internal/fetch"
	"github.com/SugiKent/sugi-loop/internal/gh"
	"github.com/SugiKent/sugi-loop/internal/model"
)

var (
	escKey   = codeKey(tea.KeyEscape)
	enterKey = codeKey(tea.KeyEnter)
	tabKey   = codeKey(tea.KeyTab)
)

// detailModel は幅 w・高さ h のサイズを与え、cards を取得完了として渡したキュー画面を返す。
func detailModel(w, h int, cards []model.Card) Model {
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: w, Height: h}, fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})
	return m
}

// issueCard は今やるタブに入る手書きの Card を組み立てる。
func issueCard(issue *model.Issue, prs []model.PR, summary string) model.Card {
	result := model.Result{Situation: model.SituationA, Priority: 1, Tab: model.TabNow, Summary: summary}
	issue.Repo, issue.Result = "org/app", result
	return model.Card{Issue: issue, PRs: prs, Result: result}
}

// prOf は手書きの PR。
func prOf(number int, state string, labels []string) model.PR {
	return model.PR{Repo: "org/app", Number: number, Title: "手書き PR", State: state, Labels: labels}
}

// linesOf は View から ANSI を除いた行を返す。
func linesOf(m Model) []string { return plain(m) }

// wantOrder は lines を連結した文字列の中で subs がこの順に現れることを確かめる。
func wantOrder(t *testing.T, lines []string, subs ...string) {
	t.Helper()
	text := strings.Join(lines, "\n")
	at := 0
	for _, sub := range subs {
		i := strings.Index(text[at:], sub)
		if i < 0 {
			t.Fatalf("%q が %d 文字目以降に無い:\n%s", sub, at, text)
		}
		at += i + len(sub)
	}
}

func TestDependsOn(t *testing.T) {
	got := dependsOn("集計が遅い。\n\ndepends on #12\nDepends on #34")
	if !slices.Equal(got, []int{12, 34}) {
		t.Errorf("dependsOn = %v, want [12 34]", got)
	}
	if got := dependsOn("集計が遅い。"); len(got) != 0 {
		t.Errorf("depends on が無い本文で %v", got)
	}
}

func TestEnterOpensCardDetailAndEscReturns(t *testing.T) {
	m := detailModel(120, 40, exampleResult(t).Cards)

	m, _ = send(m, enterKey)
	if m.screen != screenCard {
		t.Fatalf("Enter 後の画面 = %d, want カード詳細", m.screen)
	}
	if m.detail.card.Issue.Number != 108 {
		t.Errorf("詳細の対象 = #%d, want #108", m.detail.card.Issue.Number)
	}

	m, _ = send(m, escKey)
	if m.screen != screenQueue || m.tab != model.TabNow || m.cursor != 0 {
		t.Errorf("Esc 後 = screen %d tab %q cursor %d, want キュー / 今やる / 0", m.screen, m.tab, m.cursor)
	}
}

func TestEnterOnPROnlyCardOpensPRDetail(t *testing.T) {
	pr := prOf(60, "OPEN", []string{model.LabelDocs})
	pr.Result = model.Result{Situation: model.SituationD, Priority: 2, Tab: model.TabNow, Summary: "確認する"}
	card := model.Card{PRs: []model.PR{pr}, Result: pr.Result}

	m, _ := send(detailModel(120, 40, []model.Card{card}), enterKey)
	if m.screen != screenPR || m.currentPR().Number != 60 {
		t.Fatalf("Enter 後 = screen %d PR#%d, want PR 詳細 / #60", m.screen, m.currentPR().Number)
	}
	m, _ = send(m, escKey)
	if m.screen != screenQueue {
		t.Errorf("Esc 後の画面 = %d, want キュー", m.screen)
	}
}

func TestEnterOnEmptyTabDoesNothing(t *testing.T) {
	m := detailModel(120, 40, exampleResult(t).Cards)
	m, _ = send(m, runeKey('4')) // 異常タブは 0 行
	m, cmd := send(m, enterKey)
	if m.screen != screenQueue {
		t.Errorf("0 行のタブで画面が変わった: %d", m.screen)
	}
	if cmd != nil {
		t.Errorf("0 行のタブでコマンドが返った: %T", cmd())
	}
}

func TestQueueKeysDoNothingInDetail(t *testing.T) {
	m, _ := send(detailModel(120, 40, exampleResult(t).Cards), enterKey)
	for _, k := range []tea.Msg{runeKey('2'), runeKey('4'), runeKey('p'), runeKey('R')} {
		var cmd tea.Cmd
		m, cmd = send(m, k)
		if cmd != nil {
			t.Fatalf("詳細でキュー画面のキーがコマンドを返した: %T", cmd())
		}
		if m.screen != screenCard {
			t.Fatalf("詳細でキュー画面のキーが効いた: screen %d", m.screen)
		}
		if m.tab != model.TabNow {
			t.Fatalf("詳細でタブが変わった: %q", m.tab)
		}
	}
}

func TestFetchedWhileDetailOpenKeepsTarget(t *testing.T) {
	m, _ := send(detailModel(120, 40, exampleResult(t).Cards), enterKey)

	other := issueCard(&model.Issue{Number: 900, Title: "別の issue", UpdatedAt: at}, nil, "別を見る")
	m, _ = send(m, fetchedMsg{res: &fetch.Result{Cards: []model.Card{other}}, at: at})
	if m.screen != screenCard || m.detail.card.Issue.Number != 108 {
		t.Fatalf("取得完了で詳細が変わった: screen %d #%d", m.screen, m.detail.card.Issue.Number)
	}

	m, _ = send(m, escKey)
	rows := m.rows[model.TabNow]
	if len(rows) != 1 || rows[0].number != 900 {
		t.Errorf("Esc 後の今やるタブが新しい Result になっていない: %v", rows)
	}
}

func TestQuitFromDetail(t *testing.T) {
	m, _ := send(detailModel(120, 40, exampleResult(t).Cards), enterKey)
	_, cmd := send(m, runeKey('q'))
	if cmd == nil {
		t.Fatal("詳細で q に終了コマンドが返らなかった")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("返ったコマンドが終了メッセージを生まない: %T", cmd())
	}
}

func TestGoesBackAndForthBetweenIssueAndPR(t *testing.T) {
	m, _ := send(detailModel(120, 40, exampleResult(t).Cards), enterKey)
	for i, step := range []struct {
		key  tea.Msg
		want screen
	}{
		{runeKey('g'), screenPR},
		{runeKey('g'), screenCard},
		{enterKey, screenPR},
		{escKey, screenCard},
	} {
		m, _ = send(m, step.key)
		if m.screen != step.want {
			t.Fatalf("%d 回目の後の画面 = %d, want %d", i+1, m.screen, step.want)
		}
	}
	if m.detail.card.PRs[m.detail.prIdx].Number != 131 {
		t.Errorf("選択中の PR = #%d, want #131", m.detail.card.PRs[m.detail.prIdx].Number)
	}
}

func TestGDoesNothingOnPROnlyCard(t *testing.T) {
	pr := prOf(60, "OPEN", []string{model.LabelDocs})
	pr.Result = model.Result{Situation: model.SituationD, Priority: 2, Tab: model.TabNow}
	m, _ := send(detailModel(120, 40, []model.Card{{PRs: []model.PR{pr}, Result: pr.Result}}), enterKey)

	m, cmd := send(m, runeKey('g'))
	if m.screen != screenPR {
		t.Errorf("g で画面が変わった: %d", m.screen)
	}
	if cmd != nil {
		t.Errorf("g でコマンドが返った: %T", cmd())
	}
}

// threePRCard は propose 2 件（131 merged、140 merged で正本）と apply 1 件（151 open）を持つ Card。
func threePRCard() model.Card {
	p131 := prOf(131, "MERGED", []string{model.LabelPropose})
	p140 := prOf(140, "MERGED", []string{model.LabelPropose})
	p140.Canonical = true
	p151 := prOf(151, "OPEN", []string{model.LabelApply})
	p151.Body = "未確定の判断: 0 件\n\n本文。"
	p151.MergeState = &gh.PRMergeState{
		Mergeable:         "MERGEABLE",
		StatusCheckRollup: []gh.StatusCheck{{Typename: "CheckRun", Name: "test", Conclusion: "SUCCESS"}},
	}
	issue := &model.Issue{Number: 108, Title: "手書き issue", Body: "本文。", Labels: []string{model.LabelStagePropose}, UpdatedAt: at}
	return issueCard(issue, []model.PR{p131, p140, p151}, "対応する")
}

func TestTabMovesPRSelection(t *testing.T) {
	m, _ := send(detailModel(120, 40, []model.Card{threePRCard()}), enterKey)
	for i, want := range []int{1, 2, 0} {
		m, _ = send(m, tabKey)
		if m.detail.prIdx != want {
			t.Fatalf("%d 回目の Tab の後 prIdx = %d, want %d", i+1, m.detail.prIdx, want)
		}
	}
}

func TestCardDetailHeader(t *testing.T) {
	m, _ := send(detailModel(120, 40, exampleResult(t).Cards), enterKey)
	lines := linesOf(m)
	wantOrder(t, lines, "org/app #108", "ログインのセッション仕様を決める", "PR #131 の質問に答える", "段階: stage:propose", "[question]")
	for _, ng := range []string{"[blocked]", "[wip]", "depends on:"} {
		if strings.Contains(strings.Join(lines, "\n"), ng) {
			t.Errorf("ヘッダに %q がある", ng)
		}
	}
}

func TestLongHeaderLineIsTruncated(t *testing.T) {
	issue := &model.Issue{Number: 108, Title: strings.Repeat("あ", 30), UpdatedAt: at} // 表示幅 60
	m, _ := send(detailModel(40, 40, []model.Card{issueCard(issue, nil, "対応する")}), enterKey)

	line, ok := lineWith(linesOf(m), "org/app #")
	if !ok {
		t.Fatal("ヘッダ行が無い")
	}
	if w := ansi.StringWidth(line); w > 40 {
		t.Errorf("ヘッダ行の表示幅 = %d, want <= 40", w)
	}
	if !strings.HasSuffix(strings.TrimRight(line, " "), "…") {
		t.Errorf("切り詰めた行の末尾が … でない: %q", line)
	}
}

func TestHeaderWithoutStageAndWithDependsOn(t *testing.T) {
	issue := &model.Issue{Number: 200, Title: "集計", Body: "集計が遅い。\n\ndepends on #12\nDepends on #34", UpdatedAt: at}
	m, _ := send(detailModel(120, 40, []model.Card{issueCard(issue, nil, "対応する")}), enterKey)
	wantOrder(t, linesOf(m), "段階なし", "depends on: #12 #34")
}

func TestHeaderWithTwoStagesAndBadges(t *testing.T) {
	issue := &model.Issue{
		Number: 201, Title: "二重段階", UpdatedAt: at,
		Labels: []string{model.LabelStagePropose, model.LabelStageApply, model.LabelBlocked, model.LabelWip},
	}
	m, _ := send(detailModel(120, 40, []model.Card{issueCard(issue, nil, "対応する")}), enterKey)
	wantOrder(t, linesOf(m), "段階: stage:propose stage:apply", "[blocked]", "[wip]")
}

func TestPRListOfIssue108(t *testing.T) {
	m, _ := send(detailModel(120, 40, exampleResult(t).Cards), enterKey)
	lines := linesOf(m)

	row, ok := lineWith(lines, "[propose] PR#131")
	if !ok {
		t.Fatal("PR 131 の行が無い")
	}
	if !strings.HasPrefix(row, "▶") {
		t.Errorf("選択中の PR の行が ▶ で始まらない: %q", row)
	}
	// s20 で全 PR の merge 状態を取るので、pr-131.json の実値が出る（UNKNOWN・ci/legacy PENDING）。
	wantOrder(t, []string{row}, "[propose] PR#131 open", "1 行目なし", "labels: propose question", "checks 緑以外", "mergeable UNKNOWN")
	for _, want := range []string{"[apply] なし", "[archive] なし"} {
		if _, ok := lineWith(lines, want); !ok {
			t.Errorf("%q の行が無い", want)
		}
	}
}

func TestPRListMarksCanonical(t *testing.T) {
	m, _ := send(detailModel(120, 40, []model.Card{threePRCard()}), enterKey)
	lines := linesOf(m)
	wantOrder(t, lines, "[propose] PR#131 merged", "[propose] PR#140 merged（最新・正本）", "[apply] PR#151 open", "[archive] なし")

	row, _ := lineWith(lines, "PR#131")
	if strings.Contains(row, "（最新・正本）") {
		t.Errorf("PR#131 の行に正本の印がある: %q", row)
	}
	row, _ = lineWith(lines, "PR#151")
	wantOrder(t, []string{row}, "未確定 0 件", "checks 緑", "mergeable MERGEABLE")
}

func TestPRListMarkFollowsTab(t *testing.T) {
	m, _ := send(detailModel(120, 40, []model.Card{threePRCard()}), enterKey)
	for i, want := range []string{"PR#140", "PR#151", "PR#131"} {
		m, _ = send(m, tabKey)
		row, ok := lineWith(linesOf(m), "▶")
		if !ok {
			t.Fatalf("%d 回目の Tab の後に ▶ の行が無い", i+1)
		}
		if !strings.Contains(row, want) {
			t.Errorf("%d 回目の Tab の後の ▶ の行 = %q, want %s", i+1, row, want)
		}
	}
}

func TestPRWithoutStageGoesLast(t *testing.T) {
	issue := &model.Issue{Number: 300, Title: "段階無し PR", UpdatedAt: at}
	card := issueCard(issue, []model.PR{prOf(90, "OPEN", []string{model.LabelApply}), prOf(61, "OPEN", nil)}, "対応する")
	m, _ := send(detailModel(120, 40, []model.Card{card}), enterKey)
	wantOrder(t, linesOf(m), "[propose] なし", "[apply] PR#90 open", "[archive] なし", "[-] PR#61 open")
}

func TestCardBodyOfIssue108(t *testing.T) {
	m, _ := send(detailModel(120, 40, exampleResult(t).Cards), enterKey)
	text := plainText(m)
	if !strings.Contains(text, "認証まわりの仕様を決めたい") {
		t.Error("本文が出ていない")
	}
	for _, ng := range []string{"blocked-by:", "コメント: 取得失敗", "コメント: なし"} {
		if strings.Contains(text, ng) {
			t.Errorf("本文領域に %q がある", ng)
		}
	}
}

// blockedCard は blocked-by: human の AI コメント 1 件を持つ Card。
func blockedCard(body string) model.Card {
	issue := &model.Issue{
		Number: 400, Title: "方針を決める", Body: "方針。", UpdatedAt: at,
		Labels:   []string{model.LabelStagePropose, model.LabelBlocked, model.LabelQuestion},
		Comments: []model.Comment{{Author: "user-1", Body: body, CreatedAt: at, AI: true}},
	}
	return issueCard(issue, nil, "方針を決める")
}

func TestBlockedByHumanShowsQuestionsAndOptions(t *testing.T) {
	body := "<!-- routine -->\nblocked-by: human\n## Q1. 名前での絞り込みを含めるか\n- 選択肢 A（推奨）: 含めない\n- 選択肢 B: 含める"
	m, _ := send(detailModel(120, 40, []model.Card{blockedCard(body)}), enterKey)
	wantOrder(t, linesOf(m), "blocked-by: human", "Q1. 名前での絞り込みを含めるか", "A（推奨）: 含めない", "B: 含める")
}

func TestBlockedByHumanWithoutQuestionsShowsBody(t *testing.T) {
	body := "<!-- routine -->\nblocked-by: human\n次の方針をコメントで教えてください"
	m, _ := send(detailModel(120, 40, []model.Card{blockedCard(body)}), enterKey)
	wantOrder(t, linesOf(m), "blocked-by: human", "次の方針をコメントで教えてください")
}

// TestCommentsEmpty は、取得できてコメントが 0 件の issue が「なし」と出ることを検証する
// （issue-140.json の comments は空配列。s20 で全 issue のコメントを取る）。
func TestCommentsEmpty(t *testing.T) {
	res := exampleResult(t)
	m := detailModel(120, 40, res.Cards)
	// 今やるタブの 1 件目は issue 108 なので、issue 140 のカードを直接開く。
	m.detail = detailState{card: cardOf(t, res, 140)}
	m.screen = screenCard
	m.refreshDetail()

	text := plainText(m)
	if !strings.Contains(text, "起動時に設定ファイルが無いと落ちる") || !strings.Contains(text, "コメント: なし") {
		t.Errorf("issue 140 の詳細が想定と違う:\n%s", text)
	}
	for _, ng := range []string{"コメント: 取得失敗", "▌"} {
		if strings.Contains(text, ng) {
			t.Errorf("コメント 0 件の issue に %q がある", ng)
		}
	}
}

// TestCommentsFetchFailed は Comments が nil（取得失敗）の issue の表示を検証する。
func TestCommentsFetchFailed(t *testing.T) {
	issue := &model.Issue{Number: 500, Title: "取得に失敗した issue", Body: "本文。", UpdatedAt: at}
	m, _ := send(detailModel(120, 40, []model.Card{issueCard(issue, nil, "確認する")}), enterKey)

	text := plainText(m)
	if !strings.Contains(text, "コメント: 取得失敗") {
		t.Errorf("`コメント: 取得失敗` が無い:\n%s", text)
	}
	for _, ng := range []string{"コメント: なし", "▌"} {
		if strings.Contains(text, ng) {
			t.Errorf("取得失敗の issue に %q がある", ng)
		}
	}
}

// TestPRListFetchFailed は、カード詳細の PR 行の checks / mergeable の取得失敗表示を検証する。
func TestPRListFetchFailed(t *testing.T) {
	issue := &model.Issue{Number: 501, Title: "PR の詳細が取れないカード", UpdatedAt: at}
	card := issueCard(issue, []model.PR{prOf(131, "OPEN", []string{model.LabelPropose})}, "確認する")
	m, _ := send(detailModel(120, 40, []model.Card{card}), enterKey)

	row, ok := lineWith(linesOf(m), "[propose] PR#131")
	if !ok {
		t.Fatal("PR 131 の行が無い")
	}
	wantOrder(t, []string{row}, "[propose] PR#131 open", "checks 取得失敗", "mergeable 取得失敗")
}

func TestAICommentIsCollapsedAndExpandedByX(t *testing.T) {
	m, _ := send(detailModel(120, 40, exampleResult(t).Cards), enterKey)

	lines := linesOf(m)
	if _, ok := lineWith(lines, "▌AI  18:00  Q1: セッションの寿命は何日にしますか。  (+1 行)"); !ok {
		t.Fatalf("畳んだ AI コメントの見出しが無い:\n%s", strings.Join(lines, "\n"))
	}
	if strings.Contains(strings.Join(lines, "\n"), "Q2: 失効時は") {
		t.Error("畳んだ AI コメントに 2 行目が出ている")
	}
	for _, sub := range []string{"user-2  18:12", "寿命は 30 日で。"} {
		line, ok := lineWith(lines, sub)
		if !ok {
			t.Fatalf("%q の行が無い", sub)
		}
		if strings.HasPrefix(line, "▌") {
			t.Errorf("人のコメントの行が ▌ で始まる: %q", line)
		}
	}

	m, _ = send(m, runeKey('x'))
	lines = linesOf(m)
	if _, ok := lineWith(lines, "▌AI  18:00"); !ok {
		t.Error("展開後に AI の見出しが無い")
	}
	line, ok := lineWith(lines, "Q2: 失効時はログイン画面へ戻しますか。")
	if !ok {
		t.Fatal("展開後に 2 行目が無い")
	}
	if !strings.HasPrefix(line, "▌") {
		t.Errorf("展開した AI コメントの行が ▌ で始まらない: %q", line)
	}
	for _, ng := range []string{"<!-- routine -->", "(+1 行)"} {
		if strings.Contains(strings.Join(lines, "\n"), ng) {
			t.Errorf("展開後に %q がある", ng)
		}
	}

	m, _ = send(m, runeKey('x'))
	text := plainText(m)
	if !strings.Contains(text, "(+1 行)") || strings.Contains(text, "Q2: 失効時は") {
		t.Error("もう一度の x で折りたたみに戻らない")
	}
}

func TestReopenCollapsesAgain(t *testing.T) {
	m, _ := send(detailModel(120, 40, exampleResult(t).Cards), enterKey)
	m, _ = send(m, runeKey('x'), escKey, enterKey)
	text := plainText(m)
	if !strings.Contains(text, "(+1 行)") || strings.Contains(text, "Q2: 失効時は") {
		t.Error("開き直しても展開のままになっている")
	}
}

func TestPRDetailOfPR131(t *testing.T) {
	m, _ := send(detailModel(120, 40, exampleResult(t).Cards), enterKey, enterKey)
	wantOrder(t, linesOf(m),
		"org/app PR#131",
		"[propose] open  labels: propose question",
		"1 行目に未確定の判断が無い",
		"紐づく issue: #108",
		"mergeable: UNKNOWN BLOCKED",
		"test: SUCCESS",
		"ci/legacy: PENDING",
		"issue #108 の提案",
		"▌AI  19:31  Q1: マイグレーションを分けますか。  (+0 行)",
		"thread 未 resolve",
	)
	// s20 で全 PR の merge 状態と review thread を取るので、どちらも埋まる。
	for _, ng := range []string{"checks: 取得失敗", "review thread: 取得失敗"} {
		if strings.Contains(plainText(m), ng) {
			t.Errorf("PR 131 の詳細に %q がある", ng)
		}
	}
}

// TestPRDetailFetchFailed は PR 詳細の 3 箇所の取得失敗表示を検証する。
func TestPRDetailFetchFailed(t *testing.T) {
	issue := &model.Issue{Number: 502, Title: "詳細が取れない PR のカード", UpdatedAt: at}
	card := issueCard(issue, []model.PR{prOf(131, "OPEN", []string{model.LabelPropose})}, "確認する")
	m, _ := send(detailModel(120, 40, []model.Card{card}), enterKey, enterKey)

	wantOrder(t, linesOf(m), "checks: 取得失敗", "コメント: 取得失敗", "review thread: 取得失敗")
	for _, ng := range []string{"checks: なし", "コメント: なし", "review thread: なし"} {
		if strings.Contains(plainText(m), ng) {
			t.Errorf("取得失敗の PR 詳細に %q がある", ng)
		}
	}
}

func TestPRDetailReviewThreadsAndChecks(t *testing.T) {
	pr := prOf(151, "OPEN", []string{model.LabelApply})
	pr.MergeState = &gh.PRMergeState{
		Mergeable: "MERGEABLE", MergeStateStatus: "CLEAN",
		StatusCheckRollup: []gh.StatusCheck{
			{Typename: "CheckRun", Name: "test", Conclusion: "SUCCESS"},
			{Typename: "StatusContext", Context: "ci/legacy", State: "PENDING"},
		},
	}
	pr.ReviewThreads = []gh.ReviewThread{
		{IsResolved: true, Comments: []gh.ReviewComment{{Author: gh.Author{Login: "user-3"}, Body: "直しました", CreatedAt: at}}},
		{IsResolved: false, Comments: []gh.ReviewComment{{Author: gh.Author{Login: "user-1"}, Body: "<!-- routine -->\nこの分岐は残しますか", CreatedAt: at}}},
	}
	issue := &model.Issue{Number: 500, Title: "review thread", UpdatedAt: at}
	m, _ := send(detailModel(120, 40, []model.Card{issueCard(issue, []model.PR{pr}, "見る")}), enterKey, enterKey)

	lines := linesOf(m)
	wantOrder(t, lines, "mergeable: MERGEABLE CLEAN", "test: SUCCESS", "ci/legacy: PENDING")
	wantOrder(t, lines, "thread 未 resolve", "thread resolved")

	line, ok := lineWith(lines, "この分岐は残しますか")
	if !ok || !strings.HasPrefix(line, "▌") {
		t.Errorf("AI の review コメントの行が ▌ で始まらない: %q", line)
	}
	line, ok = lineWith(lines, "直しました")
	if !ok || strings.HasPrefix(line, "▌") {
		t.Errorf("人の review コメントの行が ▌ で始まる: %q", line)
	}
}

func TestPRDetailWithEmptyChecksAndThreads(t *testing.T) {
	pr := prOf(152, "OPEN", []string{model.LabelApply})
	pr.MergeState = &gh.PRMergeState{Mergeable: "UNKNOWN", StatusCheckRollup: []gh.StatusCheck{}}
	pr.ReviewThreads = []gh.ReviewThread{}
	pr.Comments = []model.Comment{}
	issue := &model.Issue{Number: 501, Title: "空", UpdatedAt: at}
	m, _ := send(detailModel(120, 40, []model.Card{issueCard(issue, []model.PR{pr}, "見る")}), enterKey, enterKey)

	text := plainText(m)
	for _, want := range []string{"mergeable: UNKNOWN", "checks: なし", "コメント: なし", "review thread: なし"} {
		if !strings.Contains(text, want) {
			t.Errorf("%q が無い:\n%s", want, text)
		}
	}
}

func TestRiskHeadingCommentIsCollapsedAsAI(t *testing.T) {
	pr := prOf(153, "OPEN", []string{model.LabelApply})
	pr.Comments = []model.Comment{model.CommentFrom(gh.Comment{
		Author:    gh.Author{Login: "user-2"},
		Body:      "PR #131 の評価です。\n\n## PR リスク評価\n\n- 影響範囲: 小",
		CreatedAt: at,
	})}
	issue := &model.Issue{Number: 502, Title: "リスク評価", UpdatedAt: at}
	m, _ := send(detailModel(120, 40, []model.Card{issueCard(issue, []model.PR{pr}, "見る")}), enterKey, enterKey)

	lines := linesOf(m)
	line, ok := lineWith(lines, "PR #131 の評価です。")
	if !ok {
		t.Fatal("リスク評価コメントの見出しが無い")
	}
	if !strings.HasPrefix(line, "▌AI") || !strings.Contains(line, "(+4 行)") {
		t.Errorf("AI として畳まれていない: %q", line)
	}
	if strings.Contains(strings.Join(lines, "\n"), "影響範囲: 小") {
		t.Error("畳んだのに本文が出ている")
	}
}

// longBodyCard は 60 段落の本文と open PR 1 件を持つ Card。
func longBodyCard() model.Card {
	paragraphs := make([]string, 60)
	for i := range paragraphs {
		paragraphs[i] = "行" + string(rune('0'+(i+1)/10)) + string(rune('0'+(i+1)%10))
	}
	issue := &model.Issue{Number: 600, Title: "長い本文", Body: strings.Join(paragraphs, "\n\n"), UpdatedAt: at}
	return issueCard(issue, []model.PR{prOf(601, "OPEN", []string{model.LabelPropose})}, "読む")
}

func TestBodyScrollsWithJ(t *testing.T) {
	m, _ := send(detailModel(100, 20, []model.Card{longBodyCard()}), enterKey)

	text := plainText(m)
	if !strings.Contains(text, "org/app #600") || !strings.Contains(text, "行01") {
		t.Fatalf("初期表示にヘッダか 行01 が無い:\n%s", text)
	}
	if strings.Contains(text, "行30") {
		t.Error("初期表示に 行30 がある")
	}

	m, _ = send(m, runeKey('j'), runeKey('j'), runeKey('j'), runeKey('j'), runeKey('j'))
	text = plainText(m)
	if !strings.Contains(text, "org/app #600") {
		t.Error("スクロール後にヘッダが消えた")
	}
	if strings.Contains(text, "行01") {
		t.Error("5 行スクロールしても 行01 が残っている")
	}
	if !strings.Contains(text, "行06") {
		t.Errorf("5 行スクロール後に 行06 が無い:\n%s", text)
	}
}

func TestScrollResetsWhenMovingBetweenScreens(t *testing.T) {
	m, _ := send(detailModel(100, 20, []model.Card{longBodyCard()}), enterKey)
	m, _ = send(m, runeKey('j'), runeKey('j'), runeKey('j'), runeKey('j'), runeKey('j'))
	m, _ = send(m, enterKey, escKey)
	if !strings.Contains(plainText(m), "行01") {
		t.Error("PR 詳細から戻ってもスクロール位置が先頭に戻っていない")
	}
}

func TestDetailFooters(t *testing.T) {
	m, _ := send(detailModel(120, 40, []model.Card{longBodyCard()}), enterKey)

	lines := linesOf(m)
	footer := lines[len(lines)-1]
	for _, want := range []string{"Esc 戻る", "Tab PR 選択", "x 展開", "a 回答", "t todo", "o ブラウザ", "? ヘルプ", "q 終了"} {
		if !strings.Contains(footer, want) {
			t.Errorf("カード詳細のフッタに %q が無い: %q", want, footer)
		}
	}
	for _, ng := range []string{"1-4/Tab タブ", "R 更新", "j/k スクロール"} {
		if strings.Contains(footer, ng) {
			t.Errorf("カード詳細のフッタに %q がある: %q", ng, footer)
		}
	}

	m, _ = send(m, enterKey)
	lines = linesOf(m)
	footer = lines[len(lines)-1]
	for _, want := range []string{"Esc 戻る", "g issue へ", "a 回答", "o ブラウザ", "? ヘルプ", "q 終了"} {
		if !strings.Contains(footer, want) {
			t.Errorf("PR 詳細のフッタに %q が無い: %q", want, footer)
		}
	}
	for _, ng := range []string{"Tab PR 選択", "t todo", "R 更新"} {
		if strings.Contains(footer, ng) {
			t.Errorf("PR 詳細のフッタに %q がある: %q", ng, footer)
		}
	}
}

func TestFooterWithoutPRs(t *testing.T) {
	res := exampleResult(t)
	m := detailModel(120, 40, res.Cards)
	m.detail = detailState{card: cardOf(t, res, 140)}
	m.screen = screenCard
	m.refreshDetail()

	lines := linesOf(m)
	footer := lines[len(lines)-1]
	for _, want := range []string{"Esc 戻る", "x 展開", "t todo", "o ブラウザ"} {
		if !strings.Contains(footer, want) {
			t.Errorf("フッタに %q が無い: %q", want, footer)
		}
	}
	for _, ng := range []string{"Tab PR 選択", "Enter PR を開く", "g PR へ"} {
		if strings.Contains(footer, ng) {
			t.Errorf("PR の無いカードのフッタに %q がある: %q", ng, footer)
		}
	}
}

func TestNarrowTerminalKeepsOnePane(t *testing.T) {
	m, _ := send(detailModel(60, 40, exampleResult(t).Cards), enterKey)
	text := plainText(m)
	for _, want := range []string{"org/app #108", "PR#131", "認証まわりの仕様を決めたい"} {
		if !strings.Contains(text, want) {
			t.Errorf("狭い端末の詳細に %q が無い:\n%s", want, text)
		}
	}
	if strings.Contains(text, "p プレビュー") {
		t.Error("詳細にプレビュー切替のヒントがある")
	}
}

func TestShortTerminalDropsPRList(t *testing.T) {
	issue := &model.Issue{Number: 700, Title: "低い端末", Body: "行01", UpdatedAt: at}
	prs := []model.PR{
		prOf(701, "OPEN", []string{model.LabelPropose}),
		prOf(702, "OPEN", []string{model.LabelApply}),
		prOf(703, "OPEN", []string{model.LabelArchive}),
	}
	m, _ := send(detailModel(120, 7, []model.Card{issueCard(issue, prs, "読む")}), enterKey)

	lines := linesOf(m)
	text := strings.Join(lines, "\n")
	for _, want := range []string{"org/app #700", "行01", "q 終了"} {
		if !strings.Contains(text, want) {
			t.Errorf("低い端末の詳細に %q が無い:\n%s", want, text)
		}
	}
	if strings.Contains(text, "[archive] PR#") {
		t.Error("本文 1 行を確保するために PR 一覧が削れていない")
	}
	if len(lines) != 7 {
		t.Errorf("行数 = %d, want 7", len(lines))
	}
}
