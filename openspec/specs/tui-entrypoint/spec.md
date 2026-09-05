# tui-entrypoint Specification

## Purpose
TBD - created by archiving change s01-bootstrap. Update Purpose after archive.
## Requirements
### Requirement: sugi-loop バイナリが起動して 1 フレーム描画する
`go build ./cmd/sugi-loop` は `sugi-loop` バイナリを MUST 生成し、端末で起動すると Bubble Tea のプログラムとして 1 フレームを MUST 描画する。
描画内容にはアプリ名 `sugi-loop` と、`q` で終了できることを示す案内文を含める。
このフレームは hello world であり、mvp.md のキュー画面ではない（キュー画面は s08 が担当）。

#### Scenario: 初期フレームにアプリ名と終了案内が出る
- **WHEN** モデルの初期状態から描画文字列を得る
- **THEN** 文字列に `sugi-loop` と `q` を含む終了案内が含まれる

#### Scenario: バイナリが生成される
- **WHEN** `go build -o bin/sugi-loop ./cmd/sugi-loop` を実行する
- **THEN** 終了コード 0 で `bin/sugi-loop` が生成される

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
Bubble Tea のプログラム実行がエラーを返した場合（端末が無い環境での起動など）、エラー内容を標準エラー出力に書き、終了コード 1 で MUST 終了する。panic しない。

#### Scenario: 端末が無い環境で起動する
- **WHEN** 標準入出力が端末でない状態でバイナリを実行し、プログラム実行がエラーを返す
- **THEN** 標準エラー出力にエラー内容が 1 行出力され、終了コードは 1 である

