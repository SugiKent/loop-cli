## Why

sugi-loop はまだ Go モジュールが無く、`go build` できるコードが 1 行も無い。後続の全 change（s02 以降）は
「`go build ./... && go vet ./... && go test ./...` が通る」ことを完了条件にするため、その土台（モジュール・依存・lint・
起動する最小 TUI・CI）を最初に 1 つの change として固める。docs/mvp/implementation-tasks.md の「## 1. 基盤セットアップ」全 4 項目に対応する。

## What Changes

- Go モジュールを作成し、D-003 の TUI 系依存（Bubble Tea v2 / Bubbles v2 / Lip Gloss v2 / Glamour / huh）を go.mod に固定する
- gofmt / `go vet` / golangci-lint（既定 linter + gofmt 相当のみ）を導入し、設定ファイルをリポジトリに置く
- `cmd/sugi-loop` に hello world の TUI を置く。起動して 1 フレーム描画し、`q` で終了する。`go build ./...` が通る
- GitHub Actions で build / vet / test / lint を回す CI を追加する（lint の CI 実行は implementation-tasks.md §1 に無く、design.md 未決事項「CI で golangci-lint を回すか」の既定値）
- `.gitignore` にビルド成果物を登録する

Go パッケージのディレクトリは `cmd/sugi-loop` だけ。D-003 の `internal/*` と `cmd/sugi-loop-cli` は後続 change（s02〜）が作る。

## Capabilities

### New Capabilities
- `go-module`: Go モジュールの存在、Go バージョン、D-003 の依存 5 つが go.mod に固定されていること、`go build ./...` が通ること
- `lint-tooling`: gofmt / `go vet` / golangci-lint の設定と、それらがリポジトリ全体に対してエラーなしで通ること
- `tui-entrypoint`: `cmd/sugi-loop` バイナリの起動・1 フレーム描画・`q` 終了の振る舞い
- `ci-pipeline`: GitHub Actions が push / pull_request で build / vet / test / lint を実行し、失敗を検知すること

### Modified Capabilities
（先行 change は無い）

## Impact

- 新規: `go.mod` / `go.sum` / `.golangci.yml` / `.gitignore` / `cmd/sugi-loop/main.go` / `cmd/sugi-loop/main_test.go` / `.github/workflows/ci.yml`
- 外部依存: charm.land 配下の Bubble Tea v2 / Bubbles v2 / Lip Gloss v2 / Glamour v2 / huh v2。beeep（通知）は s13 が追加するのでここでは入れない
- 後続 change はすべてこの change の go.mod と CI を前提にする。`tui-entrypoint` の hello world 画面は s08（今やるキュー画面）が置き換える
