## MODIFIED Requirements

### Requirement: 質問の見出しと選択肢をパースする
`internal/model` は `Question { Number int; Title string; Options []Option }` / `Option { Letter string; Text string; Recommended bool }` と `ParseQuestions(body string) []Question` を MUST 提供する。読む書式は、routine が実際に投稿する質問コメント（`routine-propose` が PR に書く `### Q<n>.` の見出しと `- **選択肢 A（推奨）**: …` の行）と、mvp.md「カード詳細」が書いた形（`## Q<n>.` と `- 選択肢 A（推奨）: …`）の両方を MUST 含む。
- 見出し行: 先頭の空白を除いて `#` が 1 つ以上並び、続いて `Q<n>.` で始まる行（`<n>` は 10 進整数、`.` の後の残りが `Title`。前後の空白を除く）。`#` の数は問わない（上流の質問コメントは `###`、mvp.md の例は `##`）。見出しの出現順に `Question` を作る
- 選択肢行: 直前の見出しに属し、先頭の空白を除いて `-` または `*` で始まり、続いて `選択肢 <Letter>` の形（`<Letter>` は 1 文字の英大文字）。`<Letter>` の直後に `（推奨）` または `(推奨)` があれば `Recommended` を true。その後の `:`（ASCII）または `：`（全角）より後ろを前後の空白を除いて `Text` にする
- 選択肢行では、Markdown の強調記号 `**` が `選択肢` の前・`<Letter>` の直後・`（推奨）` の直後に現れても読み飛ばす（`- **選択肢 A（推奨）**: x` と `- **選択肢 A**（推奨）: x` と `- 選択肢 A（推奨）: x` が同じ `Option` になる）。`Text` の中の `*` は落とさない
- 見出しの記号（`#` と `Q<n>.`）は要求する。`Q1: A` のように見出しの記号を持たない行は質問として読まない（人が書いた回答のコメントを質問に化けさせないため）
- 見出しの無い本文（issue の `blocked-by: human` コメントは見出し形式が固定でない）はパースできた分だけ返す。1 件も無ければ空の列を返す

回答テンプレートの組み立てと `$EDITOR` への事前入力は `answer-action` / `answer-question` が担当する。この Requirement はパースだけを定義する。

#### Scenario: 2 問と推奨を含む本文をパースする
- **WHEN** `ParseQuestions("以下 2 点、回答をお願いします\n## Q1. 名前での絞り込みを今回のスコープに含めるか\n- 選択肢 A（推奨）: 含めない。次の issue に回す\n- 選択肢 B: 含める\n## Q2. カードの情報量の見直しをどこまで行うか\n- 選択肢 A: 今回は触らない\n- 選択肢 B（推奨）: 幅だけ直す")` を呼ぶ
- **THEN** `Question` 2 件が返り、1 件目は `Number` 1、`Title` `名前での絞り込みを今回のスコープに含めるか`、`Options` が `A`（`Recommended` true、`Text` `含めない。次の issue に回す`）と `B`（false）、2 件目は `Number` 2 で `B` が `Recommended` true である

#### Scenario: 上流の質問コメントの書式をパースする
- **WHEN** `ParseQuestions("<!-- routine -->\n未確定の判断が 2 件あります。`Q1: A` の形で返してください。\n\n---\n\n### Q1. 状態語の色を端末の背景に追随させるか\n\n**何の話か**: 説明の段落。\n\n- **選択肢 A（推奨）**: 端末の背景色を受け取り 2 組を切り替える\n- **選択肢 B**: 状態語にも背景色を敷く\n- 依存: なし\n\n---\n\n### Q2. 色を付ける範囲\n\n- **選択肢 A**（推奨）: 状態語まで広げる\n- **選択肢 B**: ラベル名だけにする")` を呼ぶ
- **THEN** `Question` 2 件が返り、1 件目は `Number` 1、`Title` `状態語の色を端末の背景に追随させるか`、`Options` が `A`（`Recommended` true、`Text` `端末の背景色を受け取り 2 組を切り替える`）と `B`（false、`Text` `状態語にも背景色を敷く`）、2 件目は `Number` 2 で `A` が `Recommended` true である（`- 依存: なし` の行は選択肢にならない）

#### Scenario: 見出しの記号が無い質問は読まない
- **WHEN** `ParseQuestions("<!-- routine -->\nQ1: マイグレーションを分けますか。\nQ2: 期限はいつですか。")` を呼ぶ
- **THEN** 空の列が返る（`#` の見出しを持たないため）

#### Scenario: 見出しの無い本文は空
- **WHEN** `ParseQuestions("<!-- routine -->\nblocked-by: human\n次の方針をコメントで教えてください")` を呼ぶ
- **THEN** 空の列が返る
