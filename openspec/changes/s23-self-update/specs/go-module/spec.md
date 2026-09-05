## MODIFIED Requirements

### Requirement: Go モジュールが存在し依存が固定されている
リポジトリは以下を MUST 満たす。
- リポジトリ直下に `go.mod` が存在し、モジュールパスは公開リポジトリと同じ `github.com/SugiKent/loop-cli` である（s01 の既定値 `github.com/SugiKent/sugi-loop` は公開リポジトリと食い違い、`go install` も `go list -m ...@latest` も解決できないため s23 で変更した）
- リポジトリ内のすべての import パスが新しいモジュールパスを指し、`github.com/SugiKent/sugi-loop` は `*.go` / `go.mod` / `README.md` に残っていない
- `go` ディレクティブは `go mod init` が手元の Go から書き出した値（1.26 系。例 `1.26.6`）のままである
- `require` に D-003 の TUI 系依存 5 つ（Bubble Tea v2 / Bubbles v2 / Lip Gloss v2 / Glamour / huh）と beeep（s13）が含まれ、`go.sum` が同期している
- s23 の自己更新は依存を増やさない（`go list` / `go install` をサブプロセスとして呼ぶ）

#### Scenario: モジュールパスが公開リポジトリと一致する
- **WHEN** `go list -m` を実行する
- **THEN** 出力は `github.com/SugiKent/loop-cli` である

#### Scenario: 旧モジュールパスが残っていない
- **WHEN** `*.go` / `go.mod` / `README.md` を `github.com/SugiKent/sugi-loop` で検索する
- **THEN** 一致する行は無い（openspec の spec と docs の記述は経緯として残ってよい）

#### Scenario: proxy が最新版を返す
- **WHEN** モジュールを含まないディレクトリで `go list -m -json github.com/SugiKent/loop-cli@latest` を実行する
- **THEN** 終了コード 0 で `Version` を含む JSON が返る

#### Scenario: 依存 5 つが go.mod に固定されている
- **WHEN** `go list -m all` を実行する
- **THEN** 出力に `charm.land/bubbletea/v2` / `charm.land/bubbles/v2` / `charm.land/lipgloss/v2` / `charm.land/glamour/v2` / `charm.land/huh/v2` の 5 モジュールがバージョン付きで含まれる

#### Scenario: クリーンな環境でビルドできる
- **WHEN** リポジトリを clone した直後に `go build ./...` を実行する
- **THEN** 終了コード 0 で終わり、`go.mod` / `go.sum` に差分が生じない
