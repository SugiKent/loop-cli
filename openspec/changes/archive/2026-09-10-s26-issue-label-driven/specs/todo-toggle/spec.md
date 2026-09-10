## MODIFIED Requirements

### Requirement: t は画面の Card の Issue に対して確認なしで stage:todo を切り替える
`Model` の `Update` は `t` を画面の状態ごとに MUST 次のとおり扱う。対象は「画面に出ている Card の `Issue`」の 1 つの規則で決め、PR を対象にしない（承認ラベルは issue のラベルであり、`gh issue edit` で書く）。
- キュー画面: 選択行の Card の `Card.Issue`。選択行が無い、または `Issue` が nil（PR 単独のカード）なら何もしない（design.md 未決事項の既定値）
- カード詳細画面: 詳細の対象の `Card.Issue`（カード詳細は `Issue` が nil の Card では開かない。s09）
- PR 詳細画面: 何もしない
- 確認画面（s10）: 何もしない（s10「他のキー: 何もしない」のまま）
対象が決まったら、確認ダイアログを出さずに `action.ToggleTodo` を対象の `Repo` / `Number` と、`Options.Modes` から `Repo` で引いた方式で呼ぶコマンドを返す（mvp.md は `m` にだけ「確認ダイアログ必須」と書いており、`t` には無い。取り消しはもう一度 `t`。design.md 未決事項の既定値）。書くラベルは `model.TodoLabel(mode)` で、`sdd` は `stage:todo`、`label` は `To Do` である。`ctx` は 30 秒のタイムアウト付き（s10 の投稿と同じ値）。`Model` は `gh` を直接呼ばず、切り替えは必ずコマンド（別ゴルーチン）で行う。対象の `Issue.Labels` を判定に使わない（判定は `ToggleTodo` が読み直したラベルで行う）。
コマンドを返すとき、フッタの右側に `<Repo> #<Number> の <ラベル> を切り替え中` を出す（`<ラベル>` は `model.TodoLabel(mode)`）。表示名 `<Repo> #<Number>` は s09 の詳細ヘッダ・s10 の表示名と同じ表記である。

#### Scenario: バックログの issue で t を押すと stage:todo が付く
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` で新しく作った `Fake`（`Result` を作るのに使った `Fake` とは別のもの。s07 の `Fetch` は `ViewIssue` を `Calls` に記録する）を `client` にして `New` し、`example` の `Result` を取得完了として渡してバックログタブ（issue 140。`issue-140.json` の `labels` は空）を選んだ `Model` に `t` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` はちょうど 2 件で、1 件目は `Method` が `ViewIssue`、`Repo` が `org/app`、`Number` が 140、2 件目は `Method` が `AddLabel`、`Repo` が `org/app`、`Number` が 140、`Label` が `stage:todo` である。画面はキューのままである

#### Scenario: 主体が PR のカードでも対象は Issue になる
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` を `client` にして `New` し、`example` の `Result` を渡して今やるタブ（issue 108 + PR 131 の Card。主体は PR 131）を選んだ `Model` に `t` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` の 1 件目は `Method` が `ViewIssue` で `Number` が 108 であり、`Number` が 131 の要素は無い（`issue-108.json` は `stage:propose` 付きなので `ToggleTodo` は拒否し、`Calls` は 1 件だけである）

#### Scenario: stage:todo が付いている issue で t を押すと外れる
- **WHEN** `gh.NewFake("../action/testdata/todo")` を `client` にして `New` し、`Repo` `org/app`、`Number` 150、`Labels` が `stage:todo` の issue（`Comments` nil、`Result` の `Tab` は `TabInProgress`）だけの Card 1 枚を `Cards` に持つ `Result` を渡して進行中タブを選んだ `Model` に `t` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` は 2 件で、2 件目は `Method` が `RemoveLabel`、`Number` が 150、`Label` が `stage:todo` である

#### Scenario: カード詳細でも t が効く
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` を `client` にして `New` し、`example` の `Result` を渡してバックログタブの issue 140 のカード詳細を `Enter` で開いた `Model` に `t` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` は 2 件で、2 件目は `AddLabel` / `Number` 140 / `Label` `stage:todo` であり、画面はカード詳細のままで詳細の対象は issue 140 の Card のままである

#### Scenario: label 方式のリポジトリでは To Do を書く
- **WHEN** `Options.Modes` が `org/board` を `label` にし、`gh.NewFake("../action/testdata/todo")` を `client` にして `New` した `Model` に、`org/board` の `Number` 160（`labels` は空）の Card を持つ `Result` を渡して `t` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` は 2 件で、2 件目は `Method` が `AddLabel`、`Repo` が `org/board`、`Number` が 160、`Label` が `To Do` である

#### Scenario: スナップショットから描いた Card でも設定の方式で書く
- **WHEN** 同じ `Model` の `Options.Snapshot` に、`org/board` の issue 160 の Card を持つ前回の取得結果を入れ、初回取得を完了させる前に `t` を与えて返ったコマンドを実行する
- **THEN** `Fake.Calls` の 2 件目の `Label` は `To Do` である

#### Scenario: PR 単独のカードで t は何もしない
- **WHEN** `Issue` が nil で `PRs` が `docs` ラベルの open PR 60 の 1 件の Card を今やるタブに持つ `Model` に `t` を与える
- **THEN** コマンドは返らず、`Fake.Calls` は空で、画面はキューのままである

#### Scenario: PR 詳細と 0 行のタブで t は何もしない
- **WHEN** `example` の issue 108 のカード詳細から `Enter` で PR 131 の PR 詳細を開いた `Model` に `t` を与え、別に `example` の `Result` を渡して異常タブ（0 行）に切り替えた `Model` に `t` を与える
- **THEN** どちらもコマンドは返らず、`Fake.Calls` は空である

#### Scenario: 切り替え中の表示
- **WHEN** バックログの issue 140 を選んだ `Model` に `t` を与えた直後（返ったコマンドを実行する前）に `View` から ANSI エスケープを除いて読む
- **THEN** フッタに `org/app #140 の stage:todo を切り替え中` が含まれる

### Requirement: 切り替えの結果をステータスに出し、再取得しない
切り替えのコマンドは `action.ToggleTodo` の結果（表示名、書いたラベル名、`added`、エラー）を運ぶ `internal/ui` 内のメッセージを返す。`Model` はそのメッセージを受け取ったら、フッタの右側（s10 の回答のステータスと同じ場所。1 つのステータスを共有し、行を増やさない）に MUST 次を出す。`<ラベル>` は対象のリポジトリの方式の `model.TodoLabel(mode)` である。
- 付けた（`added` true、エラー nil）: `<Repo> #<Number> に <ラベル> を付けました`
- 外した（`added` false、エラー nil）: `<Repo> #<Number> から <ラベル> を外しました`
- エラー（`ToggleTodo` の拒否 `ErrOtherStage` / `ErrMultipleStages` / `ErrBlocked`、`ViewIssue` の失敗、`AddLabel` / `RemoveLabel` の失敗のいずれも）: `<Repo> #<Number> の <ラベル> を切り替えられません: <エラー文字列>` を赤で出す
切り替えの成否にかかわらず、`Cards` と最終更新時刻と詳細の対象を変えず、Card の `Issue.Labels` も書き換えず、取得のコマンドを返さない（書き込み後の対象 1 件再取得は D-002 のとおりだが s18 が担当する。それまでは s12 の `R` か s13 の自動更新で反映する。`ToggleTodo` は毎回ラベルを読み直すので、取り消しの `t` は画面の Card のラベルが古くても正しく外す）。反映後（`R` 等の再取得後）、承認ラベルだけが付いた issue は s05 の分類で進行中タブに移り、外した issue はバックログタブに戻る。このステータスは次の取得が始まったとき、または次に `t` か `a` を押したときに消える（s10 と同じ寿命）。

#### Scenario: 付けた結果がフッタに出る
- **WHEN** Requirement「t は画面の Card の Issue に対して確認なしで stage:todo を切り替える」の 1 つ目の Scenario の手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** フッタに `org/app #140 に stage:todo を付けました` が含まれ、バックログタブに issue 140 の行が残り、その行の Card の `Issue.Labels` は空のままで、`Fake.Calls` は 2 件だけである

#### Scenario: label 方式の結果は To Do で出る
- **WHEN** 同 Requirement の「label 方式のリポジトリでは To Do を書く」の Scenario の手順を行い、`View` を読む
- **THEN** フッタに `org/board #160 に To Do を付けました` が含まれる

#### Scenario: 外した結果がフッタに出る
- **WHEN** 同 Requirement の 3 つ目の Scenario（issue 150）の手順を行い、`View` を読む
- **THEN** フッタに `org/app #150 から stage:todo を外しました` が含まれる

#### Scenario: 拒否は赤で出て何も書き込まれない
- **WHEN** 同 Requirement の 2 つ目の Scenario の手順（`stage:propose` 付きの issue 108 に `t` を与える）を行い、`View` から ANSI エスケープを除いて読む
- **THEN** フッタに `org/app #108 の stage:todo を切り替えられません:` と `stage:propose` が含まれ、`Fake.Calls` に `AddLabel` / `RemoveLabel` は無く、`Cards` は変わらない

#### Scenario: gh の失敗は赤で出る
- **WHEN** `*gh.Fake` を埋め込んで `AddLabel` だけがエラー `gh issue edit 140 -R org/app --add-label stage:todo: exit 1: HTTP 403` を返す型を `client` にして 1 つ目の Scenario の手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** フッタに `org/app #140 の stage:todo を切り替えられません:` と `HTTP 403` が含まれ、`Cards` は変わらない

#### Scenario: 次の t でステータスが消える
- **WHEN** 1 つ目の Scenario の手順の後に、同じ `Model` にもう一度 `t` を与えて `View` を読む
- **THEN** フッタに `付けました` は含まれず `切り替え中` が含まれる
