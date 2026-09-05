# tui-entrypoint Specification

## Purpose
TBD - created by archiving change s01-bootstrap. Update Purpose after archive.
## Requirements
### Requirement: sugi-loop バイナリが起動して 1 フレーム描画する
`go build ./cmd/sugi-loop` は `sugi-loop` バイナリを MUST 生成し、端末で起動すると Bubble Tea のプログラムとして 1 フレームを MUST 描画する。
`main()` は次の順で起動する。
1. `config.DefaultPath()` で設定ファイルのパスを決め、そのパスにファイルが無く標準入力が端末なら s08a `onboarding` のフォームで設定ファイルを作ってから（「設定ファイルが無いときだけ onboarding に入る」）、`config.Load` で `Config` を読む（s02）
2. `gh.NewClient()` を作り、`Check(ctx)` で `gh` の存在と認証を確認する（s03。`ctx` は `context.Background()`）
3. `Config.Repos[].Name` を並べた `repos` と `client` で `fetch.Fetch`（s07）を閉じた `ui.Fetcher` と、同じ `client`、`ui.ExternalEditor(Config.Editor)`（s10 `answer-question`。`Config.Editor` は s02 が `$EDITOR` を展開済みで、空でもここでは失敗させない）を `ui.New` に渡し、Bubble Tea のプログラムとして実行する
最初のフレームは s08 `queue-screen` のキュー画面であり、`Init` が返す取得コマンドが完了するまでは `Cards` が空の表と `取得中` のスピナーを描く。取得の完了で表が埋まる。hello world の画面（s01）は無くなる。
描画内容にはアプリ名 `sugi-loop` と、`q` で終了できることを示すフッタのヒントを含める。

#### Scenario: 初期フレームにアプリ名と終了案内が出る
- **WHEN** `ui.New` で作った Model の初期状態から描画文字列を得る
- **THEN** 文字列に `sugi-loop` と `q 終了` が含まれる

#### Scenario: バイナリが生成される
- **WHEN** `go build -o bin/sugi-loop ./cmd/sugi-loop` を実行する
- **THEN** 終了コード 0 で `bin/sugi-loop` が生成される

#### Scenario: 起動直後に取得が始まる
- **WHEN** 設定ファイルがあり `gh auth status` が通る端末で `sugi-loop` を起動する
- **THEN** 最初のフレームはヘッダ `[1]今やる 0 …` と `↻ --:--`、フッタの `取得中` を含むキュー画面で、`fetch.Fetch` の完了後に設定リポジトリの Card がタブに並ぶ

#### Scenario: editor が空でも起動する
- **WHEN** 環境変数 `EDITOR` が未設定で `editor` を省略した設定ファイルがあり、`gh auth status` が通る端末で `sugi-loop` を起動する
- **THEN** キュー画面が描画される（`a` を押したときに初めて `editor が設定されていません` のエラーがフッタに出る）

#### Scenario: 設定ファイルが無い端末ではフォームの後にキュー画面が出る
- **WHEN** `$HOME/.config/sugi-loop/config.yml` が無く、`gh auth status` が通る端末で `sugi-loop` を起動し、フォームに `org/app` を入力して完了する
- **THEN** `$HOME/.config/sugi-loop/config.yml` が `repos:\n  - org/app\nrefresh_interval_sec: 120\nmerge_method: squash\neditor: $EDITOR\nnotify: true\n` の内容で作られ、続けてキュー画面が描画される

### Requirement: q で終了する
起動中に `q` または `Ctrl+C` を押すと、プログラムは終了コード 0 で MUST 終了し、端末を元の状態に戻す。それ以外のキーでは終了しない。

#### Scenario: q で終了が要求される
- **WHEN** モデルに `q` のキー入力メッセージを渡す
- **THEN** 返り値として終了コマンドが返る

#### Scenario: Ctrl+C で終了が要求される
- **WHEN** モデルに `Ctrl+C` のキー入力メッセージを渡す
- **THEN** 返り値として終了コマンドが返る

#### Scenario: 他のキーでは終了しない
- **WHEN** モデルに `j` のキー入力メッセージを渡す
- **THEN** 終了コマンドは返らず、モデルは描画可能なまま残る

### Requirement: 起動失敗は標準エラーに出て終了コード 1 になる
`main()` は次のいずれかが失敗した場合、エラー内容を標準エラー出力に書き、終了コード 1 で MUST 終了する。文言は 1 行で、`Check` の失敗だけは原因と次の一手の 2 行（1 行ずつ改行で区切る）。panic しない。Bubble Tea のプログラムは起動しない（設定と `Check` の失敗は画面を出す前に分かる）。
- `config.DefaultPath` のエラー
- 設定ファイルの存在確認のエラー（s08a `onboarding`「設定ファイルが無いときだけ onboarding に入る」）: 標準入力が端末でない状態で設定ファイルが無ければ `設定ファイルがありません: <パス>`。存在確認が「存在しない」以外のエラーを返せばそのエラーとパス
- onboarding フォームの中止（`onboarding.ErrAborted`）: `設定の作成を中止しました（<パス> は書いていません）`。設定ファイルは書かれない
- onboarding の書き出し（`onboarding.Write`）のエラー: パスと原因
- `config.Load` のエラー（s02 のエラー文字列はパスと原因を含む。壊れた設定はここで終了し、onboarding に入らない）
- `Check` のエラー（s03）。`cmd/sugi-loop` の非公開関数が s03 のエラーを次の 2 行に写す
  - `gh` が PATH に無い（`errors.Is(err, exec.ErrNotFound)`）: 1 行目 `gh が見つかりません`、2 行目 `https://cli.github.com/ から GitHub CLI をインストールしてください`
  - `gh auth status` が非 0（`*gh.Error`）: 1 行目 `gh の認証に失敗しました: <Stderr を TrimSpace したもの>`、2 行目 `gh auth login を実行してください`
  - それ以外（`ctx` の中断など）: s03 のエラー文字列を 1 行
- Bubble Tea のプログラム実行のエラー（端末が無い環境での起動など）
取得（`fetch.Fetch`）の失敗は起動失敗ではなく、s08 `queue-screen` の Requirement「取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する」に従い画面に赤で出す。設定ファイルが無く標準入力が端末の場合は `onboarding` のフォームに入り、フォームが完了すれば通常どおり起動する。

#### Scenario: 端末でない環境で設定ファイルが無い
- **WHEN** `$HOME/.config/sugi-loop/config.yml` が無く、標準入力が端末でない状態でバイナリを実行する
- **THEN** 標準エラー出力に `設定ファイルがありません` と設定ファイルのパスを含む行が 1 行出力され、終了コードは 1 で、フォームも画面も描画されない

#### Scenario: 設定ファイルが壊れている
- **WHEN** `$HOME/.config/sugi-loop/config.yml` の内容が `repos: [org/app` の状態でバイナリを実行する
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

