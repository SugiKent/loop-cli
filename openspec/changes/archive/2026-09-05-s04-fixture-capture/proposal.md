## Why

V-1（validation-plan.md）は「稼働中リポジトリから取った JSON を個人・組織情報を伏せて fixture にし、分類器を live データなしでテストする」ことを求めるが、
s03-gh-client が置くのは手書きの最小 fixture（`example`）だけで、稼働リポジトリ由来の fixture はまだ無い。s05-classify はこの fixture で局面 A〜G と「その他」バケットをテストする。
そのため s05 の前に、稼働中リポジトリ 1 件（V-2 で利用者が指定する同じリポジトリ）から `gh` の生の出力を採取し、伏せ字にして s03 の命名規則で保存するコマンドを作る。
docs/mvp/implementation-tasks.md §2「fixture 採取」に対応し、同 §2「動作確認用 CLI」のうち `cmd/sugi-loop-cli` の骨組みと `fixture capture` サブコマンドをこの change で導入する（`classify` / `notify test` は s06）。

## What Changes

- `cmd/sugi-loop-cli` を新設する。サブコマンド方式（`help` / `fixture capture`）で、標準ライブラリの `flag` と `os.Args` だけで書く。CLI フレームワークは入れない
- `fixture capture --repo owner/name --alias <alias>` を追加する。s03 の `Client`（実 `gh`）で open issue / open PR の検索と、分類に必要な詳細（issue の comments・cross-reference・timeline、PR の comments と merge ガード用の状態・review threads）を **全 issue / 全 PR** について取り、伏せ字にして `internal/gh/testdata/fixtures/<alias>/` に s03 の命名で保存する
- `internal/gh` の `Client` に、読み取りコマンドの標準出力を fixture ファイル名ごとに返す `Capture` を足す（`gh-client` capability に ADDED）。fixture は「`Client` の標準出力そのまま」（s03 design）なので、型に写した後に再エンコードせず、生のバイト列を保存する
- 伏せ字（redaction）の規則を確定する: `owner/name` → `org/<alias>`、ユーザー login → `user-N`（同一人物は同一番号、owner は `user-1`）、本文中の `@mention` → `@user-N`、メールアドレス → `user@example.com`、リポジトリ名単独 → `<alias>`。ラベル名・`<!-- routine -->` マーカー・`## Q1.`・`未確定の判断: N 件`・`blocked-by:` 行など分類に使う文字列は変えない
- 採取した fixture が個人情報を含まないことを検査するテストを `internal/gh` に足す（login が `user-N` 形式であること、メールアドレスが無いこと、環境変数で元の `owner/name` を渡したときにそれが含まれないこと）
- 実際の採取は利用者がリポジトリを指定して実行し、結果を目視確認してコミットするところまでを tasks に含める

## Capabilities

### New Capabilities
- `dev-cli`: `cmd/sugi-loop-cli` の骨組み。サブコマンドの振り分け、`help` の出力、未知のサブコマンドと失敗時の終了コード。s06 が `classify` / `notify test` を ADDED で足す
- `fixture-capture`: `fixture capture` サブコマンドの引数・採取範囲・保存先、伏せ字の規則、書き込み後の自己検査（`NewFake` で読み直し、失敗なら保存先を削除）、採取した fixture が個人情報を含まないことの検査

### Modified Capabilities
- `gh-client`: ADDED のみ。`Client` に fixture 採取用の `Capture` を足す（読み取りメソッドと同じ引数・同じ `mergeable: UNKNOWN` 再取得を通り、標準出力を生のまま返す）。既存の Requirement は変えない

## Impact

- 新規: `cmd/sugi-loop-cli/main.go` / `main_test.go` / `fixture.go` / `redact.go` / `redact_test.go`、`internal/gh/capture.go` / `capture_test.go` / `fixtures_test.go`
- 変更: `internal/gh/client.go`（読み取りメソッドの引数組み立てを非公開関数に切り出し、`pr view` の生取得（UNKNOWN 再取得付き）を `prViewRaw` に切り出す。s03 の `viewPRMergeStateOnce` は `prViewRaw` + `decodePRMergeState` に置き換える。`Capture` と共有する）、`internal/gh/fake.go`（fixture ファイル名の組み立てを `Capture` と共有する）、`.gitignore`（`sugi-loop-cli` のビルド成果物）
- 新規データ: `internal/gh/testdata/fixtures/<alias>/`（利用者が採取してコミットする。`example` は上書きしない）
- 外部依存は追加しない。実行時には `gh` CLI と `gh auth login` 済みの認証が必要（s03 の `Check` で確認する）
- 検証計画との対応: V-1 の「fixture を用意する」部分をこの change が満たす。採取は `Client.Check` と読み取り 7 種を稼働リポジトリに対して初めて live で実行するので、V-2 の読み取り側の動作確認を兼ねる。書き込みの検証手段（validation-plan.md「未定」）はこの change では扱わない
- 後続 change への影響: s05 は `<alias>` の fixture を `Fake` で読んで分類器をテストする。s06 は `dev-cli` に `classify` / `notify test` を足す
