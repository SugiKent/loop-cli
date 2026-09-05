## 1. 依存追加

- [ ] 1.1 `go get go.yaml.in/yaml/v3@latest` で YAML ライブラリを go.mod に追加し、`go list -m all` に `go.yaml.in/yaml/v3` と s01 の 5 依存（`charm.land/bubbletea/v2` / `bubbles/v2` / `lipgloss/v2` / `glamour/v2` / `huh/v2`）がすべて残っていることを確認する。`go mod tidy` で 3 依存が落ちた場合は `go get charm.land/<name>/v2@latest` で戻す

## 2. 型と読み込み

- [ ] 2.1 `internal/config/config.go` を作成する。`MergeMethod` 型と定数 3 つ（`squash` / `merge` / `rebase`）、`Repo`（`Name` / `MergeMethod`）、`Config`（`Repos` / `RefreshIntervalSec` / `MergeMethod` / `Editor` / `Notify` の 5 フィールドのみ）、`DefaultPath()`（`os.UserHomeDir()` + `.config/sugi-loop/config.yml`）を実装する
- [ ] 2.2 同ファイルに `Load(path string) (*Config, error)` を実装する。既定値入りの `Config`（120 / squash / `$EDITOR` / true）へ未知キー拒否付きでデコードし、空ファイル（`io.EOF`）は空設定として扱い、`Editor` に環境変数展開をかける。`Repo` に「文字列またはマッピング `{name, merge_method}`」を読むカスタムデコードを実装し、マッピングに `name` / `merge_method` 以外のキーがあればキー名を含むエラーを返す（未知フィールド拒否はノード単位のデコードに引き継がれない）
- [ ] 2.3 同ファイルに検証を実装する。design.md の順（repos 空 → `owner/name` 形式 → グローバル merge_method → リポジトリ別 merge_method → refresh_interval_sec ≥ 1）で最初のエラーを返し、文言に項目名と不正値を含め、すべてのエラーをパス付きで包む。検証後、`MergeMethod` が空の `Repo` にグローバル値を埋める

## 3. テスト

- [ ] 3.1 `internal/config/config_test.go` を作成し、`t.TempDir()` に YAML を書いて `Load` を検証する。正常系: mvp.md の例そのまま（2 件・120・squash・true・`Editor` が `t.Setenv` した `EDITOR` の値）/ `repos` のみで既定値が入る / `notify: false` が false のまま / 明示値（30・rebase・vim）が既定値を上書きする / `EDITOR` 未設定（`t.Setenv("EDITOR", "")` で足りる）で `Editor` が空でも成功する / `org/web` だけ rebase に上書きされ `org/app` は squash
- [ ] 3.2 同ファイルに異常系を追加する。ファイル無し（パスを含む）/ 構文不正（パスを含む）/ `repos: []`・`repos` キー無し・空ファイル（`repos` を含む）/ `app`・`org/app/extra`・`/app`・`org/ app`（不正値を含む）/ `merge_method: fast-forward`（`merge_method` と `fast-forward` を含む）/ リポジトリ別 `ff`（`org/web` と `ff` を含む）/ `refresh_interval_sec: 0` / `refresh_interval: 60`（`refresh_interval` を含む）/ `repos` 要素に `merge_methd: rebase`（`merge_methd` を含む）/ `token: ghp_xxx`（`token` を含む）。各ケースで `Config` が nil であることも確認する
- [ ] 3.3 `DefaultPath` のテストを追加する。`t.Setenv("HOME", "/Users/alice")` で `/Users/alice/.config/sugi-loop/config.yml` が返ること、`t.Setenv("HOME", "")` でエラーが返ること

## 4. 最終確認

- [ ] 4.1 `gofmt -l .` が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
