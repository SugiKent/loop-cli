issue: #35

## Why

PR 詳細の本文領域は、status（1 行目判定・紐づく issue・mergeable・checks）から Glamour でレンダリングした
PR 本文へ、区切りなく地続きで流れる。checks の各行は 2 文字字下げで並び（`internal/ui/detail.go:409`,
`internal/ui/detail.go:411`）、Glamour の出力も 2 文字字下げなので、最後の check 行と PR 本文の 1 行目が
同じ列に並び、同じ一覧の続きに見える。issue #35 が挙げた実物がこれで、`image: SUCCESS` の次の
`未確定の判断: 0 件 — レビューをお願いします` は PR 本文の 1 行目なのに、6 件目の check に見える。

同じ画面の merge の確認画面は、判断材料と PR 本文の間に幅いっぱいの区切り線を 1 行入れて同じ問題を
既に解いている（`internal/ui/merge.go:237`）。PR 詳細とカード詳細の本文領域だけが、この扱いから漏れている。

## What Changes

- PR 詳細の本文領域を、status / 本文 / コメント / review thread のセクションに分け、境目が目で追える形にする
- カード詳細の本文領域（Issue 本文 / blocked-by / コメント）も同じ形に分ける
- checks の一覧に、それが checks の一覧であると分かる見出しを与え、その `緑` / `緑以外` を
  s33 `colorful-labels` が定めた状態語の色で出す
- 取得（s20）・分類（`internal/classify`）・書き込み（`internal/action`）には触れない。変わるのは
  `internal/ui` の表示だけで、キー操作も増やさない

区切りの記法・checks の見出し・色の有無・適用範囲は PR #40 で問い、4 件とも推奨案で決まった（「確定した判断」の
最後の 4 項目）。check の行の色は、その後 issue #35 で #34 の規則に寄せる決定が出て取り下げた（同じ節の末尾）。

## Capabilities

### New Capabilities

（無し）

### Modified Capabilities

- `card-detail`: PR 詳細の本文領域（Requirement「PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す」）と
  カード詳細の本文領域（Requirement「本文領域は Issue 本文・最新 blocked-by の要約・コメント時系列を出す」）の
  並べ方を変える。セクションの区切りを入れ、checks の見出しを定める
- `card-detail`: 色を塗る位置の一覧（Requirement「カード詳細と PR 詳細はラベル名と状態語に色を付ける」）に、
  新しく出る `checks: 緑` / `checks: 緑以外` の値を足す

## Impact

- `internal/ui/detail.go`: `prBodyLines` / `cardBodyLines` / `checkLines`
- `internal/ui/detail_test.go`: 上の 3 つを読む既存テスト
- `README.md`: 「PR 詳細」の節
- `openspec/specs/card-detail/spec.md`: 上の 3 Requirement

## 確定した判断

- **原因は字下げの一致である。** `prBodyLines`（`internal/ui/detail.go:374-391`）は status ブロックと
  `renderMarkdown` の出力を区切りなく連結する。`checkLines`（同 `394-415`）は各 check を 2 文字字下げで出し、
  Glamour も本文を 2 文字字下げで出すので、両者が同じ列に並ぶ。
- **同じ問題の既存の解き方がリポジトリにある。** `renderMergeConfirm`（`internal/ui/merge.go:237`）は
  判断材料と PR 本文の間に `strings.Repeat("─", m.width)` の 1 行を入れている。詳細画面のヘッダと本文の間にも
  同じ無地の区切り線がある（`internal/ui/view.go:87`）。
- **色を足しても既存テストは壊れない。** 詳細のテストは ANSI を落とした文字列で読む（`plainText`。
  `internal/ui/detail_test.go`）。lipgloss は既に `internal/ui/view.go:29-40` で使っており、依存は増えない。
- **本文に行を足しても画面の行数は端末の高さを超えない。** 本文領域は viewport が描き（`refreshDetail`。
  `internal/ui/detail.go:136-145`）、高さの計算（`detailHeader`。同 `150-172`）はヘッダの行数だけを見る。
  区切り線の幅は左ペインの幅（`m.leftWidth()`）で、ヘッダ側の区切り線と揃う。
- **キュー画面下段のプレビューは今回の対象にしない。** プレビューは端末の高さの約半分を `cut` で切るだけで
  スクロールを持たない（`internal/ui/preview.go:17-36`, `internal/ui/view.go:70-74`）。区切り線 1 行が本文 1 行を
  押し出すので、詳細と同じ判断はできない。範囲を問うた Q4 でもプレビューを含めない案が選ばれた。
- **spec は `card-detail` の MODIFIED 2 本になる。** PR 詳細と本文領域の並べ方は
  `openspec/specs/card-detail/spec.md` の既存 Requirement が定めており、新しい capability は要らない。
- **mvp.md は詳細画面の区切りと色を決めていない。** 色を固定しているのはキューの行だけ（`docs/mvp/mvp.md:79-80`）で、
  PR 詳細については「1 行目の判定結果、`Refs #n` / `Closes #n`、会話コメント、review thread、checks 状態」を
  出すことしか書いていない（同 `98`）。よって次の 4 点は design.md の未決事項として PR #40 で人に問い、
  4 件とも推奨案が選ばれた（2026-09-11 の PR コメント「質問の回答はすべて推奨で」）。

- **Q1 の答えは「見出し付きの区切り線」である。** 各セクションの頭に `── 本文 ─────…` のような行を左ペイン幅で置く。
  無地の線だけ（選択肢 B）はヘッダの区切り線と見分けが付かず、字下げを深くするだけ（選択肢 C）はコメントと
  review thread の境目を解かない。design.md D1 / D2 がこの決定を書く。
- **Q2 の答えは「`checks: 緑` / `checks: 緑以外` の見出しを付ける」である。** 判定は `classify.ChecksGreen` を
  そのまま使い、merge の確認画面と同じ語にする。内訳の件数（選択肢 C）は採らない。design.md D3 がこの決定を書く。
- **Q3（check の行に色を付けるか）の答えは取り下げた。** PR #40 では「成功の行を緑 `#0E8A16`、失敗の行を赤 `#B60205`」と
  決めたが、この change を実装する前に #34（change `s33-colorful-labels`）が main に入り、「チェック名は塗らず、
  それに続く状態語だけを良し悪しの 4 色で塗る」という規則を同じ `checkLines` に定めた。両立しないので issue #35 で
  どちらを採るかを問い、2026-09-12 に「A（#34 の規則を採る）」と回答を得た。よって check の行の色はこの change の
  対象外にし、ADDED Requirement「checks の行は成功を緑・失敗を赤で出す」と design.md の D4 を落とす。
  この change が色について定めるのは、新しく出す `checks: 緑` / `checks: 緑以外` の値だけである。
- **`checks: 緑` / `checks: 緑以外` の値は `m.stateWord` で塗る。** 上の決定に伴う扱い。`緑` / `緑以外` は
  PR 一覧行（`internal/ui/detail.go`）と merge の確認画面（`internal/ui/merge.go`）が既に状態語として塗っており、
  同じ語を詳細の本文でだけ塗らないと、画面ごとに規則が分かれる。見出しの `checks:` とチェック名は #34 のとおり塗らない。
- **Q4 の答えは「PR 詳細とカード詳細の両方」である。** キュー画面下段のプレビューは対象外のままにする。

## 明示的に延期した判断と残るリスク

- **縦に狭い端末では、見出し 1 行のぶん一度に見える本文が減る。** 高さ 12 の PR 詳細は本文領域が 8 行で、
  1 行目判定 / 紐づく issue / mergeable / checks の見出し / check 3 行 / `── 本文 ` が 8 行を使い切り、
  スクロールするまで PR 本文が 1 行も見えない（今は 1 行見える）。Q1 で選択肢 A が選ばれた以上これは受け入れる。
  詳細画面はスクロールを持つので読む手立ては残る。狭い高さ向けに見出しを畳む仕掛けは、必要になってから別 issue で扱う。
- **見出しの語と直下の行が重なる場合がある**（`── コメント ` の下に `コメント: なし`、`── review thread ` の下に
  `review thread: なし`）。取得失敗の PR では罫線 3 本に対して中身 3 行になる。見出しと「なし」を 1 行に統合する案は
  採らず、セクションの区切りという 1 つの役割だけを見出しに持たせる（`internal/ui` の他の画面と規則を分けないため）。
- **東アジア幅の曖昧文字を全角で描く端末では、`─` を使う行が折り返る可能性がある。** `ansi.StringWidth` は `─` を
  幅 1 と数えるのでテストでは検出できない。ヘッダと本文の区切り線（`internal/ui/view.go:87`）と merge の確認画面が
  既に同じ文字を使っているので、この change が新しく持ち込むリスクではない。
- **`checks: 緑` は `checks: 緑以外` の部分文字列である。** 「`checks: 緑` を含む」だけを見るテストは常に通るので、
  この語を検証する Scenario は否定のアサーションと対で書く（spec の Scenario はその形にしてある）。
