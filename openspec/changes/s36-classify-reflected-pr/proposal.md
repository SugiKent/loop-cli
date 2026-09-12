# s36-classify-reflected-pr

issue: #46

## Why

進行中の規則 2 / 3（`classify.PR`）は「最新コメントが人か」だけで open PR を進行中タブに落としており、
worker が人の回答を反映し終えた印（本文 1 行目 `未確定の判断: 0 件` と `question` ラベルが落ちていること）を見ていない。
worker は反映のたびに `<!-- routine -->` コメントを返すとは限らず、本文とラベルだけを更新して終えることがある。
その結果、人が答えて worker が仕事を終えた PR が `StaleAfter`（3 時間）のあいだ今やるタブから消える。

行 C（`isC`）は同じ本文 1 行目を `model.ParseUndecided` で読んで merge 候補かを決めている（`internal/classify/classify.go:154`）。
同じ PR について、片方は本文を信頼し、片方はコメントの順番だけを見ているのが一貫していない。

## What Changes

- 進行中の規則 2 / 3 に「worker がまだ受け取っていない」ことの条件を足す。`Labels` に `question` が無く、
  本文 1 行目が `未確定の判断: 0 件` で、かつ PR が人の最新コメントより後に動いている（`UpdatedAt` が
  最新コメントの `CreatedAt` より後）open PR は、最新コメントが人でも進行中に落とさず、判定表（C / D / F / G / その他）へ流す
- `未確定の判断: N 件`（N > 0）、`question` が付いたまま、または人のコメントの後に PR が動いていない PR は今までどおり
  規則 2 / 3 が進行中にする。3 時間動かなければ、今までどおり `other`（要約は `PR #<n> は人のコメントに AI が応答していない`）へ落とす
- 本文 1 行目に `未確定の判断:` の行が無い PR（`ParseUndecided` が `ok=false`。外部から来た PR や `docs` PR）は
  worker が「反映し終えた」と宣言していないので、今までどおり規則 2 / 3 の対象にする
- `label` 方式は今までどおり規則 2 / 3 を適用しない。`ai-assess:requested` が付いた PR も今までどおり規則 7 が扱う
- `docs/domain/issue-driven-sdd/human-turn-signals.md` の「キューに入れないもの」の 2 つ目の項と、時間切れを説明する項に
  この条件を書き足す。`docs/mvp` は凍結されているので触らない（CLAUDE.md）
- `Issue()`・`Card()`・`fetch`・UI・設定は変えない。関数のシグネチャも変えない（必要な入力はすべて `model.PR` にある）

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `human-turn-classify`: Requirement「キューに入れないものは進行中にする」の規則 2 に、worker が反映を終えた PR を
  除外する条件を足す（規則 3 は `question` が付いている PR の規則なので、条件は文章で「適用しない」と書くだけで挙動は変わらない）

## Impact

- `internal/classify/classify.go`: `PR()` の規則 2 / 3 の分岐（`classify.go:102`）に条件を 1 つ足す。`isC` / `Card` / `Issue` は変えない
- `internal/classify/classify_test.go`: 規則 2 / 3 の Scenario を足す。`fixture_test.go` の期待値表は変えない
  （`example` / `board` の PR は規則 2 / 3 に当たらない）
- `docs/domain/issue-driven-sdd/human-turn-signals.md`: 「キューに入れないもの」の記述と最終更新日
- 変えないもの: `internal/model`（`Comment.CreatedAt` は `model.go:142` に既にあり、`CommentFrom` が写している）、
  `internal/fetch`、`internal/ui`、`internal/action`、`internal/config`、`README.md`

## 確定した判断

- **worker が反映を終えた印は「本文 1 行目 `未確定の判断: 0 件`」と「`question` ラベルが落ちていること」の 2 つ。**
  issue-driven-sdd の `routine-common`（「PR の `question` は本文 1 行目の `未確定の判断: N 件` と常に一致させる。
  N > 0 なら付いており、N = 0 で外す」）と `references/worker.md`「本文の 1 行目」が正本。行 C も同じ 2 つを見ている
  （`internal/classify/classify.go:149-156`）
- **本文の編集とラベルの取り外しで PR の `updatedAt` は動く。** issue #46 の再現（PR #40）で、人のコメントが 16:59、
  worker の本文・ラベル更新が 17:01、進行中に留まったのが 20:01 まで。20:01 は `StaleAfter`（3 時間）を `UpdatedAt` から
  数えた時刻なので、`UpdatedAt` が 17:01 に動いたことがこの事象自身の証拠になる（`classify.go:96`）
- **人の最新コメントの時刻は取得済みの入力から読める。** `model.Comment.CreatedAt`（`internal/model/model.go:142`）を
  `model.CommentFrom`（同 `:294`）が `gh` の `createdAt` から写しており、PR のコメントも同じ経路で入る。
  `fetch` も `gh` も変えずに判定できる
- **`label` 方式は対象外。** 規則 2 / 3 自体を `label` に適用しない（`classify.go:102` の `mode != model.ModeLabel`）ので、
  この change でも触らない。`human-turn-signals.md`「ILD でキューに入れないもの」に理由がある
- **`ai-assess:requested` が付いた PR は規則 7 のまま。** worker は反映し終えてからこのラベルを付けるので、
  規則 2 / 3 より先に規則 7 で扱うのが既に正しい（`classify.go:102` の `!assess`）
- **進行中の `StaleAfter`（3 時間）は変えない。** この change は「時間切れを待たずに今やるへ戻す」ものであり、
  時間切れが持つ意味（Routine が止まった PR を見えなくしない）を s32 のまま残す
- **並行中の `s30-other-grace` とは衝突しない。** あちらは `Card()` と `other` の猶予を触り、`classify.go` は変えないと
  proposal に明記している（`openspec/changes/s30-other-grace/tasks.md` 3.1）。触る Requirement も重ならない

## 未確定の判断

### Q1. 「worker が受け取っていない」をどう判定するか

- 選択肢 A（推奨）: **本文 1 行目 `未確定の判断: 0 件` ＋ `question` 無し ＋ PR が人の最新コメントより後に動いている**、の
  3 つが揃ったときだけ規則 2 / 3 を飛ばす。PR #40 は 3 つとも満たすので今やるタブへ戻る。人がレビューコメントを書いた直後の
  `apply` PR（本文は最初から `未確定の判断: 0 件`）は 3 つ目を満たさないので今までどおり進行中に留まり、worker が
  直して push（または本文・ラベルを更新）した時点で今やるタブに出る
- 選択肢 B: issue #46 の本文どおり、飛ばす条件を「本文 1 行目が `未確定の判断: 0 件`」と「`question` ラベルが無い」の
  2 つだけにする。実装は 1 行短い。ただし `apply` PR は最初から `未確定の判断: 0 件` で `question` も無いので、人が PR にレビューコメントを
  書いた瞬間に、その PR がキュー画面の今やるタブへ `merge する` として並ぶ（依頼した本人に「merge しろ」と出る）。
  s32 が規則 2 / 3 の時間切れを `other` にした理由（「判定表に流すと、反映されていない人の依頼が C に見える」）が
  3 時間待たずに起きる
- 依存: なし
