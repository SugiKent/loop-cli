## REMOVED Requirements

### Requirement: t は画面の Card の Issue に対して確認なしで stage:todo を切り替える

**Reason**: 方式の引き先が `Options.Modes`（設定由来）から取得結果の表に変わり、方式が分からないリポジトリで `t` を止める枝が加わる。Scenario「スナップショットから描いた Card でも設定の方式で書く」は「設定から方式が分かるので書ける」ことを固定しており、この change では同じ状況で書かないのが正しい振る舞いになる。Scenario 名が振る舞いを反転させるので MODIFIED では書き直せず、Requirement 名を「t は画面の Card の Issue に対して、判定した方式で確認なしに承認ラベルを切り替える」に改めて作り直す。

**Migration**: 対象を決める規則（画面に出ている Card の `Issue`、PR を対象にしない）と、確認ダイアログを出さないこと、`action.ToggleTodo` が読み直したラベルで付け外しを決めること、切り替え中のフッタの表示は、ADDED の Requirement がそのまま引き継ぐ。加わるのは「方式が分からないリポジトリでは書かずに理由を出す」の 1 点だけである。

## ADDED Requirements

### Requirement: t は画面の Card の Issue に対して、判定した方式で確認なしに承認ラベルを切り替える
`Model` の `Update` は `t` を画面の状態ごとに MUST 次のとおり扱う。対象は「画面に出ている Card の `Issue`」の 1 つの規則で決め、PR を対象にしない（承認ラベルは issue のラベルであり、`gh issue edit` で書く）。
- キュー画面: 選択行の Card の `Card.Issue`。選択行が無い、または `Issue` が nil（PR 単独のカード）なら何もしない（design.md 未決事項の既定値）
- カード詳細画面: 詳細の対象の `Card.Issue`（カード詳細は `Issue` が nil の Card では開かない。s09）
- PR 詳細画面: 何もしない
- 確認画面（s10）: 何もしない（s10「他のキー: 何もしない」のまま）

対象が決まったら、そのリポジトリの運用方式を `queue-screen`「運用方式は取得のたびに判定した結果を持つ」の表から引く。表に無いリポジトリ（初回取得の前、`ListLabels` の失敗、`stage:todo` も `To Do` も持たない）では、**コマンドを返さず、ラベルを 1 つも書かない**。フッタの右側に `<Repo> の運用方式が分かりません（stage:todo / To Do のラベルがありません）` を赤で MUST 出す。推測した方式でラベルを書くと、書いたラベルで worker が起動しないか、方式の違う worker を起動してしまうためである。

表に方式があれば、確認ダイアログを出さずに `action.ToggleTodo` を対象の `Repo` / `Number` とその方式で呼ぶコマンドを返す（mvp.md は `m` にだけ「確認ダイアログ必須」と書いており、`t` には無い。取り消しはもう一度 `t`。design.md 未決事項の既定値）。書くラベルは `model.TodoLabel(mode)` で、`sdd` は `stage:todo`、`label` は `To Do` である。`ctx` は 30 秒のタイムアウト付き（s10 の投稿と同じ値）。`Model` は `gh` を直接呼ばず、切り替えは必ずコマンド（別ゴルーチン）で行う。対象の `Issue.Labels` を判定に使わない（判定は `ToggleTodo` が読み直したラベルで行う）。
コマンドを返すとき、フッタの右側に `<Repo> #<Number> の <ラベル> を切り替え中` を出す（`<ラベル>` は `model.TodoLabel(mode)`）。表示名 `<Repo> #<Number>` は s09 の詳細ヘッダ・s10 の表示名と同じ表記である。

#### Scenario: バックログの issue で t を押すと stage:todo が付く
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` で新しく作った `Fake`（`Result` を作るのに使った `Fake` とは別のもの。s07 の `Fetch` は `ViewIssue` を `Calls` に記録する）を `client` にして `New` し、`example` の `Result`（`Modes` は `org/app` が `sdd`）を取得完了として渡してバックログタブ（issue 140。`issue-140.json` の `labels` は空）を選んだ `Model` に `t` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` はちょうど 2 件で、1 件目は `Method` が `ViewIssue`、`Repo` が `org/app`、`Number` が 140、2 件目は `Method` が `AddLabel`、`Repo` が `org/app`、`Number` が 140、`Label` が `stage:todo` である。画面はキューのままである

#### Scenario: 主体が PR のカードでも対象は Issue になる
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` を `client` にして `New` し、`example` の `Result` を渡して今やるタブ（issue 108 + PR 131 の Card。主体は PR 131）を選んだ `Model` に `t` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` の 1 件目は `Method` が `ViewIssue` で `Number` が 108 であり、`Number` が 131 の要素は無い（`issue-108.json` は `stage:propose` 付きなので `ToggleTodo` は拒否し、`Calls` は 1 件だけである）

#### Scenario: stage:todo が付いている issue で t を押すと外れる
- **WHEN** `gh.NewFake("../action/testdata/todo")` を `client` にして `New` し、`Repo` `org/app`、`Number` 150、`Labels` が `stage:todo` の issue（`Comments` nil、`Result` の `Tab` は `TabInProgress`）だけの Card 1 枚と `Modes` に `org/app` の `sdd` を持つ `Result` を渡して進行中タブを選んだ `Model` に `t` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` は 2 件で、2 件目は `Method` が `RemoveLabel`、`Number` が 150、`Label` が `stage:todo` である

#### Scenario: カード詳細でも t が効く
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` を `client` にして `New` し、`example` の `Result` を渡してバックログタブの issue 140 のカード詳細を `Enter` で開いた `Model` に `t` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` は 2 件で、2 件目は `AddLabel` / `Number` 140 / `Label` `stage:todo` であり、画面はカード詳細のままで詳細の対象は issue 140 の Card のままである

#### Scenario: label 方式のリポジトリでは To Do を書く
- **WHEN** `gh.NewFake("../action/testdata/todo")` を `client` にして `New` した `Model` に、`org/board` の `Number` 160（`labels` は空）の Card と `Modes` に `org/board` の `label` を持つ `Result` を取得完了として渡し、`t` を与えて返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` は 2 件で、2 件目は `Method` が `AddLabel`、`Repo` が `org/board`、`Number` が 160、`Label` が `To Do` である

#### Scenario: 初回取得の前は書き込まずに理由を出す
- **WHEN** 同じ `Model` の `Options.Snapshot` に `org/board` の issue 160 の Card を入れ、初回取得を完了させる前に `t` を与えて `View` から ANSI エスケープを除いて読む
- **THEN** コマンドは返らず、`Fake.Calls` は空で、フッタに `org/board の運用方式が分かりません（stage:todo / To Do のラベルがありません）` が含まれる

#### Scenario: ラベル一覧の取得に失敗したリポジトリでは書き込まない
- **WHEN** `Cards` に `org/board` の issue 160 の Card を持ち、`Modes` が空で `Errors` に `ListLabels org/board: …` を 1 件持つ `Result` を取得完了として渡した `Model` に `t` を与える
- **THEN** コマンドは返らず、`Fake.Calls` は空で、フッタに `org/board の運用方式が分かりません` が含まれる

#### Scenario: 方式が分かるリポジトリは同じ画面でも書ける
- **WHEN** 上の `Result` の `Modes` に `org/app` の `sdd` を足し、`org/app` の issue 140 の Card を選んで `t` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` の 2 件目は `AddLabel` / `Repo` `org/app` / `Label` `stage:todo` である（判定できないリポジトリがあっても、他のリポジトリの書き込みは止めない）

#### Scenario: PR 単独のカードで t は何もしない
- **WHEN** `Issue` が nil で `PRs` が `docs` ラベルの open PR 60 の 1 件の Card を今やるタブに持つ `Model` に `t` を与える
- **THEN** コマンドは返らず、`Fake.Calls` は空で、画面はキューのままである

#### Scenario: PR 詳細と 0 行のタブで t は何もしない
- **WHEN** `example` の issue 108 のカード詳細から `Enter` で PR 131 の PR 詳細を開いた `Model` に `t` を与え、別に `example` の `Result` を渡して異常タブ（0 行）に切り替えた `Model` に `t` を与える
- **THEN** どちらもコマンドは返らず、`Fake.Calls` は空である

#### Scenario: 切り替え中の表示
- **WHEN** バックログの issue 140 を選んだ `Model` に `t` を与えた直後（返ったコマンドを実行する前）に `View` から ANSI エスケープを除いて読む
- **THEN** フッタに `org/app #140 の stage:todo を切り替え中` が含まれる
