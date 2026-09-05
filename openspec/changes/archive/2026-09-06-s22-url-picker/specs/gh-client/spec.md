## ADDED Requirements

### Requirement: OpenURL は任意の URL を OS のブラウザで開く

`internal/gh` の `GHClient` interface は `OpenURL(ctx context.Context, url string) error` を MUST 持つ（s03「`GHClient` interface が読み取りと書き込みのメソッドを定義する」が定めた「後続 change が必要とするメソッドは、その change が ADDED で Requirement を足して interface・`Client`・`Fake` を同時に拡張する」に従う）。`Browse` はリポジトリと番号しか受け取れず、本文に書かれた URL を開けないため、`u`（s22 `url-picker`）はこのメソッドを使う。

`Client` の `OpenURL` は `gh` を実行せず、OS のブラウザ起動コマンドを MUST 実行する。`Client` は `gh` を起動する差し替え可能な関数とは別に、ブラウザ起動コマンドを実行する差し替え可能な関数を持ち、既定の実装は `runtime.GOOS` が `darwin` なら `open <url>`、それ以外なら `xdg-open <url>` を、`ctx` 付きでシェルを経由せずに実行する。`url` は検証せず、呼び出し側が渡した文字列をそのまま 1 つの引数として渡す。標準出力は読み捨てる。

終了コードが 0 でなければ、`<コマンド名> <url>: exit <コード>: <stderr>` の形式（s03「`gh` の失敗はコマンドと stderr を含むエラーになる」と同じ並び）のエラーを MUST 返す。コマンドを起動できないとき（`open` / `xdg-open` が無い環境）も、コマンド名と URL を含むエラーを返す。

#### Scenario: Client と Fake が OpenURL を持つ GHClient を満たす

- **WHEN** `var _ GHClient = (*Client)(nil)` と `var _ GHClient = (*Fake)(nil)` をコンパイルする
- **THEN** コンパイルが通る

#### Scenario: 実行するコマンドと引数

- **WHEN** ブラウザ起動コマンドの実行を記録するスタブに差し替えた `Client` で `OpenURL(ctx, "https://example.com/a b")` を呼ぶ
- **THEN** 記録された URL はちょうど 1 件で `https://example.com/a b` であり、`gh` を実行する関数は 1 回も呼ばれていない

#### Scenario: OS ごとのコマンド名

- **WHEN** コマンド名を `runtime.GOOS` から選ぶ関数に `darwin` と `linux` を渡す
- **THEN** 返るコマンド名は順に `open` と `xdg-open` である

#### Scenario: 終了コードが 0 でなければエラーになる

- **WHEN** 終了コード 1 と stderr `no browser` を返すスタブに差し替えた `Client` で `OpenURL(ctx, "https://example.com/a")` を呼ぶ
- **THEN** 返るエラーの文字列に `https://example.com/a`、`exit 1`、`no browser` が含まれる
