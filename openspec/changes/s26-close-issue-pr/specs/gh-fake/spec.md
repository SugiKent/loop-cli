## MODIFIED Requirements

### Requirement: Fake は書き込み呼び出しと ViewIssue を記録する
`Fake` の書き込みメソッド（`CommentIssue` / `CommentPR` / `AddLabel` / `RemoveLabel` / `MergePR` / `CreateIssue` / `ReplyReviewThread` / `Browse` / `OpenURL` / `CloseIssue` / `ClosePR`）は `gh` を呼ばず、呼び出しをフィールド `Calls []Call` に呼び出し順で MUST 追記して nil を返す。`OpenURL` は `Method` が `OpenURL`、`URL` に受け取った URL を記録し、ブラウザ起動コマンドを実行しない。`CloseIssue` と `ClosePR` は `Method` にそれぞれ `CloseIssue` / `ClosePR` と、`Repo` と `Number` を記録する（s26 `close-issue-pr` のテストが「issue と PR で書き先を混同していないか」を `Calls` で検証するため）。読み取りのうち `ViewIssue` だけは `Calls` に `Method` が `ViewIssue`、`Repo`、`Number` を MUST 記録する（不変条件 3「`stage:todo` を外し、`gh issue view --json labels` で読み直してから `stage:propose` を付ける」の順序を s15 のテストで検証するため）。他の読み取りは記録しない。`Call` は `Method string`（メソッド名）/ `Repo string` / `Number int` / `Body string` / `Label string` / `MergeMethod string` / `Title string` / `CommentID int64` / `URL string` を持ち、使わないフィールドはゼロ値のままにする。`CreateIssue` は `https://github.com/<repo>/issues/0` を返す。テストは `Calls` を見て「どの引数で何回呼ばれたか」を検証する（不変条件 1「1 操作 1 ラベル」と不変条件 3 の検証手段）。

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

#### Scenario: CloseIssue と ClosePR の呼び出しが記録される
- **WHEN** `CloseIssue(ctx, "org/app", 108)` の後に `ClosePR(ctx, "org/app", 131)` を呼ぶ
- **THEN** `Calls` は 2 件で、`Method` は順に `CloseIssue` / `ClosePR`、`Repo` はどちらも `org/app`、`Number` は順に 108 と 131 であり、`Body` と `Label` はゼロ値である
