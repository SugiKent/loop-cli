## Why

s04-fixture-capture が `cmd/sugi-loop-cli` の骨組みと `fixture capture` を、s05-classify が `internal/model` と `internal/classify` を導入したが、
分類器の結果を人が目で見る手段は `go test` の期待値表しか無く、fixture を採り直したときに「どの issue / PR がどのタブに出るか」を確かめるには TUI（s08）を待つしかない。
また s13 が使うデスクトップ通知が、この環境（macOS）で実際に出るかをコードを書く前に確かめる手段が無い。
docs/mvp/implementation-tasks.md §2「動作確認用 CLI の導入」の残り 2 用途（fixture に分類器をかけてキューをプレーンテキスト出力、テスト通知の発火）をこの change で足す。

## What Changes

- `sugi-loop-cli classify --fixture <alias>` を追加する。s03 の `Fake` で `internal/gh/testdata/fixtures/<alias>/` を読み、s05 の `classify.Issue` / `classify.PR` で全 open issue / open PR を分類し、mvp.md「画面構成（今やるキュー）」の 4 タブ（今やる / バックログ / 進行中 / 異常）別に、列「優先 / 種別 / リポジトリ / 番号 / タイトル / 経過」のプレーンテキストを標準出力に書く。live の `gh` には繋がない（横断取得は s07）
- `sugi-loop-cli notify test` を追加する。デスクトップ通知を 1 件出す。通知ライブラリ `beeep` の依存はここで追加し、s13 の自動更新通知はこの依存を再利用する
- `help` の使い方に `classify --fixture <alias>` と `notify test` の 1 行説明を足す。振り分けは s04 の `run(args, stdout, stderr) int` にケースを足すだけ
- Issue と PR の紐づけ（Card 化）は行わない。行は issue 1 件または PR 1 件で、mvp.md の画面例と同じく PR は独立した行として出る

## Capabilities

### New Capabilities
（無し）

### Modified Capabilities
- `dev-cli`: ADDED のみ。`classify --fixture <alias>` サブコマンド、`notify test` サブコマンド、`help` への 2 行追加と振り分けの Requirement を足す。s04 の 2 つの Requirement（振り分けと `help`、失敗時の終了コード）は変えない

## Impact

- 新規: `cmd/sugi-loop-cli/classify.go` / `classify_test.go`、`cmd/sugi-loop-cli/notify.go`
- 変更: `cmd/sugi-loop-cli/main.go`（usage 定数に 2 行、`run` に `classify` / `notify` のケース）、`cmd/sugi-loop-cli/main_test.go`（help と振り分けのケース追加）、`go.mod` / `go.sum`（`github.com/gen2brain/beeep` を追加）
- 依存: `cmd/sugi-loop-cli` が `internal/gh` に加えて `internal/model` と `internal/classify` を import する。逆方向の依存は無い
- 前提: s04（`cmd/sugi-loop-cli` の骨組み・`Fake` のファイル名関数）と s05（`model` / `classify`）が実装済みであること。s05 が `example/issue-140.json` を追加している前提。s06 は fixture を変更しない
- 検証計画との対応: V-1 の fixture と分類器を人が目視確認する補助手段。V-2（live 確認）は扱わない。validation-plan.md「未定」のデスクトップ通知の差分比較は s13 が担当し、ここでは「通知が 1 件出る」ことだけを確かめる
- 後続 change への影響: s07 は Card 組み立て後の live 確認を TUI 本体で行う（`classify --live` は足さない。design.md 未決事項）。s13 は `beeep` の依存をそのまま使う
