## MODIFIED Requirements

### Requirement: 不正な設定はパスと原因を含むエラーになる
`Load` は次の場合にエラーを MUST 返す。エラー文字列にはファイルパスと、どの項目がなぜ不正かを含める。エラー時に `Config` は返さない。
- ファイルが存在しない
- `repos` が無い、または空（ファイルが空の場合を含む）
- `repos` の要素が「文字列」でも「`name` を持つマッピング」でもない
- `repos` の要素が `owner/name` 形式でない。判定は「`/` がちょうど 1 つ、その両側が空でない、空白を含まない」に限る
- `merge_method`（グローバル・リポジトリ別とも）が `squash` / `merge` / `rebase` 以外（リポジトリ別の空文字列は未指定扱いで対象外）
- `refresh_interval_sec` が 1 未満
- 未知のキーがある（トップレベル、および `repos` 要素のマッピング内のどちらも）

`repos` の要素の `mode` キーは s30 で廃止した。`mode` を持つ設定ファイルに対しては、`repos の mode は廃止しました。運用方式はリポジトリのラベル一覧から判定するので、この行を削除してください` を含むエラーを MUST 返す。汎用の未知キーの文言は使わない。この 1 件だけ専用の文言を持つのは、s26 から s30 の間に `mode: label` を書いた設定ファイルが、この change の後は起動しなくなるためである。汎用の文言だけでは、値が正しいのになぜ止まるのかが読み取れない。

#### Scenario: ファイルが無い
- **WHEN** 存在しないパスを `Load` に渡す
- **THEN** エラーが返り、エラー文字列にそのパスが含まれる

#### Scenario: repos が空
- **WHEN** `repos: []` と書いたファイル、`repos` キーの無いファイル、空のファイルのそれぞれを `Load` に渡す
- **THEN** いずれもエラーが返り、エラー文字列に `repos` が含まれる

#### Scenario: repos の形式が不正
- **WHEN** `repos` に `app`（`/` 無し）、`org/app/extra`（`/` が 2 つ）、`/app`（owner 空）、`org/ app`（空白）のいずれかを含むファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列に不正だった要素の文字列が含まれる

#### Scenario: merge_method が不正
- **WHEN** `merge_method: fast-forward` と書いたファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列に `merge_method` と `fast-forward` が含まれる

#### Scenario: mode が不正
- **WHEN** `repos` の要素に `{name: org/board, mode: label}` と書いたファイル、`{name: org/board, mode: kanban}` と書いたファイルのそれぞれを `Load` に渡す
- **THEN** いずれもエラーが返り、エラー文字列に `mode は廃止しました` と `削除してください` が含まれる（値が正しくても廃止したキーとして止める）

#### Scenario: refresh_interval_sec が 0 以下
- **WHEN** `refresh_interval_sec: 0` と書いたファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列に `refresh_interval_sec` が含まれる

#### Scenario: 未知のトップレベルキーがある
- **WHEN** `refresh_interval: 60`（正しくは `refresh_interval_sec`）と書いたファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列に `refresh_interval` が含まれる

#### Scenario: repos 要素のマッピングに未知のキーがある
- **WHEN** `repos` の要素に `{name: org/web, merge_methd: rebase}`（正しくは `merge_method`）と書いたファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列に `merge_methd` が含まれる。グローバル値へ黙って落とさない

#### Scenario: repos 要素が文字列でも name 付きマッピングでもない
- **WHEN** `repos` の要素に `{merge_method: rebase}`（`name` 無し）と書いたファイル、`[org/app]`（シーケンス）と書いたファイルのそれぞれを `Load` に渡す
- **THEN** いずれもエラーが返り、エラー文字列に `repos` が含まれる

## REMOVED Requirements

### Requirement: リポジトリごとに issue の運用方式を mode で指定できる

**Reason**: 運用方式の正本がリポジトリのラベル一覧に移り、`Load` は方式を解決しなくなる。`config.Repo` から `Mode` の欄が消え、`repos[].mode` は書けなくなる。設定に方式を書かせる限り書き忘れという状態が残り、書き忘れたリポジトリに間違った承認ラベルを書く経路が閉じない。

**Migration**: 利用者は `config.yml` の `repos` の要素から `mode:` の行を消す。消さないと `Load` が Requirement「不正な設定はパスと原因を含むエラーになる」の専用の文言で止める。方式の判定は `card-fetch`「Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する」が引き継ぎ、`stage:todo` があるリポジトリは `sdd`、`To Do` があるリポジトリは `label` になる。`mode: sdd` を書いていたリポジトリは `routines-setup` が作った `stage:todo` を持つので判定が一致し、`mode: label` を書いていたリポジトリは `To Do` を持つので判定が一致する。方式ごとのラベル語彙そのもの（`To Do` / `In Progress` / `Done`）は `card-model` が持ち続ける。
