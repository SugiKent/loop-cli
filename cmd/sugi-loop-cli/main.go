// Command sugi-loop-cli は sugi-loop の動作確認用 CLI。
package main

import (
	"fmt"
	"io"
	"os"
)

const usage = `sugi-loop-cli は sugi-loop の動作確認用 CLI です。

使い方:
  sugi-loop-cli help
      この使い方を表示する
  sugi-loop-cli fixture capture --repo owner/name --alias <alias>
      指定リポジトリの open issue / PR を採取し、伏せ字にして
      internal/gh/testdata/fixtures/<alias>/ に保存する（リポジトリのルートで実行する）
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if args[0] == "fixture" && len(args) > 1 && args[1] == "capture" {
		if err := fixtureCapture(args[2:], stdout, stderr); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
	fmt.Fprint(stderr, usage)
	return 1
}
