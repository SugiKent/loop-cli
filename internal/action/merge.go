package action

import (
	"context"
	"errors"
	"fmt"

	"github.com/SugiKent/loop-cli/internal/classify"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// ErrBadMergeMethod は squash / merge / rebase 以外の merge 方式。
// gh pr merge に方式のフラグを渡さないと対話式になるので、境界で止める。
var ErrBadMergeMethod = errors.New("merge 方式が不正です")

// CheckMerge は merge の判断材料を拒否と警告に分けて返す。判定は pr の値だけで行い、gh を呼ばない。
// draft と open でない PR は拒否（カード詳細の PR 一覧は merged / closed も選べる）。
// human-turn-signals.md 不変条件 5 の 5 条件のうち draft だけを拒否に使い、残り 4 つは警告に緩める
// （docs 逸脱。人が内容を読んだうえで merge する判断を残すため。理由と申し送りは design.md）。
func CheckMerge(pr model.PR) (blocked []string, warnings []string) {
	if pr.IsDraft {
		blocked = append(blocked, "draft の PR です")
	}
	if pr.State != "" && pr.State != "OPEN" {
		blocked = append(blocked, fmt.Sprintf("%s の PR です", pr.State))
	}
	if model.HasLabel(pr.Labels, model.LabelQuestion) {
		warnings = append(warnings, "question ラベルが付いています")
	}
	if model.HasLabel(pr.Labels, model.LabelAIAssess) {
		warnings = append(warnings, "ai-assess:requested が付いています（AI 評価が未完了）")
	}
	if n, ok := model.ParseUndecided(pr.Body); ok && n >= 1 {
		warnings = append(warnings, fmt.Sprintf("本文 1 行目が「未確定の判断: %d 件」です", n))
	}
	if !classify.ChecksGreen(pr.MergeState) {
		warnings = append(warnings, "checks が緑ではありません")
	}
	return blocked, warnings
}

// Merge は PR を 1 回だけ merge する。ラベルもコメントも書かない（不変条件 2）。
// CheckMerge は呼ばない。拒否と警告の提示は internal/ui の確認画面の責務で、
// y を押した人の判断を最後の関門にする（design.md）。
func Merge(ctx context.Context, client gh.GHClient, repo string, number int, method string) error {
	switch method {
	case "squash", "merge", "rebase":
	default:
		return fmt.Errorf("%w: %q", ErrBadMergeMethod, method)
	}
	return client.MergePR(ctx, repo, number, method)
}
