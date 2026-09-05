## Why

カード詳細と PR 詳細に `checks 未取得` / `mergeable 未取得` / `review thread: 未取得` / `コメント: 未取得` が並び、一覧を見ても「結局 GitHub を開いて確かめないと分からない」状態が残っている。sugi-loop は「人の出番」を横断して 1 本のキューにまとめ、確認の手数を減らす道具なので、未取得が並ぶこと自体が目的を損なう。

未取得になるのは `card-fetch` の Requirement「分類に必要な詳細だけを遅延取得する」が、分類に要る条件に当たる issue / PR にしか詳細を取りに行かないため。例えば `propose` + `question` が付き本文が「未確定の判断: 2 件」の PR は merge 候補にも `apply` にも当たらないので、checks も mergeable も review thread も取られない。表示は正直だが、利用者から見れば「調べていないから分からない」が画面に出続ける。

## What Changes

- `Fetch` が詳細取得の条件分岐をやめ、search で得た **すべての** open issue / open PR について詳細を取る。
  - すべての issue に `ViewIssue`（`Issue.Comments`）
  - すべての PR に `ViewPR`（`PR.Comments`。現状も全件なので変更なし）
  - すべての PR に `ViewPRMergeState`（`PR.MergeState`）
  - すべての PR に `ReviewThreads`（`PR.ReviewThreads`）
- 詳細を取りに行った結果、値が得られなかったこと（`gh` の失敗・タイムアウト）と、値が無いこと（コメント 0 件・thread 0 件）を画面で区別する。取りに行かなかったという第 3 の状態が無くなるため、`nil` は「取得に失敗した」だけを意味するようになる。
- カード詳細・PR 詳細の `未取得` の文言を `取得失敗` に置き換える。`なし`（長さ 0）の表示は変えない。
- **分類結果は変わらない。** `classify.Issue` のコメント参照は `question` ラベルで、`classify.isC` は merge 候補の条件で、`classify.isD` は `apply` ラベルでそれぞれ守られており、いずれも現在の取得条件と一致する。今まで `nil` だった詳細が埋まっても、そこを読む分岐に入らない。この change は表示を変えるだけで、キューの並びとタブは変わらない。
- D-001「データ取得は『広く 2 回検索してクライアント側で分類する』」の遅延取得表が、この change で実態と食い違う。decisions.md に追記して、取得の範囲と、その代わりに増える呼び出し回数の見積もりを記録する。

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `card-fetch`: Requirement「分類に必要な詳細だけを遅延取得する」を、条件分岐を持たない全件取得の Requirement に置き換える。取得しない issue / PR が無くなり、`Comments` / `MergeState` / `ReviewThreads` が `nil` になるのは詳細取得が失敗したときだけになる。あわせて `Errors` の並びに「同じ (repo, number) の中はメソッド順」を足す（1 つの PR が 3 件のエラーを出し得るため）
- `card-detail`: カード詳細の PR 行、カード詳細の issue コメント、PR 詳細の checks / 会話コメント / review thread の各表示で、`nil` のときの文言を `未取得` から `取得失敗` に変える。`example` を使う Scenario は詳細が埋まった実値に直し、`取得失敗` は `nil` を手書きした Card の Scenario で検証する
- `human-turn-classify`: 入力の前提の記述を「D-001 の遅延取得に従って詳細が入る」から「全件入り、`nil` は取得失敗だけ」に直す。判定の内容は変えない
- `answer-question`: 「対象の `Comments` が nil（未取得）ならテンプレートは空」の前提と、issue 140 を `Comments` nil として扱う Scenario を、全件取得後の実態（長さ 0 の非 nil）に直す
- `snapshot-cache`: `nil` のスライスの説明にある表示文言を `未取得` から `取得失敗` に直す
- `fixture-capture`: 採取範囲の説明を「分類に必要な詳細」から「全件の詳細」に直す（実装は既に全件採取しており、文言だけの不一致）

## Impact

- `internal/fetch/fetch.go`: `fetchDetails` の条件分岐（`question` 判定・`isMergeCandidate`・`apply` 判定）を外す。`isMergeCandidate` は分類側に同等の判定があり `fetch` からは不要になる。`detailError` にメソッド名を持たせ、`sortErrors` のキーに足す
- `internal/fetch/fetch_test.go`: 取得条件を検証している既存ケースを、全件取得の検証に書き換える
- `internal/fetch/testdata/`: 全件取得にすると `Fake` が読むファイルが増えるため、`link` / `multirepo` / `samestage` に不足分を足す（`partial` は失敗の検証用なので意図的に足さない）
- `internal/ui/detail.go`: `未取得` を出す 4 か所（カード詳細の PR 行の `checks 未取得 mergeable 未取得`、`commentSection`、`checkLines`、review thread）の文言
- `internal/ui/detail_test.go`: 上記の期待値
- `docs/mvp/decisions.md`: D-001 の遅延取得表への追記
- `README.md`: 取得の範囲に触れている記述があれば実装に合わせる
- 1 回の更新あたりの `gh` 呼び出しが増える。open が issue N 件 / PR M 件のとき、search 2 回 + N + 3M 回になる（従来は search 2 回 + 条件に当たった分 + M 回）。増えるのは GraphQL 枠だけで、search 枠は 2 回のまま。見積もりと実測の方法は design.md の D-5
- 分類は変わらないが、詳細を読んでいる表示は変わる。`question` 無しの issue にも blocked-by 節・プレビューのコメント・`a` の回答テンプレートが出るようになる（design.md の D-6）
