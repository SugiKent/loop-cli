package classify

import (
	"testing"

	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

func aiComment() model.Comment    { return model.Comment{Body: "<!-- routine -->\n## Q1. …", AI: true} }
func humanComment() model.Comment { return model.Comment{Body: "Q1: A"} }

// mergeable は Mergeable MERGEABLE + checks 緑の PRMergeState。
func mergeable() *gh.PRMergeState {
	return &gh.PRMergeState{
		Mergeable:         "MERGEABLE",
		StatusCheckRollup: []gh.StatusCheck{{Typename: "CheckRun", Conclusion: "SUCCESS"}},
	}
}

func openPR(number int, labels ...string) model.PR {
	return model.PR{Repo: "org/app", Number: number, State: "OPEN", Labels: labels}
}

func issue(number int, labels ...string) model.Issue {
	return model.Issue{Repo: "org/app", Number: number, Labels: labels}
}

func TestEvaluationOrder(t *testing.T) {
	t.Run("進行中の除外が判定表より先", func(t *testing.T) {
		is := issue(108, model.LabelStagePropose, model.LabelWip)
		got := Issue(is)
		if got.Situation != model.SituationInProgress || got.Summary != "#108 は AI が作業中" {
			t.Errorf("Issue = %+v, want in-progress / #108 は AI が作業中", got)
		}
	})

	t.Run("詳細が nil なら A は成立しない", func(t *testing.T) {
		pr := openPR(131, model.LabelPropose, model.LabelQuestion)
		if got := PR(pr).Situation; got != model.SituationOther {
			t.Errorf("Situation = %q, want other", got)
		}
	})

	t.Run("merge 済み PR は分類しない", func(t *testing.T) {
		pr := openPR(131, model.LabelPropose, model.LabelQuestion)
		pr.State = "MERGED"
		pr.Comments = []model.Comment{aiComment()}
		if got := PR(pr); got != (model.Result{}) {
			t.Errorf("Result = %+v, want ゼロ値", got)
		}
	})
}

func TestSituationA(t *testing.T) {
	t.Run("question PR で最新コメントが AI", func(t *testing.T) {
		pr := openPR(131, model.LabelPropose, model.LabelQuestion)
		pr.Comments = []model.Comment{aiComment()}

		got := PR(pr)
		if got.Situation != model.SituationA || got.Priority != 1 || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want A / 1 / 今やる", got)
		}
		if got.Summary != "PR #131 の質問に答える" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("最新コメントが人なら A ではない", func(t *testing.T) {
		pr := openPR(131, model.LabelPropose, model.LabelQuestion)
		pr.Comments = []model.Comment{aiComment(), humanComment()}
		if got := PR(pr).Situation; got != model.SituationInProgress {
			t.Errorf("Situation = %q, want in-progress", got)
		}
	})
}

func TestSituationB(t *testing.T) {
	t.Run("question と blocked と AI の最新コメント", func(t *testing.T) {
		is := issue(108, model.LabelStagePropose, model.LabelBlocked, model.LabelQuestion)
		is.Comments = []model.Comment{{Body: "&lt;!-- routine --&gt;\nblocked-by: human", AI: true}}

		got := Issue(is)
		if got.Situation != model.SituationB || got.Priority != 2 || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want B / 2 / 今やる", got)
		}
		if got.Summary != "#108 の方針を決めてコメントする" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("段階ラベルが無くても B（E より先）", func(t *testing.T) {
		is := issue(108, model.LabelBlocked, model.LabelQuestion)
		is.Comments = []model.Comment{aiComment()}
		if got := Issue(is).Situation; got != model.SituationB {
			t.Errorf("Situation = %q, want B", got)
		}
	})
}

func TestSituationC(t *testing.T) {
	base := func() model.PR {
		pr := openPR(151, model.LabelArchive)
		pr.Body = "未確定の判断: 0 件\n\n## 概要"
		pr.MergeState = mergeable()
		return pr
	}

	t.Run("merge する PR", func(t *testing.T) {
		got := PR(base())
		if got.Situation != model.SituationC || got.Priority != 3 || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want C / 3 / 今やる", got)
		}
		if got.Summary != "PR #151 を merge する" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("draft でも C（merge 拒否は s14 のガードが持つ）", func(t *testing.T) {
		pr := base()
		pr.IsDraft = true
		if got := PR(pr).Situation; got != model.SituationC {
			t.Errorf("Situation = %q, want C", got)
		}
	})

	t.Run("未確定が 1 件以上なら C ではない", func(t *testing.T) {
		pr := base()
		pr.Body = "未確定の判断: 2 件\n…"
		if got := PR(pr).Situation; got == model.SituationC {
			t.Error("Situation = C, want C 以外")
		}
	})

	t.Run("1 行目が無ければ C ではない", func(t *testing.T) {
		pr := base()
		pr.Body = "issue #108 の提案。\n\nCloses #108"
		if got := PR(pr).Situation; got == model.SituationC {
			t.Error("Situation = C, want C 以外")
		}
	})

	t.Run("checks が失敗していれば C ではない", func(t *testing.T) {
		pr := base()
		pr.MergeState.StatusCheckRollup = append(pr.MergeState.StatusCheckRollup,
			gh.StatusCheck{Typename: "StatusContext", State: "FAILURE"})
		if ChecksGreen(pr.MergeState) {
			t.Error("ChecksGreen = true, want false")
		}
		if got := PR(pr).Situation; got == model.SituationC {
			t.Error("Situation = C, want C 以外")
		}
	})

	t.Run("mergeable が UNKNOWN なら C ではない", func(t *testing.T) {
		pr := base()
		pr.MergeState.Mergeable = "UNKNOWN"
		if got := PR(pr).Situation; got == model.SituationC {
			t.Error("Situation = C, want C 以外")
		}
	})

	t.Run("checks が無ければ緑", func(t *testing.T) {
		if !ChecksGreen(&gh.PRMergeState{Mergeable: "MERGEABLE"}) {
			t.Error("ChecksGreen = false, want true")
		}
	})

	t.Run("MergeState が nil なら緑ではない", func(t *testing.T) {
		if ChecksGreen(nil) {
			t.Error("ChecksGreen(nil) = true, want false")
		}
	})
}

func TestSituationD(t *testing.T) {
	base := func() model.PR {
		pr := openPR(151, model.LabelApply)
		pr.ReviewThreads = []gh.ReviewThread{{
			IsResolved: false,
			Comments:   []gh.ReviewComment{{Body: "<!-- routine -->\nこの分岐は残しますか"}},
		}}
		return pr
	}

	t.Run("未 resolve の thread 最終コメントが AI", func(t *testing.T) {
		got := PR(base())
		if got.Situation != model.SituationD || got.Priority != 1 || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want D / 1 / 今やる", got)
		}
		if got.Summary != "PR #151 のレビュー質問に答える" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("resolve 済みなら D ではない", func(t *testing.T) {
		pr := base()
		pr.ReviewThreads[0].IsResolved = true
		if got := PR(pr).Situation; got != model.SituationOther {
			t.Errorf("Situation = %q, want other", got)
		}
	})

	t.Run("thread 最終コメントが人なら D ではない", func(t *testing.T) {
		pr := base()
		pr.ReviewThreads[0].Comments = append(pr.ReviewThreads[0].Comments, gh.ReviewComment{Body: "残します"})
		if got := PR(pr).Situation; got == model.SituationD {
			t.Error("Situation = D, want D 以外")
		}
	})
}

func TestSituationE(t *testing.T) {
	t.Run("ラベルが無い issue", func(t *testing.T) {
		got := Issue(issue(140))
		if got.Situation != model.SituationE || got.Priority != 4 || got.Tab != model.TabBacklog {
			t.Errorf("Result = %+v, want E / 4 / バックログ", got)
		}
		if got.Summary != "#140 の着手を承認する" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("段階ラベル以外のラベルがあっても E", func(t *testing.T) {
		if got := Issue(issue(140, "bug", "enhancement")).Situation; got != model.SituationE {
			t.Errorf("Situation = %q, want E", got)
		}
	})

	t.Run("stage:todo なら E ではない", func(t *testing.T) {
		if got := Issue(issue(140, model.LabelStageTodo)).Situation; got != model.SituationInProgress {
			t.Errorf("Situation = %q, want in-progress", got)
		}
	})
}

func TestSituationF(t *testing.T) {
	t.Run("段階ラベルが 2 つの issue", func(t *testing.T) {
		got := Issue(issue(108, model.LabelStagePropose, model.LabelStageApply))
		if got.Situation != model.SituationF || got.Priority != 0 || got.Tab != model.TabAbnormal {
			t.Errorf("Result = %+v, want F / 0 / 異常", got)
		}
		if got.Summary != "#108 に段階ラベルが 2 つ以上ある" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("段階ラベルが 2 つの PR", func(t *testing.T) {
		got := PR(openPR(131, model.LabelPropose, model.LabelApply))
		if got.Situation != model.SituationF {
			t.Errorf("Situation = %q, want F", got.Situation)
		}
		if got.Summary != "PR #131 に段階ラベルが 2 つ以上ある" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("wip でも段階ラベル 2 つなら F", func(t *testing.T) {
		is := issue(108, model.LabelStagePropose, model.LabelStageApply, model.LabelWip)
		if got := Issue(is).Situation; got != model.SituationF {
			t.Errorf("Situation = %q, want F", got)
		}
	})

	t.Run("段階ラベル 2 つでも question と blocked があれば B", func(t *testing.T) {
		is := issue(108, model.LabelStagePropose, model.LabelStageApply, model.LabelBlocked, model.LabelQuestion)
		is.Comments = []model.Comment{aiComment()}
		if got := Issue(is).Situation; got != model.SituationB {
			t.Errorf("Situation = %q, want B", got)
		}
	})
}

func TestSituationG(t *testing.T) {
	t.Run("docs PR", func(t *testing.T) {
		got := PR(openPR(160, model.LabelDocs))
		if got.Situation != model.SituationG || got.Priority != 5 || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want G / 5 / 今やる", got)
		}
		if got.Summary != "docs PR #160 を merge する" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("question があれば G ではない", func(t *testing.T) {
		if got := PR(openPR(160, model.LabelDocs, model.LabelQuestion)).Situation; got != model.SituationOther {
			t.Errorf("Situation = %q, want other", got)
		}
	})
}

func TestInProgressRules(t *testing.T) {
	withComments := func(pr model.PR, cs ...model.Comment) model.PR {
		pr.Comments = cs
		return pr
	}

	t.Run("規則 1: wip の issue", func(t *testing.T) {
		got := Issue(issue(108, model.LabelStageApply, model.LabelWip))
		if got.Situation != model.SituationInProgress || got.Tab != model.TabInProgress {
			t.Errorf("Result = %+v, want in-progress / 進行中", got)
		}
		if got.Summary != "#108 は AI が作業中" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("規則 2: question 無し PR で最新コメントが人", func(t *testing.T) {
		pr := withComments(openPR(151, model.LabelApply), model.Comment{Body: "この分岐を消してください"})
		pr.Body = "未確定の判断: 0 件"
		pr.MergeState = mergeable()

		got := PR(pr)
		if got.Situation != model.SituationInProgress {
			t.Errorf("Situation = %q, want in-progress（C を満たしていても進行中が先）", got.Situation)
		}
		if got.Summary != "PR #151 は auto-fix が受け取り中" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("規則 3: question PR で最新コメントが人", func(t *testing.T) {
		pr := withComments(openPR(131, model.LabelPropose, model.LabelQuestion), aiComment(), humanComment())
		got := PR(pr)
		if got.Situation != model.SituationInProgress {
			t.Errorf("Situation = %q, want in-progress", got.Situation)
		}
		if got.Summary != "PR #131 は回答済み。worker が受け取り中" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("規則 4: 回答済みの question issue", func(t *testing.T) {
		is := issue(108, model.LabelStagePropose, model.LabelBlocked, model.LabelQuestion)
		is.Comments = []model.Comment{aiComment(), {Body: "B で進めてください"}}

		got := Issue(is)
		if got.Situation != model.SituationInProgress {
			t.Errorf("Situation = %q, want in-progress", got.Situation)
		}
		if got.Summary != "#108 は回答済み。sweep 待ち" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("規則 5: question のみの issue", func(t *testing.T) {
		is := issue(108, model.LabelQuestion)
		is.Comments = []model.Comment{aiComment()}

		got := Issue(is)
		if got.Situation != model.SituationInProgress {
			t.Errorf("Situation = %q, want in-progress（E には当たらない）", got.Situation)
		}
		if got.Summary != "#108 は question のみ。dispatcher の回収待ち" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("フォールバック: 段階ラベル付きで wip の無い issue", func(t *testing.T) {
		got := Issue(issue(108, model.LabelStagePropose))
		if got.Situation != model.SituationInProgress || got.Summary != "#108 は進行中" {
			t.Errorf("Result = %+v, want in-progress / #108 は進行中", got)
		}
	})
}

func TestOtherBucket(t *testing.T) {
	t.Run("ラベルもコメントも無い PR", func(t *testing.T) {
		pr := openPR(170)
		pr.Comments = []model.Comment{}

		got := PR(pr)
		if got.Situation != model.SituationOther || got.Priority != 6 || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want other / 6 / 今やる", got)
		}
		if got.Summary != "PR #170 はどの局面にも当たらない" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})

	t.Run("旧構成の retro PR", func(t *testing.T) {
		if got := PR(openPR(170, "retro")).Situation; got != model.SituationOther {
			t.Errorf("Situation = %q, want other", got)
		}
	})

	t.Run("question と blocked があるのにコメント未取得の issue", func(t *testing.T) {
		is := issue(108, model.LabelStagePropose, model.LabelBlocked, model.LabelQuestion)

		got := Issue(is)
		if got.Situation != model.SituationOther || got.Priority != 6 || got.Tab != model.TabNow {
			t.Errorf("Result = %+v, want other / 6 / 今やる", got)
		}
		if got.Summary != "#108 はどの局面にも当たらない" {
			t.Errorf("Summary = %q", got.Summary)
		}
	})
}
