# tui-entrypoint Specification

## Purpose
TBD - created by archiving change s01-bootstrap. Update Purpose after archive.
## Requirements
### Requirement: sugi-loop バイナリが起動して 1 フレーム描画する
`go build ./cmd/sugi-loop` は `sugi-loop` バイナリを MUST 生成し、端末で起動すると Bubble Tea のプログラムとして 1 フレームを MUST 描画する。
`main()` は次の順で起動する。
1. `config.DefaultPath()` で設定ファイルのパスを決め、`config.Load` で `Config` を読む（s02）
2. `gh.NewClient()` を作り、`Check(ctx)` で `gh` の存在と認証を確認する（s03。`ctx` は `context.Background()`）
3. `Config.Repos[].Name` を並べた `repos` と `client` で `fetch.Fetch`（s07）を閉じた `ui.Fetcher` を `ui.New` に渡し、Bubble Tea のプログラムとして実行する
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
`main()` は次のいずれかが失敗した場合、エラー内容を標準エラー出力に 1 行書き、終了コード 1 で MUST 終了する。panic しない。Bubble Tea のプログラムは起動しない（設定と `Check` の失敗は画面を出す前に分かる）。
- `config.DefaultPath` または `config.Load` のエラー（s02 のエラー文字列はパスと原因を含む）
- `Check` のエラー（s03 のエラー文字列は `gh` のインストールを促す文言、または `gh auth status` の stderr を含む）
- Bubble Tea のプログラム実行のエラー（端末が無い環境での起動など）
取得（`fetch.Fetch`）の失敗は起動失敗ではなく、s08 `queue-screen` の Requirement「取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する」に従い画面に赤で出す。

#### Scenario: 設定ファイルが無い
- **WHEN** `$HOME/.config/sugi-loop/config.yml` が無い状態でバイナリを実行する
- **THEN** 標準エラー出力に設定ファイルのパスを含むエラーが 1 行出力され、終了コードは 1 で、画面は描画されない

#### Scenario: gh が認証されていない
- **WHEN** 設定ファイルはあるが `gh auth status` が非 0 で終わる状態でバイナリを実行する
- **THEN** 標準エラー出力に `gh auth status` の stderr を含むエラーが 1 行出力され、終了コードは 1 で、画面は描画されない

#### Scenario: 端末が無い環境で起動する
- **WHEN** 設定と `Check` は通るが、標準入出力が端末でない状態でバイナリを実行し、プログラム実行がエラーを返す
- **THEN** 標準エラー出力にエラー内容が 1 行出力され、終了コードは 1 である

