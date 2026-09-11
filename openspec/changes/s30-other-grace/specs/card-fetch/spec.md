## MODIFIED Requirements

### Requirement: Fetch は設定リポジトリに対して search を 2 回だけ実行し分類済みの Card 群を返す
`internal/fetch` は `Result { Cards []model.Card; Errors []error }` と、関数 `Fetch(ctx context.Context, client gh.GHClient, repos []string, modes map[string]model.Mode, now time.Time, grace time.Duration) (*Result, error)` を MUST 提供する。`repos` は `owner/name` の列（s02 の `Config.Repos[].Name`。`Fetch` は `internal/config` を import しない）。`modes` はリポジトリ名から運用方式を引く表で、呼び出し側が設定から作る。表に無いリポジトリ、および `modes` が nil のときは `model.Mode` のゼロ値（`sdd`）として扱う。`now` は取得時刻、`grace` は「その他」を進行中に置く猶予（s30）で、呼び出し側（`cmd/loop-cli` の `Fetcher` の閉包）が呼ぶたびに `time.Now()` と `Config.OtherGraceMin` 分を渡す。`Fetch` は壁時計を読まず、`now` / `grace` を `classify.Card` にそのまま渡す。
`Fetch` は 1 回の呼び出しで `client.SearchIssues(ctx, repos)` を 1 回、`client.SearchPRs(ctx, repos)` を 1 回だけ実行し（D-001「1 回の更新で行う呼び出し」）、得た open issue / open PR を `model.IssueFromSearch` / `model.PRFromSearch` で `model` の型に写し、Requirement「Fetch は open の全 issue / 全 PR の詳細を取得する」の詳細を入れ、Requirement「PR は title と本文のパースで同一リポジトリの Issue に紐づく」で `model.Card` を組み立て、各 Card を s05 の `classify.Card` に通し（そのカードのリポジトリの方式、`now`、`grace` を渡す）て `Result.Cards` に入れる。`Result.Cards` の各要素は `Card.Result` / `Issue.Result` / `PRs[i].Result` が埋まった状態で返る（s08 は分類を呼び直さない）。
`Cards` の並びは、issue を持つカードを `SearchIssues` の返却順、続けて PR 単独カードを `SearchPRs` の返却順とする。タブ内の並び替えは s08 が `Card.Result.Tab` / `Priority` で行う。
search 結果の全 issue と全 PR は、それぞれちょうど 1 枚の Card に含まれる。`Fetch` は issue / PR を黙って落とさない。
この Requirement の Scenario で `now` / `grace` を書いていないものは `grace` が 0（猶予なし）である。

#### Scenario: example の fixture から Card 2 枚が返る
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` を `client` に、`[]string{"org/app"}` を `repos` に、nil を `modes` に渡して `Fetch` を呼ぶ（`example` は issue 108（`stage:propose` + `question`）、issue 140（ラベル無し）、PR 131（`propose` + `question`、本文に `Closes #108`）を持つ）
- **THEN** エラー無しで `Result` が返り、`Errors` は空、`Cards` は 2 枚で、1 枚目は `Issue.Number` 108 かつ `PRs` が PR 131 の 1 件、2 枚目は `Issue.Number` 140 かつ `PRs` が空である。1 枚目の `Card.Result.Situation` は `A`（`PRs[0].Result.Situation` が `A`、`Issue.Result.Situation` が `in-progress`）、2 枚目の `Card.Result.Situation` は `E` である

#### Scenario: modes で指定した方式で分類される
- **WHEN** issue-label-driven の fixture（`org/board` の `In Progress` の issue を含む）を `client` に、`map[string]model.Mode{"org/board": model.ModeLabel}` を `modes` に渡して `Fetch` を呼ぶ
- **THEN** その issue の Card の `Issue.Result.Situation` は `in-progress` で、`Summary` は `#<n> は AI が作業中` である

#### Scenario: modes に無いリポジトリは sdd として分類される
- **WHEN** 同じ fixture を `client` に、空の `modes` を渡して `Fetch` を呼ぶ
- **THEN** `In Progress` の issue の `Issue.Result.Situation` は `E` である（sdd の段階ラベルが 1 つも付いていないため）

#### Scenario: search は 1 回ずつしか呼ばれない
- **WHEN** `SearchIssues` / `SearchPRs` の呼び出し回数を数える `GHClient` で `Fetch` を呼ぶ
- **THEN** どちらもちょうど 1 回呼ばれる

#### Scenario: 全 issue と全 PR がちょうど 1 枚の Card に含まれる
- **WHEN** `LinkedIssue("fix typo", "#108 と同じ問題")` を呼ぶ
- **THEN** `0, false` が返る

#### Scenario: 複数リポジトリで同じ番号は混ざらない
- **WHEN** `search-issues.json` に `org/app` の issue 12 と `org/web` の issue 12（どちらもラベル無し）、`search-prs.json` に `org/web` の PR 30（title `[propose] #12 ログイン画面`、`propose` ラベル、本文 1 行目が `未確定の判断: 1 件`）がある fixture で `Fetch` を呼ぶ
- **THEN** `Cards` は 2 枚で、`org/web` の issue 12 の Card の `PRs` は PR 30 の 1 件、`org/app` の issue 12 の Card の `PRs` は空である

#### Scenario: now と grace が分類に届く
- **WHEN** `Labels` が空、本文に `Refs` / `Closes` が無い PR 61 がある fixture で、`now` に PR 61 の `updatedAt` の 10 分後、`grace` に 30 分を渡して `Fetch` を呼ぶ
- **THEN** PR 61 の Card の `Card.Result.Situation` は `in-progress`、`Summary` は `PR #61 はどの局面にも当たらない（更新から 30m は様子見）` である

#### Scenario: 猶予以上経った now ではその他に戻る
- **WHEN** 同じ fixture で、`now` に PR 61 の `updatedAt` の 31 分後、`grace` に 30 分を渡して `Fetch` を呼ぶ
- **THEN** PR 61 の Card の `Card.Result.Situation` は `other` である

### Requirement: Issue に紐づかない PR は PR 単独の Card になる
`Fetch` は、`LinkedIssue` が `ok` を返さない PR、または `ok` でも同じリポジトリの open issue に番号 `n` が無い PR（issue が閉じている、`n` が PR の番号、等）を、`Issue` が nil で `PRs` がその PR 1 件だけの Card に MUST する。mvp.md「PR 単独の行が出るのは、Issue に紐づかない `docs` PR と『その他』バケットの PR だけ」は定常状態の説明であり、`Fetch` は紐づかない PR をラベルで選別せず、すべて PR 単独の Card にする（消えて見えなくなる項目を作らない）。
この Requirement の Scenario で `now` / `grace` を書いていないものは `grace` が 0（猶予なし）である。

#### Scenario: docs PR は PR 単独の Card
- **WHEN** `search-prs.json` に `Labels` が `docs`、title `docs: README を直す`、本文に `Refs` / `Closes` が無い PR 60 がある fixture で `Fetch` を呼ぶ
- **THEN** `Issue` が nil で `PRs` が PR 60 の 1 件の Card があり、その `Card.Result.Situation` は `G` である

#### Scenario: ラベル無しの PR はその他バケットの PR 単独 Card
- **WHEN** `Labels` が空、本文に `Refs` / `Closes` が無い PR 61 がある fixture で `Fetch` を呼ぶ
- **THEN** `Issue` が nil で `PRs` が PR 61 の 1 件の Card があり、`Card.Result.Situation` は `other` である

#### Scenario: 紐づけ先の issue が open issue に無い PR は落ちない
- **WHEN** ラベル無しで title が `[propose] #999`、`search-issues.json` に issue 999 が無い PR 62 がある fixture で `Fetch` を呼ぶ
- **THEN** `Issue` が nil で `PRs` が PR 62 の 1 件の Card がある
