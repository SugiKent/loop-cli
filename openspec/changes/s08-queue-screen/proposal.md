## Why

s07-fetch-cards が終わると「設定リポジトリの open issue / PR を取り、分類済みの `model.Card` 群にする」層が揃うが、`cmd/sugi-loop` は s01 の hello world のままで、利用者が見る画面が無い。mvp.md「画面構成（今やるキュー）」の 4 タブ + 選択行プレビューを作り、起動 → 設定読み込み → `gh` の確認 → 取得 → 表示までを 1 本につなぐのがこの change である。
この change は docs/mvp/implementation-tasks.md §3 (P1) の項目「今やるキュー画面: 4 タブ（今やる / バックログ / 進行中 / 異常）+ 選択行プレビュー。狭い端末は 1 ペインにフォールバック」を実装する。s07 の design が「`cmd/sugi-loop` への配線と `Check` は s08」と定めた配線も引き取る。

## What Changes

- `internal/ui` を新設する。Bubble Tea の Model として、`fetch.Result` の `Cards` を `Card.Result.Tab` で 4 タブに振り分け、上段の表（優先 / 種別 / リポジトリ / # / タイトル / 経過。行の色は種別で固定）と下段の選択行プレビュー（主体の本文 1 行目 + ラベル、本文とコメント本文を Glamour で Markdown レンダリング、コメントは AI 発を左バーで区別）、ヘッダ（タブ名と件数、↻ 最終更新時刻）、フッタ（キーヒントとステータス）を描く
- キーバインドは mvp.md の表のうち `j` / `k` / `↑` / `↓`（行移動）、`1`–`4` / `Tab`（タブ切替）、`q` / `Ctrl+C`（終了）だけを実装する。`Enter`（s09）、`a` / `t`（s10 / s11）、`o` / `R` / `?`（s12）、`m` / `n` / `s` / `A` / `v` / `h` / `l` / `g` / `/`（s14 以降）は押しても何もしない
- 狭い端末では表とプレビューを切り替える 1 ペイン表示にフォールバックする
- 取得を `tea.Cmd` として非同期に実行し、取得中はステータスにスピナー、失敗時は前回の `Cards` を維持してエラーを赤で出す（D-002）。初回取得前は空の画面 + スピナー。部分失敗（`Result.Errors`）も赤で出す
- `cmd/sugi-loop/main.go` を hello world から置き換える: `config.DefaultPath` → `config.Load` → `gh.NewClient` + `Check` → `ui.New(...)` で Bubble Tea を起動し、`Init` が `fetch.Fetch` を発行する。設定・`Check`・プログラム実行のどれが失敗しても標準エラーに 1 行出して終了コード 1
- **依存追加**: Bubbles v2（スピナー）と Glamour v2（Markdown レンダリング）を go.mod に加える。s01 design は「Glamour は s09」と書いていたが、mvp.md がプレビューの Markdown レンダリングを求めるので s08 が加える
- s06 の `elapsed` と同じ規則の経過表記を `internal/ui` に置く。s06 側の重複は残す

## Capabilities

### New Capabilities
- `queue-screen`: `internal/ui` の今やるキュー画面。この capability はタブ振り分けと並び、行の主体と列の表記、色、キー操作、ヘッダ / フッタ、プレビュー、狭い端末の 1 ペイン、取得状態（スピナー / 前回結果の維持 / エラー表示）を定める

### Modified Capabilities
- `tui-entrypoint`: 「sugi-loop バイナリが起動して 1 フレーム描画する」の初期フレームを hello world からキュー画面（空 + スピナー）に変え、起動手順（`config.Load` → `Check` → 取得の発行）を定める。「起動失敗は標準エラーに出て終了コード 1 になる」の対象に設定読み込みと `Check` の失敗を加える。「q で終了する」は変えない
- `gh-client`: 「起動前に gh の存在と認証を確認する」の「`Check` を呼ぶのは起動時の s07」を s08 に改める（振る舞いは変えない）

## Impact

- 新規: `internal/ui/model.go` / `view.go` / `rows.go` / `preview.go` / `elapsed.go` と各テスト
- 変更: `cmd/sugi-loop/main.go`（hello world の `model` 型を削除し配線に置き換え）/ `cmd/sugi-loop/main_test.go`（キー操作のテストは `internal/ui` に移り、ビルド確認だけ残す）/ `go.mod` / `go.sum`
- 依存: `internal/config`（`DefaultPath` / `Load`）、`internal/gh`（`NewClient` / `Check`）、`internal/fetch`（s07。`Fetch` / `Result`）、`internal/model`（`Card` / `Result` / `Situation.Kind()` / `Situation.Tab()` / `Comment`）。`internal/ui` は `classify` を呼ばない（`Fetch` が分類済みの Card を返す）
- 前提: s07-fetch-cards が実装済みであること（`internal/fetch` を import する）。s07 が遅延取得の範囲をどう決めても（レビューで「全 open PR に `ViewPR`」へ変わる見込み）、この change は `Result{Cards, Errors}` の形だけに依存する
- 後続 change への影響: s09 が `Enter` でカード詳細画面を足す。s12 が `R` / `?` / `o`、s13 が自動更新とスナップショットの読み書き（起動直後の stale 表示）と通知を足す。いずれも `internal/ui` の Model にキーとメッセージを足す形で、この change の Requirement は変えない。ただし s13 は D-002 の stale 表示（起動直後にスナップショットの Card を出す）のため `New` の引数を拡張し、初期 `cards` と `at` を受け取れるようにする（この change の `New(fetcher)` は取得前の空状態しか作れない）
