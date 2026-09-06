package classify

import (
	"github.com/SugiKent/loop-cli/internal/model"
)

// Card は Issue と open PR 群を分類し、最上位の局面を Card.Result に置いたコピーを返す。
// 入力は変更しない（呼び出し側がスナップショットを保持したまま再分類できるようにするため）。
func Card(c model.Card) model.Card {
	out := model.Card{PRs: append([]model.PR(nil), c.PRs...)}
	if c.Issue != nil {
		is := *c.Issue
		is.Result = Issue(is)
		out.Issue = &is
	}
	for i := range out.PRs {
		out.PRs[i].Canonical = false
		out.PRs[i].Result = PR(out.PRs[i])
	}
	markCanonical(out.PRs)

	best := -1
	if out.Issue != nil && isCandidate(out.Issue.Result) {
		out.Result = out.Issue.Result
		best = out.Issue.Result.Priority
	}
	for _, pr := range out.PRs {
		if isCandidate(pr.Result) && (best < 0 || pr.Result.Priority < best) {
			out.Result = pr.Result
			best = pr.Result.Priority
		}
	}
	if best < 0 {
		out.Result = result(model.SituationInProgress, fallbackSummary(out))
	}
	return out
}

// 候補は open な要素のうち進行中でないもの。MERGED / CLOSED の PR は Result がゼロ値で候補にならない。
func isCandidate(r model.Result) bool {
	return r.Situation != "" && r.Situation != model.SituationInProgress
}

// fallbackSummary は候補が無いときに、なぜ進行中なのかを画面で読めるようにする。
func fallbackSummary(c model.Card) string {
	for _, pr := range c.PRs {
		if pr.State == "OPEN" {
			return pr.Result.Summary
		}
	}
	if c.Issue != nil {
		return c.Issue.Result.Summary
	}
	return "進行中"
}

// markCanonical は同段階の MERGED PR のうち番号最大のものを正本にする。
// 人の回答後に作り直された PR が正本であり、古い PR に残った question は無視する。
func markCanonical(prs []model.PR) {
	latest := map[string]int{}
	for i, pr := range prs {
		stages := model.PRStages(pr.Labels)
		if pr.State != "MERGED" || len(stages) != 1 {
			continue
		}
		if j, ok := latest[stages[0]]; !ok || pr.Number > prs[j].Number {
			latest[stages[0]] = i
		}
	}
	for _, i := range latest {
		prs[i].Canonical = true
	}
}
