## Purpose

`c` で issue と PR を close する経路を定める。画面が見せているものを対象に取り、
確認画面を挟んでから close し、結果をフッタのステータスに出す。

## ADDED Requirements

### Requirement: c は画面の対象を決めて close の確認画面を開く

`internal/ui` の `Model` は `c` を画面の状態ごとに MUST 次のとおり扱う。`Model` は対象のリポジトリ・番号・issue か PR か・
タイトル・ラベルを決め、表示名（issue は `<Repo> #<n>`、PR は `<Repo> PR#<n>`。`card-detail` の詳細ヘッダと同じ表記）を作り、
`gh` を呼ばずに close の確認画面へ移る。戻り先として `c` を押した画面を保持し、フッタ右のステータスを空にする
（`answer-question` の `a` と同じ。直前の書き込みの赤字を確認画面と戻り先に残さない）。

- キュー画面: 選択行の主体（`queue-screen`「行の主体は `Card.Result` を出した Issue または PR」）。issue でも PR でもよい。
  選択行が無いときは何もしない
- カード詳細画面: その Card の `Issue`（同じ画面の `m` は選択中の PR を対象にするが、`c` は `Issue` を対象にする。
  PR を閉じるときは `Enter` で PR 詳細へ入る）
- PR 詳細画面: 詳細の対象の PR
- close の確認画面・merge の確認画面・回答の確認画面・ヘルプ画面・URL 一覧画面: 何もしない

`c` のヒントは `?` のヘルプ（`help-screen`「ヘルプ画面は実装済みのキーだけを一覧する」）と、3 画面のフッタの左に
`c close` として出す。位置は `m merge` の次（`PRs` が空のカード詳細では `t todo` の次）で、文字列は `queue-screen`
「ヘッダはタブ名と件数と最終更新時刻、フッタは close を含むキーヒントとステータスを出す」と `card-detail`
「詳細の本文領域はスクロールし、ヘッダ領域は固定する」が定める。

書き込み中（`todo-toggle`「書き込み中は t と a を受け付けない」）は何もしない。対象は `c` を押した時点の値を保持し、
確認画面を読んでいる間に取得が完了して `Cards` が入れ替わっても差し替えない（`merge-pr` の確認画面と同じ）。
`c` の押下では `gh` を呼ばない（対象のラベルと状態を取り直さない。ラベルは最終取得時点の値で、直前に付いた `wip` は
確認画面に出ない。理由と申し送りは design.md）。

#### Scenario: キュー画面で c を押すと主体の確認画面が出る
- **WHEN** `example` の `Result` を取得完了として渡して今やるタブ（issue 108 + PR 131 の Card。主体は PR 131）を選んだ `Model` に
  `c` を与え、`View` から ANSI エスケープを除いて読む
- **THEN** 画面は close の確認画面で、`close の確認: org/app PR#131` の行がある

#### Scenario: 主体が issue の行でも c は効く
- **WHEN** 同じ準備を行い、主体が issue 140 であるバックログタブの行を選んだ `Model` に `c` を与え、
  `View` から ANSI エスケープを除いて読む
- **THEN** 画面は close の確認画面で、`close の確認: org/app #140` の行がある

#### Scenario: カード詳細は Issue、PR 詳細はその PR が対象になる
- **WHEN** `example` の issue 108 の Card のカード詳細を開いた `Model` に `c` を与え、別に同じカードから `Enter` で PR 詳細を開いた
  `Model` に `c` を与え、それぞれ `View` から ANSI エスケープを除いて読む
- **THEN** 前者は `close の確認: org/app #108`、後者は `close の確認: org/app PR#131` の行がある

#### Scenario: 0 行のタブとヘルプ画面で c は何もしない
- **WHEN** `example` の `Result` を渡して異常タブ（0 行）に切り替えた `Model` に `c` を与え、別に `?` でヘルプ画面を開いた `Model` に
  `c` を与える
- **THEN** どちらもコマンドは返らず、画面は変わらない

#### Scenario: c を押すと直前のステータスが消える
- **WHEN** close に失敗してフッタに赤字が出ている `Model` に `c` を与え、`View` から ANSI エスケープを除いて読む
- **THEN** 最終行に `の close に失敗:` は含まれない

#### Scenario: 確認画面で取得が完了しても対象は変わらない
- **WHEN** 今やるタブで `c` を与えて確認画面に移った `Model` に、Card が 0 件の `Result` の取得完了メッセージを渡し、
  `View` から ANSI エスケープを除いて読む
- **THEN** 画面は確認画面のままで、`close の確認: org/app PR#131` の行は変わらない

### Requirement: close の確認画面は対象と種別を出し、y で close して Esc で中止する

`Model` は画面の状態として close の確認画面を MUST 持ち、その `View` は端末の幅と高さの全体を使い、上から MUST 次の順で描く。

1. `close の確認: <表示名>  <タイトル>`
2. `種別: issue` または `種別: PR`
3. `labels: <ラベル名を空白区切り>`（ラベルが無ければ `labels: なし`）
4. `close-action`「`CheckClose` は却下として扱われる PR の注意を返す」の注意を 1 行ずつ `注意: <注意>` の形で

本文は出さない。行が残りの高さに収まらない分は切り（スクロールしない。`answer-question` と `merge-pr` の確認画面と同じ）、
フッタは常に出す。フッタの左は `y close  Esc 中止  q 終了`、右は `queue-screen` と同じステータスとする。

確認画面のキーは MUST 次のとおり。

- `y`: close のコマンド（Requirement「close の結果をステータスに出し、再取得しない」）を返し、戻り先の画面に戻る
- `Esc`: close せず戻り先の画面に戻り、フッタの右側に `close を中止しました` を出す
- `q` / `Ctrl+C`: 終了コマンドを返す
- 他のキー（`a` / `t` / `m` / `c` / `o` / `?` / `u` を含む）: 何もしない

`y` と `Esc` で戻る先がカード詳細画面または PR 詳細画面のときは、本文領域の寸法を作り直す（確認画面の間に端末の大きさが
変わっていることがある。`help-screen`「? か Esc で開いた画面に戻る」と同じ扱い）。
close の確認画面で取得完了のメッセージが届いたら、`Cards` と最終更新時刻は `queue-screen` の規則どおり更新し、確認中の対象は変えない。

#### Scenario: issue の確認画面
- **WHEN** `example` の `Result` を渡してバックログタブの issue 140 の行で `c` を与え、`View` から ANSI エスケープを除いて読む
- **THEN** `close の確認: org/app #140` とその行の issue タイトル、`種別: issue`、`labels: なし` の行がこの順で含まれ、
  フッタに `y close`、`Esc 中止`、`q 終了` が含まれ、`注意:` の行は無い

#### Scenario: propose PR の確認画面には却下の注意が出る
- **WHEN** 同じ準備で今やるタブの PR 131 の行（ラベルは `propose` と `question`）で `c` を与え、`View` から ANSI エスケープを除いて読む
- **THEN** `close の確認: org/app PR#131`、`種別: PR`、`labels: propose question`、
  `注意: merge せずに close した PR は却下として扱われ、issue に blocked-by: human が書き戻されます` の行が含まれ、
  フッタに `y close` が含まれる

#### Scenario: Esc で中止する
- **WHEN** Scenario「propose PR の確認画面には却下の注意が出る」の `Model` に `Esc` を与え、`View` から ANSI エスケープを除いて読む
- **THEN** 画面はキューで、フッタに `close を中止しました` が含まれ、`Fake.Calls` に `CloseIssue` と `ClosePR` は無い

#### Scenario: 詳細画面から入った確認画面は詳細画面に戻る
- **WHEN** `example` の issue 108 のカード詳細で `c` を押して確認画面に移った `Model` に `Esc` を与える
- **THEN** 画面はカード詳細で、詳細の対象は issue 108 の Card のままである

#### Scenario: 確認画面でリサイズしてから戻ると本文領域が新しい高さになる
- **WHEN** `Body` が `行01` から `行60` までの 60 段落である issue の Card のカード詳細を幅 100・高さ 20 で開き、`c` を与えてから
  幅 100・高さ 40 のサイズメッセージを渡し、`Esc` で戻って `View` を読む
- **THEN** 高さ 40 ぶんの本文が描かれている（高さ 20 のときより多くの段落が読める）

#### Scenario: 確認画面の他のキーは何もしない
- **WHEN** Scenario「propose PR の確認画面には却下の注意が出る」の `Model` に `a`、`t`、`m`、`c`、`o`、`?`、`u` を 1 つずつ与える
- **THEN** どのキーでもコマンドは返らず、画面は close の確認画面のままである

### Requirement: close の結果をステータスに出し、再取得しない

close のコマンドは `close-action` の close を、確認画面が保持しているリポジトリ・番号・種別で MUST 呼ぶ
（`ctx` は 30 秒のタイムアウト付き。`answer-question` / `todo-toggle` / `merge-pr` と同じ値）。`y` を押した時点の選択行や
詳細の対象からは導き直さない（確認画面を読んでいる間に自動更新が走ると `Cards` が入れ替わり選択行が動くため、
導き直すと別の対象を close しうる。`merge-pr` と同じ形）。その結果（表示名とエラー）を運ぶ `internal/ui` 内のメッセージを返す。
`Model` は `gh` を直接呼ばず、close は必ずコマンド（別ゴルーチン）で MUST 行う。

- close 中: フッタの右側に `<表示名> を close 中` を出す。この間の `t` / `a` / `m` / `c` は何もしないが、書き込みではない
  `o` / `?` / `u` / 画面移動のキーは受け付ける（`browse-open`「書き込み中でも `o` を受け付ける」と同じ）
- 成功: フッタの右側に `<表示名> を close しました` を出す
- 失敗: フッタの右側に `<表示名> の close に失敗: <エラー文字列>` を赤で出す

成否にかかわらず、`Cards` と最終更新時刻と詳細の対象を変えず、取得のコマンドを返さない（反映は `R` か自動更新に任せる。
`answer-question` / `todo-toggle` / `merge-pr` と同じ）。ラベルもコメントも書かない。このステータスは次の取得が始まったとき、
または次に `a` / `t` / `m` / `c` を押したときに消える。

#### Scenario: y で close されて成功がフッタに出る
- **WHEN** Scenario「propose PR の確認画面には却下の注意が出る」の `Model` に `y` を与え、返ったコマンドを実行して得たメッセージを
  `Update` に渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `Fake.Calls` に `Method` が `ClosePR`、`Repo` が `org/app`、`Number` が 131 の要素がちょうど 1 件あり、
  フッタに `org/app PR#131 を close しました` が含まれ、画面はキューで、`Cards` は変わらず今やるタブに PR 131 の行が残る

#### Scenario: issue の close は CloseIssue に行く
- **WHEN** Scenario「issue の確認画面」の `Model` に `y` を与え、返ったコマンドを実行する
- **THEN** `Fake.Calls` に `Method` が `CloseIssue`、`Number` が 140 の要素がちょうど 1 件あり、`ClosePR` は無い

#### Scenario: close 中は close 中の表示で t と a と m と c は効かない
- **WHEN** Scenario「issue の確認画面」の `Model` に `y` を与えた直後（close のコマンドを実行する前）に `View` を読み、
  続けて `t`、`a`、`m`、`c` を 1 つずつ与える
- **THEN** フッタに `org/app #140 を close 中` が含まれ、どのキーでもコマンドは返らない

#### Scenario: close 中でも o は受け付ける
- **WHEN** 同じ手順で `y` を与えて close のコマンドが返った直後に `o` を与える
- **THEN** コマンドが返る

#### Scenario: close の失敗は赤で出る
- **WHEN** `*gh.Fake` を埋め込んで `ClosePR` だけがエラー
  `gh pr close 131 -R org/app: exit 1: could not close pull request` を返す型を `client` にして
  Scenario「y で close されて成功がフッタに出る」と同じ手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** フッタに `org/app PR#131 の close に失敗:` と `could not close pull request` が含まれ、`Cards` は変わらない

#### Scenario: ラベルもコメントも書かない
- **WHEN** close の成功と失敗のそれぞれの手順の後に `Fake.Calls` を見る
- **THEN** `Method` が `AddLabel` / `RemoveLabel` / `CommentIssue` / `CommentPR` の要素は 1 件も無い
