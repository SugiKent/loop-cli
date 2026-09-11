// Package claude は Claude Code の `claude -p` をプロファイルごとにサブプロセスとして起動し、
// 組み込みの RemoteTrigger ツールに Claude Code Remote の API を叩かせて
// Routine のセッションのログを読む。モデルの回答文は読まず、ツールの実行結果だけを正とする。
package claude

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Client は claude サブプロセスでセッションのログを読む。
type Client struct {
	// run は claude の起動。テストは引数と CLAUDE_CONFIG_DIR を記録するスタブに差し替える。
	run func(ctx context.Context, configDir string, args []string) ([]byte, error)
}

// NewClient は os/exec で claude を起動する Client を返す。
func NewClient() *Client { return &Client{run: runClaude} }

// runLogPrompt は get_run_log を 1 回だけ呼ばせるプロンプト。
// モデルの仕事は RemoteTrigger を 1 回呼ぶことだけで、回答文は読まない。
func runLogPrompt(sessionID string) string {
	return "RemoteTrigger を action=get_run_log, session_id=" + sessionID + " で 1 回だけ呼び、「done」とだけ答えて"
}

// runLogArgs は issue #18「取得コマンド」の実測に基づく起動の引数。
// --model haiku は固定で、設定ファイルからは変えられない。
func runLogArgs(sessionID string) []string {
	return []string{
		"-p", runLogPrompt(sessionID),
		"--model", "haiku",
		"--tools", "RemoteTrigger", "--allowedTools", "RemoteTrigger",
		"--strict-mcp-config", "--setting-sources", "", "--disable-slash-commands",
		"--max-budget-usd", "0.05",
		"--output-format", "stream-json", "--verbose",
	}
}

// RunLog は configDir のプロファイルで sessionID のログを取る。
// sessionID は cse_ で始まる形も session_ で始まる形も、受け取ったままプロンプトに載せる。
func (c *Client) RunLog(ctx context.Context, configDir, sessionID string) ([]Entry, error) {
	out, err := c.run(ctx, configDir, runLogArgs(sessionID))
	if err != nil {
		return nil, err
	}
	body, err := decodeToolResult(out)
	if err != nil {
		return nil, err
	}
	return ParseLog(body), nil
}

// runClaude は claude を ctx 付きで起動し、標準出力を返す。標準入力には空を渡す
// （渡さないと claude が入力を 3 秒待つ）。再試行はしない。
func runClaude(ctx context.Context, configDir string, args []string) ([]byte, error) {
	if _, err := exec.LookPath("claude"); err != nil {
		return nil, fmt.Errorf("claude が見つかりません: %w", err)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Env = append(os.Environ(), "CLAUDE_CONFIG_DIR="+configDir)
	cmd.Stdin = strings.NewReader("")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	// claude は利用上限や未ログインを標準出力に書いて 0 以外で終わることがあるので、
	// 終了コードでは切らずに標準出力を呼び出し側（decodeToolResult）に読ませる。
	if ctx.Err() != nil {
		return nil, fmt.Errorf("claude: %w", ctx.Err())
	}
	if err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("claude: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
