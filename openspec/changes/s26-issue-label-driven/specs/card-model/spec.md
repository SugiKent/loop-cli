## ADDED Requirements

### Requirement: 運用方式の型と方式ごとのラベル語彙を model が持つ
`internal/model` は運用方式を表す型 `Mode` を MUST 公開する。値は `sdd`（issue-driven-sdd）と `label`（issue-label-driven）の 2 つ（定数 `ModeSDD` / `ModeLabel`）で、ゼロ値 `""` は `sdd` として扱う。方式は `model.Issue` / `model.PR` のフィールドとしては持たない（設定が正本であり、取得結果やスナップショットに写して持ち回ると、古い値で書き込みや分類を行う経路ができるため）。方式を必要とする関数は引数で受け取る。

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

## MODIFIED Requirements

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
