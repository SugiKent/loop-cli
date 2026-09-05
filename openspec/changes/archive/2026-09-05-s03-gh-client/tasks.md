## 1. interface と型

- [x] 1.1 `internal/gh/gh.go` を作成し、`GHClient` interface（読み取り 8: `SearchIssues` / `SearchPRs` / `ViewIssue` / `ViewPR` / `ViewPRMergeState` / `ReviewThreads` / `CrossReferencedPRs` / `LabelTimeline`、書き込み 8: `CommentIssue` / `CommentPR` / `AddLabel` / `RemoveLabel` / `MergePR` / `CreateIssue` / `ReplyReviewThread` / `Browse`）と、`Error` 型（`Args` / `ExitCode` / `Stderr`、`Error()` は `gh <args>: exit <code>: <stderr>`）を定義する。`Check` は interface に含めない（`Client` 固有のメソッド）
- [x] 1.2 `internal/gh/types.go` を作成し、spec「生の型は gh の JSON 出力を写す」のとおりに `Label` / `Repository` / `Author` / `SearchIssue` / `SearchPR` / `Comment` / `IssueDetail` / `PRDetail` / `PRMergeState` / `StatusCheck` / `ReviewThread` / `ReviewComment` / `CrossReferencedPR` / `LabelEvent` を JSON タグ付きで定義する（`StatusCheck.Typename` のタグは `__typename`、`LabelEvent.CreatedAt` のタグは `created_at`）

## 2. デコード

- [x] 2.1 `internal/gh/decode.go` を作成し、`decodeSearchIssues` / `decodeSearchPRs` / `decodeIssueDetail` / `decodePRDetail` / `decodePRMergeState` / `decodeReviewThreads` / `decodeCrossReferencedPRs` / `decodeLabelEvents` を実装する。GraphQL の 2 つは非公開の入れ子 struct で受けて平坦化し、`source` が空のノードは捨てる。`decodeLabelEvents` は `json.Decoder` で EOF までオブジェクトを繰り返し読む
- [x] 2.2 `internal/gh/decode_test.go` を作成し、design.md の Context に書いた実出力の形を写した JSON 文字列で各関数を検証する。`statusCheckRollup` に `CheckRun` と `StatusContext` が混在するケース、reviewThreads の平坦化、cross-reference の `source: {}` 除外、timeline のオブジェクト連続、壊れた JSON でエラーになることを含める

## 3. Client

- [x] 3.1 `internal/gh/client.go` を作成し、`Client`（非公開の `run` 関数と `RetryWait`）、`NewClient()`、`os/exec` による `run` の実装（`GH_PROMPT_DISABLED=1` / `GH_NO_UPDATE_NOTIFIER=1` を追加、stdin は本文があるときだけ渡す、stdout / stderr を分離、非 0 終了は `*Error`、`ctx` 由来の失敗は `ctx.Err()` を包む）を実装する
- [x] 3.2 同ファイルに読み取り 8 メソッドを実装する。引数は spec「Client は gh サブプロセスを正確な引数で実行する」のとおり。GraphQL の 2 クエリは操作宣言付きの完全な文字列をパッケージ定数に置き、`-f owner= -f name= -F number= -f query=` で渡す。`ViewPRMergeState` は 1 回目の `Mergeable` が `UNKNOWN` なら `RetryWait`（既定 2 秒）待って 1 回だけ再実行し、待ちは `ctx.Done()` で中断する。`ViewPR` は再取得しない。デコード失敗は `gh <args>: decode: <err>` で返す
- [x] 3.3 同ファイルに書き込み 8 メソッドを実装する。`CommentIssue` / `CommentPR` / `CreateIssue` は `--body-file -` と stdin、`ReplyReviewThread` は `--input -` に `{"body": …}` を `encoding/json` で作って渡す。`AddLabel` / `RemoveLabel` はラベル 1 つだけ。`MergePR` は `--<method>`。`CreateIssue` は stdout の末尾行を返し、末尾行が `https://` で始まらなければデコードエラー（`*Error` ではない）を返す
- [x] 3.4 同ファイルに `(*Client).Check(ctx)` を実装する。`exec.LookPath("gh")` 失敗 → インストールを促す日本語のエラー、`c.run` 経由の `auth status` が非 0 → `*Error`
- [x] 3.5 `internal/gh/client_test.go` を作成し、`run` を差し替えて検証する。`SearchIssues` の `--repo` 列（2 リポジトリ）/ `SearchPRs` / `ViewIssue` / `ViewPR` / `ViewPRMergeState` の `--json` フィールド列 / `ReviewThreads` と `CrossReferencedPRs` の引数列全体（`-f owner=org -f name=app -F number=<n> -f query=<Q>`。`Q` は spec のクエリ文字列と完全一致）/ `AddLabel` と `RemoveLabel` がラベル 1 つ / `CommentPR` と `CommentIssue` の stdin / `MergePR` の `--squash` / `CreateIssue` の引数と URL の返り値 / `CreateIssue` の末尾行が URL でないときデコードエラー（`*Error` にならない）/ `ReplyReviewThread` の URL と stdin の JSON / `Browse` / `LabelTimeline` のデコード / `ViewPRMergeState` の UNKNOWN 再取得 3 ケース（`RetryWait` を 0 にする）/ `ViewPR` が UNKNOWN でも 1 回しか実行しない / 非 0 終了が `*Error` で `Error()` に引数と stderr を含む / 壊れた JSON が `*Error` にならない / `ctx` タイムアウトで `errors.Is(err, context.DeadlineExceeded)`
- [x] 3.6 同ファイルに `Check` のテストを追加する。`t.Setenv("PATH", t.TempDir())` で `gh` が見つからずエラー文字列に `gh` を含むこと。`gh` の LookPath を通すため `t.TempDir()` に実行可能な空の `gh` ファイルを置いて `PATH` に加え、`run` を差し替えて `auth status` が終了コード 1 と stderr を返すと `*Error` が返り `Stderr` にその文言を含むこと、終了コード 0 なら nil が返り `run` が引数 `auth status` を 1 回受け取ること。テストは実際の `gh` を起動せず、ネットワークに出ない

## 4. Fake と fixture

- [x] 4.1 `internal/gh/testdata/fixtures/example/` に design.md「手書きの最小 fixture」の 7 ファイル（`search-issues.json` / `search-prs.json` / `issue-108.json` / `pr-131.json` / `pr-131-review-threads.json` / `issue-108-cross-refs.json` / `issue-108-timeline.json`）を、design.md の Context に書いた実出力の形で、値は架空（`org/app`）にして作る
- [x] 4.2 `internal/gh/fake.go` を作成し、`Fake`（`Dir` / `Calls`）、`Call`、`NewFake(dir)` を実装する。読み取りは命名規則でファイルを読み `decode.go` の関数に渡し（`ViewPR` / `ViewPRMergeState` は同じ `pr-<n>.json`）、無ければ `os.ReadFile` のエラーをそのまま返す。`ViewIssue` だけは読み取りでも `Calls` に `Method` / `Repo` / `Number` を記録する。書き込みは `Calls` に追記して nil を返し、`CreateIssue` は `https://github.com/<repo>/issues/0` を返す。`var _ GHClient = (*Client)(nil)` と `var _ GHClient = (*Fake)(nil)` を置く
- [x] 4.3 `internal/gh/fake_test.go` を作成し、`NewFake("testdata/fixtures/example")` で検証する。`SearchIssues` が 2 件で `Number` / `Labels` / `Repository.NameWithOwner` が一致 / `SearchPRs` 1 件 / `ViewIssue(108)` のコメント 2 件 / `ViewPR(131)` の `IsDraft` / `Comments` / `ViewPRMergeState(131)` が `UNKNOWN` のまま 1 回で返り `StatusCheckRollup` が一致 / `ReviewThreads(131)` と `CrossReferencedPRs(108)` と `LabelTimeline(108)` の件数 / `ReviewThreads(131)` の結果が、`pr-131-review-threads.json` の同じバイト列を `run` 差し替えの `Client` に返させた `Client.ReviewThreads` の結果と `reflect.DeepEqual` で一致 / `ViewIssue(999)` のエラーに `issue-999.json` を含む / `AddLabel` の記録 / `RemoveLabel` → `ViewIssue` → `AddLabel` の順序と `Label` / `ViewIssue` の記録に `Repo` と `Number` / `CommentPR` の `Body` / `SearchIssues` / `ViewPR` / `ViewPRMergeState` の後は `Calls` が空

## 5. 最終確認

- [x] 5.1 `gofmt -l .` が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
