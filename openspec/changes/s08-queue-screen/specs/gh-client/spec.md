## MODIFIED Requirements

### Requirement: 起動前に gh の存在と認証を確認する
`Client` はメソッド `Check(ctx) error` を MUST 提供する。`gh` が PATH に無ければ、`gh` のインストールを促す文言を含むエラーを返す。次に `gh auth status` を実行し、非 0 で終了すれば、その stderr を含む `*Error` を返す。どちらも通れば nil を返す。`gh auth status` の実行は他のメソッドと同じ実行関数を通り、テストで差し替えられる。`Check` を呼ぶのは起動時の `cmd/sugi-loop`（s08-queue-screen が配線する。s07 は配線しなかった）であり、`Client` の各メソッドは毎回 `Check` を呼ばない。

#### Scenario: gh が PATH に無い
- **WHEN** `PATH` に `gh` が存在しない状態で `Check` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `gh` を含む

#### Scenario: 認証されていない
- **WHEN** `gh` が PATH にあり、実行関数が引数 `auth status` に対して終了コード 1 と stderr `X Failed to log in to github.com using token (GH_TOKEN)` を返す状態で `Check` を呼ぶ
- **THEN** `*Error` が返り、`Stderr` にその文言を含む

#### Scenario: gh が使える
- **WHEN** `gh` が PATH にあり、実行関数が引数 `auth status` に対して終了コード 0 を返す状態で `Check` を呼ぶ
- **THEN** 実行関数は引数 `auth status` を 1 回受け取り、`Check` は nil を返す
