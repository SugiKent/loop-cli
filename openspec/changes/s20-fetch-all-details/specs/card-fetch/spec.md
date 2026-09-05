## MODIFIED Requirements

### Requirement: Fetch は設定リポジトリに対して search を 2 回だけ実行し分類済みの Card 群を返す
`internal/fetch` は `Result { Cards []model.Card; Errors []error }` と、関数 `Fetch(ctx context.Context, client gh.GHClient, repos []string) (*Result, error)` を MUST 提供する。`repos` は `owner/name` の列（s02 の `Config.Repos[].Name`。`Fetch` は `internal/config` を import しない）。
`Fetch` は 1 回の呼び出しで `client.SearchIssues(ctx, repos)` と `client.SearchPRs(ctx, repos)` をそれぞれ 1 回だけ実行し（D-001「1 回の更新で行う呼び出し」）、得た open issue / open PR を `model.IssueFromSearch` / `model.PRFromSearch` で `model` の型に写し、Requirement「Fetch は open の全 issue / 全 PR の詳細を取得する」の詳細を入れ、Requirement「PR は title と本文のパースで同一リポジトリの Issue に紐づく」で `model.Card` を組み立て、各 Card を s05 の `classify.Card` に通して `Result.Cards` に入れる。`Result.Cards` の各要素は `Card.Result` / `Issue.Result` / `PRs[i].Result` が埋まった状態で返る（s08 は分類を呼び直さない）。
`Cards` の並びは、issue を持つカードを `SearchIssues` の返却順、続けて PR 単独カードを `SearchPRs` の返却順とする。タブ内の並び替えは s08 が `Card.Result.Tab` / `Priority` で行う。
search 結果の全 issue と全 PR は、それぞれちょうど 1 枚の Card に含まれる。`Fetch` は issue / PR を黙って落とさない。

#### Scenario: example の fixture から Card 2 枚が返る
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` を `client` に、`[]string{"org/app"}` を `repos` に渡して `Fetch` を呼ぶ（`example` は issue 108（`stage:propose` + `question`）、issue 140（ラベル無し）、PR 131（`propose` + `question`、本文に `Closes #108`）を持つ）
- **THEN** エラー無しで `Result` が返り、`Errors` は空、`Cards` は 2 枚で、1 枚目は `Issue.Number` 108 かつ `PRs` が PR 131 の 1 件、2 枚目は `Issue.Number` 140 かつ `PRs` が空である。1 枚目の `Card.Result.Situation` は `A`（`PRs[0].Result.Situation` が `A`、`Issue.Result.Situation` が `in-progress`）、2 枚目の `Card.Result.Situation` は `E` である

#### Scenario: search は 1 回ずつしか呼ばれない
- **WHEN** `SearchIssues` / `SearchPRs` の呼び出し回数を数える `GHClient` で `Fetch` を呼ぶ
- **THEN** どちらもちょうど 1 回呼ばれる

#### Scenario: 全 issue と全 PR がちょうど 1 枚の Card に含まれる
- **WHEN** `internal/gh/testdata/fixtures/` 直下の各 `<alias>` について `gh.NewFake` で `Fetch` を呼ぶ
- **THEN** エラー無しで返り、`Errors` は空で、`search-issues.json` の各 issue 番号は `Cards[].Issue` にちょうど 1 回、`search-prs.json` の各 PR 番号は `Cards[].PRs` にちょうど 1 回現れる（`Situation` の期待値は s05 の期待値表が持ち、このテストでは見ない）

### Requirement: search の失敗と ctx の中断はエラー、詳細取得の失敗は部分失敗として Card を残す
`Fetch` は次の失敗の扱いを MUST 守る。
- `SearchIssues` または `SearchPRs` がエラーを返したら、そのエラーを（どちらの search かが分かるよう `search issues: ` / `search prs: ` を前置して）返し、`Result` は nil。D-002「失敗時は前回結果を維持してエラーを表示」の「前回結果の維持」と表示は s08 / s13 が行う
- 詳細取得の途中で引数の `ctx` がキャンセルまたは期限切れになったら、`ctx.Err()` を含むエラーを返し、`Result` は nil（途中までの Card でスナップショットを上書きしないため）
- 詳細取得 1 件が上記以外の理由で失敗したら、その issue / PR を落とさず、対応する詳細を nil のまま Card に残し、`Result.Errors` に「メソッド名・`owner/name`・番号」を含むエラー（元のエラーを `%w` で包む）を追加する。他の詳細取得は続ける。`Errors` の並びは issue のエラーを (repo, number) 順、続けて PR のエラーを (repo, number, メソッド) 順（メソッドは `ViewPR` → `ViewPRMergeState` → `ReviewThreads` の順）とする。1 つの PR が複数の詳細で失敗すると同じ (repo, number) のエラーが 3 件まで並ぶので、その中の順序もメソッドで決める（並行実行の完了順に依存しない）
- 詳細取得がすべて成功すれば `Errors` は空（nil または長さ 0）

#### Scenario: search issues の失敗
- **WHEN** `search-issues.json` が無い fixture ディレクトリで `Fetch` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `search issues` と `search-issues.json` を含み、`Result` は nil である

#### Scenario: 1 つの PR の詳細がすべて失敗しても Card を落とさない
- **WHEN** `search-prs.json` に `propose` + `question` の PR 131（本文 `Closes #108`）があるが `pr-131.json` と `pr-131-review-threads.json` が無く、issue 108 は `issue-108.json` がある fixture で `Fetch` を呼ぶ
- **THEN** エラー無しで `Result` が返り、issue 108 の Card の `PRs` に PR 131 があって `Comments` と `MergeState` と `ReviewThreads` はいずれも nil、`PRs[0].Result.Situation` は `other`、`Errors` は 3 件で、それぞれの文字列に `ViewPR` / `ViewPRMergeState` / `ReviewThreads` のいずれかと `org/app` と `131` を含む。issue 108 の `Comments` は埋まっている

#### Scenario: ctx のキャンセルは部分結果を返さない
- **WHEN** search は成功し（`SearchPRs` が結果を返す直前に親 `ctx` の `cancel()` を呼ぶ）、`ViewIssue` が `ctx.Done()` まで待ってから `ctx.Err()` を返す `GHClient` に対して `Fetch` を呼ぶ
- **THEN** エラーが返り、`errors.Is(err, context.Canceled)` が真で、`Result` は nil である

## ADDED Requirements

### Requirement: Fetch は open の全 issue / 全 PR の詳細を取得する
`Fetch` は、search で得た **すべての** open issue と open PR について、ラベル・本文・`IsDraft` を問わず詳細を MUST 取得する。取得しない対象を選ぶ条件を持たない。

1. すべての open issue → `client.ViewIssue` を呼び、`Comments` を `model.CommentFrom` で `Issue.Comments` に入れる
2. すべての open PR → `client.ViewPR` を呼び、`Comments` を `PR.Comments` に入れる
3. すべての open PR → `client.ViewPRMergeState` を呼び、`PR.MergeState` に入れる
4. すべての open PR → `client.ReviewThreads` を呼び、`PR.ReviewThreads` に入れる

したがって `Issue.Comments` / `PR.MergeState` / `PR.ReviewThreads` が nil で返るのは、その詳細取得が失敗したときだけである（Requirement「search の失敗と ctx の中断はエラー、詳細取得の失敗は部分失敗として Card を残す」）。取得に成功して中身が空なら、長さ 0 の非 nil が入る。表示側（s09 `card-detail`）はこの nil を「取得失敗」として出す。

search 結果の `Labels` / `Body` / `IsDraft` は詳細で上書きしない。この Requirement は分類の入力を変えない。`classify.Issue` がコメントを読む分岐は `question` ラベルで、`classify.isC` は「段階ラベル 1 件以上・`question` 無し・未確定 0 件」で、`classify.isD` は `apply` ラベルでそれぞれ守られており、これまで詳細を取っていなかった対象は、埋まった詳細を読む前に分岐から外れる。

#### Scenario: 全 issue に ViewIssue が呼ばれる
- **WHEN** `example` で `Fetch` を呼んだ後に `Fake.Calls` を見る（`example` は `question` 付きの issue 108 と、ラベル無しの issue 140 を持つ）
- **THEN** `Method` が `ViewIssue` の `Call` は `Number` 108 と 140 の 2 件で、issue 108 の `Comments` は `issue-108.json` のコメント 2 件、issue 140 の `Comments` は `issue-140.json` のコメント（長さ 0 なら非 nil の長さ 0）である

#### Scenario: question 付き PR でも merge 状態と review thread を取る
- **WHEN** `example` で `Fetch` を呼ぶ（PR 131 は `propose` + `question`、`pr-131.json` は `mergeable: UNKNOWN`、`pr-131-review-threads.json` がある）
- **THEN** PR 131 の `Comments` は `pr-131.json` のコメント、`MergeState` は non-nil で `Mergeable` が `UNKNOWN`、`ReviewThreads` は `pr-131-review-threads.json` の内容で、いずれも nil ではない

#### Scenario: 詳細が埋まっても分類結果は変わらない
- **WHEN** `Labels` が `propose`、`Body` の 1 行目が `未確定の判断: 2 件` の PR 132（`pr-132.json` は `mergeable: MERGEABLE` で AI のコメント 1 件、`pr-132-review-threads.json` は空）がある fixture で `Fetch` を呼ぶ
- **THEN** `Errors` は空で、PR 132 の `MergeState` は non-nil、`ReviewThreads` は長さ 0 の非 nil で、`Result.Situation` は `other` である（未確定が 0 件でないため `isC` は `MergeState` を読む前に false になる）

#### Scenario: question 無しの issue のコメントは分類に影響しない
- **WHEN** `Labels` が空で `Comments` の末尾が人である issue 140 を含む fixture で `Fetch` を呼ぶ
- **THEN** issue 140 の `Comments` は埋まっており、`Result.Situation` は `E` である（`question` が無いので規則 4 の分岐に入らない）

## REMOVED Requirements

### Requirement: 分類に必要な詳細だけを遅延取得する
**Reason**: 取得しない対象があると `Comments` / `MergeState` / `ReviewThreads` が nil のまま残り、カード詳細と PR 詳細に `未取得` が並ぶ。一覧を見ても「調べていないから分からない」が残り、確認の手数が減らない。
**Migration**: Requirement「Fetch は open の全 issue / 全 PR の詳細を取得する」が置き換える。取得条件を持たなくなるだけで、`Fetch` のシグネチャ・`Result` の形・分類結果・`Cards` の並びは変わらない。呼び出し側の変更は不要。
