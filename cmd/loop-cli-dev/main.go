// Command loop-cli-dev は loop-cli の動作確認用 CLI。
package main

import (
	"fmt"
	"io"
	"os"
)

const usage = `loop-cli-dev は loop-cli の動作確認用 CLI です。

使い方:
  loop-cli-dev help
      この使い方を表示する
  loop-cli-dev fixture capture --repo owner/name --alias <alias>
      指定リポジトリの open issue / PR を採取し、伏せ字にして
      internal/gh/testdata/fixtures/<alias>/ に保存する（リポジトリのルートで実行する）
  loop-cli-dev classify --fixture <alias> [--mode sdd|label]
      internal/gh/testdata/fixtures/<alias>/ の open issue / PR を分類し、4 タブ別に
      優先 / 種別 / リポジトリ / 番号 / タイトル / 経過 をタブ区切りで出す（リポジトリのルートで実行する）
      --mode はその fixture の運用方式（既定 sdd）
  loop-cli-dev notify test
      デスクトップ通知を 1 件出す
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		_, _ = fmt.Fprint(stdout, usage)
		return 0
	}
	switch {
	case args[0] == "fixture" && len(args) > 1 && args[1] == "capture":
		return runSub(stderr, func() error { return fixtureCapture(args[2:], stdout, stderr) })
	case args[0] == "classify":
		return runSub(stderr, func() error { return classifyFixture(args[1:], stdout, stderr) })
	case args[0] == "notify" && len(args) > 1 && args[1] == "test":
		return runSub(stderr, func() error { return notifyTest(stdout) })
	}
	_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
	_, _ = fmt.Fprint(stderr, usage)
	return 1
}

// runSub はサブコマンドを実行し、失敗を標準エラーに書いて終了コードにする。
func runSub(stderr io.Writer, f func() error) int {
	if err := f(); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
