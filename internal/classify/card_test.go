package classify

import (
	"testing"
	"time"

	"github.com/SugiKent/loop-cli/internal/model"
)

func mergedPR(number int, labels ...string) model.PR {
	return model.PR{Repo: "org/app", Number: number, State: "MERGED", Labels: labels}
}

func TestCardPicksHighestSituation(t *testing.T) {
	is := issue(108, model.LabelStagePropose, model.LabelWip)
	pr := openPR(131, model.LabelPropose, model.LabelQuestion)
	pr.Comments = []model.Comment{aiComment()}

	got := Card(model.Card{Issue: &is, PRs: []model.PR{pr}}, model.ModeSDD, updatedNow, 0)

	if got.Result.Situation != model.SituationA || got.Result.Tab != model.TabNow {
		t.Errorf("Card.Result = %+v, want A / 今やる", got.Result)
	}
	if got.Result.Summary != "PR #131 の質問に答える" {
		t.Errorf("Summary = %q", got.Result.Summary)
	}
	if got.PRs[0].Result.Situation != model.SituationA {
		t.Errorf("PRs[0] = %q, want A", got.PRs[0].Result.Situation)
	}
	if got.Issue.Result.Situation != model.SituationInProgress {
		t.Errorf("Issue = %q, want in-progress", got.Issue.Result.Situation)
	}
}

func TestCardPrefersIssueOnTie(t *testing.T) {
	is := issue(108, model.LabelStagePropose, model.LabelStageApply)
	pr := openPR(131, model.LabelPropose, model.LabelApply)

	got := Card(model.Card{Issue: &is, PRs: []model.PR{pr}}, model.ModeSDD, updatedNow, 0)

	if got.Result.Summary != "#108 に段階ラベルが 2 つ以上ある" {
		t.Errorf("Summary = %q, want issue のもの（同点は issue 優先）", got.Result.Summary)
	}
}

func TestCardAllInProgressUsesOpenPRSummary(t *testing.T) {
	is := issue(108, model.LabelStageApply, model.LabelWip)
	pr := openPR(151, model.LabelApply)
	pr.Comments = []model.Comment{humanComment()}

	got := Card(model.Card{Issue: &is, PRs: []model.PR{pr}}, model.ModeSDD, updatedNow, 0)

	if got.Result.Situation != model.SituationInProgress || got.Result.Tab != model.TabInProgress {
		t.Errorf("Card.Result = %+v, want in-progress / 進行中", got.Result)
	}
	if got.Result.Summary != "PR #151 は auto-fix が受け取り中" {
		t.Errorf("Summary = %q, want open PR のもの", got.Result.Summary)
	}
}

func TestCardWithoutOpenPRUsesIssueSummary(t *testing.T) {
	is := issue(108, model.LabelStageApply, model.LabelWip)

	got := Card(model.Card{Issue: &is, PRs: []model.PR{mergedPR(131, model.LabelPropose)}}, model.ModeSDD, updatedNow, 0)

	if got.Result.Summary != "#108 は AI が作業中" {
		t.Errorf("Summary = %q, want issue のもの", got.Result.Summary)
	}
}

func TestCardWithoutIssue(t *testing.T) {
	got := Card(model.Card{PRs: []model.PR{openPR(160, model.LabelDocs)}}, model.ModeSDD, updatedNow, 0)

	if got.Result.Situation != model.SituationG {
		t.Errorf("Card.Result.Situation = %q, want G", got.Result.Situation)
	}
}

func TestCardDoesNotMutateInput(t *testing.T) {
	is := issue(108, model.LabelStagePropose)
	in := model.Card{Issue: &is, PRs: []model.PR{mergedPR(131, model.LabelPropose), openPR(151, model.LabelApply)}}

	Card(in, model.ModeSDD, updatedNow, 0)

	if is.Result != (model.Result{}) {
		t.Errorf("入力の Issue.Result が変わっています: %+v", is.Result)
	}
	for i, pr := range in.PRs {
		if pr.Result != (model.Result{}) || pr.Canonical {
			t.Errorf("入力の PRs[%d] が変わっています: Result=%+v Canonical=%v", i, pr.Result, pr.Canonical)
		}
	}
	if in.Result != (model.Result{}) {
		t.Errorf("入力の Card.Result が変わっています: %+v", in.Result)
	}
}

func TestCardCanonicalIgnoresOlderMergedPR(t *testing.T) {
	is := issue(108, model.LabelStageApply)
	old := mergedPR(131, model.LabelPropose, model.LabelQuestion)
	latest := mergedPR(140, model.LabelPropose)
	open := openPR(151, model.LabelApply)
	open.Comments = []model.Comment{humanComment()}

	got := Card(model.Card{Issue: &is, PRs: []model.PR{old, latest, open}}, model.ModeSDD, updatedNow, 0)

	if got.PRs[0].Canonical {
		t.Error("古い merge 済み PR #131 が Canonical になっています")
	}
	if !got.PRs[1].Canonical {
		t.Error("番号最大の merge 済み PR #140 が Canonical ではありません")
	}
	if got.PRs[0].Result != (model.Result{}) || got.PRs[1].Result != (model.Result{}) {
		t.Error("merge 済み PR の Result はゼロ値であるべき")
	}
	if got.Result.Situation != model.SituationInProgress {
		t.Errorf("Card.Result.Situation = %q, want in-progress（A にも F にもならない）", got.Result.Situation)
	}
}

func TestCardCanonicalPerStage(t *testing.T) {
	got := Card(model.Card{PRs: []model.PR{
		mergedPR(131, model.LabelPropose),
		mergedPR(151, model.LabelApply),
	}}, model.ModeSDD, updatedNow, 0)

	if !got.PRs[0].Canonical || !got.PRs[1].Canonical {
		t.Errorf("段階が違えば両方 Canonical であるべき: %v / %v", got.PRs[0].Canonical, got.PRs[1].Canonical)
	}
}

// graceNow は猶予のテストの基準時刻。updatedNow（ゼロ値）だと UpdatedAt との前後を作れない。
var graceNow = time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)

// otherPR は「その他」になるラベルもコメントも無い open PR を、updatedAt の古さを指定して作る。
func otherPR(number int, age time.Duration) model.PR {
	pr := openPR(number)
	pr.UpdatedAt = graceNow.Add(-age)
	return pr
}

func TestCardSettlesRecentOtherPR(t *testing.T) {
	got := Card(model.Card{PRs: []model.PR{otherPR(61, 10*time.Minute)}}, model.ModeSDD, graceNow, 30*time.Minute)

	if got.PRs[0].Result.Situation != model.SituationInProgress {
		t.Errorf("PRs[0].Result.Situation = %q, want in-progress", got.PRs[0].Result.Situation)
	}
	if got.Result.Situation != model.SituationInProgress || got.Result.Tab != model.TabInProgress {
		t.Errorf("Card.Result = %+v, want in-progress / 進行中", got.Result)
	}
	if got.Result.Priority != 7 {
		t.Errorf("Card.Result.Priority = %d, want 7", got.Result.Priority)
	}
	if want := "PR #61 はどの局面にも当たらない（更新から 30m は様子見）"; got.Result.Summary != want {
		t.Errorf("Summary = %q, want %q", got.Result.Summary, want)
	}
}

func TestCardKeepsOtherAtExactGrace(t *testing.T) {
	got := Card(model.Card{PRs: []model.PR{otherPR(61, 30*time.Minute)}}, model.ModeSDD, graceNow, 30*time.Minute)

	if got.Result.Situation != model.SituationOther || got.Result.Tab != model.TabNow {
		t.Errorf("Card.Result = %+v, want other / 今やる（ちょうど猶予は含まない）", got.Result)
	}
	if want := "PR #61 はどの局面にも当たらない"; got.Result.Summary != want {
		t.Errorf("Summary = %q, want %q", got.Result.Summary, want)
	}
}

func TestCardZeroGraceKeepsOther(t *testing.T) {
	got := Card(model.Card{PRs: []model.PR{otherPR(61, 10*time.Minute)}}, model.ModeSDD, graceNow, 0)

	if got.Result.Situation != model.SituationOther {
		t.Errorf("Card.Result.Situation = %q, want other（猶予 0 は置き換えない）", got.Result.Situation)
	}
}

func TestCardZeroUpdatedAtKeepsOther(t *testing.T) {
	got := Card(model.Card{PRs: []model.PR{openPR(61)}}, model.ModeSDD, graceNow, 30*time.Minute)

	if got.Result.Situation != model.SituationOther {
		t.Errorf("Card.Result.Situation = %q, want other（UpdatedAt が取れていない PR は隠さない）", got.Result.Situation)
	}
}

func TestCardDoesNotSettleOtherIssue(t *testing.T) {
	is := issue(108, model.LabelStagePropose, model.LabelBlocked, model.LabelQuestion)
	is.UpdatedAt = graceNow.Add(-5 * time.Minute)

	got := Card(model.Card{Issue: &is}, model.ModeSDD, graceNow, 30*time.Minute)

	if got.Issue.Result.Situation != model.SituationOther {
		t.Errorf("Issue.Result.Situation = %q, want other", got.Issue.Result.Situation)
	}
	if got.Result.Situation != model.SituationOther || got.Result.Tab != model.TabNow {
		t.Errorf("Card.Result = %+v, want other / 今やる（人待ちの取りこぼしは隠さない）", got.Result)
	}
}

func TestCardSettlingDoesNotHideOtherSituations(t *testing.T) {
	is := issue(108, model.LabelStagePropose, model.LabelWip)
	pr := openPR(131, model.LabelPropose, model.LabelQuestion)
	pr.Comments = []model.Comment{aiComment()}

	got := Card(model.Card{Issue: &is, PRs: []model.PR{pr, otherPR(132, time.Minute)}}, model.ModeSDD, graceNow, 30*time.Minute)

	if got.Result.Situation != model.SituationA {
		t.Errorf("Card.Result.Situation = %q, want A", got.Result.Situation)
	}
	if got.PRs[1].Result.Situation != model.SituationInProgress {
		t.Errorf("PRs[1].Result.Situation = %q, want in-progress", got.PRs[1].Result.Situation)
	}
}

func TestCardSettlingOnlyAppliesToOther(t *testing.T) {
	pr := openPR(160, model.LabelDocs)
	pr.UpdatedAt = graceNow.Add(-time.Minute)

	got := Card(model.Card{PRs: []model.PR{pr}}, model.ModeSDD, graceNow, 30*time.Minute)

	if got.Result.Situation != model.SituationG {
		t.Errorf("Card.Result.Situation = %q, want G（その他以外は猶予の対象にならない）", got.Result.Situation)
	}
}

func TestCardWipIssueWithSettlingPRUsesPRSummary(t *testing.T) {
	is := issue(108, model.LabelStageApply, model.LabelWip)

	got := Card(model.Card{Issue: &is, PRs: []model.PR{otherPR(152, 5*time.Minute)}}, model.ModeSDD, graceNow, 30*time.Minute)

	if got.Result.Situation != model.SituationInProgress || got.Result.Tab != model.TabInProgress {
		t.Errorf("Card.Result = %+v, want in-progress / 進行中", got.Result)
	}
	if want := "PR #152 はどの局面にも当たらない（更新から 30m は様子見）"; got.Result.Summary != want {
		t.Errorf("Summary = %q, want %q（候補が無いので先頭 open PR の要約）", got.Result.Summary, want)
	}
}

func TestCardSettlesPRWhenNowIsBeforeUpdatedAt(t *testing.T) {
	got := Card(model.Card{PRs: []model.PR{otherPR(61, -time.Minute)}}, model.ModeSDD, graceNow, 30*time.Minute)

	if got.Result.Situation != model.SituationInProgress {
		t.Errorf("Card.Result.Situation = %q, want in-progress（now が UpdatedAt より前なら猶予内）", got.Result.Situation)
	}
}
