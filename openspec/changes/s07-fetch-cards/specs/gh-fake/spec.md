## ADDED Requirements

### Requirement: Fake は並行呼び出しでも Calls の内容を壊さない
`Fake` は複数のゴルーチンから同時にメソッドを呼ばれても、`Calls` への追記を排他して MUST 行う（`internal/gh/fake.go` の `Calls` への `append` を `sync.Mutex` で守る）。並行して呼ばれた各呼び出しはちょうど 1 件ずつ `Calls` に残り、`Method` / `Repo` / `Number` 等の内容は失われない。直列に呼んだときの記録内容と順序の規則（Requirement「Fake は書き込み呼び出しと ViewIssue を記録する」）は変わらない。並行して呼ばれた呼び出しどうしの `Calls` 内の順序は定めない。s07 の `Fetch` が `ViewIssue` を並行して呼ぶ最初の利用者である。

#### Scenario: 並行した ViewIssue が全件記録され、race が無い
- **WHEN** `NewFake("testdata/fixtures/example")` に対して 8 本のゴルーチンから `ViewIssue(ctx, "org/app", 108)` を同時に呼び、全部の完了を待ってから `Calls` を見る（`go test -race ./internal/gh/` で実行する）
- **THEN** race detector の報告が無く、`Calls` は 8 件で、全件の `Method` が `ViewIssue`、`Repo` が `org/app`、`Number` が 108 である
