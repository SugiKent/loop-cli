# ci-pipeline Specification

## Purpose
TBD - created by archiving change s01-bootstrap. Update Purpose after archive.
## Requirements
### Requirement: GitHub Actions が build / vet / test / lint を実行する
`.github/workflows/ci.yml` が MUST 存在し、`main` への push と `main` 向けの pull_request をトリガーに、`go build ./...` / `go vet ./...` / `go test ./...` を MUST 実行し、golangci-lint をリポジトリ全体に対して MUST 実行する。
使用する Go バージョンは `go.mod` の `go` ディレクティブから読み取り、ワークフロー内にハードコードしない。
いずれか 1 つでも失敗したらワークフローは失敗する。

#### Scenario: すべて通る変更で CI が成功する
- **WHEN** `main` 向けの pull_request で build / vet / test / lint がすべて通る
- **THEN** ワークフローの結論は success になる

#### Scenario: テストが失敗すると CI が失敗する
- **WHEN** `go test ./...` が失敗するコミットを push する
- **THEN** ワークフローの結論は failure になる

#### Scenario: gofmt 違反で CI が失敗する
- **WHEN** gofmt 済みでない Go ファイルを含むコミットを push する
- **THEN** lint ステップが失敗し、ワークフローの結論は failure になる

#### Scenario: Go バージョンが go.mod から決まる
- **WHEN** `go.mod` の `go` ディレクティブを変更する
- **THEN** ワークフロー側の変更なしに、CI はその Go バージョンで実行される

