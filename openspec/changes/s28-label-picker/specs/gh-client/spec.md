## ADDED Requirements

### Requirement: ListLabels はリポジトリで使えるラベルを名前順で返す

`internal/gh` の `GHClient` interface は `ListLabels(ctx context.Context, repo string) ([]RepoLabel, error)` を MUST 持つ（s03「`GHClient` interface が読み取りと書き込みのメソッドを定義する」が定めた「後続 change が必要とするメソッドは、その change が ADDED で Requirement を足して interface・`Client`・`Fake` を同時に拡張する」に従う）。`RepoLabel` は `Name string` と `Description string` と `Color string` を持ち、`gh label list` の JSON 出力をそのまま写す（s03「生の型は `gh` の JSON 出力を写す」）。

既存の `Label { Name string }` とは別の型にする。`Label` は issue / PR に付いているラベルを表し、`gh issue view --json labels` と GraphQL の `labels{nodes{name}}` の両方から作られるので `description` と `color` は取れない（s03 が「`id` / `color` / `description` は読み捨てる」と定めている）。`Label` に 2 つの欄を足すと、経路によって空になる欄を持つ型になり、`Fake` と `Client` で埋まり方が変わる。リポジトリが持つラベルの一覧は用途も取得元も違うので、型を分ける。

`Client` の `ListLabels` は `gh label list -R <repo> --json name,description,color --sort name --order asc --limit 100` を MUST 実行し、標準出力を `[]RepoLabel` にデコードして返す。`--sort name --order asc` を明示するのは、`gh` の既定が作成順であり、リポジトリごとに並びが変わると画面も fixture も非決定になるためである。並べ替えは `gh` に任せ、クライアント側では行わない。`--limit 100` を超えるラベルを持つリポジトリでは 100 件で切れるが、ページングは行わない。デコードに失敗したときは、コマンドと元のエラーを含むエラーを返す（s03「`gh` の失敗はコマンドと stderr を含むエラーになる」と同じ形）。

#### Scenario: 実行するコマンドと引数

- **WHEN** 実行を記録するスタブに差し替えた `Client` で `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** 記録された引数は `label list -R org/app --json name,description,color --sort name --order asc --limit 100` である

#### Scenario: 出力をデコードする

- **WHEN** 標準出力に `[{"name":"docs","description":".claude/ と docs/ だけの PR","color":"0075ca"},{"name":"wip","description":"","color":"ededed"}]` を返すスタブに差し替えた `Client` で `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** `RepoLabel` 2 件が順に返り、1 件目の `Name` は `docs`、`Description` は `.claude/ と docs/ だけの PR`、`Color` は `0075ca` であり、2 件目の `Name` は `wip` で `Description` は空文字列である

#### Scenario: デコードの失敗はコマンドを含むエラーになる

- **WHEN** 標準出力に `{` を返すスタブに差し替えた `Client` で `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `label list` と `decode` を含む

### Requirement: ラベルの一括編集は 1 回の gh 実行で付ける名前と外す名前を渡す

`internal/gh` の `GHClient` interface は次の 2 つを MUST 持つ。

- `EditIssueLabels(ctx context.Context, repo string, number int, add []string, remove []string) error`
- `EditPRLabels(ctx context.Context, repo string, number int, add []string, remove []string) error`

`Client` はそれぞれ `gh issue edit <number> -R <repo>` と `gh pr edit <number> -R <repo>` を**ちょうど 1 回**実行し、`add` の各要素を `--add-label <名前>`、`remove` の各要素を `--remove-label <名前>` として、`add` → `remove` の順に受け取った並びのまま引数へ並べる。フラグは 1 名前につき 1 回ずつ繰り返し、コンマで連結しない。`add` と `remove` がどちらも空のときは `gh` を実行せず nil を返す。

このメソッドは**ラベル集合の置換を行わない**。`--add-label` / `--remove-label` だけを使い、渡されなかったラベルには触れないので、他の書き手が付けたラベルは残る（不変条件 1 が守りたかったこと）。ただし `gh` はこの 1 プロセスの中で「付ける」と「外す」を別の mutation として送るので、**原子的ではない**。付けるほうが成功して外すほうが失敗すると、GitHub 側には半分だけ適用された状態が残り、返るのはエラーだけである。1 回の実行で複数のラベルが変わるため、GitHub は増えたラベルの数だけ `labeled` イベントを出す。

PR に対して `EditIssueLabels`（`gh issue edit`）を使ってはならない。`gh issue edit` は内部で `issueOrPullRequest` を引くため PR 番号でも通るが、この挙動は `gh` のヘルプに書かれておらず、`Fake` が `gh` を起動しない以上、壊れてもテストが気づけない。`internal/gh` の他の PR 操作（`ViewPR` / `CommentPR` / `MergePR`）がすべて `pr` サブコマンドを使っていることにも揃える。

既存の `AddLabel` / `RemoveLabel`（1 ラベルずつの `gh issue edit`）はそのまま残る。s11 `todo-toggle` の `t` はそちらを使い続ける。

#### Scenario: issue のラベルを 1 回の実行で足し引きする

- **WHEN** 実行を記録するスタブに差し替えた `Client` で `EditIssueLabels(ctx, "org/app", 108, []string{"docs", "wip"}, []string{"blocked"})` を呼ぶ
- **THEN** 実行はちょうど 1 回で、記録された引数は `issue edit 108 -R org/app --add-label docs --add-label wip --remove-label blocked` である

#### Scenario: PR のラベルは pr edit で書く

- **WHEN** 同じスタブで `EditPRLabels(ctx, "org/app", 131, []string{"docs"}, nil)` を呼ぶ
- **THEN** 実行はちょうど 1 回で、記録された引数は `pr edit 131 -R org/app --add-label docs` である

#### Scenario: 外すだけのときは add のフラグを出さない

- **WHEN** 同じスタブで `EditIssueLabels(ctx, "org/app", 108, nil, []string{"question"})` を呼ぶ
- **THEN** 記録された引数は `issue edit 108 -R org/app --remove-label question` である

#### Scenario: どちらも空なら gh を実行しない

- **WHEN** 同じスタブで `EditIssueLabels(ctx, "org/app", 108, nil, nil)` を呼ぶ
- **THEN** nil が返り、`gh` を実行する関数は 1 回も呼ばれていない

#### Scenario: gh の失敗はそのまま返る

- **WHEN** 終了コード 1 と stderr `HTTP 403` を返すスタブに差し替えた `Client` で `EditPRLabels(ctx, "org/app", 131, []string{"docs"}, nil)` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `pr edit`、`exit 1`、`HTTP 403` を含む

## MODIFIED Requirements

### Requirement: Client は fixture 用に読み取りコマンドの標準出力を採取する
`Client` はメソッド `Capture(ctx context.Context, repo string, progress func(name string)) (map[string][]byte, error)` を MUST 提供する。`repo` は `owner/name` 形式 1 件。返り値は s03 `gh-fake` の fixture ファイル名をキー、そのコマンドの標準出力（デコードせず生のバイト列）を値にした map である。
採取の順序と範囲は次のとおり。
1. `SearchIssues` と同じ引数（`--repo` は `repo` 1 つ）で `gh search issues` を実行し、`search-issues.json` に入れる。出力を s03 のデコード関数で読んで issue 番号の一覧を得る
2. 各 issue 番号 `n` について順に、`ViewIssue` / `CrossReferencedPRs` / `LabelTimeline` と同じ引数で実行し、`issue-<n>.json` / `issue-<n>-cross-refs.json` / `issue-<n>-timeline.json` に入れる
3. `SearchPRs` と同じ引数で `gh search prs` を実行し、`search-prs.json` に入れる。PR 番号の一覧を得る
4. 各 PR 番号 `n` について順に、`gh pr view <n> -R <repo> --json number,title,body,url,labels,isDraft,comments,mergeable,mergeStateStatus,statusCheckRollup,reviewDecision`（`ViewPR` と `ViewPRMergeState` のフィールドを合わせた 11 フィールド。s03 `gh-fake` の `pr-<n>.json` の定義どおり）と、`ReviewThreads` と同じ引数で実行し、`pr-<n>.json` / `pr-<n>-review-threads.json` に入れる

5. `ListLabels` と同じ引数で `gh label list` を実行し、`labels.json` に入れる（s28 `label-picker` が `Fake` にラベルの一覧を読ませるため。リポジトリ単位の 1 ファイルなので、issue / PR の繰り返しの後に 1 回だけ実行する）

`pr view` 以外の引数は対応する読み取りメソッドと同一でなければならない（引数の組み立てを共有する）。`pr view` は上記 11 フィールドの 1 回の出力で、`ViewPRMergeState` と同じ「`mergeable` が `UNKNOWN` なら `RetryWait`（既定 2 秒）後に 1 回だけ再取得」を通った後の標準出力を入れる（再取得の処理を `ViewPRMergeState` と共有する。fixture が `UNKNOWN` だらけになると局面 C を fixture でテストできないため）。map のキーは `Fake` がファイルを探す名前と同じ関数から組み立てる。
`progress` は各ファイルの取得直前に 1 回、これから入れるファイル名を引数に呼ぶ（呼び出し側が進捗を表示するため。`pr view` の UNKNOWN 再取得で `gh` が 2 回走っても呼び出しは 1 回）。いずれかの実行が失敗したら、その時点のエラー（s03 の `*Error` またはデコードエラー）を返し、それ以降は実行しない。

#### Scenario: issue 1 件と PR 1 件のリポジトリを採取する
- **WHEN** 実行関数を差し替え、`search issues` に issue 108 だけの配列、`search prs` に PR 131 だけの配列、その他の引数には空でない JSON を返す状態で `Capture(ctx, "org/app", progress)` を呼ぶ
- **THEN** map のキーは `search-issues.json` / `issue-108.json` / `issue-108-cross-refs.json` / `issue-108-timeline.json` / `search-prs.json` / `pr-131.json` / `pr-131-review-threads.json` / `labels.json` の 8 つで、各値は実行関数が返した標準出力のバイト列と一致する。`progress` はこの 8 つの名前をこの順で受け取る

#### Scenario: 引数は読み取りメソッドと同じである
- **WHEN** 上記の状態で `Capture` を呼び、実行関数が受け取った引数列を記録する
- **THEN** 1 回目は `search issues --repo org/app --state open --limit 200 --json repository,number,title,labels,updatedAt,url,body,commentsCount`、issue 108 の 3 回は `ViewIssue(ctx, "org/app", 108)` / `CrossReferencedPRs` / `LabelTimeline` が発行する引数と同一、`search prs` は `SearchPRs` の引数と同一、PR 131 の 1 回目は `pr view 131 -R org/app --json number,title,body,url,labels,isDraft,comments,mergeable,mergeStateStatus,statusCheckRollup,reviewDecision`、2 回目は `ReviewThreads` の引数（`api graphql -f owner=org -f name=app -F number=131 -f query=<s03 のクエリ文字列>`）と同一で、最後の 1 回は `ListLabels` の引数と同一である

#### Scenario: pr view の UNKNOWN は再取得後の出力を入れる
- **WHEN** `RetryWait` を 0 にし、実行関数が `pr view 131` の 1 回目に `mergeable: "UNKNOWN"`、2 回目に `mergeable: "MERGEABLE"` を返す状態で `Capture` を呼ぶ
- **THEN** `pr view 131`（11 フィールド）は 2 回実行され、`pr-131.json` の内容は 2 回目の標準出力である。`mergeable: "MERGEABLE"` を 1 回目に返した場合は 1 回しか実行されない

#### Scenario: open issue も open PR も無い
- **WHEN** 実行関数が `search issues` と `search prs` に空配列 `[]` を返す状態で `Capture` を呼ぶ
- **THEN** map のキーは `search-issues.json` と `search-prs.json` と `labels.json` の 3 つで、実行関数は 3 回しか呼ばれない（`labels.json` はリポジトリ単位なので issue / PR が 0 件でも採る）

#### Scenario: 途中の gh 失敗で止まる
- **WHEN** 実行関数が `issue view 108` に終了コード 1 と stderr を返す状態で `Capture` を呼ぶ
- **THEN** `*Error` が返り、`issue view 108` より後の引数（cross-refs / timeline / `search prs`）で実行関数は呼ばれない
