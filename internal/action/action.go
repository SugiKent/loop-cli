// Package action は GitHub への書き込みの直前を担う。
// human-turn-signals.md「人が書き込むときの不変条件」の判定をここ 1 か所に置き、
// internal/ui は判定を持たずにこの層の関数を呼ぶ。
package action

import (
	"context"
	"errors"
	"strings"

	"github.com/SugiKent/sugi-loop/internal/gh"
)

// routine が自分のコメント先頭に置くマーカー（model.IsAI が見る形と同じ）。
const (
	routineMarker        = "<!-- routine -->"
	routineMarkerEscaped = "&lt;!-- routine --&gt;"
	blockedByPrefix      = "blocked-by:"
)

var (
	// ErrEmptyBody は中身の無い本文。投稿すると dispatcher が question を外してしまう。
	ErrEmptyBody = errors.New("本文が空です")
	// ErrRoutineMarker は不変条件 7。人の発言に routine のマーカーを書いてはならない。
	ErrRoutineMarker = errors.New("<!-- routine --> を含む本文は投稿できません")
)

// Target は書き先。IsPR が主体の種類で、不変条件 4 の書き分けはこれだけで決まる。
type Target struct {
	Repo   string
	Number int
	IsPR   bool
}

// HasRoutineMarker は本文のどこかに routine のマーカーがあるかを返す。
func HasRoutineMarker(body string) bool {
	return strings.Contains(body, routineMarker) || strings.Contains(body, routineMarkerEscaped)
}

// BlockedByLines は `blocked-by:` で始まる行を出現順に（TrimSpace 後の形で）返す。
// 判定は model.LatestBlockedBy と同じにして、dispatcher の読み方とずらさない。
func BlockedByLines(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, blockedByPrefix) {
			out = append(out, line)
		}
	}
	return out
}

// Comment は対象の種類に応じた書き先へ本文を 1 回だけ投稿する（不変条件 4）。
// ラベルは触らない（不変条件 2。question / blocked の付け外しは sweep の仕事）。
// blocked-by: 行は拒否しない。不変条件 8 は「検出して警告する」であり、
// 確認したうえで投稿する余地を残す（警告は UI が出す）。
func Comment(ctx context.Context, client gh.GHClient, t Target, body string) error {
	if strings.TrimSpace(body) == "" {
		return ErrEmptyBody
	}
	if HasRoutineMarker(body) {
		return ErrRoutineMarker
	}
	if t.IsPR {
		return client.CommentPR(ctx, t.Repo, t.Number, body)
	}
	return client.CommentIssue(ctx, t.Repo, t.Number, body)
}
