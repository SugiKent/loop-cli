# browse-open Specification

## Purpose
TBD - created by archiving change s12-open-refresh-help. Update Purpose after archive.
## Requirements

### Requirement: o は画面の対象をブラウザで開く
`Model` の `Update` は `o` を画面の状態ごとに MUST 次のとおり扱う。対象と表示名（`<Repo> PR#<n>` または `<Repo> #<n>`。s10 の表示名と同じ表記）を決め、s03 の `GHClient.Browse(ctx, repo, number)`（`Client` は `gh browse <n> -R <repo>` を実行する。issue 番号でも PR 番号でも GitHub がその URL に案内する）を対象の `Repo` / `Number` で呼ぶコマンドを返す。
- キュー画面: 選択行の主体（s08 `Subject`。局面 A / C 等では PR、B / E 等では issue）。選択行が無ければ何もしない
- カード詳細画面: 詳細の対象の `Card.Issue`
- PR 詳細画面: 詳細の対象の PR（選択中の PR）
- 確認画面（s10）とヘルプ画面（s12 `help-screen`）: 何もしない
`Model` は `gh` を直接呼ばず、ブラウザで開く処理は必ずコマンド（別ゴルーチン）で行う。`ctx` は 30 秒のタイムアウト付き（s10 / s11 と同じ値）。書き込みではないので、s11「書き込み中は t と a を受け付けない」の書き込み中フラグでは止めない。`o` を押しただけではフッタ右側のステータス（s10 / s11 の書き込みステータス）を消さない（書き込み中に `o` を押しても `切り替え中` / `投稿中` の表示を残す。design.md 未決事項の既定値）。
mvp.md キーバインド表の `o` は「`gh browse` / `open <url>`」であり、この change は `gh browse` だけを使う（search 結果の `URL` を `open` に渡す経路は作らない。design.md）。

#### Scenario: キュー画面で o を押すと主体がブラウザで開く
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` で新しく作った `Fake`（`Result` を作るのに使った `Fake` とは別のもの。s07 の `Fetch` は `ViewIssue` を `Calls` に記録する）を `client` にして `New` し、`example` の `Result` を取得完了として渡して今やるタブ（issue 108 + PR 131 の Card。主体は PR 131）を選んだ `Model` に `o` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `o` の直後（コマンドを実行する前）の `Fake.Calls` は空で、コマンドの実行後の `Fake.Calls` はちょうど 1 件で `Method` が `Browse`、`Repo` が `org/app`、`Number` が 131 である。画面はキューのままである

#### Scenario: 主体が issue の行では issue が開く
- **WHEN** 同じ準備でバックログタブ（issue 140）を選んだ `Model` に `o` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` は 1 件で `Method` が `Browse`、`Number` が 140 である

#### Scenario: カード詳細では Issue、PR 詳細ではその PR が開く
- **WHEN** `example` の issue 108 の Card のカード詳細を開いた `Model` に `o` を与えてコマンドを実行し、別に同じカードから `Enter` で PR 詳細を開いた `Model` に `o` を与えてコマンドを実行する
- **THEN** 1 つ目の `Fake.Calls` は `Browse` / `org/app` / 108 の 1 件、2 つ目の `Fake.Calls` は `Browse` / `org/app` / 131 の 1 件であり、画面はそれぞれカード詳細 / PR 詳細のままである

#### Scenario: 0 行のタブ・確認画面・ヘルプ画面で o は何もしない
- **WHEN** `example` の `Result` を渡して異常タブ（0 行）に切り替えた `Model` に `o` を与え、別に s10 の手順で blocked-by の確認画面に移った `Model` に `o` を与え、別に `?` でヘルプ画面を開いた `Model` に `o` を与える
- **THEN** どれもコマンドは返らず、`Fake.Calls` は空で、画面は変わらない

#### Scenario: 書き込み中でも o を受け付ける
- **WHEN** バックログの issue 140 を選んだ `Model` に `t` を与えた直後（返ったコマンドを実行する前。s11 の書き込み中）に `o` を与える
- **THEN** コマンドが返り、それを実行すると `Fake.Calls` に `Browse` / 140 が入る

### Requirement: 開けなかったときはステータスに赤で出す
ブラウザで開くコマンドは `Browse` の結果（表示名とエラー）を運ぶ `internal/ui` 内のメッセージを返す。`Model` はそのメッセージを受け取ったら MUST 次のとおり扱う。
- エラーが nil のとき: `Model` は何も出さない（ブラウザが開いたこと自体が結果である。design.md 未決事項の既定値）
- エラーが非 nil のとき: `Model` はフッタの右側に `<表示名> をブラウザで開けません: <エラー文字列>` を赤で出す。この場所は s10 / s11 のステータスと同じで、1 つのステータスを共有し、行を増やさない
成否にかかわらず `Cards` と最終更新時刻と詳細の対象を変えず、取得のコマンドを返さない。このステータスは次の取得が始まったとき、または次に `a` / `t` を押したときに消える（s10 / s11 と同じ寿命）。

#### Scenario: gh の失敗は赤で出る
- **WHEN** `*gh.Fake` を埋め込んで `Browse` だけがエラー `gh browse 131 -R org/app: exit 1: no browser` を返す型を `client` にして、今やるタブ（主体 PR 131）で `o` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡し、`View` から ANSI エスケープを除いて読む
- **THEN** フッタに `org/app PR#131 をブラウザで開けません:` と `no browser` が含まれ、`Cards` は変わらず、画面はキューのままである

#### Scenario: 成功時は何も出ない
- **WHEN** Requirement「o は画面の対象をブラウザで開く」の 1 つ目の Scenario の手順の後に `View` から ANSI エスケープを除いて読む
- **THEN** フッタに `ブラウザ` を含む語は `o ブラウザ` のヒントだけで、`開けません` は含まれない

#### Scenario: 次の取得開始で消える
- **WHEN** Scenario「gh の失敗は赤で出る」の手順の後に `R` を与えて `View` から ANSI エスケープを除いて読む
- **THEN** フッタに `開けません` は含まれず `取得中` が含まれる
