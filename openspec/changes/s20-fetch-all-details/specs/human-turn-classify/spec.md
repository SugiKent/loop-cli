## MODIFIED Requirements

### Requirement: 分類は純粋関数で、進行中の除外・判定表の順・フォールバックの順に評価する
`internal/classify` は `Issue(is model.Issue) model.Result` と `PR(pr model.PR) model.Result` を MUST 提供する。どちらも入力だけから結果を決め、I/O を行わず、入力を変更しない。評価は次の順で行い、最初に当たった規則で止める（human-turn-signals.md「分類はこの順で評価し、最初に当たった行で止める」）。
1. 「キューに入れないもの」（Requirement「キューに入れないものは進行中にする」の 5 規則）
2. 判定表の行を A → B → C → D → E → F → G の順（issue に当たり得るのは B / E / F、PR に当たり得るのは A / C / D / F / G）
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

### Requirement: どの行にも当たらないものはその他バケットに入れる
`PR()` は、進行中の規則にも判定表の行にも当たらない open PR を `Situation` `other`、`Priority` 6、`Tab` `今やる` と MUST 判定する（human-turn-signals.md: ラベルなし・コメントなしの PR、旧構成の `retro` PR など。「進行中」に混ぜると人の出番かどうかを判別できなくなる。消えて見えなくなる項目を作らない）。`in-progress` を返す規則は上記 5 規則だけで、それ以外の open PR を黙って落とさない。
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
