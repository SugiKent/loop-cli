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

### Requirement: 起動失敗は標準エラーに出て終了コード 1 になる
引数なしで実行したとき（TUI の起動）、`main()` は次のいずれかが失敗した場合、エラー内容を標準エラー出力に書き、終了コード 1 で MUST 終了する。文言は 1 行で、`Check` の失敗だけは原因と次の一手の 2 行（1 行ずつ改行で区切る）。panic しない。Bubble Tea のプログラムは起動しない（設定と `Check` の失敗は画面を出す前に分かる）。サブコマンドの失敗はそれぞれの Requirement が定めるが、設定ファイルと `Check` の文言はここで定めたものを使い回す（s33 `now-command`「now は設定ファイルを読み gh を呼ぶ」）。
- `config.DefaultPath` のエラー
- 設定ファイルの存在確認のエラー（s08a `onboarding`「設定ファイルが無いときだけ onboarding に入る」）: 標準入力が端末でない状態で設定ファイルが無ければ `設定ファイルがありません: <パス>`。存在確認が「存在しない」以外のエラーを返せばそのエラーとパス
- onboarding フォームの中止（`onboarding.ErrAborted`）: `設定の作成を中止しました（<パス> は書いていません）`。設定ファイルは書かれない
- onboarding の書き出し（`onboarding.Write`）のエラー: パスと原因
- `config.Load` のエラー（s02 のエラー文字列はパスと原因を含む。壊れた設定はここで終了し、onboarding に入らない）
- `Check` のエラー（s03）。`cmd/loop-cli` の非公開関数が s03 のエラーを次の 2 行に写す
  - `gh` が PATH に無い（`errors.Is(err, exec.ErrNotFound)`）: 1 行目 `gh が見つかりません`、2 行目 `https://cli.github.com/ から GitHub CLI をインストールしてください`
  - `gh auth status` が非 0（`*gh.Error`）: 1 行目 `gh の認証に失敗しました: <Stderr を TrimSpace したもの>`、2 行目 `gh auth login を実行してください`
  - それ以外（`ctx` の中断など）: s03 のエラー文字列を 1 行
- Bubble Tea のプログラム実行のエラー（端末が無い環境での起動など）
取得（`fetch.Fetch`）の失敗は起動失敗として扱わず、s08 `queue-screen` の Requirement「取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する」に従い画面に赤で出す。設定ファイルが無く標準入力が端末の場合は `onboarding` のフォームに入り、フォームが完了すれば通常どおり起動する。

#### Scenario: 端末でない環境で設定ファイルが無い
- **WHEN** `$HOME/.config/loop-cli/config.yml` が無く、標準入力が端末でない状態でバイナリを実行する
- **THEN** 標準エラー出力に `設定ファイルがありません` と設定ファイルのパスを含む行が 1 行出力され、終了コードは 1 で、フォームも画面も描画されない

#### Scenario: 設定ファイルが壊れている
- **WHEN** `$HOME/.config/loop-cli/config.yml` の内容が `repos: [org/app` の状態でバイナリを実行する
- **THEN** 標準エラー出力に設定ファイルのパスと原因を含むエラーが 1 行出力され、終了コードは 1 で、フォームは出ず、ファイルは元の内容のまま残る

#### Scenario: フォームを中止する
- **WHEN** 設定ファイルが無い端末でバイナリを実行し、フォームで Ctrl+C を押す
- **THEN** 標準エラー出力に `設定の作成を中止しました` と設定ファイルのパスを含む行が出力され、終了コードは 1 で、設定ファイルは作られない

#### Scenario: gh が無い
- **WHEN** `exec.ErrNotFound` を `%w` で包んだエラー（s03 の `Check` が `gh` を見つけられないときに返す形）を `Check` 失敗の文言にする関数に渡す
- **THEN** 返る文字列は改行で区切った 2 行で、1 行目に `gh が見つかりません`、2 行目に `https://cli.github.com/` を含む

#### Scenario: gh が認証されていない
- **WHEN** `&gh.Error{Args: []string{"auth", "status"}, ExitCode: 1, Stderr: "X Failed to log in to github.com using token (GH_TOKEN)\n"}` を `Check` 失敗の文言にする関数に渡す
- **THEN** 返る文字列は 2 行で、1 行目に `gh の認証に失敗しました` と `X Failed to log in to github.com using token (GH_TOKEN)`、2 行目に `gh auth login` を含む

#### Scenario: Check のそれ以外の失敗は 1 行
- **WHEN** `context.DeadlineExceeded` を包んだエラーを `Check` 失敗の文言にする関数に渡す
- **THEN** 返る文字列に改行は無く、元のエラー文字列を含む

#### Scenario: 端末が無い環境で起動する
- **WHEN** 設定と `Check` は通るが、標準入出力が端末でない状態でバイナリを実行し、プログラム実行がエラーを返す
- **THEN** 標準エラー出力にエラー内容が 1 行出力され、終了コードは 1 である
