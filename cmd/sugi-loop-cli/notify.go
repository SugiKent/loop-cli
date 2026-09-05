package main

import (
	"fmt"
	"io"

	"github.com/gen2brain/beeep"
)

// notifyTest はデスクトップ通知を 1 件出す。s13 の自動更新通知は同じ依存を使う。
func notifyTest(stdout io.Writer) error {
	// icon は string か []byte でなければならない。空文字列でアイコンなし。
	if err := beeep.Notify("sugi-loop", "テスト通知", ""); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout, "通知を送りました")
	return err
}
