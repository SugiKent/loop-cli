# gh-client Specification

## Purpose
TBD - created by archiving change s03-gh-client. Update Purpose after archive.

## Requirements

### Requirement: GHClient interface が読み取りと書き込みのメソッドを定義する
`internal/gh` は interface `GHClient` を MUST 公開する。メソッドはすべて第 1 引数に `context.Context` を取り、リポジトリは `owner/name` 形式の文字列で受ける。この change で定義するメソッドは次の 16 個である。docs に無いメソッド（レートリミット取得等）はこの change では定義しない。後続 change が必要とするメソッドは、その change が ADDED で Requirement を足して interface・`Client`・`Fake` を同時に拡張する。

読み取り（D-001 の呼び出しに対応）:
- `SearchIssues(ctx, repos []string) ([]SearchIssue, error)`: 設定した全リポジトリの open issue
- `SearchPRs(ctx, repos []string) ([]SearchPR, error)`: 設定した全リポジトリの open PR
- `ViewIssue(ctx, repo string, number int) (*IssueDetail, error)`: issue のコメントとラベル
- `ViewPR(ctx, repo string, number int) (*PRDetail, error)`: PR のコメント・ラベル（`question` PR の判定）。再取得しない
- `ViewPRMergeState(ctx, repo string, number int) (*PRMergeState, error)`: merge ガード用の状態（merge 候補 PR）。`mergeable: UNKNOWN` の再取得はこのメソッドだけが行う
- `ReviewThreads(ctx, repo string, number int) ([]ReviewThread, error)`: PR の review thread（局面 D）
- `CrossReferencedPRs(ctx, repo string, number int) ([]CrossReferencedPR, error)`: issue を参照した PR（紐づけ補完）
- `LabelTimeline(ctx, repo string, number int) ([]LabelEvent, error)`: issue のラベル付け外しイベント（ラベル変遷）

書き込みメソッドは human-turn-signals.md「TUI のアクション」列と mvp.md キーバインド表「内部処理」列に対応する:
- `CommentIssue(ctx, repo string, number int, body string) error`: `a`（局面 B）
- `CommentPR(ctx, repo string, number int, body string) error`: `a`（局面 A）
- `AddLabel(ctx, repo string, number int, label string) error`: `t` / `s`
- `RemoveLabel(ctx, repo string, number int, label string) error`: `t` / `s`
- `MergePR(ctx, repo string, number int, method string) error`: `m`。`method` は `squash` / `merge` / `rebase`
- `CreateIssue(ctx, repo string, title string, body string) (string, error)`: `n`。作成した issue の URL を返す
- `ReplyReviewThread(ctx, repo string, number int, commentID int64, body string) error`: `A`。`commentID` は thread 内コメントの REST `id`（GraphQL の `databaseId`）
- `Browse(ctx, repo string, number int) error`: `o`

#### Scenario: Client と Fake が GHClient を満たす
- **WHEN** `var _ GHClient = (*Client)(nil)` と `var _ GHClient = (*Fake)(nil)` をコンパイルする
- **THEN** コンパイルが通る

### Requirement: 生の型は gh の JSON 出力を写す
`internal/gh` の型は `gh` の `--json` 出力と `gh api` の応答をそのまま写した生の型で MUST あり、分類結果や Issue と PR の紐づけを持たない（それらは s05 の `internal/model` と s07 が担当する）。型とフィールドは次のとおり。JSON のキー名は `gh` の出力どおりにタグで指定し、日時は `time.Time` で受ける。
- `Label { Name string }`（`gh` が返す `id` / `color` / `description` は読み捨てる）
- `Repository { Name string; NameWithOwner string }`
- `SearchIssue { Repository Repository; Number int; Title string; Labels []Label; UpdatedAt time.Time; URL string; Body string; CommentsCount int }`
- `SearchPR { Repository Repository; Number int; Title string; Labels []Label; UpdatedAt time.Time; URL string; Body string; IsDraft bool }`
- `Comment { ID string; Author Author; Body string; CreatedAt time.Time; URL string }`、`Author { Login string }`
- `IssueDetail { Number int; Title string; Body string; URL string; Labels []Label; Comments []Comment }`
- `PRDetail { Number int; Title string; Body string; URL string; Labels []Label; IsDraft bool; Comments []Comment }`
- `PRMergeState { Mergeable string; MergeStateStatus string; ReviewDecision string; StatusCheckRollup []StatusCheck }`
- `StatusCheck { Typename string (JSON キー __typename); Name string; Status string; Conclusion string; Context string; State string; WorkflowName string; DetailsURL string }`（`CheckRun` は `name` / `status` / `conclusion`、`StatusContext` は `context` / `state` を使う。どちらも 1 つの型で受ける）
- `ReviewThread { ID string; IsResolved bool; Comments []ReviewComment }`、`ReviewComment { DatabaseID int64; Author Author; Body string; CreatedAt time.Time }`
- `CrossReferencedPR { Number int; Title string; Body string; State string; Labels []string }`（GraphQL の `labels { nodes { name } }` をラベル名の列に平坦化する）
- `LabelEvent { CreatedAt time.Time; Event string; Label string }`（`Event` は `labeled` または `unlabeled`）

#### Scenario: search issues の JSON が SearchIssue に写る
- **WHEN** `gh search issues --json repository,number,title,labels,updatedAt,url,body,commentsCount` の出力（配列）をデコードする
- **THEN** 各要素の `repository.nameWithOwner` / `number` / `title` / `labels[].name` / `updatedAt` / `url` / `body` / `commentsCount` が対応するフィールドに入る

#### Scenario: pr view の statusCheckRollup が CheckRun と StatusContext の両方を受ける
- **WHEN** `statusCheckRollup` に `__typename: CheckRun`（`name` / `status` / `conclusion`）の要素と `__typename: StatusContext`（`context` / `state`）の要素が混在する JSON をデコードする
- **THEN** どちらの要素も `StatusCheck` に入り、`Typename` でどちらか判別できる

#### Scenario: GraphQL の reviewThreads が ReviewThread に平坦化される
- **WHEN** `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[{"id":"PRRT_x","isResolved":false,"comments":{"nodes":[{"databaseId":1,"author":{"login":"a"},"body":"b","createdAt":"2026-09-05T00:00:00Z"}]}}]}}}}}` をデコードする
- **THEN** `ReviewThread` 1 件（`ID` が `PRRT_x`、`IsResolved` が false、`Comments` 1 件で `DatabaseID` が 1）が返る

#### Scenario: GraphQL の cross-reference が CrossReferencedPR に平坦化される
- **WHEN** `timelineItems.nodes` に `source` が PR のノード（`number` / `title` / `body` / `state` / `labels.nodes[].name`）と、`source` が空オブジェクト `{}` のノード（PR 以外からの参照）が混在する応答をデコードする
- **THEN** PR のノードだけが `CrossReferencedPR` に入り、`Labels` はラベル名の列になる。`source` が空のノードは無視される

### Requirement: Client は gh サブプロセスを正確な引数で実行する
`Client` は `GHClient` の各メソッドを `gh` サブプロセスで MUST 実装する。発行するコマンドは次のとおりで、リポジトリは常に `-R owner/name` で明示し、カレントディレクトリの git リポジトリに依存しない。本文（コメント・issue 本文・返信）は標準入力から渡し、一時ファイルを作らない。
- `SearchIssues`: `gh search issues --repo <r1> --repo <r2> … --state open --limit 200 --json repository,number,title,labels,updatedAt,url,body,commentsCount`（`--repo` はリポジトリごとに 1 つ）
- `SearchPRs`: `gh search prs --repo <r1> … --state open --limit 200 --json repository,number,title,labels,updatedAt,url,body,isDraft`
- `ViewIssue`: `gh issue view <n> -R <repo> --json number,title,body,url,labels,comments`
- `ViewPR`: `gh pr view <n> -R <repo> --json number,title,body,url,labels,isDraft,comments`
- `ViewPRMergeState`: `gh pr view <n> -R <repo> --json mergeable,mergeStateStatus,statusCheckRollup,reviewDecision`
- `ReviewThreads`: `gh api graphql -f owner=<owner> -f name=<name> -F number=<n> -f query=<Q>`（`-F` は値を自動で型付けするので、文字列の `owner` / `name` は `-f`、整数の `number` は `-F`）。`Q` は次の完全なクエリ文字列（操作宣言付き）:
  `query($owner:String!,$name:String!,$number:Int!){ repository(owner:$owner,name:$name){ pullRequest(number:$number){ reviewThreads(first:50){ nodes{ id isResolved comments(first:100){ nodes{ databaseId author{login} body createdAt } } } } } } }`
- `CrossReferencedPRs`: `gh api graphql -f owner=<owner> -f name=<name> -F number=<n> -f query=<Q>`。`Q` は次の完全なクエリ文字列:
  `query($owner:String!,$name:String!,$number:Int!){ repository(owner:$owner,name:$name){ issue(number:$number){ timelineItems(first:50, itemTypes:[CROSS_REFERENCED_EVENT]){ nodes{ ... on CrossReferencedEvent { source { ... on PullRequest { number title body state labels(first:20){nodes{name}} } } } } } } } }`
- `LabelTimeline`: `gh api repos/<owner>/<name>/issues/<n>/timeline --paginate --jq '.[] | select(.event=="labeled" or .event=="unlabeled") | {created_at, event, label: .label.name}'`（D-001 の記述どおり。出力は JSON オブジェクトの連続）
- `CommentIssue`: `gh issue comment <n> -R <repo> --body-file -`（本文は標準入力）
- `CommentPR`: `gh pr comment <n> -R <repo> --body-file -`（本文は標準入力）
- `AddLabel`: `gh issue edit <n> -R <repo> --add-label <label>`（ラベル 1 つだけ。カンマ区切りで複数を渡さない。不変条件 1）
- `RemoveLabel`: `gh issue edit <n> -R <repo> --remove-label <label>`（同上）
- `MergePR`: `gh pr merge <n> -R <repo> --<method>`（`--squash` / `--merge` / `--rebase`。`--delete-branch` / `--auto` / `--admin` は付けない）
- `CreateIssue`: `gh issue create -R <repo> --title <title> --body-file -`（本文は標準入力。`--label` は付けない。不変条件 6）。標準出力の末尾行（作成した issue の URL）を返す。終了コード 0 でも末尾行が `https://` で始まらなければデコードエラー（`*Error` ではないエラー）として返す
- `ReplyReviewThread`: `gh api -X POST repos/<owner>/<name>/pulls/<n>/comments/<commentID>/replies --input -`（標準入力に `{"body":"<body>"}` を JSON として渡す）
- `Browse`: `gh browse <n> -R <repo>`

`Client` はサブプロセスの標準入力を本文以外では閉じ、環境変数 `GH_PROMPT_DISABLED=1` と `GH_NO_UPDATE_NOTIFIER=1` を付けて対話プロンプトと更新通知を抑止する。
テストで `gh` を起動せずに引数を検証できるよう、`Client` はサブプロセス実行関数を差し替え可能に持つ（design.md）。

#### Scenario: SearchIssues が --repo をリポジトリごとに並べる
- **WHEN** 実行関数を差し替えた `Client` で `SearchIssues(ctx, []string{"org/app", "org/web"})` を呼ぶ
- **THEN** 実行関数は引数 `search issues --repo org/app --repo org/web --state open --limit 200 --json repository,number,title,labels,updatedAt,url,body,commentsCount` を 1 回受け取る

#### Scenario: AddLabel はラベルを 1 つだけ渡す
- **WHEN** `AddLabel(ctx, "org/app", 108, "stage:todo")` を呼ぶ
- **THEN** 実行関数は引数 `issue edit 108 -R org/app --add-label stage:todo` を 1 回受け取る

#### Scenario: CommentPR は本文を標準入力で渡す
- **WHEN** `CommentPR(ctx, "org/app", 131, "Q1: A\nQ2: B")` を呼ぶ
- **THEN** 実行関数は引数 `pr comment 131 -R org/app --body-file -` と、標準入力の内容 `Q1: A\nQ2: B` を受け取る

#### Scenario: MergePR は設定の merge 方式をフラグにする
- **WHEN** `MergePR(ctx, "org/app", 151, "squash")` を呼ぶ
- **THEN** 実行関数は引数 `pr merge 151 -R org/app --squash` を受け取る

#### Scenario: CreateIssue が URL を返す
- **WHEN** 実行関数が標準出力に `https://github.com/org/app/issues/200\n` を返す状態で `CreateIssue(ctx, "org/app", "タイトル", "本文")` を呼ぶ
- **THEN** 実行関数は引数 `issue create -R org/app --title タイトル --body-file -` と標準入力 `本文` を受け取り、`CreateIssue` は `https://github.com/org/app/issues/200` を返す

#### Scenario: ReplyReviewThread は REST replies に JSON を渡す
- **WHEN** `ReplyReviewThread(ctx, "org/app", 88, 3935153121, "修正しました")` を呼ぶ
- **THEN** 実行関数は引数 `api -X POST repos/org/app/pulls/88/comments/3935153121/replies --input -` と、標準入力に `body` キーが `修正しました` の JSON を受け取る

#### Scenario: ReviewThreads の GraphQL 引数とクエリ文字列
- **WHEN** `ReviewThreads(ctx, "org/app", 131)` を呼ぶ
- **THEN** 実行関数は引数列 `api graphql -f owner=org -f name=app -F number=131 -f query=query($owner:String!,$name:String!,$number:Int!){ repository(owner:$owner,name:$name){ pullRequest(number:$number){ reviewThreads(first:50){ nodes{ id isResolved comments(first:100){ nodes{ databaseId author{login} body createdAt } } } } } } }` をこの順で 1 回受け取る（`query=` の値はクエリ文字列全体と一致する）

#### Scenario: CrossReferencedPRs の GraphQL 引数とクエリ文字列
- **WHEN** `CrossReferencedPRs(ctx, "org/app", 108)` を呼ぶ
- **THEN** 実行関数は引数列 `api graphql -f owner=org -f name=app -F number=108 -f query=query($owner:String!,$name:String!,$number:Int!){ repository(owner:$owner,name:$name){ issue(number:$number){ timelineItems(first:50, itemTypes:[CROSS_REFERENCED_EVENT]){ nodes{ ... on CrossReferencedEvent { source { ... on PullRequest { number title body state labels(first:20){nodes{name}} } } } } } } } }` をこの順で 1 回受け取る

#### Scenario: CreateIssue の末尾行が URL でない
- **WHEN** 実行関数が終了コード 0 と標準出力 `Creating issue in org/app\n` を返す状態で `CreateIssue(ctx, "org/app", "タイトル", "本文")` を呼ぶ
- **THEN** エラーが返り、`errors.As` で `*Error` に変換できない

#### Scenario: LabelTimeline がオブジェクトの連続をデコードする
- **WHEN** 実行関数が `{"created_at":"2026-09-01T00:00:00Z","event":"labeled","label":"stage:todo"}{"created_at":"2026-09-02T00:00:00Z","event":"unlabeled","label":"stage:todo"}` を返す状態で `LabelTimeline(ctx, "org/app", 108)` を呼ぶ
- **THEN** `LabelEvent` 2 件がこの順で返り、1 件目の `Event` は `labeled`、`Label` は `stage:todo` である

### Requirement: ViewPRMergeState は mergeable が UNKNOWN なら 2 秒後に 1 回だけ再取得する
`ViewPRMergeState` は 1 回目の結果の `Mergeable` が `UNKNOWN` のとき、2 秒待ってから同じコマンドをもう 1 回だけ MUST 実行し、2 回目の結果を返す（D-001 の merge ガード取得）。2 回目も `UNKNOWN` ならそのまま返し、3 回目は実行しない。待ち時間はテストで短縮できるよう `Client` のフィールドで持つ（既定 2 秒）。この再取得は `internal/gh` が持ち、取得層（s07）は再取得を意識しない。`ViewPR` は merge 関連フィールドを取得せず再取得も待ちも行わない（`question` PR のコメント取得が 2 秒遅れないため）。

#### Scenario: UNKNOWN で 1 回だけ再取得する
- **WHEN** 実行関数が 1 回目に `mergeable: "UNKNOWN"`、2 回目に `mergeable: "MERGEABLE"` の JSON を返す状態で `ViewPRMergeState` を呼ぶ
- **THEN** 実行関数は 2 回呼ばれ、返る `PRMergeState.Mergeable` は `MERGEABLE` である

#### Scenario: 2 回目も UNKNOWN ならそのまま返す
- **WHEN** 実行関数が 2 回とも `mergeable: "UNKNOWN"` を返す状態で `ViewPRMergeState` を呼ぶ
- **THEN** 実行関数は 2 回だけ呼ばれ、返る `PRMergeState.Mergeable` は `UNKNOWN` である

#### Scenario: MERGEABLE なら再取得しない
- **WHEN** 実行関数が 1 回目に `mergeable: "MERGEABLE"` を返す状態で `ViewPRMergeState` を呼ぶ
- **THEN** 実行関数は 1 回だけ呼ばれる

#### Scenario: ViewPR は再取得しない
- **WHEN** 実行関数が `mergeable: "UNKNOWN"` を含む JSON を返す状態で `ViewPR(ctx, "org/app", 131)` を呼ぶ
- **THEN** 実行関数は引数 `pr view 131 -R org/app --json number,title,body,url,labels,isDraft,comments` を 1 回だけ受け取り、待ちなしで `PRDetail` が返る

### Requirement: gh の失敗はコマンドと stderr を含むエラーになる
`gh` が非 0 で終了した場合、`Client` は型 `*Error`（フィールド `Args []string` / `ExitCode int` / `Stderr string`）のエラーを MUST 返す。`Error()` 文字列には `gh` に渡した引数と stderr の内容を含める。JSON のデコードに失敗した場合は、引数とデコードエラーを含む別のエラーを返す。呼び出し側は `errors.As` で `*Error` かどうかを見て、`gh` の失敗（`GraphQL: Could not resolve to an issue …` 等）と、正常終了したが出力が壊れている場合を区別できる。`context` がキャンセルまたはタイムアウトした場合はその `context` のエラーを含むエラーを返す。

#### Scenario: 存在しない issue で gh が非 0 終了する
- **WHEN** 実行関数が終了コード 1 と stderr `GraphQL: Could not resolve to an issue or pull request with the number of 99999999. (repository.issue)` を返す状態で `ViewIssue(ctx, "org/app", 99999999)` を呼ぶ
- **THEN** 返るエラーは `*Error` で、`ExitCode` は 1、`Stderr` にその文言を含み、`Error()` 文字列に `issue view 99999999 -R org/app` と stderr の文言を含む

#### Scenario: JSON が壊れている
- **WHEN** 実行関数が終了コード 0 と標準出力 `not json` を返す状態で `SearchIssues` を呼ぶ
- **THEN** エラーが返り、`errors.As` で `*Error` に変換できず、エラー文字列に `search issues` を含む

#### Scenario: context のタイムアウトで失敗する
- **WHEN** 実行関数が `ctx.Done()` まで待つ状態で、タイムアウト付きの `ctx` を渡して `SearchIssues` を呼ぶ
- **THEN** タイムアウト後にエラーが返り、`errors.Is(err, context.DeadlineExceeded)` が真である

### Requirement: 起動前に gh の存在と認証を確認する
`Client` はメソッド `Check(ctx) error` を MUST 提供する。`gh` が PATH に無ければ、`gh` のインストールを促す文言を含むエラーを返す。次に `gh auth status` を実行し、非 0 で終了すれば、その stderr を含む `*Error` を返す。どちらも通れば nil を返す。`gh auth status` の実行は他のメソッドと同じ実行関数を通り、テストで差し替えられる。`Check` を呼ぶのは起動時の `cmd/loop-cli`（s08-queue-screen が配線する。s07 は配線しなかった）であり、`Client` の各メソッドは毎回 `Check` を呼ばない。

#### Scenario: gh が PATH に無い
- **WHEN** `PATH` に `gh` が存在しない状態で `Check` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `gh` を含む

#### Scenario: 認証されていない
- **WHEN** `gh` が PATH にあり、実行関数が引数 `auth status` に対して終了コード 1 と stderr `X Failed to log in to github.com using token (GH_TOKEN)` を返す状態で `Check` を呼ぶ
- **THEN** `*Error` が返り、`Stderr` にその文言を含む

#### Scenario: gh が使える
- **WHEN** `gh` が PATH にあり、実行関数が引数 `auth status` に対して終了コード 0 を返す状態で `Check` を呼ぶ
- **THEN** 実行関数は引数 `auth status` を 1 回受け取り、`Check` は nil を返す

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

### Requirement: OpenURL は任意の URL を OS のブラウザで開く

`internal/gh` の `GHClient` interface は `OpenURL(ctx context.Context, url string) error` を MUST 持つ（s03「`GHClient` interface が読み取りと書き込みのメソッドを定義する」が定めた「後続 change が必要とするメソッドは、その change が ADDED で Requirement を足して interface・`Client`・`Fake` を同時に拡張する」に従う）。`Browse` はリポジトリと番号しか受け取れず、本文に書かれた URL を開けないため、`u`（s22 `url-picker`）はこのメソッドを使う。

`Client` の `OpenURL` は `gh` を実行せず、OS のブラウザ起動コマンドを MUST 実行する。`Client` は `gh` を起動する差し替え可能な関数とは別に、ブラウザ起動コマンドを実行する差し替え可能な関数を持ち、既定の実装は `runtime.GOOS` が `darwin` なら `open <url>`、それ以外なら `xdg-open <url>` を、`ctx` 付きでシェルを経由せずに実行する。`url` は検証せず、呼び出し側が渡した文字列をそのまま 1 つの引数として渡す。標準出力は読み捨てる。

終了コードが 0 でなければ、`<コマンド名> <url>: exit <コード>: <stderr>` の形式（s03「`gh` の失敗はコマンドと stderr を含むエラーになる」と同じ並び）のエラーを MUST 返す。コマンドを起動できないとき（`open` / `xdg-open` が無い環境）も、コマンド名と URL を含むエラーを返す。

#### Scenario: Client と Fake が OpenURL を持つ GHClient を満たす

- **WHEN** `var _ GHClient = (*Client)(nil)` と `var _ GHClient = (*Fake)(nil)` をコンパイルする
- **THEN** コンパイルが通る

#### Scenario: 実行するコマンドと引数

- **WHEN** ブラウザ起動コマンドの実行を記録するスタブに差し替えた `Client` で `OpenURL(ctx, "https://example.com/a b")` を呼ぶ
- **THEN** 記録された URL はちょうど 1 件で `https://example.com/a b` であり、`gh` を実行する関数は 1 回も呼ばれていない

#### Scenario: OS ごとのコマンド名

- **WHEN** コマンド名を `runtime.GOOS` から選ぶ関数に `darwin` と `linux` を渡す
- **THEN** 返るコマンド名は順に `open` と `xdg-open` である

#### Scenario: 終了コードが 0 でなければエラーになる

- **WHEN** 終了コード 1 と stderr `no browser` を返すスタブに差し替えた `Client` で `OpenURL(ctx, "https://example.com/a")` を呼ぶ
- **THEN** 返るエラーの文字列に `https://example.com/a`、`exit 1`、`no browser` が含まれる

### Requirement: ListLabels はリポジトリで使えるラベルを名前順で返す

`internal/gh` の `GHClient` interface は `ListLabels(ctx context.Context, repo string) ([]RepoLabel, error)` を MUST 持つ（s03「`GHClient` interface が読み取りと書き込みのメソッドを定義する」が定めた「後続 change が必要とするメソッドは、その change が ADDED で Requirement を足して interface・`Client`・`Fake` を同時に拡張する」に従う）。`RepoLabel` は `Name string` と `Description string` と `Color string` を持ち、`gh label list` の JSON 出力をそのまま写す（s03「生の型は `gh` の JSON 出力を写す」）。

既存の `Label { Name string }` とは別の型にする。`Label` は issue / PR に付いているラベルを表し、`gh issue view --json labels` と GraphQL の `labels{nodes{name}}` の両方から作られるので `description` と `color` は取れない（s03 が「`id` / `color` / `description` は読み捨てる」と定めている）。`Label` に 2 つの欄を足すと、経路によって空になる欄を持つ型になり、`Fake` と `Client` で埋まり方が変わる。リポジトリが持つラベルの一覧は用途も取得元も違うので、型を分ける。

`Client` の `ListLabels` は `gh label list -R <repo> --json name,description,color --sort name --order asc --limit 1000` を MUST 実行し、標準出力を `[]RepoLabel` にデコードして返す。`--sort name --order asc` を明示するのは、`gh` の既定が作成順であり、リポジトリごとに並びが変わると画面も fixture も非決定になるためである。並べ替えは `gh` に任せ、クライアント側では行わない。`--limit 1000` にするのは、この一覧が s30 で運用方式の判定にも使われるためである。上限で切れると名前昇順の後ろにある `stage:todo` が落ち、issue-driven-sdd のリポジトリを `label` と誤判定して `To Do` を書きに行く（`card-fetch`「Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する」）。1000 件を超えるラベルを持つリポジトリでは切れるが、ページングは行わない。デコードに失敗したときは、コマンドと元のエラーを含むエラーを返す（s03「`gh` の失敗はコマンドと stderr を含むエラーになる」と同じ形）。

#### Scenario: 実行するコマンドと引数

- **WHEN** 実行を記録するスタブに差し替えた `Client` で `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** 記録された引数は `label list -R org/app --json name,description,color --sort name --order asc --limit 1000` である

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

### Requirement: CloseIssue と ClosePR は issue と PR を close する

`internal/gh` の `GHClient` interface は `CloseIssue(ctx context.Context, repo string, number int) error` と
`ClosePR(ctx context.Context, repo string, number int) error` を MUST 持つ（s03「`GHClient` interface が読み取りと
書き込みのメソッドを定義する」が定めた「後続 change が必要とするメソッドは、その change が ADDED で Requirement を足して
interface・`Client`・`Fake` を同時に拡張する」に従う）。`c`（s26 `close-issue-pr`）がこの 2 つを使う。

`Client` の実装は `gh` を次の引数で MUST 実行し、標準出力は読み捨てる（s03「`Client` は gh サブプロセスを正確な引数で実行する」の形）。

- `CloseIssue`: `issue close <number> -R <repo>`
- `ClosePR`: `pr close <number> -R <repo>`

終了コードが 0 でなければ、s03「`gh` の失敗はコマンドと stderr を含むエラーになる」と同じエラーを MUST 返す。
close の理由（`gh issue close --reason`）とコメント（`--comment`）は渡さない。ブランチの削除（`gh pr close --delete-branch`）も行わない。

#### Scenario: Client と Fake が CloseIssue と ClosePR を持つ GHClient を満たす
- **WHEN** `var _ GHClient = (*Client)(nil)` と `var _ GHClient = (*Fake)(nil)` をコンパイルする
- **THEN** コンパイルが通る

#### Scenario: CloseIssue が実行する引数
- **WHEN** `gh` の実行を記録するスタブに差し替えた `Client` で `CloseIssue(ctx, "org/app", 108)` を呼ぶ
- **THEN** 実行関数は引数 `issue close 108 -R org/app` を 1 回受け取り、標準入力には何も渡らない

#### Scenario: ClosePR が実行する引数
- **WHEN** 同じスタブで `ClosePR(ctx, "org/app", 131)` を呼ぶ
- **THEN** 実行関数は引数 `pr close 131 -R org/app` を 1 回受け取る

#### Scenario: close の失敗はコマンドと stderr を含むエラーになる
- **WHEN** 終了コード 1 と標準エラー `could not close issue` を返すスタブで `CloseIssue(ctx, "org/app", 108)` を呼ぶ
- **THEN** 返るエラーの文字列に `issue close 108 -R org/app`、`exit 1`、`could not close issue` が含まれる
