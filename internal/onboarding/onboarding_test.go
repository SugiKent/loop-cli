package onboarding_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SugiKent/sugi-loop/internal/config"
	"github.com/SugiKent/sugi-loop/internal/onboarding"
)

func TestParseReposTrimsAndDropsEmptyLines(t *testing.T) {
	got, err := onboarding.ParseRepos("org/app\n\n  org/web  \n")
	if err != nil {
		t.Fatalf("ParseRepos: %v", err)
	}
	if len(got) != 2 || got[0] != "org/app" || got[1] != "org/web" {
		t.Errorf("ParseRepos = %v, want [org/app org/web]", got)
	}
}

func TestParseReposRejectsEmpty(t *testing.T) {
	_, err := onboarding.ParseRepos("\n  \n")
	if err == nil {
		t.Fatal("空の入力がエラーにならない")
	}
	if !strings.Contains(err.Error(), "repos") {
		t.Errorf("エラーに repos が無い: %q", err)
	}
}

func TestParseReposRejectsNonOwnerName(t *testing.T) {
	for _, tc := range []struct{ text, want string }{
		{"org/app\napp\n", "app"},
		{"org/app/extra", "org/app/extra"},
		{"org/ app", "org/ app"},
	} {
		_, err := onboarding.ParseRepos(tc.text)
		if err == nil {
			t.Fatalf("ParseRepos(%q) がエラーにならない", tc.text)
		}
		if !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "owner/name") {
			t.Errorf("ParseRepos(%q) のエラー %q に %q と owner/name が無い", tc.text, err, tc.want)
		}
	}
}

func TestMarshalMatchesMVPExample(t *testing.T) {
	got, err := onboarding.Marshal(onboarding.Answers{
		Repos:       []string{"org/app", "org/web"},
		MergeMethod: config.MergeSquash,
		Notify:      true,
		Editor:      "$EDITOR",
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := "repos:\n  - org/app\n  - org/web\nrefresh_interval_sec: 120\nmerge_method: squash\neditor: $EDITOR\nnotify: true\n"
	if string(got) != want {
		t.Errorf("Marshal =\n%q\nwant\n%q", got, want)
	}
}

func TestWriteIsReadableByConfigLoad(t *testing.T) {
	t.Setenv("EDITOR", "nvim")
	path := filepath.Join(t.TempDir(), "sugi-loop", "config.yml")

	if err := onboarding.Write(path, onboarding.Answers{
		Repos:       []string{"org/app"},
		MergeMethod: config.MergeRebase,
		Notify:      false,
		Editor:      "$EDITOR",
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("書き出したファイルが無い: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Repos) != 1 || cfg.Repos[0].Name != "org/app" || cfg.Repos[0].MergeMethod != config.MergeRebase {
		t.Errorf("Repos = %v, want [{org/app rebase}]", cfg.Repos)
	}
	if cfg.MergeMethod != config.MergeRebase {
		t.Errorf("MergeMethod = %q, want rebase", cfg.MergeMethod)
	}
	if cfg.RefreshIntervalSec != 120 {
		t.Errorf("RefreshIntervalSec = %d, want 120", cfg.RefreshIntervalSec)
	}
	if cfg.Notify {
		t.Error("Notify = true, want false")
	}
	if cfg.Editor != "nvim" {
		t.Errorf("Editor = %q, want nvim", cfg.Editor)
	}
}

func TestMarshalEmptyEditorIsReadable(t *testing.T) {
	data, err := onboarding.Marshal(onboarding.Answers{
		Repos:       []string{"org/app"},
		MergeMethod: config.MergeSquash,
		Notify:      true,
		Editor:      "",
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Editor != "" {
		t.Errorf("Editor = %q, want 空", cfg.Editor)
	}
}
