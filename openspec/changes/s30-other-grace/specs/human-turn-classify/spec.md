## MODIFIED Requirements

### Requirement: どの行にも当たらないものはその他バケットに入れる
`PR()` は、進行中の規則にも判定表の行にも当たらない open PR を `Situation` `other`、`Priority` 6、`Tab` `今やる` と MUST 判定する（human-turn-signals.md: ラベルなし・コメントなしの PR、旧構成の `retro` PR など。「進行中」に混ぜると人の出番かどうかを判別できなくなる。消えて見えなくなる項目を作らない）。`in-progress` を返す規則は Requirement「キューに入れないものは進行中にする」の 7 規則だけで、それ以外の open PR を黙って落とさない。規則 2 / 3 の条件を満たしたまま時間切れになった PR も、同じ Requirement のとおり `other` にする。
`Issue()` も、`Labels` に `question` と `blocked` の両方があるのに B に当たらない issue（`Comments` が nil または空で、最新コメントが AI か人か決められない）を `other` と MUST 判定する。この issue は本来 B か進行中の規則 4 のどちらかであり、進行中に落とすと s07 の取り忘れで人待ちの issue が見えなくなる。
`Issue()` / `PR()` は時刻を見ない。open PR の `other` を最終更新からの猶予の間だけ進行中に置く判断は `Card()` が持つ（Requirement「Card は猶予内のその他の PR を進行中に置き換える」）。issue の `other` は `Card()` でも置き換えない（上記のとおり人待ちの取りこぼしであり、隠す根拠が無い）。

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
- **WHEN** `Labels` が空、`Comments` が空、`UpdatedAt` が現在時刻の open PR を `PR()` に渡す
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
| `in-progress` | 7 | 進行中 | 進行中 | 規則 1: `#<n> は AI が作業中` / 規則 2: `PR #<n> は auto-fix が受け取り中` / 規則 3: `PR #<n> は回答済み。worker が受け取り中` / 規則 4: `#<n> は回答済み。sweep 待ち` / 規則 5: `#<n> は question のみ。dispatcher の回収待ち` / 規則 6: `#<n> は段階ラベルの書き直し中。sweep 待ち` / 規則 7: `PR #<n> は AI 評価待ち` / 猶予（`Card()` の置き換え。PR のみ）: `PR #<n> はどの局面にも当たらない（更新から <M>m は様子見）` / フォールバック: `#<n> は進行中` |
| `""` | 8 | （無し。`Tab()` は空文字列） | （空文字列） | （空文字列） |

`Issue()` / `PR()` の返り値と、`Card()` が埋める open PR / Issue / Card の `Result` では、`Priority` と `Tab` が `Situation.Priority()` / `Situation.Tab()` と一致する。`MERGED` / `CLOSED` の PR の `Result` は struct のゼロ値のまま（`Priority` フィールドも 0）で、並び順に使わない。

#### Scenario: 優先度は F が最上位で other が最下位
- **WHEN** `model.SituationF.Priority()` と `model.SituationOther.Priority()` を比べる
- **THEN** 前者は 0、後者は 6 である

#### Scenario: タブの振り分け
- **WHEN** A / B / C / D / G / other / E / F / in-progress の `Tab()` を呼ぶ
- **THEN** 順に 今やる / 今やる / 今やる / 今やる / 今やる / 今やる / バックログ / 異常 / 進行中 が返る

### Requirement: Card は Issue と open PR 群のうち最上位の局面を 1 行目に出す
`internal/classify` は `Card(c model.Card, mode model.Mode, now time.Time, grace time.Duration) model.Card` を MUST 提供する。1 枚の Card は 1 つのリポジトリの issue と PR だけを持つので、`mode` はカード全体で 1 つに決まる。`now` は取得時刻、`grace` は「その他」を進行中に置く猶予で、どちらも呼び出し側（s07 `Fetch`）が渡す。`Card()` は壁時計を読まない。返り値は入力のコピーで、`Issue.Result`（`Issue` が nil でなければ）と各 `PRs[i].Result`（`State` が `OPEN` のものだけ。それ以外はゼロ値）を `Issue()` / `PR()` に同じ `mode`（`PR()` には同じ `now` も）を渡して埋め、Requirement「Card は猶予内のその他の PR を進行中に置き換える」の置き換えを open PR ごとに当て、`Canonical`（次の Requirement）を立て、`Card.Result` を次の規則で決める。
- 候補は `Issue.Result` と、`State` が `OPEN` の各 PR の `Result` のうち、`Situation` が `in-progress` でないもの
- 候補があれば、`Priority` が最小のものを `Card.Result` にする。同点なら `Issue` を優先し、次に `PRs` の並び順で先のもの
- 候補が無ければ（すべて `in-progress`、または Issue が nil で open PR も無い）`Card.Result` は `in-progress` にし、`Summary` は `PRs` の並び順で先頭の open PR の `Summary`、open PR が無ければ `Issue` の `Summary`、どちらも無ければ `進行中`（`MERGED` / `CLOSED` の PR は `Summary` が空なので採らない）

`Card.Result.Tab` がそのカードを出すタブであり、`Card.Result.Priority` がタブ内の並び順の第 1 キーである（第 2 キー以降は s08 が決める）。以下の Scenario で `now` / `grace` を書いていないものは `grace` が 0（猶予なし）である。

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

## ADDED Requirements

### Requirement: Card は猶予内のその他の PR を進行中に置き換える
`Card()` は、`PR()` で埋めた `State` が `OPEN` の各 PR の `Result` について、次の 4 つをすべて満たすものを `Situation` `in-progress`、`Priority` 7、`Tab` `進行中` に MUST 置き換える（proposal: PR のその他の大半は routine の状態機械の途中であり、最終更新から一定時間誰も触っていないものだけを人に出す）。
- `Result.Situation` が `other`
- `grace` が 0 より大きい
- その PR の `UpdatedAt` がゼロ値でない（取得できている）
- `now.Sub(UpdatedAt)` が `grace` 未満（負の値、つまり `now` が `UpdatedAt` より前の場合も含む。ちょうど `grace` は含まない）

置き換え後の `Summary` は `PR #<n> はどの局面にも当たらない（更新から <M>m は様子見）`（`<M>` は `grace` を分に切り捨てた整数）。
4 つのいずれかを満たさない PR は触らない。`Issue.Result` はこの置き換えの対象にしない（issue の `other` は詳細取得の失敗で B を取りこぼしたもので、routine の途中ではない）。`other` 以外の `Situation` も対象にならず、`grace` が 0 なら `Card()` の結果はこの change の前と 1 件も変わらない。置き換えは `Card.Result` を決める前に行うので、猶予内の `other` は候補にならず、他の要素がすべて `in-progress` ならカードも `in-progress` になる。`now.Sub(UpdatedAt)` が `grace` 以上になった取得では `other` に戻り、今やるタブに出る（s13 `desktop-notify` がこれを「増えた」と数える）。

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

#### Scenario: question と blocked がありコメントが nil の issue は猶予の対象にならない
- **WHEN** `Issue` が `stage:propose` と `blocked` と `question`、`Comments` nil、`UpdatedAt` が `now` の 5 分前で、`PRs` が空の `Card` を `mode` `sdd`、`grace` 30 分で `Card()` に渡す
- **THEN** `Issue.Result.Situation` は `other`、`Card.Result.Situation` は `other`、`Card.Result.Tab` は `今やる` である（人待ちの取りこぼしは隠さない）

#### Scenario: 猶予内のその他は他の局面を隠さない
- **WHEN** `Issue` が `stage:propose` と `wip`（進行中）、`PRs` が open の `propose` + `question` で最新コメントが AI の PR 131（A）と、`Labels` 空・`Comments` 空・`UpdatedAt` が `now` の 1 分前の open PR 132 の `Card` を `mode` `sdd`、`grace` 30 分で `Card()` に渡す
- **THEN** `Card.Result.Situation` は `A`、`PRs[1].Result.Situation` は `in-progress` である

#### Scenario: その他以外は猶予の対象にならない
- **WHEN** `Issue` が nil、`PRs` が `docs` の open PR で `UpdatedAt` が `now` の 1 分前の `Card` を `mode` `sdd`、`grace` 30 分で `Card()` に渡す
- **THEN** `Card.Result.Situation` は `G` である
