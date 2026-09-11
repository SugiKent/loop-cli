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
	unblockWhenPrefix    = "unblock-when:"
)

// relabelPrefixes は 2 回書き（`[]` を書いてから `[stage:X]` を書く）の 1 回目の前に
// routine が投稿する行の目印。sweep はこの目印で続きを書く（routine-common / routine-sweep）。
// issue-label-driven の 2 回書きは死んだ worker の再起動だけなので `restart:` しか無い。
var (
	relabelPrefixes      = []string{"release:", "restart:", "advance:"}
	relabelPrefixesLabel = []string{"restart:"}
)

var (
	undecidedRe = regexp.MustCompile(`^未確定の判断:\s*(\d+)\s*件`)
	questionRe  = regexp.MustCompile(`^##\s*Q(\d+)\.\s*(.*)$`)
	optionRe    = regexp.MustCompile(`^[-*]\s*選択肢\s*([A-Z])\s*(（推奨）|\(推奨\))?\s*[:：]\s*(.*)$`)
	closesRe    = regexp.MustCompile(`(?i)\bcloses\s+#(\d+)`)
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

// IsMidRelabel は最新の routine コメントが 2 回書きの途中を示すかを返す。
// 段階ラベルの無い issue がこれに当たるとき、sweep が続きの段階ラベルを書く途中である。
func IsMidRelabel(mode Mode, comments []Comment) bool {
	prefixes := relabelPrefixes
	if mode == ModeLabel {
		prefixes = relabelPrefixesLabel
	}
	for i := len(comments) - 1; i >= 0; i-- {
		if !comments[i].AI {
			continue
		}
		for _, line := range strings.Split(comments[i].Body, "\n") {
			line = strings.TrimSpace(line)
			for _, p := range prefixes {
				if strings.HasPrefix(line, p) {
					return true
				}
			}
		}
		return false
	}
	return false
}

// ClosesIssue は本文の最初の `Closes #<n>` の番号を返す。`Refs #<n>` は採らない。
// issue-label-driven では、これが「routine が作った PR か」の唯一の目印になる。
func ClosesIssue(body string) (int, bool) {
	m := closesRe.FindStringSubmatch(body)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

// UnblockWhen は `blocked-by: human` のコメントに書かれた解除条件を返す。
// 値は comment / docs / #m のいずれかで、人が何をすれば動き出すかを示す。
func UnblockWhen(body string) (string, bool) {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, unblockWhenPrefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, unblockWhenPrefix)), true
		}
	}
	return "", false
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

// sessionURLRe は PR 本文に書かれる Claude Code セッションの URL。
// sessionLineRe は routine コメントの `session: <ID>` 行。ID は cse_ / session_ のどちらもある。
var (
	sessionURLRe  = regexp.MustCompile(`https://claude\.ai/code/(session_[A-Za-z0-9]+)`)
	sessionLineRe = regexp.MustCompile(`^session:\s*(\S+)`)
)

// SessionURLID は本文に最初に現れる https://claude.ai/code/session_<ID> の ID を返す。
func SessionURLID(body string) (string, bool) {
	m := sessionURLRe.FindStringSubmatch(body)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// LatestSessionID は routine コメントの `session: <ID>` 行のうち最も新しいものを返す。
// 人のコメントに同じ行があっても採らない（routine のマーカーで始まるものだけを見る）。
func LatestSessionID(comments []Comment) (string, bool) {
	for i := len(comments) - 1; i >= 0; i-- {
		body := comments[i].Body
		if !strings.HasPrefix(body, routineMarker) && !strings.HasPrefix(body, routineMarkerEscaped) {
			continue
		}
		for _, line := range strings.Split(body, "\n") {
			if m := sessionLineRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				return m[1], true
			}
		}
	}
	return "", false
}

// Link は本文から取り出した URL 1 件。Text は Markdown リンクのときだけ入る。
type Link struct {
	Text string
	URL  string
}

// linkRe は Markdown リンク `[テキスト](URL)` と裸の URL。
// Markdown リンクを先に並べるので、リンクの範囲は裸の URL の走査から外れる。
var linkRe = regexp.MustCompile(`\[([^\]]*)\]\((https?://[^)\s]*)\)|https?://[^\s)>\]）］〉」]+`)

// ParseLinks は本文から URL とリンクテキストを出現順に取り出す。
// 拾うのは Markdown リンクと裸の URL だけで、`#123` のような参照記法と相対リンクは取らない。
// 重複の除去は呼び出し側が行う。
func ParseLinks(body string) []Link {
	var out []Link
	for _, m := range linkRe.FindAllStringSubmatch(body, -1) {
		if m[2] != "" {
			out = append(out, Link{Text: m[1], URL: m[2]})
			continue
		}
		// 裸の URL は末尾に付いた句読点を削る。
		out = append(out, Link{URL: strings.TrimRight(m[0], ".,:;。、")})
	}
	return out
}
