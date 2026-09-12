issue: #60

## Why

カード詳細と PR 詳細を開いた直後、routine（AI）のコメントは見出し 1 行に畳まれている
（`internal/ui/detail.go:75` が `detailState` をゼロ値で作り、`expanded` が false のまま
`internal/ui/preview.go:53` が要約 1 行を返す）。全文を読むには毎回 `x` を押す。

畳まれているのは AI のコメントだけで、人のコメントは常に全文が出る（`internal/ui/preview.go:57-64`）。
つまり「人が答えるべき問い」を運ぶコメント — `blocked-by: human` の選択肢、PR リスク評価、grill の質問 —
だけが既定で隠れている。loop-cli は人の出番を 1 本のキューに並べて捌く TUI なので、詳細を開いた理由が
その問いを読むことである場面が多く、そこで 1 打鍵を挟む形になっている。

issue #60 はこれを「デフォルトで全て表示する」に変える。折りたたみを残して既定だけ変える案と、
折りたたみ自体を廃止する案を PR #68 で問い、**廃止する案**（選択肢 B）の回答を得た。

## What Changes

- routine コメントの折りたたみを廃止する。カード詳細画面と PR 詳細画面のコメントは、AI が書いたものも
  人が書いたものも常に全文が出る
- 畳んだときの 1 行表示 `▌AI  HH:MM  <要約>  (+<n> 行)` と、それを作る要約の規則がコードと spec から消える
- `x` はどの画面でも何もしないキーになる。キュー画面での扱い（何もしない）は元からそのままで、
  詳細画面がそれに揃う
- 詳細画面のフッタのヒントから `x 展開` を落とす。表示幅はカード詳細 144 列 → 136 列、PR 詳細 109 列 → 101 列、
  `PRs` が空のカード詳細 96 列 → 88 列になり、ヒントが切れ始める端末幅も 143 → 135 に下がる
- ヘルプ画面のキーの一覧から `x` の行を落とす。キーの行は 21 行から 20 行になり、既定の高さ 24 に 1 行の余りが戻る
- キュー画面の「他のキーは何もしない」の台帳で、`x` を詳細画面のキーの並びから外す
- キュー画面のプレビューと review thread 内のコメントは変えない。どちらも元から全文である
- 取得（`internal/gh` / `internal/fetch`）・分類（`internal/classify`）・書き込み（`internal/action`）には触れない

## Capabilities

### New Capabilities

（無し）

### Modified Capabilities

- `card-detail`: Requirement「routine コメントは折りたたみ、x で展開する」を REMOVED にし、
  「コメントは AI も人も常に全文を出す」を ADDED で置く。あわせて 4 Requirement を MODIFIED する
  - 「Enter でカード詳細を開き、Esc で 1 つ前の画面に戻る」: 開いたときに展開状態を戻す記述を落とし、
    `x` が詳細画面でも何もしないことを台帳に足す
  - 「本文領域は Issue 本文・最新 blocked-by の要約・コメント時系列を出す」: 書式の参照先の Requirement 名
  - 「PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す」: 同じ参照先の名前 2 か所
  - 「詳細の本文領域はスクロールし、ヘッダ領域は固定する」: フッタのヒントから `x 展開` を落とし、
    3 つの表示幅と列位置（`? ヘルプ` は 57 列目、`u URL` は 64 列目）を引き直す
- `help-screen`: 2 Requirement を MODIFIED する
  - 「? はヘルプ画面を開き、? か Esc で開いた画面に戻る」: ヘルプを開閉しても保つ状態の並びから「展開状態」を
    落とし、Scenario「カード詳細から開いて Esc で戻る」が `x` を押さない形になる
  - 「ヘルプ画面は実装済みのキーだけを一覧する」: キーの行から `x` の行を落とし、行数（21 → 20、
    見出しと空行を含めて 23 → 22）を書き換える
- `queue-screen`: Requirement「j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、他のキーは何もしない」の
  台帳で、`x` を「詳細画面のキー」の並びから外す。Scenario「未実装のキーは何も変えない」は `x` を元から
  含んでいるので変えない

## Impact

- `internal/ui/detail.go`: `detailState` のフィールド `expanded` を消す。`case "x"` の分岐（`:103-105`）を
  消し、コメントを描く箇所（`:375`）から引数を 1 つ落とし、フッタのヒント 3 本（`:477`, `:482`, `:484`）から
  `x 展開` を外す
- `internal/ui/preview.go`: `commentBlock` の引数 `expanded` と折りたたみの枝（`:53-55`）を消し、
  `summarize`（`:69-78`）を関数ごと消す。`commentLines`（`:43`）の呼び出しも引数を 1 つ落とす
- `internal/ui/help.go`: `helpKeys` から `x` の行（`:29`）を落とす
- `internal/ui/detail_test.go`: `TestAICommentIsCollapsedAndExpandedByX` / `TestReopenCollapsesAgain` /
  `TestRiskHeadingCommentIsCollapsedAsAI` / `TestPRDetailOfPR131` / `TestDetailFooters` /
  `TestFooterWithoutPRs` の 6 本を書き換える
- `internal/ui/help_test.go`: `TestHelpReturnsToCardDetail`（`:66`）と `TestHelpListsImplementedKeys`（`:186`）の
  2 本を書き換える
- `README.md`: 「画面とキー操作」の表から `x` の行を、カード詳細（`:172`）と PR 詳細（`:192`）の 2 か所で消す
- `openspec/specs/card-detail/spec.md`・`openspec/specs/help-screen/spec.md`・`openspec/specs/queue-screen/spec.md`:
  上の Requirement を書き換える
- 依存の追加は無い。`docs/mvp` は凍結（`CLAUDE.md`）なので書き戻さない

## 確定した判断

- **`x` を空きキーのままにする。** 折りたたみを廃止しても `x` に別の意味を割り当てない。mvp.md の
  キーバインド表は `x` を持っており（`queue-screen` の台帳が `A` / `s` / `g` / `x` / `/` などと並べている）、
  そこに新しい意味を足すのは issue #60 の範囲を超える。`Update` は `x` の分岐を持たなくなるので、
  どの画面でも押しても何も起きない。
- **フッタのヒントから落とす。** 動かないキーをヒントに残すと、押しても何も起きないキーを画面が案内する
  ことになる。落とすと 3 つのヒントの表示幅が 8 列ずつ縮み（`x 展開` の 6 列と区切りの 2 列）、
  カード詳細 136 列、PR 詳細 101 列、`PRs` が空のカード詳細 88 列になる。
- **ヘルプの一覧からも落とす。** `help-screen` の Requirement は「実装済みのキーだけを一覧する」と定めて
  おり、`x` は実装済みのキーではなくなる。キーの行は 20 行になり、s37 で使い切っていた既定の高さ 24 に
  1 行の余りが戻る。
- **要約の規則は関数ごと消す。** `summarize`（`internal/ui/preview.go:69-78`）を呼ぶ箇所は
  `commentBlock` の折りたたみの枝だけで、その枝が消えると誰も呼ばなくなる。使わない関数を残さない
  （`CLAUDE.md`「投機的な機能・将来の拡張に備えたコードは書かない」）。
- **`commentBlock` の引数を 1 つ減らす。** 呼び出し側は 2 か所あり、キュー画面のプレビューは常に `true` を
  渡し（`internal/ui/preview.go:43`）、詳細画面は `detailState.expanded` を渡していた（`internal/ui/detail.go:375`）。
  折りたたみが無くなると両方が同じ振る舞いになるので、引数そのものを落とす。
- **`detailState` からフィールドが 1 つ減る。** `expanded` を持つのは折りたたみのためだけで、他の場所は
  読まない。`detailState` をゼロ値で作るテスト 3 か所（`internal/ui/view_test.go:542`、
  `internal/ui/detail_test.go:505`, `:1032`）は、フィールドを指定していないのでそのまま通る。
- **キュー画面のプレビューと review thread は変わらない。** プレビューは `commentLines` が常に全文を描き、
  review thread は `card-detail` の Requirement が「折りたたまない」と決めている。この change で
  3 か所の表示がすべて同じ書式にそろう。
- **本文領域の行数は増えるが、画面が端末の高さを超えることは無い。** 本文領域は viewport が描き、
  収まらない分はスクロールで見る。展開したコメントが増やすのは viewport の中身の行数だけで、
  ヘッダ・区切り線・フッタの行数は変わらない。ただし画面に最初から見えている範囲は狭くなる。
  `TestPRDetailOfPR131`（`internal/ui/detail_test.go:749`）は高さ 40 で `── review thread ` まで
  見えることを検証しているので、実測して期待値を決める（tasks 1.5）。
- **`x` が何もしないことを検証するテストが要る。** キュー画面には `TestUnimplementedKeysDoNothing`
  （`internal/ui/model_test.go:87-110`）があり `x` を既に含む。詳細画面には同じ検証が無いので、
  新しい Scenario「x を押しても表示は変わらない」で足す。

## 回答で確定した判断（PR #68 のコメント、2026-09-12）

- **折りたたみを廃止する**（Q1 の回答「選択肢 b で」）。既定を展開にして `x` をトグルとして残す案（選択肢 A）は
  採らない。`x` の分岐・`detailState.expanded`・`summarize`・畳んだときの 1 行表示・フッタとヘルプの
  `x` の行が消える。この proposal と tasks と spec delta は、この回答に沿って書いてある。

## 明示的に延期した判断と残るリスク

- **長い AI コメントを畳む手立てが無くなる。** grill の質問が 20 行以上並ぶ PR では、コメントを縦に
  スクロールして読むことになる。`G` / `End` / `Home`（s37）と `PgDn` / `PgUp` で送る手立ては残る。
  読みにくさが実際に出たら、セクション単位で送るキーや検索（`/`、s19 の枠）を別 issue で決める。
- **`x` が空きキーとして残る。** mvp.md のキーバインド表にありながら何もしないキーは、`p`（s21 で切替を
  失った）に続いて 2 つ目になる。この change では何も割り当てない。
- **ヘルプの余白は 1 行だけ戻る。** キーの行が 20 行になり、見出しと空行とフッタを含めて 23 行。既定の
  高さ 24 では 1 行余る。次にキーの行が 2 本増えると、ヘルプは末尾から黙って切り詰め始める
  （`internal/ui/help.go:56-65`）。
- **`▌AI  HH:MM  <要約>  (+<n> 行)` を期待している外部の資料は追わない。** `docs/mvp/mvp.md:97` は
  「routine コメントは折りたたみ、`x` で展開」と書いているが、`docs/mvp` は MVP 時点の記録として凍結する
  規則（`CLAUDE.md`）なので書き戻さない。利用者向けの説明は `README.md` で保つ。
