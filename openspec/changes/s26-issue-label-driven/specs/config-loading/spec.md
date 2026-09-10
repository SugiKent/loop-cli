## ADDED Requirements

### Requirement: リポジトリごとに issue の運用方式を mode で指定できる
`repos` の各要素は、そのリポジトリがどちらの運用方式かを `mode` キーで MUST 指定できる。値は 2 つある。`sdd` は issue-driven-sdd を指し、`stage:*` と `wip` と PR の段階ラベルで進む方式である。`label` は issue-label-driven を指し、`To Do` と `In Progress` と `Done` の 3 ラベルで進む方式である。利用者は `merge_method` と同じく、マッピング形式で書いた要素の中に `mode` を書く。省略した要素は `sdd` になる（既存の設定ファイルは書き換えずに従来どおり動く）。
`Load` は解決済みの方式を `Repo` に埋めて返し、消費側は指定の有無を知らずに読めばよい。方式はリポジトリ単位でだけ決まり、トップレベルの既定値は持たない（2 方式を混在させて監視するのがこの機能の目的であり、全リポジトリ共通の既定を書きたい状況が無い）。
`mode` に空文字列を明示した場合は未指定として扱い、`sdd` になる。ラベル名そのものは設定で変えられない（mvp.md「ラベル名と routine マーカーはプラグインの規約に固定し、設定で変えられるようにしない」）。

#### Scenario: mode を書かないリポジトリは sdd になる
- **WHEN** `repos` に `org/app`（文字列）と `{name: org/web, merge_method: rebase}` を書いたファイルを `Load` に渡す
- **THEN** どちらの `Repo` も方式は `sdd` である

#### Scenario: mode: label を指定したリポジトリだけ label になる
- **WHEN** `repos` に `org/app` と `{name: org/board, mode: label}` を書いたファイルを `Load` に渡す
- **THEN** `org/app` の方式は `sdd`、`org/board` の方式は `label` である

#### Scenario: mode と merge_method を同じ要素に書ける
- **WHEN** `repos` の要素に `{name: org/board, mode: label, merge_method: rebase}` と書いたファイルを `Load` に渡す
- **THEN** `org/board` の方式は `label`、`MergeMethod` は rebase である

## MODIFIED Requirements

### Requirement: 不正な設定はパスと原因を含むエラーになる
`Load` は次の場合にエラーを MUST 返す。エラー文字列にはファイルパスと、どの項目がなぜ不正かを含める。エラー時に `Config` は返さない。
- ファイルが存在しない
- `repos` が無い、または空（ファイルが空の場合を含む）
- `repos` の要素が「文字列」でも「`name` を持つマッピング」でもない
- `repos` の要素が `owner/name` 形式でない。判定は「`/` がちょうど 1 つ、その両側が空でない、空白を含まない」に限る
- `merge_method`（グローバル・リポジトリ別とも）が `squash` / `merge` / `rebase` 以外（リポジトリ別の空文字列は未指定扱いで対象外）
- `mode` が `sdd` / `label` 以外（空文字列は未指定扱いで対象外）
- `refresh_interval_sec` が 1 未満
- 未知のキーがある（トップレベル、および `repos` 要素のマッピング内のどちらも）

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
- **WHEN** `repos` の要素に `{name: org/board, mode: kanban}` と書いたファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列に `org/board` と `kanban` が含まれる

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
