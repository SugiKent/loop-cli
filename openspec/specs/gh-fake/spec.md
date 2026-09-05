# gh-fake Specification

## Purpose
TBD - created by archiving change s03-gh-client. Update Purpose after archive.
## Requirements
### Requirement: fixture はリポジトリ別名ごとのディレクトリに決まった名前で置く
fixture は `internal/gh/testdata/fixtures/<repo-alias>/` 配下に MUST 置く。`<repo-alias>` は個人・組織情報を伏せた別名（例 `example`）で、ディレクトリ 1 つが稼働リポジトリ 1 件に対応する。ファイル名は次の規則に MUST 従い、内容は `Client` が発行するコマンドの標準出力をそのまま保存したものである（採取は s04 が担当する。`pr-<n>.json` だけは `ViewPR` と `ViewPRMergeState` の合成フィールド列で採る）。
- `search-issues.json`: `gh search issues …` の出力（配列）
- `search-prs.json`: `gh search prs …` の出力（配列）
- `issue-<n>.json`: `gh issue view <n> …` の出力
- `pr-<n>.json`: `gh pr view <n> …` の出力（`ViewPR` と `ViewPRMergeState` の `--json` フィールドを合わせた 1 回の出力。`Fake` は両メソッドでこの 1 ファイルを読む）
- `pr-<n>-review-threads.json`: `ReviewThreads` の GraphQL 応答（`data` から始まる生の応答）
- `issue-<n>-cross-refs.json`: `CrossReferencedPRs` の GraphQL 応答（同上）
- `issue-<n>-timeline.json`: `LabelTimeline` の `--jq` 出力（JSON オブジェクトの連続）

`<n>` は issue / PR 番号をそのまま 10 進で書く（ゼロ埋めしない）。

#### Scenario: 番号付きファイル名が決まる
- **WHEN** 別名 `example` の PR 131 の `gh pr view` 出力を保存する
- **THEN** パスは `internal/gh/testdata/fixtures/example/pr-131.json` である

### Requirement: Fake は fixture を読んで GHClient と同じ型を返す
`internal/gh` は `GHClient` を満たす型 `Fake` と、fixture ディレクトリのパスを受け取るコンストラクタ `NewFake(dir string) *Fake` を MUST 提供する。読み取りメソッドは上記の命名規則でファイルを探し、`Client` と同じデコード関数で型に写して返す。`repo` 引数はファイルの探索に使わない（ディレクトリ 1 つが 1 リポジトリに対応するため）。対応するファイルが無ければ、そのパスを含むエラーを返す。空の結果を返して黙って通さない。`Fake` の `ViewPRMergeState` は再取得も待ちも行わず、`pr-<n>.json` の merge 関連フィールドをそのまま返す。

#### Scenario: search-issues.json を読む
- **WHEN** `internal/gh/testdata/fixtures/example/search-issues.json` に issue 2 件の配列がある状態で `NewFake("testdata/fixtures/example").SearchIssues(ctx, []string{"org/app"})` を呼ぶ
- **THEN** `SearchIssue` 2 件が返り、`Number` / `Title` / `Labels` / `Repository.NameWithOwner` がファイルの内容と一致する

#### Scenario: pr-<n>.json を読む
- **WHEN** `pr-131.json` がある状態で `ViewPR(ctx, "org/app", 131)` を呼ぶ
- **THEN** `PRDetail` が返り、`IsDraft` / `Comments` がファイルの内容と一致する

#### Scenario: pr-<n>.json から merge ガード用の状態を読む
- **WHEN** `pr-131.json` が `mergeable: "UNKNOWN"` の状態で `ViewPRMergeState(ctx, "org/app", 131)` を呼ぶ
- **THEN** `PRMergeState` が返り、`Mergeable` は `UNKNOWN` のまま、`StatusCheckRollup` がファイルの内容と一致する。再取得も待ちも行わない

#### Scenario: fixture が無い
- **WHEN** `issue-999.json` が無い状態で `ViewIssue(ctx, "org/app", 999)` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `issue-999.json` を含む

#### Scenario: Client と同じデコードを通る
- **WHEN** `pr-131-review-threads.json` に GraphQL の生の応答がある状態で `ReviewThreads(ctx, "org/app", 131)` を呼ぶ
- **THEN** 実行関数を差し替えて同じバイト列を返す `Client` の `ReviewThreads` の結果と `reflect.DeepEqual` で一致する

### Requirement: Fake は書き込み呼び出しと ViewIssue を記録する
`Fake` の書き込みメソッド（`CommentIssue` / `CommentPR` / `AddLabel` / `RemoveLabel` / `MergePR` / `CreateIssue` / `ReplyReviewThread` / `Browse` / `OpenURL`）は `gh` を呼ばず、呼び出しをフィールド `Calls []Call` に呼び出し順で MUST 追記して nil を返す。`OpenURL` は `Method` が `OpenURL`、`URL` に受け取った URL を記録し、ブラウザ起動コマンドを実行しない。読み取りのうち `ViewIssue` だけは `Calls` に `Method` が `ViewIssue`、`Repo`、`Number` を MUST 記録する（不変条件 3「`stage:todo` を外し、`gh issue view --json labels` で読み直してから `stage:propose` を付ける」の順序を s15 のテストで検証するため）。他の読み取りは記録しない。`Call` は `Method string`（メソッド名）/ `Repo string` / `Number int` / `Body string` / `Label string` / `MergeMethod string` / `Title string` / `CommentID int64` / `URL string` を持ち、使わないフィールドはゼロ値のままにする。`CreateIssue` は `https://github.com/<repo>/issues/0` を返す。テストは `Calls` を見て「どの引数で何回呼ばれたか」を検証する（不変条件 1「1 操作 1 ラベル」と不変条件 3 の検証手段）。

#### Scenario: AddLabel の呼び出しが記録される
- **WHEN** `AddLabel(ctx, "org/app", 108, "stage:todo")` を呼ぶ
- **THEN** `Calls` は 1 件で、`Method` が `AddLabel`、`Repo` が `org/app`、`Number` が 108、`Label` が `stage:todo` である

#### Scenario: 複数の書き込みが順に記録される
- **WHEN** `RemoveLabel(ctx, "org/app", 108, "stage:todo")` の後に `AddLabel(ctx, "org/app", 108, "stage:propose")` を呼ぶ
- **THEN** `Calls` は 2 件で、1 件目の `Method` が `RemoveLabel`、2 件目の `Method` が `AddLabel` であり、`Label` はそれぞれ `stage:todo` と `stage:propose` である

#### Scenario: CommentPR の本文が記録される
- **WHEN** `CommentPR(ctx, "org/app", 131, "Q1: A")` を呼ぶ
- **THEN** `Calls` は 1 件で、`Method` が `CommentPR`、`Number` が 131、`Body` が `Q1: A` である

#### Scenario: OpenURL の URL が記録される
- **WHEN** `OpenURL(ctx, "https://example.com/design")` を呼ぶ
- **THEN** `Calls` は 1 件で、`Method` が `OpenURL`、`URL` が `https://example.com/design` であり、`Repo` と `Number` はゼロ値である

#### Scenario: RemoveLabel → ViewIssue → AddLabel の順序が記録される
- **WHEN** `RemoveLabel(ctx, "org/app", 108, "stage:todo")`、`ViewIssue(ctx, "org/app", 108)`、`AddLabel(ctx, "org/app", 108, "stage:propose")` をこの順で呼ぶ
- **THEN** `Calls` は 3 件で、`Method` は順に `RemoveLabel` / `ViewIssue` / `AddLabel`、2 件目の `Repo` は `org/app`、`Number` は 108 である

#### Scenario: ViewIssue 以外の読み取りは記録しない
- **WHEN** `SearchIssues` / `ViewPR` / `ViewPRMergeState` を呼んだ後に `Calls` を見る
- **THEN** `Calls` は空である

### Requirement: Fake は並行呼び出しでも Calls の内容を壊さない
`Fake` は複数のゴルーチンから同時にメソッドを呼ばれても、`Calls` への追記を排他して MUST 行う（`internal/gh/fake.go` の `Calls` への `append` を `sync.Mutex` で守る）。並行して呼ばれた各呼び出しはちょうど 1 件ずつ `Calls` に残り、`Method` / `Repo` / `Number` 等の内容は失われない。直列に呼んだときの記録内容と順序の規則（Requirement「Fake は書き込み呼び出しと ViewIssue を記録する」）は変わらない。並行して呼ばれた呼び出しどうしの `Calls` 内の順序は定めない。s07 の `Fetch` が `ViewIssue` を並行して呼ぶ最初の利用者である。

#### Scenario: 並行した ViewIssue が全件記録され、race が無い
- **WHEN** `NewFake("testdata/fixtures/example")` に対して 8 本のゴルーチンから `ViewIssue(ctx, "org/app", 108)` を同時に呼び、全部の完了を待ってから `Calls` を見る（`go test -race ./internal/gh/` で実行する）
- **THEN** race detector の報告が無く、`Calls` は 8 件で、全件の `Method` が `ViewIssue`、`Repo` が `org/app`、`Number` が 108 である

