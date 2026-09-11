package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// HTTPError は RemoteTrigger が返した 200 以外の応答。
// 404 はプロファイルが違うときにも返るので、呼び出し側が status で出し分ける。
type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.Status, e.Body) }

// LimitError はプロファイルが週の利用上限に達している状態。
// Resets は claude が書いた解除の時刻の文字列で、年を含まない表示用の書式なので解釈しない。
type LimitError struct{ Resets string }

func (e *LimitError) Error() string {
	return "週の利用上限に達しています（resets " + e.Resets + "）"
}

// ErrNotLoggedIn はプロファイルが未ログインの状態。
var ErrNotLoggedIn = errors.New("claude にログインしていません")

// streamLine は stream-json の 1 行のうち、tool_result を探すために見る部分だけ。
type streamLine struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type    string          `json:"type"`
			Content json.RawMessage `json:"content"`
		} `json:"content"`
	} `json:"message"`
}

// textBlock は tool_result の本文がブロックの配列で来る形。
type textBlock struct {
	Text string `json:"text"`
}

var (
	httpStatusRe = regexp.MustCompile(`^HTTP (\d{3})\b`)
	limitRe      = regexp.MustCompile(`hit your weekly limit\s*\S?\s*resets\s+(.+)$`)
)

// decodeToolResult は claude の標準出力から最初の tool_result の本文を取り出す。
// モデルが書いた文章（assistant の行）は読まない。RemoteTrigger のログには Routine が読んだ
// issue や Web ページの文章がそのまま入るので、その文章に書かれた指示に従った結果を読まないため。
// 本文は `HTTP <status>` の 1 行で始まり、200 なら 2 行目以降を、200 以外なら HTTPError を返す。
func decodeToolResult(out []byte) (string, error) {
	body, ok := findToolResult(out)
	if !ok {
		return "", launchError(out)
	}
	head, rest, _ := strings.Cut(body, "\n")
	m := httpStatusRe.FindStringSubmatch(strings.TrimSpace(head))
	if m == nil {
		return "", fmt.Errorf("tool_result の 1 行目が HTTP <status> ではありません: %q", head)
	}
	status, _ := strconv.Atoi(m[1])
	if status != 200 {
		return "", &HTTPError{Status: status, Body: strings.TrimSpace(rest)}
	}
	return rest, nil
}

// findToolResult は type が user の行の content[] から最初の tool_result の本文を返す。
// JSON として読めない行は捨てる。
func findToolResult(out []byte) (string, bool) {
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		var line streamLine
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil || line.Type != "user" {
			continue
		}
		for _, c := range line.Message.Content {
			if c.Type != "tool_result" {
				continue
			}
			return toolResultText(c.Content), true
		}
	}
	return "", false
}

// toolResultText は tool_result の content を読む。文字列とブロックの配列のどちらも来る。
func toolResultText(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []textBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		parts = append(parts, b.Text)
	}
	return strings.Join(parts, "")
}

// launchError は tool_result が 1 つも無かったときのエラー。
// 利用上限・未ログイン・それ以外の起動の失敗を呼び出し側が区別できる形で返す。
func launchError(out []byte) error {
	text := string(out)
	for _, line := range strings.Split(text, "\n") {
		if m := limitRe.FindStringSubmatch(line); m != nil {
			return &LimitError{Resets: strings.TrimSpace(m[1])}
		}
		if strings.Contains(line, "Not logged in") {
			return ErrNotLoggedIn
		}
	}
	head := strings.TrimSpace(text)
	if i := strings.Index(head, "\n"); i >= 0 {
		head = head[:i]
	}
	if head == "" {
		head = "（標準出力が空）"
	}
	return fmt.Errorf("claude の応答に tool_result がありません: %s", head)
}
