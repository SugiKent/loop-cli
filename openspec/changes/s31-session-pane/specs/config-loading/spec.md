## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: 設定ファイルを読み込んで Config 型を返す
`internal/config` はパスを受け取り `Config` 型を返す関数 `Load` を MUST 提供する。`Config` は mvp.md「設定ファイル」節の 5 項目
`repos` / `refresh_interval_sec` / `merge_method` / `editor` / `notify` だけを持ち、`repos` の各要素はリポジトリ名 `owner/name` と
そのリポジトリで使う merge 方式、そのリポジトリの Routine を回している Claude のプロファイルのパス（s31 の `claude_config_dir`。Requirement「repos の要素は claude_config_dir を持てる」が定める）を持つ。`merge_method` の値は `squash` / `merge` / `rebase` の 3 つで、`gh pr merge` のフラグ名と一致する型付き定数として公開する。

#### Scenario: mvp.md の例をそのまま読む
- **WHEN** mvp.md「設定ファイル」節の YAML 例（`repos` に `org/app` と `org/web`、`refresh_interval_sec: 120`、`merge_method: squash`、`editor: $EDITOR`、`notify: true`）を `Load` に渡す
- **THEN** `Repos` は `org/app` と `org/web` の 2 件がこの順で入り、`RefreshIntervalSec` は 120、`MergeMethod` は squash、`Notify` は true、`Editor` は環境変数 `EDITOR` の値になる

#### Scenario: YAML の構文が壊れている
- **WHEN** YAML として解析できない内容のファイルを `Load` に渡す
- **THEN** エラーが返り、エラー文字列にファイルパスが含まれる
