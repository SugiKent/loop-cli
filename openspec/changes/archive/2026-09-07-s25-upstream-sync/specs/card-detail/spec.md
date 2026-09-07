# card-detail Specification

## MODIFIED Requirements

### Requirement: 本文領域は Issue 本文・最新 blocked-by の要約・コメント時系列を出す
カード詳細画面の `View` は本文領域に、詳細の対象の `Card.Issue` について MUST 次を上から順に出す。
1. `Issue.Body` を Glamour で Markdown レンダリングした文字列（幅は端末の幅。s08 のプレビューと同じ方法。レンダリングが失敗したら `Body` をそのまま出す。空なら何も出さない）
2. `model.LatestBlockedBy(Issue.Comments)` が見つかれば `blocked-by: <value>` の 1 行。`value` が `human` なら、そのコメントに `model.UnblockWhen` の値があれば続けて `unblock-when: <値>` の 1 行を出し（上流 `routine-common`「解除条件を `unblock-when:` の 1 行で明示する」。`comment` は答えのコメントで解け、`docs` は方針文書の更新が要り、`#m` はその issue / PR の完了が要る。人は何をすれば動き出すかをこの行で知る）、さらに `model.ParseQuestions(そのコメントの Body)` の各質問を `Q<Number>. <Title>` の行と、その選択肢を `  <Letter>: <Text>`（`Recommended` なら `  <Letter>（推奨）: <Text>`）の行で出す（mvp.md「`human` なら『人に何を決めてほしいか』の選択肢と推奨がここにある」）。質問が 1 件もパースできなければ、そのコメントの `Body` から routine マーカー行と `blocked-by:` 行と `unblock-when:` 行を除いた本文をそのまま出す。見つからなければこの部分を出さない
3. コメント時系列: `Issue.Comments` が nil なら `コメント: 取得失敗`（s20 は全 issue のコメントを取るので、nil は取得に失敗したことを意味する）、長さ 0 なら `コメント: なし`、1 件以上なら Requirement「routine コメントは折りたたみ、x で展開する」の書式で並び順（末尾が最新）に出す

#### Scenario: issue 108 の本文とコメント
- **WHEN** `example` の issue 108 の Card の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** 本文の `認証まわりの仕様を決めたい` が含まれ、`blocked-by:` は含まれず（`example` に `blocked-by:` 行が無い）、`コメント: 取得失敗` と `コメント: なし` は含まれない

#### Scenario: blocked-by human の要約に選択肢と推奨が出る
- **WHEN** `Labels` が `stage:propose` と `blocked` と `question`、`Comments` が AI の 1 件（`Body` が `<!-- routine -->\nblocked-by: human\n## Q1. 名前での絞り込みを含めるか\n- 選択肢 A（推奨）: 含めない\n- 選択肢 B: 含める`）の issue の Card の詳細を開き、`View` を読む
- **THEN** `blocked-by: human`、`Q1. 名前での絞り込みを含めるか`、`A（推奨）: 含めない`、`B: 含める` がこの順で含まれる

#### Scenario: unblock-when が blocked-by の次の行に出る
- **WHEN** 上と同じで `Comments` の `Body` が `<!-- routine -->\nblocked-by: human\nunblock-when: docs\n## Q1. 認可の方針をどこに書くか\n- 選択肢 A（推奨）: docs/policy.md` の Card の詳細を開き、`View` を読む
- **THEN** `blocked-by: human`、`unblock-when: docs`、`Q1. 認可の方針をどこに書くか` がこの順で含まれる

#### Scenario: 見出しの無い blocked-by コメントは本文をそのまま出す
- **WHEN** 上と同じで `Comments` の `Body` が `<!-- routine -->\nblocked-by: human\nunblock-when: comment\n次の方針をコメントで教えてください` の Card の詳細を開き、`View` を読む
- **THEN** `blocked-by: human`、`unblock-when: comment`、`次の方針をコメントで教えてください` がこの順で含まれる（要約に出した `unblock-when:` 行は、続けて出す本文からは除く）

#### Scenario: コメントが 0 件の issue
- **WHEN** `example` を `Fetch` して得た issue 140 の Card の詳細を開き、`View` を読む（s20 で全 issue のコメントを取るので、`issue-140.json` のコメント 0 件が長さ 0 の非 nil で入る）
- **THEN** 本文の `起動時に設定ファイルが無いと落ちる` と `コメント: なし` が含まれ、`コメント: 取得失敗` と `▌` は含まれない
