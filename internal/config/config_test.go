package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SugiKent/sugi-loop/internal/config"
)

// writeConfig は一時ディレクトリに config.yml を書き、そのパスを返す。
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("設定ファイルを書けなかった: %v", err)
	}
	return path
}

func TestLoadMVPExample(t *testing.T) {
	t.Setenv("EDITOR", "nvim")
	path := writeConfig(t, `repos:
  - org/app
  - org/web
refresh_interval_sec: 120
merge_method: squash
editor: $EDITOR
notify: true
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load が失敗した: %v", err)
	}
	want := []config.Repo{
		{Name: "org/app", MergeMethod: config.MergeSquash},
		{Name: "org/web", MergeMethod: config.MergeSquash},
	}
	if len(cfg.Repos) != len(want) {
		t.Fatalf("Repos の件数が違う: %v", cfg.Repos)
	}
	for i, w := range want {
		if cfg.Repos[i] != w {
			t.Errorf("Repos[%d] = %v, want %v", i, cfg.Repos[i], w)
		}
	}
	if cfg.RefreshIntervalSec != 120 {
		t.Errorf("RefreshIntervalSec = %d, want 120", cfg.RefreshIntervalSec)
	}
	if cfg.MergeMethod != config.MergeSquash {
		t.Errorf("MergeMethod = %q, want squash", cfg.MergeMethod)
	}
	if !cfg.Notify {
		t.Error("Notify = false, want true")
	}
	if cfg.Editor != "nvim" {
		t.Errorf("Editor = %q, want nvim", cfg.Editor)
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("EDITOR", "nvim")
	cfg, err := config.Load(writeConfig(t, "repos:\n  - org/app\n"))
	if err != nil {
		t.Fatalf("Load が失敗した: %v", err)
	}
	if cfg.RefreshIntervalSec != 120 {
		t.Errorf("RefreshIntervalSec = %d, want 120", cfg.RefreshIntervalSec)
	}
	if cfg.MergeMethod != config.MergeSquash {
		t.Errorf("MergeMethod = %q, want squash", cfg.MergeMethod)
	}
	if !cfg.Notify {
		t.Error("Notify = false, want true")
	}
	if cfg.Editor != "nvim" {
		t.Errorf("Editor = %q, want nvim", cfg.Editor)
	}
}

func TestLoadNotifyFalseIsKept(t *testing.T) {
	cfg, err := config.Load(writeConfig(t, "repos:\n  - org/app\nnotify: false\n"))
	if err != nil {
		t.Fatalf("Load が失敗した: %v", err)
	}
	if cfg.Notify {
		t.Error("Notify = true, want false")
	}
}

func TestLoadExplicitValuesOverrideDefaults(t *testing.T) {
	cfg, err := config.Load(writeConfig(t, `repos:
  - org/app
refresh_interval_sec: 30
merge_method: rebase
editor: vim
`))
	if err != nil {
		t.Fatalf("Load が失敗した: %v", err)
	}
	if cfg.RefreshIntervalSec != 30 {
		t.Errorf("RefreshIntervalSec = %d, want 30", cfg.RefreshIntervalSec)
	}
	if cfg.MergeMethod != config.MergeRebase {
		t.Errorf("MergeMethod = %q, want rebase", cfg.MergeMethod)
	}
	if cfg.Editor != "vim" {
		t.Errorf("Editor = %q, want vim", cfg.Editor)
	}
}

func TestLoadEmptyEditorSucceeds(t *testing.T) {
	t.Setenv("EDITOR", "")
	cfg, err := config.Load(writeConfig(t, "repos:\n  - org/app\n"))
	if err != nil {
		t.Fatalf("Load が失敗した: %v", err)
	}
	if cfg.Editor != "" {
		t.Errorf("Editor = %q, want 空文字列", cfg.Editor)
	}
}

func TestLoadPerRepoMergeMethod(t *testing.T) {
	tests := []struct {
		name string
		body string
		want map[string]config.MergeMethod
	}{
		{
			name: "上書きが無ければグローバル値",
			body: "merge_method: squash\nrepos:\n  - org/app\n",
			want: map[string]config.MergeMethod{"org/app": config.MergeSquash},
		},
		{
			name: "上書きしたリポジトリだけ別の方式",
			body: "merge_method: squash\nrepos:\n  - org/app\n  - name: org/web\n    merge_method: rebase\n",
			want: map[string]config.MergeMethod{"org/app": config.MergeSquash, "org/web": config.MergeRebase},
		},
		{
			name: "空文字列の明示は未指定扱い",
			body: "merge_method: squash\nrepos:\n  - name: org/web\n    merge_method: \"\"\n",
			want: map[string]config.MergeMethod{"org/web": config.MergeSquash},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Load(writeConfig(t, tt.body))
			if err != nil {
				t.Fatalf("Load が失敗した: %v", err)
			}
			got := map[string]config.MergeMethod{}
			for _, r := range cfg.Repos {
				got[r.Name] = r.MergeMethod
			}
			if len(got) != len(tt.want) {
				t.Fatalf("Repos = %v, want %v", got, tt.want)
			}
			for name, want := range tt.want {
				if got[name] != want {
					t.Errorf("%s の MergeMethod = %q, want %q", name, got[name], want)
				}
			}
		})
	}
}

func TestLoadInvalidConfig(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		contains []string
	}{
		{name: "構文が壊れている", body: "repos: [org/app\n", contains: nil},
		{name: "repos が空配列", body: "repos: []\n", contains: []string{"repos"}},
		{name: "repos キーが無い", body: "notify: true\n", contains: []string{"repos"}},
		{name: "空ファイル", body: "", contains: []string{"repos"}},
		{name: "スラッシュ無し", body: "repos:\n  - app\n", contains: []string{"app"}},
		{name: "スラッシュが 2 つ", body: "repos:\n  - org/app/extra\n", contains: []string{"org/app/extra"}},
		{name: "owner が空", body: "repos:\n  - /app\n", contains: []string{"/app"}},
		{name: "空白を含む", body: "repos:\n  - \"org/ app\"\n", contains: []string{"org/ app"}},
		{name: "merge_method が不正", body: "repos:\n  - org/app\nmerge_method: fast-forward\n", contains: []string{"merge_method", "fast-forward"}},
		{name: "リポジトリ別 merge_method が不正", body: "repos:\n  - name: org/web\n    merge_method: ff\n", contains: []string{"org/web", "ff"}},
		{name: "refresh_interval_sec が 0", body: "repos:\n  - org/app\nrefresh_interval_sec: 0\n", contains: []string{"refresh_interval_sec"}},
		{name: "未知のトップレベルキー", body: "repos:\n  - org/app\nrefresh_interval: 60\n", contains: []string{"refresh_interval"}},
		{name: "repos 要素の未知のキー", body: "repos:\n  - name: org/web\n    merge_methd: rebase\n", contains: []string{"merge_methd"}},
		{name: "repos 要素に name が無い", body: "repos:\n  - merge_method: rebase\n", contains: []string{"repos"}},
		{name: "repos 要素がシーケンス", body: "repos:\n  - [org/app]\n", contains: []string{"repos"}},
		{name: "token キーがある", body: "repos:\n  - org/app\ntoken: ghp_xxx\n", contains: []string{"token"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfig(t, tt.body)
			cfg, err := config.Load(path)
			if err == nil {
				t.Fatalf("エラーを期待したが Load が成功した: %+v", cfg)
			}
			if cfg != nil {
				t.Errorf("エラー時に Config が返った: %+v", cfg)
			}
			msg := err.Error()
			if !strings.Contains(msg, path) {
				t.Errorf("エラー文言にパス %q が含まれない: %s", path, msg)
			}
			for _, want := range tt.contains {
				if !strings.Contains(msg, want) {
					t.Errorf("エラー文言に %q が含まれない: %s", want, msg)
				}
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yml")
	cfg, err := config.Load(path)
	if err == nil {
		t.Fatalf("エラーを期待したが Load が成功した: %+v", cfg)
	}
	if cfg != nil {
		t.Errorf("エラー時に Config が返った: %+v", cfg)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("エラー文言にパス %q が含まれない: %s", path, err)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("HOME", "/Users/alice")
	got, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath が失敗した: %v", err)
	}
	if want := "/Users/alice/.config/sugi-loop/config.yml"; got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestDefaultPathWithoutHome(t *testing.T) {
	t.Setenv("HOME", "")
	got, err := config.DefaultPath()
	if err == nil {
		t.Fatalf("エラーを期待したが %q が返った", got)
	}
	if got != "" {
		t.Errorf("エラー時にパスが返った: %q", got)
	}
}
