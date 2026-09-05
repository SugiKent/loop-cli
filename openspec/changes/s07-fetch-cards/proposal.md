## Why

s03 の `GHClient`、s05 の `internal/model` / `internal/classify` は揃ったが、live の GitHub から「設定した全リポジトリの open issue / open PR を取り、分類に必要な詳細だけを遅延取得し、Issue と PR を紐づけて `model.Card` にする」層がまだ無い。s05 は「組み立て済みの `Card` と詳細取得済みの `Issue` / `PR` を受け取る」前提で書かれ、D-001 の遅延取得の範囲と、s05 が引き継いだ「D-001 の取得範囲と除外規則 2 の不整合」を決める change がこれである。
この change は docs/mvp/implementation-tasks.md §3 (P1) の項目「横断取得」（search 2 回と D-001 の遅延詳細取得、PR title / body のパースによる Issue と PR の紐づけとカード化）を実装する。s08 のキュー画面はこの change の返り値をそのまま表示する。

## What Changes

- `internal/fetch` を新設する。`Fetch(ctx, client gh.GHClient, repos []string) (*Result, error)` が、D-001 の呼び出し一覧どおり `SearchIssues` / `SearchPRs` を 1 回ずつ実行し、分類に必要な詳細（`question` の issue → `ViewIssue`、全 open PR → `ViewPR`、merge 候補 PR → `ViewPRMergeState`、`apply` PR → `ReviewThreads`）だけを遅延取得し、PR の title `[<段階>] #<n>` と本文 `Refs #n` / `Closes #n` のパースで同一リポジトリの Issue に PR を紐づけ、`classify.Card` で分類済みの `[]model.Card` を返す
- 遅延取得の対象条件を s05 `human-turn-classify` の各局面と進行中の規則の条件から導いて固定する。PR のコメントは D-001 の表（`question` の PR のみ）より広く全 open PR で取り、進行中の規則 2「`question` 無しの open PR で最新コメントが人」を live で成立させる（auto-fix 作業中の段階 PR が merge 候補として今やるに出るのを防ぐ。REST 5,000 req/時に対し +1 回 / PR で余裕）。D-001 の表に行を足す docs 更新は design.md 未決事項に記す。s05 が引き継いだ不整合はこの決定で閉じる
- `gh` 呼び出し 1 回ごとに `context.WithTimeout` で 30 秒の期限を付け（s03 design.md が s07 に委ねた値）、詳細取得は並行で行う（上限 4 と search の直列は design.md の既定値）
- 失敗の扱いを 2 段にする。search の失敗と `ctx` のキャンセル / 期限切れは `Fetch` のエラー（Card を返さない。D-002「失敗時は前回結果を維持してエラーを表示」の保持と表示は s08 / s13）。詳細取得 1 件の失敗は部分失敗として `Result.Errors` に積み、その Issue / PR は詳細 nil のまま Card に残す（黙って落とさない）
- Issue に紐づかない PR（`docs` PR、その他バケットの PR、紐づけ先の issue が open issue に無い PR）は `Issue` が nil の PR 単独カードにする。同一 Issue に同段階の open PR が複数あれば全部をカードに入れ、段階順 → 番号順に並べる（正本の印 `Canonical` は s05 の規則どおり `MERGED` PR にしか付かず、search は open PR しか返さないので P1 では立たない。merge 済み PR を持ち込む cross-reference は s17）
- `internal/gh/fake.go` の `Calls` への追記を `sync.Mutex` で守り、`gh-fake` に「並行呼び出しでも `Calls` の内容を壊さない」Requirement を足す（`Fetch` が `ViewIssue` を並行して呼ぶ最初の利用者になる）
- テストは s03 の `Fake` で `internal/gh/testdata/fixtures/example/` を読む fixture テストと、`internal/fetch/testdata/` に置く s07 専用の fixture（2 リポジトリで同じ番号・PR 単独・詳細欠落）で行う。s05 `fixture_test.go` と s06 `classify` の「全詳細を入れる」組み立ては置き換えない（design.md 未決事項）

## Capabilities

### New Capabilities
- `card-fetch`: `internal/fetch` が設定リポジトリの open issue / open PR を search 2 回で取り、分類に必要な詳細だけを遅延取得し、PR を Issue に紐づけて分類済みの `model.Card` 群を返す。タイムアウト・並行度・部分失敗の扱いを含む

### Modified Capabilities
- `gh-fake`: ADDED Requirement「Fake は並行呼び出しでも Calls の内容を壊さない」を足す（`go test -race` で検証）。既存の Requirement（記録する内容・直列呼び出し時の順序）は変えない
- `gh-client` / `card-model` / `human-turn-classify` の Requirement は変えない

## Impact

- 新規: `internal/fetch/fetch.go` / `link.go` / `fetch_test.go` / `link_test.go` / `testdata/`
- 依存: `internal/gh`（`GHClient` / `Fake`）、`internal/model`（変換関数・`HasLabel` / `PRStages` / `ParseUndecided`）、`internal/classify`（`Card`）。標準ライブラリのみで新しい外部依存は無い
- 後続 change への影響: s08 が `config.Load` → `gh.NewClient` + `Check` → `fetch.Fetch` を `cmd/sugi-loop` に配線し、`Result.Cards` をタブ別に並べ、`Result.Errors` と `Fetch` のエラーをステータスバーに出す（s02 / s03 の design は配線を s07 と書いているが、この change は配線をせず s08 に渡す）。s13 は `Fetch` を定期実行して差分比較する。s17 は `CrossReferencedPRs` による紐づけ補完を `internal/fetch` に足す。s18 は 1 件再取得を足す
- 既存コードの変更: `internal/gh/fake.go` の `Calls` への追記を `sync.Mutex` で守る（`Fetch` が `ViewIssue` を並行して呼ぶ最初の利用者になり、s03 の `Fake` は直列呼び出し前提で書かれているため。`gh-fake` の既存 Requirement は変えず、ADDED で並行安全の Requirement を足す）
