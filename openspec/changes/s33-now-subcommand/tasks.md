## 1. 出力の組み立て

- [ ] 1.1 `cmd/loop-cli/now.go`: JSON へ写す構造体（最上位の `fetched_at` / `count` / `items` / `errors` と、1 件分の `situation` / `kind` / `priority` / `summary` / `repo` / `subject` / `issue` / `prs`）と、`*fetch.Result` と取得時刻から最上位の構造体を作る関数を書く。主体は `ui.Subject` で引き、`Card.Result.Tab == model.TabNow` で絞り、優先度昇順 → 主体の更新時刻の新しい順 → リポジトリ名昇順 → 番号昇順で並べる。空のスライスは `[]` として出す
- [ ] 1.1b `cmd/loop-cli/now.go`: PR 1 件の状態（`undecided` / `mergeable` / `merge_state_status` / `review_decision` / `checks_green` / `checks` / `unresolved_threads`）を design.md D2b の対応表どおりに写す。`MergeState` と `ReviewThreads` が `nil` の PR ではこれらを `null` にする
- [ ] 1.2 `cmd/loop-cli/now_test.go`: `gh.NewFake("../../internal/gh/testdata/fixtures/board")` と `repos` に `org/board`、固定の `now` で `fetch.Fetch` を呼び、その結果から組み立てた JSON が「今やる」の 2 枚（主体は PR#72 と PR#71 で、この順）だけを含み、バックログと異常のカードを含まないことを検証する。`repos` に fixture のリポジトリ名を渡さないと運用方式が判定されず（`fetch.Fetch` の `modes` が空になり既定の sdd に倒れて）分類が変わるので、リポジトリ名は fixture に合わせる
- [ ] 1.3 `cmd/loop-cli/now_test.go`: `internal/gh/testdata/fixtures/example` と `repos` の `org/app` の結果で、PR が主体のカードの `subject` が `{type: pr, number: 131}`・`issue.number` が 108・`prs` が番号 131 を含むことを検証する
- [ ] 1.4 `cmd/loop-cli/now_test.go`: 手で組み立てた `*fetch.Result` で、(a) 「今やる」が 0 件のとき `count` が 0 で `"items":[]` と `"errors":[]` が出ること、(b) 優先度が同じなら更新時刻の新しいカードが先に来ること、(c) 更新時刻も同じならリポジトリ名と番号の昇順になること、(d) PR 単独のカードの `issue` が `null` になることを検証する
- [ ] 1.5 `cmd/loop-cli/now_test.go`: 手で組み立てた `*fetch.Result` で、(a) `未確定の判断: 0 件` の本文・`MERGEABLE` / `CLEAN`・成功した CheckRun 1 件・未解決 thread 1 件の PR で状態の 7 欄が期待どおりに出ること、(b) `MergeState` と `ReviewThreads` が `nil` の PR で状態の 6 欄が `null` になり `number` / `title` / `url` / `labels` は出ることを検証する

## 2. サブコマンドへの接続

- [ ] 2.1 `cmd/loop-cli/now.go`: 設定パス・`gh` の確認の関数・取得の関数を引数で受け取る `runNow` を書く（design.md D7）。設定ファイルが無ければ `設定ファイルがありません: <パス>`、`config.DefaultPath` と `config.Load` の失敗はそのエラー、確認の失敗は `checkError` の文言、取得の失敗はそのエラーを標準エラーに書いて終了コード 1。成功時は整形した JSON と末尾の改行を標準出力に書いて終了コード 0
- [ ] 2.2 `cmd/loop-cli/main.go`: `run` の `switch` に `now` を足し、`config.DefaultPath()` と `gh.NewClient().Check` と `fetch.Fetch` を束ねて `runNow` に渡す。`now` の後ろに引数があればその引数を含む 1 行を標準エラーに書いて終了コード 1 で終わる。`usage` に `loop-cli now` の 1 行を足し、`run` の doc コメントの「サブコマンドは設定ファイルを読まず gh も呼ばない」を `version` / `update` に限定した文に直す
- [ ] 2.3 `cmd/loop-cli/now_test.go`: `runNow` にスタブを渡し、(a) 設定ファイルが無い、(b) 確認が `exec.ErrNotFound` を返す、(c) 取得がエラーを返す、の 3 経路で標準出力が空・標準エラーの文面・終了コード 1 を検証する
- [ ] 2.4 `cmd/loop-cli/now_test.go`: `runNow` にスタブを渡し、(a) 成功時に標準出力が 1 つの JSON オブジェクトになり終了コードが 0 であること、(b) 取得が部分失敗（`Result.Errors` が 1 件）を返しても `items` が出て `errors` に 1 件入り終了コードが 0 であること、(c) スナップショットのファイルを置いても中身と更新時刻が変わらないことを検証する
- [ ] 2.5 `cmd/loop-cli/main_test.go`: 未知のサブコマンドの使い方に `now` が出ること、`now` に余分な引数を付けると終了コード 1 になることを検証する

## 3. 利用者向けの説明

- [ ] 3.1 `README.md`: `loop-cli now` の節を足し、出力のキーと `jq` での読み方の例、`gh` と設定ファイルが要ること、スナップショットを読み書きしないこと、`errors` が空でなければ一覧が不完全であり得ること、分岐に使うなら `kind` の日本語より `situation` の記号の方が安定していることを書く

## 4. 確認

- [ ] 4.1 `go build ./... && go vet ./... && go test ./...` が通る
- [ ] 4.2 `openspec validate s33-now-subcommand --strict` が通る
