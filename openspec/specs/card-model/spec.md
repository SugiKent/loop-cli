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

ラベル名の定数を持つ: `LabelStageTodo = "stage:todo"` / `LabelStagePropose` / `LabelStageApply` / `LabelStageArchive` / `LabelPropose = "propose"` / `LabelApply` / `LabelArchive` / `LabelDocs = "docs"` / `LabelQuestion = "question"` / `LabelBlocked = "blocked"` / `LabelWip = "wip"`。issue-label-driven の語彙は Requirement「運用方式の型と方式ごとのラベル語彙を model が持つ」が定める。ラベル名は設定で変えられない（mvp.md）。
補助関数 `HasLabel(labels []string, name string) bool` と、方式を引数に取る `IssueStages(mode Mode, labels []string) []string` / `PRStages(mode Mode, labels []string) []string` を持つ（挙動は同 Requirement が定める）。

#### Scenario: IssueStages は段階ラベルだけを段階順で返す
- **WHEN** `IssueStages(ModeSDD, []string{"question", "stage:apply", "blocked", "stage:propose"})` を呼ぶ
- **THEN** `[]string{"stage:propose", "stage:apply"}` が返る

#### Scenario: PRStages は PR の段階ラベルだけを返す
- **WHEN** `PRStages(ModeSDD, []string{"question", "archive", "docs"})` を呼ぶ
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

### Requirement: 運用方式の型と方式ごとのラベル語彙を model が持つ
`internal/model` は運用方式を表す型 `Mode` を MUST 公開する。値は `sdd`（issue-driven-sdd）と `label`（issue-label-driven）の 2 つ（定数 `ModeSDD` / `ModeLabel`）で、ゼロ値 `""` は `sdd` として扱う。方式は `model.Issue` / `model.PR` のフィールドとしては持たない（方式の正本はリポジトリのラベル一覧であり、取得結果やスナップショットに写して持ち回ると、古い値で書き込みや分類を行う経路ができるため）。方式を必要とする関数は引数で受け取る。

方式ごとのラベル名の定数を持つ。issue-label-driven の語彙は `LabelToDo = "To Do"` / `LabelInProgress = "In Progress"` / `LabelDone = "Done"` である。ラベル名は設定で変えられない（mvp.md）。
方式を引数に取る補助関数 `IssueStages(mode Mode, labels []string) []string`、`PRStages(mode Mode, labels []string) []string`、`TodoLabel(mode Mode) string` を持つ。
- `IssueStages` は方式ごとの段階ラベルのうち付いているものを段階順で返す。`sdd` は `stage:todo` / `stage:propose` / `stage:apply` / `stage:archive` の順、`label` は `To Do` / `In Progress` / `Done` の順である
- `PRStages` は `sdd` では PR の段階ラベルを `propose` / `apply` / `archive` の順で返し、`label` では PR に段階ラベルが付かないので常に空を返す
- `TodoLabel` は「人が着手を承認するときに付けるラベル」を返す。`sdd` は `stage:todo`、`label` は `To Do` である

#### Scenario: ゼロ値の Mode は sdd として扱う
- **WHEN** `IssueStages("", []string{"stage:propose"})` と `PRStages("", []string{"propose"})` と `TodoLabel("")` を呼ぶ
- **THEN** 順に `[]string{"stage:propose"}`、`[]string{"propose"}`、`stage:todo` が返る

#### Scenario: label 方式の段階ラベルは To Do / In Progress / Done
- **WHEN** `IssueStages(ModeLabel, []string{"bug", "In Progress", "To Do"})` を呼ぶ
- **THEN** `[]string{"To Do", "In Progress"}` が返る

#### Scenario: label 方式では sdd の語彙を段階ラベルにしない
- **WHEN** `IssueStages(ModeLabel, []string{"stage:propose", "wip"})` と `PRStages(ModeLabel, []string{"propose", "docs"})` を呼ぶ
- **THEN** どちらも空が返る

#### Scenario: TodoLabel は方式ごとの承認ラベルを返す
- **WHEN** `TodoLabel(ModeSDD)` と `TodoLabel(ModeLabel)` を呼ぶ
- **THEN** 順に `stage:todo` と `To Do` が返る

### Requirement: PR 本文の Closes を読む
`internal/model` は `ClosesIssue(body string) (n int, ok bool)` を MUST 提供する。本文中の `Closes #<n>`（`<n>` は 10 進整数）を先頭から探し、最初に見つかった番号を返す。大文字小文字を区別せず、単語として現れる（直前の文字が英数字である `xCloses` は当たらない）ものだけを採り、`Closes` と `#` の間の空白は 1 個以上とする。`Refs #<n>` は採らない。見つからなければ `0, false`。
issue-label-driven では issue と PR の紐づけが `Closes #n` だけであり（issue #5）、分類器はこの関数の `ok` で「routine が作った PR か」を見分ける。`internal/fetch` の `LinkedIssue`（title の段階記法と `Refs` も採る）はカードの組み立てに使う別の判定であり、この関数と役割を分ける。

#### Scenario: 本文の Closes を読む
- **WHEN** `ClosesIssue("ラベル一覧をモーダルで出す。\n\nCloses #12")` を呼ぶ
- **THEN** `12, true` が返る

#### Scenario: Refs は採らない
- **WHEN** `ClosesIssue("Refs #48")` を呼ぶ
- **THEN** `0, false` が返る

#### Scenario: 番号だけの言及は採らない
- **WHEN** `ClosesIssue("#108 と同じ問題")` を呼ぶ
- **THEN** `0, false` が返る

### Requirement: リポジトリのラベル一覧から運用方式を判定する
`internal/model` は `ModeFromLabels(labels []gh.RepoLabel) (Mode, bool)` を MUST 公開する。リポジトリが持つラベルの一覧（`GHClient.ListLabels` の返り値）を受け取り、そのリポジトリの運用方式を返す。判定は次の順で行い、当たったところで止める。

1. 一覧に `stage:todo` がある: `ModeSDD` と `true` を返す
2. 一覧に `To Do`（`model.LabelToDo`）がある: `ModeLabel` と `true` を返す
3. どちらも無い: ゼロ値の `Mode` と `false` を返す

`stage:todo` を先に見るのは、両方のラベルを持つリポジトリを `sdd` に倒すためである。`stage:todo` は issue-driven-sdd の `routines-setup` が作るラベルで、issue-label-driven のリポジトリがこれを作る理由が無い。`To Do` は Projects のカンバンでよく使う名前なので、issue-driven-sdd のリポジトリが別の目的で持っていることがある。

ラベル名の比較は完全一致で、大文字小文字を区別する（ラベル名はプラグインの規約で固定されており、`model.HasLabel` と同じ規則である）。`false` を返したリポジトリを呼び出し側がどう扱うかはこの Requirement では定めない（`card-fetch` と `todo-toggle` が定める）。

#### Scenario: stage:todo があれば sdd
- **WHEN** `ModeFromLabels([]gh.RepoLabel{{Name: "bug"}, {Name: "stage:todo"}, {Name: "wip"}})` を呼ぶ
- **THEN** `ModeSDD` と `true` が返る

#### Scenario: To Do だけがあれば label
- **WHEN** `ModeFromLabels([]gh.RepoLabel{{Name: "Done"}, {Name: "In Progress"}, {Name: "To Do"}})` を呼ぶ
- **THEN** `ModeLabel` と `true` が返る

#### Scenario: 両方あれば sdd に倒す
- **WHEN** `ModeFromLabels([]gh.RepoLabel{{Name: "To Do"}, {Name: "stage:todo"}})` を呼ぶ
- **THEN** `ModeSDD` と `true` が返る

#### Scenario: どちらも無ければ判定できない
- **WHEN** `ModeFromLabels([]gh.RepoLabel{{Name: "bug"}, {Name: "todo"}, {Name: "to do"}})` を呼ぶ
- **THEN** ゼロ値の `Mode` と `false` が返る（小文字の `todo` と `to do` は `To Do` と一致しない）

#### Scenario: 空の一覧は判定できない
- **WHEN** `ModeFromLabels(nil)` を呼ぶ
- **THEN** ゼロ値の `Mode` と `false` が返る
