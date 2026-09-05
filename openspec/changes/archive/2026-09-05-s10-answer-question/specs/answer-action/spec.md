## ADDED Requirements

### Requirement: 回答の書き先は対象の種類で決まり、ラベルは触らない
`internal/action` は型 `Target { Repo string; Number int; IsPR bool }` と関数 `Comment(ctx context.Context, client gh.GHClient, t Target, body string) error` を MUST 提供する。`Comment` は human-turn-signals.md 不変条件 4「回答の書き先は 3 種類を混同しない」を実装し、`IsPR` が true なら `client.CommentPR(ctx, t.Repo, t.Number, body)`（grill の問い。PR 会話コメント。局面 A）、false なら `client.CommentIssue(ctx, t.Repo, t.Number, body)`（issue の `question`。局面 B）を 1 回だけ呼び、その戻り値をそのまま返す。不変条件 4 の 3 種類目である review thread への返信は、後続 change s16 が `ReplyReviewThread` を使って担当する。この `Comment` は review thread を扱わない。
`Comment` はラベルを書かない（不変条件 2「TUI が書くラベルは `stage:todo` と `stage:propose` に限る」。`question` / `blocked` の付け外しは sweep が行う）。`AddLabel` / `RemoveLabel` を呼ばず、他の書き込みメソッドも呼ばない。

#### Scenario: PR への回答は CommentPR になる
- **WHEN** `gh.NewFake` の `Fake` を `client` にして `Comment(ctx, client, Target{Repo: "org/app", Number: 131, IsPR: true}, "Q1: A\nQ2: B")` を呼ぶ
- **THEN** nil が返り、`Fake.Calls` はちょうど 1 件で、`Method` が `CommentPR`、`Repo` が `org/app`、`Number` が 131、`Body` が `Q1: A\nQ2: B` である

#### Scenario: issue への回答は CommentIssue になる
- **WHEN** `Comment(ctx, client, Target{Repo: "org/app", Number: 108, IsPR: false}, "Q1: A")` を呼ぶ
- **THEN** nil が返り、`Fake.Calls` はちょうど 1 件で、`Method` が `CommentIssue`、`Repo` が `org/app`、`Number` が 108、`Body` が `Q1: A` である

#### Scenario: ラベルの呼び出しは無い
- **WHEN** PR と issue のそれぞれに `Comment` を 1 回ずつ呼んだ後に `Fake.Calls` を見る
- **THEN** `Calls` は 2 件で、`Method` が `AddLabel` または `RemoveLabel` の要素は無い

### Requirement: 空の本文は投稿しない
`Comment` は `body` の前後の空白（空白・タブ・改行）を除いた結果が空文字列なら、エラー値 `ErrEmptyBody` を MUST 返し、`client` のメソッドを呼ばない。空白だけの本文を投稿すると、dispatcher が「`<!-- routine -->` で始まらないコメント」を人の回答とみなして `question` を外す（不変条件 7 の説明）ため、中身の無い回答が「回答済み」になるのを防ぐ。

#### Scenario: 空白だけの本文は拒否される
- **WHEN** `Comment(ctx, client, Target{Repo: "org/app", Number: 131, IsPR: true}, " \n\t\n")` を呼ぶ
- **THEN** `errors.Is(err, ErrEmptyBody)` が真で、`Fake.Calls` は空である

### Requirement: routine マーカーを含む本文は投稿しない
`internal/action` は判定関数 `HasRoutineMarker(body string) bool` を MUST 持ち、`body` に `<!-- routine -->` または HTML エスケープ済みの `&lt;!-- routine --&gt;` が本文のどこかに含まれていれば真を返す。`Comment` は `HasRoutineMarker(body)` が真なら、エラー値 `ErrRoutineMarker` を MUST 返し、`client` のメソッドを呼ばない。根拠は不変条件 7「TUI は `<!-- routine -->` を書かない。TUI から投稿する文章はすべて人の発言」だけである。先頭に限らず本文のどこにあっても拒否する。判定は `body` の字面で行い、大文字小文字や空白の揺れを吸収しない。`## PR リスク評価` 見出しを含む本文は拒否しない（s05 `model.IsAI` はこの見出しを AI 扱いするが、不変条件 7 はマーカーだけを定める。`IsAI` との既知の差であり、design.md の未決事項）。

#### Scenario: 先頭にマーカーがある本文は拒否される
- **WHEN** `Comment(ctx, client, Target{Repo: "org/app", Number: 131, IsPR: true}, "<!-- routine -->\nQ1: A")` を呼ぶ
- **THEN** `errors.Is(err, ErrRoutineMarker)` が真で、`Fake.Calls` は空である

#### Scenario: 途中に引用したマーカーも拒否される
- **WHEN** `Comment(ctx, client, Target{Repo: "org/app", Number: 108, IsPR: false}, "routine のコメントは <!-- routine --> で始まるはずでは？")` を呼ぶ
- **THEN** `errors.Is(err, ErrRoutineMarker)` が真で、`Fake.Calls` は空である

#### Scenario: エスケープ済みのマーカーも拒否される
- **WHEN** `Comment(ctx, client, Target{Repo: "org/app", Number: 108, IsPR: false}, "&lt;!-- routine --&gt;\nQ1: A")` を呼ぶ
- **THEN** `errors.Is(err, ErrRoutineMarker)` が真で、`Fake.Calls` は空である

### Requirement: blocked-by 行を投稿前に検出する
`internal/action` は関数 `BlockedByLines(body string) []string` を MUST 提供する。`body` を行に分け、各行を `strings.TrimSpace` した結果が `blocked-by:` で始まる行を、出現順に（`TrimSpace` した形で）返す。判定は s05 `model.LatestBlockedBy`（`internal/model/parse.go`）と同じ規則（`strings.TrimSpace` して `blocked-by:` で始まる。大文字小文字を区別する）とし、dispatcher の判定とずれないようにする。該当が無ければ空の列を返す。
`Comment` はこの行を含む本文を拒否しない。不変条件 8 は「回答エディタは投稿前にこの行を検出して警告する」であり、警告を見た人が投稿するかどうかを決める（確認は `answer-question` が担当する）。人が引用しただけの `blocked-by:` 行でも dispatcher は著者に関係なくその最新コメントを正本にするため、検出は行頭の引用記号 `>` を吸収せず、字面どおり行頭から判定する（`> blocked-by: human` は検出しない。dispatcher も検出しない）。

#### Scenario: 前後の空白を除いて blocked-by で始まる行が返る
- **WHEN** `BlockedByLines("Q1: A\n  blocked-by: human\nQ2: B\nblocked-by: #12")` を呼ぶ
- **THEN** `[]string{"blocked-by: human", "blocked-by: #12"}` が返る

#### Scenario: blocked-by 行が無ければ空
- **WHEN** `BlockedByLines("Q1: A\n> blocked-by: human を引用します\nQ2: B")` を呼ぶ
- **THEN** 空の列が返る

#### Scenario: Comment は blocked-by 行を含む本文を拒否しない
- **WHEN** `Comment(ctx, client, Target{Repo: "org/app", Number: 108, IsPR: false}, "blocked-by: human\nQ1: A")` を呼ぶ
- **THEN** nil が返り、`Fake.Calls` は 1 件で `Method` が `CommentIssue`、`Body` が `blocked-by: human\nQ1: A` である

### Requirement: 回答テンプレートは最新の routine コメントの質問から組み立てる
`internal/action` は関数 `AnswerTemplate(comments []model.Comment) string` を MUST 提供する。`comments` を末尾（最新）から先頭へ見て、最初に `AI` が true のコメント（最新の routine コメント）を選び、その `Body` を s05 の `model.ParseQuestions` でパースする。各 `Question` について `Q<Number>: <Letter>` の 1 行を作り、質問の出現順に改行 `\n` で連結して返す（末尾に改行を付けない）。`<Letter>` は `Options` のうち `Recommended` が true の最初の選択肢の `Letter`（mvp.md「推奨（`（推奨）`）を既定値にする」）、無ければ先頭の選択肢の `Letter`、選択肢が 1 つも無ければ空（行は `Q<Number>: ` で終わる）。
`comments` が nil または空、`AI` が true のコメントが無い、質問が 1 件もパースできない（issue の `blocked-by: human` コメントは見出し形式が固定でない。mvp.md「パースできた分だけ事前入力する」）のいずれでも空文字列を返す。テンプレートに `<!-- routine -->` を含めない（Requirement「routine マーカーを含む本文は投稿しない」）。

#### Scenario: 推奨を既定値にした 2 問のテンプレート
- **WHEN** `AI` が true で `Body` が `<!-- routine -->\n以下 2 点、回答をお願いします\n## Q1. 名前での絞り込みを今回のスコープに含めるか\n- 選択肢 A（推奨）: 含めない。次の issue に回す\n- 選択肢 B: 含める\n## Q2. カードの情報量の見直しをどこまで行うか\n- 選択肢 A: 今回は触らない\n- 選択肢 B（推奨）: 幅だけ直す` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** `Q1: A\nQ2: B` が返る

#### Scenario: 推奨が無ければ先頭の選択肢、選択肢が無ければ空
- **WHEN** `AI` が true で `Body` が `## Q1. 方式をどうするか\n- 選択肢 A: 案 1\n- 選択肢 B: 案 2\n## Q2. 期限はいつか` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** `Q1: A\nQ2: ` が返る

#### Scenario: 最新の routine コメントを採り、後ろの人のコメントは見ない
- **WHEN** `example` の `issue-108.json` のコメント 2 件（1 件目が AI で `Q1: セッションの寿命は何日にしますか。…`、2 件目が人の `寿命は 30 日で。`）を `model.CommentFrom` で変換した列の後ろに、`AI` が true で `Body` が `## Q1. 方式をどうするか\n- 選択肢 B（推奨）: 案 2` のコメントと、`AI` が false で `Body` が `## Q1. 人が書いた見出し\n- 選択肢 A（推奨）: x` のコメントをこの順で足した列で `AnswerTemplate` を呼ぶ
- **THEN** `Q1: B` が返る（`AI` が false のコメントは最新でも対象にしない）

#### Scenario: 見出しの無い routine コメントは空のテンプレート
- **WHEN** `example` の `pr-131.json` のコメント（`AI` が true、`Body` が `<!-- routine -->\nQ1: マイグレーションを分けますか。`。`## Q1.` の見出し形式ではない）を `model.CommentFrom` で変換した 1 件で `AnswerTemplate` を呼ぶ
- **THEN** 空文字列が返る

#### Scenario: コメントが無ければ空のテンプレート
- **WHEN** nil と、`AI` が false のコメントだけの列のそれぞれで `AnswerTemplate` を呼ぶ
- **THEN** どちらも空文字列が返る
