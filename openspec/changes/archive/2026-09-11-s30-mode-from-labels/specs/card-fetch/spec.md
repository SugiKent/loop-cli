## REMOVED Requirements

### Requirement: Fetch は設定リポジトリに対して search を 2 回だけ実行し分類済みの Card 群を返す

**Reason**: `Fetch` の引数から `modes` が消え、方式を `Fetch` 自身が判定するようになる。この Requirement の Scenario「modes で指定した方式で分類される」「modes に無いリポジトリは sdd として分類される」は、呼び出し側が渡した表で分類することを固定しており、引数そのものが無くなると成り立たない。Scenario 名が消える引数の名前を含むため MODIFIED では書き直せないので、Requirement 名を「Fetch は search を 2 回だけ実行し、方式を判定して分類済みの Card 群を返す」に改めて作り直す。

**Migration**: search を 1 回ずつしか呼ばないこと、`Cards` の並び、全 issue / 全 PR がちょうど 1 枚の Card に入ることは、ADDED の Requirement がそのまま引き継ぐ。呼び出し側（`cmd/loop-cli`）は `Fetch` に渡していた `modes` を落とす。方式ごとの分類が正しいことは、ADDED の Requirement の Scenario「判定した方式で分類される」と、ADDED の Requirement「Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する」が引き継ぐ。

## ADDED Requirements

### Requirement: Fetch は search を 2 回だけ実行し、方式を判定して分類済みの Card 群を返す
`internal/fetch` は `Result { Cards []model.Card; Modes map[string]model.Mode; Errors []error }` と、関数 `Fetch(ctx context.Context, client gh.GHClient, repos []string) (*Result, error)` を MUST 提供する。`repos` は `owner/name` の列（s02 の `Config.Repos[].Name`。`Fetch` は `internal/config` を import しない）。運用方式は引数で受け取らず、Requirement「Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する」のとおり `Fetch` が判定する。
`Fetch` は 1 回の呼び出しで `client.SearchIssues(ctx, repos)` を 1 回、`client.SearchPRs(ctx, repos)` を 1 回だけ実行し（D-001「1 回の更新で行う呼び出し」）、得た open issue / open PR を `model.IssueFromSearch` / `model.PRFromSearch` で `model` の型に写し、Requirement「Fetch は open の全 issue / 全 PR の詳細を取得する」の詳細を入れ、Requirement「PR は title と本文のパースで同一リポジトリの Issue に紐づく」で `model.Card` を組み立て、各 Card にそのカードのリポジトリの方式（`Result.Modes` を引き、無ければ `model.Mode` のゼロ値）を添えて s05 の `classify.Card` に通し、結果を `Result.Cards` に入れる。`Result.Cards` の各要素は `Card.Result` / `Issue.Result` / `PRs[i].Result` が埋まった状態で返る（s08 は分類を呼び直さない）。
`Cards` の並びは、issue を持つカードを `SearchIssues` の返却順、続けて PR 単独カードを `SearchPRs` の返却順とする。タブ内の並び替えは s08 が `Card.Result.Tab` / `Priority` で行う。
search 結果の全 issue と全 PR は、それぞれちょうど 1 枚の Card に含まれる。`Fetch` は issue / PR を黙って落とさない。

#### Scenario: example の fixture から Card 2 枚が返る
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` を `client` に、`[]string{"org/app"}` を `repos` に渡して `Fetch` を呼ぶ（`example` は issue 108（`stage:propose` + `question`）、issue 140（ラベル無し）、PR 131（`propose` + `question`、本文に `Closes #108`）を持ち、`labels.json` は `stage:todo` を含む）
- **THEN** エラー無しで `Result` が返り、`Errors` は空、`Cards` は 2 枚で、1 枚目は `Issue.Number` 108 かつ `PRs` が PR 131 の 1 件、2 枚目は `Issue.Number` 140 かつ `PRs` が空である。1 枚目の `Card.Result.Situation` は `A`（`PRs[0].Result.Situation` が `A`、`Issue.Result.Situation` が `in-progress`）、2 枚目の `Card.Result.Situation` は `E` である

#### Scenario: 判定した方式で分類される
- **WHEN** `To Do` / `In Progress` / `Done` を持ち `stage:todo` を持たない `labels.json` と、`org/board` の `In Progress` の issue を持つ fixture を `client` に渡して `Fetch` を呼ぶ
- **THEN** `Result.Modes["org/board"]` は `model.ModeLabel` で、その issue の Card の `Issue.Result.Situation` は `in-progress`、`Summary` は `#<n> は AI が作業中` である

#### Scenario: 方式を判定できないリポジトリは sdd として分類される
- **WHEN** 同じ fixture の issue に対して、`stage:todo` も `To Do` も含まないラベル一覧を返す `GHClient` で `Fetch` を呼ぶ
- **THEN** `Result.Modes` は空で、`In Progress` の issue の `Issue.Result.Situation` は `E` である（sdd の段階ラベルが 1 つも付いていないため）

#### Scenario: search は 1 回ずつしか呼ばれない
- **WHEN** `SearchIssues` / `SearchPRs` の呼び出し回数を数える `GHClient` で `Fetch` を呼ぶ
- **THEN** どちらもちょうど 1 回呼ばれる

#### Scenario: 全 issue と全 PR がちょうど 1 枚の Card に含まれる
- **WHEN** `internal/gh/testdata/fixtures/` 直下の各 `<alias>` について `gh.NewFake` で `Fetch` を呼ぶ
- **THEN** エラー無しで返り、`Errors` は空で、`search-issues.json` の各 issue 番号は `Cards[].Issue` にちょうど 1 回、`search-prs.json` の各 PR 番号は `Cards[].PRs` にちょうど 1 回現れる（`Situation` の期待値は s05 の期待値表が持ち、このテストでは見ない）

### Requirement: Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する
`Fetch` は 1 回の呼び出しで、`repos` の各リポジトリについて `client.ListLabels(ctx, repo)` を MUST ちょうど 1 回実行する。取得したラベル一覧を `model.ModeFromLabels` に渡し、判定できたリポジトリだけを `Result.Modes` に入れる。判定できなかったリポジトリ（`stage:todo` も `To Do` も持たない）と、`ListLabels` が失敗したリポジトリは `Result.Modes` に入れない。ラベル一覧そのものは `Result` に持たせず、判定した方式だけを返す（`L` のラベル一覧は `internal/ui` が自分で取り直す。`label-picker`「`L` は画面の対象のラベル一覧画面を開く」）。

方式は毎回の取得で判定し直す。`Fetch` は前回の結果を持たず、設定ファイルにもスナップショットにも書かない。GitHub 側でラベルを足す・消すと、次の `R` または自動更新の後の `Result.Modes` が変わる（loop-cli の再起動は要らない）。

判定に使うのはラベルの存在だけで、Routines の設定や `openspec/` の有無は見ない。

#### Scenario: リポジトリごとに 1 回ずつ呼ぶ
- **WHEN** `ListLabels` の呼び出しを記録する `GHClient` で、`repos` に `org/app` と `org/board` を渡して `Fetch` を呼ぶ
- **THEN** `ListLabels` はちょうど 2 回呼ばれ、引数の `repo` は `org/app` と `org/board` が 1 回ずつである

#### Scenario: リポジトリごとに違う方式を判定する
- **WHEN** `org/app` には `stage:todo` を含むラベル一覧、`org/board` には `To Do` を含み `stage:todo` を含まないラベル一覧を返す `GHClient` で `Fetch` を呼ぶ
- **THEN** `Result.Modes` は `{"org/app": ModeSDD, "org/board": ModeLabel}` であり、`org/board` の `In Progress` の issue は `label` の語彙で分類される

#### Scenario: 判定できないリポジトリは表に入らない
- **WHEN** `stage:todo` も `To Do` も含まないラベル一覧を返す `GHClient` で、`repos` に `org/app` を渡して `Fetch` を呼ぶ
- **THEN** `Result.Modes` は空で、`Errors` も空である（判定できないことは失敗ではない）

## MODIFIED Requirements

### Requirement: gh 呼び出しにはタイムアウトを付け、詳細取得は並行する
`Fetch` は `client` の各メソッド呼び出しを、引数の `ctx` から `context.WithTimeout` で派生させた 30 秒の期限付き `ctx` で MUST 実行する（s03 design.md 未決事項「サブプロセスのタイムアウト」で s07 に委ねられた値。`Client` は `ctx` に従うだけで自前の期限を持たない）。期限は定数 `CallTimeout` として `internal/fetch` が公開する。
詳細取得（`ViewIssue` / `ViewPR` / `ViewPRMergeState` / `ReviewThreads`）とリポジトリごとの `ListLabels` は並行して実行してよい。並行度の上限と search 2 回の直列は design.md 未決事項の既定値（上限 4、search は順に 1 回ずつ）であり、この Requirement は固定しない。`ListLabels` を詳細取得と同じ上限の中で走らせることで、`gh` の同時実行数を 1 か所で抑える。並行しても `Result.Cards` の並びと各 Card の `PRs` の並びは Requirement のとおり決定的でなければならない。

#### Scenario: 親 ctx の期限が伝わる
- **WHEN** `SearchIssues` が `ctx.Done()` まで待ってから `ctx.Err()` を返す `GHClient` に対して、50 ミリ秒の期限付き `ctx` で `Fetch` を呼ぶ
- **THEN** 50 ミリ秒程度でエラーが返り、`errors.Is(err, context.DeadlineExceeded)` が真である

### Requirement: search の失敗と ctx の中断はエラー、詳細取得の失敗は部分失敗として Card を残す
`Fetch` は次の失敗の扱いを MUST 守る。
- `SearchIssues` または `SearchPRs` がエラーを返したら、そのエラーを（どちらの search かが分かるよう `search issues: ` / `search prs: ` を前置して）返し、`Result` は nil。D-002「失敗時は前回結果を維持してエラーを表示」の「前回結果の維持」と表示は s08 / s13 が行う
- 詳細取得の途中で引数の `ctx` がキャンセルまたは期限切れになったら、`ctx.Err()` を含むエラーを返し、`Result` は nil（途中までの Card でスナップショットを上書きしないため）
- 詳細取得 1 件が上記以外の理由で失敗したら、その issue / PR を落とさず、対応する詳細を nil のまま Card に残し、`Result.Errors` に「メソッド名・`owner/name`・番号」を含むエラー（元のエラーを `%w` で包む）を追加する。他の詳細取得は続ける
- `ListLabels` 1 件が失敗したら、そのリポジトリを `Result.Modes` に入れず、`Result.Errors` に「`ListLabels`・`owner/name`」を含むエラー（元のエラーを `%w` で包む）を追加する。他のリポジトリと詳細取得は続ける。そのリポジトリのカードはゼロ値の方式（`sdd`）で分類する
- `Errors` の並びは、リポジトリのエラーを repo 順、続けて issue のエラーを (repo, number) 順、続けて PR のエラーを (repo, number, メソッド) 順（メソッドは `ViewPR` → `ViewPRMergeState` → `ReviewThreads` の順）とする。`ListLabels` の失敗は特定の issue / PR に属さないので番号を持つエラーより前に置く。1 つの PR が複数の詳細で失敗すると同じ (repo, number) のエラーが 3 件まで並ぶので、その中の順序もメソッドで決める（並行実行の完了順に依存しない）
- 詳細取得と `ListLabels` がすべて成功すれば `Errors` は空（nil または長さ 0）

#### Scenario: search issues の失敗
- **WHEN** `search-issues.json` が無い fixture ディレクトリで `Fetch` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `search issues` と `search-issues.json` を含み、`Result` は nil である

#### Scenario: 1 つの PR の詳細がすべて失敗しても Card を落とさない
- **WHEN** `search-prs.json` に `propose` + `question` の PR 131（本文 `Closes #108`）があるが `pr-131.json` と `pr-131-review-threads.json` が無く、issue 108 は `issue-108.json` があり `labels.json` もある fixture で `Fetch` を呼ぶ
- **THEN** エラー無しで `Result` が返り、issue 108 の Card の `PRs` に PR 131 があって `Comments` と `MergeState` と `ReviewThreads` はいずれも nil、`PRs[0].Result.Situation` は `other`、`Errors` は 3 件で、それぞれの文字列に `ViewPR` / `ViewPRMergeState` / `ReviewThreads` のいずれかと `org/app` と `131` を含む。issue 108 の `Comments` は埋まっている

#### Scenario: ラベル一覧の取得に失敗したリポジトリは方式の表に入らない
- **WHEN** `org/app` のラベル一覧だけがエラーを返し、他の呼び出しは成功する `GHClient` で、`repos` に `org/app` と `org/board` を渡して `Fetch` を呼ぶ
- **THEN** エラー無しで `Result` が返り、`Cards` は落ちておらず、`Modes` に `org/app` は無く `org/board` はあり、`Errors` の 1 件目の文字列に `ListLabels` と `org/app` を含む

#### Scenario: リポジトリのエラーは番号を持つエラーより前に並ぶ
- **WHEN** `org/app` の `ListLabels` と `ViewIssue` の両方が失敗する `GHClient` で `Fetch` を呼ぶ
- **THEN** `Errors` の 1 件目の文字列に `ListLabels`、2 件目の文字列に `ViewIssue` を含む

#### Scenario: ctx のキャンセルは部分結果を返さない
- **WHEN** search は成功し（`SearchPRs` が結果を返す直前に親 `ctx` の `cancel()` を呼ぶ）、`ViewIssue` が `ctx.Done()` まで待ってから `ctx.Err()` を返す `GHClient` に対して `Fetch` を呼ぶ
- **THEN** エラーが返り、`errors.Is(err, context.Canceled)` が真で、`Result` は nil である
