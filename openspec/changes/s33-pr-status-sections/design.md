## Context

背景と原因は proposal.md の「Why」と「確定した判断」にある。ここでは、以下の設計判断の前提になる現状だけを挙げる。

- 詳細画面の本文領域は Bubbles の viewport が描く。内容は `refreshDetail`（`internal/ui/detail.go:136-145`）が
  `prBodyLines` / `cardBodyLines` の返す行を渡して作り直す。`View` では作らない
- 行の幅の基準は左ペインの幅 `m.leftWidth()`（`internal/ui/session.go:50-55`）。1 ペインのときは端末の幅そのもの、
  2 ペインのときは `端末幅 − 41`
- ヘッダと本文の間の区切り線は `renderDetail`（`internal/ui/view.go:87`）が `strings.Repeat("─", leftW)` で引く。
  merge の確認画面も同じ記法で判断材料と本文を分ける（`internal/ui/merge.go:237`）
- 色は lipgloss で付ける。キューの行が種別ごとの `lipgloss.Style` を行全体に `Render` する形になっている
  （`internal/ui/view.go:29-35`, `internal/ui/view.go:248-252`）
- 詳細のテストは ANSI を落とした文字列を読む（`plainText`。`internal/ui/detail_test.go`）

## Goals / Non-Goals

**Goals:**

- 本文領域を目で追えるセクションに分け、どこまでが status でどこからが PR 本文かを一目で分からせる
- 赤い check を、一覧の中から文字を読まずに見つけられるようにする
- カード詳細と PR 詳細で同じ読み方ができるようにする

**Non-Goals:**

- キュー画面下段のプレビューの書式を変えること（proposal.md「確定した判断」に理由がある。Q4 で問う）
- ヘッダ領域（タイトル行・labels 行・PR 一覧）の書式を変えること
- 高さの配分・スクロールの規則・キー操作を変えること
- `mergeable` 行への着色。緑・赤の 2 色を check の行だけに使い、画面の中で色の意味を 1 つに保つ

## Decisions

### D1. セクションの見出しは `── <名前> ` に `─` を継ぎ足した 1 行にする

`── 本文 ` の後を左ペインの幅まで `─` で埋める。区切りであることは線が示し、名前は「今どこを読んでいるか」を示す。

無地の `─` だけ（proposal.md Q1 の選択肢 B）は、ヘッダと本文の間の区切り線と見た目が同じになる。本文領域は
スクロールするので、無地の線が流れてきたときに、読み手はヘッダの線なのかセクションの線なのか区別できない。
字下げを深くするだけ（同 選択肢 C）は、コメントと review thread の境目を解かない。

見出しの名前が幅に収まらない狭い端末では、`ansi.Truncate` で左ペインの幅に切る（`…` は付けない。線の一部に見えるため）。
幅が 0 以下（サイズメッセージが届く前）なら空文字列の 1 行にする。`strings.Repeat` に負の数を渡すと panic するので、
継ぎ足す本数は `max(幅 − 見出しの表示幅, 0)` で求める。

**見出し行の幅は `View` 越しには検証できない。** 本文領域は viewport が描き、viewport は各行を幅まで空白で埋める。
`── 本文 ` を `─` で継ぎ足さない実装でも、`View` から取り出した行の表示幅は本文領域の幅になる。テストは
末尾の空白を落としてから幅と末尾の文字を見るか、見出し行を作る関数の戻り値を直接見る。spec の Scenario もこの形で書く。

### D2. 見出しは、中身があり、かつ上に行があるセクションにだけ置く

条件は 2 つある。

1. **そのセクションの中身が 1 行以上ある。** `renderMarkdown` は空の `Body` に対して 0 行を返す
   （`internal/ui/preview.go:113-117`）。本文が空の PR で `── 本文 ` を出すと、直後の `── コメント ` と 2 行続き、
   D1 が避けたかった「どちらがどの境目か分からない」状態がそのまま出る。`internal/ui/detail_test.go` の
   `prOf` が作る PR の `Body` は空文字列なので、この形は既存のテストの入力でも起きる
2. **そのセクションより上に本文領域の行が 1 行以上ある。** 本文領域の 1 行目の直上には、ヘッダとの区切り線が
   既にある。そこへ見出しを重ねると線が 2 行続く

PR 詳細の 1 つ目のセクション（1 行目判定から checks まで）は必ず 2 行以上出るので、条件 2 は PR 詳細では働かない。
カード詳細で `Issue.Body` が空のときにだけ働く。

セクションの並びは spec が定めた順のままにする。PR 詳細は status / 本文 / コメント / review thread の順、
カード詳細は Issue 本文 / blocked-by / コメント の順である。並べ替えると、読み慣れた場所が動いてしまう。

### D3. checks の一覧に `checks: 緑` / `checks: 緑以外` の見出しを付ける

`mergeable:` 行の直下に字下げの一覧が続く今の形は、その一覧が checks であることをどこにも書いていない。
判定は既存の `classify.ChecksGreen` をそのまま使い、merge の確認画面（`internal/ui/merge.go:239`）と同じ
`緑` / `緑以外` の語にする。0 件の `checks: なし` と取得失敗の `checks: 取得失敗` は今までどおりの文言で、
見出しと同じ列（字下げなし）に置く。今は `checks: なし` だけが 2 文字字下げになっており（`internal/ui/detail.go:406`）、
見出しを入れるとその字下げに理由が無くなる。

内訳の件数（proposal.md Q2 の選択肢 C）は採らない。どの conclusion を成功・実行中・失敗に数えるかという分類を
新たに決めることになり、`classify.ChecksGreen` と二重の基準が生まれる。

### D4. check の行は行全体に色を付ける

キューの行が種別ごとの色を行全体に付けているのと同じ形にする（`internal/ui/view.go:248-252`）。状態の語だけを
染めると、赤の面積が数文字になって一覧の中で見つけにくい。

色の対応は次のとおり。値は mvp.md がキューの行に決めた色（`docs/mvp/mvp.md:79-80`）を流用する。

| 区分 | `CheckRun` の状態 | `StatusContext` の `State` | 色 |
| --- | --- | --- | --- |
| 成功 | `SUCCESS` / `SKIPPED` / `NEUTRAL` | `SUCCESS` | 緑 `#0E8A16` |
| 失敗 | `FAILURE` / `TIMED_OUT` / `CANCELLED` / `ACTION_REQUIRED` / `STARTUP_FAILURE` | `FAILURE` / `ERROR` | 赤 `#B60205` |
| それ以外 | 上のどちらでもない（`QUEUED` / `IN_PROGRESS` / 空 など） | 上のどちらでもない（`PENDING` / `EXPECTED` など） | 色を付けない |

緑にする集合を `Typename` ごとに分けるのは、`classify.ChecksGreen` がそう分けているからである
（`CheckRun` は `SUCCESS` / `SKIPPED` / `NEUTRAL`、`StatusContext` は `SUCCESS` だけ。
`internal/classify/classify.go:187-196`）。`StatusContext` の `SKIPPED` を緑に塗ると、その 1 件しか無い PR で
行が緑なのに D3 の見出しが `checks: 緑以外` になり、読み手はどちらを信じるか分からなくなる。
失敗の集合は GitHub の check conclusion と commit status state の語のうち、赤で出す価値があるものを挙げた。
どちらにも無い語は、増えたときに黙って赤や緑にならないよう、色を付けない側に落とす。

見出しの `checks: 緑` / `checks: 緑以外` には色を付けない。`緑以外` は失敗と実行中の両方で出るので、
赤にすると実行中を失敗と読ませてしまう。

### D5. 表示だけを変え、取得と分類には触れない

変えるのは `internal/ui/detail.go` の `prBodyLines` / `cardBodyLines` / `checkLines` と、区切り線を作る小さな
ヘルパーだけにする。`internal/gh`・`internal/fetch`・`internal/classify`・`internal/action` は変えない。
`classify.ChecksGreen` は呼ぶだけで、判定は変えない。

## Risks / Trade-offs

- **[見出しのぶん、一度に見える本文が減る]** → 高さの計算はヘッダの行数だけを見る（`detailHeader`。
  `internal/ui/detail.go:150-172`）ので、画面の行数が端末の高さを超えることはなく、押し出された行は viewport の
  スクロールで読める。ただし縦に狭い端末では実害が出る。高さ 12 の PR 詳細は本文領域が 8 行で、
  1 行目判定 / 紐づく issue / mergeable / checks の見出し / check 3 行 / `── 本文 ` で 8 行を使い切り、
  スクロールしなければ PR 本文が 1 行も見えない（今は 1 行見える）。キュー画面下段のプレビューを対象外にしたのと
  同じ性質の劣化が、狭い高さの詳細画面では残る。詳細はスクロールを持つのでプレビューと違って読む手立てはあるが、
  この劣化を避けたいなら proposal.md Q1 の選択肢 C（行を消費しない）を採ることになる。
- **[色の付いた行が 2 ペインの結合で崩れる]** → `joinPanes` は `pad`（`ansi.Truncate` + 空白埋め）で左ペインを
  揃える（`internal/ui/view.go:101-111`, `internal/ui/view.go:287-293`）。`ansi.Truncate` はエスケープを保つ実装で、
  既にキューの行の色付きテキストがこの経路を通っている。tasks に幅の検証を入れる。
- **[色を出さない端末で情報が落ちる]** → 状態の語（`SUCCESS` / `FAILURE` など）は今までどおり文字で出るので、
  色は補助にとどまる。色だけが伝える情報は作らない。
- **[GitHub が新しい conclusion を足すと色が付かない]** → D4 のとおり、未知の語は色を付けない側に落ちる。
  誤って緑や赤に見せるより安全な壊れ方を選ぶ。

## 未決事項

proposal.md の「未確定の判断」Q1〜Q4 が未回答である。この design.md は各問いの推奨案（Q1: A、Q2: A、Q3: A、Q4: A）を
採った場合の設計を書いている。回答が推奨案と違えば、D1〜D4 の該当する決定と spec の delta を書き直す。
