## 1. 分類

- [x] 1.1 `internal/classify/classify.go`: `StaleAfter`（3 時間）を定数で置き、`PR()` に `now time.Time` を足す。規則 2 / 3 は `ai-assess:requested` が無いときだけ当て、時間切れならその他（`PR #<n> は人のコメントに AI が応答していない`）を返す。規則 7 は時間切れなら飛ばす
- [x] 1.2 `internal/classify/card.go`: `Card()` に `now` を足し、`PR()` へ渡す
- [x] 1.3 `internal/classify` のテスト: 既存の `PR()` / `Card()` 呼び出しに `now` を渡す。規則 2 / 3 / 7 の時間切れと、`ai-assess:requested` があれば規則 7 で扱う Scenario を足す。fixture テストは各 PR の `UpdatedAt` を `now` にする

## 2. 取得と呼び出し側

- [x] 2.1 `internal/fetch/fetch.go`: `Fetch` と `buildCards` に `now` を足し、`classify.Card` へ渡す
- [x] 2.2 `internal/fetch` のテスト: `Fetch` に固定の `now` を渡す。`testdata/link` の PR 90 で「now が分類に渡る」Scenario を足す
- [x] 2.3 `cmd/loop-cli/main.go`: fetcher で `time.Now()` を渡す。`cmd/loop-cli` / `internal/snapshot` / `internal/ui` のテストの `Fetch` 呼び出しに `now` を渡す
- [x] 2.4 `cmd/loop-cli-dev/classify.go`: 経過の列と同じ現在時刻を `classify.PR` に渡す
- [x] 2.5 `docs/domain/issue-driven-sdd/human-turn-signals.md`: 行 C の条件と「キューに入れないもの」に時間切れを書き足し、変更履歴に 1 行足す

## 3. 確認

- [x] 3.1 `go build ./... && go vet ./... && go test ./...` が通る
- [x] 3.2 `openspec validate s32-stale-in-progress-pr --strict` が通る
