# human-turn-classify Specification

## MODIFIED Requirements

### Requirement: 分類は純粋関数で、進行中の除外・判定表の順・フォールバックの順に評価する
`internal/classify` は `Issue(is model.Issue) model.Result` と `PR(pr model.PR) model.Result` を MUST 提供する。どちらも入力だけから結果を決め、I/O を行わず、入力を変更しない。評価は次の順で行い、最初に当たった規則で止める（human-turn-signals.md「分類はこの順で評価し、最初に当たった行で止める」）。
1. 「キューに入れないもの」の規則 1〜6（Requirement「キューに入れないものは進行中にする」）
2. 判定表の行を A → B → C → D → E → F → G の順（issue に当たり得るのは B / E / F、PR に当たり得るのは A / C / D / F / G）。ただし PR の規則 7（`ai-assess:requested`）は行 A の後・行 C の前に評価する
3. フォールバック: open PR は「その他」、issue は `question` と `blocked` の両方があれば「その他」、それ以外は「進行中」（Requirement「どの行にも当たらないものはその他バケットに入れる」）

入力の前提: `Comments` / `MergeState` / `ReviewThreads` の詳細は、s20 `card-fetch`「Fetch は open の全 issue / 全 PR の詳細を取得する」に従って s07 が全件入れる。詳細が nil になるのは取得に失敗したときだけであり、そのときは、その詳細を必要とする条件は成立しない（nil の `Comments` は「最新コメントが AI」も「最新コメントが人」も偽、nil の `MergeState` は「mergeable かつ checks 緑」が偽、nil の `ReviewThreads` は「未 resolve の thread がある」が偽）。空の `Comments`（長さ 0）も同じ扱いにする。
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

### Requirement: 局面 E は段階ラベルも blocked も無い open issue
`Issue()` は、`Labels` に `stage:*`（`model.IssueStages` が空）も `blocked` も無い issue を `Situation` `E` と MUST 判定する（行 E: open issue、`stage:*` ラベルなし、`blocked` なし）。`question` だけが付いた issue は規則 5（進行中）が、2 回書きの途中の issue は規則 6（進行中）が先に当たるので E にならない。規則 6 は最新の routine コメントだけを見るので、その後に人のコメントがあっても E にはならない（sweep が同じ見方で続きの段階ラベルを書く）。

#### Scenario: 着手を承認する issue
- **WHEN** `Labels` が空、`Comments` が nil の issue を `Issue()` に渡す
- **THEN** `Situation` は `E`、`Priority` は 4、`Tab` は `バックログ` である

#### Scenario: 段階ラベル以外のラベルがあっても E
- **WHEN** `Labels` が `bug` と `enhancement` の issue を `Issue()` に渡す
- **THEN** `Situation` は `E` である

#### Scenario: stage:todo が付いていれば E ではない
- **WHEN** `Labels` が `stage:todo` の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress` である（段階ラベルがあるので E の条件は成立しない）

#### Scenario: blocked が付いていれば書き直しの途中でも E ではない
- **WHEN** `Labels` が `blocked`、`Comments` の末尾が `AI: "<!-- routine -->\nrelease: stage:apply"` の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress` である（E は `blocked` の無い issue だけで、規則 6 も `blocked` があれば当たらない）

### Requirement: キューに入れないものは進行中にする
`Issue()` / `PR()` は次の 7 規則のいずれかに当たる入力を `Situation` `in-progress`、`Tab` `進行中` と MUST 判定する（human-turn-signals.md「キューに入れないもの（進行中タブに出す）」と、F の縮小理由に書かれた `question` 単独 issue の扱い、「PR の回答は同一セッションが即座に拾う」、上流 `2b1b791` の 2 回書きと `ai-assess:requested`）。
1. issue: `model.IssueStages(Labels)` が `stage:propose` / `stage:apply` / `stage:archive` のいずれか 1 件だけで、`wip` がある（AI が動いている最中）。段階ラベルが 2 件以上なら当たらず、行 F になる
2. PR: `Labels` に `question` が無く、`Comments` の末尾の `AI` が false（auto-fix が受け取り中）
3. PR: `Labels` に `question` があり、`Comments` の末尾の `AI` が false（回答済み。同一セッションの worker が受け取り中）
4. issue: `Labels` に `question` があり、`Comments` の末尾の `AI` が false（回答済み。sweep が `question` を外して worker を起動し直すのを待っている）
5. issue: `Labels` に `question` があり `blocked` が無い（dispatcher が回収する残骸。異常扱いしない）
6. issue: `model.IssueStages(Labels)` が空で `blocked` が無く、`model.IsMidRelabel(Comments)` が true（ブロック解除・死んだ worker の再起動の 2 回書きの途中。sweep が続きの段階ラベルを書く。人が `stage:todo` を付けると段階が巻き戻るので E に出さない）
7. PR: `Labels` に `ai-assess:requested` がある（未確定 0 件になった PR の AI リスク評価が走っている最中。評価を終えた assess がラベルを外すまで人の merge 待ちにしない）

規則 1〜6 は 1 → 6 の順に判定表より先に評価し、最初に当たった規則が `Summary` を決める（`question` のみで最新コメントが人の issue は規則 4 に当たり、`Summary` は `#<n> は回答済み。sweep 待ち`）。規則 7 だけは行 A の後に評価する。`question` と `ai-assess:requested` は上流の規約では同時に付かないが、付いていた場合は人の質問（行 A）を先に出す（消えて見えなくなる項目を作らない）。

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

#### Scenario: 2 回書きの途中の issue は進行中
- **WHEN** `Labels` が空、`Comments` の末尾が `AI: "<!-- routine -->\nrestart: 1/3"` の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は段階ラベルの書き直し中。sweep 待ち` である（E には当たらない）

#### Scenario: AI 評価待ちの PR は進行中
- **WHEN** `Labels` が `propose` と `ai-assess:requested`、`Body` が `未確定の判断: 0 件 — レビューをお願いします`、`Comments` の末尾が AI、`MergeState` が `Mergeable: MERGEABLE` で `StatusCheckRollup` が `CheckRun/SUCCESS` 1 件の open PR を `PR()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `PR #<n> は AI 評価待ち` である（C の条件を満たしていても評価が終わるまで merge 待ちにしない）

#### Scenario: AI 評価待ちでも question があれば A
- **WHEN** 上と同じで `Labels` に `question` も付いた open PR を `PR()` に渡す
- **THEN** `Situation` は `A` である

#### Scenario: 段階ラベル付きで wip の無い issue は進行中
- **WHEN** `Labels` が `stage:propose` の issue を `Issue()` に渡す
- **THEN** `Situation` は `in-progress`、`Summary` は `#<n> は進行中` である

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
| `in-progress` | 7 | 進行中 | 進行中 | 規則 1: `#<n> は AI が作業中` / 規則 2: `PR #<n> は auto-fix が受け取り中` / 規則 3: `PR #<n> は回答済み。worker が受け取り中` / 規則 4: `#<n> は回答済み。sweep 待ち` / 規則 5: `#<n> は question のみ。dispatcher の回収待ち` / 規則 6: `#<n> は段階ラベルの書き直し中。sweep 待ち` / 規則 7: `PR #<n> は AI 評価待ち` / フォールバック: `#<n> は進行中` |
| `""` | 8 | （無し。`Tab()` は空文字列） | （空文字列） | （空文字列） |

`Issue()` / `PR()` の返り値と、`Card()` が埋める open PR / Issue / Card の `Result` では、`Priority` と `Tab` が `Situation.Priority()` / `Situation.Tab()` と一致する。`MERGED` / `CLOSED` の PR の `Result` は struct のゼロ値のまま（`Priority` フィールドも 0）で、並び順に使わない。

#### Scenario: 優先度は F が最上位で other が最下位
- **WHEN** `model.SituationF.Priority()` と `model.SituationOther.Priority()` を比べる
- **THEN** 前者は 0、後者は 6 である

#### Scenario: タブの振り分け
- **WHEN** A / B / C / D / G / other / E / F / in-progress の `Tab()` を呼ぶ
- **THEN** 順に 今やる / 今やる / 今やる / 今やる / 今やる / 今やる / バックログ / 異常 / 進行中 が返る

## ADDED Requirements

### Requirement: 2 回書きの途中と unblock-when を本文から読む
`internal/model` は次の 2 つを MUST 提供する。どちらもコメントの著者を見ない（routine は利用者本人のアカウントで投稿するため。human-turn-signals.md）。

- `IsMidRelabel(comments []Comment) bool`: 末尾から見て最初に見つかった `AI` が true のコメントが、`release:` / `restart:` / `advance:` のいずれかで始まる行（前後の空白は無視）を含むなら true。AI のコメントが 1 件も無ければ false。上流 `routine-common`「ブロック解除・死んだ worker の再起動は `[]` を書いてから `[stage:X]` を書く」の 2 回書きの途中を、sweep と同じ目印（`routine-sweep`「段階ラベルが無く、最新の `<!-- routine -->` コメントが `release:` / `restart:` / `advance:`」）で見分ける
- `UnblockWhen(body string) (string, bool)`: 本文の行のうち `unblock-when:` で始まる最初の行（前後の空白は無視）の、その接頭辞より後ろを空白を除いて返す。無ければ `"", false`。上流 `routine-common`「解除条件を `unblock-when:` の 1 行で明示する」の値（`comment` / `docs` / `#m`）である

#### Scenario: 最新の routine コメントが restart なら書き直しの途中
- **WHEN** `[AI: "<!-- routine -->\nadvance: stage:apply", AI: "<!-- routine -->\nrestart: 1/3"]` を `IsMidRelabel` に渡す
- **THEN** true が返る

#### Scenario: 人のコメントは飛ばして最新の routine コメントを見る
- **WHEN** `[AI: "<!-- routine -->\nrelease: stage:apply", 人: "了解しました"]` を `IsMidRelabel` に渡す
- **THEN** true が返る（sweep も人のコメントを飛ばして最新の routine コメントで続きを書くので、同じ見方に揃える）

#### Scenario: 通常の routine コメントは書き直しの途中ではない
- **WHEN** `[AI: "<!-- routine -->\nblocked-by: human\nunblock-when: comment"]` を `IsMidRelabel` に渡す
- **THEN** false が返る

#### Scenario: unblock-when の値を読む
- **WHEN** `"<!-- routine -->\nblocked-by: human\nunblock-when: docs\n\n方針を決めてください"` を `UnblockWhen` に渡す
- **THEN** `"docs"` と true が返る

#### Scenario: unblock-when が無ければ false
- **WHEN** `"<!-- routine -->\nblocked-by: #589"` を `UnblockWhen` に渡す
- **THEN** `""` と false が返る
