## MODIFIED Requirements

### Requirement: Fetch は search を 2 回だけ実行し、方式を判定して分類済みの Card 群を返す
`internal/fetch` は `Result { Cards []model.Card; Modes map[string]model.Mode; Errors []error }` と、関数 `Fetch(ctx context.Context, client gh.GHClient, repos []string, now time.Time, grace time.Duration) (*Result, error)` を MUST 提供する。`repos` は `owner/name` の列（s02 の `Config.Repos[].Name`。`Fetch` は `internal/config` を import しない）。運用方式は引数で受け取らず、Requirement「Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する」のとおり `Fetch` が判定する。`now` は分類の基準時刻、`grace` は「その他」を進行中に置く猶予（s30）で、`Fetch` は壁時計を読まずにこの 2 つを `classify.Card` へそのまま渡す（`cmd/loop-cli` の `Fetcher` の閉包が呼ぶたびに `time.Now()` と `Config.OtherGraceMin` 分を渡す）。
`Fetch` は 1 回の呼び出しで `client.SearchIssues(ctx, repos)` を 1 回、`client.SearchPRs(ctx, repos)` を 1 回だけ実行し（D-001「1 回の更新で行う呼び出し」）、得た open issue / open PR を `model.IssueFromSearch` / `model.PRFromSearch` で `model` の型に写し、Requirement「Fetch は open の全 issue / 全 PR の詳細を取得する」の詳細を入れ、Requirement「PR は title と本文のパースで同一リポジトリの Issue に紐づく」で `model.Card` を組み立て、各 Card にそのカードのリポジトリの方式（`Result.Modes` を引き、無ければ `model.Mode` のゼロ値）と `now` と `grace` を添えて s05 の `classify.Card` に通し、結果を `Result.Cards` に入れる。`Result.Cards` の各要素は `Card.Result` / `Issue.Result` / `PRs[i].Result` が埋まった状態で返る（s08 は分類を呼び直さない）。
`Cards` の並びは、issue を持つカードを `SearchIssues` の返却順、続けて PR 単独カードを `SearchPRs` の返却順とする。タブ内の並び替えは s08 が `Card.Result.Tab` / `Priority` で行う。
search 結果の全 issue と全 PR は、それぞれちょうど 1 枚の Card に含まれる。`Fetch` は issue / PR を黙って落とさない。
この Requirement の Scenario で `grace` を書いていないものは `grace` が 0（猶予なし）である。

#### Scenario: example の fixture から Card 2 枚が返る
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` を `client` に、`[]string{"org/app"}` を `repos` に渡して `Fetch` を呼ぶ（`example` は issue 108（`stage:propose` + `question`）、issue 140（ラベル無し）、PR 131（`propose` + `question`、本文に `Closes #108`）を持ち、`labels.json` は `stage:todo` を含む）
- **THEN** エラー無しで `Result` が返り、`Errors` は空、`Cards` は 2 枚で、1 枚目は `Issue.Number` 108 かつ `PRs` が PR 131 の 1 件、2 枚目は `Issue.Number` 140 かつ `PRs` が空である。1 枚目の `Card.Result.Situation` は `A`（`PRs[0].Result.Situation` が `A`、`Issue.Result.Situation` が `in-progress`）、2 枚目の `Card.Result.Situation` は `E` である

#### Scenario: 判定した方式で分類される
- **WHEN** `To Do` / `In Progress` / `Done` を持ち `stage:todo` を持たない `labels.json` と、`org/board` の `In Progress` の issue を持つ fixture を `client` に渡して `Fetch` を呼ぶ
- **THEN** `Result.Modes["org/board"]` は `model.ModeLabel` で、その issue の Card の `Issue.Result.Situation` は `in-progress`、`Summary` は `#<n> は AI が作業中` である

#### Scenario: 方式を判定できないリポジトリは sdd として分類される
- **WHEN** 同じ fixture の issue に対して、`stage:todo` も `To Do` も含まないラベル一覧を返す `GHClient` で `Fetch` を呼ぶ
- **THEN** `Result.Modes` は空で、`In Progress` の issue の `Issue.Result.Situation` は `E` である（sdd の段階ラベルが 1 つも付いていないため）

#### Scenario: now が分類に渡る
- **WHEN** `apply` ラベルで最新コメントが人の open PR（`UpdatedAt` が `2026-09-04T10:00:00Z`）を持つ fixture で、`now` を `2026-09-04T10:30:00Z` と `2026-09-05T10:00:00Z` にしてそれぞれ `Fetch` を呼ぶ
- **THEN** 前者ではその PR の `Result.Summary` は `PR #<n> は auto-fix が受け取り中`（`in-progress`）、後者では `PR #<n> は人のコメントに AI が応答していない`（`other`）である

#### Scenario: grace が分類に渡る
- **WHEN** `Labels` が空、本文に `Refs` / `Closes` が無い PR 61 がある fixture で、`now` に PR 61 の `updatedAt` の 10 分後、`grace` に 30 分を渡して `Fetch` を呼ぶ
- **THEN** PR 61 の Card の `Card.Result.Situation` は `in-progress`、`Summary` は `PR #61 はどの局面にも当たらない（更新から 30m は様子見）` である

#### Scenario: 猶予以上経った now ではその他に戻る
- **WHEN** 同じ fixture で、`now` に PR 61 の `updatedAt` の 31 分後、`grace` に 30 分を渡して `Fetch` を呼ぶ
- **THEN** PR 61 の Card の `Card.Result.Situation` は `other` である

#### Scenario: search は 1 回ずつしか呼ばれない
- **WHEN** `SearchIssues` / `SearchPRs` の呼び出し回数を数える `GHClient` で `Fetch` を呼ぶ
- **THEN** どちらもちょうど 1 回呼ばれる

#### Scenario: 全 issue と全 PR がちょうど 1 枚の Card に含まれる
- **WHEN** `internal/gh/testdata/fixtures/` 直下の各 `<alias>` について `gh.NewFake` で `Fetch` を呼ぶ
- **THEN** エラー無しで返り、`Errors` は空で、`search-issues.json` の各 issue 番号は `Cards[].Issue` にちょうど 1 回、`search-prs.json` の各 PR 番号は `Cards[].PRs` にちょうど 1 回現れる（`Situation` の期待値は s05 の期待値表が持ち、このテストでは見ない）

### Requirement: Issue に紐づかない PR は PR 単独の Card になる
`Fetch` は、`LinkedIssue` が `ok` を返さない PR、または `ok` でも同じリポジトリの open issue に番号 `n` が無い PR（issue が閉じている、`n` が PR の番号、等）を、`Issue` が nil で `PRs` がその PR 1 件だけの Card に MUST する。mvp.md「PR 単独の行が出るのは、Issue に紐づかない `docs` PR と『その他』バケットの PR だけ」は定常状態の説明であり、`Fetch` は紐づかない PR をラベルで選別せず、すべて PR 単独の Card にする（消えて見えなくなる項目を作らない）。
この Requirement の Scenario は `grace` を 0（猶予なし）で `Fetch` を呼ぶ。猶予を渡した場合の `other` の扱いは Requirement「Fetch は search を 2 回だけ実行し、方式を判定して分類済みの Card 群を返す」の Scenario が定める。

#### Scenario: docs PR は PR 単独の Card
- **WHEN** `search-prs.json` に `Labels` が `docs`、title `docs: README を直す`、本文に `Refs` / `Closes` が無い PR 60 がある fixture で `Fetch` を呼ぶ
- **THEN** `Issue` が nil で `PRs` が PR 60 の 1 件の Card があり、その `Card.Result.Situation` は `G` である

#### Scenario: ラベル無しの PR はその他バケットの PR 単独 Card
- **WHEN** `Labels` が空、本文に `Refs` / `Closes` が無い PR 61 がある fixture で `Fetch` を呼ぶ
- **THEN** `Issue` が nil で `PRs` が PR 61 の 1 件の Card があり、`Card.Result.Situation` は `other` である

#### Scenario: 紐づけ先の issue が open issue に無い PR は落ちない
- **WHEN** ラベル無しで title が `[propose] #999`、`search-issues.json` に issue 999 が無い PR 62 がある fixture で `Fetch` を呼ぶ
- **THEN** `Issue` が nil で `PRs` が PR 62 の 1 件の Card がある
