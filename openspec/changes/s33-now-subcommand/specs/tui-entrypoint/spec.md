## MODIFIED Requirements

### Requirement: 引数はサブコマンドに振り分ける

`cmd/loop-cli` は第 1 引数をサブコマンド名として MUST 振り分ける。振り分けは `run(args []string, stdout, stderr io.Writer) int` で行い、標準ライブラリの `os.Args` だけを使う（CLI フレームワークを依存に加えない）。

- 引数なし: 今までどおり TUI を起動する
- `version`: s23 `self-update`「version サブコマンドは現在の版を 1 行出す」に従う
- `update`: s23 `self-update`「update は新しい版があるときだけ go install で入れ直す」に従う
- `now`: s33 `now-command`「now サブコマンドは今やるのカードだけを JSON で出す」に従う
- それ以外（フラグを含む）: `unknown command: <args[0]>` と使い方を標準エラーに書き、終了コード 1 で終わる。使い方には `version` と `update` と `now` の 1 行説明を含める

`version` と `update` は設定ファイルを読まず、`gh` も呼ばない（`gh` が未認証でも `update` は動く）。`now` は設定ファイルの `repos` を対象にして `gh` を呼ぶ（s33 `now-command`「now は設定ファイルを読み gh を呼ぶ」）が、設定ファイルが無くても onboarding のフォームには入らない。

#### Scenario: 引数なしは TUI を起動する
- **WHEN** 引数なしで実行する
- **THEN** キュー画面が描画される

#### Scenario: 未知のサブコマンド
- **WHEN** 引数 `frobnicate` で実行する
- **THEN** 標準エラーに `unknown command: frobnicate` と `version` と `update` と `now` を含む使い方が出て、終了コードは 1 である

#### Scenario: update は設定ファイルが無くても動く
- **WHEN** `$HOME/.config/loop-cli/config.yml` が無い状態で引数 `update` を実行する
- **THEN** onboarding のフォームは出ず、更新の処理が実行される

#### Scenario: now は設定ファイルが無ければフォームを出さずに終わる
- **WHEN** `$HOME/.config/loop-cli/config.yml` が無い状態で引数 `now` を実行する
- **THEN** onboarding のフォームは出ず、標準エラーに `設定ファイルがありません` とそのパスが出て、終了コードは 1 である
