package action

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

var (
	// ErrOtherStage は別の段階ラベルが付いた issue。付けると段階ラベルが 2 つになる（局面 F）。
	ErrOtherStage = errors.New("別の段階ラベルが付いています")
	// ErrMultipleStages は段階ラベルが 2 つ以上ある issue（局面 F）。TUI から自動修復はしない。
	ErrMultipleStages = errors.New("段階ラベルが 2 つ以上あります")
	// ErrBlocked は blocked が付いた issue。blocked の付け外しは dispatcher が行う。
	ErrBlocked = errors.New("blocked が付いています")
)

// ToggleTodo は issue の承認ラベルを 1 操作 1 ラベルで切り替える（不変条件 1）。
// 書くラベルは方式ごとの model.TodoLabel（sdd は stage:todo、label は To Do）。
// 判定は ViewIssue で読み直した現在のラベルで行う。画面のラベルは最後の取得時点のもので、
// 続けて 2 回押したときの 2 回目が「もう一度付ける」になってしまう。
// added は付けたなら true、外したなら false。
func ToggleTodo(ctx context.Context, client gh.GHClient, repo string, number int, mode model.Mode) (bool, error) {
	detail, err := client.ViewIssue(ctx, repo, number)
	if err != nil {
		return false, err
	}
	labels := make([]string, len(detail.Labels))
	for i, l := range detail.Labels {
		labels[i] = l.Name
	}

	todo := model.TodoLabel(mode)
	stages := model.IssueStages(mode, labels)
	switch {
	case len(stages) >= 2:
		return false, fmt.Errorf("%w（%s）", ErrMultipleStages, strings.Join(stages, " "))
	case model.HasLabel(labels, todo):
		// 付いているものを外す操作は常に許す（blocked と同時に付いていても取り消しをできなくしない）。
		return false, client.RemoveLabel(ctx, repo, number, todo)
	case len(stages) == 1:
		return false, fmt.Errorf("%w（%s）", ErrOtherStage, stages[0])
	case model.HasLabel(labels, model.LabelBlocked):
		return false, fmt.Errorf("%w（%s）", ErrBlocked, model.LabelBlocked)
	}
	return true, client.AddLabel(ctx, repo, number, todo)
}
