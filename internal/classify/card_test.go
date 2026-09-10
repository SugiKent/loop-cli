package classify

import (
	"testing"

	"github.com/SugiKent/loop-cli/internal/model"
)

func mergedPR(number int, labels ...string) model.PR {
	return model.PR{Repo: "org/app", Number: number, State: "MERGED", Labels: labels}
}

func TestCardPicksHighestSituation(t *testing.T) {
	is := issue(108, model.LabelStagePropose, model.LabelWip)
	pr := openPR(131, model.LabelPropose, model.LabelQuestion)
	pr.Comments = []model.Comment{aiComment()}

	got := Card(model.Card{Issue: &is, PRs: []model.PR{pr}}, model.ModeSDD)

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

	got := Card(model.Card{Issue: &is, PRs: []model.PR{pr}}, model.ModeSDD)

	if got.Result.Summary != "#108 に段階ラベルが 2 つ以上ある" {
		t.Errorf("Summary = %q, want issue のもの（同点は issue 優先）", got.Result.Summary)
	}
}

func TestCardAllInProgressUsesOpenPRSummary(t *testing.T) {
	is := issue(108, model.LabelStageApply, model.LabelWip)
	pr := openPR(151, model.LabelApply)
	pr.Comments = []model.Comment{humanComment()}

	got := Card(model.Card{Issue: &is, PRs: []model.PR{pr}}, model.ModeSDD)

	if got.Result.Situation != model.SituationInProgress || got.Result.Tab != model.TabInProgress {
		t.Errorf("Card.Result = %+v, want in-progress / 進行中", got.Result)
	}
	if got.Result.Summary != "PR #151 は auto-fix が受け取り中" {
		t.Errorf("Summary = %q, want open PR のもの", got.Result.Summary)
	}
}

func TestCardWithoutOpenPRUsesIssueSummary(t *testing.T) {
	is := issue(108, model.LabelStageApply, model.LabelWip)

	got := Card(model.Card{Issue: &is, PRs: []model.PR{mergedPR(131, model.LabelPropose)}}, model.ModeSDD)

	if got.Result.Summary != "#108 は AI が作業中" {
		t.Errorf("Summary = %q, want issue のもの", got.Result.Summary)
	}
}

func TestCardWithoutIssue(t *testing.T) {
	got := Card(model.Card{PRs: []model.PR{openPR(160, model.LabelDocs)}}, model.ModeSDD)

	if got.Result.Situation != model.SituationG {
		t.Errorf("Card.Result.Situation = %q, want G", got.Result.Situation)
	}
}

func TestCardDoesNotMutateInput(t *testing.T) {
	is := issue(108, model.LabelStagePropose)
	in := model.Card{Issue: &is, PRs: []model.PR{mergedPR(131, model.LabelPropose), openPR(151, model.LabelApply)}}

	Card(in, model.ModeSDD)

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

	got := Card(model.Card{Issue: &is, PRs: []model.PR{old, latest, open}}, model.ModeSDD)

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
	}}, model.ModeSDD)

	if !got.PRs[0].Canonical || !got.PRs[1].Canonical {
		t.Errorf("段階が違えば両方 Canonical であるべき: %v / %v", got.PRs[0].Canonical, got.PRs[1].Canonical)
	}
}
