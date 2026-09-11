package claude

import (
	"regexp"
	"strings"
	"time"
)

// Entry は get_run_log の 1 行。Tool は種類が tool_use のときだけ入る。
type Entry struct {
	At   time.Time
	Kind string
	Tool string
	Text string
}

// logLineRe は `[<RFC3339 の時刻>] <種類>: <本文>`。
// 種類は env[info] のように括弧を含むので、コロンまでをまとめて取る。
var logLineRe = regexp.MustCompile(`^\[([^\]]+)\] ([^:]+): ?(.*)$`)

// ParseLog は get_run_log の応答の本文を Entry の並びに読む。並びは応答のまま（新しい順）。
// 先頭の JSON のヘッダ行と、この形に合わない行は捨てる。
func ParseLog(body string) []Entry {
	var out []Entry
	for _, line := range strings.Split(body, "\n") {
		m := logLineRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		at, err := time.Parse(time.RFC3339, m[1])
		if err != nil {
			continue
		}
		kind, tool, _ := strings.Cut(m[2], " ")
		out = append(out, Entry{At: at, Kind: kind, Tool: tool, Text: m[3]})
	}
	return out
}

// finalAnswerSep は result の行の本文が `<要約> — <最終回答>` に分かれる区切り。
const finalAnswerSep = " — "

// FinalAnswer は result の行から最終回答を取り出す。result の行が無いか、
// 本文が区切りを持たないときは false を返す。
func FinalAnswer(entries []Entry) (string, bool) {
	for _, e := range entries {
		if e.Kind != "result" {
			continue
		}
		if _, answer, ok := strings.Cut(e.Text, finalAnswerSep); ok {
			return strings.TrimSpace(answer), true
		}
	}
	return "", false
}
