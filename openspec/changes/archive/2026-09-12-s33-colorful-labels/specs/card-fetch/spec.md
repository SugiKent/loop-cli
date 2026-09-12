## MODIFIED Requirements

### Requirement: Fetch は search を 2 回だけ実行し、方式を判定して分類済みの Card 群を返す
`internal/fetch` は `Result { Cards []model.Card; Modes map[string]model.Mode; LabelColors map[string]map[string]string; Errors []error }` と、関数 `Fetch(ctx context.Context, client gh.GHClient, repos []string, now time.Time, grace time.Duration) (*Result, error)` を MUST 提供する。`repos` は `owner/name` の列（s02 の `Config.Repos[].Name`。`Fetch` は `internal/config` を import しない）。運用方式は引数で受け取らず、Requirement「Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する」のとおり `Fetch` が判定する。`LabelColors` は同じ Requirement が作るリポジトリごとのラベル色で、分類には使わない。`now` は分類の基準時刻、`grace` は「その他」を進行中に置く猶予（s30-other-grace）で、`Fetch` は壁時計を読まずにこの 2 つを `classify.Card` へそのまま渡す（`cmd/loop-cli` は更新のたびに現在時刻と `Config.OtherGraceMin` 分を渡す）。
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
- **WHEN** `Labels` が空、本文に `Refs` / `Closes` が無い PR 61（`UpdatedAt` が `2026-09-04T10:00:00Z`）を持つ fixture で、`now` を `2026-09-04T10:10:00Z`、`grace` を 30 分にして `Fetch` を呼ぶ
- **THEN** PR 61 の Card の `Card.Result.Situation` は `in-progress`、`Summary` は `PR #61 はどの局面にも当たらない（更新から 30m は様子見）` である

#### Scenario: 猶予以上経った now ではその他に戻る
- **WHEN** 同じ fixture で、`now` を `2026-09-04T10:31:00Z`、`grace` を 30 分にして `Fetch` を呼ぶ
- **THEN** PR 61 の Card の `Card.Result.Situation` は `other` である

#### Scenario: search は 1 回ずつしか呼ばれない
- **WHEN** `SearchIssues` / `SearchPRs` の呼び出し回数を数える `GHClient` で `Fetch` を呼ぶ
- **THEN** どちらもちょうど 1 回呼ばれる

#### Scenario: 全 issue と全 PR がちょうど 1 枚の Card に含まれる
- **WHEN** `internal/gh/testdata/fixtures/` 直下の各 `<alias>` について `gh.NewFake` で `Fetch` を呼ぶ
- **THEN** エラー無しで返り、`Errors` は空で、`search-issues.json` の各 issue 番号は `Cards[].Issue` にちょうど 1 回、`search-prs.json` の各 PR 番号は `Cards[].PRs` にちょうど 1 回現れる（`Situation` の期待値は s05 の期待値表が持ち、このテストでは見ない）

### Requirement: Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する
`Fetch` は 1 回の呼び出しで、`repos` の各リポジトリについて `client.ListLabels(ctx, repo)` を MUST ちょうど 1 回実行する。取得したラベル一覧を `model.ModeFromLabels` に渡し、判定できたリポジトリだけを `Result.Modes` に入れる。判定できなかったリポジトリ（`stage:todo` も `To Do` も持たない）と、`ListLabels` が失敗したリポジトリは `Result.Modes` に入れない。

同じラベル一覧から「ラベル名 → 色」の表を作り、`Result.LabelColors[<repo>]` に入れる。色は `gh label list` が返す値（`#` の無い 16 進 6 桁）をそのまま写す。`ListLabels` が失敗したリポジトリは `LabelColors` にも入れない。方式を判定できなかったリポジトリは `Modes` に入らないが `LabelColors` には入る（色は方式の判定と関係が無く、ラベルの表示は方式が決まらないリポジトリでも行う）。ラベル一覧のそれ以外の内容（説明）は `Result` に持たせない（`L` のラベル一覧は `internal/ui` が自分で取り直す。`label-picker`「`L` は画面の対象のラベル一覧画面を開く」）。

方式は毎回の取得で判定し直す。`Fetch` は前回の結果を持たず、設定ファイルにもスナップショットにも書かない。GitHub 側でラベルを足す・消すと、次の `R` または自動更新の後の `Result.Modes` が変わる（loop-cli の再起動は要らない）。色も同じで、GitHub 側で色を変えると次の取得の後の `Result.LabelColors` が変わる。

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

#### Scenario: ラベル色の表が返る
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` を `client` に、`[]string{"org/app"}` を `repos` に渡して `Fetch` を呼ぶ（`labels.json` は `stage:propose` を `0e8a16`、`question` を `d876e3` として持つ）
- **THEN** `Result.LabelColors["org/app"]["stage:propose"]` は `0e8a16`、`["question"]` は `d876e3` である

#### Scenario: 方式を判定できないリポジトリにも色は返る
- **WHEN** `stage:todo` も `To Do` も含まない 1 件のラベル（`bug`、色 `d73a4a`）だけを返す `GHClient` で、`repos` に `org/app` を渡して `Fetch` を呼ぶ
- **THEN** `Result.Modes` は空で、`Result.LabelColors["org/app"]["bug"]` は `d73a4a` である

#### Scenario: ListLabels が失敗したリポジトリは色の表にも入らない
- **WHEN** `org/app` の `ListLabels` だけが失敗し `org/board` は成功する `GHClient` で `Fetch` を呼ぶ
- **THEN** `Result.LabelColors` に `org/app` は無く `org/board` はあり、`Cards` は落ちておらず、`Errors` の 1 件目の文字列に `ListLabels` と `org/app` を含む
