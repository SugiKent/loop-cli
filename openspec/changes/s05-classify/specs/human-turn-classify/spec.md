## ADDED Requirements

### Requirement: 分類は純粋関数で、進行中の除外・判定表の順・フォールバックの順に評価する
`internal/classify` は `Issue(is model.Issue) model.Result` と `PR(pr model.PR) model.Result` を MUST 提供する。どちらも入力だけから結果を決め、I/O を行わず、入力を変更しない。評価は次の順で行い、最初に当たった規則で止める（human-turn-signals.md「分類はこの順で評価し、最初に当たった行で止める」）。
1. 「キューに入れないもの」（Requirement「キューに入れないものは進行中にする」の 5 規則）
2. 判定表の行を A → B → C → D → E → F → G の順（issue に当たり得るのは B / E / F、PR に当たり得るのは A / C / D / F / G）
3. フォールバック: open PR は「その他」、issue は `question` と `blocked` の両方があれば「その他」、それ以外は「進行中」（Requirement「どの行にも当たらないものはその他バケットに入れる」）

入力の前提: `Comments` / `MergeState` / `ReviewThreads` の詳細は D-001 の遅延取得に従って s07 が入れる。詳細が nil のときは、その詳細を必要とする条件は成立しない（nil の `Comments` は「最新コメントが AI」も「最新コメントが人」も偽、nil の `MergeState` は「mergeable かつ checks 緑」が偽、nil の `ReviewThreads` は「未 resolve の thread がある」が偽）。空の `Comments`（長さ 0）も同じ扱いにする。
「最新コメント」は `Comments` の末尾の要素である。`PR()` は `State` が `OPEN` の PR に対して使う。`MERGED` / `CLOSED` の PR は Requirement「同段階の merge 済み PR は最新が正本」で扱い、`PR()` を呼んだ場合はゼロ値の `Situation`（`""`）を返す。

#### Scenario: 進行中の除外が判定表より先に評価される
- **WHEN** `Labels` が `stage:propose` と `wip`、`Comments` が nil の `Issue` を `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は AI が作業中` である（`stage:propose` 単独の issue はどの行にも当たらないが、`wip` の規則が先に当たる）

#### Scenario: 詳細が nil なら A は成立しない
- **WHEN** `Labels` が `propose` と `question`、`Comments` が nil、`State` が `OPEN` の `PR` を `PR()` に渡す
- **THEN** `Situation` は `other` である（A の条件は成立せず、C / D / G にも当たらない）

#### Scenario: merge 済み PR は分類しない
- **WHEN** `State` が `MERGED` の `PR` を `PR()` に渡す
- **THEN** `Situation` はゼロ値 `""` である

### Requirement: 局面 A は question 付き open PR で最新コメントが AI のもの
`PR()` は、`Labels` に `question` があり、`Comments` の末尾の `AI` が true の open PR を `Situation` `A` と MUST 判定する（human-turn-signals.md 行 A: open PR、`question` ラベル、最新コメントが routine のもの）。段階ラベル（`propose` / `apply` / `archive`）の有無は問わない。

#### Scenario: 質問に答える PR
- **WHEN** `Labels` が `propose` と `question`、`Comments` が `[AI: "<!-- routine -->\n## Q1. …"]` の open PR を `PR()` に渡す
- **THEN** `Situation` は `A`、`Priority` は 1、`Tab` は `今やる` である

#### Scenario: question があっても最新コメントが人なら A ではない
- **WHEN** `Labels` が `propose` と `question`、`Comments` が `[AI: "<!-- routine -->\n## Q1. …", 人: "Q1: A"]` の open PR を `PR()` に渡す
- **THEN** `Situation` は `in-progress` である（Requirement「キューに入れないものは進行中にする」の規則 3）

### Requirement: 局面 B は question と blocked が付いた open issue で最新コメントが AI のもの
`Issue()` は、`Labels` に `question` と `blocked` の両方があり、`Comments` の末尾の `AI` が true の issue を `Situation` `B` と MUST 判定する（行 B: open issue、`question` ラベル（`blocked` に重ねて付く。`blocked-by: human` の印）、最新コメントが routine のもの）。`question` だけで `blocked` が無い issue は B にしない（Requirement「キューに入れないものは進行中にする」の規則 5）。

#### Scenario: 方針を決める issue
- **WHEN** `Labels` が `stage:propose` と `blocked` と `question`、`Comments` の末尾が `AI: "&lt;!-- routine --&gt;\nblocked-by: human\n次の方針を決めてください"` の issue を `Issue()` に渡す
- **THEN** `Situation` は `B`、`Priority` は 2、`Tab` は `今やる` である

#### Scenario: 段階ラベルが無くても B になる
- **WHEN** `Labels` が `blocked` と `question`、`Comments` の末尾が AI の issue を `Issue()` に渡す
- **THEN** `Situation` は `B` である（E より先に評価される）

### Requirement: 局面 C は段階 PR で未確定 0 件・question 無し・checks 緑・mergeable のもの
`PR()` は次をすべて満たす open PR を `Situation` `C` と MUST 判定する（行 C）。`IsDraft` は見ない（draft の merge 拒否は s14 の merge ガード（human-turn-signals.md 不変条件 5）が持つ）。
- `Labels` に `propose` / `apply` / `archive` のいずれかがある
- `Labels` に `question` が無い
- `model.ParseUndecided(Body)` が `0, true` を返す（1 行目が無い、または N > 0 なら不成立）
- `MergeState` が non-nil で、`Mergeable` が `MERGEABLE`
- `ChecksGreen(MergeState)` が true

`internal/classify` は `ChecksGreen(ms *gh.PRMergeState) bool` を MUST 公開する。`StatusCheckRollup` の全要素が「`Typename` が `CheckRun` で `Conclusion` が `SUCCESS` / `SKIPPED` / `NEUTRAL` のいずれか」または「`Typename` が `StatusContext` で `State` が `SUCCESS`」なら true。要素が 0 件なら true。`ms` が nil なら false。s14 の merge ガードもこの関数を使う。

#### Scenario: merge する PR
- **WHEN** `Labels` が `archive`、`Body` が `未確定の判断: 0 件\n…`、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件の open PR を `PR()` に渡す
- **THEN** `Situation` は `C`、`Priority` は 3、`Tab` は `今やる` である

#### Scenario: 未確定が 1 件以上なら C ではない
- **WHEN** 上と同じで `Body` が `未確定の判断: 2 件\n…` の PR を `PR()` に渡す
- **THEN** `Situation` は `C` ではない

#### Scenario: 1 行目が無ければ C ではない
- **WHEN** 上と同じで `Body` が `issue #108 の提案。\n\nCloses #108` の PR を `PR()` に渡す
- **THEN** `Situation` は `C` ではない

#### Scenario: checks が失敗していれば C ではない
- **WHEN** 上と同じで `StatusCheckRollup` が `CheckRun/SUCCESS` と `StatusContext/FAILURE` の 2 件の PR を `PR()` に渡す
- **THEN** `ChecksGreen` は false、`Situation` は `C` ではない

#### Scenario: mergeable が UNKNOWN なら C ではない
- **WHEN** 上と同じで `Mergeable` が `UNKNOWN` の PR を `PR()` に渡す
- **THEN** `Situation` は `C` ではない

#### Scenario: checks が無ければ緑
- **WHEN** `StatusCheckRollup` が空の `PRMergeState` を `ChecksGreen` に渡す
- **THEN** true が返る

### Requirement: 局面 D は apply PR に未 resolve の review thread があり thread 最終コメントが AI のもの
`PR()` は、`Labels` に `apply` があり、`ReviewThreads` に `IsResolved` が false で `Comments` の末尾の本文が `model.IsAI` で true になる thread が 1 つ以上ある open PR を `Situation` `D` と MUST 判定する（行 D: `apply` PR、未 resolve の review thread、thread 最終コメントが routine のもの）。`question` の有無は問わない（A が先に評価される）。`Comments` が空の thread は数えない。

#### Scenario: レビュー質問に答える PR
- **WHEN** `Labels` が `apply`、`Body` に `未確定の判断` の行が無く、`Comments` が nil、`ReviewThreads` が `[{IsResolved: false, Comments: [{Body: "<!-- routine -->\nこの分岐は残しますか"}]}]` の open PR を `PR()` に渡す
- **THEN** `Situation` は `D`、`Priority` は 1、`Tab` は `今やる` である

#### Scenario: thread が resolve 済みなら D ではない
- **WHEN** 上と同じで thread の `IsResolved` が true の PR を `PR()` に渡す
- **THEN** `Situation` は `other` である（D の条件は成立せず、G にも当たらない）

#### Scenario: thread 最終コメントが人なら D ではない
- **WHEN** 上と同じで thread の `Comments` が `[AI の本文, "残します"]` の PR を `PR()` に渡す
- **THEN** `Situation` は `D` ではない

### Requirement: 局面 E は段階ラベルも blocked も無い open issue
`Issue()` は、`Labels` に `stage:*`（`model.IssueStages` が空）も `blocked` も無い issue を `Situation` `E` と MUST 判定する（行 E: open issue、`stage:*` ラベルなし、`blocked` なし）。`question` だけが付いた issue は規則 5（進行中）が先に当たるので E にならない。

#### Scenario: 着手を承認する issue
- **WHEN** `Labels` が空、`Comments` が nil の issue を `Issue()` に渡す
- **THEN** `Situation` は `E`、`Priority` は 4、`Tab` は `バックログ` である

#### Scenario: 段階ラベル以外のラベルがあっても E
- **WHEN** `Labels` が `bug` と `enhancement` の issue を `Issue()` に渡す
- **THEN** `Situation` は `E` である

#### Scenario: stage:todo が付いていれば E ではない
- **WHEN** `Labels` が `stage:todo` の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress` である（段階ラベルがあるので E の条件は成立しない）

### Requirement: 局面 F は段階ラベルが 2 つ以上
`Issue()` は `model.IssueStages(Labels)` が 2 件以上の issue を、`PR()` は `model.PRStages(Labels)` が 2 件以上の open PR を、`Situation` `F` と MUST 判定する（行 F: 段階ラベルが 2 つ以上。TUI から自動修復はしない）。判定表の順に従い、issue では B、PR では A / C / D が先に当たればそちらになる。

#### Scenario: 段階ラベルが 2 つの issue
- **WHEN** `Labels` が `stage:propose` と `stage:apply` の issue を `Issue()` に渡す
- **THEN** `Situation` は `F`、`Priority` は 0、`Tab` は `異常` である

#### Scenario: 段階ラベルが 2 つの PR
- **WHEN** `Labels` が `propose` と `apply`、`Body` に `未確定の判断` の行が無い open PR を `PR()` に渡す
- **THEN** `Situation` は `F` である

#### Scenario: wip でも段階ラベル 2 つなら F
- **WHEN** `Labels` が `stage:propose` と `stage:apply` と `wip` の issue を `Issue()` に渡す
- **THEN** `Situation` は `F` である（進行中の規則 1 は `IssueStages` が 1 件のときだけ当たる）

#### Scenario: 段階ラベルが 2 つでも question と blocked があれば B
- **WHEN** `Labels` が `stage:propose` と `stage:apply` と `blocked` と `question`、`Comments` の末尾が AI の issue を `Issue()` に渡す
- **THEN** `Situation` は `B` である（表の順で B が先）

### Requirement: 局面 G は docs ラベルの open PR で question 無し
`PR()` は、`Labels` に `docs` があり `question` が無い open PR を `Situation` `G` と MUST 判定する（行 G: open PR、`docs` ラベル、`question` なし。merge しても段階は進まない）。`MergeState` は見ない（merge 可否は s14 のガードが確認する）。

#### Scenario: docs PR
- **WHEN** `Labels` が `docs`、`MergeState` が nil の open PR を `PR()` に渡す
- **THEN** `Situation` は `G`、`Priority` は 5、`Tab` は `今やる` である

#### Scenario: docs PR に question があれば G ではない
- **WHEN** `Labels` が `docs` と `question`、`Comments` が nil の open PR を `PR()` に渡す
- **THEN** `Situation` は `other` である（`question` があるので G の条件は成立せず、`Comments` が nil なので A にも進行中の規則 3 にも当たらない）

### Requirement: キューに入れないものは進行中にする
`Issue()` / `PR()` は次の 5 規則のいずれかに当たる入力を `Situation` `in-progress`、`Tab` `進行中` と MUST 判定し、判定表より先に評価する（human-turn-signals.md「キューに入れないもの（進行中タブに出す）」と、F の縮小理由に書かれた `question` 単独 issue の扱い、「PR の回答は同一セッションが即座に拾う」）。
1. issue: `model.IssueStages(Labels)` が `stage:propose` / `stage:apply` / `stage:archive` のいずれか 1 件だけで、`wip` がある（AI が動いている最中）。段階ラベルが 2 件以上なら当たらず、行 F になる
2. PR: `Labels` に `question` が無く、`Comments` の末尾の `AI` が false（auto-fix が受け取り中）
3. PR: `Labels` に `question` があり、`Comments` の末尾の `AI` が false（回答済み。同一セッションの worker が受け取り中）
4. issue: `Labels` に `question` があり、`Comments` の末尾の `AI` が false（回答済み。sweep が `question` を外して worker を起動し直すのを待っている）
5. issue: `Labels` に `question` があり `blocked` が無い（dispatcher が回収する残骸。異常扱いしない）

5 規則は 1 → 5 の順に評価し、最初に当たった規則が `Summary` を決める（`question` のみで最新コメントが人の issue は規則 4 に当たり、`Summary` は `#<n> は回答済み。sweep 待ち`）。

どの行にも当たらない issue（`stage:*` があり `wip` が無い、`blocked` があり `question` が無い、`stage:todo` のみ等）も `in-progress` にする。ただし `question` と `blocked` の両方がある issue は除く（Requirement「どの行にも当たらないものはその他バケットに入れる」）。

#### Scenario: wip の issue は進行中
- **WHEN** `Labels` が `stage:apply` と `wip` の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Tab` は `進行中`、`Summary` は `#<n> は AI が作業中` である

#### Scenario: question 無し PR で最新コメントが人なら進行中
- **WHEN** `Labels` が `apply`、`Comments` の末尾が `人: "この分岐を消してください"` の open PR を `PR()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `PR #<n> は auto-fix が受け取り中` である（C の条件を満たしていても進行中が先）

#### Scenario: question PR で最新コメントが人なら進行中
- **WHEN** `Labels` が `propose` と `question`、`Comments` の末尾が `人: "Q1: A"` の open PR を `PR()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `PR #<n> は回答済み。worker が受け取り中` である（A には当たらず、「その他」にも落とさない）

#### Scenario: 回答済みの question issue は進行中
- **WHEN** `Labels` が `stage:propose` と `blocked` と `question`、`Comments` が `[AI: "…blocked-by: human…", 人: "B で進めてください"]` の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は回答済み。sweep 待ち` である

#### Scenario: question のみの issue は進行中
- **WHEN** `Labels` が `question`（`blocked` 無し）、`Comments` の末尾が AI の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は question のみ。dispatcher の回収待ち` である（E には当たらない）

#### Scenario: 段階ラベル付きで wip の無い issue は進行中
- **WHEN** `Labels` が `stage:propose` の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は進行中` である

### Requirement: どの行にも当たらないものはその他バケットに入れる
`PR()` は、進行中の規則にも判定表の行にも当たらない open PR を `Situation` `other`、`Priority` 6、`Tab` `今やる` と MUST 判定する（human-turn-signals.md: ラベルなし・コメントなしの PR、旧構成の `retro` PR など。「進行中」に混ぜると人の出番かどうかを判別できなくなる。消えて見えなくなる項目を作らない）。`in-progress` を返す規則は上記 5 規則だけで、それ以外の open PR を黙って落とさない。
`Issue()` も、`Labels` に `question` と `blocked` の両方があるのに B に当たらない issue（`Comments` が nil または空で、最新コメントが AI か人か決められない）を `other` と MUST 判定する。この issue は本来 B か進行中の規則 4 のどちらかであり、進行中に落とすと s07 の取り忘れで人待ちの issue が見えなくなる。

#### Scenario: ラベルもコメントも無い PR
- **WHEN** `Labels` が空、`Comments` が空、`MergeState` が nil の open PR を `PR()` に渡す
- **THEN** `Situation` は `other`、`Priority` は 6、`Tab` は `今やる`、`Summary` は `PR #<n> はどの局面にも当たらない` である

#### Scenario: 旧構成の retro PR
- **WHEN** `Labels` が `retro` の open PR を `PR()` に渡す
- **THEN** `Situation` は `other` である

#### Scenario: question と blocked があるのにコメント未取得の issue
- **WHEN** `Labels` が `stage:propose` と `blocked` と `question`、`Comments` が nil の issue を `Issue()` に渡す
- **THEN** `Situation` は `other`、`Priority` は 6、`Tab` は `今やる`、`Summary` は `#<n> はどの局面にも当たらない` である（B は成立せず、進行中にも落とさない）

### Requirement: 局面ごとの優先度・タブ・種別・1 行要約が決まる
`model.Situation` の `Priority()` / `Tab()` / `Kind()` と、`Result.Summary` は次の表に MUST 従う。優先度は human-turn-signals.md の優先度列（小さいほど上）。タブは今やる = A/B/C/D/G/その他、バックログ = E、異常 = F、進行中 = in-progress。`<n>` は issue / PR 番号。

| Situation | Priority | Tab | Kind | Summary |
| --- | --- | --- | --- | --- |
| `F` | 0 | 異常 | 異常 | issue: `#<n> に段階ラベルが 2 つ以上ある` / PR: `PR #<n> に段階ラベルが 2 つ以上ある` |
| `A` | 1 | 今やる | 質問 | `PR #<n> の質問に答える` |
| `D` | 1 | 今やる | 質問 | `PR #<n> のレビュー質問に答える` |
| `B` | 2 | 今やる | 方針 | `#<n> の方針を決めてコメントする` |
| `C` | 3 | 今やる | merge | `PR #<n> を merge する` |
| `E` | 4 | バックログ | todo 候補 | `#<n> の着手を承認する` |
| `G` | 5 | 今やる | merge | `docs PR #<n> を merge する` |
| `other` | 6 | 今やる | その他 | PR: `PR #<n> はどの局面にも当たらない` / issue: `#<n> はどの局面にも当たらない` |
| `in-progress` | 7 | 進行中 | 進行中 | 規則 1: `#<n> は AI が作業中` / 規則 2: `PR #<n> は auto-fix が受け取り中` / 規則 3: `PR #<n> は回答済み。worker が受け取り中` / 規則 4: `#<n> は回答済み。sweep 待ち` / 規則 5: `#<n> は question のみ。dispatcher の回収待ち` / フォールバック: `#<n> は進行中` |
| `""` | 8 | （無し。`Tab()` は空文字列） | （空文字列） | （空文字列） |

`Issue()` / `PR()` の返り値と、`Card()` が埋める open PR / Issue / Card の `Result` では、`Priority` と `Tab` が `Situation.Priority()` / `Situation.Tab()` と一致する。`MERGED` / `CLOSED` の PR の `Result` は struct のゼロ値のまま（`Priority` フィールドも 0）で、並び順に使わない。

#### Scenario: 優先度は F が最上位で other が最下位
- **WHEN** `model.SituationF.Priority()` と `model.SituationOther.Priority()` を比べる
- **THEN** 前者は 0、後者は 6 である

#### Scenario: タブの振り分け
- **WHEN** A / B / C / D / G / other / E / F / in-progress の `Tab()` を呼ぶ
- **THEN** 順に 今やる / 今やる / 今やる / 今やる / 今やる / 今やる / バックログ / 異常 / 進行中 が返る

### Requirement: Card は Issue と open PR 群のうち最上位の局面を 1 行目に出す
`internal/classify` は `Card(c model.Card) model.Card` を MUST 提供する。返り値は入力のコピーで、`Issue.Result`（`Issue` が nil でなければ）と各 `PRs[i].Result`（`State` が `OPEN` のものだけ。それ以外はゼロ値）を `Issue()` / `PR()` で埋め、`Canonical`（次の Requirement）を立て、`Card.Result` を次の規則で決める。
- 候補は `Issue.Result` と、`State` が `OPEN` の各 PR の `Result` のうち、`Situation` が `in-progress` でないもの
- 候補があれば、`Priority` が最小のものを `Card.Result` にする。同点なら `Issue` を優先し、次に `PRs` の並び順で先のもの
- 候補が無ければ（すべて `in-progress`、または Issue が nil で open PR も無い）`Card.Result` は `in-progress` にし、`Summary` は `PRs` の並び順で先頭の open PR の `Summary`、open PR が無ければ `Issue` の `Summary`、どちらも無ければ `進行中`（`MERGED` / `CLOSED` の PR は `Summary` が空なので採らない）

`Card.Result.Tab` がそのカードを出すタブであり、`Card.Result.Priority` がタブ内の並び順の第 1 キーである（第 2 キー以降は s08 が決める）。

#### Scenario: PR の A が issue の進行中より上に出る
- **WHEN** `Issue` が `stage:propose` と `wip`（進行中）、`PRs` が open の `propose` + `question` で最新コメントが AI の PR 131 の `Card` を `Card()` に渡す
- **THEN** `Card.Result.Situation` は `A`、`Summary` は `PR #131 の質問に答える`、`Tab` は `今やる`、`PRs[0].Result.Situation` は `A`、`Issue.Result.Situation` は `in-progress` である

#### Scenario: 同点なら issue を優先する
- **WHEN** `Issue` が `stage:propose` と `stage:apply`（F、優先度 0）、`PRs` が `propose` + `apply` の open PR（F、優先度 0）の `Card` を `Card()` に渡す
- **THEN** `Card.Result.Summary` は issue の `#<n> に段階ラベルが 2 つ以上ある` である

#### Scenario: すべて進行中なら進行中
- **WHEN** `Issue` が `stage:apply` と `wip`、`PRs` が `apply` で最新コメントが人の open PR の `Card` を `Card()` に渡す
- **THEN** `Card.Result.Situation` は `in-progress`、`Tab` は `進行中`、`Summary` は open PR の `PR #<n> は auto-fix が受け取り中` である

#### Scenario: open PR が無ければ issue の要約
- **WHEN** `Issue` が `stage:apply` と `wip`、`PRs` が `[propose, MERGED, #131]` だけの `Card` を `Card()` に渡す
- **THEN** `Card.Result.Summary` は issue の `#<n> は AI が作業中` である

#### Scenario: Issue の無い docs PR カード
- **WHEN** `Issue` が nil、`PRs` が `docs` の open PR 1 件の `Card` を `Card()` に渡す
- **THEN** `Card.Result.Situation` は `G` である

#### Scenario: 入力を変更しない
- **WHEN** `Card()` を呼んだ後に入力の `Card` を見る
- **THEN** 入力の `Issue.Result` / `PRs[i].Result` / `Result` はゼロ値のままである

### Requirement: 同段階の merge 済み PR は最新が正本で、古い PR の question を異常扱いしない
`Card()` は、`PRs` のうち `State` が `MERGED` で `model.PRStages(Labels)` が同じ段階の PR が複数あるとき、番号が最大のものだけ `Canonical` を true に MUST する（human-turn-signals.md 実データ節: 人の回答後に proposal を直す propose PR がもう 1 本作られ、古い方には `question` が残る。dispatcher は最新の merge だけを見る）。同段階の merge 済み PR が 1 件だけならそれを `Canonical` にする。段階ラベルが 1 件でない（`model.PRStages(Labels)` が 0 件または 2 件以上の）`MERGED` PR は `Canonical` の候補にしない。`MERGED` / `CLOSED` の PR は `Result` がゼロ値のままで、`question` が残っていても `Card.Result` の候補にならず、F や A にもならない。

#### Scenario: 古い merge 済み propose PR の question は無視される
- **WHEN** `Issue` が `stage:apply`（進行中）、`PRs` が `[propose + question, MERGED, #131]`, `[propose, MERGED, #140]`, `[apply, OPEN, #151, 最新コメントが人]` の `Card` を `Card()` に渡す
- **THEN** `PRs[0].Canonical` は false、`PRs[1].Canonical` は true、`PRs[0].Result.Situation` と `PRs[1].Result.Situation` はゼロ値、`Card.Result.Situation` は `in-progress` である（`A` にも `F` にもならない）

#### Scenario: 段階が違えばそれぞれが正本
- **WHEN** `PRs` が `[propose, MERGED, #131]`, `[apply, MERGED, #151]` の `Card` を `Card()` に渡す
- **THEN** 両方の `Canonical` が true である

### Requirement: fixture と期待値表で分類器をテストする
`internal/classify` のテストは、`internal/gh/testdata/fixtures/` 直下の全 `<alias>` ディレクトリについて、s03 の `gh.NewFake` で `SearchIssues` / `SearchPRs` を読み、各 issue は `ViewIssue` のコメントを `Comments` に、各 PR は `ViewPR` のコメント・`ViewPRMergeState`・`ReviewThreads` を `Comments` / `MergeState` / `ReviewThreads` に入れて `Issue()` / `PR()` を呼び、テストコード内の期待値表（`<alias>` → `issue-<n>` / `pr-<n>` → `Situation`）と MUST 突き合わせる（V-1）。
- 期待値表に無い `<alias>` があればテストは失敗する（採取した fixture を黙って通さない）
- fixture にある issue / PR が期待値表に無い、または期待値表にあって fixture に無い場合もテストは失敗する（網羅性）
- `example` の期待値は `issue-108` → `in-progress`（`question` 付きで最新コメントが人（規則 4））、`issue-140` → `E`、`pr-131` → `A` である
- s04 で採取した `<alias>` の期待値は、実装者が human-turn-signals.md の判定表を fixture の各 issue / PR に手で当てて書く（分類器の出力を写さない）

判定表の行ごとの Scenario（上記の各 Requirement）は、fixture に依存しない手書きの `model.Issue` / `model.PR` でもテストする（`example` は A / E / 進行中しか含まないため）。

#### Scenario: example の期待値と一致する
- **WHEN** `fixtures/example` を読んで全 issue / PR を分類する
- **THEN** `issue-108` は `in-progress`、`issue-140` は `E`、`pr-131` は `A` で、期待値表と一致する

#### Scenario: 期待値表に無い fixture は失敗する
- **WHEN** 期待値表に `<alias>` の項目が無い fixture ディレクトリがある
- **THEN** テストはその `<alias>` を含むメッセージで失敗する

#### Scenario: 期待値表と fixture の件数が合わなければ失敗する
- **WHEN** fixture に `issue-140` があるが期待値表に `issue-140` が無い
- **THEN** テストは `issue-140` を含むメッセージで失敗する
