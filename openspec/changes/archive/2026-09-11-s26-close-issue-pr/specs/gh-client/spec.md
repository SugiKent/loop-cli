## ADDED Requirements

### Requirement: CloseIssue と ClosePR は issue と PR を close する

`internal/gh` の `GHClient` interface は `CloseIssue(ctx context.Context, repo string, number int) error` と
`ClosePR(ctx context.Context, repo string, number int) error` を MUST 持つ（s03「`GHClient` interface が読み取りと
書き込みのメソッドを定義する」が定めた「後続 change が必要とするメソッドは、その change が ADDED で Requirement を足して
interface・`Client`・`Fake` を同時に拡張する」に従う）。`c`（s26 `close-issue-pr`）がこの 2 つを使う。

`Client` の実装は `gh` を次の引数で MUST 実行し、標準出力は読み捨てる（s03「`Client` は gh サブプロセスを正確な引数で実行する」の形）。

- `CloseIssue`: `issue close <number> -R <repo>`
- `ClosePR`: `pr close <number> -R <repo>`

終了コードが 0 でなければ、s03「`gh` の失敗はコマンドと stderr を含むエラーになる」と同じエラーを MUST 返す。
close の理由（`gh issue close --reason`）とコメント（`--comment`）は渡さない。ブランチの削除（`gh pr close --delete-branch`）も行わない。

#### Scenario: Client と Fake が CloseIssue と ClosePR を持つ GHClient を満たす
- **WHEN** `var _ GHClient = (*Client)(nil)` と `var _ GHClient = (*Fake)(nil)` をコンパイルする
- **THEN** コンパイルが通る

#### Scenario: CloseIssue が実行する引数
- **WHEN** `gh` の実行を記録するスタブに差し替えた `Client` で `CloseIssue(ctx, "org/app", 108)` を呼ぶ
- **THEN** 実行関数は引数 `issue close 108 -R org/app` を 1 回受け取り、標準入力には何も渡らない

#### Scenario: ClosePR が実行する引数
- **WHEN** 同じスタブで `ClosePR(ctx, "org/app", 131)` を呼ぶ
- **THEN** 実行関数は引数 `pr close 131 -R org/app` を 1 回受け取る

#### Scenario: close の失敗はコマンドと stderr を含むエラーになる
- **WHEN** 終了コード 1 と標準エラー `could not close issue` を返すスタブで `CloseIssue(ctx, "org/app", 108)` を呼ぶ
- **THEN** 返るエラーの文字列に `issue close 108 -R org/app`、`exit 1`、`could not close issue` が含まれる
