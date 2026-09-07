## 1. ラベルと本文の読み取り

- [x] 1.1 `internal/model` に `LabelAIAssess = "ai-assess:requested"` を足す
- [x] 1.2 `internal/model/parse.go` に `IsMidRelabel(comments []Comment) bool` と `UnblockWhen(body string) (string, bool)` を足す
- [x] 1.3 `internal/model/parse_test.go` に両関数のテストを足す（最新の routine コメントが `release:` / `restart:` / `advance:`、人のコメントが後にある、通常の routine コメント、`unblock-when:` の有無）

## 2. 分類器

- [x] 2.1 `classify.Issue` に規則 6（段階ラベル無し・`blocked` 無し・`IsMidRelabel`）を規則 5 の直後に足す
- [x] 2.2 `classify.PR` に規則 7（`ai-assess:requested`）を行 A の後・行 C の前に足す
- [x] 2.3 `internal/classify/classify_test.go` に 2 つの規則と「AI 評価待ちでも question があれば A」「blocked があれば E ではない」のテストを足す

## 3. カード詳細の unblock-when

- [x] 3.1 `internal/ui/detail.go` の `cardBodyLines` で `blocked-by: human` の次に `unblock-when: <値>` を出す
- [x] 3.2 `questionLines` のフォールバックから `unblock-when:` 行を除く
- [x] 3.3 `internal/ui/detail_test.go` に 2 つの Scenario のテストを足す

## 4. merge の警告

- [x] 4.1 `action.CheckMerge` に `ai-assess:requested` の警告を question の次に足す
- [x] 4.2 `action.CheckMerge` に未実装だった `State` の拒否（空文字列でも `OPEN` でもなければ `<State> の PR です`）を足す
- [x] 4.3 `internal/action/merge_test.go` に 4.1 と 4.2 のテストを足す

## 5. ドキュメント

- [x] 5.1 `docs/domain/issue-driven-sdd/human-turn-signals.md` を上流 `2b1b791` に同期する（同期点・判定表・キューに入れないもの・不変条件 1・変更履歴）
- [x] 5.2 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
