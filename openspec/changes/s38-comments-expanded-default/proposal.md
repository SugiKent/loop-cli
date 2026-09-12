issue: #60

## Why

カード詳細と PR 詳細を開いた直後、routine（AI）のコメントは見出し 1 行に畳まれている
（`internal/ui/detail.go:75` が `detailState` をゼロ値で作り、`expanded` が false のまま
`internal/ui/preview.go:53` が要約 1 行を返す）。全文を読むには毎回 `x` を押す。

畳まれているのは AI のコメントだけで、人のコメントは常に全文が出る（`internal/ui/preview.go:57-64`）。
つまり「人が答えるべき問い」を運ぶコメント — `blocked-by: human` の選択肢、PR リスク評価、grill の質問 —
だけが既定で隠れている。loop-cli は人の出番を 1 本のキューに並べて捌く TUI なので、詳細を開いた理由が
その問いを読むことである場面が多く、そこで 1 打鍵を挟む形になっている。

issue #60 はこれを「デフォルトで全て表示する」に変える。

## What Changes

- カード詳細画面と PR 詳細画面を開いたときの routine コメントの既定を、折りたたみから展開に変える
- `x` は残し、向きを逆にする。押すと畳み、もう一度押すと展開に戻る。キュー画面から `Enter` で開き直すと
  既定の展開に戻る（今は折りたたみに戻る）
- フッタのヒントを `x 展開` から `x 畳む` に変える。表示幅が同じ 6 列なので、フッタ全体の列数
  （カード詳細 144 列 / PR 詳細 109 列 / `PRs` が空のカード詳細 96 列）と切り詰めの規則は変わらない
- ヘルプ画面のキーの行は変えない。`x` の説明は「routine コメントの展開 / 折りたたみ（詳細）」で、
  向きが変わっても正しい（`internal/ui/help.go:29`）
- キュー画面のプレビューは変えない。元から展開で描いている（`internal/ui/preview.go:43` が `true` を渡す）
- review thread 内のコメントは変えない。元から折りたたまない
- 折りたたみの書式（`▌AI  HH:MM  <要約>  (+<n> 行)`）と `summarize` は残す。`x` で畳んだときに使う
- 取得（`internal/gh` / `internal/fetch`）・分類（`internal/classify`）・書き込み（`internal/action`）には触れない

## Capabilities

### New Capabilities

（無し）

### Modified Capabilities

- `card-detail`: Requirement「routine コメントは折りたたみ、x で展開する」を REMOVED にし、
  「routine コメントは既定で展開し、x で折りたたむ」を ADDED で置く。Scenario 名に折りたたみの向きが
  入っているので、MODIFIED では書き直せない（`s26-close-issue-pr` が `queue-screen` のフッタで採ったのと
  同じ形）。あわせて次の 4 Requirement を MODIFIED する
  - 「Enter でカード詳細を開き、Esc で 1 つ前の画面に戻る」: 開いたときの復帰先を「展開」にする 1 行
  - 「本文領域は Issue 本文・最新 blocked-by の要約・コメント時系列を出す」: 書式の参照先の Requirement 名
  - 「PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す」: 同じ参照先の名前 2 か所
  - 「詳細の本文領域はスクロールし、ヘッダ領域は固定する」: フッタのヒントの `x 展開` を `x 畳む` に

## Impact

- `internal/ui/detail.go`: `detailState` のフラグ `expanded` を `collapsed` に変える（ゼロ値が既定の展開に
  なる）。あわせて `case "x"` の反転、`commentBlock` へ渡す値（`:375`）、フッタのヒント 3 本（`:477`, `:482`, `:484`）を直す
- `internal/ui/preview.go`: 変更しない。`commentBlock` の引数 `expanded` の意味は呼び出し側が決める
- `internal/ui/detail_test.go`: `TestAICommentIsCollapsedAndExpandedByX` / `TestReopenCollapsesAgain` /
  `TestRiskHeadingCommentIsCollapsedAsAI` / `TestPRDetailOfPR131` / `TestDetailFooters` /
  `TestFooterWithoutPRs` の 6 本を新しい既定に合わせて書き換える
- `README.md`: 「画面とキー操作」の表にある `x` の行を、カード詳細（`:172`）と PR 詳細（`:192`）の 2 か所で直す
- `openspec/specs/card-detail/spec.md`: 上の 5 Requirement（置き換え 1 本と MODIFIED 4 本）
- 依存の追加は無い。`docs/mvp` は凍結（`CLAUDE.md`）なので書き戻さない

## 確定した判断

- **既定はゼロ値で表す。** `detailState` は 4 か所で構成され、そのうち 3 か所
  （`internal/ui/view_test.go:542`、`internal/ui/detail_test.go:505`, `:1032`）がフィールドを
  指定しないゼロ値である。`expanded: true` を `openDetail`（`internal/ui/detail.go:75`）にだけ書くと、
  この 3 か所が既定と違う状態になり、テストが実物と違う画面を見ることになる。フラグ名を `collapsed` に
  変えれば、ゼロ値がそのまま「展開」になり、書き足す初期化が 1 つも要らない。
- **ヒントは `x 畳む` にする。** 表示幅が `x 展開` と同じ 6 列なので、フッタのヒント全体の列数
  （144 / 109 / 96）も、「カード詳細のヒントは幅 143 以下の端末で末尾から切れる」という記述も変わらない。
  `x 折りたたみ` は 12 列で、3 つの列数と切り詰めの境目をすべて書き換えることになり、幅 150 の端末で
  カード詳細のヒントがちょうど端まで届く（`internal/ui/detail_test.go:998` の Scenario が使う幅）。
  「畳む」はコード内の語（`internal/ui/preview.go:49`「見出し 1 行に畳む」）とも揃う。
- **`x` をフッタに残す。** フッタのヒントから外すのは移動系のキー（`j/k スクロール`）だけで、これは
  `card-detail` の Requirement「詳細の本文領域はスクロールし…」が決めている。`x` が切り替えるのは表示の
  形であり、スクロール位置ではない。したがって既存の規則どおりヒントに残す。
- **ヘルプ画面は触らない。** `help-screen` の Requirement がキーの行を 1 字単位で固定しているが、
  `x` の行は「routine コメントの展開 / 折りたたみ（詳細）」で、どちら向きのトグルでも正しい。
  行を変えなければ `help-screen` の MODIFIED が要らず、s37 で 21 行になった行数の記述にも触れない。
- **キュー画面と review thread は既に展開である。** プレビューは `commentLines` が `expanded` に `true` を
  渡し（`internal/ui/preview.go:43`）、review thread は `card-detail` の Requirement が「折りたたまない」と
  決めている。この change で 3 か所の既定がすべて展開にそろう。
- **キュー画面の `x` は何もしないままである。** `queue-screen` の Requirement「j / k / ↑ / ↓ で行を移動し…
  他のキーは何もしない」が `x` を詳細画面のキーとして名指ししており、キーの持ち主も意味の分類も変わらない
  ので、その台帳は書き換えない。
- **折りたたみの実装は消えない。** `summarize`（`internal/ui/preview.go:69-78`）と `▌AI  HH:MM  <要約>
  (+<n> 行)` の書式は `x` を押したときに使う。Q1 で選択肢 B を採るなら、ここが丸ごと消える。
- **本文領域の行数は増えるが、画面が端末の高さを超えることは無い。** 本文領域は viewport が描き、収まらない分は
  スクロールで見る（`card-detail`「詳細の本文領域はスクロールし、ヘッダ領域は固定する」）。展開した
  コメントが増やすのは viewport の中身の行数だけで、ヘッダ・区切り線・フッタの行数は変わらない。
  ただし **画面に最初から見えている範囲は狭くなる**。`TestPRDetailOfPR131`（`internal/ui/detail_test.go:749`）は
  高さ 40 で `── review thread ` まで見えることを検証しており、AI コメント 1 件が数行に増えた後も
  見えるかを実測して期待値を決める（tasks 1.6）。
- **`x` を押した後の再描画は既にある。** `case "x"` は `refreshDetail()` を呼んでおり
  （`internal/ui/detail.go:103-105`）、向きが変わっても呼び方は同じである。

## 未確定の判断

### Q1. `x` の折りたたみを残すか、折りたたみ自体を廃止するか
issue の本文は「デフォルトで全て表示する」、title は「コメントは常に展開する」で、`x` を残すかどうかが
決まらない。

- 選択肢 A（推奨）: **残す。** 既定を展開にして、`x` は「畳む / 戻す」の逆向きのトグルにする。
  AI コメントが 10 件以上並ぶ PR で見出しだけの一覧に戻す手立てが残り、変更は `card-detail` の
  5 Requirement（置き換え 1 本 + MODIFIED 4 本）とテスト 6 本で収まる。この proposal と tasks は A で書いてある。
- 選択肢 B: **廃止する。** 折りたたみの書式・`summarize`・`detailState` のフラグ・`x` の分岐・フッタの
  ヒントを消し、`x` をキュー画面と同じく「何もしないキー」に戻す。コードは減るが、`help-screen` の
  キーの行（21 行 → 20 行。行数を Requirement が固定している）と `queue-screen` の「他のキーは何もしない」の
  台帳も書き換えることになり、長い AI コメントが並ぶ PR を畳む手立ては無くなる。
- 依存: なし
