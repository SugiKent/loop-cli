## MODIFIED Requirements

### Requirement: Fetch は設定リポジトリに対して search を 2 回だけ実行し分類済みの Card 群を返す
`internal/fetch` は `Result { Cards []model.Card; Errors []error }` と、関数 `Fetch(ctx context.Context, client gh.GHClient, repos []string, modes map[string]model.Mode) (*Result, error)` を MUST 提供する。`repos` は `owner/name` の列（s02 の `Config.Repos[].Name`。`Fetch` は `internal/config` を import しない）。`modes` はリポジトリ名から運用方式を引く表で、呼び出し側が設定から作る。表に無いリポジトリ、および `modes` が nil のときは `model.Mode` のゼロ値（`sdd`）として扱う。
`Fetch` は 1 回の呼び出しで `client.SearchIssues(ctx, repos)` を 1 回、`client.SearchPRs(ctx, repos)` を 1 回だけ実行し（D-001「1 回の更新で行う呼び出し」）、得た open issue / open PR を `model.IssueFromSearch` / `model.PRFromSearch` で `model` の型に写し、Requirement「Fetch は open の全 issue / 全 PR の詳細を取得する」の詳細を入れ、Requirement「PR は title と本文のパースで同一リポジトリの Issue に紐づく」で `model.Card` を組み立て、各 Card をそのカードのリポジトリの方式とともに s05 の `classify.Card` に通して `Result.Cards` に入れる。`Result.Cards` の各要素は `Card.Result` / `Issue.Result` / `PRs[i].Result` が埋まった状態で返る（s08 は分類を呼び直さない）。
`Cards` の並びは、issue を持つカードを `SearchIssues` の返却順、続けて PR 単独カードを `SearchPRs` の返却順とする。タブ内の並び替えは s08 が `Card.Result.Tab` / `Priority` で行う。
search 結果の全 issue と全 PR は、それぞれちょうど 1 枚の Card に含まれる。`Fetch` は issue / PR を黙って落とさない。

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
- **WHEN** `internal/gh/testdata/fixtures/` 直下の各 `<alias>` について `gh.NewFake` で `Fetch` を呼ぶ
- **THEN** エラー無しで返り、`Errors` は空で、`search-issues.json` の各 issue 番号は `Cards[].Issue` にちょうど 1 回、`search-prs.json` の各 PR 番号は `Cards[].PRs` にちょうど 1 回現れる（`Situation` の期待値は s05 の期待値表が持ち、このテストでは見ない）

### Requirement: Card 内の PR は段階順・番号順に並ぶ
`Fetch` は各 Card の `PRs` を、`model.PRStages(mode, Labels)` の先頭の段階が `propose` → `apply` → `archive` の順、段階ラベルが無い PR はその後、同じ段階の中では `Number` の昇順に MUST 並べる（mvp.md「カード詳細」: 紐づく PR を段階順に並べる）。`mode` はそのカードのリポジトリの方式で、`label` では段階ラベルが無いので全 PR が `Number` の昇順に並ぶ。同一 Issue に同段階の open PR が複数あれば全部を `PRs` に入れる。この並びは s05 `classify.Card` の同点判定（`PRs` の並び順で先のもの）に使われる。
search は open PR しか返さないので、`Canonical`（同段階の `MERGED` PR のうち最新）は P1 では立たない。merge 済み PR を持ち込むのは s17 の cross-reference である。

#### Scenario: 段階順に並ぶ
- **WHEN** issue 108 に紐づく PR が `archive` の 151、`propose` の 131、`apply` の 140 の順で `search-prs.json` にある fixture で `Fetch` を呼ぶ
- **THEN** issue 108 の Card の `PRs` は番号順に 131（propose）、140（apply）、151（archive）である

#### Scenario: 同段階の open PR が複数あれば番号順に全部入る
- **WHEN** issue 108 に紐づく `propose` + `question` の PR 131（最新コメントが AI）と `propose` の PR 140（本文 1 行目が `未確定の判断: 1 件`）の 2 件が `search-prs.json` に 140、131 の順である fixture で `Fetch` を呼ぶ
- **THEN** issue 108 の Card の `PRs` は 131、140 の順で 2 件あり、どちらも `Canonical` は false、`Card.Result.Situation` は `A`（PR 131）である

#### Scenario: label 方式では番号順に並ぶ
- **WHEN** `label` の issue に紐づく open PR が 62、61 の順で `search-prs.json` にある fixture で `Fetch` を呼ぶ
- **THEN** その Card の `PRs` は 61、62 の順である
