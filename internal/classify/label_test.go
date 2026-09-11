package classify

import (
	"testing"

	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// labelIssue は org/board（label 方式）の issue。
func labelIssue(number int, labels ...string) model.Issue {
	return model.Issue{Repo: "org/board", Number: number, Labels: labels}
}

// closesPR は Closes #12 を持つ緑の open PR（label 方式で routine が作った PR の形）。
func closesPR(number int, labels ...string) model.PR {
	return model.PR{
		Repo:       "org/board",
		Number:     number,
		State:      "OPEN",
		Labels:     labels,
		Body:       "ラベル一覧をモーダルで出す。\n\nCloses #12",
		Comments:   []model.Comment{aiComment()},
		MergeState: mergeable(),
	}
}

func TestLabelModeIssue(t *testing.T) {
	t.Run("In Progress の issue は進行中", func(t *testing.T) {
		got := Issue(labelIssue(61, model.LabelInProgress), model.ModeLabel)
		if got.Situation != model.SituationInProgress || got.Tab != model.TabInProgress {
			t.Errorf("Result = %+v, want in-progress / 進行中", got)
		}
		if got.Summary != "#61 は AI が作業中" {
			t.Errorf("Summary = %q, want #61 は AI が作業中", got.Summary)
		}
	})

	t.Run("In Progress でも question と blocked があれば方針を決める", func(t *testing.T) {
		is := labelIssue(61, model.LabelInProgress, model.LabelBlocked, model.LabelQuestion)
		is.Comments = []model.Comment{aiComment()}

		got := Issue(is, model.ModeLabel)
		if got.Situation != model.SituationB || got.Priority != 2 || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want B / 2 / 今やる", got)
		}
	})

	t.Run("In Progress の question に人が答えた後は進行中", func(t *testing.T) {
		is := labelIssue(61, model.LabelInProgress, model.LabelBlocked, model.LabelQuestion)
		is.Comments = []model.Comment{aiComment(), humanComment()}

		got := Issue(is, model.ModeLabel)
		if got.Situation != model.SituationInProgress || got.Summary != "#61 は回答済み。sweep 待ち" {
			t.Errorf("Result = %+v, want in-progress / #61 は回答済み。sweep 待ち", got)
		}
	})

	t.Run("ラベルの無い issue はバックログに出る", func(t *testing.T) {
		got := Issue(labelIssue(62, "bug"), model.ModeLabel)
		if got.Situation != model.SituationE || got.Tab != model.TabBacklog {
			t.Errorf("Result = %+v, want E / バックログ", got)
		}
	})

	t.Run("To Do と In Progress が同時なら異常", func(t *testing.T) {
		got := Issue(labelIssue(63, model.LabelToDo, model.LabelInProgress), model.ModeLabel)
		if got.Situation != model.SituationF || got.Priority != 0 || got.Tab != model.TabAbnormal {
			t.Errorf("Result = %+v, want F / 0 / 異常", got)
		}
	})

	t.Run("sdd の段階ラベルは段階として数えない", func(t *testing.T) {
		is := labelIssue(64, model.LabelStagePropose, model.LabelStageApply, model.LabelWip)
		if got := Issue(is, model.ModeLabel).Situation; got != model.SituationE {
			t.Errorf("Situation = %q, want E", got)
		}
	})

	t.Run("To Do だけならフォールバックの進行中", func(t *testing.T) {
		got := Issue(labelIssue(65, model.LabelToDo), model.ModeLabel)
		if got.Situation != model.SituationInProgress || got.Summary != "#65 は進行中" {
			t.Errorf("Result = %+v, want in-progress / #65 は進行中", got)
		}
	})

	t.Run("restart の残骸は書き直しの途中", func(t *testing.T) {
		is := labelIssue(66)
		is.Comments = []model.Comment{{Body: "<!-- routine -->\nrestart: 1/3", AI: true}}

		got := Issue(is, model.ModeLabel)
		if got.Situation != model.SituationInProgress || got.Summary != "#66 は段階ラベルの書き直し中。sweep 待ち" {
			t.Errorf("Result = %+v, want in-progress / 書き直し中", got)
		}
	})

	t.Run("release は残骸の目印にしない", func(t *testing.T) {
		is := labelIssue(67)
		is.Comments = []model.Comment{{Body: "<!-- routine -->\nrelease: To Do", AI: true}}

		if got := Issue(is, model.ModeLabel).Situation; got != model.SituationE {
			t.Errorf("Situation = %q, want E", got)
		}
	})
}

func TestLabelModePR(t *testing.T) {
	t.Run("Closes を持つ緑の PR は 1 行目が無くても merge 行", func(t *testing.T) {
		got := PR(closesPR(61), model.ModeLabel, updatedNow)
		if got.Situation != model.SituationC || got.Priority != 3 || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want C / 3 / 今やる", got)
		}
	})

	t.Run("未 resolve の AI thread があればレビュー質問が先", func(t *testing.T) {
		pr := closesPR(61)
		pr.ReviewThreads = []gh.ReviewThread{{Comments: []gh.ReviewComment{{Body: "<!-- routine -->\nこの分岐は残しますか"}}}}

		got := PR(pr, model.ModeLabel, updatedNow)
		if got.Situation != model.SituationD || got.Priority != 1 {
			t.Errorf("Result = %+v, want D / 1", got)
		}
	})

	t.Run("紐づく issue の無い緑の PR も merge 行に出る", func(t *testing.T) {
		pr := closesPR(61)
		pr.Body = "依存を更新する"
		if got := PR(pr, model.ModeLabel, updatedNow).Situation; got != model.SituationC {
			t.Errorf("Situation = %q, want C（label の行 C は Closes #n を見ない）", got)
		}
	})

	t.Run("Closes の無い PR でも未 resolve の AI thread はレビュー質問になる", func(t *testing.T) {
		pr := closesPR(61)
		pr.Body = "依存を更新する"
		pr.ReviewThreads = []gh.ReviewThread{{Comments: []gh.ReviewComment{{Body: "<!-- routine -->\nこの分岐は残しますか"}}}}

		got := PR(pr, model.ModeLabel, updatedNow)
		if got.Situation != model.SituationD || got.Priority != 1 {
			t.Errorf("Result = %+v, want D / 1", got)
		}
	})

	t.Run("最新コメントが人の PR も今やるに出る", func(t *testing.T) {
		pr := closesPR(61)
		pr.Comments = []model.Comment{{Body: "この分岐を消してください"}}

		got := PR(pr, model.ModeLabel, updatedNow)
		if got.Situation != model.SituationC || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want C / 今やる（進行中の規則 2 を適用しない）", got)
		}
	})

	t.Run("question の PR に人が答えても質問のまま今やるに出る", func(t *testing.T) {
		pr := closesPR(62, model.LabelQuestion)
		pr.Comments = []model.Comment{aiComment(), humanComment()}

		got := PR(pr, model.ModeLabel, updatedNow)
		if got.Situation != model.SituationA || got.Priority != 1 || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want A / 1 / 今やる（進行中の規則 3 を適用しない）", got)
		}
	})

	t.Run("merge できない PR はその他に出る", func(t *testing.T) {
		pr := closesPR(61)
		pr.Body = "依存を更新する"
		pr.Comments = nil
		pr.MergeState = &gh.PRMergeState{
			Mergeable:         "MERGEABLE",
			StatusCheckRollup: []gh.StatusCheck{{Typename: "CheckRun", Conclusion: "FAILURE"}},
		}

		got := PR(pr, model.ModeLabel, updatedNow)
		if got.Situation != model.SituationOther || got.Priority != 6 || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want other / 6 / 今やる", got)
		}
	})

	t.Run("question の PR は質問が先", func(t *testing.T) {
		got := PR(closesPR(62, model.LabelQuestion), model.ModeLabel, updatedNow)
		if got.Situation != model.SituationA || got.Priority != 1 {
			t.Errorf("Result = %+v, want A / 1", got)
		}
	})

	t.Run("docs ラベルは merge 行を作らない", func(t *testing.T) {
		pr := model.PR{Repo: "org/board", Number: 63, State: "OPEN", Labels: []string{model.LabelDocs}, Body: "README を直す"}
		if got := PR(pr, model.ModeLabel, updatedNow).Situation; got != model.SituationOther {
			t.Errorf("Situation = %q, want other", got)
		}
	})

	t.Run("ai-assess:requested は見ない", func(t *testing.T) {
		if got := PR(closesPR(61, model.LabelAIAssess), model.ModeLabel, updatedNow).Situation; got != model.SituationC {
			t.Errorf("Situation = %q, want C", got)
		}
	})
}

func TestLabelModeCard(t *testing.T) {
	t.Run("カードの全要素が同じ方式で分類される", func(t *testing.T) {
		is := labelIssue(12, model.LabelInProgress)
		got := Card(model.Card{Issue: &is, PRs: []model.PR{closesPR(61)}}, model.ModeLabel, updatedNow)

		if got.Issue.Result.Situation != model.SituationInProgress {
			t.Errorf("Issue.Result = %+v, want in-progress", got.Issue.Result)
		}
		if got.PRs[0].Result.Situation != model.SituationC {
			t.Errorf("PRs[0].Result = %+v, want C", got.PRs[0].Result)
		}
		if got.Result.Situation != model.SituationC {
			t.Errorf("Card.Result = %+v, want C", got.Result)
		}
	})

	t.Run("Canonical は立たない", func(t *testing.T) {
		merged := model.PR{Repo: "org/board", Number: 61, State: "MERGED"}
		got := Card(model.Card{PRs: []model.PR{merged}}, model.ModeLabel, updatedNow)
		if got.PRs[0].Canonical {
			t.Error("label 方式で Canonical が立っている")
		}
	})
}
