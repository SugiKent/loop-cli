## ADDED Requirements

### Requirement: ListLabels はリポジトリで使えるラベルを名前順で返す

`internal/gh` の `GHClient` interface は `ListLabels(ctx context.Context, repo string) ([]Label, error)` を MUST 持つ（s03「`GHClient` interface が読み取りと書き込みのメソッドを定義する」が定めた「後続 change が必要とするメソッドは、その change が ADDED で Requirement を足して interface・`Client`・`Fake` を同時に拡張する」に従う）。`Label` は `Name string` と `Description string` と `Color string` を持ち、`gh` の JSON 出力をそのまま写す（s03「生の型は `gh` の JSON 出力を写す」）。

`Client` の `ListLabels` は `gh label list -R <repo> --json name,description,color --sort name --order asc --limit 100` を MUST 実行し、標準出力を `[]Label` にデコードして返す。`--sort name --order asc` を明示するのは、`gh` の既定が作成順であり、リポジトリごとに並びが変わると画面も fixture も非決定になるためである。並べ替えは `gh` に任せ、クライアント側では行わない。`--limit 100` を超えるラベルを持つリポジトリでは 100 件で切れるが、ページングは行わない。デコードに失敗したときは、コマンドと元のエラーを含むエラーを返す（s03「`gh` の失敗はコマンドと stderr を含むエラーになる」と同じ形）。

#### Scenario: 実行するコマンドと引数

- **WHEN** 実行を記録するスタブに差し替えた `Client` で `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** 記録された引数は `label list -R org/app --json name,description,color --sort name --order asc --limit 100` である

#### Scenario: 出力をデコードする

- **WHEN** 標準出力に `[{"name":"docs","description":".claude/ と docs/ だけの PR","color":"0075ca"},{"name":"wip","description":"","color":"ededed"}]` を返すスタブに差し替えた `Client` で `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** 2 件が順に返り、1 件目の `Name` は `docs`、`Description` は `.claude/ と docs/ だけの PR`、`Color` は `0075ca` であり、2 件目の `Name` は `wip` で `Description` は空文字列である

#### Scenario: デコードの失敗はコマンドを含むエラーになる

- **WHEN** 標準出力に `{` を返すスタブに差し替えた `Client` で `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `label list` と `decode` を含む

### Requirement: ラベルの一括編集は 1 回の gh 実行で付ける名前と外す名前を渡す

`internal/gh` の `GHClient` interface は次の 2 つを MUST 持つ。

- `EditIssueLabels(ctx context.Context, repo string, number int, add []string, remove []string) error`
- `EditPRLabels(ctx context.Context, repo string, number int, add []string, remove []string) error`

`Client` はそれぞれ `gh issue edit <number> -R <repo>` と `gh pr edit <number> -R <repo>` を**ちょうど 1 回**実行し、`add` の各要素を `--add-label <名前>`、`remove` の各要素を `--remove-label <名前>` として、`add` → `remove` の順に受け取った並びのまま引数へ並べる。フラグは 1 名前につき 1 回ずつ繰り返し、コンマで連結しない（`gh` はコンマで分割するため、名前にコンマが含まれると壊れる）。`add` と `remove` がどちらも空のときは `gh` を実行せず nil を返す。

このメソッドは**ラベル集合の置換を行わない**。`--add-label` / `--remove-label` だけを使い、渡されなかったラベルには触れないので、他の書き手が付けたラベルは残る（不変条件 1 が守りたかったこと）。1 回の実行で複数のラベルが変わるため、GitHub は増えたラベルの数だけ `labeled` イベントを出す。

PR に対して `EditIssueLabels`（`gh issue edit`）を使ってはならない。`gh issue edit` は内部で `issueOrPullRequest` を引くため PR 番号でも通るが、この挙動は `gh` のヘルプに書かれておらず、`Fake` が `gh` を起動しない以上、将来変わってもテストが気づけない。`internal/gh` の他の PR 操作（`ViewPR` / `CommentPR` / `MergePR`）がすべて `pr` サブコマンドを使っていることにも揃える。

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
