## 1. 前提と依存

- [x] 1.1 s13-auto-refresh-notify までが archive 済みで、`gofmt -l .` が空、`go build ./... && go vet ./... && go test ./...` と `golangci-lint run ./...` が通り、`openspec validate --all --strict` がグリーンであることを確認する。この change が MODIFIED / REMOVED する Requirement（`card-fetch` の 3 件、`card-detail` の 3 件、`human-turn-classify` の 2 件、`answer-question` / `snapshot-cache` / `fixture-capture` の各 1 件）を本流の specs と突き合わせ、本文がレビューで変わっていれば写し直す
- [x] 1.2 `internal/classify/classify.go` の `Issue` / `isC` / `isD` と `internal/classify/card.go` を読み、design.md の D-1 の前提（コメントを読む分岐は `question` で、`isC` は「段階ラベル 1 件以上・`question` 無し・未確定 0 件」で、`isD` は `apply` ラベルで、それぞれ詳細を読む前に守られている。`classify.Card` は詳細フィールドを直接読まない）が現在のコードで成り立っていることを確認する。成り立っていなければ実装を止めて、分類が変わる条件を報告する

## 2. テスト fixture の補完

- [x] 2.1 不足ファイルを足す。`Fake` はディレクトリと番号だけでファイル名を決める（`repo` を見ない）ので、必要なのは次のとおり。中身は design.md 未決事項 5 の既定値に従い、コメント 0 件・thread 0 件の最小の JSON にする。**`pr-<n>-review-threads.json` は `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[]}}}}}` の形で書く**（`decodeReviewThreads` はこの GraphQL の入れ子に `json.Unmarshal` する。裸の `[]` を置くと unmarshal が失敗し、thread 0 件のつもりのファイルが取得失敗として扱われる）。他のファイルの形は `internal/gh/testdata/fixtures/example` の同名ファイルに合わせる
  - `internal/fetch/testdata/link`: `issue-108.json` と、`pr-131` / `pr-132` / `pr-151` / `pr-60` / `pr-61` / `pr-62` の `-review-threads.json` 6 件
  - `internal/fetch/testdata/multirepo`: `issue-12.json` と `pr-30-review-threads.json`
  - `internal/fetch/testdata/samestage`: `pr-131-review-threads.json` と `pr-140-review-threads.json`
  - `internal/fetch/testdata/nosearch`: 追加なし（search の失敗を検証するディレクトリ）
  - `internal/fetch/testdata/partial`: **追加しない。** `pr-131.json` と `pr-131-review-threads.json` が無いことが「詳細取得の失敗」を作る
  - `internal/gh/testdata/fixtures/example`: 追加なし（issue 108 / 140 と PR 131 の全ファイルが揃っていることを確認するだけ）
- [x] 2.2 `multirepo` は `org/app#12` と `org/web#12` が同じ `issue-12.json` を読むことを確認し、リポジトリごとに違うコメントを前提にした fixture を書かない。`TestFetchMultiRepo` は現在 `res.Errors` を検査していないので、`Errors` が空であることのアサーションを足す（fixture の足し忘れを黙って通さないため）
- [x] 2.3 fixture を足した時点で `go test ./internal/fetch/` を実行し、**既存のテストが見ている `Situation` の期待値が 1 つも変わらないこと**を確認する（この時点では実装は変えていないので、追加ファイルは読まれず全て通るはず）

## 3. internal/fetch の全件取得

- [x] 3.1 `internal/fetch/fetch.go` の `fetchDetails` から条件分岐を外す。issue のループから `question` の判定を、PR のループから `mergeCandidate` と `isApply` の判定を消し、すべての issue に `ViewIssue`、すべての PR に `ViewPR` / `ViewPRMergeState` / `ReviewThreads` を `run` で回す。`fail` の書式（`"%s %s#%d: %w"`）・`detailConcurrency`（4）・`CallTimeout`（30 秒）は変えない。使われなくなった `isMergeCandidate` を消す
- [x] 3.2 `detailError` にメソッド名のフィールドを足し、`sortErrors` のキーを `(isPR, repo, number)` から `(isPR, repo, number, メソッド順)` にする。メソッド順は `ViewPR` → `ViewPRMergeState` → `ReviewThreads`（issue 側は `ViewIssue` の 1 種類だけ）。1 つの PR が 3 件のエラーを出しても並びが実行ごとに変わらないようにする。`Errors` の要素は `error` のままで、文字列の書式は変えない
- [x] 3.3 `internal/fetch/fetch_test.go` の既存アサーションを全件取得に合わせて直す。落ちるのは次の箇所（行番号は現状のもの。実際の位置は確認する）
  - 冒頭の example テスト: `viewed == []int{108}` → `{108, 140}`、`second.Issue.Comments != nil` の反転（issue 140 は長さ 0 の非 nil になる）、`pr.MergeState != nil`・`pr.ReviewThreads != nil` の反転
  - `TestFetchLink`: `pr151.ReviewThreads != nil`、`pr140.MergeState != nil`、`pr132.MergeState != nil`、`pr90.MergeState != nil` の各アサーション
  - `TestFetchSameStage`: `pr88.MergeState != nil` のアサーション
- [x] 3.4 delta の Scenario に対応するテストを足す
  - 「全 issue に ViewIssue が呼ばれる」: `example` で `Fetch` を呼び、`Fake.Calls` の `ViewIssue` が issue 108 と 140 の 2 件で、issue 140 の `Comments` が長さ 0 の非 nil であること
  - 「question 付き PR でも merge 状態と review thread を取る」: `example` の PR 131 の `MergeState` が non-nil で `Mergeable` が `UNKNOWN`、`ReviewThreads` が non-nil で 1 件であること
  - 「詳細が埋まっても分類結果は変わらない」: 未確定 2 件の `propose` PR 132 で `MergeState` が non-nil・`ReviewThreads` が長さ 0 の非 nil・`Situation` が `other` であること
  - 「question 無しの issue のコメントは分類に影響しない」: ラベル無しでコメント末尾が人の issue の `Situation` が `E` であること
- [x] 3.5 部分失敗のテストを delta に合わせる。`testdata/partial` で `Fetch` を呼ぶと `Errors` が 3 件で、`ViewPR` → `ViewPRMergeState` → `ReviewThreads` の順に並び、いずれも `org/app` と `131` を含み、PR 131 の `Comments` / `MergeState` / `ReviewThreads` がすべて nil で `Situation` が `other`、issue 108 の `Comments` は埋まっていることを検証する
- [x] 3.6 `go test ./internal/fetch/ ./internal/classify/ ./internal/gh/` が通ることを確認する。`internal/classify` と `internal/gh` のテストは 1 行も変えずに通るはずで、変えないと通らないなら D-1 の前提が崩れているので止めて報告する

## 4. internal/ui の表示

- [x] 4.1 `internal/ui/detail.go` の `未取得` を `取得失敗` に置き換える。対象は 4 か所: カード詳細の PR 行（`checks 未取得 mergeable 未取得` → `checks 取得失敗 mergeable 取得失敗`）、`commentSection`、`checkLines`、review thread。行の構造・並び・`なし` の表示は変えない。コード中の説明コメント（「s07 は merge 候補にしか取らない」など）も、nil が取得失敗を意味するようになったことに合わせて直す
- [x] 4.2 `internal/ui/detail_test.go` の `example` 由来のケースを**実値に**直す。これらは `exampleResult(t)`（実際の `fetch.Fetch`）を使うので、文言を `取得失敗` に置き換えるだけでは落ちる
  - カード詳細の PR 131 の行: `checks 未取得 mergeable 未取得` → `checks 緑以外 mergeable UNKNOWN`（`pr-131.json` は `mergeable: UNKNOWN`、`statusCheckRollup` に `StatusContext ci/legacy PENDING` があり `ChecksGreen` が false）
  - カード詳細の issue 140: `コメント: 未取得` → `コメント: なし`（`issue-140.json` は `comments` が空配列）
  - PR 詳細の PR 131: `checks: 未取得` → `mergeable: UNKNOWN BLOCKED` と `test: SUCCESS` と `ci/legacy: PENDING`、`review thread: 未取得` → `thread 未 resolve`
- [x] 4.3 `取得失敗` を検証するケースを、`MergeState` / `Comments` / `ReviewThreads` を `nil` にした**手書きの Card** で足す（既存の「同段階の merge 済み PR は最新に印が付く」と同じ組み立て方が使える）。カード詳細の PR 行・カード詳細の issue コメント・PR 詳細の 3 箇所を覆う
- [x] 4.4 `go test ./internal/ui/` が通ることを確認する

## 5. docs と README

- [x] 5.1 `docs/mvp/decisions.md` の D-001 に追記する。既存の「2026-09-05-1805 追記」と同じ形で、遅延取得表を全件取得に置き換えたこと、その理由（未取得が並ぶと確認の手数が減らない）、design.md D-5 の見積もり（1 更新あたり search 2 回は REST search 枠のまま、詳細は GraphQL 枠で `N + 3M`。1 呼び出し 1 point 換算・`refresh_interval_sec` 120 秒なら `N + 3M ≤ 164` が境目）を書く。`## 変更履歴` に相当する表があればそこにも 1 行足す
- [x] 5.2 `README.md` を読み、取得の範囲や「未取得」に触れている記述があれば実装に合わせて直す。無ければ何もしない（無理に足さない）

## 6. 最終確認

- [x] 6.1 `gofmt -l .` が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する（`golangci-lint run ./...` も通す）。`openspec validate --all --strict` がグリーンであることを確認する
- [ ] 6.2 利用者が稼働リポジトリで起動し、次を見る。カード詳細と PR 詳細のどこにも `未取得` が出ないこと。`gh` を止めた状態で起動すると `取得失敗` が出ること。キューが出るまでの待ち時間が実用の範囲で、自動更新の間隔（既定 120 秒）で取得が終わっていること。あわせて `gh api graphql -f query='{rateLimit{cost remaining resetAt}}'` を取得の前後で実行し、1 更新あたりの GraphQL 消費を実測して design.md D-5 の見積もり（1 呼び出し 1 point 換算）と突き合わせる
