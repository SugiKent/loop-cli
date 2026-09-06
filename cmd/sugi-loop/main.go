// Command sugi-loop は今やるキュー画面を出す TUI。
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/gen2brain/beeep"

	"github.com/SugiKent/loop-cli/internal/config"
	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/onboarding"
	"github.com/SugiKent/loop-cli/internal/snapshot"
	"github.com/SugiKent/loop-cli/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	if err := ensureConfig(path, term.IsTerminal(os.Stdin.Fd()), onboarding.Run); err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	client := gh.NewClient()
	if err := client.Check(context.Background()); err != nil {
		return checkError(err)
	}

	repos := make([]string, len(cfg.Repos))
	for i, r := range cfg.Repos {
		repos[i] = r.Name
	}

	var fetcher ui.Fetcher = func(ctx context.Context) (*fetch.Result, error) {
		return fetch.Fetch(ctx, client, repos)
	}
	opts := ui.Options{RefreshInterval: time.Duration(cfg.RefreshIntervalSec) * time.Second}
	if cfg.Notify {
		// icon は string か []byte でなければならない。空文字列でアイコンなし（s06 と同じ）。
		opts.Notify = func(title, body string) error { return beeep.Notify(title, body, "") }
	}
	// スナップショットは派生データなので、パスも読み込みも失敗したら諦めて通常起動する。
	if snapPath, err := snapshot.DefaultPath(); err == nil {
		if snap, err := snapshot.Load(snapPath); err == nil {
			opts.Snapshot = &snap
		}
		fetcher = savingFetcher(fetcher, snapPath)
	}

	// editor が空でも起動は失敗させない（a を押したときにフッタにエラーが出る）。
	m := ui.New(fetcher, client, ui.ExternalEditor(cfg.Editor), opts)
	_, err = tea.NewProgram(m).Run()
	return err
}

// savingFetcher は取得が成功するたびにスナップショットを保存する包み。
// 保存の失敗は無視する（次の起動が空の画面から始まるだけで、画面は止めない）。
func savingFetcher(fetcher ui.Fetcher, path string) ui.Fetcher {
	return func(ctx context.Context) (*fetch.Result, error) {
		res, err := fetcher(ctx)
		if err != nil {
			return nil, err
		}
		_ = snapshot.Save(path, snapshot.Snapshot{Cards: res.Cards, At: time.Now()})
		return res, nil
	}
}

// ensureConfig は設定ファイルが無いときだけ onboarding のフォームを起動する。
// 「存在しない」以外は Load に任せる（壊れた設定を上書きしないため）。
func ensureConfig(path string, isTerminal bool, form func(path string) error) error {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, fs.ErrNotExist):
		if !isTerminal {
			return fmt.Errorf("設定ファイルがありません: %s", path)
		}
		if err := form(path); err != nil {
			if errors.Is(err, onboarding.ErrAborted) {
				return fmt.Errorf("設定の作成を中止しました（%s は書いていません）", path)
			}
			return err
		}
		return nil
	default:
		return fmt.Errorf("config %s: %w", path, err)
	}
}

// checkError は s03 の Check の失敗を、原因と次の一手の 2 行に写す。
func checkError(err error) error {
	if errors.Is(err, exec.ErrNotFound) {
		return errors.New("gh が見つかりません\nhttps://cli.github.com/ から GitHub CLI をインストールしてください")
	}
	var ghErr *gh.Error
	if errors.As(err, &ghErr) {
		return fmt.Errorf("gh の認証に失敗しました: %s\ngh auth login を実行してください", strings.TrimSpace(ghErr.Stderr))
	}
	return err
}
