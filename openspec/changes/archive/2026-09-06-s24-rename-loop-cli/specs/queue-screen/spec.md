## MODIFIED Requirements

### Requirement: Model は Card をタブ別に並べ、選択行を 1 つ持つ
`internal/ui` は Bubble Tea の Model として型 `Model` と、取得関数・`gh` クライアント・エディタ起動・起動時の選択肢を受け取る `New(fetcher Fetcher, client gh.GHClient, editor Editor, opts Options) Model` を MUST 提供する。`Fetcher` は `func(ctx context.Context) (*fetch.Result, error)` で、s07 の `fetch.Fetch` を `client` と `repos` で閉じ、s13 `snapshot-cache`「取得が成功するたびにスナップショットを保存する」の保存で包んだものを `cmd/loop-cli` が渡す。`client` は書き込み（s10 `answer-question` の `a`。後続の `t` / `m` / `n` / `s` も同じ `client` を使う）に使い、取得には使わない。`Editor` は s10 `answer-question`「a は画面の対象を決めて回答テンプレートを入れたエディタを開く」の型で、`cmd/loop-cli` は `ExternalEditor(Config.Editor)` を、テストは固定文字列を返すスタブを渡す。`Options` は公開する構造体で、次の 3 つのフィールドを持ち、ゼロ値は「スナップショット無し・自動更新無し・通知無し」である（s13 が導入。テストは `Options{}` を渡してよい）。
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

### Requirement: ヘッダはタブ名と件数と最終更新時刻、フッタはキーヒントとステータスを出す
画面の状態がキューのとき、`View` の 1 行目（ヘッダ）は、アプリ名 `loop-cli` と空白 2 列に続けて 4 タブを mvp.md の形式 `[1]今やる <n>  [2]バックログ <n>  [3]進行中 <n>  [4]異常 <n>`（タブ間は空白 2 列）で MUST 出す。`<n>` はそのタブの行数。現在のタブは太字で、他のタブは通常で描く。ヘッダの右端に右寄せで、左に最低 1 列の空白を置いて `↻ HH:MM`（最後に取得が完了した時刻。24 時間表記。完了時刻はメッセージが運ぶ `time.Time` をそのタイムゾーンのまま書く。`cmd/loop-cli` は `time.Now()` を渡すのでローカル時刻になる）を出し、初回取得の完了前は `↻ --:--` とする。s23 `self-update` の更新の確認が「新しい版がある」を返しているときは、`↻ HH:MM` の左に空白 2 列を空けて `↑ update` を MUST 出す。返していないとき・確認が失敗したとき・確認を行わないときは出さない。
ヘッダの表示幅が端末幅を超えるときは、件数付きタブ名を `[4]` → `[3]` → `[2]` の順に `[n] <n>`（タブ名を落とし番号と件数だけ）に短縮し、収まった時点で止める。`[1]` は短縮しない。`[2]`〜`[4]` を全部短縮しても超えれば `↑ update` を省き、それでも超えれば `↻ HH:MM` を省き、それでも超えれば行を端末幅で切る。この規則で `example` のヘッダ `loop-cli  [1]今やる 1  [2]バックログ 1  [3]進行中 0  [4]異常 0` + 空白 1 + `↻ 12:04` は 70 列（全角 2 列）になり、`↑ update` が出ているときは 80 列ちょうどになる。
`View` の最終行（フッタ = ステータスバー）は、左にキュー画面で動く操作キーのヒント `Enter 開く  a 回答  t todo  o ブラウザ  R 更新  ? ヘルプ  u URL  q 終了`（表示幅 71 列。mvp.md の画面構成のフッタ例と同じ構成で、移動系のキー `j` / `k` / `1`–`4` / `Tab` はヒントに出さず `?` のヘルプに委ねる。s11 までのヒント `j/k 移動  1-4/Tab タブ  …` に 3 キーを足すと 88 列になり、`Model` の既定幅 80 で `q 終了` が切れ、取得中はステータスに押されてヒント全体が消えるため。design.md 未決事項）を出し、右にステータス（Requirement「取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する」と、s10 `answer-question` の回答のステータス、s11 `todo-toggle` の切り替えのステータス、s12 `browse-open` の失敗のステータス、s22 `url-picker` の `URL がありません` と失敗のステータス）を出す。後続 change が自分のキーのヒントを足す。動かないキーのヒントは出さない。カード詳細画面と PR 詳細画面のヘッダとフッタは s09 `card-detail` が、確認画面は s10 `answer-question` が、ヘルプ画面は s12 `help-screen` が定める。

#### Scenario: ヘッダの件数
- **WHEN** `example` の `Result` を完了時刻 `2026-09-05T12:04:00+09:00` で `Model` に渡し、`View` から ANSI エスケープを除いて読む
- **THEN** 1 行目に `[1]今やる 1`、`[2]バックログ 1`、`[3]進行中 0`、`[4]異常 0`、`↻ 12:04` が含まれる

#### Scenario: 狭い端末ではタブ名を右から短縮する
- **WHEN** `example` の `Result` を完了時刻 `2026-09-05T12:04:00+09:00` で渡した `Model` に幅 60・高さ 40 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** 1 行目に `[1]今やる 1`、`[2]バックログ 1`、`[3] 0`、`[4] 0`、`↻ 12:04` が含まれ、`進行中` / `異常` は含まれない（`[4]` と `[3]` を短縮した時点で 60 列に収まり、`[2]` は短縮しない）

#### Scenario: 更新があるとヘッダに印が出る
- **WHEN** `example` の `Result` を完了時刻 `2026-09-05T12:04:00+09:00` で渡した `Model` に「新しい版がある」の結果のメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** 1 行目に `↑ update` が `↻ 12:04` より左に含まれる

#### Scenario: 既定幅 80 では短縮せずに収まる
- **WHEN** 同じ `Model`（幅 80・高さ 24）の `View` から ANSI エスケープを除いて読む
- **THEN** 1 行目に `[1]今やる 1`、`[2]バックログ 1`、`[3]進行中 0`、`[4]異常 0`、`↑ update`、`↻ 12:04` が含まれる

#### Scenario: 幅 79 では [4] のタブ名だけが落ちる
- **WHEN** 同じ `Model` に幅 79・高さ 40 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** 1 行目に `[1]今やる 1`、`[2]バックログ 1`、`[3]進行中 0`、`[4] 0`、`↑ update`、`↻ 12:04` が含まれ、`異常` は含まれない

#### Scenario: 幅が足りなければ更新の印を先に落とす
- **WHEN** 同じ `Model` に幅 59・高さ 40 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** 1 行目に `↻ 12:04` は含まれ、`↑ update` は含まれない

#### Scenario: 初回取得前のヘッダ
- **WHEN** `New` 直後の `Model` の `View` から ANSI エスケープを除いて読む
- **THEN** 1 行目に `[1]今やる 0` と `↻ --:--` が含まれる

#### Scenario: フッタのキーヒント
- **WHEN** `Model` の `View` から ANSI エスケープを除いて読む
- **THEN** 最終行に `Enter 開く`、`a 回答`、`t todo`、`o ブラウザ`、`R 更新`、`? ヘルプ`、`u URL`、`q 終了` がこの順で含まれ、`j/k 移動` / `1-4/Tab タブ` / `m merge` / `Esc 戻る` は含まれない

#### Scenario: 既定幅 80 で取得中でもヒントとスピナーが両方出る
- **WHEN** `New` 直後（幅 80・高さ 24、初回取得中）の `Model` の `View` から ANSI エスケープを除いて読む
- **THEN** 最終行に `Enter 開く` と `q 終了` と `取得中` がすべて含まれる
