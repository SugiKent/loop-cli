# todo-action Specification

## Purpose
TBD - created by archiving change s11-todo-toggle. Update Purpose after archive.

## Requirements

### Requirement: ToggleTodo は現在のラベルを読み直してから stage:todo を付けるか外すかを決める
`internal/action` は関数 `ToggleTodo(ctx context.Context, client gh.GHClient, repo string, number int, mode model.Mode) (added bool, err error)` を MUST 提供する。`mode` はそのリポジトリの運用方式（`model.Mode`。ゼロ値は `sdd`）で、書くラベルは `model.TodoLabel(mode)`（`sdd` は `stage:todo`、`label` は `To Do`）である。
`ToggleTodo` はまず `client.ViewIssue(ctx, repo, number)` を 1 回呼び、返った `IssueDetail.Labels` のラベル名で次の順に判定する（画面の Card が持つ `Labels` は最後の取得時点の値であり、判定に使わない。mvp.md「付け直しは dispatcher への『もう一度評価しろ』の合図」は付いていない承認ラベルを付ける操作、付いている承認ラベルを外すのが取り消しであり、判定は GitHub の現在の状態で行う）。
1. `model.IssueStages(mode, labels)` が 2 件以上（局面 F。段階ラベルが 2 つ以上）: 書き込まず、`ErrMultipleStages` を返す。human-turn-signals.md 局面 F「TUI から自動修復はしない」に従い、承認ラベルが含まれていても外さない
2. `labels` に `model.TodoLabel(mode)` がある（段階ラベルは承認ラベルの 1 件だけ）: `client.RemoveLabel(ctx, repo, number, model.TodoLabel(mode))` を 1 回呼び、`added` false とその戻り値を返す
3. `model.IssueStages(mode, labels)` が 1 件（`sdd` は `stage:propose` / `stage:apply` / `stage:archive`、`label` は `In Progress` / `Done` のいずれか）: 書き込まず、`ErrOtherStage` を返す（mvp.md `s`「段階ラベルは同時に 1 つ」。付けると局面 F になる）
4. `labels` に `blocked` がある: 書き込まず、`ErrBlocked` を返す（局面 E の条件は `blocked` 無し。`blocked` の付け外しは dispatcher が行い、人は触らない）
5. それ以外: `client.AddLabel(ctx, repo, number, model.TodoLabel(mode))` を 1 回呼び、`added` true とその戻り値を返す

`ViewIssue` がエラーを返したら書き込まず、そのエラーを返す。拒否のエラー値 `ErrMultipleStages` / `ErrOtherStage` / `ErrBlocked` は `errors.Is` で判別でき、`Error()` 文字列は理由と該当するラベル名を含む（`ErrOtherStage` は付いている段階ラベル名、`ErrMultipleStages` は段階ラベル名を空白区切りで、`ErrBlocked` は `blocked`）。この判定の 1・3・4 は docs に記述が無く、design.md 未決事項の既定値である。
`ToggleTodo` は判定に使う `IssueDetail` を返さず、画面の Card を書き換えない（書き込み後の再取得は s18 の担当）。

#### Scenario: ラベルの無い issue には stage:todo を付ける
- **WHEN** `gh.NewFake("testdata/todo")` の `Fake`（`issue-153.json` は `labels` が空）を `client` にして `ToggleTodo(ctx, client, "org/app", 153, model.ModeSDD)` を呼ぶ
- **THEN** `added` は true、`err` は nil で、`Fake.Calls` はちょうど 2 件、1 件目は `Method` が `ViewIssue` で `Repo` が `org/app`、`Number` が 153、2 件目は `Method` が `AddLabel` で `Repo` が `org/app`、`Number` が 153、`Label` が `stage:todo` である

#### Scenario: stage:todo が付いている issue からは外す
- **WHEN** `issue-150.json` の `labels` が `stage:todo` と `bug` の状態で `ToggleTodo(ctx, client, "org/app", 150, model.ModeSDD)` を呼ぶ
- **THEN** `added` は false、`err` は nil で、`Fake.Calls` は 2 件、1 件目は `ViewIssue`、2 件目は `Method` が `RemoveLabel` で `Number` が 150、`Label` が `stage:todo` である

#### Scenario: 段階ラベル以外のラベルがあっても付ける
- **WHEN** `issue-154.json` の `labels` が `bug` と `enhancement` の状態で `ToggleTodo(ctx, client, "org/app", 154, model.ModeSDD)` を呼ぶ
- **THEN** `added` は true で、`Fake.Calls` の 2 件目は `AddLabel` / `stage:todo` である

#### Scenario: 別の段階ラベルが付いていれば拒否する
- **WHEN** `issue-151.json` の `labels` が `stage:propose` と `question` の状態で `ToggleTodo(ctx, client, "org/app", 151, model.ModeSDD)` を呼ぶ
- **THEN** `errors.Is(err, ErrOtherStage)` が真で、`err.Error()` に `stage:propose` を含み、`Fake.Calls` は `ViewIssue` の 1 件だけである

#### Scenario: 段階ラベルが 2 つ以上あれば stage:todo を含んでいても拒否する
- **WHEN** `issue-155.json` の `labels` が `stage:todo` と `stage:propose` の状態で `ToggleTodo(ctx, client, "org/app", 155, model.ModeSDD)` を呼ぶ
- **THEN** `errors.Is(err, ErrMultipleStages)` が真で、`err.Error()` に `stage:todo` と `stage:propose` を含み、`Fake.Calls` は `ViewIssue` の 1 件だけである

#### Scenario: blocked が付いていれば拒否する
- **WHEN** `issue-152.json` の `labels` が `blocked` だけの状態で `ToggleTodo(ctx, client, "org/app", 152, model.ModeSDD)` を呼ぶ
- **THEN** `errors.Is(err, ErrBlocked)` が真で、`Fake.Calls` は `ViewIssue` の 1 件だけである

#### Scenario: 読み直しに失敗したら書き込まない
- **WHEN** `issue-999.json` が無い状態で `ToggleTodo(ctx, client, "org/app", 999, model.ModeSDD)` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `issue-999.json` を含み、`Fake.Calls` は `ViewIssue` の 1 件だけである

#### Scenario: label 方式ではラベルの無い issue に To Do を付ける
- **WHEN** `issue-160.json` の `labels` が空の状態で `ToggleTodo(ctx, client, "org/board", 160, model.ModeLabel)` を呼ぶ
- **THEN** `added` は true で、`Fake.Calls` の 2 件目は `Method` が `AddLabel`、`Label` が `To Do` である

#### Scenario: label 方式では To Do を外す
- **WHEN** `issue-161.json` の `labels` が `To Do` の状態で `ToggleTodo(ctx, client, "org/board", 161, model.ModeLabel)` を呼ぶ
- **THEN** `added` は false で、`Fake.Calls` の 2 件目は `Method` が `RemoveLabel`、`Label` が `To Do` である

#### Scenario: label 方式で In Progress が付いていれば拒否する
- **WHEN** `issue-162.json` の `labels` が `In Progress` の状態で `ToggleTodo(ctx, client, "org/board", 162, model.ModeLabel)` を呼ぶ
- **THEN** `errors.Is(err, ErrOtherStage)` が真で、`err.Error()` に `In Progress` を含み、`Fake.Calls` は `ViewIssue` の 1 件だけである

#### Scenario: label 方式では sdd の段階ラベルを段階として数えない
- **WHEN** `issue-163.json` の `labels` が `stage:propose` の状態で `ToggleTodo(ctx, client, "org/board", 163, model.ModeLabel)` を呼ぶ
- **THEN** `added` は true で、`Fake.Calls` の 2 件目は `Method` が `AddLabel`、`Label` が `To Do` である

### Requirement: ラベルの書き込みは 1 操作 1 ラベルで、書くラベルは stage:todo だけ
`ToggleTodo` は 1 回の呼び出しで `client.AddLabel` または `client.RemoveLabel` を MUST 高々 1 回呼び、その `label` 引数は常に `model.TodoLabel(mode)`（`sdd` は `stage:todo`、`label` は `To Do`）である（human-turn-signals.md 不変条件 1「1 操作 1 ラベル。`--add-label` / `--remove-label` は 1 ラベルずつ別プロセスで呼ぶ。ラベル集合の置換は絶対にしない」。s03 の `Client` は `gh issue edit <n> -R <repo> --add-label <label>` / `--remove-label <label>` をラベル 1 つで発行する）。`AddLabel` と `RemoveLabel` を同じ呼び出しの中で両方呼ばない。
`ToggleTodo` は承認ラベル以外のラベル（`stage:propose` / `stage:apply` / `stage:archive` / `In Progress` / `Done` / `blocked` / `question` / `wip` を含む）を付けも外しもしない（不変条件 2「TUI が書くラベルは `stage:todo` と、`s` の強制操作で付ける `stage:propose` の 2 つに限る」。`stage:propose` を書くのは s15 の `s` であり、この関数は書かない）。`CommentIssue` / `CommentPR` / `MergePR` / `CreateIssue` / `ReplyReviewThread` / `Browse` も呼ばない。

#### Scenario: 付けるときも外すときもラベル書き込みは 1 回で stage:todo だけ
- **WHEN** `issue-153.json`（ラベル無し、`sdd`）と `issue-150.json`（`stage:todo`、`sdd`）と `issue-160.json`（ラベル無し、`label`）と `issue-161.json`（`To Do`、`label`）のそれぞれに `ToggleTodo` を 1 回ずつ呼んだ後に `Fake.Calls` を見る
- **THEN** `Method` が `AddLabel` または `RemoveLabel` の要素はそれぞれの呼び出しにつき 1 件だけで、`sdd` の呼び出しの `Label` は `stage:todo`、`label` の呼び出しの `Label` は `To Do` であり、`Method` が `CommentIssue` / `CommentPR` / `MergePR` / `CreateIssue` / `ReplyReviewThread` / `Browse` の要素は無い

#### Scenario: 拒否したときはラベル書き込みが無い
- **WHEN** `issue-151.json`（`stage:propose`）、`issue-152.json`（`blocked`）、`issue-155.json`（段階ラベル 2 つ）、`issue-162.json`（`In Progress`、`label`）のそれぞれに `ToggleTodo` を呼んだ後に `Fake.Calls` を見る
- **THEN** `Method` が `AddLabel` または `RemoveLabel` の要素は 1 件も無い
