## ADDED Requirements

### Requirement: Client は fixture 用に読み取りコマンドの標準出力を採取する
`Client` はメソッド `Capture(ctx context.Context, repo string, progress func(name string)) (map[string][]byte, error)` を MUST 提供する。`repo` は `owner/name` 形式 1 件。返り値は s03 `gh-fake` の fixture ファイル名をキー、そのコマンドの標準出力（デコードせず生のバイト列）を値にした map である。
採取の順序と範囲は次のとおり。
1. `SearchIssues` と同じ引数（`--repo` は `repo` 1 つ）で `gh search issues` を実行し、`search-issues.json` に入れる。出力を s03 のデコード関数で読んで issue 番号の一覧を得る
2. 各 issue 番号 `n` について順に、`ViewIssue` / `CrossReferencedPRs` / `LabelTimeline` と同じ引数で実行し、`issue-<n>.json` / `issue-<n>-cross-refs.json` / `issue-<n>-timeline.json` に入れる
3. `SearchPRs` と同じ引数で `gh search prs` を実行し、`search-prs.json` に入れる。PR 番号の一覧を得る
4. 各 PR 番号 `n` について順に、`gh pr view <n> -R <repo> --json number,title,body,url,labels,isDraft,comments,mergeable,mergeStateStatus,statusCheckRollup,reviewDecision`（`ViewPR` と `ViewPRMergeState` のフィールドを合わせた 11 フィールド。s03 `gh-fake` の `pr-<n>.json` の定義どおり）と、`ReviewThreads` と同じ引数で実行し、`pr-<n>.json` / `pr-<n>-review-threads.json` に入れる

`pr view` 以外の引数は対応する読み取りメソッドと同一でなければならない（引数の組み立てを共有する）。`pr view` は上記 11 フィールドの 1 回の出力で、`ViewPRMergeState` と同じ「`mergeable` が `UNKNOWN` なら `RetryWait`（既定 2 秒）後に 1 回だけ再取得」を通った後の標準出力を入れる（再取得の処理を `ViewPRMergeState` と共有する。fixture が `UNKNOWN` だらけになると局面 C を fixture でテストできないため）。map のキーは `Fake` がファイルを探す名前と同じ関数から組み立てる。
`progress` は各ファイルの取得直前に 1 回、これから入れるファイル名を引数に呼ぶ（呼び出し側が進捗を表示するため。`pr view` の UNKNOWN 再取得で `gh` が 2 回走っても呼び出しは 1 回）。いずれかの実行が失敗したら、その時点のエラー（s03 の `*Error` またはデコードエラー）を返し、それ以降は実行しない。

#### Scenario: issue 1 件と PR 1 件のリポジトリを採取する
- **WHEN** 実行関数を差し替え、`search issues` に issue 108 だけの配列、`search prs` に PR 131 だけの配列、その他の引数には空でない JSON を返す状態で `Capture(ctx, "org/app", progress)` を呼ぶ
- **THEN** map のキーは `search-issues.json` / `issue-108.json` / `issue-108-cross-refs.json` / `issue-108-timeline.json` / `search-prs.json` / `pr-131.json` / `pr-131-review-threads.json` の 7 つで、各値は実行関数が返した標準出力のバイト列と一致する。`progress` はこの 7 つの名前をこの順で受け取る

#### Scenario: 引数は読み取りメソッドと同じである
- **WHEN** 上記の状態で `Capture` を呼び、実行関数が受け取った引数列を記録する
- **THEN** 1 回目は `search issues --repo org/app --state open --limit 200 --json repository,number,title,labels,updatedAt,url,body,commentsCount`、issue 108 の 3 回は `ViewIssue(ctx, "org/app", 108)` / `CrossReferencedPRs` / `LabelTimeline` が発行する引数と同一、`search prs` は `SearchPRs` の引数と同一、PR 131 の 1 回目は `pr view 131 -R org/app --json number,title,body,url,labels,isDraft,comments,mergeable,mergeStateStatus,statusCheckRollup,reviewDecision`、2 回目は `ReviewThreads` の引数（`api graphql -f owner=org -f name=app -F number=131 -f query=<s03 のクエリ文字列>`）と同一である

#### Scenario: pr view の UNKNOWN は再取得後の出力を入れる
- **WHEN** `RetryWait` を 0 にし、実行関数が `pr view 131` の 1 回目に `mergeable: "UNKNOWN"`、2 回目に `mergeable: "MERGEABLE"` を返す状態で `Capture` を呼ぶ
- **THEN** `pr view 131`（11 フィールド）は 2 回実行され、`pr-131.json` の内容は 2 回目の標準出力である。`mergeable: "MERGEABLE"` を 1 回目に返した場合は 1 回しか実行されない

#### Scenario: open issue も open PR も無い
- **WHEN** 実行関数が `search issues` と `search prs` に空配列 `[]` を返す状態で `Capture` を呼ぶ
- **THEN** map のキーは `search-issues.json` と `search-prs.json` の 2 つだけで、実行関数は 2 回しか呼ばれない

#### Scenario: 途中の gh 失敗で止まる
- **WHEN** 実行関数が `issue view 108` に終了コード 1 と stderr を返す状態で `Capture` を呼ぶ
- **THEN** `*Error` が返り、`issue view 108` より後の引数（cross-refs / timeline / `search prs`）で実行関数は呼ばれない
