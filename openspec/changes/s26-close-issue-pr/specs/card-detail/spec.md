## MODIFIED Requirements

### Requirement: Enter でカード詳細を開き、Esc で 1 つ前の画面に戻る
`internal/ui` の `Model` は画面の状態として キュー / カード詳細 / PR 詳細 / 回答の確認 / merge の確認 / close の確認 / ヘルプ / URL 一覧 の 8 つを MUST 持ち、初期状態はキューである。回答の確認画面は s10 `answer-question`「確認画面では投稿・編集に戻る・中止を選ぶ」が、merge の確認画面は s14 `merge-pr`「merge の確認画面は判断材料を出し、y で merge して Esc で中止する」が、ヘルプ画面は s12 `help-screen`「? はヘルプ画面を開き、? か Esc で開いた画面に戻る」が、URL 一覧画面は s22 `url-picker`「u は画面の対象の URL 一覧画面を開く」が、close の確認画面は s26 `close-issue-pr`「close の確認画面は対象と種別を出し、y で close して Esc で中止する」が定める。`Update` はキー入力を画面の状態ごとに次のとおり扱う。
- キュー画面で `Enter`: 選択行があれば、その行の `model.Card` のコピーを詳細の対象として保持し、カード詳細画面に移る。`Card.Issue` が nil（PR 単独のカード）なら、カード詳細を挟まず `PRs[0]` の PR 詳細画面に直接移る（Issue の情報が無く、カード詳細に出すものが PR 一覧 1 件しか無いため。design.md 未決事項の既定値）。選択行が無い（タブが 0 行）なら何もしない。開いたときに routine コメントの展開状態は折りたたみに戻り、スクロール位置は先頭に戻る
- カード詳細画面で `Esc`: キュー画面に戻る。キュー画面の現在のタブと選択行は開く前のまま（戻るキーは mvp.md に無く、`Esc` は design.md 未決事項の既定値）
- PR 詳細画面で `Esc`: カード詳細画面から入ったならカード詳細に戻り、キュー画面から直接入った（`Issue` が nil）ならキュー画面に戻る
- `q` / `Ctrl+C`: どの画面でも終了コマンドを返す（s01 `tui-entrypoint`「q で終了する」）
- カード詳細画面と PR 詳細画面では、キュー画面のキー `1`〜`4`（タブ切替）と `R`（全件再取得。s12 `manual-refresh`）は何もしない。`p` はキュー画面でも詳細画面でも何もしない（s21 `queue-screen` が 2 ペインを常設にして切替を廃止した）。`j` / `k` / `↑` / `↓` はスクロール（Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」）に使う。`Tab` はカード詳細画面でだけ PR の選択（Requirement「紐づく PR 一覧は段階順に 1 行ずつ出し、選択中の PR に印を付ける」）に使い、PR 詳細画面では何もしない。詳細画面での `j` / `k` / `↑` / `↓` と `Tab` の意味は mvp.md に無く、design.md 未決事項の既定値である
- カード詳細画面と PR 詳細画面で `?`: 戻り先を保持してヘルプ画面に移る。`o` はその画面の対象をブラウザで開く（s12 `browse-open`）。`u` は戻り先を保持して URL 一覧画面に移る（s22 `url-picker`）。`m` は対象の PR の状態を取り直し、結果が届いたときに merge の確認画面に移る（s14 `merge-pr`）。`c` は戻り先を保持して close の確認画面に移り、対象はカード詳細では `Issue`、PR 詳細ではその PR である（s26 `close-issue-pr`）
- 詳細を開いている間に取得完了のメッセージ（s08 の `fetchedMsg`）が届いたら、キュー画面の `Cards` と最終更新時刻は s08 の規則どおり更新するが、詳細の対象として保持している Card のコピーは差し替えない。`Esc` でキューに戻った後の `Enter` で新しい Card を開く（自動更新中の追従は s13 が決める）

詳細を開いても `gh` を呼ばない。表示するのは s07 の `Fetch` が Card に入れた値だけである（design.md 未決事項「詳細を開いたときの追加取得」。mvp.md の `Enter` 内部処理 `gh issue view` / `gh pr view --json` からの逸脱で、design.md「追加取得はしない」に理由と申し送りがある）。human-turn-signals.md「merge 可否は表示時に取り直す」は、詳細を開いたときではなく `m` を押したときの取り直し（s14 `merge-pr`）で満たす。

#### Scenario: Enter でカード詳細が開き Esc で戻る
- **WHEN** `example` の `Result`（issue 108 + PR 131 の Card が今やるタブ）を渡した `Model` に `Enter` を与え、次に `Esc` を与える
- **THEN** `Enter` の後の画面はカード詳細で、詳細の対象は issue 108 の Card である。`Esc` の後の画面はキューで、現在のタブは今やる、選択行の添字は 0 のままである

#### Scenario: PR 単独のカードは PR 詳細が直接開く
- **WHEN** `Issue` が nil で `PRs` が `docs` ラベルの open PR 60 の 1 件の Card を今やるタブに持つ `Model` に `Enter` を与え、次に `Esc` を与える
- **THEN** `Enter` の後の画面は PR 詳細で対象は PR 60、`Esc` の後の画面はキューである

#### Scenario: 0 行のタブで Enter は何もしない
- **WHEN** `example` の `Result` を渡して異常タブ（0 行）に切り替えた `Model` に `Enter` を与える
- **THEN** 画面はキューのままで、コマンドは返らない

#### Scenario: 詳細画面ではタブ切替キーと R が効かない
- **WHEN** カード詳細を開いた `Model` に `2`、`4`、`p`、`R` を 1 つずつ与える
- **THEN** 画面はカード詳細のままで、キュー画面の現在のタブは今やるのままで、どのキーでもコマンドは返らない

#### Scenario: 詳細を開いている間の取得完了は対象の Card を差し替えない
- **WHEN** `example` の `Result` で issue 108 のカード詳細を開いた `Model` に、issue 108 の Card を含まない `Result` を取得完了として渡す
- **THEN** 画面はカード詳細のままで、詳細の対象は issue 108 の Card のままである。`Esc` でキューに戻ると今やるタブの行は新しい `Result` のものになっている

#### Scenario: どの画面でも q で終了する
- **WHEN** カード詳細を開いた `Model` に `q` を与える
- **THEN** 終了コマンドが返る
