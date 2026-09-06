package version

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"runtime/debug"
	"strings"
	"testing"
)

const module = "github.com/SugiKent/loop-cli"

// fakeGo は go を起動しない run で、渡された引数と、書き込ませる出力・返すエラーを扱う。
type fakeGo struct {
	args   []string
	stdout string
	stderr string
	err    error
}

func (f *fakeGo) client() *Client {
	return &Client{run: func(_ context.Context, stdout, stderr io.Writer, args ...string) error {
		f.args = args
		_, _ = io.WriteString(stdout, f.stdout)
		_, _ = io.WriteString(stderr, f.stderr)
		return f.err
	}}
}

// TestCurrentReadsBuildInfo は Current が実行中のバイナリの build info を読むことを検証する。
// テストバイナリは go install 経由ではないので、手元 build と判定されなければならない。
func TestCurrentReadsBuildInfo(t *testing.T) {
	mod, ver, local := NewClient().Current()
	if ver == "" {
		t.Fatal("version が空")
	}
	if mod != module {
		t.Errorf("module = %q, want %q", mod, module)
	}
	if !local {
		t.Errorf("local = false, want true（版 %q）", ver)
	}
}

// TestFromBuildInfoDetectsLocalBuild は手元 build の見分け方を検証する。
// go install で入れたバイナリには vcs.* が付かず、作業ツリーの go build には付く。
func TestFromBuildInfoDetectsLocalBuild(t *testing.T) {
	vcs := []debug.BuildSetting{{Key: "vcs", Value: "git"}, {Key: "vcs.revision", Value: "6b7c85f9cb32"}, {Key: "vcs.modified", Value: "true"}}
	for _, tc := range []struct {
		name      string
		info      debug.BuildInfo
		wantVer   string
		wantLocal bool
	}{
		{
			name:    "go install で入れたバイナリ",
			info:    debug.BuildInfo{Main: debug.Module{Path: module, Version: "v0.0.0-20260905143706-3ce8a526b4ec"}},
			wantVer: "v0.0.0-20260905143706-3ce8a526b4ec",
		},
		{
			name:      "作業ツリーの go build（版は擬似バージョン）",
			info:      debug.BuildInfo{Main: debug.Module{Path: module, Version: "v0.0.0-20260905154637-6b7c85f9cb32+dirty"}, Settings: vcs},
			wantVer:   "v0.0.0-20260905154637-6b7c85f9cb32+dirty",
			wantLocal: true,
		},
		{
			name:      "版が空",
			info:      debug.BuildInfo{Main: debug.Module{Path: module}},
			wantVer:   Devel,
			wantLocal: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mod, ver, local := fromBuildInfo(&tc.info)
			if mod != module {
				t.Errorf("module = %q, want %q", mod, module)
			}
			if ver != tc.wantVer {
				t.Errorf("version = %q, want %q", ver, tc.wantVer)
			}
			if local != tc.wantLocal {
				t.Errorf("local = %v, want %v", local, tc.wantLocal)
			}
		})
	}
}

// TestLatestReadsVersion は go list の JSON から Version を取ることを検証する。
func TestLatestReadsVersion(t *testing.T) {
	f := &fakeGo{stdout: `{"Path":"github.com/SugiKent/loop-cli","Version":"v0.0.0-20260905143706-3ce8a526b4ec","Query":"latest"}`}

	got, err := f.client().Latest(context.Background(), module)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if got != "v0.0.0-20260905143706-3ce8a526b4ec" {
		t.Errorf("Latest = %q", got)
	}
	want := []string{"list", "-m", "-json", module + "@latest"}
	if strings.Join(f.args, " ") != strings.Join(want, " ") {
		t.Errorf("args = %v, want %v", f.args, want)
	}
}

// TestLatestGoFailure は go の失敗がコマンドと標準エラーを含むエラーになることを検証する。
func TestLatestGoFailure(t *testing.T) {
	f := &fakeGo{stderr: "module lookup disabled", err: errors.New("exit status 1")}

	_, err := f.client().Latest(context.Background(), module)
	if err == nil {
		t.Fatal("エラーが返らない")
	}
	if !strings.Contains(err.Error(), "go list") || !strings.Contains(err.Error(), "module lookup disabled") {
		t.Errorf("エラーにコマンドか標準エラーが無い: %v", err)
	}
}

// TestLatestGoNotFound は go が PATH に無い失敗を errors.Is で判別できることを検証する。
func TestLatestGoNotFound(t *testing.T) {
	f := &fakeGo{err: exec.ErrNotFound}

	_, err := f.client().Latest(context.Background(), module)
	if !errors.Is(err, exec.ErrNotFound) {
		t.Errorf("errors.Is(err, exec.ErrNotFound) = false: %v", err)
	}
}

// TestLatestUnreadableOutput は JSON でない出力と Version が空の出力がエラーになることを検証する。
func TestLatestUnreadableOutput(t *testing.T) {
	for _, tc := range []struct{ name, stdout string }{
		{"JSON でない", "not json"},
		{"Version が空", `{"Path":"github.com/SugiKent/loop-cli"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeGo{stdout: tc.stdout}

			_, err := f.client().Latest(context.Background(), module)
			if err == nil {
				t.Fatal("エラーが返らない")
			}
			if !strings.Contains(err.Error(), tc.stdout) {
				t.Errorf("エラーに出力が含まれない: %v", err)
			}
		})
	}
}

// TestInstallRunsGoInstall は go install のパスと、出力が呼び出し側に流れることを検証する。
func TestInstallRunsGoInstall(t *testing.T) {
	f := &fakeGo{stdout: "downloading", stderr: "go: finding"}

	var stdout, stderr bytes.Buffer
	if err := f.client().Install(context.Background(), module, &stdout, &stderr); err != nil {
		t.Fatalf("Install: %v", err)
	}
	want := "install " + module + "/cmd/loop-cli@latest"
	if strings.Join(f.args, " ") != want {
		t.Errorf("args = %v, want %q", f.args, want)
	}
	if stdout.String() != "downloading" || stderr.String() != "go: finding" {
		t.Errorf("出力が流れていない: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

// TestInstallFailure は go install の失敗がコマンドを含むエラーになることを検証する。
func TestInstallFailure(t *testing.T) {
	f := &fakeGo{err: errors.New("exit status 1")}

	err := f.client().Install(context.Background(), module, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "go install") {
		t.Errorf("エラーにコマンドが無い: %v", err)
	}
}
