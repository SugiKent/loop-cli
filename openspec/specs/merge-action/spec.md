# merge-action Specification

## Purpose
TBD - created by archiving change s14-merge-pr. Update Purpose after archive.
## Requirements
### Requirement: CheckMerge は draft と open でない PR を拒否とし、他の判断材料は警告として返す
`internal/action` は関数 `CheckMerge(pr model.PR) (blocked []string, warnings []string)` を MUST 提供する。判定は `pr` の値だけで行い、`gh` を呼ばない。
- `pr.IsDraft` が true なら `blocked` に `draft の PR です` を足す
- `pr.State` が空文字列でも `OPEN` でもなければ `blocked` に `<State> の PR です` を足す（カード詳細の PR 一覧は merged / closed の PR も選べる。s09 `card-detail`）。空文字列は「状態が分からない」として拒否しない
- `model.HasLabel(pr.Labels, model.LabelQuestion)` が true なら `warnings` に `question ラベルが付いています` を足す
- `model.ParseUndecided(pr.Body)` が `ok` true かつ `n` が 1 以上なら `warnings` に `本文 1 行目が「未確定の判断: <n> 件」です` を足す
- `classify.ChecksGreen(pr.MergeState)` が false なら `warnings` に `checks が緑ではありません` を足す（`MergeState` が nil のときも `ChecksGreen` は false を返すので、この警告になる）
`warnings` の順番は上の箇条書きの順（question → 未確定 → checks）とする。該当が無ければ空の列を返す。
`blocked` が空でないとき、`internal/ui` は merge の選択肢を出さない（`merge-pr`「merge の確認画面は判断材料を出し、y で merge して Esc で中止する」）。判定は human-turn-signals.md 不変条件 5 の 4 条件のうち draft だけを拒否に使い、残り 3 つを警告に緩めたものである（docs 逸脱。理由と申し送りは design.md）。

#### Scenario: draft は拒否になる
- **WHEN** `IsDraft` が true、`State` が `OPEN`、`Labels` が空、`Body` が `未確定の判断: 0 件`、`MergeState` の `StatusCheckRollup` が空の `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `blocked` は `[]string{"draft の PR です"}` で、`warnings` は空である

#### Scenario: merged 済みの PR は拒否になる
- **WHEN** `State` が `MERGED`、`IsDraft` が false、`Labels` が空、`Body` が `未確定の判断: 0 件`、`MergeState` の `StatusCheckRollup` が空の `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `blocked` は `[]string{"MERGED の PR です"}` で、`warnings` は空である

#### Scenario: question と未確定と checks は警告になる
- **WHEN** `IsDraft` が false、`Labels` が `propose` と `question`、`Body` が `未確定の判断: 2 件 — merge しないでください`、`MergeState` の `StatusCheckRollup` に `StatusContext` の `ci/legacy` が `PENDING` で入っている `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `blocked` は空で、`warnings` は `[]string{"question ラベルが付いています", "本文 1 行目が「未確定の判断: 2 件」です", "checks が緑ではありません"}` である

#### Scenario: 判断材料が揃っていれば空の列が返る
- **WHEN** `IsDraft` が false、`Labels` が `apply`、`Body` が `未確定の判断: 0 件`、`MergeState` の `StatusCheckRollup` が `CheckRun` の `test` `SUCCESS` 1 件の `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `blocked` と `warnings` はどちらも空である

#### Scenario: 1 行目が未確定の形でなければ未確定の警告は出ない
- **WHEN** `Body` が `issue #108 の提案。\n\nCloses #108` で、`Labels` が空、`IsDraft` が false、`MergeState` の `StatusCheckRollup` が空の `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `blocked` と `warnings` はどちらも空である

#### Scenario: MergeState が nil なら checks の警告が出る
- **WHEN** `MergeState` が nil、`Labels` が空、`IsDraft` が false、`Body` が `未確定の判断: 0 件` の `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `warnings` は `[]string{"checks が緑ではありません"}` で、`blocked` は空である

### Requirement: Merge は MergePR を 1 回だけ呼び、ラベルもコメントも書かない
`internal/action` は関数 `Merge(ctx context.Context, client gh.GHClient, repo string, number int, method string) error` を MUST 提供する。`method` が `squash` / `merge` / `rebase` のいずれかなら `client.MergePR(ctx, repo, number, method)` を 1 回だけ呼び、その戻り値をそのまま返す。いずれでもなければエラー値 `ErrBadMergeMethod` を返し、`client` のメソッドを呼ばない（`gh pr merge` に空のフラグを渡すと対話式になるため、境界で止める）。`ErrBadMergeMethod` は `errors.Is` で判別でき、`Error()` 文字列は渡された `method` を含む。
`Merge` は `CheckMerge` を呼ばない（拒否と警告の提示は `internal/ui` の確認画面の責務であり、`y` を押した人の判断を最後の関門とする。design.md）。`AddLabel` / `RemoveLabel` / `CommentIssue` / `CommentPR` / `CreateIssue` / `ReplyReviewThread` / `Browse` / `OpenURL` を呼ばない（不変条件 2「TUI が書くラベルは `stage:todo` と `stage:propose` に限る」）。

#### Scenario: 設定の方式で MergePR が 1 回呼ばれる
- **WHEN** `gh.NewFake("")` の `Fake` を `client` にして `Merge(ctx, client, "org/app", 151, "squash")` を呼ぶ
- **THEN** nil が返り、`Fake.Calls` はちょうど 1 件で、`Method` が `MergePR`、`Repo` が `org/app`、`Number` が 151、`MergeMethod` が `squash` である

#### Scenario: rebase と merge も通る
- **WHEN** 新しい `Fake` のそれぞれに `Merge(ctx, client, "org/app", 151, "merge")` と `Merge(ctx, client, "org/app", 151, "rebase")` を呼ぶ
- **THEN** どちらも nil が返り、`Calls` の 1 件目の `MergeMethod` はそれぞれ `merge` と `rebase` である

#### Scenario: 方式が不正なら呼ばない
- **WHEN** `Merge(ctx, client, "org/app", 151, "")` と `Merge(ctx, client, "org/app", 151, "SQUASH")` を呼ぶ
- **THEN** どちらも `errors.Is(err, ErrBadMergeMethod)` が真で、`err.Error()` は渡された `method` を含み、`Fake.Calls` は空である

#### Scenario: gh の失敗はそのまま返る
- **WHEN** `*gh.Fake` を埋め込んで `MergePR` だけがエラー `gh pr merge 151 -R org/app --squash: exit 1: Pull request is not mergeable` を返す型を `client` にして `Merge(ctx, client, "org/app", 151, "squash")` を呼ぶ
- **THEN** そのエラーがそのまま返る

#### Scenario: ラベルもコメントも書かない
- **WHEN** `Merge` を成功と失敗のそれぞれで呼んだ後に `Fake.Calls` を見る
- **THEN** `Method` が `AddLabel` / `RemoveLabel` / `CommentIssue` / `CommentPR` / `CreateIssue` / `ReplyReviewThread` / `Browse` / `OpenURL` の要素は 1 件も無い

