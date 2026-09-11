## MODIFIED Requirements

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

#### Scenario: ラベルもコメントも無い PR
- **WHEN** `Labels` が空、`Comments` が空、`MergeState` が nil の open PR を `PR()` に渡す
- **THEN** `Situation` は `other`、`Priority` は 6、`Tab` は `今やる`、`Summary` は `PR #<n> はどの局面にも当たらない` である

#### Scenario: 旧構成の retro PR
- **WHEN** `Labels` が `retro` の open PR を `PR()` に渡す
- **THEN** `Situation` は `other` である

#### Scenario: question と blocked があるのにコメントが nil の issue
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
| `other` | 6 | 今やる | その他 | PR: `PR #<n> はどの局面にも当たらない` / 規則 2 / 3 の時間切れ: `PR #<n> は人のコメントに AI が応答していない` / issue: `#<n> はどの局面にも当たらない` |
| `in-progress` | 7 | 進行中 | 進行中 | 規則 1: `#<n> は AI が作業中` / 規則 2: `PR #<n> は auto-fix が受け取り中` / 規則 3: `PR #<n> は回答済み。worker が受け取り中` / 規則 4: `#<n> は回答済み。sweep 待ち` / 規則 5: `#<n> は question のみ。dispatcher の回収待ち` / 規則 6: `#<n> は段階ラベルの書き直し中。sweep 待ち` / 規則 7: `PR #<n> は AI 評価待ち` / フォールバック: `#<n> は進行中` |
| `""` | 8 | （無し。`Tab()` は空文字列） | （空文字列） | （空文字列） |

`Issue()` / `PR()` の返り値と、`Card()` が埋める open PR / Issue / Card の `Result` では、`Priority` と `Tab` が `Situation.Priority()` / `Situation.Tab()` と一致する。`MERGED` / `CLOSED` の PR の `Result` は struct のゼロ値のまま（`Priority` フィールドも 0）で、並び順に使わない。

#### Scenario: 優先度は F が最上位で other が最下位
- **WHEN** `model.SituationF.Priority()` と `model.SituationOther.Priority()` を比べる
- **THEN** 前者は 0、後者は 6 である

#### Scenario: タブの振り分け
- **WHEN** A / B / C / D / G / other / E / F / in-progress の `Tab()` を呼ぶ
- **THEN** 順に 今やる / 今やる / 今やる / 今やる / 今やる / 今やる / バックログ / 異常 / 進行中 が返る

### Requirement: Card は Issue と open PR 群のうち最上位の局面を 1 行目に出す
`internal/classify` は `Card(c model.Card, mode model.Mode, now time.Time) model.Card` を MUST 提供する。1 枚の Card は 1 つのリポジトリの issue と PR だけを持つので、`mode` はカード全体で 1 つに決まる。返り値は入力のコピーで、`Issue.Result`（`Issue` が nil でなければ）と各 `PRs[i].Result`（`State` が `OPEN` のものだけ。それ以外はゼロ値）を `Issue()` / `PR()` に同じ `mode`（`PR()` には同じ `now` も）を渡して埋め、`Canonical`（次の Requirement）を立て、`Card.Result` を次の規則で決める。
- 候補は `Issue.Result` と、`State` が `OPEN` の各 PR の `Result` のうち、`Situation` が `in-progress` でないもの
- 候補があれば、`Priority` が最小のものを `Card.Result` にする。同点なら `Issue` を優先し、次に `PRs` の並び順で先のもの
- 候補が無ければ（すべて `in-progress`、または Issue が nil で open PR も無い）`Card.Result` は `in-progress` にし、`Summary` は `PRs` の並び順で先頭の open PR の `Summary`、open PR が無ければ `Issue` の `Summary`、どちらも無ければ `進行中`（`MERGED` / `CLOSED` の PR は `Summary` が空なので採らない）

`Card.Result.Tab` がそのカードを出すタブであり、`Card.Result.Priority` がタブ内の並び順の第 1 キーである（第 2 キー以降は s08 が決める）。

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
