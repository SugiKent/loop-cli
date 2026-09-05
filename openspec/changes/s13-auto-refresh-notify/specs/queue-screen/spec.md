## MODIFIED Requirements

### Requirement: Model は Card をタブ別に並べ、選択行を 1 つ持つ
`internal/ui` は Bubble Tea の Model として型 `Model` と、取得関数・`gh` クライアント・エディタ起動・起動時の選択肢を受け取る `New(fetcher Fetcher, client gh.GHClient, editor Editor, opts Options) Model` を MUST 提供する。`Fetcher` は `func(ctx context.Context) (*fetch.Result, error)` で、s07 の `fetch.Fetch` を `client` と `repos` で閉じ、s13 `snapshot-cache`「取得が成功するたびにスナップショットを保存する」の保存で包んだものを `cmd/sugi-loop` が渡す。`client` は書き込み（s10 `answer-question` の `a`。後続の `t` / `m` / `n` / `s` も同じ `client` を使う）に使い、取得には使わない。`Editor` は s10 `answer-question`「a は画面の対象を決めて回答テンプレートを入れたエディタを開く」の型で、`cmd/sugi-loop` は `ExternalEditor(Config.Editor)` を、テストは固定文字列を返すスタブを渡す。`Options` は公開する構造体で、次の 3 つのフィールドを持ち、ゼロ値は「スナップショット無し・自動更新無し・通知無し」である（s13 が導入。テストは `Options{}` を渡してよい）。
- `Snapshot *snapshot.Snapshot`: `Model` が起動時の stale 表示に使う Card 群と保存時刻を持つ。`nil` なら `Model` は空の画面から始める（s13 `snapshot-cache`「起動時にスナップショットがあれば stale 表示から始める」）
- `RefreshInterval time.Duration`: 自動更新の間隔。0 なら自動更新しない（s13 `auto-refresh`）
- `Notify Notifier`: デスクトップ通知の関数。`nil` なら通知しない（s13 `desktop-notify`）
`Model` は `classify` を呼ばず、`fetch.Result.Cards` の `Card.Result` をそのまま使う。
`Model` は保持中の `Cards` を `Card.Result.Tab` で mvp.md の 4 タブ（`model.TabNow` / `TabBacklog` / `TabInProgress` / `TabAbnormal`）に振り分け、1 枚の Card を 1 行にする。Card は必ずどれか 1 つのタブに入る（s05 の `classify.Card` は `Card.Result.Tab` を空にしない）。タブ内の並びは第 1 キー `Card.Result.Priority` 昇順、第 2 キー行の主体の `UpdatedAt` 降順（新しいものが上）、第 3 キー主体の `Repo` 昇順、第 4 キー主体の番号昇順とする（第 1 キーは s05 が定め、第 2 キー以降は design.md の未決事項で定めた既定値）。
`Model` は現在のタブ（初期値は今やる）と、画面全体で 1 つの選択行の添字（初期値 0）を持つ。選択行の添字をタブごとに持たないのは、タブ切替のたびに先頭を見る mvp.md の「先頭から捌く」体験に合わせるためである。選択行の添字は常に `0 ≤ 添字 < そのタブの行数` に収め、行数 0 のタブでは選択行が無い。タブ切替と `Cards` の差し替えで行数が減ったら添字を末尾に丸める。

#### Scenario: example の Card が 4 タブに振り分けられる
- **WHEN** s07 の `fetch.Fetch` に `gh.NewFake("../gh/testdata/fixtures/example")` を渡して得た `Result`（issue 108 + PR 131 の Card が `A`、issue 140 の Card が `E`）を `Model` に取得完了として渡す
- **THEN** 今やるタブの行は issue 108 の Card の 1 行、バックログタブの行は issue 140 の Card の 1 行、進行中タブと異常タブは 0 行である

#### Scenario: タブ内は優先度順、同じ優先度なら新しい順
- **WHEN** `Card.Result.Priority` が 3 で主体の `UpdatedAt` が 5 時間前・9 時間前・1 日前の Card 3 枚と、`Priority` が 1 で 1 時間前・12 分前の Card 2 枚を、この順で `Cards` に持つ `Result` を `Model` に渡す
- **THEN** 今やるタブの行は上から 12 分前（1）、1 時間前（1）、5 時間前（3）、9 時間前（3）、1 日前（3）の順である

#### Scenario: Cards の差し替えで行数が減ったら選択行を末尾に丸める
- **WHEN** 今やるタブに 3 行ある状態で選択行を 2（3 行目）に動かした後、今やるタブが 1 行になる `Result` を `Model` に渡す
- **THEN** 選択行の添字は 0 である

#### Scenario: New は client と editor を保持する
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` で新しく作った `Fake`（`Result` を作るのに使った `Fake` とは別のもの。s07 の `Fetch` は `ViewIssue` を `Calls` に記録する）と、呼ばれた回数を数えるスタブ `Editor` で `New` した `Model` に、`example` の `Result` を取得完了として渡してから `a` を与え、返ったコマンドを実行する
- **THEN** スタブは 1 回呼ばれ、`New` に渡した `Fake` の `Calls` は空である（エディタを開いただけでは書き込まない）

#### Scenario: Options のゼロ値は従来どおりの Model になる
- **WHEN** `Options{}` を渡して `New` した `Model` の `Init` を呼び（コマンドは実行しない）、`View` から ANSI エスケープを除いて読む
- **THEN** 表は 0 行、ヘッダに `↻ --:--`、フッタに `取得中` が含まれ、`fetching` は true である

### Requirement: 取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する
`internal/ui` は `Fetcher` を `context.Background()` で実行し、その完了を `Result`（または error）と完了時刻を運ぶメッセージ `fetchedMsg` として返すコマンドを非公開関数 `fetchCmd(fetcher Fetcher)` として MUST 持ち、`Model` の `Init` はスピナーの tick とそのコマンドをまとめて返す（s12 `manual-refresh` の `startFetch()`。s13 `auto-refresh` の間隔が 0 より大きければ自動更新の tick のコマンドも束ねる）。完了メッセージは `Update` に届く。`Model` は `gh` を直接呼ばず、取得の実行は必ずコマンド（別ゴルーチン）で行う。
- 取得中: フッタの右側にスピナー（Bubbles のスピナー）と `取得中` を出す。スナップショット無し（s13 `snapshot-cache`）の初回取得前は `Cards` が空なので表は 0 行、プレビューは 0 行の表示、ヘッダは `↻ --:--`。スナップショットがあればその `Cards` と保存時刻を表示したまま取得中になる
- 成功（error が nil）: `Cards` を `Result.Cards` で差し替え、最終更新時刻を完了時刻にし、スピナーを消す。`Result.Errors` が 1 件以上なら、フッタの右側に `詳細取得の失敗 <n> 件: <Errors[0] の文字列>` を赤で出す（部分失敗でも Card は `Result.Cards` のとおり表示する。s07「詳細取得の失敗は部分失敗として Card を残す」）。差し替えの前後の `Cards` の差分による通知は s13 `desktop-notify` が定める
- 失敗（error が非 nil）: `Cards` と最終更新時刻を変えず（D-002「失敗時は前回結果を維持」。スナップショットからの stale 表示も維持する）、スピナーを消し、フッタの右側に error の文字列を赤で出す。スナップショット無しの初回取得の失敗なら `Cards` は空のまま
- エラー表示は次の取得が始まったとき（スピナーに置き換わる）に消える。次の取得は s12 `manual-refresh` の `R` と s13 `auto-refresh` の tick が起こす

#### Scenario: fetchCmd が Fetcher を実行して fetchedMsg を返す
- **WHEN** 呼ばれた回数を数える `Fetcher` で `fetchCmd` を作って実行し、得たメッセージを同じ `Fetcher` で `New` した `Model` の `Update` に渡す
- **THEN** `Fetcher` が 1 回呼ばれ、メッセージは `fetchedMsg` で、`Update` 後の `Cards` はその `Result.Cards` である

#### Scenario: 初回取得前は空の画面とスピナー
- **WHEN** スナップショット無しで `New` した直後に `Init` を呼び（コマンドは実行しない）、`View` から ANSI エスケープを除いて読む
- **THEN** 表は 0 行で、フッタに `取得中` が含まれ、ヘッダに `↻ --:--` が含まれる

#### Scenario: 取得成功で Cards と時刻が入る
- **WHEN** `example` の `Result` と完了時刻 `12:04` を取得完了として渡す
- **THEN** 今やるタブに 1 行、バックログに 1 行あり、ヘッダに `↻ 12:04` が出て、フッタに `取得中` は無い

#### Scenario: 取得失敗で前回の Cards が残りエラーが赤で出る
- **WHEN** `example` の `Result` を取得完了として渡した後、error `search issues: gh search issues: exit 1: rate limited` を取得失敗として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** 今やるタブに issue 108 の Card の行が残り、ヘッダの `↻ 12:04` は変わらず、フッタに `rate limited` が含まれ、`取得中` は含まれない

#### Scenario: 部分失敗は Card を出しつつ件数を赤で出す
- **WHEN** `Cards` 1 枚と `Errors` 2 件（先頭が `ViewPR org/app#131: open pr-131.json: no such file`）の `Result` を取得完了として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** 表に 1 行あり、フッタに `詳細取得の失敗 2 件: ViewPR org/app#131` が含まれる

#### Scenario: 初回取得の失敗
- **WHEN** スナップショット無しで `New` した直後に error を取得失敗として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** 表は 0 行、ヘッダは `↻ --:--`、フッタに error の文字列が含まれ、`取得中` は含まれない
