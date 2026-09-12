// Command loop-cli は今やるキュー画面を出す TUI。
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/gen2brain/beeep"

	"github.com/SugiKent/loop-cli/internal/claude"
	"github.com/SugiKent/loop-cli/internal/config"
	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/onboarding"
	"github.com/SugiKent/loop-cli/internal/snapshot"
	"github.com/SugiKent/loop-cli/internal/ui"
	"github.com/SugiKent/loop-cli/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// usage はサブコマンドの一覧。未知のサブコマンドのときに標準エラーへ出す。
const usage = `使い方:
  loop-cli            今やるキュー画面を開く
  loop-cli version    現在の版を出す
  loop-cli update     最新の版に入れ直す（go install）
  loop-cli now        今やるのカードを JSON で出す
`

// updater は版の確認と入れ直し。テストは version.Client の代わりにスタブを渡す。
type updater interface {
	Current() (module, current string, local bool)
	Latest(ctx context.Context, module string) (string, error)
	Install(ctx context.Context, module string, stdout, stderr io.Writer) error
}

// run は第 1 引数でサブコマンドに振り分ける。引数なしは TUI を起動する。
// version と update は設定ファイルを読まず gh も呼ばない。now は両方を使う。
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		if err := runTUI(); err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	switch args[0] {
	case "version":
		_, current, _ := version.NewClient().Current()
		_, _ = fmt.Fprintf(stdout, "loop-cli %s\n", current)
		return 0
	case "update":
		return runUpdate(context.Background(), version.NewClient(), stdout, stderr)
	case "now":
		// now はフラグを 1 つも持たない。余分な引数は黙って捨てない（design.md D8）。
		if len(args) > 1 {
			_, _ = fmt.Fprintf(stderr, "now は引数を取りません: %s\n", strings.Join(args[1:], " "))
			return 1
		}
		client := gh.NewClient()
		deps := nowDeps{
			configPath: config.DefaultPath,
			check:      client.Check,
			fetch: func(ctx context.Context, repos []string, now time.Time, grace time.Duration) (*fetch.Result, error) {
				return fetch.Fetch(ctx, client, repos, now, grace)
			},
		}
		return runNow(context.Background(), deps, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		_, _ = fmt.Fprint(stderr, usage)
		return 1
	}
}

// runUpdate は最新版を調べ、現在の版と違えば go install で入れ直す。
func runUpdate(ctx context.Context, up updater, stdout, stderr io.Writer) int {
	module, current, local := up.Current()
	latest, err := up.Latest(ctx, module)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			_, _ = fmt.Fprintln(stderr, "go が見つかりません")
			_, _ = fmt.Fprintln(stderr, "https://go.dev/dl/ から Go をインストールしてください")
			return 1
		}
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	// 手元 build の版は公開された版と比べられない。明示的に打たれた以上は入れ直す。
	if !local && current == latest {
		_, _ = fmt.Fprintf(stdout, "最新版です: %s\n", latest)
		return 0
	}
	if err := up.Install(ctx, module, stdout, stderr); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "更新しました: %s → %s\n", current, latest)
	return 0
}

// updateChecker は起動時の更新確認を作る。手元 build は調べず、
// 失敗は「新しい版は無い」と同じに扱う（ネットワークが無い場所で画面にエラーを常駐させないため）。
func updateChecker(up updater) ui.UpdateChecker {
	return func(ctx context.Context) bool {
		module, current, local := up.Current()
		if local {
			return false
		}
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		latest, err := up.Latest(ctx, module)
		if err != nil {
			return false
		}
		return latest != current
	}
}

func runTUI() error {
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

	otherGrace := time.Duration(cfg.OtherGraceMin) * time.Minute
	var fetcher ui.Fetcher = func(ctx context.Context) (*fetch.Result, error) {
		return fetch.Fetch(ctx, client, repos, time.Now(), otherGrace)
	}
	claudeClient := claude.NewClient()
	opts := ui.Options{
		RefreshInterval:  time.Duration(cfg.RefreshIntervalSec) * time.Second,
		CheckUpdate:      updateChecker(version.NewClient()),
		MergeMethods:     mergeMethods(cfg),
		ClaudeConfigDirs: claudeConfigDirs(cfg),
		SessionLog:       claudeClient.RunLog,
	}
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

// mergeMethods は リポジトリ名 -> merge 方式 の対応表を作る（s14 merge-pr）。
// internal/ui は internal/config を import しないので、対応表はここで作って Options に渡す。
func mergeMethods(cfg *config.Config) map[string]string {
	out := make(map[string]string, len(cfg.Repos))
	for _, r := range cfg.Repos {
		out[r.Name] = string(r.MergeMethod)
	}
	return out
}

// claudeConfigDirs は リポジトリ名 -> Claude のプロファイルのパス の対応表を作る
// （s31 session-pane）。設定に書いていないリポジトリのセッションは取得しない。
func claudeConfigDirs(cfg *config.Config) map[string]string {
	out := make(map[string]string, len(cfg.Repos))
	for _, r := range cfg.Repos {
		out[r.Name] = r.ClaudeConfigDir
	}
	return out
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
// isTerminal が false なら form は呼ばれない（now はそれを当てにして nil を渡す）。
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
