## 1. 出力の組み立て

- [ ] 1.1 `cmd/loop-cli/now.go`: JSON へ写す構造体（最上位の `fetched_at` / `count` / `items` / `errors` と、1 件分の `priority` / `situation` / `kind` / `summary` / `repo` / `subject` / `issue` / `prs`）と、`*fetch.Result` と取得時刻から最上位の構造体を作る関数を書く。主体は `ui.Subject` で引き、`Card.Result.Tab == model.TabNow` で絞り、優先度昇順 → 主体の更新時刻の新しい順 → リポジトリ名昇順 → 番号昇順で並べる
- [ ] 1.2 `cmd/loop-cli/now_test.go`: `gh.NewFake("../../internal/gh/testdata/fixtures/example")` と `fetch.Fetch` で作った結果から JSON を組み立て、キー名・並び順・「今やる」以外が入らないことを検証する。`items` が空のときに `"items":[]` と `"errors":[]` が出て、`count` が 0 になることも検証する
- [ ] 1.3 `cmd/loop-cli/now_test.go`: PR が主体で issue に紐づくカードで `subject.type` が `pr`・`issue.number` が紐づく issue・`prs` が PR を含むこと、PR 単独のカードで `issue` が `null` になることを検証する

## 2. サブコマンドへの接続

- [ ] 2.1 `cmd/loop-cli/now.go`: 設定の読み込み → `gh.Client.Check` → `fetch.Fetch` → 標準出力へ JSON、の順に進む関数を書く。設定ファイルが無ければ `設定ファイルがありません: <パス>` で終了コード 1、`Check` の失敗は `checkError` と同じ 2 行、検索の失敗はそのエラー 1 行で終了コード 1。詳細取得の部分失敗は `errors` に入れて終了コード 0。onboarding のフォームには入らない
- [ ] 2.2 `cmd/loop-cli/main.go`: `run` の `switch` に `now` を足し、`usage` に `loop-cli now  「今やる」の状態を JSON で出す` の 1 行を足す
- [ ] 2.3 `cmd/loop-cli/main_test.go`: `now` の失敗経路（設定ファイルが無い / `Check` が失敗する / 検索が失敗する）で標準出力が空・標準エラーの文面・終了コード 1 を検証し、未知のサブコマンドの使い方に `now` が出ることを検証する

## 3. 利用者向けの説明

- [ ] 3.1 `README.md`: `loop-cli now` の節を足し、出力のキーと `jq` での読み方の例、`gh` と設定ファイルが要ること、スナップショットを読み書きしないことを書く

## 4. 確認

- [ ] 4.1 `go build ./... && go vet ./... && go test ./...` が通る
- [ ] 4.2 `openspec validate s33-now-subcommand --strict` が通る
