## MODIFIED Requirements

### Requirement: 回答テンプレートは最新の routine コメントの質問から組み立てる
`internal/action` は関数 `AnswerTemplate(comments []model.Comment) string` を MUST 提供する。`comments` を末尾（最新）から先頭へ見て、最初に `AI` が true のコメント（最新の routine コメント）を選ぶ。テンプレートは、そのコメントの**引用ブロック**と**回答行**を、間に空行 1 行を挟んでこの順に連結したものとする（末尾に改行を付けない）。引用ブロックが空なら回答行だけ、回答行が 0 行なら引用ブロックだけを返し、どちらも空なら空文字列を返す。引用を先に置くのは、投稿したときに GitHub 上で「質問 → 回答」の読み順になるからである。

**引用ブロック**は、選んだコメントの `Body` を次のとおり写す。
- routine マーカー（`<!-- routine -->` または HTML エスケープ済みの `&lt;!-- routine --&gt;`）を含む行は落とす。マーカーを含む本文は投稿できない（Requirement「routine マーカーを含む本文は投稿しない」）ので、これは引用の前提条件である
- 残った各行は、行末の空白を落とした結果が空なら `>` の 1 文字、そうでなければ `> ` を前に付けた 1 行にする
- 末尾に連なる `>` だけの行は出さない（引用ブロックと回答行の間の空行を 1 行に保つため）
- ただし、次のどちらも満たさないコメントは**引用しない**（引用ブロックが空になる）
  - `model.ParseQuestions(Body)` が 1 件以上の質問を返す（grill の質問コメント）
  - `BlockedByLines(Body)` が 1 行以上を返す（issue の `blocked-by: human` コメント。見出し形式が固定でない）

  この条件は、人への問いではない routine コメント（`## PR リスク評価` の評価コメント、`started:` / `session:` の作業印、`restart:` / `release:` の記録）が最新の routine コメントであるときに、それをエディタへ引用しないためにある。

**回答行**は、`model.ParseQuestions(Body)` の各 `Question` について `Q<Number>: <Letter>` の 1 行を作り、質問の出現順に改行 `\n` で連結したものとする。`<Letter>` は `Options` のうち `Recommended` が true の最初の選択肢の `Letter`（mvp.md「推奨（`（推奨）`）を既定値にする」）、無ければ先頭の選択肢の `Letter`、選択肢が 1 つも無ければ空（行は `Q<Number>: ` で終わる）。質問が 1 件もパースできなければ回答行は 0 行である。

`comments` が nil または空、`AI` が true のコメントが無い、選んだコメントに質問も `blocked-by:` 行も無いのいずれでも空文字列を返す。テンプレートに `<!-- routine -->` を含めない。

引用ブロックはテンプレートの一部であり、人が消さなければそのまま投稿される本文になる。`Comment` は引用行を落とさない。引用した `blocked-by:` 行は行頭が `>` になるので、Requirement「blocked-by 行を投稿前に検出する」の判定に当たらない（人が引用を消さずに投稿しても、確認画面には入らず、dispatcher の正本も動かない）。

#### Scenario: 推奨を既定値にした 2 問のテンプレート
- **WHEN** `AI` が true で `Body` が `<!-- routine -->\n以下 2 点、回答をお願いします\n## Q1. 名前での絞り込みを今回のスコープに含めるか\n- 選択肢 A（推奨）: 含めない。次の issue に回す\n- 選択肢 B: 含める\n## Q2. カードの情報量の見直しをどこまで行うか\n- 選択肢 A: 今回は触らない\n- 選択肢 B（推奨）: 幅だけ直す` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** `> 以下 2 点、回答をお願いします\n> ## Q1. 名前での絞り込みを今回のスコープに含めるか\n> - 選択肢 A（推奨）: 含めない。次の issue に回す\n> - 選択肢 B: 含める\n> ## Q2. カードの情報量の見直しをどこまで行うか\n> - 選択肢 A: 今回は触らない\n> - 選択肢 B（推奨）: 幅だけ直す\n\nQ1: A\nQ2: B` が返る（マーカーの行は引用に入らない）

#### Scenario: 推奨が無ければ先頭の選択肢、選択肢が無ければ空
- **WHEN** `AI` が true で `Body` が `## Q1. 方式をどうするか\n- 選択肢 A: 案 1\n- 選択肢 B: 案 2\n## Q2. 期限はいつか` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** `> ## Q1. 方式をどうするか\n> - 選択肢 A: 案 1\n> - 選択肢 B: 案 2\n> ## Q2. 期限はいつか\n\nQ1: A\nQ2: ` が返る

#### Scenario: 最新の routine コメントを採り、後ろの人のコメントは見ない
- **WHEN** `example` の `issue-108.json` のコメント 2 件（1 件目が AI で `Q1: セッションの寿命は何日にしますか。…`、2 件目が人の `寿命は 30 日で。`）を `model.CommentFrom` で変換した列の後ろに、`AI` が true で `Body` が `## Q1. 方式をどうするか\n- 選択肢 B（推奨）: 案 2` のコメントと、`AI` が false で `Body` が `## Q1. 人が書いた見出し\n- 選択肢 A（推奨）: x` のコメントをこの順で足した列で `AnswerTemplate` を呼ぶ
- **THEN** `> ## Q1. 方式をどうするか\n> - 選択肢 B（推奨）: 案 2\n\nQ1: B` が返る（`AI` が false のコメントは最新でも対象にしない）

#### Scenario: blocked-by のコメントは引用だけのテンプレートになる
- **WHEN** `AI` が true で `Body` が `<!-- routine -->\nblocked-by: human\nunblock-when: comment\n\n認可の方針をどこに書きますか。docs/policy.md を新しく作るか、README に足すかを決めてください。` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** `> blocked-by: human\n> unblock-when: comment\n>\n> 認可の方針をどこに書きますか。docs/policy.md を新しく作るか、README に足すかを決めてください。` が返る（質問が 1 件もパースできないので回答行は無く、空行は `>` の 1 文字で引用される）

#### Scenario: 末尾の空行は引用に出さない
- **WHEN** `AI` が true で `Body` が `## Q1. 方式をどうするか\n- 選択肢 A（推奨）: 案 1\n\n` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** `> ## Q1. 方式をどうするか\n> - 選択肢 A（推奨）: 案 1\n\nQ1: A` が返る（引用ブロックの末尾に `>` の行が残らず、回答行の前の空行は 1 行である）

#### Scenario: 見出しの無い routine コメントは空のテンプレート
- **WHEN** `example` の `pr-131.json` のコメント（`AI` が true、`Body` が `<!-- routine -->\nQ1: マイグレーションを分けますか。`。質問は `Q1:` の 1 行で書かれており `## Q1.` の見出しも `blocked-by:` 行も持たない）を `model.CommentFrom` で変換した 1 件で `AnswerTemplate` を呼ぶ
- **THEN** 空文字列が返る（回答行も引用も作れない）

#### Scenario: 作業印の routine コメントは引用しない
- **WHEN** `AI` が true で `Body` が `<!-- routine -->\nstarted: 2026-09-11T17:05:00Z\nsession: session_01ABC` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** 空文字列が返る（人への問いではないコメントをエディタへ写さない）

#### Scenario: エスケープ済みのマーカーも引用に入らない
- **WHEN** `AI` が true で `Body` が `&lt;!-- routine --&gt;\n## Q1. 方式をどうするか\n- 選択肢 A（推奨）: 案 1` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** `> ## Q1. 方式をどうするか\n> - 選択肢 A（推奨）: 案 1\n\nQ1: A` が返り、`HasRoutineMarker` にその戻り値を渡すと false である

#### Scenario: コメントが無ければ空のテンプレート
- **WHEN** nil と、`AI` が false のコメントだけの列のそれぞれで `AnswerTemplate` を呼ぶ
- **THEN** どちらも空文字列が返る
