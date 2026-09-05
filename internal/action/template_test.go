package action

import (
	"context"
	"testing"

	"github.com/SugiKent/sugi-loop/internal/model"
)

// aiComment / humanComment は AI 判定を本文に依存させずに列を組み立てる。
func aiComment(body string) model.Comment    { return model.Comment{Body: body, AI: true} }
func humanComment(body string) model.Comment { return model.Comment{Body: body, AI: false} }

// exampleIssueComments は example fixture の issue-108 のコメント列。
func exampleIssueComments(t *testing.T) []model.Comment {
	t.Helper()
	detail, err := newExampleFake().ViewIssue(context.Background(), "org/app", 108)
	if err != nil {
		t.Fatalf("ViewIssue: %v", err)
	}
	out := make([]model.Comment, len(detail.Comments))
	for i, c := range detail.Comments {
		out[i] = model.CommentFrom(c)
	}
	return out
}

func TestAnswerTemplate(t *testing.T) {
	twoQuestions := "## Q1. 認証の失効を今回の範囲に含めるか\n" +
		"- 選択肢 A（推奨）: 含めない。次の issue に回す\n" +
		"- 選択肢 B: 含める\n" +
		"## Q2. ログイン画面の余白\n" +
		"- 選択肢 A: 今回は触らない\n" +
		"- 選択肢 B（推奨）: 幅だけ直す"

	for name, tc := range map[string]struct {
		comments []model.Comment
		want     string
	}{
		"推奨の選択肢が既定値になる": {
			comments: []model.Comment{aiComment(twoQuestions)},
			want:     "Q1: A\nQ2: B",
		},
		"推奨が無ければ先頭、選択肢が無ければ空": {
			comments: []model.Comment{aiComment(
				"## Q1. 方式をどうするか\n- 選択肢 A: 案 1\n- 選択肢 B: 案 2\n## Q2. 期限はいつか")},
			want: "Q1: A\nQ2: ",
		},
		"見出しの無い質問コメントからは作れない": {
			comments: []model.Comment{aiComment("<!-- routine -->\nQ1: マイグレーションを分けますか。")},
			want:     "",
		},
		"コメントが無い": {comments: nil, want: ""},
		"人のコメントだけ": {
			comments: []model.Comment{humanComment("## Q1. 人が書いた見出し\n- 選択肢 A（推奨）: x")},
			want:     "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := AnswerTemplate(tc.comments); got != tc.want {
				t.Errorf("AnswerTemplate = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestAnswerTemplateUsesLatestRoutineComment は「最新の routine のコメント」を選ぶことを検証する。
// 後ろに人のコメントがあっても routine の最新を採り、古い routine のコメントは使わない。
func TestAnswerTemplateUsesLatestRoutineComment(t *testing.T) {
	comments := append(exampleIssueComments(t),
		aiComment("## Q1. 方式をどうするか\n- 選択肢 B（推奨）: 案 2"),
		humanComment("## Q1. 人が書いた見出し\n- 選択肢 A（推奨）: x"),
	)
	if got := AnswerTemplate(comments); got != "Q1: B" {
		t.Errorf("AnswerTemplate = %q, want %q", got, "Q1: B")
	}
}
