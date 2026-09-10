## MODIFIED Requirements

### Requirement: Fetch は設定リポジトリに対して search を 2 回だけ実行し分類済みの Card 群を返す
`internal/fetch` は `Result { Cards []model.Card; Errors []error }` と、関数 `Fetch(ctx context.Context, client gh.GHClient, repos []string, modes map[string]model.Mode) (*Result, error)` を MUST 提供する。`repos` は `owner/name` の列（s02 の `Config.Repos[].Name`。`Fetch` は `internal/config` を import しない）。`modes` はリポジトリ名から運用方式を引く表で、`repos` の各要素に対応する値を呼び出し側が入れる。表に無いリポジトリ、および `modes` が nil のときは `model.Mode` のゼロ値（`sdd`）になる。
`Fetch` は 1 回の呼び出しで `client.SearchIssues(ctx, repos)` と `client.SearchPRs(ctx, repos)` をそれぞれ 1 回だけ実行し（D-001「1 回の更新で行う呼び出し」）、得た open issue / open PR を `model.IssueFromSearch` / `model.PRFromSearch` で `model` の型に写し、各 issue / PR の `Mode` に `modes[Repo]` を入れ、Requirement「Fetch は open の全 issue / 全 PR の詳細を取得する」の詳細を入れ、Requirement「PR は title と本文のパースで同一リポジトリの Issue に紐づく」で `model.Card` を組み立て、各 Card を s05 の `classify.Card` に通して `Result.Cards` に入れる。`Result.Cards` の各要素は `Card.Result` / `Issue.Result` / `PRs[i].Result` が埋まった状態で返る（s08 は分類を呼び直さない）。
`Cards` の並びは、issue を持つカードを `SearchIssues` の返却順、続けて PR 単独カードを `SearchPRs` の返却順とする。タブ内の並び替えは s08 が `Card.Result.Tab` / `Priority` で行う。
search 結果の全 issue と全 PR は、それぞれちょうど 1 枚の Card に含まれる。`Fetch` は issue / PR を黙って落とさない。

#### Scenario: example の fixture から Card 2 枚が返る
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` を `client` に、`[]string{"org/app"}` を `repos` に、nil を `modes` に渡して `Fetch` を呼ぶ（`example` は issue 108（`stage:propose` + `question`）、issue 140（ラベル無し）、PR 131（`propose` + `question`、本文に `Closes #108`）を持つ）
- **THEN** エラー無しで `Result` が返り、`Errors` は空、`Cards` は 2 枚で、1 枚目は `Issue.Number` 108 かつ `PRs` が PR 131 の 1 件、2 枚目は `Issue.Number` 140 かつ `PRs` が空である。1 枚目の `Card.Result.Situation` は `A`（`PRs[0].Result.Situation` が `A`、`Issue.Result.Situation` が `in-progress`）、2 枚目の `Card.Result.Situation` は `E` である

#### Scenario: modes で指定した方式が issue と PR に入る
- **WHEN** issue-label-driven の fixture を `client` に、`map[string]model.Mode{"org/board": model.ModeLabel}` を `modes` に渡して `Fetch` を呼ぶ
- **THEN** `Cards` の各 `Issue.Mode` と各 `PRs[i].Mode` が `label` である

#### Scenario: modes に無いリポジトリは sdd になる
- **WHEN** `map[string]model.Mode{"org/board": model.ModeLabel}` を `modes` に渡し、fixture が `org/app` の issue / PR を返す状態で `Fetch` を呼ぶ
- **THEN** `org/app` の issue / PR の `Mode` はゼロ値（`sdd`）である

#### Scenario: search は 1 回ずつしか呼ばれない
- **WHEN** `SearchIssues` / `SearchPRs` の呼び出し回数を数える `GHClient` で `Fetch` を呼ぶ
- **THEN** どちらもちょうど 1 回呼ばれる

#### Scenario: 全 issue と全 PR がちょうど 1 枚の Card に含まれる
- **WHEN** `internal/gh/testdata/fixtures/` 直下の各 `<alias>` について `gh.NewFake` で `Fetch` を呼ぶ
- **THEN** エラー無しで返り、`Errors` は空で、`search-issues.json` の各 issue 番号は `Cards[].Issue` にちょうど 1 回、`search-prs.json` の各 PR 番号は `Cards[].PRs` にちょうど 1 回現れる（`Situation` の期待値は s05 の期待値表が持ち、このテストでは見ない）

### Requirement: PR は title と本文のパースで同一リポジトリの Issue に紐づく
紐づけの規則は s05 `card-model`「PR が紐づく issue 番号のパースを model が持つ」の `model.LinkedIssue(title, body string) (n int, ok bool)` が MUST 持ち、D-001「Issue と PR の紐づけ」の 1（PR 側）を実装する。search 結果の title / body だけで済み、追加呼び出しをしない。`internal/fetch` は同名の関数を公開せず、`model.LinkedIssue` を呼ぶ（分類器も同じ関数を使うため、規則を 2 か所に置かない）。
1. title の先頭（空白を除く）が `[propose]` / `[apply]` / `[archive]` のいずれかで、その直後（空白 0 個以上）に `#<n>`（`<n>` は 10 進整数）が続けば `n, true`
2. 1 に当たらなければ、本文中の `Refs #<n>` または `Closes #<n>` を本文の先頭から探し、最初に見つかった番号を `n, true` として返す。`Refs` / `Closes` は大文字小文字を区別せず、単語として現れる（直前の文字が英数字である `xRefs` は当たらない）ものだけを採り、キーワードと `#` の間の空白は 1 個以上とする
3. どちらも無ければ `0, false`

`Fetch` は各 open PR について `model.LinkedIssue(Title, Body)` を呼び、`ok` なら PR と同じ `Repo` の open issue のうち `Number` が `n` のものを探し、あればその issue の Card の `PRs` に PR を追加する。リポジトリが違えば番号が同じでも紐づけない。

#### Scenario: title の段階と番号で紐づく
- **WHEN** `model.LinkedIssue("[propose] #108 選択 UI をモーダル化する", "")` を呼ぶ
- **THEN** `108, true` が返る

#### Scenario: 本文の Closes で紐づく
- **WHEN** `model.LinkedIssue("propose: ログインのセッション仕様", "issue #108 の提案。\n\nCloses #108")` を呼ぶ
- **THEN** `108, true` が返る

#### Scenario: 本文の Refs で紐づく
- **WHEN** `model.LinkedIssue("docs: README を直す", "Refs #48")` を呼ぶ
- **THEN** `48, true` が返る

#### Scenario: title が本文より優先される
- **WHEN** `model.LinkedIssue("[apply] #48 監視ツールを導入する", "Refs #12\nCloses #48")` を呼ぶ
- **THEN** `48, true` が返る

#### Scenario: 番号だけの言及は紐づけない
- **WHEN** `model.LinkedIssue("fix typo", "#108 と同じ問題")` を呼ぶ
- **THEN** `0, false` が返る

#### Scenario: 複数リポジトリで同じ番号は混ざらない
- **WHEN** `search-issues.json` に `org/app` の issue 12 と `org/web` の issue 12（どちらもラベル無し）、`search-prs.json` に `org/web` の PR 30（title `[propose] #12 ログイン画面`、`propose` ラベル、本文 1 行目が `未確定の判断: 1 件`）がある fixture で `Fetch` を呼ぶ
- **THEN** `Cards` は 2 枚で、`org/web` の issue 12 の Card の `PRs` は PR 30 の 1 件、`org/app` の issue 12 の Card の `PRs` は空である
