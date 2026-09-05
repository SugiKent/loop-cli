# url-picker Specification

## Purpose
TBD - created by archiving change s22-url-picker. Update Purpose after archive.
## Requirements

### Requirement: 本文から URL とリンクテキストを出現順に取り出す

`internal/model` は、本文の文字列 1 つを受け取り、その中の URL を出現順に返す関数を MUST 公開する。1 件は URL とリンクテキストの組で、リンクテキストは Markdown リンクのときだけ入り、他は空文字列とする。取り出す書き方は次の 2 つだけである。

- Markdown リンク `[テキスト](URL)`: `URL` を取り、`テキスト` をリンクテキストにする。`URL` が `http://` / `https://` で始まらないものは取らない
- 裸の URL: `http://` または `https://` で始まり、空白・改行・`)` `>` `]` `）` `］` `〉` `」` のいずれかの直前までを 1 件とする

Markdown リンクとして取った範囲（`[` から対応する `)` まで）は、裸の URL の走査から MUST 外す（同じ URL を 2 回返さないため）。裸の URL は、末尾に付いた `.` `,` `:` `;` `。` `、` を MUST 削る。`#123` / `org/repo#123` / `@user` は URL ではないので取らない。同じ本文に同じ URL が複数回あれば、その回数ぶん返す（重複の除去は呼び出し側が行う）。

#### Scenario: 裸の URL と Markdown リンクを出現順に取る

- **WHEN** 本文 `設計は [設計メモ](https://example.com/design) を見る。\n参考: https://example.com/ref\n` を渡す
- **THEN** 2 件が順に返り、1 件目は URL が `https://example.com/design` でリンクテキストが `設計メモ`、2 件目は URL が `https://example.com/ref` でリンクテキストが空である

#### Scenario: 末尾の句読点と閉じ括弧を含めない

- **WHEN** 本文 `詳細は https://example.com/a。\n（https://example.com/b）\n<https://example.com/c>` を渡す（2 行目は全角括弧）
- **THEN** 3 件の URL は順に `https://example.com/a`、`https://example.com/b`、`https://example.com/c` である

#### Scenario: 参照記法と相対リンクは取らない

- **WHEN** 本文 `Closes #108\nissue org/app#140 も参照。\n[手順](./docs/mvp/mvp.md) と @user-1` を渡す
- **THEN** 返る件数は 0 である

#### Scenario: 同じ URL が 2 回あれば 2 件返る

- **WHEN** 本文 `https://example.com/a と https://example.com/a` を渡す
- **THEN** 返る件数は 2 で、どちらも URL が `https://example.com/a` である

### Requirement: u は画面の対象の URL 一覧画面を開く

`Model` は画面の状態として、s12 までの 5 つ（キュー / カード詳細 / PR 詳細 / 確認 / ヘルプ）に加えて URL 一覧 を MUST 持つ。`Update` は `u` を画面の状態ごとに MUST 次のとおり扱う。対象は `o`（s12 `browse-open`）と同じ規則で決まる。

- キュー画面: 選択行の主体（s08 `Subject`）。選択行が無ければ何もしない
- カード詳細画面: 詳細の対象の `Card.Issue`
- PR 詳細画面: 詳細の対象の選択中の PR
- 確認画面（s10）とヘルプ画面（s12）: 何もしない

対象から Requirement「一覧は対象の本文・コメント・review thread の URL を出典付きで並べる」の規則で URL を集め、1 件以上あれば、戻り先（`u` を押した画面）を保持して URL 一覧画面に移り、選択位置を先頭にする。1 件も無ければ画面を変えず、フッタの右側のステータスに `URL がありません` を通常の色（赤ではない）で出す（design.md 未決事項の既定値）。このステータスは s10 / s11 / s12 と同じ場所を共有し、同じ寿命（次の取得の開始、または次に `a` / `t` を押したときに消える）を持つ。

`u` は `gh` を呼ばず、URL を集めるのは画面の状態を変えるのと同じ処理の中で行う（コマンドを返さない）。書き込み中フラグ（s11）では止めない。

#### Scenario: PR 詳細で u を押すと一覧が開く

- **WHEN** 本文が `設計は [設計メモ](https://example.com/design) を見る` の open PR 131 だけを持つ手書きの Card を今やるタブに置いた `Model` で `Enter` を与えて PR 詳細を開き、`u` を与える
- **THEN** 画面は URL 一覧で、コマンドは返らず、一覧は 1 件（`https://example.com/design`）で選択位置は 0 である

#### Scenario: キュー画面の u は選択行の主体から集める

- **WHEN** 同じ手書きの Card を今やるタブに置いた `Model`（キュー画面）に `u` を与える
- **THEN** 画面は URL 一覧で、一覧は 1 件（`https://example.com/design`）である

#### Scenario: カード詳細の u は Issue から集める

- **WHEN** 本文が `https://example.com/issue` の Issue と、本文が `https://example.com/pr` の open PR を持つ手書きの Card のカード詳細を開いた `Model` に `u` を与える
- **THEN** 一覧は 1 件で、URL は `https://example.com/issue` である

#### Scenario: URL が無ければ画面を変えずステータスに出す

- **WHEN** `example` の `Result` を渡して今やるタブ（issue 108 + PR 131。どの本文にも URL が無い）を選んだ `Model` に `u` を与え、`View` から ANSI エスケープを除いて読む
- **THEN** 画面はキューのままでコマンドは返らず、フッタに `URL がありません` が含まれる

#### Scenario: 0 行のタブ・確認画面・ヘルプ画面で u は何もしない

- **WHEN** `example` の `Result` を渡して異常タブ（0 行）に切り替えた `Model` に `u` を与え、別に s10 の手順で blocked-by の確認画面に移った `Model` に `u` を与え、別に `?` でヘルプ画面を開いた `Model` に `u` を与える
- **THEN** どれもコマンドは返らず、画面は変わらず、フッタに `URL がありません` は含まれない

### Requirement: 一覧は対象の本文・コメント・review thread の URL を出典付きで並べる

`Model` は対象から MUST 次の順で URL を集め、1 件を 出典 / リンクテキスト / URL の組として保持する。

- 対象が Issue のとき: `Issue.Body`（出典 `本文`）→ `Issue.Comments` の各 `Body`（出典 `コメント`）
- 対象が PR のとき: `PR.Body`（出典 `本文`）→ `PR.Comments` の各 `Body`（出典 `コメント`）→ `PR.ReviewThreads` の各 thread の各コメントの本文（出典 `thread`）

コメントと review thread は `Fetch` が入れた並びのまま辿る。`Comments` / `ReviewThreads` が `nil`（s20 以降は詳細の取得に失敗したことを意味する）ときは、その収集元を飛ばし、一覧には何も出さない。同じ URL が複数回出てきたら初出の 1 件だけを残し、後の出現は捨てる（リンクテキストも初出のものを使う）。

#### Scenario: 本文・コメント・thread の順に並び出典が付く

- **WHEN** 本文が `https://example.com/body`、コメントが 1 件で本文が `[CI](https://ci.example.com/build/42)`、review thread が 1 件でそのコメントの本文が `https://example.com/thread` の open PR の PR 詳細を開いた `Model` に `u` を与える
- **THEN** 一覧は 3 件で、順に（出典 `本文` / テキスト空 / `https://example.com/body`）、（出典 `コメント` / テキスト `CI` / `https://ci.example.com/build/42`）、（出典 `thread` / テキスト空 / `https://example.com/thread`）である

#### Scenario: 重複した URL は初出だけ残る

- **WHEN** 本文が `[設計メモ](https://example.com/design)`、コメントが 1 件で本文が `https://example.com/design も参照` の open PR の PR 詳細で `u` を与える
- **THEN** 一覧は 1 件で、出典は `本文`、リンクテキストは `設計メモ` である

#### Scenario: 取得に失敗したコメントは飛ばす

- **WHEN** 本文が `https://example.com/body` で `Comments` と `ReviewThreads` が nil（取得失敗）の open PR の PR 詳細で `u` を与える
- **THEN** 一覧は 1 件で、URL は `https://example.com/body` である

### Requirement: 一覧画面は j / k で選び Enter で開き Esc で戻る

`Model` の `Update` は、画面の状態が URL 一覧のとき、キー入力を MUST 次のとおり扱う。

- `j` / `↓`: 選択を 1 つ下へ。末尾では動かない（循環しない）
- `k` / `↑`: 選択を 1 つ上へ。先頭では動かない
- `Enter`: 選択中の URL を開くコマンドを返す。画面は URL 一覧のままで、選択位置も変えない（続けて別の URL を開ける）
- `Esc`: 戻り先の画面に戻る。戻り先の状態（タブ・選択行・詳細の対象・選択中の PR・展開状態・スクロール位置）は開く前のまま。戻り先がカード詳細 / PR 詳細なら本文領域の内容と寸法を作り直す（s12 `help-screen` と同じ理由。一覧を出している間の端末サイズ変更が詳細に反映されないため）
- `q` / `Ctrl+C`: 終了コマンドを返す
- 他のキー（`u` / `a` / `t` / `o` / `?` / `R` / `x` / `g` / `p` / `1`〜`4` / `Tab` を含む）: 何もしない。`u` も含めるので、一覧を出したまま `u` を押しても戻り先は上書きされない（design.md 未決事項の既定値）

`Update` は URL 一覧画面の振り分けを `a` / `t` / `o` / `?` の判定より前に置く（後ろに置くと、一覧の裏にある対象に対して回答・ラベル書き込み・ブラウザ起動が起きるため）。一覧を出している間に取得完了（s08）・投稿結果（s10）・切り替え結果（s11）・ブラウザで開く結果（s12）のメッセージが届いたら、各 change の規則どおりに処理し、画面は URL 一覧のままにする。端末サイズの変更も受け取り、一覧の幅と高さに反映する。

#### Scenario: j と k で選択が動く

- **WHEN** URL が 3 件ある PR の PR 詳細で `u` を与えて一覧を開いた `Model` に、`j` を 3 回、`k` を 1 回与える
- **THEN** 選択位置は順に 1、2、2、1 になる

#### Scenario: Enter で選択中の URL が開く

- **WHEN** URL が 3 件ある一覧で `j` を 1 回与えてから `Enter` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` はちょうど 1 件で `Method` が `OpenURL`、`URL` が 2 件目の URL であり、画面は URL 一覧のままで選択位置は 1 のままである

#### Scenario: Esc で戻り先に戻る

- **WHEN** Issue と、本文に URL を 1 件持つ open PR とを持つ手書きの Card のカード詳細を開き、`Enter` で PR 詳細に移った `Model` に `u` を与え、次に `Esc` を与える
- **THEN** 画面は PR 詳細で、対象は同じ PR である

#### Scenario: 一覧画面で他のキーは何もしない

- **WHEN** URL 一覧を開いた `Model` に `u`、`a`、`t`、`o`、`?`、`R`、`x`、`g`、`p`、`2` を 1 つずつ与える
- **THEN** どのキーでもコマンドは返らず、画面は URL 一覧のまま、選択位置と戻り先は変わらず、`Fake.Calls` は空でスタブ `Editor` は呼ばれない

#### Scenario: 一覧画面でも q で終了する

- **WHEN** URL 一覧を開いた `Model` に `q` を与える
- **THEN** 終了コマンドが返る

### Requirement: 一覧画面は見出しと URL の行とフッタを出す

画面の状態が URL 一覧のとき、`View` は端末の幅と高さの全体を使い、上から MUST 次の順で描く。

1. 見出し `URL を開く`
2. 空行
3. URL の行。1 行は `<印>[<出典>] <リンクテキスト>  <URL>` で、印は選択中の行が `▶ `、他の行が空白 2 列。リンクテキストが空のときは `<印>[<出典>]  <URL>`（出典と URL の間は空白 2 列）とする。出典の語は `本文` / `コメント` / `thread`（design.md 未決事項の既定値）
4. 残りの高さは空行
5. 最終行はフッタ。左は `j/k 選択  Enter 開く  Esc 戻る  q 終了`、右は s08 と同じステータス（スピナー / エラー / 書き込みステータス）

各行は端末の幅で切り詰める。URL の件数が表示できる行数（端末の高さ − 見出し 1 行 − 空行 1 行 − フッタ 1 行）を超えるときは、選択中の行が必ず見えるように表示の開始位置をずらす（選択が表示範囲より上なら開始位置を選択に合わせ、下なら開始位置を「選択 − 表示行数 + 1」にする）。一覧はスクロール位置を独立に持たず、選択位置だけを持つ。

#### Scenario: 一覧の行と見出しとフッタ

- **WHEN** 本文が `[設計メモ](https://example.com/design)` とコメント 1 件（本文が `https://ci.example.com/build/42`）の open PR の PR 詳細で `u` を与えた `Model`（幅 80・高さ 24）の `View` から ANSI エスケープを除いて読む
- **THEN** 1 行目は `URL を開く` で、`▶ [本文] 設計メモ` と `https://example.com/design` を含む行、`[コメント]` と `https://ci.example.com/build/42` を含み `▶` を含まない行がこの順で含まれ、最終行に `j/k 選択`、`Enter 開く`、`Esc 戻る`、`q 終了` が含まれる

#### Scenario: 選択が下に動くと印も動く

- **WHEN** 上の `Model` に `j` を与えて `View` から ANSI エスケープを除いて読む
- **THEN** `▶` を含む行は `https://ci.example.com/build/42` の行で、`https://example.com/design` の行は `▶` を含まない

#### Scenario: 件数が高さを超えると選択に追従して表示がずれる

- **WHEN** URL を 30 件（`https://example.com/0` から `https://example.com/29`）持つ open PR の PR 詳細で `u` を与えた `Model`（幅 80・高さ 10。表示できる行数は 7）で、`View` を読んでから `j` を 10 回与えて `View` を読む
- **THEN** 1 回目は `https://example.com/0` を含み `https://example.com/7` を含まない。2 回目は `https://example.com/10` を `▶` の行として含み、`https://example.com/0` を含まない

### Requirement: URL を開けなかったときはステータスに赤で出す

URL を開くコマンドは、s03 の `GHClient.OpenURL(ctx, url)` を呼び、URL とエラーを運ぶ `internal/ui` 内のメッセージを返す。`ctx` は 30 秒のタイムアウト付き（s10 / s11 / s12 と同じ値）。`Model` はそのメッセージを MUST 次のとおり扱う。

- エラーが nil のとき: 何も出さない（ブラウザが開いたこと自体が結果である。s12 `browse-open` と同じ扱い）
- エラーが非 nil のとき: フッタの右側に `<URL> を開けません: <エラー文字列>` を赤で出す。この場所は s10 / s11 / s12 のステータスと同じで、1 つのステータスを共有し、行を増やさない

成否にかかわらず `Cards` と最終更新時刻と詳細の対象と一覧の内容を変えず、取得のコマンドを返さない。このステータスは次の取得が始まったとき、または次に `a` / `t` を押したときに消える。

#### Scenario: OpenURL の失敗は赤で出る

- **WHEN** `*gh.Fake` を埋め込んで `OpenURL` だけがエラー `open https://example.com/design: exit 1: no browser` を返す型を `client` にして、URL 一覧で `Enter` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡し、`View` から ANSI エスケープを除いて読む
- **THEN** フッタに `https://example.com/design を開けません:` と `no browser` が含まれ、画面は URL 一覧のままで一覧の内容と選択位置は変わらない

#### Scenario: 成功時は何も出ない

- **WHEN** Requirement「一覧画面は j / k で選び Enter で開き Esc で戻る」の Scenario「Enter で選択中の URL が開く」の手順の後に `View` から ANSI エスケープを除いて読む
- **THEN** フッタに `開けません` は含まれない
