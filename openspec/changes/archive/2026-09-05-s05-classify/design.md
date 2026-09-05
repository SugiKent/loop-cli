## Context

s03-gh-client が `internal/gh` の生の型（`SearchIssue` / `SearchPR` / `Comment` / `IssueDetail` / `PRDetail` / `PRMergeState` / `ReviewThread` / `CrossReferencedPR`）と `Fake`、fixture の命名規則を定めている。s04-fixture-capture が稼働リポジトリの fixture を `internal/gh/testdata/fixtures/<alias>/` に置く（採取は利用者が実行。実採取前は s03 の `example` だけがある）。
`internal/model` と `internal/classify` はまだ無い。正本は human-turn-signals.md の判定表（行 A〜G、評価順、優先度、「キューに入れないもの」、「その他」バケット、実データ節の AI 判定と正本 PR）で、D-003 の内部構成に従い `model/`（Card・Comment・分類結果）と `classify/`（判定表を実装する純粋関数）に分ける。

human-turn-signals.md を読んで確定した事実:
- 判定表は「この順で評価し、最初に当たった行で止める」。優先度列（F=0, A=1, D=1, B=2, C=3, E=4, G=5）は並び順であり、評価順とは別
- 行 B は `question` が `blocked` に重ねて付くことを前提にし、F の縮小理由で「`question` 単独（`blocked` 無し）の issue は進行中に置く」と明記している
- D-004 は B の検知を「`question` ラベル」と短く書き、「`is:open label:question` で人待ちを全部出せる」としている。判定表が正本であり、D-004 の記述は判定表の要約と読む。この change の B は `question` + `blocked` + 最新コメントが AI を条件にし、`question` 単独は進行中にする
- 「キューに入れないもの」は 3 項目（`wip` の issue、`question` 無し PR で最新コメントが人、`question` issue で最新コメントが人）。同じ節が「PR の回答は同一セッションが即座に拾う」と書いており、`question` PR で最新コメントが人のものも人待ちではない
- AI 判定は本文マーカーのみ。author では判定できない
- 同段階の merge 済み PR が複数あれば最新が正本。古い PR の `question` を異常扱いしない

s03 の `example` fixture の内容（分類の期待値の根拠）: issue 108 は `stage:propose` + `question`（`blocked` 無し）でコメント 2 件の末尾が人 → 進行中。issue 140 はラベル無し → E。PR 131 は `propose` + `question` でコメント 1 件が `<!-- routine -->` 始まり → A。
`example` には `issue-140.json` が無い（`search-issues.json` にだけ 140 がある）。fixture テストは全 issue の `ViewIssue` を呼ぶので、この change が `issue-140.json` を手書きで足す（tasks 5.0）。s06 はこの追加を前提にする。

## Goals / Non-Goals

**Goals:**
- 判定表の行・進行中・その他を Requirement と 1 対 1 で実装し、fixture と手書きの入力でテストする（V-1）
- `internal/model` の型を後続（s06〜s17）が共有できる形で確定する。分類に使う値だけを持ち、画面の都合を持ち込まない
- s07 が Card を組み立てて `classify.Card` を呼ぶだけで 4 タブに振り分けられる状態にする

**Non-Goals:**
- この change は Issue と PR を紐づけない。PR title `[<段階>] #<n>` と本文 `Refs #n` / `Closes #n` のパース、search 結果からの Card の組み立ては s07（P1）が担当し、cross-reference による補完は s17（P2）が担当する
- D-001 の遅延取得（どの issue / PR にどの詳細を取るか）。s07 が担当する
- ラベル変遷（`LabelTimeline`）の解釈（再起動回数・段階の経過時間）。s17 が担当する
- この change は回答テンプレート `Q1: A\nQ2: A` を組み立てない（s10 が担当する）。merge ガードの理由文言は s14 が、通知の差分比較は s13 が担当する
- `sugi-loop-cli classify` サブコマンド（s06）

## Decisions

### ファイル構成

```
internal/model/model.go        # Comment / Issue / PR / Card / Result / Situation / Tab、ラベル定数、HasLabel / IssueStages / PRStages、IssueFromSearch / PRFromSearch / CommentFrom
internal/model/model_test.go   # 変換と段階ラベル抽出
internal/model/parse.go        # IsAI / ParseUndecided / LatestBlockedBy / ParseQuestions
internal/model/parse_test.go   # spec card-model の Scenario をそのまま
internal/classify/classify.go  # Issue / PR / ChecksGreen と、進行中の 5 規則・行 A〜G の述語
internal/classify/card.go      # Card（集約と Canonical）
internal/classify/classify_test.go  # 手書きの model.Issue / model.PR で行ごとの Scenario
internal/classify/card_test.go      # Card 集約・Canonical・入力不変
internal/classify/fixture_test.go   # fixtures/* を Fake で読み、期待値表と突き合わせる
```

### model は生の型を写さず、分類に使う値だけを持つ

`model.Issue` / `model.PR` は `gh.SearchIssue` / `gh.SearchPR` を埋め込まず、`Repo` / `Number` / `Title` / `URL` / `Body` / `Labels []string` / `UpdatedAt` を写す。理由は 2 つ。(1) PR は search 由来（open）と cross-reference 由来（merged。s17）の 2 経路で入り、生の型が違う（`SearchPR` と `CrossReferencedPR`）。共通の型に写しておけば分類は経路を知らずに済む。(2) `Labels` を `[]string` にすると `HasLabel` が 1 行で済み、テストの入力が `[]string{"propose", "question"}` と短く書ける。
一方 `MergeState *gh.PRMergeState` と `ReviewThreads []gh.ReviewThread` は写さずそのまま持つ。使うのは分類（C / D）と s14 / s16 だけで、写しても同じフィールドが並ぶだけだからである。`model` が `gh` を import する方向の依存は許容する（逆は無い）。

`Comments []Comment` の nil は「未取得」、長さ 0 は「取得したがコメント無し」だが、分類はどちらも同じに扱う（最新コメントが無いので「最新コメントが AI / 人」のどちらも偽）。区別が必要になるのは表示（s09）であり、分類には要らない。

### 評価順は「進行中の除外 → 判定表 A〜G → フォールバック」

判定表の「この順で評価し、最初に当たった行で止める」は A〜G の行の順序を定めているが、「キューに入れないもの」をどこで評価するかは書かれていない。除外を先に評価する理由:
- `question` だけの issue（`blocked` も `stage:*` も無い）は行 E（`stage:*` なし、`blocked` なし）に当たる。しかし F の縮小理由が「`question` 単独の issue は進行中に置く」と明記している。除外を先に評価しないとこれが満たせない
- 「キューに入れない」という表現は、キューに載せる判定の前に外すと読むのが自然
- `question` issue で最新コメントが人（除外 3）は行 B の否定であり、どちらを先にしても結果は同じ

この順序の帰結として、`question` 無しの段階 PR で最新コメントが人のものは、mergeable で checks 緑でも進行中（auto-fix 受け取り中）になる。文書の除外 2 をそのまま採る。
除外 1（段階ラベル + `wip`）は `IssueStages` が 1 件のときだけ当てる。段階ラベルが 2 つ以上の issue は `wip` があっても F（最上位）に出す。F は「TUI から自動修復はしない」異常であり、`wip` の陰に隠すと人が気づけないため。
`question` PR で最新コメントが人のものは進行中にする（spec の規則 3）。文書の除外リストには無いが、同じ節が「PR の回答は同一セッションが即座に拾う」と書いており、人待ちでも異常でもない。「その他」に出すと今やるタブに人の出番でない項目が並ぶ。

行の順序は文書どおり A → B → C → D → E → F → G。F の優先度は 0（最上位）だが評価順は 6 番目なので、issue に段階ラベルが 2 つあっても `question` + `blocked` + AI 最新なら B になる。文書の順序をそのまま採り、並び順（優先度）と評価順を混ぜない。

C は `IsDraft` を見ない。draft の merge 拒否は s14 の merge ガード（human-turn-signals.md 不変条件 5）が持ち、分類は判定表の行 C の条件（段階ラベル・未確定 0 件・`question` 無し・checks 緑・mergeable）だけを実装する。draft の PR が C に出て `m` を押しても s14 が理由を表示して拒否する。

### 詳細が nil のときは条件不成立

D-001 は詳細取得を「`question` の issue / PR はコメント、merge 候補 PR は merge 状態、`apply` PR は review thread」に絞る。分類はどの詳細が来るか知らず、nil なら「その詳細を必要とする条件は偽」とする。これで s07 が詳細を取り忘れた場合、`question` PR は A にならず「その他」に落ちて画面に出る（黙って消えない）。issue も同じで、`question` + `blocked` の issue は `Comments` が nil なら B にも進行中の規則 4 にもならないので、フォールバックで「進行中」に落とさず「その他」に出す。フォールバックで進行中にする issue は `question` + `blocked` 以外に限る。
D-001 の絞り込みと除外 2（`question` 無し PR で最新コメントが人）は噛み合わない。除外 2 は `question` 無し PR のコメントを見る必要があるが、D-001 は `question` PR のコメントしか取らない。この change は「コメントが無ければ除外 2 は偽」と定めるだけで、取得範囲の判断は s07 に委ねる（未決事項に記す）。

### 2 層の API

`classify.Issue` / `classify.PR` は 1 件の判定、`classify.Card` は Card 単位の集約。Card の集約規則は「候補（進行中でない open な要素）の優先度最小、同点は Issue 優先、次に `PRs` の並び順」の 1 つだけにする。mvp.md の「カードは『いま人が何をすべきか』を 1 行目に出す」は勝者の `Summary` をそのまま `Card.Result.Summary` にすることで満たす。
`Card` は入力のコピーを返す（`PRs` スライスも新しく作る）。呼び出し側がスナップショット（D-002）を保持したまま再分類できるようにするため。

### Summary の文言は Situation と番号だけで決める

`Summary` は「局面 → 定型文 + 番号」で、タイトル・リポジトリを含めない（画面の列で別に出る）。進行中だけは 5 つの規則とフォールバックで文言を変え、なぜ進行中なのかを画面で読めるようにする（文書の括弧書き「AI が動いている最中」「auto-fix が受け取り中」「sweep を待っている」をそのまま使う）。文言は spec の表で固定し、テストで文字列一致を見る。

### 正本 PR は番号最大

`CrossReferencedPR` に日時が無いので、同段階の merge 済み PR のうち番号が最大のものを最新（正本）とする。PR 番号は単調増加であり、同じ issue に対する propose PR の作り直しは常に後の番号になる。

### fixture テストは期待値表を網羅させる

`fixture_test.go` は `../gh/testdata/fixtures/` 直下の全ディレクトリを走査し、期待値表（`map[string]map[string]model.Situation`。キーは `<alias>` と `issue-<n>` / `pr-<n>`）と突き合わせる。表に無い alias・表に無い issue / PR・fixture に無い項目はすべて失敗にする。これで s04 で採取した fixture が「期待値を書くまでテストが通らない」状態になり、採取後の期待値記入（tasks 5.2）を忘れられない。期待値は分類器の出力を写さず、文書の判定表を手で当てて書く（写すとテストが実装の複製になる）。
各 issue / PR の詳細は fixture に全件あるので（s04 の採取範囲）、テストは D-001 の絞り込みをせず全詳細を入れて分類する。

## Risks / Trade-offs

- [除外を先に評価するため、merge 可能な PR に人がコメントすると進行中に落ちる] → 文書の除外 2 をそのまま採った結果。実運用で「merge したい PR が進行中にある」ことが起きたら human-turn-signals.md 側を直してから追従する
- [`question` PR で最新コメントが人のものを進行中にするのは文書の除外リストに無い] → 「PR の回答は同一セッションが即座に拾う」を根拠に spec の規則 3 として足した。セッションが落ちて `question` が残り続けた場合は進行中に留まり続ける。実運用で起きたら human-turn-signals.md 側に「一定時間経過で異常」を足してから追従する
- [D-001 の取得範囲と除外 2 の不整合] → この change は不整合を s07 の未決事項として引き継ぐ。分類は nil を「条件不成立」にするだけで、取得範囲を仮定しない
- [`ChecksGreen` が `SKIPPED` / `NEUTRAL` を緑に含める] → GitHub の UI でも merge を妨げない値。`PENDING` / `IN_PROGRESS`（`Conclusion` が空）は緑にしない
- [F を PR にも適用する] → 文書は「段階ラベルが 2 つ以上」としか書いておらず issue に限定していない。PR に `propose` と `apply` が同時に付く事故も同じ壊れ方なので同じ扱いにする
- [期待値表を Go のマップで持つ] → fixture の件数が数十件なら十分。JSON 等の別ファイルにすると型検査が効かず、`Situation` の綴り間違いが実行時まで分からない

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| 「キューに入れないもの」の評価位置 | 判定表 A〜G より先 | 上記 Decisions。`question` 単独 issue の扱いを満たすため |
| どの行にも当たらない issue の扱い | `question` + `blocked` なら「その他」（`Summary` は `#<n> はどの局面にも当たらない`）、それ以外は進行中（`Summary` は `#<n> は進行中`） | 文書は PR の「その他」しか定めていない。段階ラベル付きは dispatcher / worker の担当中。ただし `question` + `blocked` は本来 B か回答済みであり、`Comments` 未取得で進行中に落とすと人待ちが消えて見えなくなる |
| 「その他」バケットのタブと優先度 | 今やるタブ、優先度 6（最下位） | 文書は「必ず画面に出す」「進行中に混ぜない」。今やるの末尾に置くのが両方を満たす |
| `question` PR で最新コメントが人 | 進行中（spec の規則 3。`Summary` は `PR #<n> は回答済み。worker が受け取り中`） | 除外リストに無いが、human-turn-signals.md「PR の回答は同一セッションが即座に拾う」。人待ちでも異常でもない |
| `wip` issue に段階ラベルが 2 つ以上 | F（除外 1 は `IssueStages` が 1 件のときだけ） | F は最上位の異常。`wip` の陰に隠さない |
| `wip` が付いた issue（段階ラベル 1 件）に `question` + `blocked` + 最新コメント AI も揃っている | B より進行中（除外 1）を優先する | dispatcher が `blocked-by: human` を書くときに `wip` を外す前提。`wip` が残っていれば AI がまだ動いている |
| `IsAI` の先頭判定 | 字面どおり `<!-- routine -->` / `&lt;!-- routine --&gt;` で始まる（TrimSpace しない）。`## PR リスク評価` は行頭から始まる行が本文のどこかにあれば AI | dispatcher の「`<!-- routine -->` で始まらないコメントは人の回答」と同じ判定にする。文書は見出しの位置を定めていないので行の位置は問わない |
| 詳細（`Comments` / `MergeState` / `ReviewThreads`）が nil のとき | その詳細を要する条件は偽 | 上記 Decisions。黙って落とさず「その他」に出る |
| D-001 の取得範囲と除外 2 の不整合 | この change では nil を偽にするだけ。取得範囲は s07 が決める | D-001 は `question` PR のコメントしか取らないが、除外 2 は `question` 無し PR のコメントを見る |
| `ChecksGreen` の緑の定義 | `CheckRun` は `SUCCESS` / `SKIPPED` / `NEUTRAL`、`StatusContext` は `SUCCESS`、要素 0 件は緑 | 上記 Risks |
| C に `IsDraft` を含めるか | 含めない（draft でも C になる） | draft の merge 拒否は s14 のガード（不変条件 5）が持つ。分類は判定表の行 C の条件だけを実装する |
| F を PR に適用するか | 適用する（`PRStages` が 2 件以上） | 上記 Risks |
| 最新コメントの決め方 | `Comments` の末尾 | `gh` は作成順で返す。`CreatedAt` で並べ直さない |
| 正本 PR の決め方 | 同段階の `MERGED` PR のうち番号最大。段階ラベルが 1 件でない `MERGED` PR は候補にしない | `CrossReferencedPR` に日時が無い |
| `MERGED` / `CLOSED` PR の `Result` | struct のゼロ値（`Situation` `""`。`Situation("").Priority()` は 8 だが `Result.Priority` フィールドは 0 のまま） | 分類対象は open だけ。ゼロ値なら「分類していない」と読める |
| 進行中の `Priority` | 7 | キューに出ないが、`Priority()` を全 Situation で定義してソートを単純にする |
| `Situation.Kind()` の文言 | 質問（A / D）/ 方針 / merge（C / G）/ todo 候補 / 異常 / その他 / 進行中 | mvp.md の種別 5 つ（質問・方針・merge・todo 候補・異常）に寄せる。D は優先度も A と同じ 1 なので `質問` に含め、その他 / 進行中だけを足す |
| `ParseUndecided` の 1 行目 | 先頭の空行を除いた最初の行。`:` は ASCII のみ | 文書の例が `未確定の判断: 0 件`。全角コロンの例は無い |
| `ParseQuestions` の選択肢の区切り | `:` と `：` の両方 | mvp.md の例は `選択肢 A（推奨）: 含めない。` で ASCII だが、日本語本文で全角が混ざりやすい。ここだけ両方許す |
| 期待値表の置き場所 | `internal/classify/classify_test.go` 内の Go のマップ | 上記 Risks |
| `Card.Result` の候補が無いときの `Summary` | 先頭の open PR の `Summary`、open PR が無ければ Issue の `Summary`、どちらも無ければ `進行中` | 進行中の理由を画面で読めるようにする。`MERGED` / `CLOSED` の PR は `Summary` が空なので採らない |
