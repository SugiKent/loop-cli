package action

import (
	"fmt"
	"strings"

	"github.com/SugiKent/sugi-loop/internal/model"
)

// AnswerTemplate は最新の routine コメントの質問から `Q1: A` 形式の下書きを作る。
// 既定値は（推奨）の選択肢、無ければ先頭の選択肢、選択肢が無ければ空。
// routine のコメントが無い / 質問が 1 件も無い場合は空文字列を返す。
func AnswerTemplate(comments []model.Comment) string {
	var body string
	for i := len(comments) - 1; i >= 0; i-- {
		if comments[i].AI {
			body = comments[i].Body
			break
		}
	}

	var lines []string
	for _, q := range model.ParseQuestions(body) {
		lines = append(lines, fmt.Sprintf("Q%d: %s", q.Number, defaultLetter(q)))
	}
	return strings.Join(lines, "\n")
}

// defaultLetter は質問の既定の選択肢の記号を返す。
func defaultLetter(q model.Question) string {
	for _, o := range q.Options {
		if o.Recommended {
			return o.Letter
		}
	}
	if len(q.Options) > 0 {
		return q.Options[0].Letter
	}
	return ""
}
