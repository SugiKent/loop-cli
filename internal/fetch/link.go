// Package fetch は設定した全リポジトリの open issue / open PR を取得し、
// 分類に必要な詳細だけを遅延取得して分類済みの Card 群にする。
package fetch

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// title の先頭が [propose] / [apply] / [archive] で、その直後に #<n> が続く形。
	titleLinkRe = regexp.MustCompile(`^\[(propose|apply|archive)\]\s*#(\d+)`)
	// 本文の Refs / Closes。単語として現れるものだけを採る（xRefs は当たらない）。
	bodyLinkRe = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9])(refs|closes)\s+#(\d+)`)
)

// LinkedIssue は PR の title / body から紐づく issue 番号を返す（D-001 の PR 側の紐づけ）。
// title の段階と番号を優先し、無ければ本文の最初の Refs / Closes を採る。
func LinkedIssue(title, body string) (int, bool) {
	if m := titleLinkRe.FindStringSubmatch(strings.TrimLeft(title, " \t")); m != nil {
		n, err := strconv.Atoi(m[2])
		if err == nil {
			return n, true
		}
	}
	if m := bodyLinkRe.FindStringSubmatch(body); m != nil {
		n, err := strconv.Atoi(m[3])
		if err == nil {
			return n, true
		}
	}
	return 0, false
}
