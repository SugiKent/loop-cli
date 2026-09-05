## Context

s01-bootstrap で Go モジュール（`github.com/SugiKent/sugi-loop`）、s02-config で `internal/config` ができている。`internal/gh` はまだ無い。
この change は D-003 の内部構成にある `internal/gh/`（`GHClient` interface と `gh` サブプロセス実装、JSON fixture の fake）を作る。
正本は D-001 の呼び出し一覧と、human-turn-signals.md「TUI のアクション」列、mvp.md キーバインド表「内部処理」列である。

手元の `gh` 2.93.0 で確認した事実（読み取りのみ。公開リポジトリ `cli/cli` に対して実行）:
- `gh search issues` / `gh search prs` の `--json` フィールドに D-001 の指定（`repository,number,title,labels,updatedAt,url,body,commentsCount` / `…,isDraft`）はすべて存在する。`repository` は `{name, nameWithOwner}`、`labels[]` は `{id, name, color, description}`
- `gh issue view --json comments` / `gh pr view --json comments` の要素は `{id, author{login}, authorAssociation, body, createdAt, includesCreatedEdit, isMinimized, minimizedReason, reactionGroups, url, viewerDidAuthor}`
- `gh pr view --json statusCheckRollup` の要素は `__typename` を持ち、`CheckRun` は `name/status/conclusion/workflowName/detailsUrl/startedAt/completedAt`、`StatusContext` は `context/state/targetUrl`。`mergeable` は `MERGEABLE` / `CONFLICTING` / `UNKNOWN`、`mergeStateStatus` は `CLEAN` 等、`reviewDecision` は `APPROVED` 等の文字列
- GraphQL の `reviewThreads` 応答は `data.repository.pullRequest.reviewThreads.nodes[]`。`databaseId` は REST の review comment `id` と同じ整数（REST replies エンドポイントに渡せる）
- GraphQL の `timelineItems(itemTypes:[CROSS_REFERENCED_EVENT])` で、参照元が PR でないノードは `source` が `{}` になる
- `gh api … --paginate --jq '…'` はページごとに jq の結果を連結して出す。D-001 の jq はオブジェクトを 1 件ずつ出すので、標準出力は JSON オブジェクトの連続になる
- 存在しない issue は終了コード 1、stderr に `GraphQL: Could not resolve to an issue or pull request with the number of 99999999. (repository.issue)`
- `gh auth status` は失敗時に理由を stderr に出して終了コード 1。`GH_TOKEN` に無効な値を入れると再現できる
- `gh` が無い環境では `exec.LookPath("gh")` が失敗する
- `gh issue comment` / `gh pr comment` / `gh issue create` は `--body-file -` で標準入力から本文を読む。`gh api` は `--input -` で標準入力からリクエストボディを読む

## Goals / Non-Goals

**Goals:**
- `GHClient` interface（読み取り 8 + 書き込み 8）と生の型を確定し、s04〜s16 が同じ型を使えるようにする
- `Client` が発行する `gh` の引数を 1 つずつ確定し、テストで検証できる形にする
- 失敗（`gh` 無し / 未認証 / 非 0 終了 / デコード失敗 / タイムアウト）を区別できるエラーにする。テストはネットワークに出ない
- fixture の命名規則と `Fake` を確定し、s04 が採取し s05 が消費できるようにする

**Non-Goals:**
- 稼働リポジトリ由来の fixture の採取（s04）。この change では fake のテスト用に手書きの最小 fixture だけ置く
- `internal/model` と分類（s05）、Issue と PR の紐づけ（s07）
- `cmd/sugi-loop` からの配線と `Check` の呼び出し（s07）
- レートリミット取得（s18）と GraphQL 1 リクエストへの統合（s19）と GitHub Notifications（s20）のメソッドは、この change では定義しない。必要になった change が ADDED でメソッドを足す
- 書き込みの dry-run は作らない。validation-plan.md が検証手段を未定としているので、この change では発明しない
- リトライ・バックオフ（`mergeable: UNKNOWN` の 1 回再取得を除く）

## Decisions

### ファイル構成

```
internal/gh/gh.go            # GHClient interface、Error 型
internal/gh/types.go         # 生の型（SearchIssue / SearchPR / IssueDetail / PRDetail / PRMergeState / … / LabelEvent）
internal/gh/decode.go        # バイト列 → 型 のデコード関数（Client と Fake が共用）
internal/gh/client.go        # Client（gh サブプロセス）と Check
internal/gh/fake.go          # Fake（fixture + Calls）
internal/gh/client_test.go   # 実行関数を差し替えて引数・エラー・UNKNOWN 再取得を検証
internal/gh/decode_test.go   # 各デコード関数を JSON 文字列で検証
internal/gh/fake_test.go     # fixtures/example を読んで Fake を検証
internal/gh/testdata/fixtures/example/   # 手書きの最小 fixture（下記）
```

### interface を最初から全部定義する

s10〜s16 が使う書き込み 8 メソッドもこの change で定義する。代替案は「各 change が必要なメソッドを足す」だが、interface に 1 メソッド足すたびに `Client` と `Fake` の両方を触ることになり、change ごとの差分が 3 ファイルに散る。書き込みの集合は docs（human-turn-signals.md の「TUI のアクション」列と mvp.md のキーバインド表）で確定しており、投機ではない。
docs に無いメソッド（レートリミット等）は含めない。後続 change が足す場合は、その change の `gh-client` capability に ADDED で Requirement を書き、interface・`Client`・`Fake` を同時に更新する。

### 生の型は gh の出力を写し、GraphQL だけ平坦化する

`--json` 系（search / issue view / pr view）は `gh` の出力キーをそのまま struct タグに写す。使わないキー（`labels[].color`、`comments[].reactionGroups` 等）は struct に持たず読み捨てる。
GraphQL の応答は `data.repository.pullRequest.reviewThreads.nodes[].comments.nodes[]` のように深く、消費側（s05 / s16）がこの入れ子をたどるのは冗長なので、`decode.go` の中で `[]ReviewThread` / `[]CrossReferencedPR` に平坦化する。入れ子を受ける struct は `decode.go` 内の非公開型に閉じ込める。
fixture には平坦化前の生の応答を保存する。fixture が `Client` の標準出力そのものであれば、s04 の採取が「標準出力をファイルに書く」だけで済み、`Fake` と `Client` が同じデコード関数を通ることで fixture がデコード関数のテストにもなる。例外は `pr-<n>.json` で、`ViewPR` と `ViewPRMergeState` の合成フィールド列（未決事項の表を参照）で採る。

`StatusCheck` は `CheckRun` と `StatusContext` を 1 つの struct で受ける（`__typename` で判別）。判別用の型を 2 つ作って interface にする案は、消費側（s14 の merge ガード）が「1 つでも失敗があるか」を見るだけなので過剰。

### Client の実行関数を差し替え可能にする

```go
type Client struct {
    run       func(ctx context.Context, stdin string, args ...string) (stdout []byte, err error)
    RetryWait time.Duration // mergeable UNKNOWN の再取得までの待ち。既定 2s
}
func NewClient() *Client
```

`run` は非公開フィールドで、`NewClient` が `os/exec` 実装を入れる。テストは同じパッケージ内で `run` を差し替え、受け取った `args` と `stdin` を記録し、任意の `stdout` / `*Error` を返す。`gh` を起動しないので CI で動く。`exec.Cmd` そのものをモックする案は API が広くテストが読みにくいので採らない。

`os/exec` 実装:
- `exec.CommandContext(ctx, "gh", args...)`
- `cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1")`
- `stdin` が空でなければ `cmd.Stdin = strings.NewReader(stdin)`。空なら nil のまま（`/dev/null` 相当）。TUI が端末を握っているので `gh` に端末を渡さない
- `stdout` と `stderr` を別々のバッファに取る
- `cmd.Run()` が `*exec.ExitError` なら `&Error{Args: args, ExitCode: code, Stderr: stderr}`。`ctx.Err() != nil` なら `fmt.Errorf("gh %s: %w", strings.Join(args, " "), ctx.Err())`。それ以外（起動失敗）はそのまま `fmt.Errorf("gh %s: %w", …, err)`

### エラー型

```go
type Error struct {
    Args     []string
    ExitCode int
    Stderr   string
}
func (e *Error) Error() string // "gh <args>: exit <code>: <stderr を TrimSpace したもの>"
```

デコード失敗は `fmt.Errorf("gh %s: decode: %w", strings.Join(args, " "), err)` で返す。`*Error` にしないのは、`gh` が正常終了したことを呼び出し側が区別できるようにするため（s07 が「前回結果を維持してエラーを赤で表示」する際、どちらも表示はするが、原因の切り分けに使う）。
エラー文言の翻訳・整形はしない。`gh` の stderr をそのまま見せるのが利用者にとって最も情報が多い。

### Check

```go
func (c *Client) Check(ctx context.Context) error
```

1. `exec.LookPath("gh")` が失敗したら `fmt.Errorf("gh が見つかりません。GitHub CLI をインストールして gh auth login を実行してください: %w", err)`
2. `c.run(ctx, "", "auth", "status")` を実行し、非 0 なら `*Error`（stderr に理由が入る）。`run` を通すので、テストは `gh` を起動せず github.com にも接続しない（CI runner には `gh` が入っているため、live で `gh auth status` を呼ぶテストは CI で毎回ネットワークに出てしまう）

`Client` の各メソッドは `Check` を呼ばない。起動時に 1 回で足りる（`gh` が途中で消えることは想定しない）。

### ViewPR と ViewPRMergeState を分け、UNKNOWN 再取得は ViewPRMergeState だけが持つ

D-001 の表は `question` PR の取得（`--json comments`）と merge 候補 PR の取得（`--json mergeable,mergeStateStatus,statusCheckRollup,reviewDecision`）を別の行にしており、2 秒後の再取得は merge ガード取得に限定している。1 メソッドで両方を取って再取得まで行うと、`question` PR のコメント取得も新しい PR では 2 秒遅れる。そのため `ViewPR`（コメント・ラベル。再取得なし）と `ViewPRMergeState`（merge ガード用 4 フィールド。UNKNOWN 再取得あり）に分け、型も `PRDetail` と `PRMergeState` に分ける。`PRDetail` に `Mergeable` を持たせないので、merge ガードが再取得なしの値を誤って使うことがない。代替案の `ViewPR(ctx, repo, n, withMergeState bool)` は、bool 引数で返る型と待ち時間が変わる分かりにくい API になるので採らない。
再取得を `Client.ViewPRMergeState` の中で行うのは、取得層（s07）に置く案と比べ、`GHClient` の呼び出し側が `mergeable` の遅延計算という GitHub の都合を知らずに済むためである。`Fake` は再取得しない（fixture は決定的で、待つ意味が無い）。待ち時間は `select { case <-time.After(c.RetryWait): case <-ctx.Done(): return nil, ctx.Err() }` で待ち、キャンセルに応答する（`time.Sleep` はキャンセルできないので使わない）。

### 本文は標準入力で渡す

mvp.md は `--body-file` を挙げている。一時ファイルを作って渡す案と、`--body-file -` で標準入力から渡す案のうち後者を採る。一時ファイルの作成・削除・失敗時の後始末が不要で、`run` の `stdin` 引数だけで済む。`ReplyReviewThread` も同様に `gh api --input -` へ `{"body": …}` を `encoding/json` で作って渡す（`-f body=…` で引数に載せる案は本文が長い場合と改行の扱いが引数依存になる）。

### GraphQL は操作宣言付きのクエリを -f / -F で渡す

`ReviewThreads` / `CrossReferencedPRs` のクエリは `query($owner:String!,$name:String!,$number:Int!){ … }` の操作宣言付きの完全な文字列を `-f query=<Q>` で渡す。変数は `gh api graphql` の規約どおり、文字列の `owner` / `name` は `-f`（型付けしない）、整数の `number` は `-F`（自動型付け）で渡す。`owner` / `name` を `-F` にすると数字だけの名前が整数に型付けされ、`String!` の変数と合わなくなる。クエリ文字列は `Client` のパッケージ定数に置き、テストが引数列全体と一致比較する。

### Fake

```go
type Fake struct {
    Dir   string
    Calls []Call
}
type Call struct {
    Method      string
    Repo        string
    Number      int
    Body        string
    Label       string
    MergeMethod string
    Title       string
    CommentID   int64
}
func NewFake(dir string) *Fake
```

- 読み取り: `filepath.Join(f.Dir, name)` を `os.ReadFile` し、`decode.go` の関数に渡す。ファイルが無ければ `os.ReadFile` のエラー（パスを含む）をそのまま返す。`ViewPR` と `ViewPRMergeState` はどちらも `pr-<n>.json` を読み、それぞれ `decodePRDetail` / `decodePRMergeState` に渡す（未知のキーは読み捨てるので 1 ファイルで足りる。fixture を 2 ファイルに分けない）
- `repo` 引数はファイル探索に使わない。fixture ディレクトリ 1 つがリポジトリ 1 件に対応するためである。複数リポジトリのテストが必要になったら、そのときに `<repo-alias>` とリポジトリ名の対応を足す
- 書き込み: `f.Calls = append(f.Calls, Call{…})` して nil を返す。`CreateIssue` は `https://github.com/<repo>/issues/0`
- `ViewIssue` は記録してから読む（読み込み失敗でも `Calls` には残る）
- 読み取りのうち `ViewIssue` だけは `Calls` に記録する（`Method: "ViewIssue"`、`Repo`、`Number`）。human-turn-signals.md 不変条件 3「`stage:todo` を外し、`gh issue view --json labels` で読み直してから `stage:propose` を付ける」の順序を s15 のテストが `Calls` で検証するためである。他の読み取りは記録しない。テストが検証したいのは書き込みの引数と不変条件 3 の順序であり、全読み取りを記録すると `Calls` の検証が読み取り回数に依存して壊れやすくなる
- `Fake` は書き込みで失敗を返す手段を持たない。書き込み失敗時の UI の振る舞い（s10〜s16）が必要になったら、その change が `Fake` にエラー注入を足す

### 手書きの最小 fixture

`internal/gh/testdata/fixtures/example/` に、fake のテストと s04 の採取結果の見本を兼ねる最小 fixture を置く。内容は Context に書いた実出力の形を写し、値は架空（リポジトリ `org/app`、issue 108、PR 131）にする。
- `search-issues.json`: issue 2 件（108 に `stage:propose` と `question`、140 はラベル無し）
- `search-prs.json`: PR 1 件（131、`propose` と `question`、`isDraft: false`）
- `issue-108.json`: コメント 2 件（1 件目の body が `<!-- routine -->` で始まる）
- `pr-131.json`: `ViewPR` と `ViewPRMergeState` の `--json` フィールドを合わせた 1 回の `gh pr view` 出力。`mergeable: "UNKNOWN"`、`statusCheckRollup` に `CheckRun` と `StatusContext` を 1 件ずつ
- `pr-131-review-threads.json`: thread 1 件、コメント 1 件
- `issue-108-cross-refs.json`: PR ノード 1 件と `source: {}` のノード 1 件
- `issue-108-timeline.json`: `labeled` と `unlabeled` を 1 件ずつ

s04 が稼働リポジトリから採取した fixture は別の `<repo-alias>` ディレクトリに置き、`example` は上書きしない。

## Risks / Trade-offs

- [`gh` のバージョン差で `--json` のフィールド名や GraphQL の形が変わる] → 型は使うフィールドだけを持ち、未知のキーは読み捨てるので追加には耐える。削除・改名は fixture のデコードテストで検知する
- [`--limit 200` を超えるリポジトリ横断の open issue] → D-001 の値をそのまま採る。超えた分は取れない。s07 が件数を見て警告するかは s07 の判断
- [`gh api --paginate --jq` の出力形式（オブジェクトの連続）が変わる] → `json.Decoder` で EOF まで繰り返し読む実装は、配列 1 個でも壊れないよう「先頭が `[` なら配列としてデコード」を入れる代わりに、D-001 の jq に固定して形を 1 つに保つ。分岐は入れない
- [`GH_PROMPT_DISABLED` の環境変数名が `gh` の版で違う] → 標準入力が端末でないので `gh` はプロンプトを出さない。環境変数は二重の保険であり、名前が違っても動作は変わらない
- [`Check` が `gh auth status` の終了コードだけを見る] → 複数アカウントのうち 1 つが失敗しても非 0 になる（手元で確認: `GH_TOKEN` 無効 + keyring 有効で終了コード 1）。この場合 stderr に両方の状態が出るので、利用者が読んで対処できる
- [`Fake` の `repo` 無視] → 2 リポジトリ横断のテストは書けない。s05 の分類テストは 1 リポジトリの fixture で足りる（V-1）

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| サブプロセスのタイムアウト | `Client` は持たない。呼び出し側が `context.WithTimeout` で与える。s07 が値（既定 30 秒）を決める | docs に無い。`Client` に固定値を持たせると TUI のキャンセル（`R` 連打・終了）と二重管理になる。`Client` は `ctx` に従うだけにする |
| interface を全部定義するか | 全部（読み取り 8 + 書き込み 8）。後続は ADDED で足す | 上記 Decisions |
| `ViewPR` と merge ガード取得の分離 | `ViewPR` / `ViewPRMergeState` の 2 メソッド。再取得は後者だけ | 上記 Decisions。D-001 が再取得を merge ガード取得に限定している |
| 書き込みの本文の渡し方 | `--body-file -` / `--input -` で標準入力 | 上記 Decisions |
| `MergePR` の `method` の型 | `string`（`squash` / `merge` / `rebase`）。`internal/config` の `MergeMethod` を import しない | `internal/gh` を `internal/config` に依存させない。値の検証は s02 が済ませている。s14 は `string(repo.MergeMethod)` で渡す |
| `CreateIssue` の返り値 | 作成した issue の URL（標準出力の末尾行）。終了コード 0 でも末尾行が `https://` で始まらなければデコードエラー（`*Error` ではない）にする | `gh issue create` は URL を標準出力に出す。s15 が「作成した issue を開く / 表示する」のに必要。URL でない値を返して s15 が開こうとするのを防ぐ |
| `ReplyReviewThread` の識別子 | REST の comment `id`（GraphQL `databaseId`）を `int64` で受ける | mvp.md の `A` の内部処理（`…/comments/{id}/replies`）に合わせる。`ReviewComment.DatabaseID` をそのまま渡せる |
| `Browse` の実装 | `gh browse <n> -R <repo>` | mvp.md は `gh browse` / `open <url>` の 2 案。`gh browse` は OS 差を `gh` に任せられる |
| fixture ファイル名 | `search-issues.json` / `search-prs.json` / `issue-<n>.json` / `pr-<n>.json` / `pr-<n>-review-threads.json` / `issue-<n>-cross-refs.json` / `issue-<n>-timeline.json` | コマンドと番号が名前から読める。s04 が採取時にこの名前で書く |
| fixture の内容 | `Client` の標準出力そのまま（GraphQL は `data` から始まる生の応答、timeline は jq 出力の連続）。`pr-<n>.json` だけは `ViewPR` と `ViewPRMergeState` の合成フィールド列で採る: `gh pr view <n> -R <owner/name> --json number,title,body,url,labels,isDraft,comments,mergeable,mergeStateStatus,statusCheckRollup,reviewDecision` | `pr-<n>.json` 以外は s04 の採取が「標準出力を保存」で済み、`Fake` と `Client` が同じデコードを通る。`pr-<n>.json` は 2 メソッドが 1 ファイルを共有するため、`Client` の実コマンドとは別の合成コマンドで採る |
| `Fake` の `repo` 引数 | 無視する | fixture ディレクトリ 1 つがリポジトリ 1 件に対応する。V-1 は 1 リポジトリで足りる |
| `Fake.CreateIssue` の返り値 | `https://github.com/<repo>/issues/0` | 固定値で十分。番号 0 は実在しないので実データと混同しない |
| `Fake` のエラー注入 | 持たない | docs に無い。必要になった change が足す |
| `gh` に付ける環境変数 | `GH_PROMPT_DISABLED=1` と `GH_NO_UPDATE_NOTIFIER=1` を追加 | TUI が端末を握っている間に `gh` がプロンプトや更新通知を出すと画面が壊れる |
| `LabelTimeline` の取得方法 | D-001 の `--paginate --jq` をそのまま使い、jq 出力の連続を `json.Decoder` で読む | D-001 に書かれた形をそのまま採る。jq を外して全イベントを Go 側で絞る案は、fixture が数十倍大きくなる |
| `ReviewThreads` / `CrossReferencedPRs` のページング | `first:50` / `first:100` / `labels(first:20)` の 1 ページのみ | D-001 の `reviewThreads(first:50)` に揃える。それ以上は取らない |
| エラー文言の言語 | `Check` の「gh が見つかりません」だけ日本語。`gh` の stderr はそのまま | 利用者向けの案内は日本語、`gh` 由来はそのまま |
