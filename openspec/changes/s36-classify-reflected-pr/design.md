# design: s36-classify-reflected-pr

## Context

動機は proposal.md の「Why」を見る。現状の `PR()` は、規則 2 / 3 を 1 つの分岐で書いている（`internal/classify/classify.go:99-112`）。

```go
if mode != model.ModeLabel && !assess && hasComments && !aiLatest {
	if !fresh { /* other: 人のコメントに AI が応答していない */ }
	if question { /* in-progress: 回答済み。worker が受け取り中 */ }
	/* in-progress: auto-fix が受け取り中 */
}
```

判定に使える入力はすべて `model.PR` にある。`Body`（`model.ParseUndecided` で 1 行目を読む）、`Labels`、`UpdatedAt`、
`Comments[i].CreatedAt`（`internal/model/model.go:142`、`CommentFrom` が `gh` の `createdAt` から写す）。
`fetch` も `gh` も変えずに済む。

## Goals / Non-Goals

**Goals:**

- worker が人の回答を反映し終えた PR を、コメントの有無にかかわらず今やるタブへ戻す
- 人がレビューコメントを書いた直後の PR を、今までどおり進行中に留める（merge 候補として見せない）
- `PR()` の純粋性（I/O 無し・入力を変更しない・`now` 以外の時刻を読まない）を保つ

**Non-Goals:**

- `StaleAfter`（3 時間）の値と意味を変えること
- `Issue()` の規則 4（回答済みの question issue）を変えること。issue の回答は sweep 間隔でしか拾われず、
  issue 側には「反映し終えた」ことを表す本文の印が無い
- `Card()` / `fetch` / UI / 設定を変えること（`s30-other-grace` が触る範囲と重ねない）

## Decisions

### D1. 「worker が反映を終えた」を本文 1 行目・`question`・`UpdatedAt` の 3 つで判定する

規則 2 の分岐に `!reflected(pr)` を足し、`reflected` を次の 2 条件（`question` が無いことは規則 2 の分岐が既に見ている）とする。

- `n, ok := model.ParseUndecided(pr.Body)` が `ok == true` かつ `n == 0`
- `pr.UpdatedAt.After(pr.Comments[len-1].CreatedAt)`

前者だけでは `apply` PR を守れない。`apply` PR の本文 1 行目は最初から `未確定の判断: 0 件` で `question` も付かないので、
人がレビューコメントを書いた瞬間に行 C（merge する）へ出てしまう。s32 が規則 2 / 3 の時間切れを `other` にしたのは
「判定表に流すと、反映されていない人の依頼が C に見える」ためで、前者だけを条件にするとそれが 3 時間待たずに起きる。
後者は「人の最新コメントの後に PR 自身が動いた」ことを表し、worker が本文・ラベル・コミットのどれを動かしても真になる。

代替案として、`Comments` の末尾より後の `<!-- routine -->` コメントを探す案（＝今の条件そのもの）、
`ReviewThreads` の更新を見る案、`gh` から timeline events を取る案があるが、前 2 つは今回の欠陥そのもので、
3 つ目は取得を増やす（D-001 の「1 リポジトリあたりの `gh` 呼び出しを増やさない」に反する）。

### D2. 印がある PR は規則 2 の時間切れ（`other`）にも落とさない

`reflected` は分岐の条件に置く。印がある PR は「AI が次に動く」前提の対象から外れるので、
時間切れの概念ごと外れ、判定表で分類される。結果は緑で mergeable なら行 C、そうでなければ行 F / G / その他になる。
3 時間経った反映済みの PR が `PR #<n> は人のコメントに AI が応答していない` と要約されるのは事実に反するので、この置き方を採る。

### D3. `CreatedAt` のゼロ値を特別扱いしない

`pr.UpdatedAt.After(zero)` は真になるので、`createdAt` を持たない手書き fixture の PR は「印あり」と判定される。
実際の `gh` の出力には `createdAt` が必ず入り（`internal/gh/types.go:58`）、手書き fixture はテストの入力なので、
起こり得ない状態への防御コードは書かない（CLAUDE.md）。既存の 2 fixture（`example` / `board`）に規則 2 / 3 に当たる PR は無く、
期待値表は変わらない。

### D4. `label` 方式と `ai-assess:requested` の扱いは変えない

どちらも規則 2 / 3 の分岐に入る前に外れている（`mode != model.ModeLabel`、`!assess`）。この change は分岐の中だけを触る。

## Risks / Trade-offs

- **人が自分のコメントを編集すると `UpdatedAt` が動き、worker が何もしていない `apply` PR が行 C に出る** →
  編集した本人がその PR を見ている状況なので、誤って merge する可能性は低い。編集時刻を追うには timeline events が要り、
  取得が増える。受け入れる
- **worker がコミットを push しただけで、人の依頼を反映していない場合も行 C に出る** → 反映したかどうかを GitHub の状態から
  区別する手段は無い。PR が動いた後は人が読んで判断する段階であり、issue-driven-sdd でも merge は人の判断（`routine-common`）
- **`docs` PR と外部からの PR は今までどおり 3 時間進行中に留まる** → これらは `未確定の判断:` の 1 行目を持たないので
  印が付かない。worker が反映を終えたと宣言していない以上、コメントの順番で判断するしかない
- **`s30-other-grace` が merge されると、判定表で `other` になった反映済み PR は更新から 30 分は進行中タブに出る** →
  これは「その他になってすぐは routine に任せる」というあちらの意図どおりで、30 分後に今やるタブへ出る。矛盾しない
