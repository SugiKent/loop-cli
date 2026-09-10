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

### Requirement: PR のラベルは gh pr edit で 1 つずつ付け外しする

`internal/gh` の `GHClient` interface は `AddLabelPR(ctx context.Context, repo string, number int, label string) error` と `RemoveLabelPR(ctx context.Context, repo string, number int, label string) error` を MUST 持つ。`Client` はそれぞれ `gh pr edit <number> -R <repo> --add-label <label>` と `gh pr edit <number> -R <repo> --remove-label <label>` を MUST 実行する。1 回の実行で渡すラベルは 1 つだけとする（不変条件 1「ラベルは 1 つずつ付け外しする」）。

既存の `AddLabel` / `RemoveLabel`（`gh issue edit`）を PR 番号に対して使ってはならない。`gh issue edit` は内部で `issueOrPullRequest` を引くため PR 番号でも通るが、この挙動は `gh` のヘルプに書かれておらず、`Fake` が `gh` を起動しない以上、壊れてもテストが気づけない。`internal/gh` の他の PR 操作（`ViewPR` / `CommentPR` / `MergePR`）がすべて `pr` サブコマンドを使っていることにも揃える。

#### Scenario: PR にラベルを付ける引数

- **WHEN** 実行を記録するスタブに差し替えた `Client` で `AddLabelPR(ctx, "org/app", 131, "docs")` を呼ぶ
- **THEN** 記録された引数は `pr edit 131 -R org/app --add-label docs` である

#### Scenario: PR からラベルを外す引数

- **WHEN** 同じスタブで `RemoveLabelPR(ctx, "org/app", 131, "docs")` を呼ぶ
- **THEN** 記録された引数は `pr edit 131 -R org/app --remove-label docs` である

#### Scenario: gh の失敗はそのまま返る

- **WHEN** 終了コード 1 と stderr `HTTP 403` を返すスタブに差し替えた `Client` で `AddLabelPR(ctx, "org/app", 131, "docs")` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `pr edit`、`exit 1`、`HTTP 403` を含む
