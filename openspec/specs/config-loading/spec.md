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
`repos` / `refresh_interval_sec` / `merge_method` / `editor` / `notify` と、s30-other-grace で足した `other_grace_min` の 6 項目だけを持ち、`repos` の各要素はリポジトリ名 `owner/name` と
そのリポジトリで使う merge 方式、そのリポジトリの Routine を回している Claude のプロファイルのパス（s31 の `claude_config_dir`。Requirement「repos の要素は claude_config_dir を持てる」が定める）を持つ。`merge_method` の値は `squash` / `merge` / `rebase` の 3 つで、`gh pr merge` のフラグ名と一致する型付き定数として公開する。
`other_grace_min` は「その他」の PR を進行中に置く猶予を分で表す整数で、`Config.OtherGraceMin` に入る（消費側は `cmd/loop-cli` が分から `time.Duration` に変えて s07 `Fetch` に渡す）。mvp.md は凍結されているので例に載せず、利用者向けの説明は README の設定表に書く。

#### Scenario: mvp.md の例をそのまま読む
- **WHEN** mvp.md「設定ファイル」節の YAML 例（`repos` に `org/app` と `org/web`、`refresh_interval_sec: 120`、`merge_method: squash`、`editor: $EDITOR`、`notify: true`）を `Load` に渡す
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
- `other_grace_min` が 0 未満
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

### Requirement: repos の要素は claude_config_dir を持てる

`repos` の要素のマッピングは、`name` / `merge_method` に加えてキー `claude_config_dir` を MUST 受け付ける。値はそのリポジトリの Routine を回している Claude のプロファイルのパス（`CLAUDE_CONFIG_DIR` に渡す値）で、`Load` は次のとおり扱う。

- 値の中の `$NAME` 形式の環境変数参照を展開する（`editor` と同じ扱い）
- 展開後の値が `~` で始まるときは、`~` をユーザーのホームディレクトリに置き換える。`~alice` のような他人のホームの書き方は展開しない
- 省略できる。省略したリポジトリの `claude_config_dir` は空文字列になり、消費側（s31 `session-pane`）はそのリポジトリのセッションを取得しない
- グローバルの既定値は持たない。`~/.claude` を使いたい場合もそのパスを明示して書く（プロファイルを書き忘れたリポジトリで、意図しないプロファイルの利用枠を使わないため）
- 展開後のパスが存在するかどうかは `Load` では MUST 検証しない（設定ファイルを別のマシンから持ってきた場合に起動できなくなるため。取得に失敗したことは s31 `session-pane` が右ペインに出す）

#### Scenario: claude_config_dir を読んでホームを展開する
- **WHEN** `HOME` が `/Users/alice` の状態で、`repos` の要素に `{name: org/app, claude_config_dir: ~/.claude-personal}` と書いたファイルを `Load` に渡す
- **THEN** `org/app` の `claude_config_dir` は `/Users/alice/.claude-personal` になる

#### Scenario: 環境変数を展開する
- **WHEN** 環境変数 `CC_PROFILE` が `/opt/profiles/max` の状態で、`repos` の要素に `{name: org/app, claude_config_dir: $CC_PROFILE}` と書いたファイルを `Load` に渡す
- **THEN** `org/app` の `claude_config_dir` は `/opt/profiles/max` になる

#### Scenario: 省略したリポジトリは空になる
- **WHEN** `repos` に文字列の `org/web` と `{name: org/app, claude_config_dir: ~/.claude-max}` を書いたファイルを `Load` に渡す
- **THEN** `org/web` の `claude_config_dir` は空文字列、`org/app` の `claude_config_dir` は展開されたパスになる

#### Scenario: 存在しないパスでも Load は成功する
- **WHEN** `repos` の要素に存在しないディレクトリを `claude_config_dir` として書いたファイルを `Load` に渡す
- **THEN** `Load` はエラーを返さず、`claude_config_dir` はそのパスのままになる
