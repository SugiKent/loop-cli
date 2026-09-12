# design: s36-classify-reflected-pr

## Context

動機と、この change が分類を変える 3 経路は proposal.md の「Why」を見る。
現状の `PR()` は規則 2 / 3 を 1 つの分岐で書いている（`internal/classify/classify.go:99-112`）。

```go
if mode != model.ModeLabel && !assess && hasComments && !aiLatest {
	if !fresh { /* other: 人のコメントに AI が応答していない */ }
	if question { /* in-progress: 回答済み。worker が受け取り中 */ }
	/* in-progress: auto-fix が受け取り中 */
}
```

判定に使える入力はすべて `model.PR` にある。`Body`（`model.ParseUndecided` で 1 行目を読む）、`Labels`、`UpdatedAt`、
`Comments[i].CreatedAt`（`internal/model/model.go:142`、`CommentFrom` が `gh` の `createdAt` から写す）。
`fetch` も `gh` も変えずに済む。`Comments` に入るのは会話コメントだけで、inline の review コメントは `ReviewThreads` 側にある
（`internal/gh/client.go:107`）。

## Goals / Non-Goals

**Goals:**

- worker が人の回答を反映し終えた PR を、コメントの有無にかかわらず今やるタブへ戻す
- 人がレビューコメントを書いた直後の PR を、今までどおり進行中に留める（merge 候補として見せない）
- `PR()` の純粋性（I/O 無し・入力を変更しない・`now` 以外の時刻を読まない）を保つ

**Non-Goals:**

- `StaleAfter`（3 時間）の値と意味を変えること
- 規則 7（`ai-assess:requested`）を変えること。worker が反映を終えた PR の本道はこちらで、既に正しく流れる
- `Issue()` の規則 4（回答済みの question issue）を変えること。issue の回答は sweep 間隔でしか拾われず、
  issue 側には「反映し終えた」ことを表す本文の印が無い
- `Card()` / `fetch` / UI / 設定を変えること（`s30-other-grace` が触る範囲と重ねない）
- merge ガード（不変条件 5）を変えること

## Decisions

### D1. 「worker が反映を終えた」を本文 1 行目・`question`・`UpdatedAt` の 3 つで判定する

規則 2 の分岐に `!reflected(pr)` を足し、`reflected` を次の 2 条件（`question` が無いことは分岐が既に見ている）とする。

- `n, ok := model.ParseUndecided(pr.Body)` が `ok == true` かつ `n == 0`
- `pr.UpdatedAt.After(pr.Comments[len-1].CreatedAt)`

前者だけでは `apply` PR を守れない。`apply` PR の本文 1 行目は最初から `未確定の判断: 0 件` で `question` も付かないので、
人がレビューコメントを書いた瞬間に行 C（merge する）へ出る。s32 が規則 2 / 3 の時間切れを `other` にしたのは
「判定表に流すと、反映されていない人の依頼が C に見える」ためで、前者だけを条件にすると同じ害が 3 時間待たずに起きる。
後者は「人の最新コメントの後に PR 自身が動いた」ことを表し、worker が本文・ラベル・コミットのどれを動かしても真になる。
弱い代理指標であることは Risks に書き、採否は proposal.md の Q2 として人に問う。

代替案として、`Comments` の末尾より後の `<!-- routine -->` コメントを探す案（＝今の条件そのもの）、
`ReviewThreads` の更新を見る案、`gh` から timeline events を取る案がある。前 2 つは今回の欠陥そのもので、
3 つ目は 1 リポジトリあたりの `gh` 呼び出しを増やす（D-001）。

### D2. 印がある PR は規則 2 の時間切れ（`other`）にも落とさない

`reflected` は分岐の条件に置く。印がある PR は「AI が次に動く」前提の対象から外れるので、時間切れの概念ごと外れ、
判定表で分類される。3 時間経った反映済みの PR が `PR #<n> は人のコメントに AI が応答していない` と要約されるのは
事実に反するので、この置き方を採る。

### D3. `CreatedAt` のゼロ値を特別扱いしない

`pr.UpdatedAt.After(zero)` は真になるので、`createdAt` を持たない手書き fixture の PR は「印あり」と判定される。
実際の `gh` の出力には `createdAt` が必ず入り（`internal/gh/types.go:58`、`internal/gh/decode_test.go` が固定）、
手書き fixture はテストの入力なので、起こり得ない状態への防御コードは書かない（CLAUDE.md）。
既存の 2 fixture（`example` / `board`）に規則 2 / 3 に当たる PR は無く、期待値表は変わらない。

### D4. `label` 方式と `ai-assess:requested` の扱いは変えない

どちらも規則 2 / 3 の分岐に入る前に外れている（`mode != model.ModeLabel`、`!assess`）。この change は分岐の中だけを触る。

### D5. 印がある PR は判定表へそのまま流し、C にも出す

印がある緑の PR を行 C（`merge する`）に出すのが issue #46 の期待する振る舞いで、局面 C の merge ガード
（`question` / `ai-assess:requested` / N > 0 / checks 失敗 / draft のいずれかで merge を拒否する）は今までどおり動く。
代替案は、印がある PR を「その他」（優先度 6）に `PR #<n> は人のコメントの後に AI が動いた。中身を確認する` として出し、
merge を促さない形にすることで、こちらは反映が半端な PR で事故らない代わりに、緑で mergeable な PR も優先度 6 に沈む。
採否は proposal.md の Q3 として人に問う。

## Risks / Trade-offs

- **`UpdatedAt` は「worker が依頼を反映した」ことまでは保証しない。** auto-fix が途中のコミットを 1 本 push しただけでも
  印は立ち、依頼が半分しか入っていない `apply` PR が行 C に並ぶ（Why の経路 (c)）。`StaleAfter` の 3 時間は worker の
  作業窓を吸収する時間でもあったので、その役割はこの change で失われる → 反映したかどうかを GitHub の状態から区別する手段は無い。
  Q3 の選択肢 B を採ると、C の代わりに「中身を確認する」と出るので、merge を促す形にはならない
- **人自身の操作でも印が立つ。** 会話コメントを書いた後に inline review を submit すると、review コメントは `Comments` に入らない
  ため最新コメントは人のままで、`UpdatedAt` だけが後ろへ動く。`isD` は thread 末尾が人なので当たらず、行 C に出る。
  コメントの編集・ラベルの追加・title の編集も同じ → 人がその PR を開いている最中の操作なので、誤って merge する可能性は低い。
  timeline events を引けば区別できるが、取得が増える（D-001）
- **「人が会話コメントを書いただけの PR で `UpdatedAt` と `CreatedAt` が一致する」ことを実データで確認していない。**
  このリポジトリで観測できたコメントと `updatedAt` の差（PR #41 の 11 分、PR #48 の 55 秒）は、いずれも後続のラベル書き込みで
  説明できるが、一致することの確認にはなっていない。1 秒でもずれる経路があれば D1 の後者の条件は常に真になり、Q2 の選択肢 B に退化する
  → 退化しても、退化後の挙動は issue #46 が書いた期待値そのものである
- **2 回書きの途中で死んだ worker の PR は、AI 評価を 1 度も受けずに C へ出る。** 経路 (a) では `ai-assess:requested` が
  一度も付かないので、規則 7 の評価待ちを通らない → その PR は今まで 3 時間隠れた後に「その他」へ出ていたもので、
  隠れ続けるより出るほうが人の判断に近い。評価の有無は PR 詳細のラベル欄で読める
- **`docs` PR と外部からの PR は今までどおり 3 時間進行中に留まる。** これらは `未確定の判断:` の 1 行目を持たないので
  印が付かない → worker が反映を終えたと宣言していない以上、コメントの順番で判断するしかない
- **`s30-other-grace` と組み合わせると、印はあるが判定表のどの行にも当たらない PR は更新から 30 分だけ進行中タブに出る**
  → あちらの「その他になってすぐは routine に任せる」という意図どおりで、30 分後に今やるタブへ出る。矛盾しない
