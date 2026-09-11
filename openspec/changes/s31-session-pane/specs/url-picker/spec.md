## MODIFIED Requirements

### Requirement: 一覧は対象の本文・コメント・review thread の URL を出典付きで並べる

`Model` は対象から MUST 次の順で URL を集め、1 件を 出典 / リンクテキスト / URL の組として保持する。

- カード詳細画面 / PR 詳細画面で `u` を押したとき、右ペインのセッションが決まっていれば、その `https://claude.ai/code/session_<ID>` を先頭の 1 件（出典 `session`、リンクテキストは空）として置く（s31 `session-pane`「出すセッションは PR 本文と issue の routine コメントから決める」がセッションの決め方を定める）。キュー画面には右ペインが無いので置かない
- 対象が Issue のとき: `Issue.Body`（出典 `本文`）→ `Issue.Comments` の各 `Body`（出典 `コメント`）
- 対象が PR のとき: `PR.Body`（出典 `本文`）→ `PR.Comments` の各 `Body`（出典 `コメント`）→ `PR.ReviewThreads` の各 thread の各コメントの本文（出典 `thread`）

コメントと review thread は `Fetch` が入れた並びのまま辿る。`Comments` / `ReviewThreads` が `nil`（s20 以降は詳細の取得に失敗したことを意味する）ときは、その収集元を飛ばし、一覧には何も出さない。同じ URL が複数回出てきたら初出の 1 件だけを残し、後の出現は捨てる（リンクテキストも初出のものを使う）。セッションの URL を先頭に置くのはこの重複の規則のためで、PR 本文に同じ URL が書かれていても出典は `session` になる（人が「どれがセッションか」を探さずに開けるようにする）。

#### Scenario: 本文・コメント・thread の順に並び出典が付く

- **WHEN** 本文が `https://example.com/body`、コメントが 1 件で本文が `[CI](https://ci.example.com/build/42)`、review thread が 1 件でそのコメントの本文が `https://example.com/thread` の open PR の PR 詳細を開いた `Model` に `u` を与える
- **THEN** 一覧は 3 件で、順に（出典 `本文` / テキスト空 / `https://example.com/body`）、（出典 `コメント` / テキスト `CI` / `https://ci.example.com/build/42`）、（出典 `thread` / テキスト空 / `https://example.com/thread`）である

#### Scenario: 重複した URL は初出だけ残る

- **WHEN** 本文が `[設計メモ](https://example.com/design)`、コメントが 1 件で本文が `https://example.com/design も参照` の open PR の PR 詳細で `u` を与える
- **THEN** 一覧は 1 件で、出典は `本文`、リンクテキストは `設計メモ` である

#### Scenario: 取得に失敗したコメントは飛ばす

- **WHEN** 本文が `https://example.com/body` で `Comments` と `ReviewThreads` が nil（取得失敗）の open PR の PR 詳細で `u` を与える
- **THEN** 一覧は 1 件で、URL は `https://example.com/body` である

#### Scenario: セッションの URL が先頭に出る

- **WHEN** 本文が `https://claude.ai/code/session_01ABC で進めています。設計は https://example.com/design` の open PR の PR 詳細で `u` を与える
- **THEN** 一覧は 2 件で、1 件目は（出典 `session` / `https://claude.ai/code/session_01ABC`）、2 件目は（出典 `本文` / `https://example.com/design`）である

#### Scenario: コメントの session 行から作った URL も一覧に出る

- **WHEN** `<!-- routine -->` で始まり `session: session_01BBB` を持つコメントと、本文に URL を持たない Issue のカード詳細で `u` を与える
- **THEN** 一覧は 1 件で、出典は `session`、URL は `https://claude.ai/code/session_01BBB` である

### Requirement: 一覧画面は見出しと URL の行とフッタを出す

画面の状態が URL 一覧のとき、`View` は端末の幅と高さの全体を使い、上から MUST 次の順で描く。

1. 見出し `URL を開く`
2. 空行
3. URL の行。1 行は `<印>[<出典>] <リンクテキスト>  <URL>` で、印は選択中の行が `▶ `、他の行が空白 2 列。リンクテキストが空のときは `<印>[<出典>]  <URL>`（出典と URL の間は空白 2 列）とする。出典の語は `session` / `本文` / `コメント` / `thread`（`session` は s31 が足した。他は design.md 未決事項の既定値）
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

#### Scenario: session の出典が行に出る

- **WHEN** 本文が `https://claude.ai/code/session_01ABC` の open PR の PR 詳細で `u` を与えた `Model`（幅 80・高さ 24）の `View` から ANSI エスケープを除いて読む
- **THEN** `▶ [session]` と `https://claude.ai/code/session_01ABC` を含む行がある
