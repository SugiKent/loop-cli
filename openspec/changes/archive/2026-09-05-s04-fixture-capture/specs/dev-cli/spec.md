## ADDED Requirements

### Requirement: sugi-loop-cli はサブコマンドを振り分け help で使い方を出す
`go build ./cmd/sugi-loop-cli` は `sugi-loop-cli` バイナリを MUST 生成する。`cmd/sugi-loop-cli` は動作確認用の CLI（implementation-tasks.md §2）で、標準ライブラリの `flag` と `os.Args` だけで書き、CLI フレームワークを依存に加えない。
第 1 引数をサブコマンド名として振り分ける。`help` / `-h` / `--help`、または引数なしのときは使い方を標準出力に書き、終了コード 0 で終わる。使い方には `help` と `fixture capture --repo owner/name --alias <alias>` の 1 行説明を含める。
未知のサブコマンド、または `fixture` の後に `capture` 以外（無しを含む）が続く場合は、`unknown command: <args[0]>`（第 1 引数のみ。`fixture foo` でも `fixture` 単独でも `unknown command: fixture`）と使い方を標準エラーに書き、終了コード 1 で終わる。
振り分けは `run(args []string, stdout, stderr io.Writer) int` のような関数で行い、`main` はその返り値で `os.Exit` する（テストがプロセスを起動せずに検証するため）。この change で定義するサブコマンドは `help` と `fixture capture` だけである。`classify` / `notify test` は s06 が ADDED で足す。

#### Scenario: help が使い方を出す
- **WHEN** 引数 `help` で実行する
- **THEN** 標準出力に `fixture capture` と `--repo` と `--alias` を含む使い方が出て、終了コードは 0 である

#### Scenario: 引数なしも使い方を出す
- **WHEN** 引数なしで実行する
- **THEN** 標準出力に使い方が出て、終了コードは 0 である

#### Scenario: 未知のサブコマンド
- **WHEN** 引数 `frobnicate` で実行する
- **THEN** 標準エラーに `unknown command: frobnicate` と使い方が出て、終了コードは 1 である

#### Scenario: fixture の後にサブコマンドが無い
- **WHEN** 引数 `fixture` だけで実行する
- **THEN** 標準エラーに `unknown command: fixture` と使い方が出て、終了コードは 1 である

#### Scenario: fixture の後が capture 以外
- **WHEN** 引数 `fixture foo` で実行する
- **THEN** 標準エラーに `unknown command: fixture` と使い方が出て、終了コードは 1 である

### Requirement: サブコマンドの失敗は標準エラーに出て終了コード 1 になる
サブコマンドがエラーを返した場合、CLI はエラー文字列を標準エラーに 1 行書き、終了コード 1 で MUST 終わる。panic しない。フラグの解析エラー（未知のフラグ・必須フラグの欠落）も同じ扱いにする。成功時は終了コード 0 である。

#### Scenario: 必須フラグの欠落
- **WHEN** 引数 `fixture capture --repo org/app`（`--alias` 無し）で実行する
- **THEN** 標準エラーに `--alias` を含むエラーが出て、終了コードは 1 である

#### Scenario: 未知のフラグ
- **WHEN** 引数 `fixture capture --repo org/app --alias app --out x` で実行する
- **THEN** 標準エラーに `out` を含むエラーが出て、終了コードは 1 である
