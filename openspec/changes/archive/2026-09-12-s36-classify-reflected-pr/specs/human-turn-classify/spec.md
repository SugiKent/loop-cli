## MODIFIED Requirements

### Requirement: キューに入れないものは進行中にする
`Issue()` / `PR()` は次の 7 規則のいずれかに当たる入力を `Situation` `in-progress`、`Tab` `進行中` と MUST 判定する（human-turn-signals.md「キューに入れないもの（進行中タブに出す）」と、F の縮小理由に書かれた `question` 単独 issue の扱い、「PR の回答は同一セッションが即座に拾う」、上流 `2b1b791` の 2 回書きと `ai-assess:requested`）。
1. issue: `model.IssueStages(mode, Labels)` が作業中の段階の 1 件だけである。`sdd` は `stage:propose` / `stage:apply` / `stage:archive` のいずれか 1 件で `wip` があるとき、`label` は `In Progress` の 1 件だけで `question` が無いとき（AI が動いている最中）。段階ラベルが 2 件以上なら当たらず、行 F になる
2. PR: `Labels` に `question` も `ai-assess:requested` も無く、`Comments` の末尾の `AI` が false で、worker が反映を終えた印が無く、PR が動いてから間もない（auto-fix が受け取り中）
3. PR: `Labels` に `question` があり `ai-assess:requested` が無く、`Comments` の末尾の `AI` が false で、PR が動いてから間もない（回答済み。同一セッションの worker が受け取り中）
4. issue: `Labels` に `question` があり、`Comments` の末尾の `AI` が false（回答済み。sweep が `question` を外して worker を起動し直すのを待っている）
5. issue: `Labels` に `question` があり `blocked` が無い（dispatcher が回収する残骸。異常扱いしない）
6. issue: `model.IssueStages(mode, Labels)` が空で `blocked` が無く、`model.IsMidRelabel(mode, Comments)` が true（ブロック解除・死んだ worker の再起動の 2 回書きの途中。sweep が続きの段階ラベルを書く。人が承認ラベルを付けると段階が巻き戻るので E に出さない）
7. PR: `Labels` に `ai-assess:requested` があり `mode` が `sdd` で、PR が動いてから間もない（未確定 0 件になった PR の AI リスク評価が走っている最中。評価を終えた assess がラベルを外すまで人の merge 待ちにしない）

「worker が反映を終えた印」は、規則 2 の PR が次の 2 つを同時に満たすことをいう。

- `Body` の（先頭の空行を除いた）1 行目が `未確定の判断: 0 件` である（`model.ParseUndecided` が `0` と `true` を返す）
- `UpdatedAt` が `Comments` の末尾の `CreatedAt` より後である（人の最新コメントの後に PR 自身が動いた）

印がある PR は規則 2 に当たらず、そのまま判定表へ進む。worker は人の回答を反映し終えたことを本文 1 行目 `未確定の判断: 0 件` と `question` ラベルを外すことで表し、そのたびにコメントを返すとは限らない（`routine-common`「PR の `question` は本文 1 行目の `未確定の判断: N 件` と常に一致させる」）。コメントの順番だけで進行中に落とすと、worker が仕事を終えた PR が `StaleAfter` のあいだ今やるタブから消え、同じ本文 1 行目を読む行 C と判断が食い違う。1 行目が `未確定の判断:` の行でない PR（外部から来た PR、`docs` PR）は worker が反映し終えたと宣言していないので、印は無いものとして規則 2 に当てる。規則 3 は `question` が付いている PR の規則であり、`question` が付いているあいだ worker は反映を終えていないので、印による除外を適用しない。

「PR が動いてから間もない」は、`now` から `UpdatedAt` を引いた時間が `classify.StaleAfter`（3 時間）未満であることをいう。`UpdatedAt` が `now` より後なら間もないとみなす。3 時間は上流 `routine-sweep` が「最新のコメントが人のもので、そこから 3 時間を超えて routine の返信が無い PR」を止まったとみなす時間に揃える。規則 2 / 3 / 7 は「次に AI が動く」ことを前提に PR を人の目から外す規則なので、時間切れの PR は進行中に留めない（Routine が止まった・ラベルを外し忘れた、のどちらでも PR を見えなくしない）。
- 規則 2 / 3 の条件（`ai-assess:requested` が無く、最新コメントが人で、規則 2 では worker が反映を終えた印が無い）を満たすのに時間切れの PR は、判定表を評価せずに `Situation` `other`、`Summary` `PR #<n> は人のコメントに AI が応答していない` にする。判定表に流すと、反映されていない人の依頼が C（merge する）に見えるためである
- 規則 7 の条件を満たすのに時間切れの PR は、規則 7 を飛ばして判定表の行 C 以降で分類する
- `ai-assess:requested` が付いた PR に規則 2 / 3 を当てないのは、worker が人の回答を反映し終えて未確定が 0 件になってから、このラベルを付けるためである（`routine-common` の worker の規約）。最新コメントが人のままでも、その PR は規則 7 で扱う

規則 1〜6 は 1 → 6 の順に判定表より先に評価し、最初に当たった規則が `Summary` を決める（`question` のみで最新コメントが人の issue は規則 4 に当たり、`Summary` は `#<n> は回答済み。sweep 待ち`）。規則 7 だけは行 A の後に評価する。`question` と `ai-assess:requested` は上流の規約では同時に付かないが、付いていた場合は人の質問（行 A）を先に出す（消えて見えなくなる項目を作らない）。

どの行にも当たらない issue（段階ラベルがあり `wip` が無い、`blocked` があり `question` が無い、`stage:todo` や `To Do` のみ等）も `in-progress` にする。ただし `question` と `blocked` の両方がある issue は除く（Requirement「どの行にも当たらないものはその他バケットに入れる」）。

#### Scenario: wip の issue は進行中
- **WHEN** `Labels` が `stage:apply` と `wip` の issue を `mode` `sdd` で `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Tab` は `進行中`、`Summary` は `#<n> は AI が作業中` である

#### Scenario: question 無し PR で最新コメントが人なら進行中
- **WHEN** `Labels` が `apply`、`Body` の 1 行目が `未確定の判断: 0 件 — レビューをお願いします`、`Comments` の末尾が `人: "この分岐を消してください"` でその `CreatedAt` が `UpdatedAt` と同時刻、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件、`UpdatedAt` が `now` の 10 分前の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `PR #<n> は auto-fix が受け取り中` である（人が `apply` PR にレビューを書いた直後がこれで、C の条件を満たしていても worker が動くまで merge 候補に出さない）

#### Scenario: 反映を終えた印があれば最新コメントが人でも判定表へ流す
- **WHEN** `Labels` が `propose`、`Body` の 1 行目が `未確定の判断: 0 件 — レビューをお願いします`、`Comments` の末尾が `now` の 61 分前の `人: "質問の回答はすべて推奨で"`、`UpdatedAt` が `now` の 59 分前、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `C`、`Summary` は `PR #<n> を merge する` である（worker が本文とラベルだけを更新してコメントを返さなかった PR を 3 時間隠さない）

#### Scenario: 未確定が残る PR は反映を終えた印にならない
- **WHEN** `Labels` が `apply`、`Body` の 1 行目が `未確定の判断: 2 件 — このまま merge すると worker が推奨案で進めます`、`Comments` の末尾が `now` の 61 分前の `人: "Q1: A"`、`UpdatedAt` が `now` の 59 分前の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `PR #<n> は auto-fix が受け取り中` である

#### Scenario: 1 行目が未確定の判断で始まらない PR は反映を終えた印にならない
- **WHEN** `Labels` が `docs`、`Body` が `Closes #<m>`、`Comments` の末尾が `now` の 61 分前の `人: "この段落を直してください"`、`UpdatedAt` が `now` の 59 分前の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `PR #<n> は auto-fix が受け取り中` である（G の条件を満たしていても進行中が先）

#### Scenario: 反映を終えた印があり判定表のどの行にも当たらない PR はその他
- **WHEN** `Labels` が `apply`、`Body` の 1 行目が `未確定の判断: 0 件 — レビューをお願いします`、`Comments` の末尾が `now` の 5 時間前の `人: "この分岐を消してください"`、`UpdatedAt` が `now` の 4 時間前、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/FAILURE` 1 件の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `other`、`Summary` は `PR #<n> はどの局面にも当たらない` である（規則 2 に当たらないので時間切れの要約にはならない）

#### Scenario: question 無し PR で最新コメントが人のまま 3 時間動かなければ応答なしのその他
- **WHEN** `Labels` が `apply`、`Body` が `未確定の判断: 0 件`、`Comments` の末尾が `人: "この分岐を消してください"` でその `CreatedAt` が `UpdatedAt` と同時刻、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件、`UpdatedAt` が `now` のちょうど 3 時間前の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `other`、`Summary` は `PR #<n> は人のコメントに AI が応答していない` である（C の条件を満たしていても merge する PR には見せない）

#### Scenario: question PR で最新コメントが人なら進行中
- **WHEN** `Labels` が `propose` と `question`、`Comments` の末尾が `人: "Q1: A"`、`UpdatedAt` が `now` の 10 分前の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `PR #<n> は回答済み。worker が受け取り中` である（A には当たらず、「その他」にも落とさない）

#### Scenario: question PR は本文が未確定 0 件でも規則 3 のまま
- **WHEN** `Labels` が `propose` と `question`、`Body` の 1 行目が `未確定の判断: 0 件 — レビューをお願いします`、`Comments` の末尾が `now` の 61 分前の `人: "Q1: A"`、`UpdatedAt` が `now` の 59 分前の open PR を `mode` `sdd` で `PR()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `PR #<n> は回答済み。worker が受け取り中` である（`question` が落ちるまでは反映を終えたとみなさない）

#### Scenario: question PR で最新コメントが人のまま 3 時間動かなければ応答なしのその他
- **WHEN** 「question PR で最新コメントが人なら進行中」と同じで `UpdatedAt` が `now` の 4 時間前の open PR を `PR()` に渡す
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
