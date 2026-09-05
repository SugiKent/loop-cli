# card-model Specification

## Purpose
TBD - created by archiving change s05-classify. Update Purpose after archive.
## Requirements
### Requirement: model は Issue / PR / Comment / Card と分類結果の型を定義する
`internal/model` は次の型を MUST 公開する。生の JSON の写し（`internal/gh`）と画面（`internal/ui`）の間で使う共通の型であり、分類に使う値だけを持つ。ラベルはラベル名の文字列の列で持つ。

- `Comment { Author string; Body string; CreatedAt time.Time; URL string; AI bool }`。`AI` は本文で判定した「routine（AI）の発言か」。
- `Issue { Repo string; Number int; Title string; URL string; Body string; Labels []string; UpdatedAt time.Time; Comments []Comment; Result Result }`。`Repo` は `owner/name`。`Comments` は `gh` の出力順（作成順）で、末尾が最新。nil は「詳細を取得していない」を表す。
- `PR { Repo string; Number int; Title string; URL string; Body string; Labels []string; IsDraft bool; State string; UpdatedAt time.Time; Comments []Comment; MergeState *gh.PRMergeState; ReviewThreads []gh.ReviewThread; Canonical bool; Result Result }`。`State` は `OPEN` / `MERGED` / `CLOSED`（search 由来は `OPEN`、s17 の cross-reference 由来は GraphQL の `state`）。`MergeState` / `ReviewThreads` の nil は「詳細を取得していない」。`Canonical` は「同段階の merge 済み PR のうち最新（正本）」の印で、分類が立てる。
- `Card { Issue *Issue; PRs []PR; Result Result }`。`Issue` が nil のカードは Issue に紐づかない PR 単独のカード（mvp.md: `docs` PR と「その他」バケットの PR だけ）。`PRs` の順序は組み立て側（s07）が決め、分類は順序を変えない。
- `Result { Situation Situation; Priority int; Tab Tab; Summary string }`。`Summary` は「いま人が何をすべきか」の 1 行。
- `Situation` は文字列型で、値は `A` / `B` / `C` / `D` / `E` / `F` / `G` / `other` / `in-progress`（定数 `SituationA` 〜 `SituationG` / `SituationOther` / `SituationInProgress`）と、未分類を表すゼロ値 `""`。`Priority()` と `Tab()` と `Kind()` をメソッドで持つ（値は `human-turn-classify` の Requirement「局面ごとの優先度・タブ・種別・1 行要約が決まる」に定める）。
- `Tab` は文字列型で、値は `今やる` / `バックログ` / `進行中` / `異常`（定数 `TabNow` / `TabBacklog` / `TabInProgress` / `TabAbnormal`）。

ラベル名の定数を持つ: `LabelStageTodo = "stage:todo"` / `LabelStagePropose` / `LabelStageApply` / `LabelStageArchive` / `LabelPropose = "propose"` / `LabelApply` / `LabelArchive` / `LabelDocs = "docs"` / `LabelQuestion = "question"` / `LabelBlocked = "blocked"` / `LabelWip = "wip"`。ラベル名は設定で変えられない（mvp.md）。
補助関数 `HasLabel(labels []string, name string) bool`、`IssueStages(labels []string) []string`（`stage:todo` / `stage:propose` / `stage:apply` / `stage:archive` のうち付いているものを、この順で返す）、`PRStages(labels []string) []string`（`propose` / `apply` / `archive` のうち付いているものを、この順で返す）を持つ。

#### Scenario: IssueStages は段階ラベルだけを段階順で返す
- **WHEN** `IssueStages([]string{"question", "stage:apply", "blocked", "stage:propose"})` を呼ぶ
- **THEN** `[]string{"stage:propose", "stage:apply"}` が返る

#### Scenario: PRStages は PR の段階ラベルだけを返す
- **WHEN** `PRStages([]string{"question", "archive", "docs"})` を呼ぶ
- **THEN** `[]string{"archive"}` が返る

### Requirement: s03 の生の型から model の型に変換する
`internal/model` は変換関数 `IssueFromSearch(gh.SearchIssue) Issue` / `PRFromSearch(gh.SearchPR) PR` / `CommentFrom(gh.Comment) Comment` を MUST 提供する。`IssueFromSearch` は `Repo` に `Repository.NameWithOwner`、`Labels` に `Labels[].Name`、`Comments` に nil を入れる。`PRFromSearch` は同様に写し、`State` に `OPEN`、`Comments` / `MergeState` / `ReviewThreads` に nil を入れる。`CommentFrom` は `Author` に `Author.Login` を入れ、`AI` を `IsAI(Body)` で埋める。
Issue と PR の紐づけ（PR title `[<段階>] #<n>` と本文 `Refs #n` / `Closes #n` のパース、search 結果からの `Card` の組み立て）はこの change では定義しない。後続 change s07 が担当する。

#### Scenario: SearchIssue を Issue に変換する
- **WHEN** `Repository.NameWithOwner` が `org/app`、`Number` が 108、`Labels` が `stage:propose` と `question` の `gh.SearchIssue` を `IssueFromSearch` に渡す
- **THEN** `Repo` が `org/app`、`Number` が 108、`Labels` が `[]string{"stage:propose", "question"}`、`Comments` が nil の `Issue` が返る

#### Scenario: gh.Comment を Comment に変換すると AI 判定が付く
- **WHEN** `Body` が `<!-- routine -->\nQ1: …` の `gh.Comment` を `CommentFrom` に渡す
- **THEN** `AI` が true の `Comment` が返る

### Requirement: AI コメントは本文のマーカーで判定し、author では判定しない
`internal/model` は `IsAI(body string) bool` を MUST 提供する。routine は利用者本人のアカウントで投稿するため author による判定はできない（human-turn-signals.md）。判定は本文だけで行い、次のいずれかで true になる。
1. 本文が字面どおり `<!-- routine -->` で始まる（dispatcher の規約「`<!-- routine -->` で始まらないコメントは人の回答」と同じ基準。先頭の空白や改行を取り除かない）
2. 本文が字面どおり HTML エスケープ済みの `&lt;!-- routine --&gt;` で始まる
3. 本文のいずれかの行が行頭から `## PR リスク評価` で始まる（旧構成の `assess-pr-risk` のコメント。D-004 で判定維持を決めている）。human-turn-signals.md は「見出し」としか書いていないので、本文の先頭行に限定せず、どの行でも見出しなら AI とする

本文の途中に `<!-- routine -->` が現れるだけ（人がマーカーを引用した場合）や、空白・改行の後にマーカーが来る場合は true にしない。

#### Scenario: マーカーで始まるコメントは AI
- **WHEN** `IsAI("<!-- routine -->\n以下 2 点、回答をお願いします")` を呼ぶ
- **THEN** true が返る

#### Scenario: エスケープ済みマーカーで始まるコメントも AI
- **WHEN** `IsAI("&lt;!-- routine --&gt;\nblocked-by: human")` を呼ぶ
- **THEN** true が返る

#### Scenario: PR リスク評価の見出しを含むコメントは AI
- **WHEN** `IsAI("PR #131 の評価です。\n\n## PR リスク評価\n\n- 影響範囲: …")` を呼ぶ
- **THEN** true が返る（見出しが 3 行目にあっても AI と判定する）

#### Scenario: 空白の後にマーカーが来るコメントは人
- **WHEN** `IsAI(" <!-- routine -->\nQ1: A")` を呼ぶ（先頭に空白がある）
- **THEN** false が返る（dispatcher と同じく字面どおり先頭で判定する）

#### Scenario: マーカーの無いコメントは人
- **WHEN** `IsAI("Q1: A\nQ2: B")` を呼ぶ
- **THEN** false が返る

#### Scenario: マーカーを途中で引用した人のコメントは人
- **WHEN** `IsAI("routine のコメントは <!-- routine --> で始まるはずでは？")` を呼ぶ（マーカーが本文の途中にある）
- **THEN** false が返る

### Requirement: PR 本文 1 行目の未確定の判断をパースする
`internal/model` は `ParseUndecided(body string) (n int, ok bool)` を MUST 提供する。本文の先頭の空行を除いた 1 行目が `未確定の判断: N 件`（`N` は 10 進の非負整数。`:` は ASCII、`:` の後と `件` の前の空白は 0 個以上）の形なら `N` と true を返す。1 行目がこの形でなければ `0, false` を返す。2 行目以降は見ない。局面 C の判定（`human-turn-classify`）と merge ガード（s14。不変条件 5「1 行目 `未確定の判断: N 件` で N > 0 なら merge を拒否」）がこれを使う。

#### Scenario: 0 件をパースする
- **WHEN** `ParseUndecided("未確定の判断: 0 件\n\n## 概要\n…")` を呼ぶ
- **THEN** `0, true` が返る

#### Scenario: 2 件をパースする
- **WHEN** `ParseUndecided("未確定の判断: 2 件 — merge しないでください\n…")` を呼ぶ
- **THEN** `2, true` が返る

#### Scenario: 1 行目にその行が無い
- **WHEN** `ParseUndecided("issue #108 の提案。\n\n未確定の判断: 0 件")` を呼ぶ
- **THEN** `0, false` が返る（2 行目以降は見ない）

### Requirement: blocked-by 行を含む最新コメントを正本として検出する
`internal/model` は `LatestBlockedBy(comments []Comment) (c *Comment, value string, ok bool)` を MUST 提供する。`comments` を末尾（最新）から先頭へ見て、先頭の空白を除いて `blocked-by:` で始まる行を含む最初のコメントを返し、`value` にはその行の `blocked-by:` より後ろを前後の空白を除いて入れる（例: `human`、`#12`）。author は見ない（dispatcher は「著者に関係なく `blocked-by:` 行を含む最新コメント」を正本にする。不変条件 8）。無ければ `nil, "", false`。同一コメント内に `blocked-by:` 行が複数あれば最初の行を採る。s09 のカード詳細がこれを上部に要約表示する。

#### Scenario: 最新の blocked-by コメントが正本になる
- **WHEN** コメント 3 件の本文が順に `<!-- routine -->\nblocked-by: #12`、`了解です`、`<!-- routine -->\nblocked-by: human\n## Q1. …` である列で `LatestBlockedBy` を呼ぶ
- **THEN** 3 件目のコメントと `value` が `human`、`ok` が true で返る

#### Scenario: 同趣旨のコメントが連続しても最新を採る
- **WHEN** 本文がすべて `<!-- routine -->\nblocked-by: human` のコメント 3 件で `LatestBlockedBy` を呼ぶ
- **THEN** 3 件目（末尾）のコメントが返る

#### Scenario: blocked-by 行が無い
- **WHEN** 本文が `Q1: A` と `ありがとうございます` の 2 件で `LatestBlockedBy` を呼ぶ
- **THEN** `nil, "", false` が返る

### Requirement: 質問の見出しと選択肢をパースする
`internal/model` は `Question { Number int; Title string; Options []Option }` / `Option { Letter string; Text string; Recommended bool }` と `ParseQuestions(body string) []Question` を MUST 提供する。mvp.md「カード詳細」の形式に従う。
- 見出し行: 先頭の空白を除いて `## Q<n>.` で始まる行（`<n>` は 10 進整数、`.` の後の残りが `Title`。前後の空白を除く）。見出しの出現順に `Question` を作る
- 選択肢行: 直前の見出しに属し、先頭の空白を除いて `-` または `*` で始まり、続いて `選択肢 <Letter>` の形（`<Letter>` は 1 文字の英大文字）。`<Letter>` の直後に `（推奨）` または `(推奨)` があれば `Recommended` を true。その後の `:`（ASCII）または `：`（全角）より後ろを前後の空白を除いて `Text` にする
- 見出しの無い本文（issue の `blocked-by: human` コメントは見出し形式が固定でない）はパースできた分だけ返す。1 件も無ければ空の列を返す

回答テンプレート `Q1: A\nQ2: A` の組み立てと `$EDITOR` への事前入力は s10 が担当する。この change はパースだけを定義する。

#### Scenario: 2 問と推奨を含む本文をパースする
- **WHEN** `ParseQuestions("以下 2 点、回答をお願いします\n## Q1. 名前での絞り込みを今回のスコープに含めるか\n- 選択肢 A（推奨）: 含めない。次の issue に回す\n- 選択肢 B: 含める\n## Q2. カードの情報量の見直しをどこまで行うか\n- 選択肢 A: 今回は触らない\n- 選択肢 B（推奨）: 幅だけ直す")` を呼ぶ
- **THEN** `Question` 2 件が返り、1 件目は `Number` 1、`Title` `名前での絞り込みを今回のスコープに含めるか`、`Options` が `A`（`Recommended` true、`Text` `含めない。次の issue に回す`）と `B`（false）、2 件目は `Number` 2 で `B` が `Recommended` true である

#### Scenario: 見出しの無い本文は空
- **WHEN** `ParseQuestions("<!-- routine -->\nblocked-by: human\n次の方針をコメントで教えてください")` を呼ぶ
- **THEN** 空の列が返る

