package claude

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// recorder は Client の run を差し替え、受け取った引数と CLAUDE_CONFIG_DIR を記録する。
type recorder struct {
	dirs []string
	args [][]string
	out  []byte
	// block が true なら ctx が切れるまで返らない。
	block bool
}

func (r *recorder) run(ctx context.Context, configDir string, args []string) ([]byte, error) {
	r.dirs = append(r.dirs, configDir)
	r.args = append(r.args, args)
	if r.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return r.out, nil
}

// fixture は testdata の 1 件を読む。
func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return b
}

func TestRunLogArgsAndProfile(t *testing.T) {
	rec := &recorder{out: fixture(t, "get_run_log_200.jsonl")}
	c := &Client{run: rec.run}

	if _, err := c.RunLog(t.Context(), "/home/alice/.claude-personal", "session_01ABC"); err != nil {
		t.Fatalf("RunLog: %v", err)
	}
	if len(rec.args) != 1 {
		t.Fatalf("実行回数 = %d, want 1", len(rec.args))
	}
	args := rec.args[0]
	for _, want := range []string{
		"-p", "--model", "haiku", "--tools", "RemoteTrigger", "--allowedTools",
		"--strict-mcp-config", "--disable-slash-commands",
		"--max-budget-usd", "0.05", "--output-format", "stream-json", "--verbose",
	} {
		if !slices.Contains(args, want) {
			t.Errorf("引数に %q が無い: %v", want, args)
		}
	}
	prompt := args[slices.Index(args, "-p")+1]
	if !strings.Contains(prompt, "action=get_run_log") || !strings.Contains(prompt, "session_id=session_01ABC") {
		t.Errorf("プロンプト = %q", prompt)
	}
	if rec.dirs[0] != "/home/alice/.claude-personal" {
		t.Errorf("CLAUDE_CONFIG_DIR = %q, want /home/alice/.claude-personal", rec.dirs[0])
	}
}

func TestRunLogCanceledCtx(t *testing.T) {
	rec := &recorder{block: true}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := (&Client{run: rec.run}).RunLog(ctx, "/profile", "session_01ABC")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestRunLogReturnsEntries(t *testing.T) {
	rec := &recorder{out: fixture(t, "get_run_log_200.jsonl")}
	entries, err := (&Client{run: rec.run}).RunLog(t.Context(), "/profile", "session_01ABC")
	if err != nil {
		t.Fatalf("RunLog: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("Entry が 0 件")
	}
	want := Entry{At: time.Date(2026, 9, 10, 12, 3, 0, 0, time.UTC), Kind: "tool_use", Tool: "Bash", Text: "git push -u origin HEAD"}
	if !entries[0].At.Equal(want.At) || entries[0].Kind != want.Kind || entries[0].Tool != want.Tool || entries[0].Text != want.Text {
		t.Errorf("entries[0] = %+v, want %+v", entries[0], want)
	}
}

func TestRunLogWithoutClaudeInPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := NewClient().RunLog(t.Context(), "/profile", "session_01ABC")
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("err = %v, want exec.ErrNotFound", err)
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("エラー文字列に claude が無い: %v", err)
	}
}
