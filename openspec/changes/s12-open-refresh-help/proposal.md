## Why

s11-todo-toggle までの TUI は、取得を起動時の 1 回しか行わず（`R` も自動更新も無い）、`a` / `t` の書き込み結果を画面に反映する手段が再起動しかない。局面 F（段階ラベル 2 つ以上）の TUI のアクションは human-turn-signals.md で「表示して警告、ブラウザで開く」と定められているのに、ブラウザで開くキーが無い。キーが増えてフッタに収まらなくなってきたのに、キー一覧を見る場所も無い。この change は docs/mvp/implementation-tasks.md §3 (P1)「`o` ブラウザで開く / `R` 全件再取得 / `?` ヘルプ」を実装する。

## What Changes

- `o`: 画面の対象（キュー画面は選択行の主体、カード詳細は Issue、PR 詳細はその PR）を s03 の `GHClient.Browse`（`gh browse <n> -R <repo>`）でブラウザで開く。`Model` は `gh` を直接呼ばず、コマンドとして実行する。失敗はフッタのステータスに赤で出す。確認画面とヘルプ画面では何もしない
- `R`: キュー画面で s08 の取得コマンド（`fetchCmd`）をもう一度発行する。取得中の `R` は無視する（多重発行しない）。取得開始時に取得のエラー表示と s10 / s11 の書き込みステータスを消す（両 change が「次の取得が始まったとき」に消えると定めたその時点）。カード詳細 / PR 詳細 / 確認 / ヘルプでは何もしない。自動更新は s13
- `?`: ヘルプ画面を開く。この時点で実装済みのキーだけ（`j` / `k` / `↑` / `↓`、`1`–`4` / `Tab`、`Enter`、`a`、`t`、`o`、`g`、`R`、`?`、`q`、`Esc`、`Tab`（PR 選択）、`x`、`PgUp` / `PgDn`、`p`）を mvp.md のキーバインド表の「動作」で一覧する。未実装のキーは出さない（後続 change が足す）。`?` または `Esc` で開いた画面に戻る。キュー / カード詳細 / PR 詳細から開け、確認画面からは開けない。画面の状態は キュー / カード詳細 / PR 詳細 / 確認 / ヘルプ の 5 つになる
- キュー画面のフッタのヒントを `Enter 開く  a 回答  t todo  o ブラウザ  R 更新  ? ヘルプ  q 終了` に組み直す。s11 のヒントに 3 キーを足すと 88 列になり、`Model` の既定幅 80 で `q 終了` が切れ、取得中はステータスに押されてヒントが消える。mvp.md のフッタ例（`Enter 開く  a 回答  t todo  m merge  n 新規issue  o ブラウザ  R 更新  ? ヘルプ  q 終了`）と同じく移動系のキーは出さず、`?` のヘルプに委ねる
- カード詳細と PR 詳細のフッタのヒントに `o ブラウザ` と `? ヘルプ` を足す（`? ヘルプ` は `a 回答` の前）
- `internal/gh` は変えない。`Browse` は s03 が `GHClient` / `Client` / `Fake` に定義済みで、`Fake.Browse` は `Calls` に `Method` `Browse` / `Repo` / `Number` を記録する（`internal/gh/fake.go` で確認済み）

## Capabilities

### New Capabilities
- `browse-open`: `internal/ui` の `o`。画面ごとの対象の決め方、`Browse` をコマンドとして呼ぶこと、失敗のステータス表示
- `manual-refresh`: `internal/ui` の `R`。キュー画面での再取得、取得中の無視、取得開始時に消すもの、他の画面では何もしないこと
- `help-screen`: `internal/ui` の `?`。ヘルプ画面をどのキーで開閉しどの画面に戻るか、どの行をどの順で一覧するか、開いている間に届いたメッセージをどう扱うかを定める

### Modified Capabilities
- `queue-screen`: Requirement「j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、他のキーは何もしない」（s11 が MODIFIED した最新版）の `o` / `R` / `?` を「何もしない」から外す。Requirement「ヘッダはタブ名と件数と最終更新時刻、フッタはキーヒントとステータスを出す」（同）のフッタのヒントを組み直す
- `card-detail`: Requirement「Enter でカード詳細を開き、Esc で 1 つ前の画面に戻る」（s10 が MODIFIED した最新版）の画面の状態を 4 つ → 5 つ（ヘルプを追加）にする。Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」（s11 が MODIFIED した最新版）のカード詳細 / PR 詳細のフッタに `o ブラウザ` と `? ヘルプ` を足す（`? ヘルプ` は `a 回答` の前）

## Impact

- この change が新しく作るファイルは次のとおり。`internal/ui/browse.go` は `o` のキー処理とコマンドと結果メッセージを持つ。`internal/ui/browse_test.go` はそのテストである。`internal/ui/help.go` はヘルプ画面のキー処理と `View` を持つ。`internal/ui/help_test.go` はそのテストである
- 変更: `internal/ui/model.go`（`R`、取得開始の共通経路、`o` / `?` の振り分け、ヘルプ画面の状態）/ `internal/ui/view.go`（フッタのヒント、ヘルプ画面の `View` への振り分け）/ `internal/ui/model_test.go` / `view_test.go` / `detail_test.go`（`o` / `R` / `?` を「何も変えない」と見ていた検証とフッタの検証を直す）
- 依存: `internal/gh`（`GHClient.Browse`、`Fake.Calls`。**interface / Client / Fake は変えない**）、s08 の `fetchCmd` と `Fetcher`、s10 の `New(fetcher, client, editor)` の `client` とフッタ右のステータス。`New` の署名と `cmd/sugi-loop/main.go` は変えない。新しい外部依存は無い
- 前提: s11-todo-toggle が実装済みであること（画面の状態 4 つ、書き込みステータス、`client`）。archive の順は s08 → s09 → s10 → s11 → s12。s08a-onboarding の `queue-screen` delta は ADDED（空キューのヒント）だけで、この change の MODIFIED と衝突しない
- 後続 change への影響: s13 の自動更新は `R` と同じ取得開始の経路を使う。s14（`m`）/ s15（`n` / `s`）/ s16（`A`）/ s17（`v` / `h` / `l`）/ s19（`/`）はヘルプ画面の一覧に自分のキーの行を MODIFIED で足し、フッタのヒントを足す
