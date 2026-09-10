package action

import (
	"context"

	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// rejectedPRNote は merge されずに close された propose / apply PR が上流に与える影響。
// dispatcher と sweep はこれを「人が却下した」とみなし、issue に blocked-by: human を書き戻す。
const rejectedPRNote = "merge せずに close した PR は却下として扱われ、issue に blocked-by: human が書き戻されます"

// Close は対象の種類に応じた書き先へ 1 回だけ close を送る（不変条件 4）。
// ラベルもコメントも書かない（不変条件 2）。close の可否は判定しない
// （画面に出ている issue と PR は取得時点で必ず open。理由は design.md「拒否を持たない」）。
func Close(ctx context.Context, client gh.GHClient, t Target) error {
	if t.IsPR {
		return client.ClosePR(ctx, t.Repo, t.Number)
	}
	return client.CloseIssue(ctx, t.Repo, t.Number)
}

// CheckClose は close の前に人へ知らせる注意を返す。判定は渡された値だけで行い、gh を呼ばない。
// 止めるかどうかは internal/ui の確認画面で人が決める（CheckMerge の警告と同じ扱い）。
func CheckClose(t Target, labels []string) []string {
	if !t.IsPR {
		return nil
	}
	if model.HasLabel(labels, model.LabelPropose) || model.HasLabel(labels, model.LabelApply) {
		return []string{rejectedPRNote}
	}
	return nil
}
