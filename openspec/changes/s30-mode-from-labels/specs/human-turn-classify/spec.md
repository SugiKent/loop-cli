## REMOVED Requirements

### Requirement: 方式が label の入力は issue-label-driven の語彙と規則で分類する

**Reason**: `label` の行 C と行 D から `Closes #n` の絞り込みが外れ、進行中の規則 2 / 3 を適用しない差分が加わる。Scenario「紐づく issue の無い PR は merge 行にもレビュー質問にも出さない」「Refs だけの PR は routine の PR と見なさない」は、`Closes #n` を持たない PR がキューに出ないことを固定しており、この change ではどちらも出るのが正しい振る舞いになる。Scenario 名が振る舞いを反転させるので MODIFIED では書き直せず、Requirement 名を「方式が label の入力は issue-label-driven の語彙と規則で分類し、open PR を全件今やるに出す」に改めて作り直す。

**Migration**: 差分 1・2・3・6（進行中の規則 1 の当たり方、`restart:` だけを目印にすること、`ai-assess:requested` を見ないこと、行 G が当たらないこと）と、段階ラベルの語彙は ADDED の Requirement がそのまま引き継ぐ。行 C と行 D を参照している Requirement「局面 C は段階 PR で未確定 0 件・question 無し・checks 緑・mergeable のもの」「局面 D は apply PR に未 resolve の review thread があり thread 最終コメントが AI のもの」は、参照先の名前だけを新しい名前に差し替える。

## ADDED Requirements

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

## MODIFIED Requirements

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
