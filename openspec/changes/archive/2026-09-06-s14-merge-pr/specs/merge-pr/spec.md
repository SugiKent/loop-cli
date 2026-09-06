## ADDED Requirements

### Requirement: m は画面の対象 PR を決めて GitHub から状態を取り直す
`internal/ui` の `Model` の `Update` は `m` を画面の状態ごとに MUST 次のとおり扱う。対象の PR（リポジトリと番号）と表示名（`<Repo> PR#<n>`。s09 の詳細ヘッダと同じ表記）を決め、`client.ViewPR(ctx, repo, number)` と `client.ViewPRMergeState(ctx, repo, number)` を並行に呼んで両方の完了を待ち、その結果を運ぶ `internal/ui` 内のメッセージを返すコマンドを返す。`ctx` は 30 秒のタイムアウト付き（s10 / s11 / s12 と同じ値）。
- キュー画面: 選択行の主体（s08 `Subject`）が PR ならその PR。主体が issue の行、または選択行が無いときは何もしない
- カード詳細画面: 選択中の PR。`PRs` が空なら何もしない
- PR 詳細画面: 詳細の対象の PR
- merge の確認画面・s10 の回答の確認画面・ヘルプ画面・URL 一覧画面: 何もしない

`Model` は `gh` を直接呼ばず、取り直しは必ずコマンド（別ゴルーチン）で行う。取り直しの間はフッタの右側に `<表示名> の状態を取得中` を出し、この間は `t` / `a` / `m` を受け付けない（s11 `todo-toggle`「書き込み中は t と a を受け付けない」と同じ扱い。merge の手続きが始まってからは重ねて始めない。design.md 未決事項の既定値）。
画面が持つ `Card` の値（`Labels` / `Body` / `IsDraft` / `MergeState`）は取得時点のもので、merge の判断には使わない（human-turn-signals.md「merge 可否は表示時に取り直す」。s09「詳細を開いても `gh` を呼ばない」の逸脱をここで回収する）。

#### Scenario: キュー画面で m を押すと主体の PR の状態を取りに行く
- **WHEN** `gh.NewFake("testdata/merge")` の `Fake`（`Result` を作るのに使った `Fake` とは別のもの）を `client` にして `New` し、`example` の `Result` を取得完了として渡して今やるタブ（issue 108 + PR 131 の Card。主体は PR 131）を選んだ `Model` に `m` を与え、`View` から ANSI エスケープを除いて読む
- **THEN** コマンドが返り、フッタに `org/app PR#131 の状態を取得中` が含まれ、画面はキューのままである

#### Scenario: 主体が issue の行では m は何もしない
- **WHEN** 同じ準備を行い、主体が issue 140 であるバックログタブの行を選んだ `Model` に `m` を与える
- **THEN** コマンドは返らず、画面はキューのままで、フッタに `の状態を取得中` は含まれない

#### Scenario: カード詳細では選択中の PR、PR 詳細ではその PR が対象になる
- **WHEN** `example` の issue 108 の Card のカード詳細を開いた `Model` に `m` を与えて返ったコマンドを実行し、別に同じカードから `Enter` で PR 詳細を開いた `Model` に `m` を与えて返ったコマンドを実行する
- **THEN** どちらも対象は `org/app` の PR 131 で、続けて `Update` にメッセージを渡すとどちらも merge の確認画面に移る

#### Scenario: 取り直し中は t と a と m を受け付けない
- **WHEN** 今やるタブで `m` を与えた直後（返ったコマンドを実行する前）に `t`、`a`、`m` を 1 つずつ与える
- **THEN** どのキーでもコマンドは返らず、画面はキューのままである

#### Scenario: 0 行のタブとヘルプ画面で m は何もしない
- **WHEN** `example` の `Result` を渡して異常タブ（0 行）に切り替えた `Model` に `m` を与え、別に `?` でヘルプ画面を開いた `Model` に `m` を与える
- **THEN** どちらもコマンドは返らず、画面は変わらない

### Requirement: 取り直しに成功したら確認画面へ移り、失敗したら赤で出す
`Model` は取り直しの結果のメッセージを MUST 次の順で判定して扱う。どの分岐でも `Cards` と最終更新時刻と詳細の対象は変えない。
1. 結果が届いた時点の画面が `m` を押した画面と違うとき（取り直しの間に `?` / `u` / `Enter` / `Esc` / `g` で画面が変わったとき）は、取り直しの成否を問わず、確認画面に移らず、フッタの右側に `<表示名> の merge を中止しました（画面が変わりました）` を出す。ヘルプ画面と URL 一覧画面を確認画面で踏み潰さないためである（s12 `help-screen`「画面はヘルプのままにする」、s22 `url-picker`）
2. 画面が同じで、`ViewPR` と `ViewPRMergeState` のどちらか一方でもエラーなら、確認画面に移らず、フッタの右側に `<表示名> の状態を取得できません: <エラー文字列>` を赤で出す（両方エラーなら `ViewPR` のエラーを出す）
3. 画面が同じで両方成功なら、`Model` は `PRDetail`（`Title` / `Body` / `Labels` / `IsDraft`）と `PRMergeState` から `model.PR` を組み立て、`State` には画面の Card が持つその PR の `State` を写し（`gh pr view` の `--json` に `state` が無い。s03 `gh-client`）、その `model.PR` と `action.CheckMerge` の結果、対象のリポジトリ名・番号・表示名・merge 方式を保持して merge の確認画面（Requirement「merge の確認画面は判断材料を出し、y で merge して Esc で中止する」）に移る。戻り先には `m` を押した画面を保持する

どの分岐でも取り直し中のステータスは消え、`t` / `a` / `m` を再び受け付ける。

#### Scenario: 取り直しに成功すると確認画面が出る
- **WHEN** `gh.NewFake("testdata/merge")`（`pr-131.json` は `labels` が `propose` のみ、`isDraft` false、画面の Card の PR 131 は search 由来で `State` が `OPEN`、`body` の 1 行目が `未確定の判断: 0 件`、`mergeable` が `MERGEABLE`、`mergeStateStatus` が `CLEAN`、`statusCheckRollup` が `CheckRun` の `test` `SUCCESS` 1 件）を `client` にして今やるタブで `m` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡し、`View` から ANSI エスケープを除いて読む
- **THEN** 画面は merge の確認画面で、`merge の確認: org/app PR#131` の行があり、`注意:` で始まる行は無い（画面の Card は `question` ラベルと `PENDING` の checks を持つが、取り直した値で判定するため警告が出ない）

#### Scenario: 取り直しの失敗は赤で出て確認画面に移らない
- **WHEN** `gh.NewFake("")`（fixture のディレクトリが無く `pr-131.json` を読めない）を `client` にして今やるタブで `m` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡し、`View` から ANSI エスケープを除いて読む
- **THEN** 画面はキューのままで、フッタに `org/app PR#131 の状態を取得できません:` と `pr-131.json` が含まれ、`Fake.Calls` に `MergePR` は無い

### Requirement: merge の確認画面は判断材料を出し、y で merge して Esc で中止する
`Model` は画面の状態として merge の確認画面を MUST 持ち、その `View` は端末の幅と高さの全体を使い、上から MUST 次の順で描く。
1. `merge の確認: <表示名>  <Title>`（`Title` は取り直した `PRDetail` の値）
2. `方式: <method>`（Requirement「merge 方式はリポジトリ名から引く」で決まる値）
3. `labels: <ラベル名を空白区切り>`（ラベルが無ければ `labels: なし`）
4. `mergeable: <Mergeable> <MergeStateStatus>`（s09 の PR 詳細と同じ表記）
5. `checks: 緑` または `checks: 緑以外`（`classify.ChecksGreen` の結果）
6. `action.CheckMerge` の `blocked` を 1 行ずつ `merge できません: <理由>` の形で（赤）
7. `action.CheckMerge` の `warnings` を 1 行ずつ `注意: <警告>` の形で
8. 区切り線
9. 取り直した PR の本文

行が残りの高さに収まらない分は切り（スクロールしない。s10 の確認画面と同じ）、フッタは常に出す。フッタの左は `blocked` が空なら `y merge  Esc 中止  q 終了`、空でなければ `Esc 中止  q 終了`。右は s08 と同じステータスとする。
確認画面のキーは MUST 次のとおり。
- `y`: `blocked` が空のときだけ、merge のコマンド（Requirement「merge の結果をステータスに出し、再取得しない」）を返し、戻り先の画面に戻る。`blocked` が空でなければ何もしない
- `Esc`: merge せず戻り先の画面に戻り、フッタの右側に `merge を中止しました` を出す
- `q` / `Ctrl+C`: 終了コマンドを返す
- 他のキー（`t` / `a` / `m` / `o` / `?` / `u` を含む）: 何もしない
merge の確認画面で取得完了のメッセージ（s08 `fetchedMsg`）が届いたら、`Cards` と最終更新時刻は s08 の規則どおり更新し、確認中の対象と取り直した値は変えない（s10 の確認画面と同じ）。

#### Scenario: 警告のある確認画面
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")`（`pr-131.json` は `labels` が `propose` と `question`、`isDraft` false、`body` の 1 行目が `issue #108 の提案。`、`mergeable` が `UNKNOWN`、`mergeStateStatus` が `BLOCKED`、`statusCheckRollup` に `PENDING` の `ci/legacy` を含む）を `client` にして今やるタブで `m` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `merge の確認: org/app PR#131` とその行の PR タイトル、`方式: squash`、`labels: propose question`、`mergeable: UNKNOWN BLOCKED`、`checks: 緑以外`、`注意: question ラベルが付いています`、`注意: checks が緑ではありません`、本文の `issue #108 の提案` がこの順で含まれ、フッタに `y merge`、`Esc 中止`、`q 終了` が含まれる

#### Scenario: draft の PR には y が出ない
- **WHEN** `gh.NewFake("testdata/merge-draft")`（`pr-131.json` は `isDraft` が true）を `client` にして同じ手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** `merge できません: draft の PR です` の行があり、フッタに `Esc 中止` と `q 終了` はあって `y merge` は無い

#### Scenario: draft の確認画面で y は何もしない
- **WHEN** 同じ確認画面の `Model` に `y` を与える
- **THEN** コマンドは返らず、画面は確認画面のままで、`Fake.Calls` に `MergePR` は無い

#### Scenario: Esc で中止する
- **WHEN** Scenario「警告のある確認画面」の `Model` に `Esc` を与え、`View` から ANSI エスケープを除いて読む
- **THEN** 画面はキューで、フッタに `merge を中止しました` が含まれ、`Fake.Calls` に `MergePR` は無い

#### Scenario: 詳細画面から入った確認画面は詳細画面に戻る
- **WHEN** `example` の issue 108 のカード詳細で `m` を押して確認画面に移った `Model` に `Esc` を与える
- **THEN** 画面はカード詳細で、詳細の対象は issue 108 の Card のままである

#### Scenario: 確認画面で取得が完了して選択行が動いても merge の対象は変わらない
- **WHEN** Scenario「取り直しに成功すると確認画面が出る」の後の `Model` に、Card が 0 件の `Result` の取得完了メッセージ（選択行が丸められる）を渡してから `y` を与え、返ったコマンドを実行する
- **THEN** `Fake.Calls` の `MergePR` は `Repo` が `org/app`、`Number` が 131 である

#### Scenario: 取り直し中に画面を変えると確認画面に移らない
- **WHEN** 今やるタブで `m` を与えた直後に `?` でヘルプ画面に移り、取り直しの結果のメッセージを `Update` に渡し、`View` から ANSI エスケープを除いて読む
- **THEN** 画面はヘルプのままで、フッタに `org/app PR#131 の merge を中止しました（画面が変わりました）` が含まれる

#### Scenario: 確認画面で取得が完了しても対象は変わらない
- **WHEN** Scenario「警告のある確認画面」の `Model` に、issue 108 の Card を含む `Result` の取得完了メッセージを渡し、`View` から ANSI エスケープを除いて読む
- **THEN** 画面は確認画面のままで、`merge の確認: org/app PR#131` の行と警告の行は変わらない

### Requirement: merge の結果をステータスに出し、再取得しない
merge のコマンドは `action.Merge` を、確認画面が保持しているリポジトリ名・番号・merge 方式で MUST 呼ぶ（`ctx` は 30 秒のタイムアウト付き）。`y` を押した時点の選択行や詳細の対象からは導かない（s10 が `a` の押下時に `action.Target` を捕まえるのと同じ形。確認画面を読んでいる間に s13 の自動更新が走ると `Cards` が入れ替わり選択行が動くため、導き直すと別の PR を merge しうる）、その結果（表示名とエラー）を運ぶ `internal/ui` 内のメッセージを返す。`Model` は `gh` を直接呼ばず、merge は必ずコマンド（別ゴルーチン）で MUST 行う。
- merge 中: フッタの右側に `<表示名> を merge 中` を出す。この間の `t` / `a` / `m` は何もしないが、書き込みではない `o` / `?` / `u` / 画面移動のキーは受け付ける（s12 `browse-open`「書き込み中でも `o` を受け付ける」と同じ）
- 成功: フッタの右側に `<表示名> を merge しました` を出す
- 失敗: フッタの右側に `<表示名> の merge に失敗: <エラー文字列>` を赤で出す
成否にかかわらず、`Cards` と最終更新時刻と詳細の対象を変えず、取得のコマンドを返さない（書き込み後の対象 1 件再取得は D-002 のとおりだが s18 が担当する。それまでは s12 の `R` か s13 の自動更新で反映する）。ラベルもコメントも書かない。このステータスは次の取得が始まったとき、または次に `a` / `t` / `m` を押したときに消える（s10 / s11 のステータスと同じ場所・同じ寿命）。

#### Scenario: y で merge されて成功がフッタに出る
- **WHEN** Scenario「取り直しに成功すると確認画面が出る」の後の `Model` に `y` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `Fake.Calls` に `Method` が `MergePR`、`Repo` が `org/app`、`Number` が 131、`MergeMethod` が `squash` の要素がちょうど 1 件あり、フッタに `org/app PR#131 を merge しました` が含まれ、画面はキューで、`Cards` は変わらず今やるタブに PR 131 の行が残る

#### Scenario: merge 中は merge 中の表示で t と a と m は効かない
- **WHEN** 同じ手順で `y` を与えた直後（merge のコマンドを実行する前）に `View` を読み、続けて `t`、`a`、`m` を 1 つずつ与える
- **THEN** フッタに `org/app PR#131 を merge 中` が含まれ、どのキーでもコマンドは返らない

#### Scenario: merge の失敗は赤で出る
- **WHEN** `*gh.Fake` を埋め込んで `MergePR` だけがエラー `gh pr merge 131 -R org/app --squash: exit 1: Pull request is not mergeable` を返す型を `client` にして同じ手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** フッタに `org/app PR#131 の merge に失敗:` と `Pull request is not mergeable` が含まれ、`Cards` は変わらない

#### Scenario: ラベルもコメントも書かない
- **WHEN** merge の成功と失敗のそれぞれの手順の後に `Fake.Calls` を見る
- **THEN** `Method` が `AddLabel` / `RemoveLabel` / `CommentIssue` / `CommentPR` の要素は 1 件も無い

### Requirement: merge 方式はリポジトリ名から引く
`internal/ui` の `Options` は MUST リポジトリ名から merge 方式を引く対応表（`map[string]string`。キーは `owner/name`、値は `squash` / `merge` / `rebase`）を持ち、`Model` は対象の PR の `Repo` でこれを引いて確認画面の `方式:` と `action.Merge` の `method` に使う。表に無いリポジトリ、または `Options` に対応表が無いときは `squash` を使う（design.md 未決事項の既定値。`gh pr merge` の既定は対話式なので方式を空で渡さない）。
`internal/ui` は `internal/config` を import しない（s03 design「`internal/gh` を `internal/config` に依存させない」と同じ方針）。対応表を作るのは `cmd/loop-cli` である（s01 `tui-entrypoint`）。

#### Scenario: リポジトリ別の方式が確認画面と MergePR に出る
- **WHEN** 対応表 `{"org/app": "rebase"}` を `Options` に渡して Scenario「取り直しに成功すると確認画面が出る」と同じ手順を行い、`View` を読んでから `y` を与えて返ったコマンドを実行する
- **THEN** 確認画面に `方式: rebase` が含まれ、`Fake.Calls` の `MergePR` の `MergeMethod` は `rebase` である

#### Scenario: 表に無いリポジトリは squash になる
- **WHEN** 対応表を空にして同じ手順を行う
- **THEN** 確認画面に `方式: squash` が含まれ、`Fake.Calls` の `MergePR` の `MergeMethod` は `squash` である
