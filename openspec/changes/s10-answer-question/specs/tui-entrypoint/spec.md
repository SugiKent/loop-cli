## MODIFIED Requirements

### Requirement: sugi-loop バイナリが起動して 1 フレーム描画する
`go build ./cmd/sugi-loop` は `sugi-loop` バイナリを MUST 生成し、端末で起動すると Bubble Tea のプログラムとして 1 フレームを MUST 描画する。
`main()` は次の順で起動する。
1. `config.DefaultPath()` で設定ファイルのパスを決め、`config.Load` で `Config` を読む（s02）
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
