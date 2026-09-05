## Why

sugi-loop は監視対象リポジトリ・更新間隔・merge 方式・エディタ・通知の有無を `~/.config/sugi-loop/config.yml` から読む（mvp.md「設定ファイル」節）が、
まだ読み込む実装が無い。後続の横断取得（s07）・自動更新（s13）・merge（s14）・回答（s10）・通知（s13）はすべてこの設定値を前提にするため、
先に `internal/config` として固める。docs/mvp/implementation-tasks.md §2「`internal/config`: `~/.config/sugi-loop/config.yml` の読み込み」に対応する。

## What Changes

- `internal/config` パッケージを新設し、`config.yml` を読み込んで検証済みの `Config` 型を返す `Load` を提供する
- 設定項目は mvp.md のとおり `repos` / `refresh_interval_sec` / `merge_method`（リポジトリ別上書き可）/ `editor` / `notify` の 5 つ。それ以外は持たない
- 既定値（`refresh_interval_sec` 120、`merge_method` squash、`notify` true、`editor` は環境変数 `EDITOR`）を適用する
- ファイルが無い・YAML が壊れている・`repos` が空・`owner/name` 形式でない・`merge_method` が不正・`refresh_interval_sec` が 1 未満・未知のキーがある場合は、原因とファイルパスを含むエラーを返す
- 認証トークンは設定に持たない（mvp.md「認証は `gh auth` を再利用する」）。YAML ライブラリを 1 つ依存に追加する

この change で作るのは `internal/config` だけ。`cmd/sugi-loop` から `Load` を呼ぶ配線は、設定値を最初に消費する s07 が担当する。

## Capabilities

### New Capabilities
- `config-loading`: 設定ファイルのパス解決、YAML の読み込みと `Config` 型への変換、既定値の適用、検証エラーの振る舞い、リポジトリ別 `merge_method` の解決、認証情報を持たないこと

### Modified Capabilities
（無し。s01-bootstrap の capability は変更しない）

## Impact

- 新規: `internal/config/config.go` / `internal/config/config_test.go`
- 外部依存: YAML ライブラリを 1 つ `go.mod` に追加する（選定は design.md の未決事項）
- 後続 change への影響: s07（`Repos` / `RefreshIntervalSec`）、s10（`Editor`）、s13（`RefreshIntervalSec` / `Notify`）、s14（`Repo.MergeMethod`）がこの change の `Config` 型を読む。設定値の解釈はここで確定し、消費側で再検証しない
