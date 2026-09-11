## MODIFIED Requirements

### Requirement: 今やるタブの差分は主体のキーで比較する純粋関数で求める
`internal/ui` は非公開の純粋関数 `addedNow(prev, next []model.Card) []model.Card` を MUST 持つ。`prev` と `next` のそれぞれから `Card.Result.Tab` が `model.TabNow` の Card だけを取り、各 Card のキーを s08 `Subject` の（リポジトリ、番号、主体が PR か）の 3 つ組とし、`next` にあって `prev` に無いキーの Card を `next` での並び順で返す。`prev` にあって `next` に無い Card（減ったもの）と、両方にある Card（同じ。`Title` / `UpdatedAt` / `Summary` が変わっていても同じ）は返さない。他のタブの Card は比較に入れない。`gh` も時刻も読まない。
主体の種別をキーに含めるので、同じ Card（issue 108 + PR 131）の主体が PR 131（局面 A）から issue 108（局面 B）に移ったときは「増えた」と数える（人がすべきことが変わっているため。design.md 未決事項の既定値）。局面 `other`（その他）の PR も今やるタブにあるので比較に入る。s30 の猶予で進行中タブに置かれていた `other` の PR の Card が、猶予が明けた取得で今やるタブに現れたときも、`prev` では今やるタブに無かったので「増えた」と数える。これが「その他のまま誰も触っていない」項目の知らせになる。

#### Scenario: 増えたカードだけが返る
- **WHEN** `prev` が今やるタブの Card `org/app PR#131` と `org/app #91` の 2 枚、`next` が `org/app #91`（`UpdatedAt` は新しい）と `org/web PR#88` と バックログの `org/app #140` の 3 枚で `addedNow` を呼ぶ
- **THEN** 返るのは `org/web PR#88` の 1 枚である（PR 131 は減ったので返らず、issue 91 は同じなので返らず、issue 140 はバックログなので比較しない）

#### Scenario: 同じ集合なら空
- **WHEN** `example` の `Result.Cards` を `prev` と `next` の両方に渡す
- **THEN** 長さ 0 である

#### Scenario: 前回が空なら今やるの全部が返る
- **WHEN** `prev` が `nil`、`next` が `example` の `Result.Cards` で呼ぶ
- **THEN** 今やるタブの issue 108 の Card（主体 PR 131）の 1 枚が返る

#### Scenario: 主体が PR から Issue に移ると増えたと数える
- **WHEN** `prev` が主体 PR 131 の issue 108 の Card、`next` が同じ issue 108 で `Result` を局面 `B`（主体が Issue、タブは今やる）にした Card で呼ぶ
- **THEN** issue 108 の Card の 1 枚が返る

#### Scenario: 猶予が明けて今やるに現れたその他は増えたと数える
- **WHEN** `prev` が PR 61 単独の Card で `Result` が `in-progress`（猶予中。タブは進行中）、`next` が同じ PR 61 単独の Card で `Result` が `other`（タブは今やる）で `addedNow` を呼ぶ
- **THEN** PR 61 の Card の 1 枚が返る
