# config-loading Specification

## Purpose
TBD - created by archiving change s02-config. Update Purpose after archive.
## Requirements
### Requirement: 設定ファイルのパスが決まる
`internal/config` は設定ファイルの既定パスを返す関数 `DefaultPath` を MUST 提供する。返すパスは design.md の未決事項で定めた既定値
（ユーザーのホームディレクトリ直下の `.config/loop-cli/config.yml`）である。ホームディレクトリが決定できない場合はエラーを返す。

#### Scenario: 既定パスがホーム直下の .config になる
- **WHEN** `HOME` が `/Users/alice` の状態で `DefaultPath` を呼ぶ
- **THEN** `/Users/alice/.config/loop-cli/config.yml` が返る

#### Scenario: ホームディレクトリが決まらない
- **WHEN** ホームディレクトリを決定できない環境で `DefaultPath` を呼ぶ
- **THEN** エラーが返り、パスは返らない

### Requirement: 設定ファイルを読み込んで Config 型を返す
`internal/config` はパスを受け取り `Config` 型を返す関数 `Load` を MUST 提供する。`Config` は mvp.md「設定ファイル」節の 5 項目
`repos` / `refresh_interval_sec` / `merge_method` / `editor` / `notify` だけを持ち、`repos` の各要素はリポジトリ名 `owner/name` と
そのリポジトリで使う merge 方式を持つ。`merge_method` の値は `squash` / `merge` / `rebase` の 3 つで、`gh pr merge` のフラグ名と一致する型付き定数として公開する。

#### Scenario: mvp.md の例をそのまま読む
- **WHEN** mvp.md「設定ファイル」節の YAML 例（`repos` に `org/app` と `org/web`、`refresh_interval_sec: 120`、`merge_method: squash`、`editor: $EDITOR`、`notify: true`）を `Load` に渡す
- **THEN** `Repos` は `org/app` と `org/web` の 2 件がこの順で入り、`RefreshIntervalSec` は 120、`MergeMethod` は squash、`Notify` は true、`Editor` は環境変数 `EDITOR` の値になる

#### Scenario: YAML の構文が壊れている
- **WHEN** YAML として解析できない内容のファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列にファイルパスが含まれる

### Requirement: 省略した項目に既定値が入る
`repos` 以外の項目は省略できる。省略時は `refresh_interval_sec` が 120、`merge_method` が squash、`notify` が true、`editor` が環境変数 `EDITOR` の値に MUST なる。
明示した値は既定値より優先される。`notify: false` の明示は省略と区別し、false のまま MUST 保持する。

#### Scenario: repos だけのファイル
- **WHEN** `repos` に 1 件だけ書き、他の項目を省略したファイルを `Load` に渡す
- **THEN** `RefreshIntervalSec` は 120、`MergeMethod` は squash、`Notify` は true、`Editor` は環境変数 `EDITOR` の値になる

#### Scenario: notify を false と明示する
- **WHEN** `notify: false` と書いたファイルを `Load` に渡す
- **THEN** `Notify` は false になる

#### Scenario: 明示した値が既定値を上書きする
- **WHEN** `refresh_interval_sec: 30`、`merge_method: rebase`、`editor: vim` と書いたファイルを `Load` に渡す
- **THEN** `RefreshIntervalSec` は 30、`MergeMethod` は rebase、`Editor` は `vim` になる

### Requirement: editor は環境変数を展開する
`editor` の値に含まれる `$NAME` 形式の環境変数参照は、`Load` の時点で展開 MUST される（mvp.md の例 `editor: $EDITOR` をそのまま書いて動くため）。
展開結果が空文字列でも `Load` は失敗しない。空のときの扱いは `Editor` を消費する後続 change（s10）が担当し、ここでは定義しない。

#### Scenario: $EDITOR が展開される
- **WHEN** 環境変数 `EDITOR` が `nvim` の状態で `editor: $EDITOR` と書いたファイルを `Load` に渡す
- **THEN** `Editor` は `nvim` になる

#### Scenario: EDITOR が未設定または空でも Load は成功する
- **WHEN** 環境変数 `EDITOR` が未設定または空の状態で `editor` を省略したファイルを `Load` に渡す
- **THEN** `Load` はエラーを返さず、`Editor` は空文字列になる

### Requirement: 不正な設定はパスと原因を含むエラーになる
`Load` は次の場合にエラーを MUST 返す。エラー文字列にはファイルパスと、どの項目がなぜ不正かを含める。エラー時に `Config` は返さない。
- ファイルが存在しない
- `repos` が無い、または空（ファイルが空の場合を含む）
- `repos` の要素が「文字列」でも「`name` を持つマッピング」でもない
- `repos` の要素が `owner/name` 形式でない。判定は「`/` がちょうど 1 つ、その両側が空でない、空白を含まない」に限る
- `merge_method`（グローバル・リポジトリ別とも）が `squash` / `merge` / `rebase` 以外（リポジトリ別の空文字列は未指定扱いで対象外）
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

### Requirement: リポジトリ別に merge_method を上書きできる
`repos` の要素は design.md の未決事項で定めた既定の表現でリポジトリ別の `merge_method` を MUST 指定できる。`Load` は各リポジトリの merge 方式を
「リポジトリ別の指定があればそれ、無ければグローバルの `merge_method`」に解決して `Repo.MergeMethod` に埋めて返す。リポジトリ別の `merge_method` に空文字列を明示した場合は未指定として扱い、グローバル値になる。消費側（s14）は上書きの有無を知らずに
`Repo.MergeMethod` を読めばよい。

#### Scenario: 上書きの無いリポジトリはグローバル値になる
- **WHEN** `merge_method: squash` と、上書き指定の無い `org/app` を書いたファイルを `Load` に渡す
- **THEN** `org/app` の `MergeMethod` は squash になる

#### Scenario: 上書きしたリポジトリだけ別の方式になる
- **WHEN** `merge_method: squash` と、`org/app`（上書き無し）、`org/web`（rebase を指定）を書いたファイルを `Load` に渡す
- **THEN** `org/app` の `MergeMethod` は squash、`org/web` の `MergeMethod` は rebase になる

#### Scenario: リポジトリ別の merge_method が不正
- **WHEN** `org/web` に `merge_method: ff` を指定したファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列に `org/web` と `ff` が含まれる

### Requirement: 設定ファイルは認証情報を持たない
`Config` 型は認証トークンや認証情報のフィールドを MUST 持たない。認証は `gh auth` を再利用する（mvp.md）。設定ファイルに `token` 等の認証キーを書いた場合は
未知のキーとしてエラーになる。

#### Scenario: token キーを書くとエラーになる
- **WHEN** `token: ghp_xxx` と書いたファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列に `token` が含まれる

### Requirement: リポジトリ名の検証規則を公開する
`internal/config` は関数 `IsRepoName(name string) bool` を MUST 提供する。`owner/name` の形式（`/` がちょうど 1 つ、`owner` と `name` が空でない、空白とタブを含まない）なら true。`Load` の検証（「不正な設定はパスと原因を含むエラーになる」）はこの関数を使い、onboarding のフォームも同じ関数で各行を検証する。規則を 2 か所に書かない。

#### Scenario: owner/name 形式を受け入れる
- **WHEN** `IsRepoName("org/app")` を呼ぶ
- **THEN** true が返る

#### Scenario: owner/name でない文字列を拒否する
- **WHEN** `IsRepoName("app")`、`IsRepoName("org/app/extra")`、`IsRepoName("/app")`、`IsRepoName("org/")`、`IsRepoName("org/ app")` をそれぞれ呼ぶ
- **THEN** いずれも false が返る

