# new-issue-action Specification

## Purpose
TUI から新しい issue を起こすときの書き込みの直前を担う。下書きをタイトルと本文に分ける規則と、`gh` を呼ぶ前の検査（空・routine マーカー）を 1 か所に置き、`internal/ui` が判定を持たずに済むようにする。

## Requirements

### Requirement: CreateIssue はラベルを付けずに issue を 1 つ作り URL を返す
`internal/action` は関数 `CreateIssue(ctx context.Context, client gh.GHClient, repo string, title string, body string) (string, error)` を MUST 提供する。検査を通ったときだけ `client.CreateIssue(ctx, repo, title, body)` を 1 回呼び、その戻り値（作成された issue の URL とエラー）をそのまま返す。
`CreateIssue` はラベルを 1 つも書かない（human-turn-signals.md 不変条件 6「Issue 作成は承認ではない。`n` で作った issue に段階ラベルは付けない」と不変条件 2「TUI が書くラベルは `stage:todo` と `stage:propose` に限る」）。`AddLabel` / `RemoveLabel` / `CommentIssue` / `CommentPR` / `MergePR` / `ReplyReviewThread` を呼ばず、assignee も指定しない。作った issue に着手させるかどうかは、人が続けて `t` を押して決める。

#### Scenario: 作成に成功すると URL が返る
- **WHEN** `gh.NewFake` の `Fake` を `client` にして `CreateIssue(ctx, client, "org/app", "タイトル", "本文")` を呼ぶ
- **THEN** `https://github.com/org/app/issues/0` と nil が返り、`Fake.Calls` はちょうど 1 件で、`Method` が `CreateIssue`、`Repo` が `org/app`、`Title` が `タイトル`、`Body` が `本文` である

#### Scenario: ラベルの呼び出しは無い
- **WHEN** `CreateIssue` を 1 回呼んだ後に `Fake.Calls` を見る
- **THEN** `Method` が `AddLabel` / `RemoveLabel` / `CommentIssue` / `CommentPR` / `MergePR` / `ReplyReviewThread` の要素は 1 件も無い

#### Scenario: gh の失敗はそのまま返る
- **WHEN** `*gh.Fake` を埋め込んで `CreateIssue` だけがエラー `gh issue create -R org/app: exit 1: HTTP 403` を返す型を `client` にして `CreateIssue(ctx, client, "org/app", "タイトル", "本文")` を呼ぶ
- **THEN** 返る URL は空文字列で、エラー文字列に `HTTP 403` を含む

### Requirement: タイトルか本文が空、または routine マーカーを含む下書きでは作らない
`CreateIssue` は `client` を呼ぶ前に MUST 次の順で検査し、当たったところでエラーを返して `client` のメソッドを 1 つも呼ばない。
1. `title` の前後の空白を除いた結果が空: エラー値 `ErrEmptyTitle`
2. `body` の前後の空白を除いた結果が空: エラー値 `ErrEmptyBody`（`answer-action`「空の本文は投稿しない」の `Comment` と同じ値）
3. `HasRoutineMarker(title)` または `HasRoutineMarker(body)` が真: エラー値 `ErrRoutineMarker`（`answer-action`「routine マーカーを含む本文は投稿しない」と同じ値・同じ判定）

`ErrEmptyTitle` は `errors.Is` で判別でき、`Error()` はタイトルが空であることを示す。タイトルの無い issue はキューの表で見分けられず、本文の無い issue は routine が読む材料を持たない。マーカーの禁止は不変条件 7「TUI は `<!-- routine -->` を書かない。TUI から投稿する文章はすべて人の発言」で、コメントと同じく issue の本文とタイトルにも及ぶ。

#### Scenario: タイトルが空なら作らない
- **WHEN** `CreateIssue(ctx, client, "org/app", "   ", "本文")` を呼ぶ
- **THEN** `errors.Is(err, ErrEmptyTitle)` が真で、`Fake.Calls` は空である

#### Scenario: 本文が空なら作らない
- **WHEN** `CreateIssue(ctx, client, "org/app", "タイトル", "\n\t\n")` を呼ぶ
- **THEN** `errors.Is(err, ErrEmptyBody)` が真で、`Fake.Calls` は空である

#### Scenario: 本文にマーカーがあれば作らない
- **WHEN** `CreateIssue(ctx, client, "org/app", "タイトル", "routine のコメントは <!-- routine --> で始まる")` を呼ぶ
- **THEN** `errors.Is(err, ErrRoutineMarker)` が真で、`Fake.Calls` は空である

#### Scenario: タイトルにエスケープ済みのマーカーがあれば作らない
- **WHEN** `CreateIssue(ctx, client, "org/app", "&lt;!-- routine --&gt; を含むタイトル", "本文")` を呼ぶ
- **THEN** `errors.Is(err, ErrRoutineMarker)` が真で、`Fake.Calls` は空である

### Requirement: エディタに渡す下書きはプレースホルダー行 2 本を持つ
`internal/action` は、`n` で開くエディタの初期テキストになる下書きを MUST 提供する。中身は次の 4 行（末尾に改行）で固定である。

```
タイトル（この下の行に入力してください）

概要（この下に入力してください）

```

この 2 本のプレースホルダー行は、Requirement「プレースホルダー行 2 本を区切りにタイトルと本文を分ける」が区切りとして使う行そのものである。区切りに使う文字列と分割する関数を同じパッケージに置き、`internal/ui` が文言を持たないことで、片方だけを直して分割できなくなる事故を防ぐ（`answer-action`「回答テンプレートは最新の routine コメントの質問から作る」の `AnswerTemplate` と同じ形で、`internal/ui` は下書きを組み立てない）。
プレースホルダー行の下に空行を 1 つずつ置くのは、人がその空行に書き始められるようにするためである。人が書かずに保存した下書きは、分割ではタイトルも本文も空になり、`new-issue`「編集結果を検査して作成の確認画面に移る」の空の検査で止まる。

#### Scenario: 初期の下書きはプレースホルダー行 2 本と空行を持つ
- **WHEN** `internal/action` が提供する初期の下書きを読む
- **THEN** `タイトル（この下の行に入力してください）\n\n概要（この下に入力してください）\n\n` である

#### Scenario: 書かずに保存した下書きはタイトルも本文も空になる
- **WHEN** 初期の下書きをそのまま `SplitNewIssue` に渡す
- **THEN** `ok` は真で、`title` と `body` はどちらも空文字列である

### Requirement: プレースホルダー行 2 本を区切りにタイトルと本文を分ける
`internal/action` は関数 `SplitNewIssue(text string) (title, body string, ok bool)` を MUST 提供する。`text` を改行で行に分け、次の順で扱う。

1. 各行の末尾の `\r` を落とす（CRLF で保存された下書きを LF と同じに扱う。落とすのは行末の `\r` だけで、行末の空白は本文では残す。Markdown は行末の空白 2 つを改行として読むので、本文から空白を落とすと人が書いた改行が消える）
2. 前後の空白を落とした結果が `タイトル（この下の行に入力してください）` と一致する行を、先頭から探して最初に見つかった 1 本を**タイトルの区切り**にする。1 本も無ければ `title` と `body` を空文字列、`ok` を偽にして返す
3. タイトルの区切りより後の行から、前後の空白を落とした結果が `概要（この下に入力してください）` と一致する行を探し、最初に見つかった 1 本を**本文の区切り**にする。1 本も無ければ `title` と `body` を空文字列、`ok` を偽にして返す
4. 2 本の区切り以外に、前後の空白を落とした結果が 2 つの文言のどちらかと一致する行が残っていたら、`title` と `body` を空文字列、`ok` を偽にして返す
5. `title` は 2 本の区切りに挟まれた行から作る。各行の前後の空白を落とし、空になった行を捨て、残った行を半角空白 1 つで連結する（GitHub の issue のタイトルは 1 行なので、複数行のまま渡す道が無い。書いた文字を捨てず、連結後の姿は `new-issue` の確認画面が `タイトル: <タイトル>` の行で見せる）
6. `body` は本文の区切りより後の行を改行で連結し、前後の空白を落とした結果にする。本文の中の空行と改行はそのまま残す
7. `ok` を真にして返す

**プレースホルダー行の文言と一致する行は、この関数が返す `title` と `body` に 1 行も含まれない。** 手順 4 が保つのは行全体の一致だけで、人が行の一部として引用した文言（`直したいのは 概要（この下に入力してください） の文言です`）は本文としてそのまま残る（区切りに使えない行なので、区切りが壊れる心配は無い）。区切りに使う 2 本を除いても一致する行が残る下書き（案内の行を複製した・案内をもう 1 組貼り直した・案内の文言を本文に引用した）は分割せず、`ok` を偽にする。案内の文字列が issue のタイトルや本文になって GitHub へ出ていく方が、作成を止めるより悪い。この結果、案内の文言そのものを引用した issue は TUI から起こせない（`internal/action` の routine マーカーの判定が「マーカーを引用した issue は TUI から起こせない」を受け入れているのと同じ形で、そのときはブラウザで起票する）。
行の前後の空白を落としてから比べるので、エディタが行末に残した空白や `\r` は区切りの判定に影響しない。逆に、プレースホルダー行に文字を書き足した行（`タイトル（この下の行に入力してください）キュー画面の色を見直す`）は一致しないので区切りにならず、`ok` は偽になる。
見出し記号や引用記号は解釈せず、字面どおりに分ける。`ok` が偽の下書きから作成へ進まないことは `new-issue`「編集結果を検査して作成の確認画面に移る」が定める。

#### Scenario: 区切りに挟まれた行がタイトルになり、下が本文になる
- **WHEN** `SplitNewIssue("タイトル（この下の行に入力してください）\nキュー画面の色を見直す\n\n概要（この下に入力してください）\n種別の色が背景色と競合している。\n")` を呼ぶ
- **THEN** `title` は `キュー画面の色を見直す`、`body` は `種別の色が背景色と競合している。`、`ok` は真である

#### Scenario: 初期の下書きの空行に書き足した形でも分割できる
- **WHEN** `SplitNewIssue("タイトル（この下の行に入力してください）\nキュー画面の色を見直す\n概要（この下に入力してください）\n種別の色が背景色と競合している。\n")` を呼ぶ（4 行の初期の下書きの空行にそのまま打ち込むと、タイトルの下に空行が残らない）
- **THEN** `title` は `キュー画面の色を見直す`、`body` は `種別の色が背景色と競合している。`、`ok` は真である

#### Scenario: 本文は概要の区切りより下すべてになる
- **WHEN** `SplitNewIssue("タイトル（この下の行に入力してください）\n色を見直す\n\n概要（この下に入力してください）\n\n背景色と競合している。\n\n直したい行は 2 つ。\n")` を呼ぶ
- **THEN** `title` は `色を見直す`、`body` は `背景色と競合している。\n\n直したい行は 2 つ。`、`ok` は真である

#### Scenario: タイトルに複数行書かれたら半角空白 1 つで連結する
- **WHEN** `SplitNewIssue("タイトル（この下の行に入力してください）\nキュー画面の色を見直す\n\n（配色）\n概要（この下に入力してください）\n本文")` を呼ぶ
- **THEN** `title` は `キュー画面の色を見直す （配色）`、`body` は `本文`、`ok` は真である

#### Scenario: プレースホルダー行を消した下書きは分割できない
- **WHEN** `SplitNewIssue("キュー画面の色を見直す\n\n種別の色が背景色と競合している。")` を呼ぶ
- **THEN** `ok` は偽で、`title` と `body` はどちらも空文字列である

#### Scenario: 概要のプレースホルダー行だけが無い下書きも分割できない
- **WHEN** `SplitNewIssue("タイトル（この下の行に入力してください）\nキュー画面の色を見直す\n\n種別の色が背景色と競合している。")` を呼ぶ
- **THEN** `ok` は偽で、`title` と `body` はどちらも空文字列である

#### Scenario: 概要の区切りがタイトルの区切りより前にある下書きは分割できない
- **WHEN** `SplitNewIssue("概要（この下に入力してください）\n本文\n\nタイトル（この下の行に入力してください）\nタイトル")` を呼ぶ
- **THEN** `ok` は偽で、`title` と `body` はどちらも空文字列である

#### Scenario: プレースホルダー行に書き足した下書きは分割できない
- **WHEN** `SplitNewIssue("タイトル（この下の行に入力してください）キュー画面の色を見直す\n\n概要（この下に入力してください）\n本文")` を呼ぶ
- **THEN** `ok` は偽で、`title` と `body` はどちらも空文字列である

#### Scenario: 行末の空白と CRLF は区切りの判定に影響せず、本文に `\r` が残らない
- **WHEN** `SplitNewIssue("タイトル（この下の行に入力してください）  \r\n色を見直す\r\n\r\n概要（この下に入力してください）\t\r\n背景色と競合している。\r\n直したい行は 2 つ。\r\n")` を呼ぶ
- **THEN** `title` は `色を見直す`、`body` は `背景色と競合している。\n直したい行は 2 つ。`（`\r` を 1 つも含まない）、`ok` は真である

#### Scenario: 本文の行末の空白は落とさない
- **WHEN** `SplitNewIssue("タイトル（この下の行に入力してください）\n色を見直す\n\n概要（この下に入力してください）\n1 行目です  \n2 行目です\n")` を呼ぶ（本文の 1 行目が半角空白 2 つで終わっている）
- **THEN** `body` は `1 行目です  \n2 行目です` で、行末の空白 2 つが残っている（Markdown はこれを改行として読むので、落とすと人が書いた改行が消える）

#### Scenario: 案内の行を複製した下書きは分割できない
- **WHEN** `SplitNewIssue("タイトル（この下の行に入力してください）\nキュー画面の色を見直す\nタイトル（この下の行に入力してください）\n\n概要（この下に入力してください）\n本文")` を呼ぶ（Vim の行複製などで案内の行が 2 本になった下書き）
- **THEN** `ok` は偽で、`title` と `body` はどちらも空文字列である（区切りの先頭優先だけで分けると、案内の文言がタイトルに入って GitHub へ出ていく）

#### Scenario: 案内の文言を本文に引用した下書きは分割できない
- **WHEN** `SplitNewIssue("タイトル（この下の行に入力してください）\n案内の文言を直したい\n\n概要（この下に入力してください）\n概要（この下に入力してください）\nこの行の文言を変えたい")` を呼ぶ
- **THEN** `ok` は偽で、`title` と `body` はどちらも空文字列である
