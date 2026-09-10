## MODIFIED Requirements

### Requirement: ヘッダはリポジトリ・番号・いま人が何をすべきか・現在の段階・バッジ・depends on を出す
カード詳細画面の `View` はヘッダ領域の先頭に、詳細の対象の Card について MUST 次の行をこの順で出す（mvp.md「カード詳細」のヘッダ）。行番号は固定せず順序だけを定める（s17 が ADDED で行を足せるようにするため）。方式は `Options.Modes`（設定由来。`queue-screen`「リポジトリごとの運用方式を設定から受け取る」）に対象の `Repo` を引いて決め、Card の中の値からは決めない。
- `<Repo> #<Number>  <Title>`（`Card.Issue` の値）
- `Card.Result.Summary`（空なら出さない。s05 が定めた「いま人が何をすべきか」。mvp.md「カードは『いま人が何をすべきか』を 1 行目に出す」。s08 のプレビューはこれを出さず、この change のヘッダに委ねている）
- `段階: ` に続けて `model.IssueStages(mode, Issue.Labels)` の段階ラベルを空白区切りで並べる。1 件も無ければ `段階なし`。2 件以上あればすべて並べる（異常の状態を隠さない）。続けてバッジを出す。`sdd`（ゼロ値を含む）なら `[blocked]` / `[wip]` / `[question]` の順、`label` なら `[blocked]` / `[question]` の順で、`Labels` にあるものだけ出す（`label` の方式に `wip` ラベルは無く、作業中は段階ラベル `In Progress` が示す）
- `Issue.Body` の中に `depends on #<n>`（大文字小文字を区別しない。`<n>` は 10 進整数）が 1 つ以上あれば `depends on: #<n> #<m> …` を出現順に出す。無ければこの行を出さない（書式は design.md 未決事項の既定値）

ヘッダ領域の各行（上の各行と Requirement「紐づく PR 一覧は段階順に 1 行ずつ出し、選択中の PR に印を付ける」の PR 一覧の行、PR 詳細のヘッダ領域の行）は、表示幅が端末の幅を超えれば端末の幅に切り詰めて末尾を `…` にする（design.md 未決事項の既定値）。
「この段階に入ってからの経過時間」と段階の変遷タイムラインはデータ源が REST の timeline（D-001「ラベル変遷」。`GHClient.LabelTimeline`）であり、s17 が担当する。この change はヘッダに枠を確保せず、s17 が ADDED で行を足す。

#### Scenario: issue 108 のヘッダ
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、`example` の issue 108 の Card（`Labels` が `stage:propose` と `question`、`Card.Result.Summary` が `PR #131 の質問に答える`）の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `org/app #108`、issue 108 の `Title`、`PR #131 の質問に答える`、`段階: stage:propose`、`[question]` がこの順で含まれ、`[blocked]` と `[wip]` と `depends on:` は含まれない

#### Scenario: 長いヘッダ行は幅で切り詰める
- **WHEN** 幅 40・高さ 40 のサイズメッセージを与えた後、`Title` が表示幅 60 の issue の Card の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `org/app #` で始まる行の表示幅は 40 以下で、末尾が `…` である

#### Scenario: 段階ラベルが無い issue と depends on
- **WHEN** `Labels` が空、`Body` が `集計が遅い。\n\ndepends on #12\nDepends on #34` の issue の Card の詳細を開き、`View` を読む
- **THEN** `段階なし` と `depends on: #12 #34` が含まれる

#### Scenario: 段階ラベルが 2 つある issue は両方出す
- **WHEN** `Labels` が `stage:propose` と `stage:apply` と `blocked` と `wip` の issue の Card の詳細を開き、`View` を読む
- **THEN** `段階: stage:propose stage:apply` と `[blocked]` と `[wip]` がこの順で含まれる

#### Scenario: label 方式の issue の段階とバッジ
- **WHEN** `Options.Modes` が `org/board` を `label` にした `Model` で、`org/board` の `Labels` が `In Progress` と `question` の issue の Card の詳細を開き、`View` を読む
- **THEN** `段階: In Progress` と `[question]` が含まれ、`[wip]` は含まれない

#### Scenario: label 方式では sdd の段階ラベルを段階行に出さない
- **WHEN** 同じ `Model` で、`org/board` の `Labels` が `stage:propose` の issue の Card の詳細を開き、`View` を読む
- **THEN** `段階なし` が含まれる

### Requirement: 紐づく PR 一覧は段階順に 1 行ずつ出し、選択中の PR に印を付ける
カード詳細画面の `View` はヘッダ領域のヘッダ行の下に、`Card.PRs`（s07 が段階順 → 番号順に並べたもの）を 1 件 1 行で MUST 出す。方式が `sdd`（ゼロ値を含む）なら、`propose` / `apply` / `archive` の各段階について、`model.PRStages(mode, Labels)` の先頭がその段階である PR の行を並べ、その段階の PR が 1 件も無ければ `[<段階>] なし` の 1 行を出す（mvp.md「`[apply] なし`」）。段階ラベルの無い PR は 3 段階の後に `[-]` を段階名の代わりにして並べる。
方式が `label` なら段階の見出しを持たず、`PRs` の並び順のまま各 PR の行を `[-]` で始めて並べる（`label` の PR に段階ラベルは付かないので、`なし` の 3 行は情報を持たない）。`PRs` が空なら PR の行を 1 行も出さない。方式の引き方はヘッダの Requirement と同じである。
各行の書式は `[<段階>] PR#<Number> <状態>` に続けて次を空白区切りで並べる。列は方式で変えない。
- 状態: `State` が `OPEN` なら `open`、`MERGED` なら `merged`、`CLOSED` なら `closed`。`Canonical` が true なら状態の直後に `（最新・正本）`（s05「同段階の merge 済み PR は最新が正本」。search は open PR しか返さないので P1 では立たず、s17 の cross-reference で merge 済み PR が入って初めて付く。`label` では段階ラベルが無いので立たない）
- 1 行目判定: `model.ParseUndecided(Body)` が `n, true` なら `未確定 <n> 件`、false なら `1 行目なし`（`label` の PR は 1 行目の規約を持たないので常に `1 行目なし` になる。事実として出し、方式で列を落とさない）
- `labels: <Labels を空白区切り>`
- checks: `MergeState` が nil なら `checks 取得失敗`（s20 は全 PR の merge 状態を取るので、nil は取得に失敗したことを意味する）、non-nil で `classify.ChecksGreen(MergeState)` が true なら `checks 緑`、false なら `checks 緑以外`
- merge 状態: `MergeState` が nil なら `mergeable 取得失敗`、non-nil なら `mergeable <Mergeable>`

`Model` は詳細の中で選択中の PR の添字（初期値 0）を持ち、その PR の行の先頭に `▶` を置く。他の PR の行と `なし` の行は同じ幅の空白で始める。`Tab` で選択を次の PR（`PRs` の並び順）に移し、末尾の次は先頭に戻る。`PRs` が空なら `Tab` は何もしない。
このヘッダ領域はスクロールせず、常に表示される（Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」）。各行は端末の幅で切り詰める（Requirement「ヘッダはリポジトリ・番号・いま人が何をすべきか・現在の段階・バッジ・depends on を出す」）。

#### Scenario: issue 108 の PR 一覧
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、`example` を `Fetch` して得た issue 108 の Card（`PRs` は `propose` + `question` の open PR 131 のみ。本文 1 行目は `issue #108 の提案。`。s20 で全 PR の merge 状態を取るので `MergeState` は non-nil で `Mergeable` が `UNKNOWN`、`StatusCheckRollup` に `StatusContext ci/legacy PENDING` を含む）の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `▶` で始まり `[propose] PR#131 open`、`1 行目なし`、`labels: propose question`、`checks 緑以外`、`mergeable UNKNOWN` をこの順で含む行があり、`[apply] なし` と `[archive] なし` の行がある

#### Scenario: merge 状態の取得に失敗した PR
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、`PRs` が `[propose, OPEN, #131, MergeState nil]` の 1 件だけである手書きの Card の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `[propose] PR#131 open` の行に `checks 取得失敗` と `mergeable 取得失敗` がこの順で含まれる

#### Scenario: 同段階の merge 済み PR は最新に印が付く
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、`PRs` が `[propose, MERGED, #131]`、`[propose, MERGED, #140, Canonical true]`、`[apply, OPEN, #151, Body 1 行目 未確定の判断: 0 件, MergeState{Mergeable: MERGEABLE, StatusCheckRollup: [CheckRun test SUCCESS]}]` の順である Card の詳細を開き、`View` を読む
- **THEN** `[propose] PR#131 merged` の行、`[propose] PR#140 merged（最新・正本）` の行、`[apply] PR#151 open` と `未確定 0 件` と `checks 緑` と `mergeable MERGEABLE` を含む行、`[archive] なし` の行がこの順で含まれ、`PR#131` の行に `（最新・正本）` は無い

#### Scenario: Tab で PR の選択が移る
- **WHEN** 上の Card（PR 3 件）の詳細を開いた `Model` に `Tab` を 3 回与え、各回の後に `View` を読む
- **THEN** `▶` が付く行は順に `PR#140`、`PR#151`、`PR#131` の行である

#### Scenario: 段階ラベルの無い PR は末尾に出る
- **WHEN** `PRs` が `[apply, OPEN, #90]`、`[ラベル無し, OPEN, #61]` の Card の詳細を開き、`View` を読む
- **THEN** `[propose] なし`、`[apply] PR#90 open`、`[archive] なし`、`[-] PR#61 open` の行がこの順で含まれる

#### Scenario: label 方式の PR 一覧は段階の見出しを持たない
- **WHEN** `Options.Modes` が `org/board` を `label` にした `Model` で、`org/board` の Card（`PRs` が `[ラベル無し, OPEN, #61, Body に Closes #12]`、`[question, OPEN, #62]`）の詳細を開き、`View` を読む
- **THEN** `▶` で始まる `[-] PR#61 open` の行と `[-] PR#62 open` の行がこの順で含まれ、`[propose] なし` と `[apply] なし` と `[archive] なし` は含まれない

#### Scenario: label 方式で PR が 1 件も無ければ PR の行を出さない
- **WHEN** 同じ `Model` で、`org/board` の `PRs` が空の Card の詳細を開き、`View` を読む
- **THEN** `[-] PR#` と `[propose] なし` のどちらも含まれない
