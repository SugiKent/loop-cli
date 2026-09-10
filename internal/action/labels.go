package action

import (
	"context"
	"slices"

	"github.com/SugiKent/loop-cli/internal/gh"
)

// SetLabels は対象のラベルを want（送信後にこうなっていてほしいラベル名の集合）に 1 回の書き込みで
// 合わせ、付けた名前と外した名前を返す。ラベル名による拒否は行わない（人が一覧から明示的に
// 選んだラベルはそのまま書く。stage:todo 固有の拒否は ToggleTodo が持つ）。
//
// 差分は画面のラベルではなく、書き込みの直前に読み直した現在のラベルから作る。画面のラベルは
// 最後の取得の時点のもので、その間に他の経路で変わっていると差分がずれる（ToggleTodo と同じ理由）。
func SetLabels(ctx context.Context, client gh.GHClient, t Target, want []string) (add, remove []string, err error) {
	current, err := currentLabels(ctx, client, t)
	if err != nil {
		return nil, nil, err
	}

	add, remove = labelDiff(current, want)
	if len(add) == 0 && len(remove) == 0 {
		return nil, nil, nil
	}
	if t.IsPR {
		return add, remove, client.EditPRLabels(ctx, t.Repo, t.Number, add, remove)
	}
	return add, remove, client.EditIssueLabels(ctx, t.Repo, t.Number, add, remove)
}

// currentLabels は対象に今付いているラベル名を読み直す（不変条件 4 の書き分けは Target だけで決まる）。
func currentLabels(ctx context.Context, client gh.GHClient, t Target) ([]string, error) {
	var labels []gh.Label
	if t.IsPR {
		detail, err := client.ViewPR(ctx, t.Repo, t.Number)
		if err != nil {
			return nil, err
		}
		labels = detail.Labels
	} else {
		detail, err := client.ViewIssue(ctx, t.Repo, t.Number)
		if err != nil {
			return nil, err
		}
		labels = detail.Labels
	}
	names := make([]string, len(labels))
	for i, l := range labels {
		names[i] = l.Name
	}
	return names, nil
}

// labelDiff は付ける名前と外す名前を名前の昇順で返す（gh に渡す引数を決定的にするため）。
func labelDiff(current, want []string) (add, remove []string) {
	for _, name := range want {
		if !slices.Contains(current, name) {
			add = append(add, name)
		}
	}
	for _, name := range current {
		if !slices.Contains(want, name) {
			remove = append(remove, name)
		}
	}
	slices.Sort(add)
	slices.Sort(remove)
	return add, remove
}
