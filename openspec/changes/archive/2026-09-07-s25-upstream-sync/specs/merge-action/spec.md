# merge-action Specification

## MODIFIED Requirements

### Requirement: CheckMerge は draft と open でない PR を拒否とし、他の判断材料は警告として返す
`internal/action` は関数 `CheckMerge(pr model.PR) (blocked []string, warnings []string)` を MUST 提供する。判定は `pr` の値だけで行い、`gh` を呼ばない。
- `pr.IsDraft` が true なら `blocked` に `draft の PR です` を足す
- `pr.State` が空文字列でも `OPEN` でもなければ `blocked` に `<State> の PR です` を足す（カード詳細の PR 一覧は merged / closed の PR も選べる。s09 `card-detail`）。空文字列は「状態が分からない」として拒否しない
- `model.HasLabel(pr.Labels, model.LabelQuestion)` が true なら `warnings` に `question ラベルが付いています` を足す
- `model.HasLabel(pr.Labels, model.LabelAIAssess)` が true なら `warnings` に `ai-assess:requested が付いています（AI 評価が未完了）` を足す（上流 `routine-common`「評価を終えた assess が外す」。分類では規則 7 で進行中に落ちるが、カード詳細と PR 詳細から `m` を押す経路が残るため、ここでも人に知らせる）
- `model.ParseUndecided(pr.Body)` が `ok` true かつ `n` が 1 以上なら `warnings` に `本文 1 行目が「未確定の判断: <n> 件」です` を足す
- `classify.ChecksGreen(pr.MergeState)` が false なら `warnings` に `checks が緑ではありません` を足す（`MergeState` が nil のときも `ChecksGreen` は false を返すので、この警告になる）
`warnings` の順番は上の箇条書きの順（question → ai-assess → 未確定 → checks）とする。該当が無ければ空の列を返す。
`blocked` が空でないとき、`internal/ui` は merge の選択肢を出さない（`merge-pr`「merge の確認画面は判断材料を出し、y で merge して Esc で中止する」）。判定は human-turn-signals.md 不変条件 5 の 5 条件のうち draft だけを拒否に使い、残り 4 つを警告に緩めたものである（docs 逸脱。理由と申し送りは design.md）。

#### Scenario: draft は拒否になる
- **WHEN** `IsDraft` が true、`State` が `OPEN`、`Labels` が空、`Body` が `未確定の判断: 0 件`、`MergeState` の `StatusCheckRollup` が空の `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `blocked` は `[]string{"draft の PR です"}` で、`warnings` は空である

#### Scenario: merged 済みの PR は拒否になる
- **WHEN** `State` が `MERGED`、`IsDraft` が false、`Labels` が空、`Body` が `未確定の判断: 0 件`、`MergeState` の `StatusCheckRollup` が空の `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `blocked` は `[]string{"MERGED の PR です"}` で、`warnings` は空である

#### Scenario: question と未確定と checks は警告になる
- **WHEN** `IsDraft` が false、`Labels` が `propose` と `question`、`Body` が `未確定の判断: 2 件 — merge しないでください`、`MergeState` の `StatusCheckRollup` に `StatusContext` の `ci/legacy` が `PENDING` で入っている `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `blocked` は空で、`warnings` は `[]string{"question ラベルが付いています", "本文 1 行目が「未確定の判断: 2 件」です", "checks が緑ではありません"}` である

#### Scenario: AI 評価が未完了なら警告になる
- **WHEN** `IsDraft` が false、`Labels` が `propose` と `ai-assess:requested`、`Body` が `未確定の判断: 0 件`、`MergeState` の `StatusCheckRollup` が `CheckRun` の `test` `SUCCESS` 1 件の `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `blocked` は空で、`warnings` は `[]string{"ai-assess:requested が付いています（AI 評価が未完了）"}` である

#### Scenario: 判断材料が揃っていれば空の列が返る
- **WHEN** `IsDraft` が false、`Labels` が `apply`、`Body` が `未確定の判断: 0 件`、`MergeState` の `StatusCheckRollup` が `CheckRun` の `test` `SUCCESS` 1 件の `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `blocked` と `warnings` はどちらも空である

#### Scenario: 1 行目が未確定の形でなければ未確定の警告は出ない
- **WHEN** `Body` が `issue #108 の提案。\n\nCloses #108` で、`Labels` が空、`IsDraft` が false、`MergeState` の `StatusCheckRollup` が空の `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `blocked` と `warnings` はどちらも空である

#### Scenario: MergeState が nil なら checks の警告が出る
- **WHEN** `MergeState` が nil、`Labels` が空、`IsDraft` が false、`Body` が `未確定の判断: 0 件` の `model.PR` で `CheckMerge` を呼ぶ
- **THEN** `warnings` は `[]string{"checks が緑ではありません"}` で、`blocked` は空である
