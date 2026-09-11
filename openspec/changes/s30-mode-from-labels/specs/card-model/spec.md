## MODIFIED Requirements

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

## ADDED Requirements

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
