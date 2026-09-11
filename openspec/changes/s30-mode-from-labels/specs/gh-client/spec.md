## MODIFIED Requirements

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
