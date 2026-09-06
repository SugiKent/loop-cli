## MODIFIED Requirements

### Requirement: ExternalEditor は設定のエディタを一時ファイルで開く
`internal/ui` は `ExternalEditor(command string) Editor` を MUST 公開する。`command` は config の `editor`（s02 が `$EDITOR` を展開済み）で、`cmd/loop-cli` がこれを `New` に渡す。返る `Editor` は次の振る舞いをする。
- `command` の前後の空白を除いた結果が空なら、外部プロセスを起動せず、`editor が設定されていません（config の editor か環境変数 EDITOR）` を含むエラーを運ぶメッセージを返すコマンドを返す（s02 `config-loading`「空のときの扱いは s10 が担当」。既定値は design.md の未決事項）
- それ以外は、一時ディレクトリに `loop-cli-answer-*.md` の一時ファイルを作って `initial` を書き込み、`command` を空白で分割した先頭を実行ファイル、残りを引数とし、その末尾に一時ファイルのパスを足したプロセスを、Bubble Tea の外部プロセス実行の仕組み（描画を止め、端末の標準入出力をプロセスに渡す）で起動する。シェルを通さない
- プロセスが終了コード 0 で終わったら一時ファイルを読み、その内容を編集後の本文としてメッセージで返し、一時ファイルを削除する。終了コードが非 0 か起動に失敗したら、そのエラーを運ぶメッセージを返し、一時ファイルは削除する

#### Scenario: editor が空なら起動せずエラーになる
- **WHEN** `ExternalEditor("")` が返す `Editor` を `initial` `Q1: A` で呼び、返ったコマンドを実行する
- **THEN** 返るメッセージはエラーを持ち、エラー文字列に `editor` を含む

#### Scenario: コマンドの分割と一時ファイル
- **WHEN** `ExternalEditor("code --wait")` が組み立てるプロセスの引数と一時ファイルを、`initial` `Q1: A\nQ2: B` で確認する
- **THEN** 引数は `code` `--wait` `<一時ファイルのパス>` の順で、パスの末尾は `.md`、そのファイルの内容は `Q1: A\nQ2: B` である

#### Scenario: 終了後に一時ファイルを読んで削除する
- **WHEN** 内容が `Q1: B` の一時ファイルに対して、終了コード 0 の完了として結果を読む処理を呼ぶ
- **THEN** メッセージの本文は `Q1: B` で、一時ファイルは存在しない

#### Scenario: 非 0 で終了したら投稿せずエラーになる
- **WHEN** 一時ファイルに対して、終了コード 1 のエラーを完了として結果を読む処理を呼ぶ
- **THEN** メッセージはそのエラーを持ち、一時ファイルは存在しない
