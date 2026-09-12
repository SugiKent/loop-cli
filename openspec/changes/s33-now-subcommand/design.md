## Context

動機は proposal.md の「Why」を見る。

現状の `cmd/loop-cli/main.go` は `run` が第 1 引数で振り分け、`version` / `update` は
設定ファイルも `gh` も使わない軽い経路、引数なしは `runTUI` の重い経路という 2 極になっている。
`now` は「設定と `gh` は使うが Bubble Tea は起動しない」という 3 番目の経路になる。

`runTUI` が組み立てている依存のうち、`now` に要るのは設定の読み込み・`gh.Client.Check`・`fetch.Fetch` だけである。
エディタ・通知・スナップショット・更新確認・`claude` のセッション取得は要らない。

## Goals / Non-Goals

**Goals:**

- `fetch.Fetch` の結果を「今やる」だけに絞って JSON にする層を、`cmd/loop-cli` の中に薄く作る
- 出力の形（キー名と型）を単体テストで固定し、agent から見た契約が実装の都合で動かないようにする
- 失敗経路を `gh` もネットワークも使わずにテストできる形にする

**Non-Goals:**

- 分類・並び順・局面の定義を変えること。`internal/classify` と `internal/model` と `internal/ui` は読むだけで触らない
- TUI 側に新しい表示を足すこと
- `now` から issue / PR に書き込むこと（回答・ラベル・merge は TUI の担当のまま）

## Decisions

### D1. 出力の組み立ては `cmd/loop-cli` に置き、`internal` に新しいパッケージを作らない

`now` の出力は `model.Card` から JSON 用の構造体に写すだけで、他から使う予定は無い。
実装は `cmd/loop-cli/now.go` に出力の組み立てと `run` から呼ぶ関数を置き、`cmd/loop-cli/now_test.go` に
そのテストを置けば足りる。`internal/nowjson` のようなパッケージを作ると、1 箇所からしか呼ばれない
公開 API を増やすことになる（CLAUDE.md「一度しか使う予定のない処理のために抽象化を増やさない」）。

代案として `internal/ui` の `buildRows` を公開して使い回すことも考えたが、`buildRows` は 4 タブ分の
`row`（本文・コメント・カードを抱えた画面用の型）を返すので、JSON に要らない依存を持ち込む。
`now` 側で `Card.Result.Tab == model.TabNow` に絞り、同じキーで `sort.SliceStable` する方が短い。
並び順の規則は `now-command` の Requirement に書いてあるので、両者がずれればテストで落ちる。

### D2. `subject` は `{type, number}` だけにする

主体（カードの分類結果を出した要素）は必ずカードの `Issue` かカードの `PRs` の 1 本である。
`subject` にタイトル・URL・ラベルまで載せると `issue` / `prs` と同じ値を二重に持つので、
`subject` は指し先の種別と番号だけにし、中身は `issue` / `prs` の該当要素から引かせる。

主体の判定には `internal/ui` の `Subject`（公開済み）を使う。これが返す `isPR` と `number` が
そのまま `subject` になるので、`internal/ui` に手を入れずに済み、主体の判定を書き写す必要も無い。

### D2b. PR の状態は既にある判定と表示をそのまま写す

状態の欄は新しい判定を作らず、次を呼ぶだけにする。

| 欄 | 出どころ |
| --- | --- |
| `undecided` | `model.ParseUndecided(pr.Body)`。2 つ目の戻り値が false なら `null` |
| `mergeable` / `merge_state_status` / `review_decision` | `pr.MergeState` の各欄。`MergeState` が `nil` なら `null` |
| `checks_green` | `classify.ChecksGreen(pr.MergeState)`。merge のガード（s14）と同じ判定 |
| `checks` | `StatusCheckRollup` を PR 詳細画面と同じ規則で 1 件 1 要素にする。CheckRun は `Name` と `Conclusion`（空なら `Status`）、StatusContext は `Context` と `State` |
| `unresolved_threads` | `pr.ReviewThreads` のうち `IsResolved` が false の件数。`nil` なら `null` |

`MergeState` と `ReviewThreads` の `nil` は詳細取得の失敗を表す（`internal/model` の定義）。
`checks_green` はここで `false` にせず `null` にする。`ChecksGreen` は `nil` に `false` を返すが、
それは「merge させない」という意味であって「checks が赤い」ではないので、JSON でそのまま出すと読み違える。

### D3. `errors` と `items` は必ず配列として出す

Go の `nil` スライスは `null` になる。agent 側で `null` と `[]` の分岐を書かせないため、
JSON へ写す時点で長さ 0 のスライスを割り当てる。テストで `"items":[]` と `"errors":[]` を確認する。

### D4. 時刻は RFC 3339、`fetched_at` は取得を始めた時刻

`updated_at` は GitHub が返した値をそのまま `time.Time` として出す。`fetched_at` は `time.Now()` を
1 回だけ取り、`fetch.Fetch` の `now` 引数にも同じ値を渡す（`classify` の時間切れ判定と出力の時刻をそろえる）。
経過時間（`12m` / `3h` / `2d`）は出さない。agent は `fetched_at` と `updated_at` の差を自分で計算できる。

### D5. 部分失敗は文字列の配列にする

`fetch.Result.Errors` は `error` の配列で、構造化された情報を持たない（`ViewPR org/app#131: ...` の形）。
JSON でも `err.Error()` の文字列をそのまま並べる。`gh` の失敗を型として設計し直すのは、この change の範囲を超える。

ここにはリポジトリのラベル一覧（`ListLabels`）の失敗も入る。これが落ちると運用方式を判定できず、
そのリポジトリは既定の方式（sdd）として分類される。分類が静かにずれた状態になるので、
`errors` が空でないときは `items` が不完全であり得ることを spec と README に書く。

### D6. `now` は 30 秒などの追加の時間制限を持たない

`fetch.Fetch` は 1 回の `gh` 呼び出しごとに 30 秒（`fetch.CallTimeout`）を掛け、詳細取得は 4 並行で走る。
`now` 全体の制限は掛けず、agent 側のタイムアウトに任せる。TUI と同じ取得の重さになる。

### D7. `runNow` は `runUpdate` と同じく依存を引数で受け取る

`runTUI` は設定パスも `gh.NewClient()` も関数の中でべた書きしていて、テストが 1 本も無い。
一方 `runUpdate(ctx, up updater, stdout, stderr)` はスタブを渡してテストしている（`update_test.go`）。
`now` は後者に合わせ、`runNow(ctx, deps, stdout, stderr) int` が

- 設定ファイルのパス（文字列）
- `gh` の確認（`func(context.Context) error`）
- 取得（`func(context.Context, []string, time.Time) (*fetch.Result, error)`）

を受け取る形にする。`run` の `case "now"` は本物（`config.DefaultPath()` / `gh.NewClient().Check` /
`fetch.Fetch`）を組み立てて渡すだけにする。これで「設定ファイルが無い」「`gh` が無い」「取得が失敗する」
「部分失敗がある」の 4 経路を、`gh` もネットワークも使わずにテストできる。

`gh.Client.Check` は `gh.GHClient` インタフェースに無い `*gh.Client` のメソッドなので、
インタフェースを広げず、関数値 1 つとして受け取る。

### D8. 余分な引数は黙って捨てず、エラーにする

いまの `run` は `args[1:]` を見ていないので、`loop-cli now --repo x` が `now` として素通りする。
`loop-cli-dev` は余分な位置引数も未知フラグも拒否しているので、そちらに合わせて `now` も拒否する。
`flag` パッケージは使わない（`now` はフラグを 1 つも持たないため）。

### D9. JSON は 2 スペースで整形し、末尾に改行を 1 つ付ける

`jq` はどちらでも読めるが、人が `loop-cli now` を直接打ったときに読めた方がよい。
整形しても agent 側の読み方は変わらない。

## Risks / Trade-offs

- **[人が `loop-cli now` を打つと JSON が流れる]** → JSON だけを出すことは PR #39 のコメントで決まった。
  README に「人が読む画面は `loop-cli`、agent が読むのは `loop-cli now`」と書き分け、2 スペースの整形で
  人が直接打っても読める形にする（D9）
- **[出力のキー名が後から変わると agent 側が壊れる]** → 単体テストで JSON のキーを固定する。
  変えるときは spec の Requirement を MODIFIED することになるので、気付かずに変わることはない
- **[`situation` の記号は `internal/model` の change で増減し得る]** → `situation` は局面を表す記号をそのまま
  出すので、局面が増えれば agent 側の分岐も見直しが要る。`summary` を併せて出し、未知の記号が来ても人が読める形にしておく
- **[agent が短い間隔で叩くと `gh` のレート制限に当たる]** → `now` は 1 回の実行で TUI の 1 回の取得と同じだけ
  `gh` を呼ぶ。呼ぶ間隔は agent 側の判断に任せ、CLI ではキャッシュも制限も持たない（スナップショットを
  読まないのは proposal.md の「確定した判断」4 のとおり）
- **[「今やる」が空のときに agent が失敗と読む]** → 終了コード 0 と `count: 0` を Requirement とテストで固定する
- **[fixture だけでは並び順の tiebreak を作れない]** → `board` fixture は 1 リポジトリ 2 枚のカードなので、
  優先度の違いは確かめられるが、更新時刻が同じときのリポジトリ名・番号の順は確かめられない。
  その分は手で組み立てた `fetch.Result` でテストする
