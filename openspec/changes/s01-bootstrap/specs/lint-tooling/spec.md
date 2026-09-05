## ADDED Requirements

### Requirement: gofmt と go vet がリポジトリ全体で通る
リポジトリ内のすべての Go ソースは gofmt 済みで MUST あり、`go vet ./...` はエラーなしで MUST 終了する。

#### Scenario: gofmt 差分が無い
- **WHEN** `gofmt -l .` を実行する
- **THEN** 出力は空である

#### Scenario: go vet が通る
- **WHEN** `go vet ./...` を実行する
- **THEN** 終了コード 0 で終わる

### Requirement: golangci-lint の設定がリポジトリにある
リポジトリ直下に golangci-lint の設定ファイル `.golangci.yml` が MUST 存在し、`golangci-lint run ./...` はエラーなしで MUST 終了する。
有効化する linter 集合は golangci-lint の既定集合と gofmt 相当のフォーマットチェックに限り、それ以外の linter は追加しない。

#### Scenario: 既定 linter + gofmt 相当だけが有効
- **WHEN** `.golangci.yml` を読む
- **THEN** 明示的に有効化されているのは golangci-lint の既定集合と gofmt 相当のフォーマッタだけであり、それ以外の linter 名は記載されていない

#### Scenario: golangci-lint が通る
- **WHEN** design.md 未決事項「golangci-lint の固定バージョン」で定めたバージョンの golangci-lint で `golangci-lint run ./...` を実行する
- **THEN** 終了コード 0 で終わる

#### Scenario: gofmt 違反を golangci-lint が検出する
- **WHEN** gofmt に反するインデントの Go ファイルを一時的に置いて `golangci-lint run ./...` を実行する
- **THEN** 終了コードは 0 以外になり、当該ファイルがフォーマット違反として報告される
