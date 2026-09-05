## ADDED Requirements

### Requirement: Go モジュールが存在し依存が固定されている
リポジトリは以下を MUST 満たす。
- リポジトリ直下に `go.mod` が存在し、モジュールパスは design.md の未決事項で定めた既定値 `github.com/SugiKent/sugi-loop` である
- `go` ディレクティブは `go mod init` が手元の Go から書き出した値（1.26 系。例 `1.26.6`）のままである
- `require` に D-003 の TUI 系依存 5 つ（Bubble Tea v2 / Bubbles v2 / Lip Gloss v2 / Glamour / huh）が含まれ、`go.sum` が同期している
- beeep はこの change では含めない（s13 が追加する）

#### Scenario: 依存 5 つが go.mod に固定されている
- **WHEN** `go list -m all` を実行する
- **THEN** 出力に `charm.land/bubbletea/v2` / `charm.land/bubbles/v2` / `charm.land/lipgloss/v2` / `charm.land/glamour/v2` / `charm.land/huh/v2` の 5 モジュールがバージョン付きで含まれる
- **THEN** 出力に beeep のモジュールは含まれない

#### Scenario: クリーンな環境でビルドできる
- **WHEN** リポジトリを clone した直後に `go build ./...` を実行する
- **THEN** 終了コード 0 で終わり、`go.mod` / `go.sum` に差分が生じない

### Requirement: ビルド成果物をコミットしない
`.gitignore` は、`go build` が生成するバイナリと `go build -o` の出力先を、ルートからの位置を固定した（先頭 `/` 付きの）パターン `/bin/` / `/sugi-loop` / `/cmd/sugi-loop/sugi-loop` で MUST 除外する。
先頭 `/` の無い `sugi-loop` を書いてはならない。ディレクトリ `cmd/sugi-loop/` 自体が無視され、ソースが追跡されなくなるため。

#### Scenario: バイナリが untracked に現れない
- **WHEN** `go build -o bin/sugi-loop ./cmd/sugi-loop` と `go build ./cmd/sugi-loop`（カレントディレクトリにバイナリを出す）を実行し `git status --porcelain` を見る
- **THEN** `bin/` と `sugi-loop` バイナリは出力に現れない
- **THEN** `cmd/sugi-loop/main.go` は追跡対象（未 commit なら untracked として出力に現れる）のままである
