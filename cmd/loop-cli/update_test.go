package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/SugiKent/loop-cli/internal/version"
)

const testModule = "github.com/SugiKent/loop-cli"

// stubUpdater は go を呼ばない updater。install に渡されたモジュールパスと回数を記録する。
type stubUpdater struct {
	current       string
	local         bool
	latest        string
	latestErr     error
	installErr    error
	installs      int
	installModule string
	latests       int
}

func (s *stubUpdater) Current() (string, string, bool) { return testModule, s.current, s.local }

func (s *stubUpdater) Latest(context.Context, string) (string, error) {
	s.latests++
	return s.latest, s.latestErr
}

func (s *stubUpdater) Install(_ context.Context, module string, _, _ io.Writer) error {
	s.installs++
	s.installModule = module
	return s.installErr
}

// TestRunVersionPrintsCurrent は version が現在の版を 1 行出すことを検証する。
func TestRunVersionPrintsCurrent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	_, current, _ := version.NewClient().Current()
	if stdout.String() != "loop-cli "+current+"\n" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "loop-cli "+current+"\n")
	}
}

// TestRunUnknownCommand は未知のサブコマンドが使い方とともに失敗することを検証する。
func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"frobnicate"}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	for _, want := range []string{"unknown command: frobnicate", "version", "update", "now"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr に %q が無い: %q", want, stderr.String())
		}
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want 空", stdout.String())
	}
}

// TestRunUpdateInstallsNewVersion は新しい版があれば入れ直して結果を出すことを検証する。
func TestRunUpdateInstallsNewVersion(t *testing.T) {
	s := &stubUpdater{current: "v0.0.0-20260901000000-aaaaaaaaaaaa", latest: "v0.0.0-20260905143706-3ce8a526b4ec"}

	var stdout, stderr bytes.Buffer
	if code := runUpdate(context.Background(), s, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", code, stderr.String())
	}
	if s.installs != 1 || s.installModule != testModule {
		t.Errorf("go install が %d 回 module=%q, want 1 回 module=%q", s.installs, s.installModule, testModule)
	}
	want := "更新しました: v0.0.0-20260901000000-aaaaaaaaaaaa → v0.0.0-20260905143706-3ce8a526b4ec\n"
	if stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
}

// TestRunUpdateSkipsWhenLatest は同じ版なら install しないことを検証する。
func TestRunUpdateSkipsWhenLatest(t *testing.T) {
	s := &stubUpdater{current: "v1.2.3", latest: "v1.2.3"}

	var stdout, stderr bytes.Buffer
	if code := runUpdate(context.Background(), s, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if s.installs != 0 {
		t.Errorf("go install が %d 回, want 0", s.installs)
	}
	if stdout.String() != "最新版です: v1.2.3\n" {
		t.Errorf("stdout = %q", stdout.String())
	}
}

// TestRunUpdateInstallsWhenLocalBuild は手元 build の版が比較されず install されることを検証する。
func TestRunUpdateInstallsWhenLocalBuild(t *testing.T) {
	s := &stubUpdater{current: "v0.0.0-20260905154637-6b7c85f9cb32+dirty", local: true, latest: "v0.0.0-20260905154637-6b7c85f9cb32+dirty"}

	var stdout, stderr bytes.Buffer
	if code := runUpdate(context.Background(), s, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if s.installs != 1 {
		t.Errorf("go install が %d 回, want 1", s.installs)
	}
}

// TestRunUpdateInstallFailure は go install の失敗が標準エラーと終了コード 1 になることを検証する。
func TestRunUpdateInstallFailure(t *testing.T) {
	s := &stubUpdater{current: "v1.0.0", latest: "v1.2.3", installErr: errors.New("go install: permission denied")}

	var stdout, stderr bytes.Buffer
	if code := runUpdate(context.Background(), s, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "permission denied") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

// TestRunUpdateGoNotFound は go が無いときに 2 行の案内を出すことを検証する。
func TestRunUpdateGoNotFound(t *testing.T) {
	s := &stubUpdater{current: "v1.0.0", latestErr: exec.ErrNotFound}

	var stdout, stderr bytes.Buffer
	if code := runUpdate(context.Background(), s, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	for _, want := range []string{"go が見つかりません", "https://go.dev/dl/"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr に %q が無い: %q", want, stderr.String())
		}
	}
	if s.installs != 0 {
		t.Errorf("go install が %d 回, want 0", s.installs)
	}
}

// TestRunUpdateLatestFailure は最新版を取れない失敗がそのまま出ることを検証する。
func TestRunUpdateLatestFailure(t *testing.T) {
	s := &stubUpdater{current: "v1.0.0", latestErr: errors.New("module lookup disabled")}

	var stdout, stderr bytes.Buffer
	if code := runUpdate(context.Background(), s, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "module lookup disabled") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

// TestUpdateChecker は起動時の確認が、手元 build と失敗で「新しい版は無い」になり、
// 手元 build では最新版を調べにいかないことを検証する。
func TestUpdateChecker(t *testing.T) {
	for _, tc := range []struct {
		name       string
		stub       stubUpdater
		want       bool
		wantLatest int
	}{
		{"新しい版がある", stubUpdater{current: "v1.0.0", latest: "v1.2.3"}, true, 1},
		{"最新版", stubUpdater{current: "v1.2.3", latest: "v1.2.3"}, false, 1},
		{"手元 build は調べない", stubUpdater{current: version.Devel, local: true, latest: "v1.2.3"}, false, 0},
		{"失敗は無視する", stubUpdater{current: "v1.0.0", latestErr: errors.New("boom")}, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.stub
			if got := updateChecker(&s)(context.Background()); got != tc.want {
				t.Errorf("updateChecker = %v, want %v", got, tc.want)
			}
			if s.latests != tc.wantLatest {
				t.Errorf("最新版を %d 回調べた, want %d", s.latests, tc.wantLatest)
			}
		})
	}
}
