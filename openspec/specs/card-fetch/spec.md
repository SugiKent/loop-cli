# card-fetch Specification

## Purpose
TBD - created by archiving change s07-fetch-cards. Update Purpose after archive.
## Requirements
### Requirement: Fetch は設定リポジトリに対して search を 2 回だけ実行し分類済みの Card 群を返す
`internal/fetch` は `Result { Cards []model.Card; Errors []error }` と、関数 `Fetch(ctx context.Context, client gh.GHClient, repos []string) (*Result, error)` を MUST 提供する。`repos` は `owner/name` の列（s02 の `Config.Repos[].Name`。`Fetch` は `internal/config` を import しない）。
`Fetch` は 1 回の呼び出しで `client.SearchIssues(ctx, repos)` と `client.SearchPRs(ctx, repos)` をそれぞれ 1 回だけ実行し（D-001「1 回の更新で行う呼び出し」）、得た open issue / open PR を `model.IssueFromSearch` / `model.PRFromSearch` で `model` の型に写し、Requirement「Fetch は open の全 issue / 全 PR の詳細を取得する」の詳細を入れ、Requirement「PR は title と本文のパースで同一リポジトリの Issue に紐づく」で `model.Card` を組み立て、各 Card を s05 の `classify.Card` に通して `Result.Cards` に入れる。`Result.Cards` の各要素は `Card.Result` / `Issue.Result` / `PRs[i].Result` が埋まった状態で返る（s08 は分類を呼び直さない）。
`Cards` の並びは、issue を持つカードを `SearchIssues` の返却順、続けて PR 単独カードを `SearchPRs` の返却順とする。タブ内の並び替えは s08 が `Card.Result.Tab` / `Priority` で行う。
search 結果の全 issue と全 PR は、それぞれちょうど 1 枚の Card に含まれる。`Fetch` は issue / PR を黙って落とさない。

#### Scenario: example の fixture から Card 2 枚が返る
- **WHEN** `gh.NewFake("../gh/testdata/fixtures/example")` を `client` に、`[]string{"org/app"}` を `repos` に渡して `Fetch` を呼ぶ（`example` は issue 108（`stage:propose` + `question`）、issue 140（ラベル無し）、PR 131（`propose` + `question`、本文に `Closes #108`）を持つ）
- **THEN** エラー無しで `Result` が返り、`Errors` は空、`Cards` は 2 枚で、1 枚目は `Issue.Number` 108 かつ `PRs` が PR 131 の 1 件、2 枚目は `Issue.Number` 140 かつ `PRs` が空である。1 枚目の `Card.Result.Situation` は `A`（`PRs[0].Result.Situation` が `A`、`Issue.Result.Situation` が `in-progress`）、2 枚目の `Card.Result.Situation` は `E` である

#### Scenario: search は 1 回ずつしか呼ばれない
- **WHEN** `SearchIssues` / `SearchPRs` の呼び出し回数を数える `GHClient` で `Fetch` を呼ぶ
- **THEN** どちらもちょうど 1 回呼ばれる

#### Scenario: 全 issue と全 PR がちょうど 1 枚の Card に含まれる
- **WHEN** `internal/gh/testdata/fixtures/` 直下の各 `<alias>` について `gh.NewFake` で `Fetch` を呼ぶ
- **THEN** エラー無しで返り、`Errors` は空で、`search-issues.json` の各 issue 番号は `Cards[].Issue` にちょうど 1 回、`search-prs.json` の各 PR 番号は `Cards[].PRs` にちょうど 1 回現れる（`Situation` の期待値は s05 の期待値表が持ち、このテストでは見ない）

### Requirement: PR は title と本文のパースで同一リポジトリの Issue に紐づく
`internal/fetch` は `LinkedIssue(title, body string) (n int, ok bool)` を MUST 提供し、D-001「Issue と PR の紐づけ」の 1（PR 側）を実装する。search 結果の title / body だけで済み、追加呼び出しをしない。
1. title の先頭（空白を除く）が `[propose]` / `[apply]` / `[archive]` のいずれかで、その直後（空白 0 個以上）に `#<n>`（`<n>` は 10 進整数）が続けば `n, true`
2. 1 に当たらなければ、本文中の `Refs #<n>` または `Closes #<n>` を本文の先頭から探し、最初に見つかった番号を `n, true` として返す。`Refs` / `Closes` は大文字小文字を区別せず、単語として現れる（直前の文字が英数字である `xRefs` は当たらない）ものだけを採り、キーワードと `#` の間の空白は 1 個以上とする
3. どちらも無ければ `0, false`

`Fetch` は各 open PR について `LinkedIssue(Title, Body)` を呼び、`ok` なら PR と同じ `Repo` の open issue のうち `Number` が `n` のものを探し、あればその issue の Card の `PRs` に PR を追加する。リポジトリが違えば番号が同じでも紐づけない。
Issue 側（cross-reference）による補完は s17 が担当し、この change では `CrossReferencedPRs` を呼ばない。

#### Scenario: title の段階と番号で紐づく
- **WHEN** `LinkedIssue("[propose] #108 選択 UI をモーダル化する", "")` を呼ぶ
- **THEN** `108, true` が返る

#### Scenario: 本文の Closes で紐づく
- **WHEN** `LinkedIssue("propose: ログインのセッション仕様", "issue #108 の提案。\n\nCloses #108")` を呼ぶ
- **THEN** `108, true` が返る

#### Scenario: 本文の Refs で紐づく
- **WHEN** `LinkedIssue("docs: README を直す", "Refs #48")` を呼ぶ
- **THEN** `48, true` が返る

#### Scenario: title が本文より優先される
- **WHEN** `LinkedIssue("[apply] #48 監視ツールを導入する", "Refs #12\nCloses #48")` を呼ぶ
- **THEN** `48, true` が返る

#### Scenario: 番号だけの言及は紐づけない
- **WHEN** `LinkedIssue("fix typo", "#108 と同じ問題")` を呼ぶ
- **THEN** `0, false` が返る

#### Scenario: 複数リポジトリで同じ番号は混ざらない
- **WHEN** `search-issues.json` に `org/app` の issue 12 と `org/web` の issue 12（どちらもラベル無し）、`search-prs.json` に `org/web` の PR 30（title `[propose] #12 ログイン画面`、`propose` ラベル、本文 1 行目が `未確定の判断: 1 件`）がある fixture で `Fetch` を呼ぶ
- **THEN** `Cards` は 2 枚で、`org/web` の issue 12 の Card の `PRs` は PR 30 の 1 件、`org/app` の issue 12 の Card の `PRs` は空である

### Requirement: Issue に紐づかない PR は PR 単独の Card になる
`Fetch` は、`LinkedIssue` が `ok` を返さない PR、または `ok` でも同じリポジトリの open issue に番号 `n` が無い PR（issue が閉じている、`n` が PR の番号、等）を、`Issue` が nil で `PRs` がその PR 1 件だけの Card に MUST する。mvp.md「PR 単独の行が出るのは、Issue に紐づかない `docs` PR と『その他』バケットの PR だけ」は定常状態の説明であり、`Fetch` は紐づかない PR をラベルで選別せず、すべて PR 単独の Card にする（消えて見えなくなる項目を作らない）。

#### Scenario: docs PR は PR 単独の Card
- **WHEN** `search-prs.json` に `Labels` が `docs`、title `docs: README を直す`、本文に `Refs` / `Closes` が無い PR 60 がある fixture で `Fetch` を呼ぶ
- **THEN** `Issue` が nil で `PRs` が PR 60 の 1 件の Card があり、その `Card.Result.Situation` は `G` である

#### Scenario: ラベル無しの PR はその他バケットの PR 単独 Card
- **WHEN** `Labels` が空、本文に `Refs` / `Closes` が無い PR 61 がある fixture で `Fetch` を呼ぶ
- **THEN** `Issue` が nil で `PRs` が PR 61 の 1 件の Card があり、`Card.Result.Situation` は `other` である

#### Scenario: 紐づけ先の issue が open issue に無い PR は落ちない
- **WHEN** ラベル無しで title が `[propose] #999`、`search-issues.json` に issue 999 が無い PR 62 がある fixture で `Fetch` を呼ぶ
- **THEN** `Issue` が nil で `PRs` が PR 62 の 1 件の Card がある

### Requirement: Card 内の PR は段階順・番号順に並ぶ
`Fetch` は各 Card の `PRs` を、`model.PRStages(Labels)` の先頭の段階が `propose` → `apply` → `archive` の順、段階ラベルが無い PR はその後、同じ段階の中では `Number` の昇順に MUST 並べる（mvp.md「カード詳細」: 紐づく PR を段階順に並べる）。同一 Issue に同段階の open PR が複数あれば全部を `PRs` に入れる。この並びは s05 `classify.Card` の同点判定（`PRs` の並び順で先のもの）に使われる。
search は open PR しか返さないので、`Canonical`（同段階の `MERGED` PR のうち最新）は P1 では立たない。merge 済み PR を持ち込むのは s17 の cross-reference である。

#### Scenario: 段階順に並ぶ
- **WHEN** issue 108 に紐づく PR が `archive` の 151、`propose` の 131、`apply` の 140 の順で `search-prs.json` にある fixture で `Fetch` を呼ぶ
- **THEN** issue 108 の Card の `PRs` は番号順に 131（propose）、140（apply）、151（archive）である

#### Scenario: 同段階の open PR が複数あれば番号順に全部入る
- **WHEN** issue 108 に紐づく `propose` + `question` の PR 131（最新コメントが AI）と `propose` の PR 140（本文 1 行目が `未確定の判断: 1 件`）の 2 件が `search-prs.json` に 140、131 の順である fixture で `Fetch` を呼ぶ
- **THEN** issue 108 の Card の `PRs` は 131、140 の順で 2 件あり、どちらも `Canonical` は false、`Card.Result.Situation` は `A`（PR 131）である

### Requirement: gh 呼び出しにはタイムアウトを付け、詳細取得は並行する
`Fetch` は `client` の各メソッド呼び出しを、引数の `ctx` から `context.WithTimeout` で派生させた 30 秒の期限付き `ctx` で MUST 実行する（s03 design.md 未決事項「サブプロセスのタイムアウト」で s07 に委ねられた値。`Client` は `ctx` に従うだけで自前の期限を持たない）。期限は定数 `CallTimeout` として `internal/fetch` が公開する。
詳細取得（`ViewIssue` / `ViewPR` / `ViewPRMergeState` / `ReviewThreads`）は並行して実行してよい。並行度の上限と search 2 回の直列は design.md 未決事項の既定値（上限 4、search は順に 1 回ずつ）であり、この Requirement は固定しない。並行しても `Result.Cards` の並びと各 Card の `PRs` の並びは Requirement のとおり決定的でなければならない。

#### Scenario: 親 ctx の期限が伝わる
- **WHEN** `SearchIssues` が `ctx.Done()` まで待ってから `ctx.Err()` を返す `GHClient` に対して、50 ミリ秒の期限付き `ctx` で `Fetch` を呼ぶ
- **THEN** 50 ミリ秒程度でエラーが返り、`errors.Is(err, context.DeadlineExceeded)` が真である

### Requirement: search の失敗と ctx の中断はエラー、詳細取得の失敗は部分失敗として Card を残す
`Fetch` は次の失敗の扱いを MUST 守る。
- `SearchIssues` または `SearchPRs` がエラーを返したら、そのエラーを（どちらの search かが分かるよう `search issues: ` / `search prs: ` を前置して）返し、`Result` は nil。D-002「失敗時は前回結果を維持してエラーを表示」の「前回結果の維持」と表示は s08 / s13 が行う
- 詳細取得の途中で引数の `ctx` がキャンセルまたは期限切れになったら、`ctx.Err()` を含むエラーを返し、`Result` は nil（途中までの Card でスナップショットを上書きしないため）
- 詳細取得 1 件が上記以外の理由で失敗したら、その issue / PR を落とさず、対応する詳細を nil のまま Card に残し、`Result.Errors` に「メソッド名・`owner/name`・番号」を含むエラー（元のエラーを `%w` で包む）を追加する。他の詳細取得は続ける。`Errors` の並びは issue のエラーを (repo, number) 順、続けて PR のエラーを (repo, number, メソッド) 順（メソッドは `ViewPR` → `ViewPRMergeState` → `ReviewThreads` の順）とする。1 つの PR が複数の詳細で失敗すると同じ (repo, number) のエラーが 3 件まで並ぶので、その中の順序もメソッドで決める（並行実行の完了順に依存しない）
- 詳細取得がすべて成功すれば `Errors` は空（nil または長さ 0）

#### Scenario: search issues の失敗
- **WHEN** `search-issues.json` が無い fixture ディレクトリで `Fetch` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `search issues` と `search-issues.json` を含み、`Result` は nil である

#### Scenario: 1 つの PR の詳細がすべて失敗しても Card を落とさない
- **WHEN** `search-prs.json` に `propose` + `question` の PR 131（本文 `Closes #108`）があるが `pr-131.json` と `pr-131-review-threads.json` が無く、issue 108 は `issue-108.json` がある fixture で `Fetch` を呼ぶ
- **THEN** エラー無しで `Result` が返り、issue 108 の Card の `PRs` に PR 131 があって `Comments` と `MergeState` と `ReviewThreads` はいずれも nil、`PRs[0].Result.Situation` は `other`、`Errors` は 3 件で、それぞれの文字列に `ViewPR` / `ViewPRMergeState` / `ReviewThreads` のいずれかと `org/app` と `131` を含む。issue 108 の `Comments` は埋まっている

#### Scenario: ctx のキャンセルは部分結果を返さない
- **WHEN** search は成功し（`SearchPRs` が結果を返す直前に親 `ctx` の `cancel()` を呼ぶ）、`ViewIssue` が `ctx.Done()` まで待ってから `ctx.Err()` を返す `GHClient` に対して `Fetch` を呼ぶ
- **THEN** エラーが返り、`errors.Is(err, context.Canceled)` が真で、`Result` は nil である

### Requirement: Fetch は open の全 issue / 全 PR の詳細を取得する
`Fetch` は、search で得た **すべての** open issue と open PR について、ラベル・本文・`IsDraft` を問わず詳細を MUST 取得する。取得しない対象を選ぶ条件を持たない。

1. すべての open issue → `client.ViewIssue` を呼び、`Comments` を `model.CommentFrom` で `Issue.Comments` に入れる
2. すべての open PR → `client.ViewPR` を呼び、`Comments` を `PR.Comments` に入れる
3. すべての open PR → `client.ViewPRMergeState` を呼び、`PR.MergeState` に入れる
4. すべての open PR → `client.ReviewThreads` を呼び、`PR.ReviewThreads` に入れる

したがって `Issue.Comments` / `PR.MergeState` / `PR.ReviewThreads` が nil で返るのは、その詳細取得が失敗したときだけである（Requirement「search の失敗と ctx の中断はエラー、詳細取得の失敗は部分失敗として Card を残す」）。取得に成功して中身が空なら、長さ 0 の非 nil が入る。表示側（s09 `card-detail`）はこの nil を「取得失敗」として出す。

search 結果の `Labels` / `Body` / `IsDraft` は詳細で上書きしない。この Requirement は分類の入力を変えない。`classify.Issue` がコメントを読む分岐は `question` ラベルで、`classify.isC` は「段階ラベル 1 件以上・`question` 無し・未確定 0 件」で、`classify.isD` は `apply` ラベルでそれぞれ守られており、これまで詳細を取っていなかった対象は、埋まった詳細を読む前に分岐から外れる。

#### Scenario: 全 issue に ViewIssue が呼ばれる
- **WHEN** `example` で `Fetch` を呼んだ後に `Fake.Calls` を見る（`example` は `question` 付きの issue 108 と、ラベル無しの issue 140 を持つ）
- **THEN** `Method` が `ViewIssue` の `Call` は `Number` 108 と 140 の 2 件で、issue 108 の `Comments` は `issue-108.json` のコメント 2 件、issue 140 の `Comments` は `issue-140.json` のコメント（長さ 0 なら非 nil の長さ 0）である

#### Scenario: question 付き PR でも merge 状態と review thread を取る
- **WHEN** `example` で `Fetch` を呼ぶ（PR 131 は `propose` + `question`、`pr-131.json` は `mergeable: UNKNOWN`、`pr-131-review-threads.json` がある）
- **THEN** PR 131 の `Comments` は `pr-131.json` のコメント、`MergeState` は non-nil で `Mergeable` が `UNKNOWN`、`ReviewThreads` は `pr-131-review-threads.json` の内容で、いずれも nil ではない

#### Scenario: 詳細が埋まっても分類結果は変わらない
- **WHEN** `Labels` が `propose`、`Body` の 1 行目が `未確定の判断: 2 件` の PR 132（`pr-132.json` は `mergeable: MERGEABLE` で AI のコメント 1 件、`pr-132-review-threads.json` は空）がある fixture で `Fetch` を呼ぶ
- **THEN** `Errors` は空で、PR 132 の `MergeState` は non-nil、`ReviewThreads` は長さ 0 の非 nil で、`Result.Situation` は `other` である（未確定が 0 件でないため `isC` は `MergeState` を読む前に false になる）

#### Scenario: question 無しの issue のコメントは分類に影響しない
- **WHEN** `Labels` が空で `Comments` の末尾が人である issue 140 を含む fixture で `Fetch` を呼ぶ
- **THEN** issue 140 の `Comments` は埋まっており、`Result.Situation` は `E` である（`question` が無いので規則 4 の分岐に入らない）

