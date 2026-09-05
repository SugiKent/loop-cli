## Why

配布は `go install` だけで、入れた後に新しい版が出たことを知る手立ても、上げ直す手順を思い出させる仕組みも無い。README の 1 コマンドを覚えている人しか更新できない。

さらに、`go.mod` のモジュールパス `github.com/SugiKent/sugi-loop` が公開リポジトリ `github.com/SugiKent/loop-cli` と食い違っている。proxy.golang.org は前者に 404 を返すので、README に書いてある `go install github.com/SugiKent/sugi-loop/cmd/sugi-loop@latest` は今の時点で失敗する。update 機構は `go install <module>@latest` の上に乗るため、この不一致を直さないと機構ごと動かない。

## What Changes

- `go.mod` のモジュールパスを `github.com/SugiKent/loop-cli` に変え、リポジトリ内の全 import パスを追随させる（利用者の決定）。README のインストール手順も新しいパスに直す
- `internal/version` を新設する。現在の版は `debug.ReadBuildInfo()` の `Main.Version`（`go install` で入れたバイナリはタグまたは擬似バージョン、手元の `go build` は `(devel)`）、最新の版は `go list -m -json <module>@latest` の `Version` から取る
- `cmd/sugi-loop` に `version` と `update` のサブコマンドを足す。引数なしは今までどおり TUI。`update` は最新版を調べ、現在の版と違えば `go install <module>/cmd/sugi-loop@latest` を実行し、`更新しました: <現在> → <最新>` を出す。同じなら install せず `最新版です: <版>` を出す
- TUI は起動時に 1 度だけ非同期で最新版を調べ、新しい版があればキュー画面のヘッダ右に `↑ update` を出す。調べに失敗したとき・現在の版が `(devel)` のときは何も出さない（画面を止めない、エラーも出さない）
- GitHub Releases からのバイナリ配布は行わない。Go は README の前提であり、リリースのワークフローも無いため、`go install` に一本化する

## Capabilities

### New Capabilities

- `self-update`: 現在の版と最新の版の取得、`version` / `update` サブコマンド、TUI の起動時チェック

### Modified Capabilities

- `go-module`: モジュールパスの変更
- `tui-entrypoint`: 引数によるサブコマンドの振り分けと、起動時チェックの `ui.Options` への受け渡し
- `queue-screen`: ヘッダ右への `↑ update` の追加
