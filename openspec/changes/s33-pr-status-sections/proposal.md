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
- checks の一覧に、それが checks の一覧であると分かる見出しを与える
- checks の状態（SUCCESS / FAILURE / IN_PROGRESS など）を、赤が一目で見つかるように出す
- 取得（s20）・分類（`internal/classify`）・書き込み（`internal/action`）には触れない。変わるのは
  `internal/ui` の表示だけで、キー操作も増やさない

区切りの記法・checks の見出し・色の有無・適用範囲は「未確定の判断」に置く。

## Capabilities

### New Capabilities

（無し）

### Modified Capabilities

- `card-detail`: PR 詳細の本文領域（Requirement「PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す」）と
  カード詳細の本文領域（Requirement「本文領域は Issue 本文・最新 blocked-by の要約・コメント時系列を出す」）の
  並べ方を変える。セクションの区切りを入れ、checks の見出しと状態の色を定める

## Impact

- `internal/ui/detail.go`: `prBodyLines` / `cardBodyLines` / `checkLines`
- `internal/ui/view.go`: 色の定義を置く場所
- `internal/ui/detail_test.go`: 上の 3 つを読む既存テスト
- `internal/ui/testdata/`: 状態の混ざった checks を持つ PR fixture を 1 つ足す。
  `internal/gh/testdata/fixtures/example` は 10 以上のテストが共有しているので触らない
- `README.md`: 「PR 詳細」の節
- `openspec/specs/card-detail/spec.md`: 上の 2 Requirement

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
  押し出すので、詳細と同じ判断はできない。ただし範囲は Q4 で問う。
- **spec は `card-detail` の MODIFIED 2 本になる。** PR 詳細と本文領域の並べ方は
  `openspec/specs/card-detail/spec.md` の既存 Requirement が定めており、新しい capability は要らない。
- **mvp.md は詳細画面の区切りと色を決めていない。** 色を固定しているのはキューの行だけ（`docs/mvp/mvp.md:79-80`）で、
  PR 詳細については「1 行目の判定結果、`Refs #n` / `Closes #n`、会話コメント、review thread、checks 状態」を
  出すことしか書いていない（同 `98`）。よって以下は design.md の未決事項であり、人に問う。

## 未確定の判断

### Q1. 本文領域のセクションをどう区切るか

PR 詳細で `mergeable` と checks の後に PR 本文が始まる、その境目の出し方。

- **選択肢 A（推奨）: 見出し付きの区切り線**。各セクションの先頭に `── 本文 ─────…` `── コメント ─────…`
  `── review thread ─────…` を左ペイン幅いっぱいで入れる。ヘッダと本文の間にある無地の `─` と見分けが付き、
  スクロールで流れてきても今どのセクションを読んでいるかが分かる。1 セクションにつき 1 行使うので、
  縦に狭い端末では見える本文が減る（高さ 12 の PR 詳細ではスクロールするまで PR 本文が見えなくなる。
  design.md の Risks に計算がある）
- 選択肢 B: 無地の `─` の区切り線だけを入れる。merge の確認画面と同じ記法になるが、ヘッダの区切り線と
  見た目が同じなので、読み手はスクロールした後にどちらの線を見ているのか判断できない
- 選択肢 C: 区切り線を入れず、checks の字下げを深くする（2 文字 → 4 文字など）。行を消費しないが、
  Glamour の字下げが変わればまた同じ列に並ぶ。コメントと review thread の境目は解けない
- 依存: なし

### Q2. checks の一覧に見出し行を付けるか

今は `mergeable: MERGEABLE UNSTABLE` の直下にいきなり各 check が字下げで並ぶ（0 件のときだけ `checks: なし`）。

- **選択肢 A（推奨）: `checks: 緑` / `checks: 緑以外` の 1 行を出し、その下に各 check を字下げで並べる**。
  判定には既存の `classify.ChecksGreen` を使う。merge の確認画面が既に `checks: 緑` を出しているので表記が揃う
- 選択肢 B: 見出しを出さず今までどおりにする。行は増えないが、字下げの一覧に持ち主が無いままになる
- 選択肢 C: 見出しに内訳の件数も出す（`checks: 緑以外（成功 2 / 実行中 3 / 失敗 0）`）。
  一覧を読まずに状況が分かるが、状態の分類（どの conclusion を成功・実行中・失敗に数えるか）を新たに決める必要がある
- 依存: なし

### Q3. checks の状態に色を付けるか

- **選択肢 A（推奨）: 付ける**。失敗（`FAILURE` / `TIMED_OUT` / `CANCELLED` / `ACTION_REQUIRED` / `STARTUP_FAILURE`、
  StatusContext の `FAILURE` / `ERROR`）は赤 `#B60205`、成功（`SUCCESS` / `SKIPPED` / `NEUTRAL`）は緑 `#0E8A16`、
  それ以外（実行中・不明）は既定色のままにする。色は mvp.md がキューの行に使っている値をそのまま流用する
  （`docs/mvp/mvp.md:79-80`）。赤い check が一覧のどこにあっても目に飛び込む
- 選択肢 B: 付けない。mvp.md が色を決めているのはキューの行だけなので、詳細画面は無彩色のままにする。
  この場合、失敗した check は文字を読んで探すことになる
- 依存: なし

### Q4. 同じ区切りをどこまで入れるか

- **選択肢 A（推奨）: PR 詳細とカード詳細の本文領域の両方を区切る**。カード詳細の本文も Issue 本文 / blocked-by /
  コメントが地続きで同じ読みづらさがあり、`g` で 2 つの画面を行き来するので、片方だけ書式が違うと戸惑う。
  キュー画面下段のプレビューは対象外（上の「確定した判断」の理由）
- 選択肢 B: PR 詳細だけ。issue #35 が挙げた範囲ちょうどで、変更は最小になる。カード詳細は今までどおり
- 選択肢 C: プレビューも含めた 3 箇所すべて。プレビューは高さが端末の半分でスクロールを持たないので、
  区切り線に使った行のぶん本文が見えなくなる
- 依存: なし
