## MODIFIED Requirements

### Requirement: 回答テンプレートは最新の routine コメントの質問から組み立てる
`internal/action` は関数 `AnswerTemplate(comments []model.Comment) string` を MUST 提供する。テンプレートは**引用ブロック**と**回答行**を、間に空行 1 行を挟んでこの順に連結したものとする（末尾に改行を付けない）。引用ブロックが空なら回答行だけ、回答行が 0 行なら引用ブロックだけを返し、どちらも空なら空文字列を返す。引用を先に置くのは、投稿したときに GitHub 上で「引用された問い → 回答」の読み順になるからである。

**回答行**は、`comments` を末尾（最新）から先頭へ見て最初に `AI` が true のコメント（最新の routine コメント）を選び、その `Body` を `model.ParseQuestions`（`card-model`「質問の見出しと選択肢をパースする」）に渡して作る。各 `Question` について `Q<Number>: <Letter>` の 1 行を質問の出現順に改行 `\n` で連結する。`<Letter>` は `Options` のうち `Recommended` が true の最初の選択肢の `Letter`（mvp.md「推奨（`（推奨）`）を既定値にする」）、無ければ先頭の選択肢の `Letter`、選択肢が 1 つも無ければ空（行は `Q<Number>: ` で終わる）。質問が 1 件もパースできなければ回答行は 0 行である。人のコメントが後ろにあっても最新の routine コメントを選ぶ（この選び方はこの change でも変えない）。

**引用ブロック**は、次の 3 つを MUST すべて満たすときだけ作る。1 つでも満たさなければ空とする。
1. `comments` の末尾のコメントが `AI` である（人が答えた後の、解決済みの問いを引用しないため）
2. そのコメントが人への問いである。すなわち `model.ParseQuestions` が 1 件以上の質問を返すか、`BlockedByLines` が 1 行以上を返す。この 2 つが、人への問いが届く形（grill の質問コメントと issue の `blocked-by: human` コメント）である
3. 次の写し方で残る行が 1 行以上ある

引用ブロックは、末尾のコメントの `Body` を次のとおり写す。
- 前後の空白を除いた結果が routine マーカー（`<!-- routine -->` または `&lt;!-- routine --&gt;`）と一致する行は落とす
- 前後の空白を除いた結果が `blocked-by:` または `unblock-when:` で始まる行は落とす（dispatcher の正本の宣言を引用として持ち回らないため。`docs/domain/issue-driven-sdd/human-turn-signals.md` 不変条件 8）
- 残った行に routine マーカーが含まれていれば、その箇所を `[routine マーカー]` に置き換える。行ごと落とさないのは、質問行にマーカーが含まれていたときに質問文が消えたまま回答行が残るのを避けるためである。判定に使うマーカーの文字列は Requirement「routine マーカーを含む本文は投稿しない」と同一のものを使い、テンプレート側で定義し直さない
- 残った各行は、行末の空白を落とした結果が空なら `>` の 1 文字、そうでなければ `> ` を前に付けた 1 行にする
- 末尾に連なる `>` だけの行は出さない（引用ブロックと回答行の間の空行を 1 行に保つため）

`AnswerTemplate` の戻り値に routine マーカーを含めない。

引用ブロックはテンプレートの一部であり、人が消さなければそのまま投稿される本文になる。`Comment` は引用行を落とさない。

#### Scenario: 上流の質問コメントから引用と回答行ができる
- **WHEN** `AI` が true で `Body` が `<!-- routine -->\n未確定の判断が 2 件あります。\n\n### Q1. 状態語の色を端末の背景に追随させるか\n\n- **選択肢 A（推奨）**: 2 組を切り替える\n- **選択肢 B**: 背景色を敷く\n\n### Q2. 色を付ける範囲\n\n- **選択肢 A**: ラベル名だけ\n- **選択肢 B（推奨）**: 状態語まで` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** 戻り値は `> 未確定の判断が 2 件あります。` から始まり、`> ### Q1. 状態語の色を端末の背景に追随させるか`・`> - **選択肢 A（推奨）**: 2 組を切り替える`・`>`（本文の空行）の各行を含み、`> ` で始まらない最初の行の手前が空行 1 行で、最後の 2 行が `Q1: A` と `Q2: B` である

#### Scenario: 推奨を既定値にした 2 問のテンプレート
- **WHEN** `AI` が true で `Body` が `<!-- routine -->\n以下 2 点、回答をお願いします\n## Q1. 名前での絞り込みを今回のスコープに含めるか\n- 選択肢 A（推奨）: 含めない。次の issue に回す\n- 選択肢 B: 含める\n## Q2. カードの情報量の見直しをどこまで行うか\n- 選択肢 A: 今回は触らない\n- 選択肢 B（推奨）: 幅だけ直す` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** `> 以下 2 点、回答をお願いします\n> ## Q1. 名前での絞り込みを今回のスコープに含めるか\n> - 選択肢 A（推奨）: 含めない。次の issue に回す\n> - 選択肢 B: 含める\n> ## Q2. カードの情報量の見直しをどこまで行うか\n> - 選択肢 A: 今回は触らない\n> - 選択肢 B（推奨）: 幅だけ直す\n\nQ1: A\nQ2: B` が返る（マーカーの行は引用に入らない）

#### Scenario: 推奨が無ければ先頭の選択肢、選択肢が無ければ空
- **WHEN** `AI` が true で `Body` が `## Q1. 方式をどうするか\n- 選択肢 A: 案 1\n- 選択肢 B: 案 2\n## Q2. 期限はいつか` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** 回答行は `Q1: A` と `Q2: ` の 2 行である

#### Scenario: 最新の routine コメントを採り、後ろの人のコメントは見ない
- **WHEN** `example` の `issue-108.json` のコメント 2 件（1 件目が AI で `Q1: セッションの寿命は何日にしますか。…`、2 件目が人の `寿命は 30 日で。`）を `model.CommentFrom` で変換した列の後ろに、`AI` が true で `Body` が `## Q1. 方式をどうするか\n- 選択肢 B（推奨）: 案 2` のコメントと、`AI` が false で `Body` が `## Q1. 人が書いた見出し\n- 選択肢 A（推奨）: x` のコメントをこの順で足した列で `AnswerTemplate` を呼ぶ
- **THEN** `Q1: B` が返る（回答行は最新の `AI` コメントから作り、末尾が人のコメントなので引用は付かない）

#### Scenario: blocked-by のコメントは引用だけのテンプレートになる
- **WHEN** `AI` が true で `Body` が `<!-- routine -->\nblocked-by: human\nunblock-when: comment\n\n認可の方針をどこに書きますか。docs/policy.md を新しく作るか、README に足すかを決めてください。` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** `> 認可の方針をどこに書きますか。docs/policy.md を新しく作るか、README に足すかを決めてください。` が返る（質問がパースできないので回答行は無く、マーカー行と `blocked-by:` 行と `unblock-when:` 行は引用に入らず、先頭の `>` だけの行も残らない）

#### Scenario: 行の途中のマーカーは置き換えて残す
- **WHEN** `AI` が true で `Body` が `<!-- routine -->\n## Q1. 先頭に <!-- routine --> を入れるか\n- 選択肢 A（推奨）: 入れない` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** 戻り値は `> ## Q1. 先頭に [routine マーカー] を入れるか` の行を含み、`Q1: A` の行で終わり、`HasRoutineMarker` にその戻り値を渡すと false である（質問文が消えたまま回答行だけが残ることはない）

#### Scenario: 見出しの無い routine コメントは空のテンプレート
- **WHEN** `example` の `pr-131.json` のコメント（`AI` が true、`Body` が `<!-- routine -->\nQ1: マイグレーションを分けますか。`。質問は `Q1:` の 1 行で書かれており、`#` の見出しも `blocked-by:` 行も持たない）を `model.CommentFrom` で変換した 1 件で `AnswerTemplate` を呼ぶ
- **THEN** 空文字列が返る（回答行も引用も作れない）

#### Scenario: 人への問いではない routine コメントは引用しない
- **WHEN** `AI` が true で `Body` が `<!-- routine -->\nstarted: 2026-09-11T17:05:00Z\nsession: session_01ABC` のコメント 1 件と、`AI` が true で `Body` が `## PR リスク評価\n\n影響範囲は小さい。` のコメント 1 件のそれぞれで `AnswerTemplate` を呼ぶ
- **THEN** どちらも空文字列が返る

#### Scenario: エスケープ済みのマーカーも引用に入らない
- **WHEN** `AI` が true で `Body` が `&lt;!-- routine --&gt;\n## Q1. 方式をどうするか\n- 選択肢 A（推奨）: 案 1` のコメント 1 件で `AnswerTemplate` を呼ぶ
- **THEN** `> ## Q1. 方式をどうするか\n> - 選択肢 A（推奨）: 案 1\n\nQ1: A` が返り、`HasRoutineMarker` にその戻り値を渡すと false である

#### Scenario: コメントが無ければ空のテンプレート
- **WHEN** nil と、`AI` が false のコメントだけの列のそれぞれで `AnswerTemplate` を呼ぶ
- **THEN** どちらも空文字列が返る

### Requirement: 空の本文は投稿しない
`internal/action` は判定関数 `IsBlankAnswer(body string) bool` を MUST 持ち、`body` の各行のうち、前後の空白を除いて空でも `>` で始まってもいない行が 1 つも無ければ真を返す（空白だけの本文、引用行だけの本文、その組み合わせが真になる）。`Comment` は `IsBlankAnswer(body)` が真なら、エラー値 `ErrEmptyBody` を MUST 返し、`client` のメソッドを呼ばない。

空白だけの本文を投稿すると、dispatcher が「`<!-- routine -->` で始まらないコメント」を人の回答とみなして `question` を外す（不変条件 7 の説明）ため、中身の無い回答が「回答済み」になるのを防ぐ。引用行だけの本文を同じ扱いにするのは、Requirement「回答テンプレートは最新の routine コメントの質問から組み立てる」が引用だけのテンプレート（issue の `blocked-by: human`）を作るので、それを開いてそのまま閉じた下書きが「人が答えた」と読まれる経路を作らないためである。人が引用に 1 行でも書き足せば偽になる。

#### Scenario: 空白だけの本文は拒否される
- **WHEN** `Comment(ctx, client, Target{Repo: "org/app", Number: 131, IsPR: true}, " \n\t\n")` を呼ぶ
- **THEN** `errors.Is(err, ErrEmptyBody)` が真で、`Fake.Calls` は空である

#### Scenario: 引用行だけの本文は拒否される
- **WHEN** `Comment(ctx, client, Target{Repo: "org/app", Number: 108, IsPR: false}, "> 認可の方針をどこに書きますか。\n>\n> - 選択肢 A（推奨）: docs/policy.md\n")` を呼ぶ
- **THEN** `errors.Is(err, ErrEmptyBody)` が真で、`Fake.Calls` は空である

#### Scenario: 引用に 1 行でも書き足した本文は投稿される
- **WHEN** `Comment(ctx, client, Target{Repo: "org/app", Number: 108, IsPR: false}, "> 認可の方針をどこに書きますか。\n\nA でお願いします")` を呼ぶ
- **THEN** nil が返り、`Fake.Calls` は 1 件で `Method` が `CommentIssue`、`Body` が引用行を含む本文そのままである
