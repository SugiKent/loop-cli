package classify

import (
	"fmt"
	"time"

	"github.com/SugiKent/loop-cli/internal/model"
)

// Card は Issue と open PR 群を分類し、最上位の局面を Card.Result に置いたコピーを返す。
// 1 枚の Card は 1 リポジトリ分なので、mode はカード全体で 1 つに決まる。now は PR() へ渡す。
// grace は「その他」の open PR を進行中に置く猶予で、0 なら置き換えない。
// 入力は変更しない（呼び出し側がスナップショットを保持したまま再分類できるようにするため）。
func Card(c model.Card, mode model.Mode, now time.Time, grace time.Duration) model.Card {
	out := model.Card{PRs: append([]model.PR(nil), c.PRs...)}
	if c.Issue != nil {
		is := *c.Issue
		is.Result = Issue(is, mode)
		out.Issue = &is
	}
	for i := range out.PRs {
		out.PRs[i].Canonical = false
		out.PRs[i].Result = PR(out.PRs[i], mode, now)
		if settling(out.PRs[i], mode, now, grace) {
			summary := fmt.Sprintf("PR #%d はどの局面にも当たらない（更新から %dm は様子見）", out.PRs[i].Number, int(grace.Minutes()))
			out.PRs[i].Result = result(model.SituationInProgress, summary)
		}
	}
	markCanonical(out.PRs, mode)

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

// settling は「その他」の open PR が最終更新からの猶予の中にいるかを返す。
// 猶予は sdd だけに当てる。label（ILD）は 1 issue = 1 PR を人が捌く方式で、open PR を
// 全件今やるに出すと決めてある（human-turn-classify「方式が label の入力は…」差分 7）。
// UpdatedAt がゼロ値（取得できなかった）なら、いつ触られたか分からないので猶予を当てない。
// now が UpdatedAt より前（ローカル時計の遅れ）なら差は負で、猶予内として扱う。
func settling(pr model.PR, mode model.Mode, now time.Time, grace time.Duration) bool {
	return mode != model.ModeLabel && pr.Result.Situation == model.SituationOther && grace > 0 &&
		!pr.UpdatedAt.IsZero() && now.Sub(pr.UpdatedAt) < grace
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
func markCanonical(prs []model.PR, mode model.Mode) {
	latest := map[string]int{}
	for i, pr := range prs {
		stages := model.PRStages(mode, pr.Labels)
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
