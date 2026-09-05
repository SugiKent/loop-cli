package model

import (
	"regexp"
	"strconv"
	"strings"
)

// routineMarker は dispatcher が AI のコメント先頭に置くマーカー。
const (
	routineMarker        = "<!-- routine -->"
	routineMarkerEscaped = "&lt;!-- routine --&gt;"
	riskHeading          = "## PR リスク評価"
	blockedByPrefix      = "blocked-by:"
)

var (
	undecidedRe = regexp.MustCompile(`^未確定の判断:\s*(\d+)\s*件`)
	questionRe  = regexp.MustCompile(`^##\s*Q(\d+)\.\s*(.*)$`)
	optionRe    = regexp.MustCompile(`^[-*]\s*選択肢\s*([A-Z])\s*(（推奨）|\(推奨\))?\s*[:：]\s*(.*)$`)
)

// IsAI は本文だけで routine（AI）の発言かを判定する。
// routine は利用者本人のアカウントで投稿するため author では判定できない（human-turn-signals.md）。
func IsAI(body string) bool {
	if strings.HasPrefix(body, routineMarker) || strings.HasPrefix(body, routineMarkerEscaped) {
		return true
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, riskHeading) {
			return true
		}
	}
	return false
}

// ParseUndecided は本文の（先頭の空行を除いた）1 行目の `未確定の判断: N 件` を読む。
// 2 行目以降は見ない。局面 C と s14 の merge ガードが使う。
func ParseUndecided(body string) (int, bool) {
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		m := undecidedRe.FindStringSubmatch(line)
		if m == nil {
			return 0, false
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

// LatestBlockedBy は末尾から見て `blocked-by:` 行を含む最初のコメントを返す（不変条件 8）。
// value はその行の `blocked-by:` より後ろ。author は見ない。
func LatestBlockedBy(comments []Comment) (*Comment, string, bool) {
	for i := len(comments) - 1; i >= 0; i-- {
		for _, line := range strings.Split(comments[i].Body, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, blockedByPrefix) {
				return &comments[i], strings.TrimSpace(strings.TrimPrefix(line, blockedByPrefix)), true
			}
		}
	}
	return nil, "", false
}

// Option は質問の選択肢。
type Option struct {
	Letter      string
	Text        string
	Recommended bool
}

// Question は `## Q<n>.` の見出しと、それに属する選択肢。
type Question struct {
	Number  int
	Title   string
	Options []Option
}

// ParseQuestions は mvp.md「カード詳細」の形式で質問と選択肢を読む。
// 見出しの無い本文はパースできた分だけ返す。回答テンプレートの組み立ては s10 が担当する。
func ParseQuestions(body string) []Question {
	var out []Question
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimLeft(raw, " \t")
		if m := questionRe.FindStringSubmatch(line); m != nil {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			out = append(out, Question{Number: n, Title: strings.TrimSpace(m[2])})
			continue
		}
		if len(out) == 0 {
			continue
		}
		if m := optionRe.FindStringSubmatch(line); m != nil {
			q := &out[len(out)-1]
			q.Options = append(q.Options, Option{
				Letter:      m[1],
				Text:        strings.TrimSpace(m[3]),
				Recommended: m[2] != "",
			})
		}
	}
	return out
}
