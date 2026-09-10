## ADDED Requirements

### Requirement: ListLabels はリポジトリで使えるラベルを返す

`internal/gh` の `GHClient` interface は `ListLabels(ctx context.Context, repo string) ([]Label, error)` を MUST 持つ（s03「`GHClient` interface が読み取りと書き込みのメソッドを定義する」が定めた「後続 change が必要とするメソッドは、その change が ADDED で Requirement を足して interface・`Client`・`Fake` を同時に拡張する」に従う）。`Label` は `Name string` と `Description string` と `Color string` を持ち、`gh` の JSON 出力をそのまま写す（s03「生の型は `gh` の JSON 出力を写す」）。

`Client` の `ListLabels` は `gh label list --repo <repo> --json name,description,color --limit 100` を MUST 実行し、標準出力を `[]Label` にデコードして返す。並びは `gh` が返した順のままにする（並べ替えない）。`--limit 100` を超えるラベルを持つリポジトリでは 100 件で切れるが、ページングは行わない。デコードに失敗したときは、コマンドと元のエラーを含むエラーを返す（s03「`gh` の失敗はコマンドと stderr を含むエラーになる」と同じ形）。

#### Scenario: 実行するコマンドと引数

- **WHEN** 実行を記録するスタブに差し替えた `Client` で `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** 記録された引数は `label list --repo org/app --json name,description,color --limit 100` である

#### Scenario: 出力をデコードする

- **WHEN** 標準出力に `[{"name":"docs","description":".claude/ と docs/ だけの PR","color":"0075ca"},{"name":"wip","description":"","color":"ededed"}]` を返すスタブに差し替えた `Client` で `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** 2 件が順に返り、1 件目の `Name` は `docs`、`Description` は `.claude/ と docs/ だけの PR`、`Color` は `0075ca` であり、2 件目の `Name` は `wip` で `Description` は空文字列である

#### Scenario: デコードの失敗はコマンドを含むエラーになる

- **WHEN** 標準出力に `{` を返すスタブに差し替えた `Client` で `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `label list` と `decode` を含む
