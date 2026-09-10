package action

import (
	"context"
	"errors"
	"strings"

	"github.com/SugiKent/loop-cli/internal/gh"
)

// ErrEmptyTitle は 1 行目が空の下書き。タイトルの無い issue はキューの表で見分けられない。
var ErrEmptyTitle = errors.New("タイトルが空です")

// SplitNewIssue は $EDITOR で書いた 1 枚の下書きを、最初の改行でタイトルと本文に分ける。
// 見出し記号や引用記号は解釈せず字面どおりに分ける。改行が無ければ本文は空。
func SplitNewIssue(text string) (title, body string) {
	first, rest, _ := strings.Cut(text, "\n")
	return strings.TrimSpace(first), strings.TrimSpace(rest)
}

// CreateIssue は検査を通った下書きで issue を 1 つ作り、作成された issue の URL を返す。
// ラベルも assignee も指定しない（不変条件 6「Issue 作成は承認ではない」・不変条件 2）。
// 作った issue に着手させるかは、人が続けて `t` を押して決める。
func CreateIssue(ctx context.Context, client gh.GHClient, repo string, title string, body string) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", ErrEmptyTitle
	}
	if strings.TrimSpace(body) == "" {
		return "", ErrEmptyBody
	}
	// 不変条件 7「TUI は <!-- routine --> を書かない」は本文とタイトルの両方に及ぶ。
	if HasRoutineMarker(title) || HasRoutineMarker(body) {
		return "", ErrRoutineMarker
	}
	return client.CreateIssue(ctx, repo, title, body)
}
