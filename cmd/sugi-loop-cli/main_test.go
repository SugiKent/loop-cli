package main

import (
	"bytes"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errBuf bytes.Buffer
	code = run(args, &out, &errBuf)
	return code, out.String(), errBuf.String()
}

func TestRunHelp(t *testing.T) {
	for _, args := range [][]string{{"help"}, {}, {"-h"}, {"--help"}} {
		code, stdout, stderr := runCLI(t, args...)
		if code != 0 {
			t.Errorf("%v: 終了コード = %d, want 0", args, code)
		}
		for _, want := range []string{"fixture capture", "--repo", "--alias"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("%v: 標準出力に %q が無い:\n%s", args, want, stdout)
			}
		}
		if stderr != "" {
			t.Errorf("%v: 標準エラー = %q, want 空", args, stderr)
		}
	}
}

func TestRunUnknownCommand(t *testing.T) {
	cases := map[string][]string{
		"frobnicate":  {"frobnicate"},
		"fixture 単独":  {"fixture"},
		"fixture foo": {"fixture", "foo"},
	}
	want := map[string]string{
		"frobnicate":  "unknown command: frobnicate",
		"fixture 単独":  "unknown command: fixture",
		"fixture foo": "unknown command: fixture",
	}
	for label, args := range cases {
		code, stdout, stderr := runCLI(t, args...)
		if code != 1 {
			t.Errorf("%s: 終了コード = %d, want 1", label, code)
		}
		if !strings.Contains(stderr, want[label]) {
			t.Errorf("%s: 標準エラー = %q, want %q を含む", label, stderr, want[label])
		}
		if !strings.Contains(stderr, "fixture capture") {
			t.Errorf("%s: 標準エラーに使い方が無い:\n%s", label, stderr)
		}
		if stdout != "" {
			t.Errorf("%s: 標準出力 = %q, want 空", label, stdout)
		}
	}
}

// フラグ検証で止まるので gh は起動しない。
func TestFixtureCaptureFlagErrors(t *testing.T) {
	cases := []struct {
		label string
		args  []string
		want  string
	}{
		{"alias 無し", []string{"fixture", "capture", "--repo", "org/app"}, "--alias"},
		{"未知のフラグ", []string{"fixture", "capture", "--repo", "org/app", "--alias", "app", "--out", "x"}, "out"},
		{"repo の形式", []string{"fixture", "capture", "--repo", "name", "--alias", "app"}, "owner/name"},
		{"alias が example", []string{"fixture", "capture", "--repo", "acme/widgets", "--alias", "example"}, "example"},
		{"alias が name を含む", []string{"fixture", "capture", "--repo", "acme/widgets", "--alias", "my-widgets"}, "widgets"},
		{"alias が owner を含む", []string{"fixture", "capture", "--repo", "Acme/widgets", "--alias", "acme-app"}, "Acme"},
	}
	for _, tc := range cases {
		code, stdout, stderr := runCLI(t, tc.args...)
		if code != 1 {
			t.Errorf("%s: 終了コード = %d, want 1", tc.label, code)
		}
		if !strings.Contains(stderr, tc.want) {
			t.Errorf("%s: 標準エラー = %q, want %q を含む", tc.label, stderr, tc.want)
		}
		if stdout != "" {
			t.Errorf("%s: 標準出力 = %q, want 空", tc.label, stdout)
		}
	}
}

// リポジトリのルート以外では gh を起動する前にエラーになる。
func TestFixtureCaptureOutsideRepoRoot(t *testing.T) {
	t.Chdir(t.TempDir())
	code, _, stderr := runCLI(t, "fixture", "capture", "--repo", "acme/widgets", "--alias", "app")
	if code != 1 {
		t.Errorf("終了コード = %d, want 1", code)
	}
	if !strings.Contains(stderr, fixturesDir) {
		t.Errorf("標準エラー = %q, want %q を含む", stderr, fixturesDir)
	}
}
