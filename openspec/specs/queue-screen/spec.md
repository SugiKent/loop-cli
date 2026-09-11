# queue-screen Specification

## Purpose
TBD - created by archiving change s08-queue-screen. Update Purpose after archive.

## Requirements

### Requirement: Model は Card をタブ別に並べ、選択行を 1 つ持つ
`internal/ui` は Bubble Tea の Model として型 `Model` と、取得関数・`gh` クライアント・エディタ起動・起動時の選択肢を受け取る `New(fetcher Fetcher, client gh.GHClient, editor Editor, opts Options) Model` を MUST 提供する。`Fetcher` は `func(ctx context.Context) (*fetch.Result, error)` で、s07 の `fetch.Fetch` を `client` と `repos` で閉じ、s13 `snapshot-cache`「取得が成功するたびにスナップショットを保存する」の保存で包んだものを `cmd/loop-cli` が渡す。`client` は書き込み（s10 `answer-question` の `a`。後続の `t` / `m` / `n` / `s` も同じ `client` を使う）に使い、取得には使わない。`Editor` は s10 `answer-question`「a は画面の対象を決めて回答テンプレートを入れたエディタを開く」の型で、`cmd/loop-cli` は `ExternalEditor(Config.Editor)` を、テストは固定文字列を返すスタブを渡す。`Options` は公開する構造体で、次の 5 つのフィールドを持ち、ゼロ値は「スナップショット無し・自動更新無し・通知無し・更新の確認無し・merge 方式は既定」である（s13 が導入。テストは `Options{}` を渡してよい）。
- `Snapshot *snapshot.Snapshot`: `Model` が起動時の stale 表示に使う Card 群と保存時刻を持つ。`nil` なら `Model` は空の画面から始める（s13 `snapshot-cache`「起動時にスナップショットがあれば stale 表示から始める」）
- `RefreshInterval time.Duration`: 自動更新の間隔。0 なら自動更新しない（s13 `auto-refresh`）
- `Notify Notifier`: デスクトップ通知の関数。`nil` なら通知しない（s13 `desktop-notify`）
- `CheckUpdate UpdateChecker`: 起動時に 1 度だけ新しい版があるかを調べる関数。`nil` なら調べない（s23 `self-update`）
- `MergeMethods map[string]string`: リポジトリ名から merge 方式を引く対応表。`nil` または該当が無ければ `squash` を使う（s14 `merge-pr`「merge 方式はリポジトリ名から引く」）
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

### Requirement: 行の主体は Card.Result を出した Issue または PR
`Model` は各 Card について、リポジトリ / 番号 / タイトル / 経過の列に使う「主体」を MUST 次の順で決める。`model.Result` は比較可能な struct であり、`==` で比べる。
1. `Card.Issue` があり、`Issue.Result == Card.Result` なら Issue
2. そうでなければ `Card.PRs` の並び順で最初に `PRs[i].Result == Card.Result` を満たす PR
3. どちらにも当たらなければ `Card.Issue`（nil でなければ）、それも無ければ `PRs[0]`
この順は s05 `classify.Card` の同点判定（Issue を優先し、次に `PRs` の並び順で先のもの）と、候補が無いときの `Summary` の採り方（先頭の open PR、無ければ Issue）に一致する。
`internal/ui` はこの決定を関数 `Subject(c model.Card) (repo string, number int, isPR bool, title string, updatedAt time.Time)` として公開する（s09 のカード詳細のヘッダも同じ主体を使う）。

#### Scenario: PR が局面を決めた Card の主体は PR
- **WHEN** `example` の issue 108 の Card（`Issue.Result.Situation` が `in-progress`、`PRs[0].Result.Situation` が `A`、`Card.Result.Situation` が `A`）で `Subject` を呼ぶ
- **THEN** `repo` は `org/app`、`number` は 131、`isPR` は true、`title` は PR 131 の `Title`、`updatedAt` は PR 131 の `UpdatedAt` である

#### Scenario: Issue が局面を決めた Card の主体は Issue
- **WHEN** `example` の issue 140 の Card（PR 無し、`Card.Result.Situation` が `E`）で `Subject` を呼ぶ
- **THEN** `repo` は `org/app`、`number` は 140、`isPR` は false である

#### Scenario: 進行中に落ちた Card の主体は先頭の open PR
- **WHEN** Issue の `Result` が `in-progress`（`Summary` `#5 は進行中`）、`PRs[0]` の `Result` が `in-progress`（`Summary` `PR #9 は auto-fix が受け取り中`）で、`Card.Result` が `PRs[0].Result` と等しい Card で `Subject` を呼ぶ
- **THEN** `number` は 9、`isPR` は true である

### Requirement: 表の行は優先記号・種別・リポジトリ・番号・タイトル・経過を種別の色で出す
`Model` の `View` は現在のタブの各行を、mvp.md の列順 優先 / 種別 / リポジトリ / # / タイトル / 経過 で MUST 描く。各列の表記は次のとおりで、`internal/ui` は 1 枚の Card 分の行を返す関数を持つ（タイトルの折り返しにより 1 枚が 2 行以上になることがある。列幅は design.md の未決事項の既定値）。
- 優先: `Card.Result.Priority` を記号にする。mvp.md の画面例にある 1 → `!!`、2 → `!`、3 → `●` を固定とし、それ以外の値の記号は design.md の未決事項の既定値（0 → `!!!`、5 → `●`、4 / 6 / 7 → `-`）
- 種別: `Card.Result.Situation.Kind()` の文字列（s05 が定めた 質問 / 方針 / merge / todo 候補 / 異常 / その他 / 進行中）
- リポジトリ: 主体の `Repo`（`owner/name`）
- 番号: 主体が Issue なら `#<n>`、PR なら `PR<n>`（s06 の `classify` サブコマンドと同じ表記）
- タイトル: 主体の `Title`。列幅は端末幅から他の列の合計 48 を引いた残り。**列幅に収まらなければ列幅で折り返し、`Title` を全文出す**（#2）。折り返して生まれた継続行は、タイトル列の開始位置（先頭から 43 列）まで空白を置いてからタイトルの続きを書き、経過の列には何も書かない。これにより継続行はタイトルの開始位置に縦が揃う。幅の判定と折り返し位置は表示幅（`ansi.StringWidth` が返す値）で決め、全角文字と絵文字を含むタイトルでも桁がずれない。折り返し位置にあった空白 1 個は改行に置き換わって消える（`ansi.Wrap` の仕様。design.md D1）。列幅が 2 列未満なら（端末幅が 49 以下。タイトル列に全角 1 文字が入らない幅）タイトルを出さず、行は端末幅で切る
- 経過: Requirement「経過は最終更新時刻を基準に m / h / d で書く」の表記。1 枚の Card が複数行になるときは 1 行目にだけ書く
- 選択行の先頭に `▶` を置き、非選択行は同じ幅の空白にする。1 枚の Card が複数行になるときは 1 行目にだけ `▶` を置く

1 枚の Card が生む各行は、表示幅が端末の幅と等しくなるように末尾まで空白で埋める。これにより異常（黄の背景）の Card が複数行になっても、背景色が行ごとに途切れない。
表はスクロールを持たないので、折り返しで増えた行は表の高さに収まらなくなった分だけ捨てられる（Requirement「キュー画面は端末サイズによらず表とプレビューを 2 ペインで出す」が定めた高さの割り当てに従う）。1 枚の Card の途中の行までしか表示されないことがある。
行全体の色は `Kind()` で固定する（mvp.md「行の色は種別で固定」）: 質問 = マゼンタ（`#D876E3`。GitHub の `question` ラベル色）、方針 = 赤（`#B60205`。`blocked` の色）、merge = 緑（`#0E8A16`。`propose` の色）、todo 候補 = シアン、異常 = 黄の背景。その他と進行中は色を付けない（mvp.md に色の指定が無い）。継続行にも 1 行目と同じ色を付ける。シアンと黄の具体値は design.md の未決事項の既定値。

#### Scenario: A の Card の行
- **WHEN** `example` の issue 108 の Card（`Priority` 1、`Kind()` 質問、主体は PR 131）を行にし、ANSI エスケープを除いて読む
- **THEN** 1 行目に `!!`、`質問`、`org/app`、`PR131` が、この順で含まれ、全行のタイトル列を連結すると PR 131 の `Title` と一致する

#### Scenario: E の Card の行
- **WHEN** `example` の issue 140 の Card（`Priority` 4、`Kind()` todo 候補、主体は issue 140）を行にし、ANSI エスケープを除いて読む
- **THEN** 1 行目に `-`、`todo 候補`、`org/app`、`#140` が、この順で含まれ、全行のタイトル列を連結すると issue 140 の `Title` と一致する

#### Scenario: 長いタイトルは切り詰める
- **WHEN** 端末幅が 49 以下（タイトル列の幅が 2 列未満）の `Model` で Card を行にする
- **THEN** 行は 1 行だけで、タイトルは出ず、行の表示幅は端末幅以下である

#### Scenario: 長いタイトルは折り返して全文出す
- **WHEN** 幅 80 の `Model` で、タイトル列の幅（80 − 48 = 32）より表示幅が大きい `Title` を持つ Card を行にし、ANSI エスケープを除いて読む
- **THEN** 行は 2 行以上あり、各行の 43 列目以降（タイトル列）を取り出して末尾の空白を除いて連結すると `Title` と一致し（折り返し位置に空白があった場合はその 1 個だけが消えるので、空白を除いて比べる）、`…` で終わる行は無く、どの行も表示幅が 80 である

#### Scenario: 折り返した行はタイトルの開始位置に揃い、経過は 1 行目だけに出る
- **WHEN** 幅 80 の `Model` で、`Title` が全角 30 字、`UpdatedAt` が取得完了時刻の 12 分前の Card を行にし、ANSI エスケープを除いて読む
- **THEN** 行は 2 行あり、1 行目にだけ `12m` が含まれ、2 行目の先頭 43 列はすべて空白で、44 列目からタイトルの続きが始まる

#### Scenario: 選択行に印が付く
- **WHEN** 今やるタブに 2 枚の Card がある状態で `View` の文字列から ANSI エスケープを除いて読む
- **THEN** 1 枚目の 1 行目の先頭に `▶` があり、その継続行と 2 枚目の行の先頭には無い

### Requirement: 経過は最終更新時刻を基準に m / h / d で書く
`internal/ui` は `Elapsed(now, t time.Time) string` を MUST 提供し、s06 `dev-cli` の `elapsed` と同じ規則で書く: 差が 1 時間未満なら `<m>m`、24 時間未満なら `<h>h`、それ以上なら `<d>d`。いずれも切り捨て。`t` が `now` より後なら `0m`。
画面の `now` は最後に取得が完了した時刻（Requirement「取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する」の完了時刻）であり、`Model` は壁時計を直接読まない。次の取得（s12 の `R`、s13 の自動更新）まで経過の表示は変わらない。

#### Scenario: 経過の表記
- **WHEN** `Elapsed(now, t)` に `t` が `now` の 12 分前・3 時間前・2 日前・未来（`now` より後）の値を渡す
- **THEN** 戻り値はそれぞれ `12m` / `3h` / `2d` / `0m` である

#### Scenario: 経過は取得完了時刻を基準にする
- **WHEN** 主体の `UpdatedAt` が `2026-09-05T09:00:00Z` の Card を持つ `Result` を、完了時刻 `2026-09-05T12:04:00Z` の取得完了として `Model` に渡し、`View` を読む
- **THEN** その行の経過は `3h` である

### Requirement: j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、他のキーは何もしない
`Model` の `Update` は、画面の状態がキュー（s09 `card-detail`「Enter でカード詳細を開き、Esc で 1 つ前の画面に戻る」）のとき、キー入力を MUST 次のとおり扱う。
- `j` / `↓`: 選択行を 1 つ下へ。末尾では動かない（循環しない）
- `k` / `↑`: 選択行を 1 つ上へ。先頭では動かない
- `1` / `2` / `3` / `4` を押すと、それぞれ今やる / バックログ / 進行中 / 異常のタブに切り替える。`Tab` を押すと次のタブに切り替える（異常の次は今やる）。切替後の選択行の添字は Requirement「Model は Card をタブ別に並べ、選択行を 1 つ持つ」の丸め規則に従う（切替前の添字を引き継ぎ、行数を超えていれば末尾）
- `Enter`: 選択行の Card のカード詳細画面（`Issue` が nil なら PR 詳細画面）を開く。選択行が無ければ何もしない。振る舞いは s09 `card-detail` が定める
- `a`: 選択行の主体に回答するエディタを開く。選択行が無ければ何もしない。振る舞いは s10 `answer-question` が定める
- `t`: 選択行の Card の `Issue` の `stage:todo` を切り替える。選択行が無い、または `Issue` が nil なら何もしない。振る舞いは s11 `todo-toggle` が定める
- `m`: 選択行の主体が PR なら、その PR の状態を取り直して merge の確認画面を開く。選択行が無い、または主体が issue なら何もしない。振る舞いは s14 `merge-pr` が定める
- `c`: 選択行の主体（issue でも PR でもよい）の close の確認画面を開く。選択行が無ければ何もしない。振る舞いは s26 `close-issue-pr` が定める
- `n`: 選択行の主体のリポジトリに新しい issue を作るエディタを開く。選択行が無ければ画面を変えずステータスに出す。振る舞いは s15 `new-issue` が定める
- `o`: 選択行の主体をブラウザで開く。選択行が無ければ何もしない。振る舞いは s12 `browse-open` が定める
- `R`: 全件を再取得する。取得中なら何もしない。振る舞いは s12 `manual-refresh` が定める
- `?`: ヘルプ画面を開く。振る舞いは s12 `help-screen` が定める
- `u`: 選択行の主体に含まれる URL の一覧画面を開く。URL が 1 件も無ければ画面を変えずステータスに出す。選択行が無ければ何もしない。振る舞いは s22 `url-picker` が定める
- `L`（`Shift` + `l`）: 選択行の主体（issue でも PR でもよい）のラベル一覧画面を開く。選択行が無ければ何もしない。振る舞いは s28 `label-picker` が定める
- `q` / `Ctrl+C`: 終了コマンドを返す（s01 `tui-entrypoint`「q で終了する」を `internal/ui` の `Model` が満たす）
- mvp.md の表にある他のキー `A` / `s` / `g` / `x` / `/` / `v` / `h` / `l` / `←` / `→` と、表に無いキー（`p` と `Esc` を含む）は、キュー画面では `Model` を変えず、コマンドも返さない。担当は `s` が後続 change（s15 の枠のうち `n` だけをこの change が実装した）、`A` が s16、`v` / `h` / `l` / `←` / `→` が s17、`/` が s19。`g` / `x` / `Esc` は詳細画面のキーであり s09 `card-detail` が定める（キュー画面では何もしない）。`p` は s21 まで表とプレビューの切替だったが、2 ペインが常設になったため何もしないキーに戻った。`u` は s22 が、`L` は s28 が、`c` は s26 が mvp.md キーバインド表に足すキーで、どれも上の箇条書きのとおりキュー画面で動く（`c` は s26 より前は表に無いキーとして何もしなかった）。小文字の `l` は s17（カンバンの列移動）の予約のままで、キュー画面では何もしない

#### Scenario: j と k で選択行が動く
- **WHEN** 今やるタブに 3 行ある `Model` に `j` を 2 回、`k` を 1 回、`↓` を 1 回、`↑` を 1 回の順で与える
- **THEN** 選択行の添字は順に 1、2、1、2、1 になる

#### Scenario: 末尾と先頭で止まる
- **WHEN** 今やるタブに 2 行ある `Model` に `j` を 3 回与えた後、`k` を 3 回与える
- **THEN** `j` の後の添字は 1、`k` の後の添字は 0 である

#### Scenario: 数字と Tab でタブが切り替わる
- **WHEN** `Model` に `2`、`4`、`Tab`、`1`、`3` の順で与える
- **THEN** 現在のタブは順に バックログ、異常、今やる、今やる、進行中 になる

#### Scenario: 未実装のキーは何も変えない
- **WHEN** 今やるタブに 2 行あり選択行が 1 の `Model` に `v`、`s`、`A`、`g`、`x`、`Esc`、`/`、`p`、`l` を 1 つずつ与える
- **THEN** どのキーでもコマンドは返らず、画面の状態はキューのままで、現在のタブ・選択行・`Cards` は変わらない

#### Scenario: q で終了する
- **WHEN** `Model` に `q` を与える
- **THEN** 終了コマンドが返る

### Requirement: プレビューは選択行の 1 行目・本文・コメントを出す
`View` はプレビュー領域に、選択行の Card について MUST 次を上から順に描く。
1. 1 行目: 主体の `Body` の 1 行目（先頭の空行を除いた最初の行）に続けて `labels: <主体の Labels を空白区切り>`（mvp.md の画面例の形）。`Labels` が空なら `labels:` を省く。`Body` も空なら 1 行目を出さない。`Card.Result.Summary` はこの画面には出さない（s09 のカード詳細ヘッダで使う）
2. 本文: 主体の `Body` を Glamour で幅 = プレビュー幅で Markdown レンダリングした文字列。レンダリングが失敗したら `Body` をそのまま出す
3. コメント: 主体の `Comments` を並び順（末尾が最新）に出す。各コメントは見出し行と本文の行からなる。本文は Glamour で幅 = プレビュー幅 − 1 で Markdown レンダリングし（D-003「Glamour: Issue / PR 本文とコメントを色付きで端末表示」）、失敗したら本文をそのまま出す。`AI` が true のコメントは見出しを `AI  HH:MM`（`CreatedAt` を取得完了時刻と同じタイムゾーンに直した 24 時間表記）とし、見出しとレンダリング後の本文の全行の左端に縦バー `▌` を付ける。`AI` が false のコメントは見出しを `<Author>  HH:MM` とし、バーを付けない。`Comments` が nil または空ならコメント部分は出さない（詳細を取っていない主体もある。何を取るかは s07 の範囲）
コメント本文の routine マーカー行は、レンダラが出すかどうかに関わらず、レンダリング前に取り除いてプレビューに出さない。除去規則は「`strings.TrimSpace` 後の行全体が `<!-- routine -->` または `&lt;!-- routine --&gt;` に一致する行を落とす」であり、取り除くのはその行だけである（マーカーを含む他の行や部分一致は触らない）。
プレビューは領域の高さを超えた分を出さない。プレビューのスクロールキーは mvp.md に無く、この change では持たない。選択行が無い（タブが 0 行）ときは `（このタブにはカードがありません）` と 1 行出す。

#### Scenario: A の Card のプレビュー
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、`example` の `Result` を渡し、今やるタブで issue 108 の Card（主体 PR 131。コメント 1 件が `<!-- routine -->` 始まり）を選択した `View` から ANSI エスケープを除いて読む
- **THEN** プレビューの 1 行目に `issue #108 の提案。` と `labels: propose question` が含まれ、`▌AI` で始まる行と `▌` 付きの `Q1: マイグレーションを分けますか。` が含まれ、`PR #131 の質問に答える`（`Summary`）、`<!-- routine -->`、`&lt;!-- routine --&gt;` は含まれない

#### Scenario: 人のコメントにはバーが付かない
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、主体の `Comments` に `AI` が false、`Author` が `user-2`、本文 `Q1: A` のコメントを持つ Card を選択した `View` から ANSI エスケープを除いて読む
- **THEN** `user-2` を含む見出し行と `Q1: A` の行があり、どちらも `▌` で始まらない

#### Scenario: コメントもラベルも無い Card のプレビュー
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、`example` のバックログタブで issue 140 の Card（`Labels` 空、`Comments` nil）を選択した `View` を読む
- **THEN** プレビューに `起動時に設定ファイルが無いと落ちる` が含まれ、`labels:` と `▌` は含まれない

#### Scenario: 0 行のタブ
- **WHEN** `example` の `Result` を渡して異常タブに切り替えた `View` を読む
- **THEN** プレビューに `（このタブにはカードがありません）` が含まれる

### Requirement: キュー画面は端末サイズによらず表とプレビューを 2 ペインで出す
`Model` は端末サイズのメッセージで幅と高さを MUST 保持し、サイズを受け取る前は幅 80・高さ 24 とみなす。キュー画面は端末の幅と高さによらず、ヘッダ / 表 / 区切り線 / プレビュー / フッタを上下に並べた 2 ペインで MUST 描く。表とプレビューの高さの上限は、端末の高さからヘッダ 1 行・区切り線 1 行・フッタ 1 行を引いた残りを表に切り上げで按分した値とする（残りを `rest` として表は最大 `(rest + 1) / 2` 行、プレビューは最大 `rest - (rest + 1) / 2` 行）。残りが負なら 0 行とする。行数が上限に足りないときは足りないまま描き、空行で埋めない（フレームは端末の高さ以下になる）。ただし高さが 3 行未満の端末では、常に出るヘッダ・区切り線・フッタの 3 行が端末の高さを超える。行を落とす分岐は持たない。狭い端末でも 1 ペインへのフォールバックは行わない。
表とプレビューを切り替える操作は持たない。フッタ左のヒントは Requirement「ヘッダはタブ名と件数と最終更新時刻、フッタはキーヒントとステータスを出す」が定めるものだけで、端末サイズによる切替のヒントを足さない。

#### Scenario: 広い端末は表とプレビューの両方を出す
- **WHEN** `example` の `Result` を渡した `Model` に幅 120・高さ 40 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** 表の行 `PR131` とプレビューの `issue #108 の提案` の両方が含まれる

#### Scenario: 幅 79 の端末でも表とプレビューの両方を出す
- **WHEN** 同じ `Model` に幅 79・高さ 40 のサイズメッセージを与えて `View` を読む
- **THEN** `PR131` の行と `issue #108 の提案` の両方が含まれ、フッタに `Enter 開く` と `q 終了` が含まれ、`p プレビュー` も `p 一覧` も含まれない

#### Scenario: 高さ 15 の端末でも表とプレビューの両方を出す
- **WHEN** 同じ `Model` に幅 120・高さ 15 のサイズメッセージを与えて `View` を読む
- **THEN** `PR131` の行と `issue #108 の提案` の両方が含まれる

#### Scenario: p を押しても画面は変わらない
- **WHEN** 同じ `Model` に幅 79・高さ 40 のサイズメッセージを与えた後、`p` を与える
- **THEN** コマンドは返らず、`View` は `p` を与える前と同じである

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

### Requirement: 取得が成功して Card が 0 件のときは表の領域にヒントを出す
キュー画面は、直近の取得が成功して `Cards` が 0 件（全タブが空）のとき、表の領域の代わりに次の 2 行を MUST 出す（mvp.md「初回起動（onboarding）」の文言。Markdown のバッククォートは付けない）。2 方式のどちらの利用者も直せるように、両方の入口を示す。方式の判定材料はリポジトリのラベル一覧なので、2 行目はラベルを作る手立てを案内する。
```
stage:* / To Do ラベルの無いリポジトリは何も出ません。
issue-driven-sdd の routines-setup を回すか、issue-label-driven の To Do ラベルを作ってください
```
2 行は表の領域（表の高さ）の縦中央に置き、各行を端末幅の横中央に置く（左に空白を置き、右には足さない）。端末幅より長い行は幅で切る。プレビューの領域は変えない。
次のときは出さない。
- 取得中（`fetching` が true。初回のスピナーを含む）: `Cards` が空でも表は 0 行のまま（「取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する」の初回取得前の表示を変えない）
- 取得失敗（フッタに error の文字列を出しているとき）: 前回結果の維持と誤解させないため。初回取得の失敗で `Cards` が空でも出さない
- 部分失敗（`Result.Errors` が 1 件以上）で `Cards` が 0 件: search は成功しているので出す（フッタの `詳細取得の失敗 …` はそのまま）
ヒントはキュー画面だけに出す。他の画面（s09 の詳細）は変えない。

#### Scenario: 取得成功で 0 件ならヒントが出る
- **WHEN** `Cards` が空で `Errors` も空の `Result` を取得完了として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `stage:* / To Do ラベルの無いリポジトリは何も出ません。` と `issue-driven-sdd の routines-setup を回すか、issue-label-driven の To Do ラベルを作ってください` の 2 行が含まれ、ヘッダの `[1]今やる 0` とフッタは従来どおり出る

#### Scenario: 取得中はヒントを出さない
- **WHEN** `New` 直後（初回取得前）の `Model` の `View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含まず、フッタに `取得中` が含まれる

#### Scenario: 取得失敗ではヒントを出さない
- **WHEN** `New` 直後に error `search issues: gh search issues: exit 1: rate limited` を取得失敗として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含まず、フッタに `rate limited` が含まれる

#### Scenario: Card があればヒントを出さない
- **WHEN** `example` の `Result` を取得完了として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含まず、表に `PR131` の行がある

#### Scenario: ヒントは横中央に置かれる
- **WHEN** `Cards` が空の `Result` を渡した `Model` に幅 120・高さ 40 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含む行の左側の空白の数は、（120 − その行の表示幅）÷ 2 の切り捨てに等しい

#### Scenario: 幅 60 でもヒントは表の領域に出てプレビューも描かれる
- **WHEN** `Cards` が空の `Result` を渡した `Model` に幅 60・高さ 40 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含む行と、プレビュー領域の `（このタブにはカードがありません）` の両方が含まれる

### Requirement: ヘッダはタブ名と件数と最終更新時刻、フッタは close を含むキーヒントとステータスを出す
画面の状態がキューのとき、`View` の 1 行目（ヘッダ）は、アプリ名 `loop-cli` と空白 2 列に続けて 4 タブを mvp.md の形式 `[1]今やる <n>  [2]バックログ <n>  [3]進行中 <n>  [4]異常 <n>`（タブ間は空白 2 列）で MUST 出す。`<n>` はそのタブの行数。現在のタブは太字で、他のタブは通常で描く。ヘッダの右端に右寄せで、左に最低 1 列の空白を置いて `↻ HH:MM`（最後に取得が完了した時刻。24 時間表記。完了時刻はメッセージが運ぶ `time.Time` をそのタイムゾーンのまま書く。`cmd/loop-cli` は `time.Now()` を渡すのでローカル時刻になる）を出し、初回取得の完了前は `↻ --:--` とする。s23 `self-update` の更新の確認が「新しい版がある」を返しているときは、`↻ HH:MM` の左に空白 2 列を空けて `↑ update` を MUST 出す。返していないとき・確認が失敗したとき・確認を行わないときは出さない。
ヘッダの表示幅が端末幅を超えるときは、件数付きタブ名を `[4]` → `[3]` → `[2]` の順に `[n] <n>`（タブ名を落とし番号と件数だけ）に短縮し、収まった時点で止める。`[1]` は短縮しない。`[2]`〜`[4]` を全部短縮しても超えれば `↑ update` を省き、それでも超えれば `↻ HH:MM` を省き、それでも超えれば行を端末幅で切る。この規則で `example` のヘッダ `loop-cli  [1]今やる 1  [2]バックログ 1  [3]進行中 0  [4]異常 0` + 空白 1 + `↻ 12:04` は 70 列（全角 2 列）になり、`↑ update` が出ているときは 80 列ちょうどになる。
`View` の最終行（フッタ = ステータスバー）は、左にキュー画面で動く操作キーのヒント `Enter 開く  a 回答  t todo  L ラベル  m merge  c close  o ブラウザ  R 更新  ? ヘルプ  u URL  q 終了`（表示幅 99 列。`c close` は s26 が `m merge` の次に置く。書き込みを起こすキーを 1 か所にまとめて読めるようにする。`m merge` は mvp.md の画面構成のフッタ例と同じく `t todo` の次に置く。この 9 列を足したことで、幅 80 の端末では取得中や書き込みのステータスが出ている間ヒントが丸ごと消える（s08 のフッタは両方が入らなければステータスを優先する）。キーを隠すよりキーを出すことを採った。s14 design.md。mvp.md の画面構成のフッタ例と同じ構成で、移動系のキー `j` / `k` / `1`–`4` / `Tab` はヒントに出さず `?` のヘルプに委ねる。s11 までのヒント `j/k 移動  1-4/Tab タブ  …` に 3 キーを足すと 88 列になり、`Model` の既定幅 80 で `q 終了` が切れ、取得中はステータスに押されてヒント全体が消えるため。design.md 未決事項）を出し、右にステータス（Requirement「取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する」と、s10 `answer-question` の回答のステータス、s11 `todo-toggle` の切り替えのステータス、s12 `browse-open` の失敗のステータス、s22 `url-picker` の `URL がありません` と失敗のステータス、s14 `merge-pr` の取り直し・merge のステータス、s15 `new-issue` の `新規作成の対象がありません` と作成のステータス、s28 `label-picker` の取得中・更新中・結果のステータス、s26 `close-issue-pr` の中止・close のステータス）を出す。後続 change が自分のキーのヒントを足す。動かないキーのヒントは出さない。
s15 が実装した `n` は、キュー画面で動くがこのヒントには出さない。`n` の入口は `?` のヘルプと詳細画面のフッタ（s09 `card-detail`）が見せる。
s28 `label-picker` の `L ラベル` は `t todo` の次に置き、s26 `close-issue-pr` の `c close` は `m merge` の次に置いて、キュー画面のヒントは 99 列になる。ラベルを触るキーを `t` の隣に並べるためで、`u URL` は 78〜82 列目に動き、既定幅 80 の端末では途中で切れる（s22 は `u URL` を 72 列目までに収めていた）。s14 と s22 が 2 度採った「キーを隠すよりキーを出す」を続け、`L` を幅 80 で見える位置に置くほうを採った（s28 design.md）。ヒントとステータスが両方出る最小の幅は、ヒント 99 列 + 空白 1 列 + 取得中のステータス 8 列（スピナー 1 列 + `" 取得中"` 7 列）で **108 列**になる。
カード詳細画面と PR 詳細画面のヘッダとフッタは s09 `card-detail` が、回答の確認画面は s10 `answer-question` が、merge の確認画面は s14 `merge-pr` が、作成の確認画面は s15 `new-issue` が、close の確認画面は s26 `close-issue-pr` が、ヘルプ画面は s12 `help-screen` が定める。

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
- **WHEN** `Model` に幅 108・高さ 24 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** 最終行に `Enter 開く`、`a 回答`、`t todo`、`L ラベル`、`m merge`、`c close`、`o ブラウザ`、`R 更新`、`? ヘルプ`、`u URL`、`q 終了` がこの順で含まれ、`j/k 移動` / `1-4/Tab タブ` / `Esc 戻る` / `n 新規` は含まれない

#### Scenario: 既定幅 80 で取得中はステータスだけになる
- **WHEN** `New` 直後（幅 80・高さ 24、初回取得中）の `Model` の `View` から ANSI エスケープを除いて読む
- **THEN** 最終行に `取得中` が含まれ、`Enter 開く` と `q 終了` は含まれない（ヒント 99 列とステータスが幅 80 に収まらず、s08 のフッタの規則でステータスを優先する）

#### Scenario: 幅 108 では取得中でもヒントとスピナーが両方出る
- **WHEN** `New` 直後の `Model` に幅 108・高さ 24 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** 最終行に `Enter 開く` と `L ラベル` と `m merge` と `c close` と `q 終了` と `取得中` がすべて含まれる（ヒント 99 列 + 空白 1 列 + ステータス 8 列がちょうど幅 108 に収まる）

#### Scenario: 幅 107 では取得中はステータスだけになる
- **WHEN** `New` 直後の `Model` に幅 107・高さ 24 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** 最終行に `取得中` が含まれ、`Enter 開く` と `c close` は含まれない（1 列足りず、s08 のフッタの規則でステータスを優先する）

### Requirement: 運用方式は取得のたびに判定した結果を持つ
`internal/ui` の `Model` は「リポジトリ名 → 運用方式」の表を状態として MUST 持つ。表の中身は `internal/fetch` の `Result.Modes`（`card-fetch`「Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する」）で、取得が成功したメッセージを受け取るたびに丸ごと差し替える。取得が失敗したメッセージでは触らず、前回の表を残す（D-002「失敗時は前回結果を維持する」）。`Options` は方式を受け取らず、`cmd/loop-cli` は方式を作らない。

表は「そのリポジトリの方式が分かっているか」を区別できる形で持つ。次の 3 つはいずれも「分からない」であり、表に入らない。

- 初回の取得がまだ終わっていない（スナップショットだけを描いている）
- そのリポジトリの `ListLabels` が失敗した
- そのリポジトリが `stage:todo` も `To Do` も持たない

表示（カード詳細の段階行・バッジ・PR 一覧の段階）は表を引き、分からないリポジトリはゼロ値の `sdd` の語彙で描く。分類も同じで、`Card.Result` は必ず 1 つ決まる（`card-fetch`）。書き込み（`t`）だけは分からないリポジトリで止める（`todo-toggle`「t は画面の Card の Issue に対して、判定した方式で確認なしに承認ラベルを切り替える」）。`Cards` の中の値や前回のスナップショットから方式を決めない。スナップショットは前回の実行時の派生データなので、そこから方式を決めると、ラベルを変えた後も古い方式でラベルを書く経路ができる。

#### Scenario: 取得の結果で方式の表が入れ替わる
- **WHEN** `org/board` を `label` と判定した `Result` を取得完了として渡した `Model` に、次の取得で `org/board` を `sdd` と判定した `Result` を渡し、`org/board` の `Labels` が `stage:propose` の issue の Card の詳細を開いて `View` を読む
- **THEN** `段階: stage:propose` が含まれる（前の取得の `label` は残らない）

#### Scenario: 取得が失敗したら前回の表を残す
- **WHEN** `org/board` を `label` と判定した `Result` を渡した `Model` に、取得失敗のメッセージを渡し、`org/board` の `Labels` が `In Progress` の issue の Card の詳細を開いて `View` を読む
- **THEN** `段階: In Progress` が含まれる

#### Scenario: 初回取得の前は方式が分からない
- **WHEN** `Options.Snapshot` に `org/board` の `In Progress` の issue の Card を持たせた `Model` を初回取得の前に描き、その Card の詳細を開いて `View` を読む
- **THEN** `段階なし` が含まれる（方式が分からないので `sdd` の語彙で描く）
