// Package version は実行中のバイナリの版と、go install による入れ直しを扱う。
package version

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
)

// Devel は版を刻めなかったバイナリの版。
const Devel = "(devel)"

// Client は go サブプロセスで版を調べ、入れ直す。
type Client struct {
	run func(ctx context.Context, stdout, stderr io.Writer, args ...string) error
}

// NewClient は os/exec で go を起動する Client を返す。
func NewClient() *Client { return &Client{run: runGo} }

func runGo(ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "go", args...)
	// module の外で走らせる。vendor ディレクトリを持つプロジェクトの中では
	// go list も go install も -mod=vendor になり @latest を問い合わせられない。
	cmd.Dir = os.TempDir()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// Current は実行中のバイナリのモジュールパス・版・手元 build かどうかを返す。
func (c *Client) Current() (module, version string, local bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", Devel, true
	}
	return fromBuildInfo(info)
}

// fromBuildInfo は build info から版と手元 build かどうかを読む。
// Go は作業ツリーでの go build にも VCS 由来の擬似バージョン（末尾 +dirty）を刻むので、
// 版の文字列では go install <module>@<version> と区別できない。vcs.revision の設定は
// module cache から入れたバイナリには付かないので、それを手元 build の目印にする。
func fromBuildInfo(info *debug.BuildInfo) (module, version string, local bool) {
	version = info.Main.Version
	if version == "" {
		version = Devel
	}
	local = version == Devel
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			local = true
		}
	}
	return info.Main.Path, version, local
}

// Latest は module の最新版を go list -m -json module@latest から取る。
// 利用者の GOPROXY / GOPRIVATE をそのまま尊重するため、proxy へ直接 HTTP を投げない。
func (c *Client) Latest(ctx context.Context, module string) (string, error) {
	args := []string{"list", "-m", "-json", module + "@latest"}
	var stdout, stderr bytes.Buffer
	if err := c.run(ctx, &stdout, &stderr, args...); err != nil {
		return "", goError(args, stderr.String(), err)
	}
	var out struct{ Version string }
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil || out.Version == "" {
		return "", fmt.Errorf("go %s: 版を読み取れません: %s", strings.Join(args, " "), strings.TrimSpace(stdout.String()))
	}
	return out.Version, nil
}

// Install は module の sugi-loop を go install で入れ直す。
// go の出力は呼び出し側の Writer にそのまま流す（ダウンロードの進捗と失敗理由が見えるように）。
func (c *Client) Install(ctx context.Context, module string, stdout, stderr io.Writer) error {
	args := []string{"install", module + "/cmd/sugi-loop@latest"}
	if err := c.run(ctx, stdout, stderr, args...); err != nil {
		return fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

// goError は失敗したコマンドと標準エラーを 1 つのエラーにまとめる。
func goError(args []string, stderr string, err error) error {
	if s := strings.TrimSpace(stderr); s != "" {
		return fmt.Errorf("go %s: %w: %s", strings.Join(args, " "), err, s)
	}
	return fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
}
