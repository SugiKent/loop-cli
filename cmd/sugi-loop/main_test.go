package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/SugiKent/loop-cli/internal/config"
	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/onboarding"
	"github.com/SugiKent/loop-cli/internal/snapshot"
	"github.com/SugiKent/loop-cli/internal/ui"
)

// stubForm は呼ばれた回数とパスを記録し、固定のエラーを返す。
type stubForm struct {
	calls int
	path  string
	err   error
}

func (s *stubForm) run(path string) error {
	s.calls++
	s.path = path
	return s.err
}

// writeConfig は一時ディレクトリに設定ファイルを書き、そのパスを返す。
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestEnsureConfigSkipsFormWhenFileExists(t *testing.T) {
	path := writeConfig(t, "repos:\n  - org/app\n")
	form := &stubForm{}

	if err := ensureConfig(path, true, form.run); err != nil {
		t.Fatalf("ensureConfig: %v", err)
	}
	if form.calls != 0 {
		t.Errorf("フォームが %d 回呼ばれた, want 0", form.calls)
	}
}

func TestEnsureConfigDoesNotOverwriteBrokenConfig(t *testing.T) {
	const body = "repos: [org/app\n"
	path := writeConfig(t, body)
	form := &stubForm{}

	if err := ensureConfig(path, true, form.run); err != nil {
		t.Fatalf("ensureConfig: %v", err)
	}
	if form.calls != 0 {
		t.Errorf("フォームが %d 回呼ばれた, want 0", form.calls)
	}

	if _, err := config.Load(path); err == nil {
		t.Fatal("壊れた設定が Load を通った")
	} else if !strings.Contains(err.Error(), path) {
		t.Errorf("Load のエラーにパスが無い: %q", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != body {
		t.Errorf("ファイルが書き換わった: %q", got)
	}
}

func TestEnsureConfigRunsFormOnTerminal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	want := errors.New("x")
	form := &stubForm{err: want}

	if err := ensureConfig(path, true, form.run); !errors.Is(err, want) {
		t.Fatalf("ensureConfig = %v, want %v", err, want)
	}
	if form.calls != 1 || form.path != path {
		t.Errorf("フォームの呼び出し = %d 回 path %q, want 1 回 path %q", form.calls, form.path, path)
	}
}

func TestEnsureConfigFailsWithoutTerminal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	form := &stubForm{}

	err := ensureConfig(path, false, form.run)
	if err == nil {
		t.Fatal("端末でないのにエラーにならない")
	}
	if form.calls != 0 {
		t.Errorf("フォームが %d 回呼ばれた, want 0", form.calls)
	}
	if !strings.Contains(err.Error(), "設定ファイルがありません") || !strings.Contains(err.Error(), path) {
		t.Errorf("エラー %q に文言かパスが無い", err)
	}
}

func TestEnsureConfigReportsAbort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	form := &stubForm{err: onboarding.ErrAborted}

	err := ensureConfig(path, true, form.run)
	if err == nil {
		t.Fatal("中止がエラーにならない")
	}
	if !strings.Contains(err.Error(), "設定の作成を中止しました") || !strings.Contains(err.Error(), path) {
		t.Errorf("エラー %q に文言かパスが無い", err)
	}
}

func TestCheckErrorGhNotFound(t *testing.T) {
	err := checkError(fmt.Errorf("gh が見つかりません: %w", exec.ErrNotFound))

	lines := strings.Split(err.Error(), "\n")
	if len(lines) != 2 {
		t.Fatalf("行数 = %d, want 2: %q", len(lines), err)
	}
	if !strings.Contains(lines[0], "gh が見つかりません") {
		t.Errorf("1 行目 %q に原因が無い", lines[0])
	}
	if !strings.Contains(lines[1], "https://cli.github.com/") {
		t.Errorf("2 行目 %q にインストール先が無い", lines[1])
	}
}

func TestCheckErrorGhAuthFailed(t *testing.T) {
	const stderr = "X Failed to log in to github.com using token (GH_TOKEN)\n"
	err := checkError(&gh.Error{Args: []string{"auth", "status"}, ExitCode: 1, Stderr: stderr})

	lines := strings.Split(err.Error(), "\n")
	if len(lines) != 2 {
		t.Fatalf("行数 = %d, want 2: %q", len(lines), err)
	}
	if !strings.Contains(lines[0], "gh の認証に失敗しました") || !strings.Contains(lines[0], strings.TrimSpace(stderr)) {
		t.Errorf("1 行目 %q に原因か stderr が無い", lines[0])
	}
	if !strings.Contains(lines[1], "gh auth login") {
		t.Errorf("2 行目 %q に次の一手が無い", lines[1])
	}
}

func TestCheckErrorOtherIsOneLine(t *testing.T) {
	orig := fmt.Errorf("gh auth status: %w", context.DeadlineExceeded)
	err := checkError(orig)

	if strings.Contains(err.Error(), "\n") {
		t.Errorf("改行が含まれる: %q", err)
	}
	if !strings.Contains(err.Error(), orig.Error()) {
		t.Errorf("%q に元のエラー %q が無い", err, orig)
	}
}

// exampleFetcher は example fixture から s07 の Fetch で作った Result を返す Fetcher。
func exampleFetcher(t *testing.T) (ui.Fetcher, *fetch.Result) {
	t.Helper()
	res, err := fetch.Fetch(context.Background(), gh.NewFake("../../internal/gh/testdata/fixtures/example"), []string{"org/app"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	return func(context.Context) (*fetch.Result, error) { return res, nil }, res
}

func TestSavingFetcherSavesOnSuccess(t *testing.T) {
	fetcher, res := exampleFetcher(t)
	path := filepath.Join(t.TempDir(), "sugi-loop", "snapshot.json")

	before := time.Now()
	got, err := savingFetcher(fetcher, path)(context.Background())
	after := time.Now()

	if err != nil {
		t.Fatalf("savingFetcher: %v", err)
	}
	if got != res {
		t.Errorf("返った Result が元のものでない: %p, want %p", got, res)
	}
	snap, err := snapshot.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(snap.Cards, res.Cards) {
		t.Error("保存された Cards が Result のものと違う")
	}
	if snap.At.Before(before) || snap.At.After(after) {
		t.Errorf("保存時刻 %v が呼び出しの前後 %v–%v の外", snap.At, before, after)
	}
}

func TestSavingFetcherDoesNotSaveOnError(t *testing.T) {
	want := errors.New("search issues: gh search issues: exit 1: rate limited")
	fetcher := func(context.Context) (*fetch.Result, error) { return nil, want }
	path := filepath.Join(t.TempDir(), "no-such", "snapshot.json")

	res, err := savingFetcher(fetcher, path)(context.Background())

	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if res != nil {
		t.Errorf("Result = %v, want nil", res)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("取得失敗でファイルが作られた: %v", err)
	}
}

func TestSavingFetcherIgnoresSaveFailure(t *testing.T) {
	fetcher, res := exampleFetcher(t)
	// ディレクトリと同じパスには書けない。
	path := t.TempDir()

	got, err := savingFetcher(fetcher, path)(context.Background())

	if err != nil {
		t.Fatalf("保存の失敗が取得の結果を変えた: %v", err)
	}
	if got != res {
		t.Errorf("返った Result が元のものでない: %p, want %p", got, res)
	}
}
