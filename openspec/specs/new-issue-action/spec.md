# new-issue-action Specification

## Purpose
TUI から新しい issue を起こすときの書き込みの直前を担う。下書きをタイトルと本文に分ける規則と、`gh` を呼ぶ前の検査（空・routine マーカー）を 1 か所に置き、`internal/ui` が判定を持たずに済むようにする。

## Requirements

### Requirement: 下書きの 1 行目がタイトル、残りが本文になる
`internal/action` は関数 `SplitNewIssue(text string) (title, body string)` を MUST 提供する。`text` を最初の改行で 2 つに分け、前半を `strings.TrimSpace` した結果を `title`、後半を `strings.TrimSpace` した結果を `body` として返す。改行が 1 つも無ければ `body` は空文字列である。
この規則は `$EDITOR` で書いた 1 枚のテキストから issue を作るためのもので、人が「1 行目にタイトル、空行を挟んで本文」と書いた下書きをそのまま扱える（前後の空行と末尾の改行は落ちる）。見出し記号や引用記号を解釈せず、字面どおりに分ける。

#### Scenario: 1 行目がタイトルになり空行が落ちる
- **WHEN** `SplitNewIssue("n キーで issue を作る\n\n選択中の repo に作る。\n本文はここから。\n")` を呼ぶ
- **THEN** `title` は `n キーで issue を作る`、`body` は `選択中の repo に作る。\n本文はここから。` である

#### Scenario: 1 行だけの下書きは本文が空になる
- **WHEN** `SplitNewIssue("  タイトルだけ  ")` を呼ぶ
- **THEN** `title` は `タイトルだけ`、`body` は空文字列である

#### Scenario: 空行から始まる下書きはタイトルが空になる
- **WHEN** `SplitNewIssue("\n本文だけ書いた")` を呼ぶ
- **THEN** `title` は空文字列、`body` は `本文だけ書いた` である

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
