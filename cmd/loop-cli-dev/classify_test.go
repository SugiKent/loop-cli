package main

import (
	"strings"
	"testing"
	"time"
)

// example は issue-108 / issue-140 / PR131 を持ち、s05 の期待値は順に in-progress / E / A。
func TestClassifyFixtureExample(t *testing.T) {
	t.Chdir("../..")
	code, stdout, stderr := runCLI(t, "classify", "--fixture", "example")
	if code != 0 {
		t.Fatalf("終了コード = %d, want 0（標準エラー: %s）", code, stderr)
	}
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(lines) != 7 {
		t.Fatalf("行数 = %d, want 7:\n%s", len(lines), stdout)
	}
	wants := [][]string{
		{"[1]今やる 1"},
		{"PR131", "質問", "org/app"},
		{"[2]バックログ 1"},
		{"#140", "todo 候補"},
		{"[3]進行中 1"},
		{"#108", "進行中"},
		{"[4]異常 0"},
	}
	for i, want := range wants {
		for _, w := range want {
			if !strings.Contains(lines[i], w) {
				t.Errorf("%d 行目 = %q, want %q を含む", i+1, lines[i], w)
			}
		}
	}
}

func TestClassifyFixtureNotFound(t *testing.T) {
	t.Chdir("../..")
	code, stdout, stderr := runCLI(t, "classify", "--fixture", "nonexistent")
	if code != 1 {
		t.Errorf("終了コード = %d, want 1", code)
	}
	if want := fixturesDir + "/nonexistent"; !strings.Contains(stderr, want) {
		t.Errorf("標準エラー = %q, want %q を含む", stderr, want)
	}
	if stdout != "" {
		t.Errorf("標準出力 = %q, want 空", stdout)
	}
}

func TestClassifyOutsideRepoRoot(t *testing.T) {
	t.Chdir(t.TempDir())
	code, stdout, stderr := runCLI(t, "classify", "--fixture", "example")
	if code != 1 {
		t.Errorf("終了コード = %d, want 1", code)
	}
	if !strings.Contains(stderr, fixturesDir) {
		t.Errorf("標準エラー = %q, want %q を含む", stderr, fixturesDir)
	}
	if stdout != "" {
		t.Errorf("標準出力 = %q, want 空", stdout)
	}
}

func TestClassifyFlagErrors(t *testing.T) {
	cases := []struct {
		label string
		args  []string
		want  string
	}{
		{"fixture 無し", []string{"classify"}, "--fixture"},
		{"未知のフラグ", []string{"classify", "--fixture", "example", "--live"}, "live"},
		{"余分な位置引数", []string{"classify", "--fixture", "example", "extra"}, "extra"},
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

func TestElapsed(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		label string
		t     time.Time
		want  string
	}{
		{"12 分前", now.Add(-12 * time.Minute), "12m"},
		{"3 時間前", now.Add(-3 * time.Hour), "3h"},
		{"2 日前", now.Add(-48 * time.Hour), "2d"},
		{"未来", now.Add(time.Hour), "0m"},
	}
	for _, tc := range cases {
		if got := elapsed(now, tc.t); got != tc.want {
			t.Errorf("%s: elapsed = %q, want %q", tc.label, got, tc.want)
		}
	}
}
