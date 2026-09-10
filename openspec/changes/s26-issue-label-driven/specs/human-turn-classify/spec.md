## ADDED Requirements

### Requirement: 方式が label の入力は issue-label-driven の語彙と規則で分類する
`Issue()` / `PR()` は、入力の `Mode` が `label` のとき、この Requirement が定める差分を適用して MUST 分類する。差分に挙げていない点（評価順、行 A / B、進行中の規則 2〜5、フォールバック、優先度・タブ・種別・要約の表）は方式によらず共通で、既存の Requirement のとおりである。`Mode` が `sdd`（ゼロ値を含む）の入力の分類結果は、この change の前後で 1 件も変わらない。

段階ラベルの語彙は `model.IssueStages(Mode, Labels)` / `model.PRStages(Mode, Labels)` が持つ（`card-model`）。`label` では issue の段階ラベルが `To Do` / `In Progress` / `Done` の 3 つ、PR の段階ラベルは無い。したがって行 E（段階ラベルも `blocked` も無い open issue）と行 F（段階ラベルが 2 つ以上）は、この 3 ラベルを数えて判定する。

差分は次の 6 点である。
1. 進行中の規則 1（AI が作業中）は「`IssueStages` が `In Progress` の 1 件だけ」で当たる。`label` に `wip` ラベルは無く、作業中の印は `In Progress` 単独である。`To Do` だけ、または `Done` だけが付いた open issue はこの規則に当たらず、フォールバックの `in-progress`（`#<n> は進行中`）になる
2. 進行中の規則 6（2 回書きの途中）の目印は `restart:` の行だけである。`label` の残骸は `restart:` コメントだけで、`release:` / `advance:` を書く経路が無い
3. 進行中の規則 7（`ai-assess:requested`）は適用しない。`label` にこのラベルは無い
4. 行 C（merge する）の対象は、`model.LinkedIssue(Title, Body)` が `ok` を返す open PR である（`label` の PR には段階ラベルが付かず、issue との紐づけは `Closes #n` だけであり、これが routine の作った PR の印になる）。条件は「`question` が無い」「`MergeState` が non-nil で `Mergeable` が `MERGEABLE`」「`ChecksGreen(MergeState)` が true」の 3 つで、本文 1 行目（`未確定の判断: N 件`）は読まない。`ok` が false の PR（Dependabot や外部からの PR）は C にならず、他の行にも当たらなければ「その他」に出る
5. 行 D（レビュー質問に答える）の対象も、`model.LinkedIssue(Title, Body)` が `ok` を返す open PR である（`label` に `apply` ラベルが無いため）。未 resolve の review thread の最終コメントが `model.IsAI` で true になる条件は共通である
6. 行 G（`docs` PR）は当たらない。`label` に `docs` ラベルは無い

#### Scenario: In Progress の issue は進行中
- **WHEN** `Mode` が `label`、`Labels` が `In Progress`、`Comments` が nil の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Tab` は `進行中`、`Summary` は `#<n> は AI が作業中` である

#### Scenario: ラベルの無い issue はバックログに出る
- **WHEN** `Mode` が `label`、`Labels` が `bug` だけ、`Comments` が nil の issue を `Issue()` に渡す
- **THEN** `Situation` は `E`、`Tab` は `バックログ` である

#### Scenario: To Do と In Progress が同時に付いた issue は異常
- **WHEN** `Mode` が `label`、`Labels` が `To Do` と `In Progress` の issue を `Issue()` に渡す
- **THEN** `Situation` は `F`、`Priority` は 0、`Tab` は `異常` である

#### Scenario: sdd の段階ラベルは label 方式では段階として数えない
- **WHEN** `Mode` が `label`、`Labels` が `stage:propose` と `stage:apply` と `wip` の issue を `Issue()` に渡す
- **THEN** `Situation` は `E` である（`label` の段階ラベルが 1 つも付いておらず `blocked` も無い）

#### Scenario: blocked と question の issue で最新コメントが AI なら方針を決める
- **WHEN** `Mode` が `label`、`Labels` が `In Progress` と `blocked` と `question`、`Comments` の末尾が AI の issue を `Issue()` に渡す
- **THEN** `Situation` は `B`、`Tab` は `今やる` である

#### Scenario: Closes を持つ緑の PR は 1 行目が無くても merge 行に出る
- **WHEN** `Mode` が `label`、`Labels` が空、`Body` が `ラベル一覧をモーダルで出す。\n\nCloses #12`、`Comments` の末尾が AI、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件の open PR を `PR()` に渡す
- **THEN** `Situation` は `C`、`Priority` は 3、`Tab` は `今やる` である

#### Scenario: 紐づく issue の無い PR は merge 行に出さない
- **WHEN** 上と同じで `Body` が `依存を更新する`（`Closes` も `Refs` も無い）の open PR を `PR()` に渡す
- **THEN** `Situation` は `other` である

#### Scenario: question の PR は質問が先に出る
- **WHEN** `Mode` が `label`、`Labels` が `question`、`Body` に `Closes #12`、`Comments` の末尾が AI の open PR を `PR()` に渡す
- **THEN** `Situation` は `A`、`Priority` は 1 である

#### Scenario: 紐づく PR の未 resolve thread はレビュー質問になる
- **WHEN** `Mode` が `label`、`Labels` が空、`Body` に `Closes #12`、`Comments` が nil、`ReviewThreads` が `[{IsResolved: false, Comments: [{Body: "<!-- routine -->\nこの分岐は残しますか"}]}]` の open PR を `PR()` に渡す
- **THEN** `Situation` は `D`、`Priority` は 1 である

#### Scenario: docs ラベルは label 方式では merge 行を作らない
- **WHEN** `Mode` が `label`、`Labels` が `docs`、`Body` が `README を直す`（紐づけ無し）、`Comments` が nil、`MergeState` が nil の open PR を `PR()` に渡す
- **THEN** `Situation` は `other` である

#### Scenario: ai-assess:requested は label 方式では見ない
- **WHEN** `Mode` が `label`、`Labels` が `ai-assess:requested`、`Body` に `Closes #12`、`Comments` の末尾が AI、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件の open PR を `PR()` に渡す
- **THEN** `Situation` は `C` である

#### Scenario: restart: の残骸は書き直しの途中として進行中にする
- **WHEN** `Mode` が `label`、`Labels` が空、`Comments` の末尾が `AI: "<!-- routine -->\nrestart: 1/3"` の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は段階ラベルの書き直し中。sweep 待ち` である

#### Scenario: label 方式では release: を残骸の目印にしない
- **WHEN** `Mode` が `label`、`Labels` が空、`Comments` の末尾が `AI: "<!-- routine -->\nrelease: To Do"` の issue を `Issue()` に渡す
- **THEN** `Situation` は `E` である

## MODIFIED Requirements

### Requirement: 2 回書きの途中と unblock-when を本文から読む
`internal/model` は次の 2 つを MUST 提供する。どちらもコメントの著者を見ない（routine は利用者本人のアカウントで投稿するため。human-turn-signals.md）。

- `IsMidRelabel(mode Mode, comments []Comment) bool`: 末尾から見て最初に見つかった `AI` が true のコメントが、その方式の目印で始まる行（前後の空白は無視）を含むなら true。AI のコメントが 1 件も無ければ false。目印は `sdd`（ゼロ値を含む）が `release:` / `restart:` / `advance:` の 3 つ、`label` が `restart:` の 1 つである。上流 `routine-common`「ブロック解除・死んだ worker の再起動は `[]` を書いてから `[stage:X]` を書く」の 2 回書きの途中を、sweep と同じ目印（`routine-sweep`「段階ラベルが無く、最新の `<!-- routine -->` コメントが `release:` / `restart:` / `advance:`」）で見分ける。issue-label-driven には 1 回書きの遷移しか無く、残骸は `restart:` コメントだけである
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

### Requirement: fixture と期待値表で分類器をテストする
`internal/classify` のテストは、`internal/gh/testdata/fixtures/` 直下の全 `<alias>` ディレクトリについて、s03 の `gh.NewFake` で `SearchIssues` / `SearchPRs` を読み、各 issue は `ViewIssue` のコメントを `Comments` に、各 PR は `ViewPR` のコメント・`ViewPRMergeState`・`ReviewThreads` を `Comments` / `MergeState` / `ReviewThreads` に入れ、期待値表がその `<alias>` に定めた方式を `Mode` に入れて `Issue()` / `PR()` を呼び、テストコード内の期待値表（`<alias>` → 方式と、`issue-<n>` / `pr-<n>` → `Situation`）と MUST 突き合わせる（V-1）。
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
- **THEN** `In Progress` の issue は `in-progress`、ラベルの無い issue は `E`、`To Do` と `In Progress` が同時に付いた issue は `F`、`Closes #n` を持ち `question` が無く mergeable で checks 緑の PR は `C`、`question` 付きで最新コメントが AI の PR は `A` である
