// Package classify は human-turn-signals.md の判定表を純粋関数で実装する。
// I/O を行わず、入力も変更しない。
package classify

import (
	"fmt"
	"slices"

	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

func result(s model.Situation, summary string) model.Result {
	return model.Result{Situation: s, Priority: s.Priority(), Tab: s.Tab(), Summary: summary}
}

// latestIsAI は最新（末尾）コメントが AI かを返す。コメントが無ければ false。
func latestIsAI(comments []model.Comment) (isAI bool, ok bool) {
	if len(comments) == 0 {
		return false, false
	}
	return comments[len(comments)-1].AI, true
}

// Issue は issue 1 件の局面を返す。評価順は「キューに入れないもの」→ 判定表 B / E / F → フォールバック。
func Issue(is model.Issue) model.Result {
	stages := model.IssueStages(is.Labels)
	question := model.HasLabel(is.Labels, model.LabelQuestion)
	blocked := model.HasLabel(is.Labels, model.LabelBlocked)
	aiLatest, hasComments := latestIsAI(is.Comments)

	// 規則 1: 段階ラベル 1 件 + wip。段階が 2 件以上なら当たらず F になる。
	if len(stages) == 1 && stages[0] != model.LabelStageTodo && model.HasLabel(is.Labels, model.LabelWip) {
		return result(model.SituationInProgress, fmt.Sprintf("#%d は AI が作業中", is.Number))
	}
	// 規則 4: question で最新コメントが人（sweep が question を外すのを待っている）。
	if question && hasComments && !aiLatest {
		return result(model.SituationInProgress, fmt.Sprintf("#%d は回答済み。sweep 待ち", is.Number))
	}
	// 規則 5: question だけで blocked が無い（dispatcher が回収する残骸）。
	if question && !blocked {
		return result(model.SituationInProgress, fmt.Sprintf("#%d は question のみ。dispatcher の回収待ち", is.Number))
	}

	if question && blocked && hasComments && aiLatest {
		return result(model.SituationB, fmt.Sprintf("#%d の方針を決めてコメントする", is.Number))
	}
	if len(stages) == 0 && !blocked {
		return result(model.SituationE, fmt.Sprintf("#%d の着手を承認する", is.Number))
	}
	if len(stages) >= 2 {
		return result(model.SituationF, fmt.Sprintf("#%d に段階ラベルが 2 つ以上ある", is.Number))
	}

	// question + blocked は本来 B か規則 4 であり、コメント未取得で進行中に落とすと人待ちが見えなくなる。
	if question && blocked {
		return result(model.SituationOther, fmt.Sprintf("#%d はどの局面にも当たらない", is.Number))
	}
	return result(model.SituationInProgress, fmt.Sprintf("#%d は進行中", is.Number))
}

// PR は open PR 1 件の局面を返す。OPEN 以外はゼロ値の Result を返す（分類対象は open だけ）。
func PR(pr model.PR) model.Result {
	if pr.State != "OPEN" {
		return model.Result{}
	}
	stages := model.PRStages(pr.Labels)
	question := model.HasLabel(pr.Labels, model.LabelQuestion)
	aiLatest, hasComments := latestIsAI(pr.Comments)

	// 規則 2 / 3: 最新コメントが人。question の有無で文言だけ変える。
	if hasComments && !aiLatest {
		if question {
			return result(model.SituationInProgress, fmt.Sprintf("PR #%d は回答済み。worker が受け取り中", pr.Number))
		}
		return result(model.SituationInProgress, fmt.Sprintf("PR #%d は auto-fix が受け取り中", pr.Number))
	}

	if question && hasComments && aiLatest {
		return result(model.SituationA, fmt.Sprintf("PR #%d の質問に答える", pr.Number))
	}
	if isC(pr, stages, question) {
		return result(model.SituationC, fmt.Sprintf("PR #%d を merge する", pr.Number))
	}
	if isD(pr) {
		return result(model.SituationD, fmt.Sprintf("PR #%d のレビュー質問に答える", pr.Number))
	}
	if len(stages) >= 2 {
		return result(model.SituationF, fmt.Sprintf("PR #%d に段階ラベルが 2 つ以上ある", pr.Number))
	}
	if model.HasLabel(pr.Labels, model.LabelDocs) && !question {
		return result(model.SituationG, fmt.Sprintf("docs PR #%d を merge する", pr.Number))
	}
	return result(model.SituationOther, fmt.Sprintf("PR #%d はどの局面にも当たらない", pr.Number))
}

// isC は行 C（段階 PR・未確定 0 件・question 無し・mergeable・checks 緑）を判定する。
// IsDraft は見ない。draft の merge 拒否は s14 の merge ガードが持つ。
func isC(pr model.PR, stages []string, question bool) bool {
	if len(stages) == 0 || question {
		return false
	}
	if n, ok := model.ParseUndecided(pr.Body); !ok || n != 0 {
		return false
	}
	if pr.MergeState == nil || pr.MergeState.Mergeable != "MERGEABLE" {
		return false
	}
	return ChecksGreen(pr.MergeState)
}

// isD は行 D（apply PR に未 resolve の review thread があり、thread 最終コメントが AI）を判定する。
func isD(pr model.PR) bool {
	if !model.HasLabel(pr.Labels, model.LabelApply) {
		return false
	}
	for _, th := range pr.ReviewThreads {
		if th.IsResolved || len(th.Comments) == 0 {
			continue
		}
		if model.IsAI(th.Comments[len(th.Comments)-1].Body) {
			return true
		}
	}
	return false
}

// ChecksGreen は statusCheckRollup が merge を妨げない状態かを返す。
// 要素 0 件は緑、nil の PRMergeState は緑ではない。s14 の merge ガードもこれを使う。
func ChecksGreen(ms *gh.PRMergeState) bool {
	if ms == nil {
		return false
	}
	for _, c := range ms.StatusCheckRollup {
		switch c.Typename {
		case "CheckRun":
			if !slices.Contains([]string{"SUCCESS", "SKIPPED", "NEUTRAL"}, c.Conclusion) {
				return false
			}
		case "StatusContext":
			if c.State != "SUCCESS" {
				return false
			}
		default:
			return false
		}
	}
	return true
}
