# human-turn-classify Specification

## Purpose
TBD - created by archiving change s05-classify. Update Purpose after archive.

## Requirements

### Requirement: 分類は純粋関数で、進行中の除外・判定表の順・フォールバックの順に評価する
`internal/classify` は `Issue(is model.Issue, mode model.Mode) model.Result` と `PR(pr model.PR, mode model.Mode, now time.Time) model.Result` を MUST 提供する。どちらも入力だけから結果を決め、I/O を行わず、入力を変更しない。`mode` はそのリポジトリの運用方式（`model.Mode`。ゼロ値は `sdd`）で、設定を正本として呼び出し側が渡す。`now` は分類の基準にする現在時刻で、`PR()` は壁時計を読まずに呼び出し側が渡した値だけを使う（Requirement「キューに入れないものは進行中にする」の規則 2 / 3 / 7 の時間切れに使う）。評価は次の順で行い、最初に当たった規則で止める（human-turn-signals.md「分類はこの順で評価し、最初に当たった行で止める」）。
1. 「キューに入れないもの」の規則 1〜6（Requirement「キューに入れないものは進行中にする」）
2. 判定表の行を A → B → C → D → E → F → G の順（issue に当たり得るのは B / E / F、PR に当たり得るのは A / C / D / F / G）。ただし PR の規則 7（`ai-assess:requested`）は行 A の後・行 C の前に評価する。`mode` が `label` のときだけ、行 D を行 C の前に評価する（Requirement「方式が label の入力は issue-label-driven の語彙と規則で分類する」の差分 4）
3. フォールバック: open PR は「その他」、issue は `question` と `blocked` の両方があれば「その他」、それ以外は「進行中」（Requirement「どの行にも当たらないものはその他バケットに入れる」）

入力の前提: `Comments` / `MergeState` / `ReviewThreads` の詳細は、s20 `card-fetch`「Fetch は open の全 issue / 全 PR の詳細を取得する」に従って s07 が全件入れる。詳細が nil になるのは取得に失敗したときだけであり、そのときは、その詳細を必要とする条件は成立しない（nil の `Comments` は「最新コメントが AI」も「最新コメントが人」も偽、nil の `MergeState` は「mergeable かつ checks 緑」が偽、nil の `ReviewThreads` は「未 resolve の thread がある」が偽）。空の `Comments`（長さ 0）も同じ扱いにする。
「最新コメント」は `Comments` の末尾の要素である。`PR()` は `State` が `OPEN` の PR に対して使う。`MERGED` / `CLOSED` の PR は Requirement「同段階の merge 済み PR は最新が正本」で扱い、`PR()` を呼んだ場合はゼロ値の `Situation`（`""`）を返す。

#### Scenario: 進行中の除外が判定表より先に評価される
- **WHEN** `Labels` が `stage:propose` と `wip`、`Comments` が nil の `Issue` を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は AI が作業中` である（`stage:propose` 単独の issue はどの行にも当たらないが、`wip` の規則が先に当たる）

#### Scenario: 詳細が nil なら A は成立しない
- **WHEN** `Labels` が `propose` と `question`、`Comments` が nil、`State` が `OPEN` の `PR` を `mode` `sdd` で `PR()` に渡す
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
- **WHEN** `Labels` が `propose` と `question`、`Comments` が `[AI: "<!-- routine -->\n## Q1. …", 人: "Q1: A"]`、`UpdatedAt` が `now` の 10 分前の open PR を `PR()` に渡す
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
`PR()` は、`mode` が `sdd`（ゼロ値を含む）のとき、次をすべて満たす open PR を `Situation` `C` と MUST 判定する（行 C）。`IsDraft` は見ない（draft の merge 拒否は s14 の merge ガード（human-turn-signals.md 不変条件 5）が持つ）。`mode` が `label` のときの条件は Requirement「方式が label の入力は issue-label-driven の語彙と規則で分類し、open PR を全件今やるに出す」の差分 5 が定める。
- `Labels` に `propose` / `apply` / `archive` のいずれかがある
- `Labels` に `question` が無い
- `model.ParseUndecided(Body)` が `0, true` を返す（1 行目が無い、または N > 0 なら不成立）
- `MergeState` が non-nil で、`Mergeable` が `MERGEABLE`
- `ChecksGreen(MergeState)` が true

`internal/classify` は `ChecksGreen(ms *gh.PRMergeState) bool` を MUST 公開する。`StatusCheckRollup` の全要素が「`Typename` が `CheckRun` で `Conclusion` が `SUCCESS` / `SKIPPED` / `NEUTRAL` のいずれか」または「`Typename` が `StatusContext` で `State` が `SUCCESS`」なら true。要素が 0 件なら true。`ms` が nil なら false。s14 の merge ガードもこの関数を使う。

#### Scenario: merge する PR
- **WHEN** `Labels` が `archive`、`Body` が `未確定の判断: 0 件\n…`、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件の open PR を `mode` `sdd` で `PR()` に渡す
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
`PR()` は、`mode` が `sdd`（ゼロ値を含む）のとき、`Labels` に `apply` があり、`ReviewThreads` に `IsResolved` が false で `Comments` の末尾の本文が `model.IsAI` で true になる thread が 1 つ以上ある open PR を `Situation` `D` と MUST 判定する（行 D: `apply` PR、未 resolve の review thread、thread 最終コメントが routine のもの）。`question` の有無は問わない（A が先に評価される）。`Comments` が空の thread は数えない。`mode` が `label` のときの対象と評価順は Requirement「方式が label の入力は issue-label-driven の語彙と規則で分類し、open PR を全件今やるに出す」の差分 4 が定める。

#### Scenario: レビュー質問に答える PR
- **WHEN** `Labels` が `apply`、`Body` に `未確定の判断` の行が無く、`Comments` が nil、`ReviewThreads` が `[{IsResolved: false, Comments: [{Body: "<!-- routine -->\nこの分岐は残しますか"}]}]` の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `D`、`Priority` は 1、`Tab` は `今やる` である

#### Scenario: thread が resolve 済みなら D ではない
- **WHEN** 上と同じで thread の `IsResolved` が true の PR を `PR()` に渡す
- **THEN** `Situation` は `other` である（D の条件は成立せず、G にも当たらない）

#### Scenario: thread 最終コメントが人なら D ではない
- **WHEN** 上と同じで thread の `Comments` が `[AI の本文, "残します"]` の PR を `PR()` に渡す
- **THEN** `Situation` は `D` ではない

### Requirement: 局面 E は段階ラベルも blocked も無い open issue
`Issue()` は、`Labels` に `model.IssueStages(mode, Labels)` が空になる（その方式の段階ラベルが 1 つも無い）issue で `blocked` も無いものを `Situation` `E` と MUST 判定する（行 E: open issue、段階ラベルなし、`blocked` なし）。`question` だけが付いた issue は規則 5（進行中）が、2 回書きの途中の issue は規則 6（進行中）が先に当たるので E にならない。規則 6 は最新の routine コメントだけを見るので、その後に人のコメントがあっても E にはならない（sweep が同じ見方で続きの段階ラベルを書く）。

#### Scenario: 着手を承認する issue
- **WHEN** `Labels` が空、`Comments` が nil の issue を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `E`、`Priority` は 4、`Tab` は `バックログ` である

#### Scenario: 段階ラベル以外のラベルがあっても E
- **WHEN** `Labels` が `bug` と `enhancement` の issue を `Issue()` に渡す
- **THEN** `Situation` は `E` である

#### Scenario: stage:todo が付いていれば E ではない
- **WHEN** `Labels` が `stage:todo` の issue を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `in-progress` である（段階ラベルがあるので E の条件は成立しない）

#### Scenario: blocked が付いていれば書き直しの途中でも E ではない
- **WHEN** `Labels` が `blocked`、`Comments` の末尾が `AI: "<!-- routine -->\nrelease: stage:apply"` の issue を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `in-progress` である（E は `blocked` の無い issue だけで、規則 6 も `blocked` があれば当たらない）

### Requirement: 局面 F は段階ラベルが 2 つ以上
`Issue()` は `model.IssueStages(mode, Labels)` が 2 件以上の issue を、`PR()` は `model.PRStages(mode, Labels)` が 2 件以上の open PR を、`Situation` `F` と MUST 判定する（行 F: 段階ラベルが 2 つ以上。TUI から自動修復はしない）。判定表の順に従い、issue では B、PR では A / C / D が先に当たればそちらになる。`label` の PR には段階ラベルが無いので、PR が F になるのは `sdd` のときだけである。

#### Scenario: 段階ラベルが 2 つの issue
- **WHEN** `Labels` が `stage:propose` と `stage:apply` の issue を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `F`、`Priority` は 0、`Tab` は `異常` である

#### Scenario: 段階ラベルが 2 つの PR
- **WHEN** `Labels` が `propose` と `apply`、`Body` に `未確定の判断` の行が無い open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `F` である

#### Scenario: wip でも段階ラベル 2 つなら F
- **WHEN** `Labels` が `stage:propose` と `stage:apply` と `wip` の issue を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `F` である（進行中の規則 1 は `IssueStages` が 1 件のときだけ当たる）

#### Scenario: 段階ラベルが 2 つでも question と blocked があれば B
- **WHEN** `Labels` が `stage:propose` と `stage:apply` と `blocked` と `question`、`Comments` の末尾が AI の issue を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `B` である（表の順で B が先）

### Requirement: 局面 G は docs ラベルの open PR で question 無し
`PR()` は、`mode` が `sdd`（ゼロ値を含む）のとき、`Labels` に `docs` があり `question` が無い open PR を `Situation` `G` と MUST 判定する（行 G: open PR、`docs` ラベル、`question` なし。merge しても段階は進まない）。`MergeState` は見ない（merge 可否は s14 のガードが確認する）。`mode` が `label` のときは行 G を評価しない（`docs` ラベルが無い方式である）。

#### Scenario: docs PR
- **WHEN** `Labels` が `docs`、`MergeState` が nil の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `G`、`Priority` は 5、`Tab` は `今やる` である

#### Scenario: docs PR に question があれば G ではない
- **WHEN** `Labels` が `docs` と `question`、`Comments` が nil の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `other` である（`question` があるので G の条件は成立せず、`Comments` が nil なので A にも進行中の規則 3 にも当たらない）

### Requirement: キューに入れないものは進行中にする
`Issue()` / `PR()` は次の 7 規則のいずれかに当たる入力を `Situation` `in-progress`、`Tab` `進行中` と MUST 判定する（human-turn-signals.md「キューに入れないもの（進行中タブに出す）」と、F の縮小理由に書かれた `question` 単独 issue の扱い、「PR の回答は同一セッションが即座に拾う」、上流 `2b1b791` の 2 回書きと `ai-assess:requested`）。
1. issue: `model.IssueStages(mode, Labels)` が作業中の段階の 1 件だけである。`sdd` は `stage:propose` / `stage:apply` / `stage:archive` のいずれか 1 件で `wip` があるとき、`label` は `In Progress` の 1 件だけで `question` が無いとき（AI が動いている最中）。段階ラベルが 2 件以上なら当たらず、行 F になる
2. PR: `Labels` に `question` も `ai-assess:requested` も無く、`Comments` の末尾の `AI` が false で、PR が動いてから間もない（auto-fix が受け取り中）
3. PR: `Labels` に `question` があり `ai-assess:requested` が無く、`Comments` の末尾の `AI` が false で、PR が動いてから間もない（回答済み。同一セッションの worker が受け取り中）
4. issue: `Labels` に `question` があり、`Comments` の末尾の `AI` が false（回答済み。sweep が `question` を外して worker を起動し直すのを待っている）
5. issue: `Labels` に `question` があり `blocked` が無い（dispatcher が回収する残骸。異常扱いしない）
6. issue: `model.IssueStages(mode, Labels)` が空で `blocked` が無く、`model.IsMidRelabel(mode, Comments)` が true（ブロック解除・死んだ worker の再起動の 2 回書きの途中。sweep が続きの段階ラベルを書く。人が承認ラベルを付けると段階が巻き戻るので E に出さない）
7. PR: `Labels` に `ai-assess:requested` があり `mode` が `sdd` で、PR が動いてから間もない（未確定 0 件になった PR の AI リスク評価が走っている最中。評価を終えた assess がラベルを外すまで人の merge 待ちにしない）

「PR が動いてから間もない」は、`now` から `UpdatedAt` を引いた時間が `classify.StaleAfter`（3 時間）未満であることをいう。`UpdatedAt` が `now` より後なら間もないとみなす。3 時間は上流 `routine-sweep` が「最新のコメントが人のもので、そこから 3 時間を超えて routine の返信が無い PR」を止まったとみなす時間に揃える。規則 2 / 3 / 7 は「次に AI が動く」ことを前提に PR を人の目から外す規則なので、時間切れの PR は進行中に留めない（Routine が止まった・ラベルを外し忘れた、のどちらでも PR を見えなくしない）。
- 規則 2 / 3 の条件（`ai-assess:requested` が無く、最新コメントが人）を満たすのに時間切れの PR は、判定表を評価せずに `Situation` `other`、`Summary` `PR #<n> は人のコメントに AI が応答していない` にする。判定表に流すと、反映されていない人の依頼が C（merge する）に見えるためである
- 規則 7 の条件を満たすのに時間切れの PR は、規則 7 を飛ばして判定表の行 C 以降で分類する
- `ai-assess:requested` が付いた PR に規則 2 / 3 を当てないのは、worker が人の回答を反映し終えて未確定が 0 件になってから、このラベルを付けるためである（`routine-common` の worker の規約）。最新コメントが人のままでも、その PR は規則 7 で扱う

規則 1〜6 は 1 → 6 の順に判定表より先に評価し、最初に当たった規則が `Summary` を決める（`question` のみで最新コメントが人の issue は規則 4 に当たり、`Summary` は `#<n> は回答済み。sweep 待ち`）。規則 7 だけは行 A の後に評価する。`question` と `ai-assess:requested` は上流の規約では同時に付かないが、付いていた場合は人の質問（行 A）を先に出す（消えて見えなくなる項目を作らない）。

どの行にも当たらない issue（段階ラベルがあり `wip` が無い、`blocked` があり `question` が無い、`stage:todo` や `To Do` のみ等）も `in-progress` にする。ただし `question` と `blocked` の両方がある issue は除く（Requirement「どの行にも当たらないものはその他バケットに入れる」）。

#### Scenario: wip の issue は進行中
- **WHEN** `Labels` が `stage:apply` と `wip` の issue を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Tab` は `進行中`、`Summary` は `#<n> は AI が作業中` である

#### Scenario: question 無し PR で最新コメントが人なら進行中
- **WHEN** `Labels` が `apply`、`Comments` の末尾が `人: "この分岐を消してください"`、`UpdatedAt` が `now` の 10 分前の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `PR #<n> は auto-fix が受け取り中` である（C の条件を満たしていても進行中が先）

#### Scenario: question 無し PR で最新コメントが人のまま 3 時間動かなければ応答なしのその他
- **WHEN** `Labels` が `apply`、`Body` が `未確定の判断: 0 件`、`Comments` の末尾が `人: "この分岐を消してください"`、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件、`UpdatedAt` が `now` のちょうど 3 時間前の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `other`、`Summary` は `PR #<n> は人のコメントに AI が応答していない` である（C の条件を満たしていても merge する PR には見せない）

#### Scenario: question PR で最新コメントが人なら進行中
- **WHEN** `Labels` が `propose` と `question`、`Comments` の末尾が `人: "Q1: A"`、`UpdatedAt` が `now` の 10 分前の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `PR #<n> は回答済み。worker が受け取り中` である（A には当たらず、「その他」にも落とさない）

#### Scenario: question PR で最新コメントが人のまま 3 時間動かなければ応答なしのその他
- **WHEN** 上と同じで `UpdatedAt` が `now` の 4 時間前の open PR を `PR()` に渡す
- **THEN** `Situation` は `other`、`Summary` は `PR #<n> は人のコメントに AI が応答していない` である

#### Scenario: 最新コメントが人でも ai-assess:requested があれば規則 7 で扱う
- **WHEN** `Labels` が `propose` と `ai-assess:requested`、`Body` が `未確定の判断: 0 件 — レビューをお願いします`、`Comments` の末尾が `人: "Q1: A"`、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が空の open PR を `mode` `sdd` で、`UpdatedAt` を `now` の 10 分前と 1 日前にしてそれぞれ `PR()` に渡す
- **THEN** 前者の `Summary` は `PR #<n> は AI 評価待ち`、後者の `Situation` は `C`、`Summary` は `PR #<n> を merge する` である

#### Scenario: 回答済みの question issue は進行中
- **WHEN** `Labels` が `stage:propose` と `blocked` と `question`、`Comments` が `[AI: "…blocked-by: human…", 人: "B で進めてください"]` の issue を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は回答済み。sweep 待ち` である

#### Scenario: question のみの issue は進行中
- **WHEN** `Labels` が `question`（`blocked` 無し）、`Comments` の末尾が AI の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は question のみ。dispatcher の回収待ち` である（E には当たらない）

#### Scenario: 2 回書きの途中の issue は進行中
- **WHEN** `Labels` が空、`Comments` の末尾が `AI: "<!-- routine -->\nrestart: 1/3"` の issue を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は段階ラベルの書き直し中。sweep 待ち` である（E には当たらない）

#### Scenario: AI 評価待ちの PR は進行中
- **WHEN** `Labels` が `propose` と `ai-assess:requested`、`Body` が `未確定の判断: 0 件 — レビューをお願いします`、`Comments` の末尾が AI、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件、`UpdatedAt` が `now` の 10 分前の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `PR #<n> は AI 評価待ち` である（C の条件を満たしていても評価が終わるまで merge 待ちにしない）

#### Scenario: AI 評価待ちのまま 3 時間動かなければ判定表で分類する
- **WHEN** 上と同じで `UpdatedAt` が `now` の 1 日前の open PR を `PR()` に渡す
- **THEN** `Situation` は `C`、`Summary` は `PR #<n> を merge する` である

#### Scenario: 2 時間 59 分なら AI 評価待ちのまま
- **WHEN** 上と同じで `UpdatedAt` が `now` の 2 時間 59 分前の open PR を `PR()` に渡す
- **THEN** `Situation` は `in-progress` である

#### Scenario: AI 評価待ちでも question があれば A
- **WHEN** 「AI 評価待ちの PR は進行中」と同じで `Labels` に `question` も付いた open PR を `PR()` に渡す
- **THEN** `Situation` は `A` である

#### Scenario: 段階ラベル付きで wip の無い issue は進行中
- **WHEN** `Labels` が `stage:propose` の issue を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は進行中` である

### Requirement: どの行にも当たらないものはその他バケットに入れる
`PR()` は、進行中の規則にも判定表の行にも当たらない open PR を `Situation` `other`、`Priority` 6、`Tab` `今やる` と MUST 判定する（human-turn-signals.md: ラベルなし・コメントなしの PR、旧構成の `retro` PR など。「進行中」に混ぜると人の出番かどうかを判別できなくなる。消えて見えなくなる項目を作らない）。`in-progress` を返す規則は Requirement「キューに入れないものは進行中にする」の 7 規則だけで、それ以外の open PR を黙って落とさない。規則 2 / 3 の条件を満たしたまま時間切れになった PR も、同じ Requirement のとおり `other` にする。
`Issue()` も、`Labels` に `question` と `blocked` の両方があるのに B に当たらない issue（`Comments` が nil または空で、最新コメントが AI か人か決められない）を `other` と MUST 判定する。この issue は本来 B か進行中の規則 4 のどちらかであり、進行中に落とすと s07 の取り忘れで人待ちの issue が見えなくなる。
`PR()` が `now` で見るのは進行中の規則 2 / 3 / 7 の時間切れだけで、`other` を最終更新からの猶予の間だけ進行中に置く判断は `Card()` が持つ（Requirement「Card は猶予内のその他の PR を進行中に置き換える」）。issue の `other` は `Card()` でも置き換えない（上記のとおり人待ちの取りこぼしであり、隠す根拠が無い）。

#### Scenario: ラベルもコメントも無い PR
- **WHEN** `Labels` が空、`Comments` が空、`MergeState` が nil の open PR を `PR()` に渡す
- **THEN** `Situation` は `other`、`Priority` は 6、`Tab` は `今やる`、`Summary` は `PR #<n> はどの局面にも当たらない` である

#### Scenario: 旧構成の retro PR
- **WHEN** `Labels` が `retro` の open PR を `PR()` に渡す
- **THEN** `Situation` は `other` である

#### Scenario: question と blocked があるのにコメントが nil の issue
- **WHEN** `Labels` が `stage:propose` と `blocked` と `question`、`Comments` が nil の issue を `Issue()` に渡す
- **THEN** `Situation` は `other`、`Priority` は 6、`Tab` は `今やる`、`Summary` は `#<n> はどの局面にも当たらない` である（B は成立せず、進行中にも落とさない）

#### Scenario: UpdatedAt が直前でも PR() はその他を返す
- **WHEN** `Labels` が空、`Comments` が空、`UpdatedAt` が `now` の open PR を `PR()` に渡す
- **THEN** `Situation` は `other` である（猶予の判断は `PR()` にない）

### Requirement: 局面ごとの優先度・タブ・種別・1 行要約が決まる
`model.Situation` の `Priority()` / `Tab()` / `Kind()` と、`Result.Summary` は次の表に MUST 従う。優先度は human-turn-signals.md の優先度列（小さいほど上）。タブは今やる = A/B/C/D/G/その他、バックログ = E、異常 = F、進行中 = in-progress。`<n>` は issue / PR 番号。`<M>` は猶予を分に切り捨てた整数。

| Situation | Priority | Tab | Kind | Summary |
| --- | --- | --- | --- | --- |
| `F` | 0 | 異常 | 異常 | issue: `#<n> に段階ラベルが 2 つ以上ある` / PR: `PR #<n> に段階ラベルが 2 つ以上ある` |
| `A` | 1 | 今やる | 質問 | `PR #<n> の質問に答える` |
| `D` | 1 | 今やる | 質問 | `PR #<n> のレビュー質問に答える` |
| `B` | 2 | 今やる | 方針 | `#<n> の方針を決めてコメントする` |
| `C` | 3 | 今やる | merge | `PR #<n> を merge する` |
| `E` | 4 | バックログ | todo 候補 | `#<n> の着手を承認する` |
| `G` | 5 | 今やる | merge | `docs PR #<n> を merge する` |
| `other` | 6 | 今やる | その他 | PR: `PR #<n> はどの局面にも当たらない` / 規則 2 / 3 の時間切れ: `PR #<n> は人のコメントに AI が応答していない` / issue: `#<n> はどの局面にも当たらない` |
| `in-progress` | 7 | 進行中 | 進行中 | 規則 1: `#<n> は AI が作業中` / 規則 2: `PR #<n> は auto-fix が受け取り中` / 規則 3: `PR #<n> は回答済み。worker が受け取り中` / 規則 4: `#<n> は回答済み。sweep 待ち` / 規則 5: `#<n> は question のみ。dispatcher の回収待ち` / 規則 6: `#<n> は段階ラベルの書き直し中。sweep 待ち` / 規則 7: `PR #<n> は AI 評価待ち` / 猶予（`Card()` の置き換え。sdd の PR のみ）: `PR #<n> はどの局面にも当たらない（更新から <M>m は様子見）` / フォールバック: `#<n> は進行中` |
| `""` | 8 | （無し。`Tab()` は空文字列） | （空文字列） | （空文字列） |

`Issue()` / `PR()` の返り値と、`Card()` が埋める open PR / Issue / Card の `Result` では、`Priority` と `Tab` が `Situation.Priority()` / `Situation.Tab()` と一致する。`MERGED` / `CLOSED` の PR の `Result` は struct のゼロ値のまま（`Priority` フィールドも 0）で、並び順に使わない。

#### Scenario: 優先度は F が最上位で other が最下位
- **WHEN** `model.SituationF.Priority()` と `model.SituationOther.Priority()` を比べる
- **THEN** 前者は 0、後者は 6 である

#### Scenario: タブの振り分け
- **WHEN** A / B / C / D / G / other / E / F / in-progress の `Tab()` を呼ぶ
- **THEN** 順に 今やる / 今やる / 今やる / 今やる / 今やる / 今やる / バックログ / 異常 / 進行中 が返る

### Requirement: Card は Issue と open PR 群のうち最上位の局面を 1 行目に出す
`internal/classify` は `Card(c model.Card, mode model.Mode, now time.Time, grace time.Duration) model.Card` を MUST 提供する。1 枚の Card は 1 つのリポジトリの issue と PR だけを持つので、`mode` はカード全体で 1 つに決まる。`grace` は「その他」を進行中に置く猶予で、`now` とともに呼び出し側（s07 `Fetch`）が渡す。返り値は入力のコピーで、`Issue.Result`（`Issue` が nil でなければ）と各 `PRs[i].Result`（`State` が `OPEN` のものだけ。それ以外はゼロ値）を `Issue()` / `PR()` に同じ `mode`（`PR()` には同じ `now` も）を渡して埋め、Requirement「Card は猶予内のその他の PR を進行中に置き換える」の置き換えを open PR ごとに当て、`Canonical`（次の Requirement）を立て、`Card.Result` を次の規則で決める。
- 候補は `Issue.Result` と、`State` が `OPEN` の各 PR の `Result` のうち、`Situation` が `in-progress` でないもの
- 候補があれば、`Priority` が最小のものを `Card.Result` にする。同点なら `Issue` を優先し、次に `PRs` の並び順で先のもの
- 候補が無ければ（すべて `in-progress`、または Issue が nil で open PR も無い）`Card.Result` は `in-progress` にし、`Summary` は `PRs` の並び順で先頭の open PR の `Summary`、open PR が無ければ `Issue` の `Summary`、どちらも無ければ `進行中`（`MERGED` / `CLOSED` の PR は `Summary` が空なので採らない）

`Card.Result.Tab` がそのカードを出すタブであり、`Card.Result.Priority` がタブ内の並び順の第 1 キーである（第 2 キー以降は s08 が決める）。以下の Scenario で `grace` を書いていないものは `grace` が 0（猶予なし）である。

#### Scenario: PR の A が issue の進行中より上に出る
- **WHEN** `Issue` が `stage:propose` と `wip`（進行中）、`PRs` が open の `propose` + `question` で最新コメントが AI の PR 131 の `Card` を `mode` `sdd` で `Card()` に渡す
- **THEN** `Card.Result.Situation` は `A`、`Summary` は `PR #131 の質問に答える`、`Tab` は `今やる`、`PRs[0].Result.Situation` は `A`、`Issue.Result.Situation` は `in-progress` である

#### Scenario: 同点なら issue を優先する
- **WHEN** `Issue` が `stage:propose` と `stage:apply`（F、優先度 0）、`PRs` が `propose` + `apply` の open PR（F、優先度 0）の `Card` を `mode` `sdd` で `Card()` に渡す
- **THEN** `Card.Result.Summary` は issue の `#<n> に段階ラベルが 2 つ以上ある` である

#### Scenario: すべて進行中なら進行中
- **WHEN** `Issue` が `stage:apply` と `wip`、`PRs` が `apply` で最新コメントが人かつ `UpdatedAt` が `now` の 10 分前の open PR の `Card` を `mode` `sdd` で `Card()` に渡す
- **THEN** `Card.Result.Situation` は `in-progress`、`Tab` は `進行中`、`Summary` は open PR の `PR #<n> は auto-fix が受け取り中` である

#### Scenario: open PR が無ければ issue の要約
- **WHEN** `Issue` が `stage:apply` と `wip`、`PRs` が `[propose, MERGED, #131]` だけの `Card` を `mode` `sdd` で `Card()` に渡す
- **THEN** `Card.Result.Summary` は issue の `#<n> は AI が作業中` である

#### Scenario: Issue の無い docs PR カード
- **WHEN** `Issue` が nil、`PRs` が `docs` の open PR 1 件の `Card` を `mode` `sdd` で `Card()` に渡す
- **THEN** `Card.Result.Situation` は `G` である

#### Scenario: label 方式のカードは同じ mode で全要素が分類される
- **WHEN** `Issue` が `In Progress`、`PRs` が `Closes #12` を持つ緑の open PR の `Card` を `mode` `label` で `Card()` に渡す
- **THEN** `Issue.Result.Situation` は `in-progress`、`PRs[0].Result.Situation` は `C`、`Card.Result.Situation` は `C` である

#### Scenario: 入力を変更しない
- **WHEN** `Card()` を呼んだ後に入力の `Card` を見る
- **THEN** 入力の `Issue.Result` / `PRs[i].Result` / `Result` はゼロ値のままである

#### Scenario: wip の issue と猶予中の PR のカードは先頭 open PR の要約で進行中
- **WHEN** `Issue` が `stage:apply` と `wip`、`PRs` が `Labels` 空・`Comments` 空・`UpdatedAt` が `now` の 5 分前の open PR 152 の `Card` を `mode` `sdd`、`grace` 30 分で `Card()` に渡す
- **THEN** `Card.Result.Situation` は `in-progress`、`Tab` は `進行中`、`Summary` は `PR #152 はどの局面にも当たらない（更新から 30m は様子見）` である（候補が無いので先頭の open PR の要約を採る。この change の前は `other` として今やるタブに出ていた）

### Requirement: 同段階の merge 済み PR は最新が正本で、古い PR の question を異常扱いしない
`Card()` は、`PRs` のうち `State` が `MERGED` で `model.PRStages(mode, Labels)` が同じ段階の PR が複数あるとき、番号が最大のものだけ `Canonical` を true に MUST する（human-turn-signals.md 実データ節: 人の回答後に proposal を直す propose PR がもう 1 本作られ、古い方には `question` が残る。dispatcher は最新の merge だけを見る）。同段階の merge 済み PR が 1 件だけならそれを `Canonical` にする。段階ラベルが 1 件でない（`model.PRStages(mode, Labels)` が 0 件または 2 件以上の）`MERGED` PR は `Canonical` の候補にしない。したがって `label` では PR に段階ラベルが無く、`Canonical` は立たない（`label` は issue 1 件に PR 1 本の運用であり、同段階の複数 PR という状態を持たない）。`MERGED` / `CLOSED` の PR は `Result` がゼロ値のままで、`question` が残っていても `Card.Result` の候補にならず、F や A にもならない。

#### Scenario: 古い merge 済み propose PR の question は無視される
- **WHEN** `Issue` が `stage:apply`（進行中）、`PRs` が `[propose + question, MERGED, #131]`, `[propose, MERGED, #140]`, `[apply, OPEN, #151, 最新コメントが人]` の `Card` を `mode` `sdd` で `Card()` に渡す
- **THEN** `PRs[0].Canonical` は false、`PRs[1].Canonical` は true、`PRs[0].Result.Situation` と `PRs[1].Result.Situation` はゼロ値、`Card.Result.Situation` は `in-progress` である（`A` にも `F` にもならない）

#### Scenario: 段階が違えばそれぞれが正本
- **WHEN** `PRs` が `[propose, MERGED, #131]`, `[apply, MERGED, #151]` の `Card` を `mode` `sdd` で `Card()` に渡す
- **THEN** 両方の `Canonical` が true である

#### Scenario: label 方式では Canonical が立たない
- **WHEN** `PRs` が `[ラベル無し, MERGED, #61]` の `Card` を `mode` `label` で `Card()` に渡す
- **THEN** `PRs[0].Canonical` は false である

### Requirement: fixture と期待値表で分類器をテストする
`internal/classify` のテストは、`internal/gh/testdata/fixtures/` 直下の全 `<alias>` ディレクトリについて、s03 の `gh.NewFake` で `SearchIssues` / `SearchPRs` を読み、各 issue は `ViewIssue` のコメントを `Comments` に、各 PR は `ViewPR` のコメント・`ViewPRMergeState`・`ReviewThreads` を `Comments` / `MergeState` / `ReviewThreads` に入れ、期待値表がその `<alias>` に定めた方式を `mode` として `Issue()` / `PR()` を呼び、テストコード内の期待値表（`<alias>` → 方式と、`issue-<n>` / `pr-<n>` → `Situation`）と MUST 突き合わせる（V-1）。`PR()` の `now` にはその PR の `UpdatedAt` を渡す（採取した時点の局面で期待値を書くため。テストを実行した日によって結果を変えない）。
- 期待値表に無い `<alias>` があればテストは失敗する（採取した fixture を黙って通さない）
- fixture にある issue / PR が期待値表に無い、または期待値表にあって fixture に無い場合もテストは失敗する（網羅性）
- `example` の方式は `sdd` で、期待値は `issue-108` → `in-progress`（`question` 付きで最新コメントが人（規則 4））、`issue-140` → `E`、`pr-131` → `A` である
- issue-label-driven の `<alias>` は方式が `label` で、`To Do` / `In Progress` / `Done` の 3 ラベルと `Closes #n` だけを持つ issue / PR を含む
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

#### Scenario: label 方式の fixture がその方式で分類される
- **WHEN** 方式を `label` と定めた `<alias>` の fixture を読んで全 issue / PR を分類する
- **THEN** `In Progress` の issue は `in-progress`、ラベルの無い issue は `E`、`To Do` と `In Progress` が同時に付いた issue は `F`、`blocked` と `question` が付き最新コメントが AI の issue は `B`、`Closes #n` を持ち `question` が無く mergeable で checks 緑の PR は `C`、`question` 付きで最新コメントが AI の PR は `A` である

### Requirement: 2 回書きの途中と unblock-when を本文から読む
`internal/model` は次の 2 つを MUST 提供する。どちらもコメントの著者を見ない（routine は利用者本人のアカウントで投稿するため。human-turn-signals.md）。

- `IsMidRelabel(mode Mode, comments []Comment) bool`: 末尾から見て最初に見つかった `AI` が true のコメントが、その方式の目印で始まる行（前後の空白は無視）を含むなら true。AI のコメントが 1 件も無ければ false。目印は `sdd`（ゼロ値を含む）が `release:` / `restart:` / `advance:` の 3 つ、`label` が `restart:` の 1 つである。上流 `routine-common`「ブロック解除・死んだ worker の再起動は `[]` を書いてから `[stage:X]` を書く」の 2 回書きの途中を、sweep と同じ目印（`routine-sweep`「段階ラベルが無く、最新の `<!-- routine -->` コメントが `release:` / `restart:` / `advance:`」）で見分ける。issue-label-driven は 1 回書きで遷移し、残骸は `restart:` コメントだけである
- `UnblockWhen(body string) (string, bool)`: 本文の行のうち `unblock-when:` で始まる最初の行（前後の空白は無視）の、その接頭辞より後ろを空白を除いて返す。無ければ `"", false`。上流 `routine-common`「解除条件を `unblock-when:` の 1 行で明示する」の値（`comment` / `docs` / `#m`）である

#### Scenario: 最新の routine コメントが restart なら書き直しの途中
- **WHEN** `IsMidRelabel(ModeSDD, [AI: "<!-- routine -->\nadvance: stage:apply", AI: "<!-- routine -->\nrestart: 1/3"])` を呼ぶ
- **THEN** true が返る

#### Scenario: 人のコメントは飛ばして最新の routine コメントを見る
- **WHEN** `IsMidRelabel(ModeSDD, [AI: "<!-- routine -->\nrelease: stage:apply", 人: "了解しました"])` を呼ぶ
- **THEN** true が返る（sweep も人のコメントを飛ばして最新の routine コメントで続きを書くので、同じ見方に揃える）

#### Scenario: 通常の routine コメントは書き直しの途中ではない
- **WHEN** `IsMidRelabel(ModeSDD, [AI: "<!-- routine -->\nblocked-by: human\nunblock-when: comment"])` を呼ぶ
- **THEN** false が返る

#### Scenario: label 方式は restart 以外を目印にしない
- **WHEN** `IsMidRelabel(ModeLabel, [AI: "<!-- routine -->\nrelease: To Do"])` と `IsMidRelabel(ModeLabel, [AI: "<!-- routine -->\nrestart: 1/3"])` を呼ぶ
- **THEN** 順に false と true が返る

#### Scenario: unblock-when の値を読む
- **WHEN** `"<!-- routine -->\nblocked-by: human\nunblock-when: docs\n\n方針を決めてください"` を `UnblockWhen` に渡す
- **THEN** `"docs"` と true が返る

#### Scenario: unblock-when が無ければ false
- **WHEN** `"<!-- routine -->\nblocked-by: #589"` を `UnblockWhen` に渡す
- **THEN** `""` と false が返る

### Requirement: 方式が label の入力は issue-label-driven の語彙と規則で分類し、open PR を全件今やるに出す
`Issue()` / `PR()` は、`mode` が `label` のとき、この Requirement が定める差分を適用して MUST 分類する。差分に挙げていない点（行 A / B、進行中の規則 4 / 5、フォールバック、優先度・タブ・種別・要約の表）は方式によらず共通で、既存の Requirement のとおりである。`mode` が `sdd`（ゼロ値を含む）の入力の分類結果は、この change の前後で 1 件も変わらない。

段階ラベルの語彙は `model.IssueStages(mode, Labels)` / `model.PRStages(mode, Labels)` が持つ（`card-model`）。`label` では issue の段階ラベルが `To Do` / `In Progress` / `Done` の 3 つ、PR の段階ラベルは無い。したがって行 E（段階ラベルも `blocked` も無い open issue）と行 F（段階ラベルが 2 つ以上）は、この 3 ラベルを数えて判定する。

差分は次の 7 点である。
1. 進行中の規則 1（AI が作業中）は「`IssueStages` が `In Progress` の 1 件だけで、かつ `question` が付いていない」で当たる。`label` に `wip` ラベルは無く、`In Progress` が段階と作業中の印を兼ねるので、`question` を除外しないと行 B（方針を決める）に永久に到達しない。`To Do` だけ、または `Done` だけが付いた open issue はこの規則に当たらず、フォールバックの `in-progress`（`#<n> は進行中`）になる
2. 進行中の規則 6（2 回書きの途中）の目印は `restart:` の行だけである。`label` の残骸は `restart:` コメントだけで、`release:` / `advance:` を書く経路が無い
3. 進行中の規則 7（`ai-assess:requested`）は適用しない。`label` にこのラベルは無い
4. 行 D（レビュー質問に答える）を行 C（merge する）より**先に**評価する。対象は open PR のすべてで、未 resolve の review thread の最終コメントが `model.IsAI` で true になるものが当たる。行 C の条件から本文 1 行目のゲートが外れる以上、C を先に評価すると緑の PR でレビュー質問が永久に埋もれる。優先度も D が 1、C が 3 でこの順に一致する
5. 行 C の対象も open PR のすべてである。条件は「`question` が無い」「`MergeState` が non-nil で `Mergeable` が `MERGEABLE`」「`ChecksGreen(MergeState)` が true」の 3 つで、本文 1 行目（`未確定の判断: N 件`）も `Closes #n` も読まない（issue #5 のラベル表: `label` では `question` が PR の merge 禁止の印である）
6. 行 G（`docs` PR）は当たらない。`label` に `docs` ラベルは無い
7. 進行中の規則 2 / 3（PR の最新コメントが人）は適用しない。`sdd` では auto-fix や worker が同じ PR を受け取って続きを進めるので「進行中」に落とす意味があるが、`label` は 1 issue = 1 PR を人が捌く方式で、キューから外れると人の出番が見えなくなる

差分 4 / 5 / 7 の結果、`label` の open PR は「`question` あり → 行 A」「未 resolve の AI thread あり → 行 D」「`question` 無しで mergeable かつ checks 緑 → 行 C」「それ以外 → その他」の 4 つに分かれる。その他の `Tab` も `今やる` なので、`label` のリポジトリの open PR は全件が `[1]今やる` に並び、`[3]進行中` には 1 件も入らない。行 C と行 D の対象から `Closes #n` の絞り込みを外したのは、routine が作った PR だけを拾う形では外部からの PR や `Closes` を書き忘れた PR が人から見えなくなるためである。

#### Scenario: In Progress の issue は進行中
- **WHEN** `Labels` が `In Progress`、`Comments` が nil の issue を `mode` `label` で `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Tab` は `進行中`、`Summary` は `#<n> は AI が作業中` である

#### Scenario: In Progress でも question と blocked があれば方針を決める
- **WHEN** `Labels` が `In Progress` と `blocked` と `question`、`Comments` の末尾が AI の issue を `mode` `label` で `Issue()` に渡す
- **THEN** `Situation` は `B`、`Priority` は 2、`Tab` は `今やる` である

#### Scenario: In Progress の question に人が答えた後は進行中
- **WHEN** `Labels` が `In Progress` と `blocked` と `question`、`Comments` が `[AI, 人]` の issue を `mode` `label` で `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は回答済み。sweep 待ち` である

#### Scenario: ラベルの無い issue はバックログに出る
- **WHEN** `Labels` が `bug` だけ、`Comments` が nil の issue を `mode` `label` で `Issue()` に渡す
- **THEN** `Situation` は `E`、`Tab` は `バックログ` である

#### Scenario: To Do と In Progress が同時に付いた issue は異常
- **WHEN** `Labels` が `To Do` と `In Progress` の issue を `mode` `label` で `Issue()` に渡す
- **THEN** `Situation` は `F`、`Priority` は 0、`Tab` は `異常` である

#### Scenario: sdd の段階ラベルは label 方式では段階として数えない
- **WHEN** `Labels` が `stage:propose` と `stage:apply` と `wip` の issue を `mode` `label` で `Issue()` に渡す
- **THEN** `Situation` は `E` である（`label` の段階ラベルが 1 つも付いておらず `blocked` も無い）

#### Scenario: Closes を持つ緑の PR は 1 行目が無くても merge 行に出る
- **WHEN** `Labels` が空、`Body` が `ラベル一覧をモーダルで出す。\n\nCloses #12`、`Comments` の末尾が AI、`ReviewThreads` が空、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件の open PR を `mode` `label` で `PR()` に渡す
- **THEN** `Situation` は `C`、`Priority` は 3、`Tab` は `今やる` である

#### Scenario: 緑の PR でも未 resolve の AI thread があればレビュー質問が先
- **WHEN** 上と同じ PR に `ReviewThreads` が `[{IsResolved: false, Comments: [{Body: "<!-- routine -->\nこの分岐は残しますか"}]}]` の状態を足して `PR()` に渡す
- **THEN** `Situation` は `D`、`Priority` は 1 である

#### Scenario: 紐づく issue の無い緑の PR も merge 行に出る
- **WHEN** `Body` が `依存を更新する`（`Closes` が無い）で、他は「Closes を持つ緑の PR」と同じ open PR を `mode` `label` で `PR()` に渡す
- **THEN** `Situation` は `C` である（`label` の行 C は `Closes #n` を見ない）

#### Scenario: Closes の無い PR でも未 resolve の AI thread はレビュー質問になる
- **WHEN** 上の PR に `ReviewThreads` が `[{IsResolved: false, Comments: [{Body: "<!-- routine -->\nこの分岐は残しますか"}]}]` の状態を足して `PR()` に渡す
- **THEN** `Situation` は `D`、`Priority` は 1 である

#### Scenario: 最新コメントが人の PR も今やるに出る
- **WHEN** `Labels` が空、`Body` に `Closes #12`、`Comments` の末尾が `人: "この分岐を消してください"`、`ReviewThreads` が空、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件の open PR を `mode` `label` で `PR()` に渡す
- **THEN** `Situation` は `C`、`Tab` は `今やる` である（進行中の規則 2 を適用しない）

#### Scenario: question の PR に人が答えても質問のまま今やるに出る
- **WHEN** `Labels` が `question`、`Body` に `Closes #12`、`Comments` の末尾が `人: "Q1: A"` の open PR を `mode` `label` で `PR()` に渡す
- **THEN** `Situation` は `A`、`Priority` は 1、`Tab` は `今やる` である（進行中の規則 3 を適用しない）

#### Scenario: merge できない PR はその他に出る
- **WHEN** `Labels` が空、`Body` が `依存を更新する`、`Comments` が nil、`ReviewThreads` が空、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/FAILURE` 1 件の open PR を `mode` `label` で `PR()` に渡す
- **THEN** `Situation` は `other`、`Priority` は 6、`Tab` は `今やる` である

#### Scenario: question の PR は質問が先に出る
- **WHEN** `Labels` が `question`、`Body` に `Closes #12`、`Comments` の末尾が AI の open PR を `mode` `label` で `PR()` に渡す
- **THEN** `Situation` は `A`、`Priority` は 1 である

#### Scenario: docs ラベルは label 方式では merge 行を作らない
- **WHEN** `Labels` が `docs`、`Body` が `README を直す`（紐づけ無し）、`Comments` が nil、`MergeState` が nil の open PR を `mode` `label` で `PR()` に渡す
- **THEN** `Situation` は `other` である（`MergeState` が nil なので行 C の条件を満たさず、`docs` は `label` では見ない）

#### Scenario: ai-assess:requested は label 方式では見ない
- **WHEN** `Labels` が `ai-assess:requested`、他は「Closes を持つ緑の PR」と同じ open PR を `mode` `label` で `PR()` に渡す
- **THEN** `Situation` は `C` である

#### Scenario: restart: の残骸は書き直しの途中として進行中にする
- **WHEN** `Labels` が空、`Comments` の末尾が `AI: "<!-- routine -->\nrestart: 1/3"` の issue を `mode` `label` で `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は段階ラベルの書き直し中。sweep 待ち` である

#### Scenario: label 方式では release: を残骸の目印にしない
- **WHEN** `Labels` が空、`Comments` の末尾が `AI: "<!-- routine -->\nrelease: To Do"` の issue を `mode` `label` で `Issue()` に渡す
- **THEN** `Situation` は `E` である

### Requirement: Card は猶予内のその他の PR を進行中に置き換える
`Card()` は、`mode` が `sdd`（ゼロ値を含む）のとき、`PR()` で埋めた `State` が `OPEN` の各 PR の `Result` について、次の 4 つをすべて満たすものを `Situation` `in-progress`、`Priority` 7、`Tab` `進行中` に MUST 置き換える（proposal: sdd の PR のその他の大半は routine の状態機械の途中であり、最終更新から一定時間誰も触っていないものだけを人に出す）。
- `Result.Situation` が `other`
- `grace` が 0 より大きい
- その PR の `UpdatedAt` がゼロ値でない（取得できている）
- `now.Sub(UpdatedAt)` が `grace` 未満（負の値、つまり `now` が `UpdatedAt` より前の場合も含む。ちょうど `grace` は含まない）

置き換え後の `Summary` は `PR #<n> はどの局面にも当たらない（更新から <M>m は様子見）`（`<M>` は `grace` を分に切り捨てた整数）。
4 つのいずれかを満たさない PR は触らない。`mode` が `label` のときはこの置き換えを行わない（Requirement「方式が label の入力は issue-label-driven の語彙と規則で分類し、open PR を全件今やるに出す」のとおり、`label` の open PR は全件 `今やる` に並び、`進行中` には 1 件も入らない）。`Issue.Result` も対象にしない（issue の `other` は詳細取得の失敗で B を取りこぼしたもので、routine の途中ではない）。`other` 以外の `Situation` も対象にならず、`grace` が 0 なら `Card()` の結果はこの change の前と 1 件も変わらない。進行中の規則 2 / 3 の時間切れで `other` になった PR（`Summary` が `PR #<n> は人のコメントに AI が応答していない`）は `UpdatedAt` から 3 時間以上経っているので、猶予が 3 時間未満ならこの置き換えに当たらない。置き換えは `Card.Result` を決める前に行うので、猶予内の `other` は候補にならず、他の要素がすべて `in-progress` ならカードも `in-progress` になる。`now.Sub(UpdatedAt)` が `grace` 以上になった取得では `other` に戻り、今やるタブに出る（s13 `desktop-notify` がこれを「増えた」と数える）。

#### Scenario: 更新から猶予未満のラベル無し PR は進行中
- **WHEN** `Issue` が nil、`PRs` が `Labels` 空・`Comments` 空・`MergeState` nil・`UpdatedAt` が `now` の 10 分前の open PR 61 の `Card` を、`mode` `sdd`、`grace` 30 分で `Card()` に渡す
- **THEN** `PRs[0].Result.Situation` は `in-progress`、`Card.Result.Situation` は `in-progress`、`Card.Result.Tab` は `進行中`、`Card.Result.Priority` は 7、`Card.Result.Summary` は `PR #61 はどの局面にも当たらない（更新から 30m は様子見）` である

#### Scenario: 更新から猶予以上のラベル無し PR はその他のまま
- **WHEN** 上と同じで `UpdatedAt` が `now` の 30 分前の PR 61 の `Card` を `grace` 30 分で `Card()` に渡す
- **THEN** `PRs[0].Result.Situation` は `other`、`Card.Result.Situation` は `other`、`Card.Result.Tab` は `今やる`、`Summary` は `PR #61 はどの局面にも当たらない` である（ちょうど 30 分は猶予に含まれない）

#### Scenario: 猶予が 0 なら置き換えない
- **WHEN** 上と同じで `UpdatedAt` が `now` の 10 分前の PR 61 の `Card` を `grace` 0 で `Card()` に渡す
- **THEN** `Card.Result.Situation` は `other` である

#### Scenario: UpdatedAt がゼロ値なら置き換えない
- **WHEN** 上と同じで `UpdatedAt` がゼロ値の PR 61 の `Card` を `grace` 30 分で `Card()` に渡す
- **THEN** `Card.Result.Situation` は `other` である

#### Scenario: label 方式の PR は置き換えない
- **WHEN** `Issue` が nil、`PRs` が `Labels` 空・`Comments` nil・`MergeState` が `StatusCheckRollup` に `CheckRun/FAILURE` 1 件・`UpdatedAt` が `now` の 1 分前の open PR 61 の `Card` を `mode` `label`、`grace` 30 分で `Card()` に渡す
- **THEN** `Card.Result.Situation` は `other`、`Card.Result.Tab` は `今やる` である（`label` の open PR は全件今やるに出す）

#### Scenario: question と blocked がありコメントが nil の issue は猶予の対象にならない
- **WHEN** `Issue` が `stage:propose` と `blocked` と `question`、`Comments` nil、`UpdatedAt` が `now` の 5 分前で、`PRs` が空の `Card` を `mode` `sdd`、`grace` 30 分で `Card()` に渡す
- **THEN** `Issue.Result.Situation` は `other`、`Card.Result.Situation` は `other`、`Card.Result.Tab` は `今やる` である（人待ちの取りこぼしは隠さない）

#### Scenario: 猶予内のその他は他の局面を隠さない
- **WHEN** `Issue` が `stage:propose` と `wip`（進行中）、`PRs` が open の `propose` + `question` で最新コメントが AI の PR 131（A）と、`Labels` 空・`Comments` 空・`UpdatedAt` が `now` の 1 分前の open PR 132 の `Card` を `mode` `sdd`、`grace` 30 分で `Card()` に渡す
- **THEN** `Card.Result.Situation` は `A`、`PRs[1].Result.Situation` は `in-progress` である

#### Scenario: その他以外は猶予の対象にならない
- **WHEN** `Issue` が nil、`PRs` が `docs` の open PR で `UpdatedAt` が `now` の 1 分前の `Card` を `mode` `sdd`、`grace` 30 分で `Card()` に渡す
- **THEN** `Card.Result.Situation` は `G` である
