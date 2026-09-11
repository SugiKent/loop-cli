## MODIFIED Requirements

### Requirement: 設定ファイルを読み込んで Config 型を返す
`internal/config` はパスを受け取り `Config` 型を返す関数 `Load` を MUST 提供する。`Config` は mvp.md「設定ファイル」節の 6 項目
`repos` / `refresh_interval_sec` / `merge_method` / `editor` / `notify` / `other_grace_min` だけを持ち、`repos` の各要素はリポジトリ名 `owner/name` と
そのリポジトリで使う merge 方式を持つ。`merge_method` の値は `squash` / `merge` / `rebase` の 3 つで、`gh pr merge` のフラグ名と一致する型付き定数として公開する。
`other_grace_min` は「その他」を進行中に置く猶予を分で表す整数で、`Config.OtherGraceMin` に入る（消費側は `cmd/loop-cli` が分から `time.Duration` に変えて s07 `Fetch` に渡す）。

#### Scenario: mvp.md の例をそのまま読む
- **WHEN** mvp.md「設定ファイル」節の YAML 例（`repos` に `org/app` と `org/web`、`refresh_interval_sec: 120`、`merge_method: squash`、`editor: $EDITOR`、`notify: true`、`other_grace_min: 30`）を `Load` に渡す
- **THEN** `Repos` は `org/app` と `org/web` の 2 件がこの順で入り、`RefreshIntervalSec` は 120、`MergeMethod` は squash、`Notify` は true、`OtherGraceMin` は 30、`Editor` は環境変数 `EDITOR` の値になる

#### Scenario: YAML の構文が壊れている
- **WHEN** YAML として解析できない内容のファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列にファイルパスが含まれる

### Requirement: 省略した項目に既定値が入る
`repos` 以外の項目は省略できる。省略時は `refresh_interval_sec` が 120、`merge_method` が squash、`notify` が true、`other_grace_min` が 30、`editor` が環境変数 `EDITOR` の値に MUST なる。
明示した値は既定値より優先される。`notify: false` の明示は省略と区別し、false のまま MUST 保持する。`other_grace_min: 0` の明示も省略と区別し、0（猶予なし）のまま MUST 保持する。

#### Scenario: repos だけのファイル
- **WHEN** `repos` に 1 件だけ書き、他の項目を省略したファイルを `Load` に渡す
- **THEN** `RefreshIntervalSec` は 120、`MergeMethod` は squash、`Notify` は true、`OtherGraceMin` は 30、`Editor` は環境変数 `EDITOR` の値になる

#### Scenario: notify を false と明示する
- **WHEN** `notify: false` と書いたファイルを `Load` に渡す
- **THEN** `Notify` は false になる

#### Scenario: other_grace_min を 0 と明示する
- **WHEN** `other_grace_min: 0` と書いたファイルを `Load` に渡す
- **THEN** `OtherGraceMin` は 0 になる（30 に戻らない）

#### Scenario: 明示した値が既定値を上書きする
- **WHEN** `refresh_interval_sec: 30`、`merge_method: rebase`、`editor: vim`、`other_grace_min: 5` と書いたファイルを `Load` に渡す
- **THEN** `RefreshIntervalSec` は 30、`MergeMethod` は rebase、`Editor` は `vim`、`OtherGraceMin` は 5 になる

### Requirement: 不正な設定はパスと原因を含むエラーになる
`Load` は次の場合にエラーを MUST 返す。エラー文字列にはファイルパスと、どの項目がなぜ不正かを含める。エラー時に `Config` は返さない。
- ファイルが存在しない
- `repos` が無い、または空（ファイルが空の場合を含む）
- `repos` の要素が「文字列」でも「`name` を持つマッピング」でもない
- `repos` の要素が `owner/name` 形式でない。判定は「`/` がちょうど 1 つ、その両側が空でない、空白を含まない」に限る
- `merge_method`（グローバル・リポジトリ別とも）が `squash` / `merge` / `rebase` 以外（リポジトリ別の空文字列は未指定扱いで対象外）
- `mode` が `sdd` / `label` 以外（空文字列は未指定扱いで対象外）
- `refresh_interval_sec` が 1 未満
- `other_grace_min` が 0 未満
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

#### Scenario: other_grace_min が負数
- **WHEN** `other_grace_min: -1` と書いたファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列に `other_grace_min` が含まれる

#### Scenario: 未知のトップレベルキーがある
- **WHEN** `refresh_interval: 60`（正しくは `refresh_interval_sec`）と書いたファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列に `refresh_interval` が含まれる

#### Scenario: repos 要素のマッピングに未知のキーがある
- **WHEN** `repos` の要素に `{name: org/web, merge_methd: rebase}`（正しくは `merge_method`）と書いたファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列に `merge_methd` が含まれる。グローバル値へ黙って落とさない

#### Scenario: repos 要素が文字列でも name 付きマッピングでもない
- **WHEN** `repos` の要素に `{merge_method: rebase}`（`name` 無し）と書いたファイル、`[org/app]`（シーケンス）と書いたファイルのそれぞれを `Load` に渡す
- **THEN** いずれもエラーが返り、エラー文字列に `repos` が含まれる
