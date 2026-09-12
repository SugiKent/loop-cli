// Package classify は human-turn-signals.md の判定表を純粋関数で実装する。
// I/O を行わず、入力も変更しない。
package classify

import (
	"fmt"
	"slices"
	"time"

	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// StaleAfter は、AI が次に動く前提で進行中にした PR（規則 2 / 3 / 7）を人に戻すまでの時間。
// sweep が「最新コメントが人のまま 3 時間 routine の返信が無い PR」を止まったとみなすのに合わせる。
const StaleAfter = 3 * time.Hour

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
// mode はそのリポジトリの運用方式（設定が正本。ゼロ値は sdd）。
func Issue(is model.Issue, mode model.Mode) model.Result {
	stages := model.IssueStages(mode, is.Labels)
	question := model.HasLabel(is.Labels, model.LabelQuestion)
	blocked := model.HasLabel(is.Labels, model.LabelBlocked)
	aiLatest, hasComments := latestIsAI(is.Comments)

	// 規則 1: 作業中の段階ラベルが 1 件だけ。段階が 2 件以上なら当たらず F になる。
	// label は In Progress が段階と作業中の印を兼ねるので、question を除かないと行 B に届かない。
	if isWorking(mode, stages, is.Labels, question) {
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

	// 規則 6: 2 回書きの途中（段階ラベルが無く、最新の routine コメントが release: / restart: / advance:）。
	// sweep が続きの段階ラベルを書くので、人に stage:todo を付けさせない。
	if len(stages) == 0 && !blocked && model.IsMidRelabel(mode, is.Comments) {
		return result(model.SituationInProgress, fmt.Sprintf("#%d は段階ラベルの書き直し中。sweep 待ち", is.Number))
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

// isWorking は規則 1（AI が作業中）に当たるかを返す。
func isWorking(mode model.Mode, stages []string, labels []string, question bool) bool {
	if len(stages) != 1 {
		return false
	}
	if mode == model.ModeLabel {
		return stages[0] == model.LabelInProgress && !question
	}
	return stages[0] != model.LabelStageTodo && model.HasLabel(labels, model.LabelWip)
}

// PR は open PR 1 件の局面を返す。OPEN 以外はゼロ値の Result を返す（分類対象は open だけ）。
// mode はそのリポジトリの運用方式（設定が正本。ゼロ値は sdd）。now は規則 2 / 3 / 7 の時間切れの基準。
func PR(pr model.PR, mode model.Mode, now time.Time) model.Result {
	if pr.State != "OPEN" {
		return model.Result{}
	}
	stages := model.PRStages(mode, pr.Labels)
	question := model.HasLabel(pr.Labels, model.LabelQuestion)
	aiLatest, hasComments := latestIsAI(pr.Comments)
	// 規則 2 / 3 / 7 は AI が次に動く前提なので、StaleAfter 動かない PR は進行中に留めない。
	fresh := now.Sub(pr.UpdatedAt) < StaleAfter
	assess := model.HasLabel(pr.Labels, model.LabelAIAssess)

	// worker は反映し終えたことを本文 1 行目と question を外すことで表し、そのたびにコメントを返すとは限らない。
	// question が付いているあいだは反映を終えていないので、規則 3 には適用しない。
	reflected := !question && hasComments && isReflected(pr)

	// 規則 2 / 3: 最新コメントが人。question の有無で文言だけ変える。
	// label では適用しない（1 issue = 1 PR を人が捌く方式なので、キューから外すと人の出番が見えなくなる）。
	// ai-assess:requested は worker が人の回答を反映し終えてから付けるので、付いていれば規則 7 に任せる。
	if mode != model.ModeLabel && !assess && hasComments && !aiLatest && !reflected {
		if !fresh {
			// 判定表に流すと未反映の依頼が merge 候補に見えるので、その他に出す。
			return result(model.SituationOther, fmt.Sprintf("PR #%d は人のコメントに AI が応答していない", pr.Number))
		}
		if question {
			return result(model.SituationInProgress, fmt.Sprintf("PR #%d は回答済み。worker が受け取り中", pr.Number))
		}
		return result(model.SituationInProgress, fmt.Sprintf("PR #%d は auto-fix が受け取り中", pr.Number))
	}

	// 行 A: sdd は最新コメントが AI のものだけ（人が答えたものは規則 3 が進行中にする）。
	// label は規則 3 を適用しないので、人が答えた question の PR も質問のまま今やるに残す。
	if question && (mode == model.ModeLabel || aiLatest) {
		return result(model.SituationA, fmt.Sprintf("PR #%d の質問に答える", pr.Number))
	}
	// 規則 7: AI リスク評価が走っている最中。assess がラベルを外すまで merge 待ちにしない。
	// 行 A の後に置くのは、質問が残っている PR では人の番が先だから。label にこのラベルは無い。
	if mode != model.ModeLabel && fresh && assess {
		return result(model.SituationInProgress, fmt.Sprintf("PR #%d は AI 評価待ち", pr.Number))
	}
	// label の行 C には本文 1 行目のゲートが無いので、先に C を見るとレビュー質問が緑の PR に埋もれる。
	if mode == model.ModeLabel && isD(pr, mode) {
		return result(model.SituationD, fmt.Sprintf("PR #%d のレビュー質問に答える", pr.Number))
	}
	if isC(pr, mode, stages, question) {
		return result(model.SituationC, fmt.Sprintf("PR #%d を merge する", pr.Number))
	}
	if mode != model.ModeLabel && isD(pr, mode) {
		return result(model.SituationD, fmt.Sprintf("PR #%d のレビュー質問に答える", pr.Number))
	}
	if len(stages) >= 2 {
		return result(model.SituationF, fmt.Sprintf("PR #%d に段階ラベルが 2 つ以上ある", pr.Number))
	}
	// 行 G: label に docs ラベルは無い。
	if mode != model.ModeLabel && model.HasLabel(pr.Labels, model.LabelDocs) && !question {
		return result(model.SituationG, fmt.Sprintf("docs PR #%d を merge する", pr.Number))
	}
	return result(model.SituationOther, fmt.Sprintf("PR #%d はどの局面にも当たらない", pr.Number))
}

// isReflected は worker が人の回答を反映し終えた印があるかを返す。印は本文 1 行目が
// `未確定の判断: 0 件` であることと、人の最新コメントより後に PR 自身が動いていること。
// 1 行目が `未確定の判断:` でない PR（外部から来た PR、docs PR）は反映し終えたと宣言していない。
// 呼び出し側が Comments が空でないことを保証する。
func isReflected(pr model.PR) bool {
	if n, ok := model.ParseUndecided(pr.Body); !ok || n != 0 {
		return false
	}
	return pr.UpdatedAt.After(pr.Comments[len(pr.Comments)-1].CreatedAt)
}

// isC は行 C（question 無し・mergeable・checks 緑）を判定する。対象の絞り込みは方式で分かれ、
// sdd は段階ラベルと未確定 0 件を見る。label は絞り込まない（open PR が全件対象）。
// IsDraft は見ない。draft の merge 拒否は s14 の merge ガードが持つ。
func isC(pr model.PR, mode model.Mode, stages []string, question bool) bool {
	if question {
		return false
	}
	if mode != model.ModeLabel {
		if len(stages) == 0 {
			return false
		}
		if n, ok := model.ParseUndecided(pr.Body); !ok || n != 0 {
			return false
		}
	}
	if pr.MergeState == nil || pr.MergeState.Mergeable != "MERGEABLE" {
		return false
	}
	return ChecksGreen(pr.MergeState)
}

// isD は行 D（未 resolve の review thread があり、thread 最終コメントが AI）を判定する。
// 対象は sdd が apply PR、label は open PR の全件。
func isD(pr model.PR, mode model.Mode) bool {
	if mode != model.ModeLabel && !model.HasLabel(pr.Labels, model.LabelApply) {
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
