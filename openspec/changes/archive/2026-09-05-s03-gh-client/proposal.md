## Why

sugi-loop は GitHub のデータ層を `gh` CLI のサブプロセスで持つ（D-003）が、まだ `gh` を呼ぶコードが無い。
後続の change は「`gh` を呼んで JSON を型に写す」層と「live なしで同じ型を返す fake」を前提にする。横断取得は s07、分類器のテスト（V-1）は s05、書き込み操作は s10〜s16 が使う。
そのため先に `internal/gh` として固める。この change は docs/mvp/implementation-tasks.md §2 の `internal/gh` の項目（`GHClient` interface、`gh` サブプロセス実装、JSON fixture の fake）に対応する。

## What Changes

- `internal/gh` パッケージを新設し、`GHClient` interface を定義する
- interface の読み取りメソッドは D-001 の呼び出し（search issues / search prs / issue view / pr view / GraphQL の reviewThreads / GraphQL の cross-reference / REST timeline）から導く
- interface の書き込みメソッドは human-turn-signals.md「TUI のアクション」列と mvp.md キーバインド表「内部処理」列から導く（issue comment / pr comment / add-label / remove-label / pr merge / issue create / review thread reply / browse）
- `gh` サブプロセス実装 `Client` を提供する。すべて `--json` または `gh api` の JSON で受け、`-R owner/repo` を明示する。`mergeable: UNKNOWN` の 2 秒後 1 回再取得（D-001）は merge ガード取得用の `ViewPRMergeState` だけが持ち、`question` PR のコメント取得（`ViewPR`）は待たない
- `gh` の `--json` 出力を写した生の型を `internal/gh` 内に置く。`internal/model` は s05 が導入するのでここでは触れない
- `gh` 未インストールと `gh auth status` 失敗を起動前に検知する `Check` を提供する。非 0 終了（stderr 付き）と JSON デコード失敗を区別できるエラー型を提供する
- JSON fixture を返す fake `Fake` を提供する。`internal/gh/testdata/fixtures/<repo-alias>/` のファイル命名規則をこの change で決める（s04 が採取し、s05 が消費する）。書き込み系は呼び出し記録を持ち、テストで引数を検証できる

この change で作るのは `internal/gh` だけである。稼働リポジトリ由来の fixture は置かない（s04 が採取する）。`cmd/sugi-loop` からの配線は s07 が担当する。

## Capabilities

### New Capabilities
- `gh-client`: `GHClient` interface のメソッド集合と生の型、`gh` サブプロセス実装が発行する正確なコマンド引数、エラー型と失敗時の振る舞い、`mergeable: UNKNOWN` の再取得、起動前チェック
- `gh-fake`: fixture ディレクトリの命名規則、fixture を読んで `GHClient` と同じ型を返す fake、書き込み呼び出しの記録

### Modified Capabilities
（無し。s01-bootstrap と s02-config の capability は変更しない）

## Impact

- この change は `internal/gh/gh.go` / `internal/gh/types.go` / `internal/gh/decode.go` / `internal/gh/client.go` / `internal/gh/fake.go` と、そのテスト `internal/gh/decode_test.go` / `internal/gh/client_test.go` / `internal/gh/fake_test.go` を新規に作る。fake のテストに使う手書きの最小 fixture を `internal/gh/testdata/fixtures/example/` に置く
- `gh` 未インストールと未認証の検知は `Client.Check` として提供する。テストは `gh` を起動せず、ネットワークに出ない
- 外部依存は追加しない。標準ライブラリ（`os/exec` / `encoding/json` / `context`）だけで書く
- 実行時には `gh` CLI が PATH 上に存在し、`gh auth login` が済んでいる必要がある
- 後続 change への影響: s04 は `Client` で取った生 JSON をこの change の命名規則で保存する（`pr-<n>.json` だけは `ViewPR` と `ViewPRMergeState` の合成フィールド列で採る。コマンドは design.md 未決事項「fixture の内容」）。s05 は `Fake` で fixture を読んで分類器をテストする。s07 は `Check` と読み取りメソッドを使い、s10〜s16 は書き込みメソッドを使う
- 後続 change が新しいメソッド（例: s18 のレートリミット取得、s19 の GraphQL 統合）を必要とする場合は、その change が `gh-client` capability に ADDED で Requirement を足し、interface と `Client` と `Fake` の 3 か所を同時に拡張する
