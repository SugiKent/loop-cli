# s36-classify-reflected-pr

issue: #46

## Why

進行中の規則 2（`classify.PR`）は「最新コメントが人か」だけで open PR を進行中タブに落としており、
worker が人の回答を反映し終えた印（本文 1 行目 `未確定の判断: 0 件`、`question` ラベルが落ちていること）を見ていない。
行 C（`isC`）は同じ本文 1 行目を `model.ParseUndecided` で読んで merge 候補かを決めている（`internal/classify/classify.go:154`）ので、
同じ PR について、片方は本文を信頼し、片方はコメントの順番だけを見ている。

issue #46 は PR #40 を再現として挙げるが、実データを読み直すと **#40 を 17:01 から隠していた規則は 7（AI 評価待ち）である**。
worker は本文を `未確定の判断: 0 件` にするとき `ai-assess:requested` も付ける規約で（`routine-common/references/worker.md`）、
`ai-assess:requested` が付いた PR には規則 2 が当たらない（`classify.go:102` の `!assess`）。#40 には 18:40:54 の
`## PR リスク評価` コメントが残っており、ラベルが実際に付いていたことを示す。規則 2 が隠すのは、そのラベルが**無い**次の 3 経路である。

- (a) worker が 2 回書き（`[<段階>]` → `[<段階>, ai-assess:requested]`）の途中で死に、`ai-assess:requested` が付かなかった PR
- (b) 人が `ai-assess:requested` を手で外した PR（2026-09-11 の #37 / #39 / #40 / #41 がこれ。当時は外し手の Routine が無かった）
- (c) `apply` PR（本文 1 行目は最初から `未確定の判断: 0 件`）で、人のコメントの後に auto-fix が push した PR

どの経路でも、PR は 3 時間（`StaleAfter`）今やるタブから消え、その後 `PR #<n> は人のコメントに AI が応答していない` という
事実に反する要約で「その他」に出る。「消えて見えなくなる項目を作らない」（human-turn-signals.md）に反する。

## What Changes

- 進行中の規則 2 に「worker が反映を終えた印が無い」ことを条件として足す。印は次の 3 つが揃うこと。
  `question` ラベルが無い（規則 2 の既存条件）、本文 1 行目が `未確定の判断: 0 件`、`UpdatedAt` が `Comments` の末尾の
  `CreatedAt` より後。印がある open PR は、最新コメントが人でも進行中に落とさず判定表（C / D / F / G / その他）へ流す
- 印が無い PR は今までどおり規則 2 が進行中にする。3 時間動かなければ、今までどおり `other`（要約は
  `PR #<n> は人のコメントに AI が応答していない`）へ落とす
- 本文 1 行目に `未確定の判断:` の行が無い PR（`ParseUndecided` が `ok=false`。外部から来た PR や `docs` PR）は
  worker が「反映し終えた」と宣言していないので、今までどおり規則 2 の対象にする
- 規則 3（`question` 付き PR）は変えない。`question` が付いているあいだ worker は反映を終えていない
- `label` 方式は今までどおり規則 2 / 3 を適用しない。`ai-assess:requested` が付いた PR も今までどおり規則 7 が扱う
- `docs/domain/issue-driven-sdd/human-turn-signals.md` の「キューに入れないもの」の 2 つ目の項と、時間切れを説明する項に
  この条件を書き足す。`docs/mvp` は凍結されているので触らない（CLAUDE.md）
- `Issue()`・`Card()`・`fetch`・UI・設定と関数シグネチャは変えない（必要な入力はすべて `model.PR` にある）

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `human-turn-classify`: Requirement「キューに入れないものは進行中にする」の規則 2 に、worker が反映を終えた PR を
  除外する条件を足す（規則 3 は `question` が付いている PR の規則なので、条件は文章で「適用しない」と書くだけで挙動は変わらない）

## Impact

- `internal/classify/classify.go`: `PR()` の規則 2 / 3 の分岐（`classify.go:102`）に条件を 1 つ足す。`isC` / `Card` / `Issue` は変えない
- `internal/classify/classify_test.go`: 規則 2 の Scenario を足し、既存の 3 時間の Scenario に人のコメント時刻の指定を足す
- `internal/classify/fixture_test.go`: 期待値表は変えない（`board` は `label` 方式、`example` の `pr-131` は最新コメントが AI）
- `docs/domain/issue-driven-sdd/human-turn-signals.md`: 「キューに入れないもの」の記述・変更履歴・最終更新日
- 変えないもの: `internal/model`（`Comment.CreatedAt` は `model.go:142` に既にあり、`CommentFrom` が写している）、
  `internal/fetch`、`internal/ui`、`internal/action`、`internal/config`、`README.md`

## 確定した判断

- **worker が反映を終えた印は「本文 1 行目 `未確定の判断: 0 件`」と「`question` ラベルが落ちていること」の 2 つ。**
  `routine-common`（「PR の `question` は本文 1 行目の `未確定の判断: N 件` と常に一致させる。N > 0 なら付いており、N = 0 で外す」）と
  `references/worker.md`「本文の 1 行目」が正本。行 C も同じ 2 つを見ている（`internal/classify/classify.go:149-156`）
- **`ai-assess:requested` が付いた PR は規則 7 のまま。** worker は反映し終えてからこのラベルを付け、`assess-pr-risk` は
  評価コメントを投稿してからラベルを外す（`.claude/skills/assess-pr-risk/SKILL.md` 手順 3 → 4）。投稿後は最新コメントが AI になるので、
  規則 2 に当たらず判定表へ進む。この経路は既に正しく、この change は触らない
- **worker がコメントを返す PR は今の実装でも正しく流れる。** PR #41 は 17:46:33 に反映完了のコメントを投稿しており、
  最新コメントが AI なので規則 2 に当たらない。この change が分類を変えるのは Why の (a) / (b) / (c) の 3 経路だけである
- **本文の編集とラベルの取り外しで PR の `updatedAt` は動く。** PR #41 は最新コメント 17:46:33 に対し `updatedAt` が 17:57:31
  （人によるラベルの取り外し）、PR #48 はコメント 18:11:55 に対し `updatedAt` が 18:12:50（worker のラベル書き込み）で、
  コメント以外の書き込みが `updatedAt` を動かすことを示す
- **人の最新コメントの時刻は取得済みの入力から読める。** `model.Comment.CreatedAt`（`internal/model/model.go:142`）を
  `model.CommentFrom`（同 `:294`）が `gh` の `createdAt` から写しており、PR のコメントも同じ経路で入る。
  `fetch` も `gh` も変えずに判定できる
- **`model.PR.Comments` に inline の review コメントは入らない。** `gh pr view --json comments`（`internal/gh/client.go:107`）が
  返すのは会話コメントだけで、inline は `ReviewThreads` 側にある。規則 2 が見る「最新コメント」は会話コメントである
- **進行中の `StaleAfter`（3 時間）は変えない。** この change は「時間切れを待たずに今やるへ戻す」ものであり、
  時間切れが持つ意味（Routine が止まった PR を見えなくしない）を s32 のまま残す

## 先行 change との関係

並行中の `s30-other-grace`（issue #43。PR #47 が apply 段階）と同じ capability を触るが、コードの衝突は無い。
あちらは `card.go` の `Card()` に猶予を足し、`classify.go` は変えないと tasks 3.1 に明記している。触る Requirement も重ならない。

意味の上では次の 2 点が重なるので、**後から archive する側が文章を合わせる**。

- この change は「印はあるが判定表のどの行にも当たらない」PR（checks が pending、`MergeState` の取得失敗、段階ラベル無し）を
  新しく作る。`s30-other-grace` の猶予（既定 30 分）はこれにかかるので、worker が動いた直後の 30 分は進行中タブ、その後で今やるタブに出る
- `s30-other-grace` の ADDED Requirement は猶予の対象を「routine の状態機械の途中」と説明している。この change の後は
  「反映は終えたが merge できない PR」も対象に入るので、後から archive する側がその説明を書き足す
- `docs/domain/issue-driven-sdd/human-turn-signals.md` の「キューに入れないもの」の節・変更履歴・最終更新行は両方が触る。
  後から実装する側が相手の段落を読んでから書き足す

## 人に問うて確定した判断

PR #49 のコメントで 3 問とも選択肢 A に決まった（2026-09-12）。

- **分類器を直す。** 規則 2 が PR を隠す 3 経路（Why の (a) / (b) / (c)）は運用の修正では消えず、3 時間後に出るときの要約
  `PR #<n> は人のコメントに AI が応答していない` は反映を終えた PR について事実に反する
- **「人の最新コメントより後に PR が動いた」を印の条件に入れる。** これが無いと、`apply` PR は本文 1 行目が最初から
  `未確定の判断: 0 件` なので、人がレビューコメントを書いた瞬間に `merge する` として今やるタブへ並ぶ
- **印がある PR は判定表へそのまま流す。** 緑で mergeable な PR は行 C（`merge する`）に並ぶ。「その他」に出して
  merge を促さない案は採らない

## 明示的に延期した判断と残るリスク

- **`UpdatedAt` は「worker が依頼を反映した」ことまでは保証しない。** auto-fix が途中のコミットを 1 本 push しただけでも
  印は立ち、依頼が半分しか入っていない `apply` PR が行 C に並ぶ。`StaleAfter` の 3 時間が吸収していた worker の作業窓は、
  この change で失われる。区別するには timeline events が要り、1 リポジトリあたりの `gh` 呼び出しが増える（D-001）ので延期する
- **人自身の操作でも印が立つ。** 会話コメントの後に inline review を submit すると、review コメントは `Comments` に入らない
  ため最新コメントは人のままで、`UpdatedAt` だけが後ろへ動く。コメントの編集・ラベルの追加・title の編集も同じ。
  いずれもその PR を開いている最中の操作なので、誤って merge する可能性は低いとみて受け入れる
- **「人が会話コメントを書いただけの PR で `UpdatedAt` と `CreatedAt` が一致する」ことは実データで確認できていない。**
  観測できた差（PR #41 の 11 分、PR #48 の 55 秒）はどちらも後続のラベル書き込みで説明できるが、一致の確認にはなっていない。
  ずれる経路があれば印の 3 つ目の条件は常に真になり、issue #46 が書いた期待値どおりの挙動に退化する
- **2 回書きの途中で死んだ worker の PR は、AI リスク評価を 1 度も受けずに行 C へ出る。** 経路 (a) では
  `ai-assess:requested` が付かないので規則 7 を通らない。今までは 3 時間隠れた後に「その他」へ出ていたもので、
  評価の有無は PR 詳細のラベル欄で読める
- **`docs` PR と外部から来た PR は今までどおり 3 時間進行中に留まる。** 本文 1 行目に `未確定の判断:` を持たないので印が付かない
- **`CreatedAt` がゼロ値の入力は「印あり」に倒れる**（design.md D3）。実データでは `createdAt` が必ず入るので防御コードは書かない
- **`s30-other-grace` との archive 順は決めない。** 後から archive する側が、「その他」の猶予の説明と
  `human-turn-signals.md` の同じ節を読み合わせて書き足す（「先行 change との関係」）
