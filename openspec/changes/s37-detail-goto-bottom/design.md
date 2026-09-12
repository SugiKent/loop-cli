## Context

詳細画面の本文領域を Bubbles の viewport が描き、`updateDetailKey` がキーを 1 か所で振り分けている
（`internal/ui/detail.go:88-131`）。スクロールの分岐は 1 行ずつの `j` / `k` / `↓` / `↑` と 1 画面ずつの
`PgDn` / `PgUp` の 4 つで、末尾へ飛ぶ分岐だけが無い。動機は proposal.md の「Why」を見る。

制約は 3 つある。第 1 に `g` を PR ↔ issue の相互ジャンプが既に使っている。第 2 に `card-detail` の
Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」がスクロールのキーとフッタのヒントを
文面で固定しており、`help-screen` の Requirement がヘルプのキーの行と行数を固定している。第 3 に
`s33-pr-status-sections` が同じ `internal/ui/detail.go` の描画側（`prBodyLines` / `cardBodyLines` /
`checkLines`）を触る予定で、キーの分岐とは関数が別である。

## Goals / Non-Goals

**Goals:**

- カード詳細画面と PR 詳細画面で、本文領域を一度に末尾まで送れるようにする
- 既存のキーの意味を 1 つも変えない（Q1 で推奨案 A を採る場合）
- ヘルプの一覧から新しいキーを見つけられるようにする

**Non-Goals:**

- キュー画面の表にスクロールを足すこと（`s35-queue-grouping` が別の判断として扱う）
- URL 一覧・ラベル一覧のカーソルを末尾へ飛ばすこと（Q3 の回答で B / C を採る場合だけ範囲に入る）
- 本文領域の中身・並び・区切りを変えること（`s33-pr-status-sections` の範囲）
- 検索（`/`）や折りたたみのような、末尾へ行く以外の読み方を足すこと

## Decisions

### D1. 末尾へ送るキーを `G`（Shift+G）にする

`g` は `updateDetailKey` の `case "enter", "g"` が使っており、1 打鍵目でその場で画面が移る。
issue の字面どおりの `gg` を実現するには `g` を前置キーに変えることになり、「`g` を待っている」状態を
`detailState` に足して、他のキー・画面遷移・ヘルプの開閉のすべてで落とす必要がある。落とし忘れると、
後から押した `g` が意図せず最下部へ飛ぶ。`G` なら分岐を 1 つ足すだけで済み、vim の `G` と意味も揃う。

採らなかった案は 2 つある。`gg` を前置キーで実現する案は上の状態が増える。`Home` / `End` を割り当てる案は、
`j` / `k` を使う人の手をホームポジションから外す。どちらも Q1 として人へ問うており、回答が推奨案と
違えばこの決定を差し替える。

### D2. 分岐は `updateDetailKey` の `switch` に 1 つ足す

`case "G": m.detail.vp.GotoBottom()` を `pgup` の次に置く。viewport は先頭へ飛ぶ手立てを既に
`openDetail` と画面遷移で使っており（`internal/ui/detail.go:82`, `:106`, `:129`）、末尾へ飛ぶ手立ても
同じ形で呼べる。`refreshDetail` を呼ばないので、本文の作り直しは起きない。

`updateDetailKey` は `Model` を値で受け取り、書き換えた `Model` を返す。`j` / `k` / `PgDn` / `PgUp` の
4 つが同じ形で viewport を書き換えているので、パターンを増やさずに揃う。

### D3. 本文が収まっているときも受け取り、何も起きないままにする

本文領域の高さに本文が収まっていれば、末尾へ飛んでもスクロール位置は変わらない。押しても何も起きない
だけなので、「本文が収まっているか」を判定する分岐は足さない（CLAUDE.md「起こり得ない状態への防御コードを
追加しない」）。spec の Scenario がこの振る舞いを固定する。

### D4. 詳細画面の外では何もしないことを queue-screen の台帳に書く

`Model.Update` はキュー画面・確認画面・一覧画面・ヘルプ画面をそれぞれ別の関数へ振り分けており、
どの関数も `G` の分岐を持たない。したがって `G` は詳細画面の外で何も起こさない。ただし
`queue-screen` の Requirement「j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、他のキーは何もしない」が
詳細画面のキー（`g` / `x` / `Esc`）を名指しで列挙し、Scenario「未実装のキーは何も変えない」で検証しているので、
`G` もその台帳に足す。card-detail 側に「他の画面では何もしない」と書いて済ませると、他の capability の
振る舞いを検証の無いまま主張することになる。

### D5. 末尾は本文領域の末尾であり、コメントの末尾では止まらない

PR 詳細の本文領域は checks → 本文 → コメント → review thread の順に並ぶ（`prBodyLines`。
`internal/ui/detail.go:371-391`）。`G` は viewport の末尾へ動かすので、review thread を持つ PR では
最後のコメントを通り越す。コメントの末尾で止める案は、本文領域の行とセクションの対応を viewport の外に
持つことになり、`refreshDetail` が作り直すたびに同期が要る。issue #58 の「一気に最下部」は末尾で満たし、
コメントの末尾へ戻るのは `k` / `PgUp` に任せる。spec の Scenario がこの振る舞いを固定する。

### D6. フッタのヒントは変えず、ヘルプに 1 行足す

`card-detail` の Requirement が「s11 までにあった `j/k スクロール` は…ヒントから外す」と決めており、
`PgUp` / `PgDn` もヘルプにだけ出ている。`G` も同じ扱いにするので、フッタのヒントの表示幅
（PR 詳細 109 列、カード詳細 144 列）は変わらず、幅 80 の端末での切れ方も変わらない。
ヘルプのキーの行は 19 行から 20 行になり、画面全体は 22 行で既定の高さ 24 に収まる。

### D7. テストは `View` 越しに読む

既存の詳細のテストは ANSI を落とした `View` の文字列を読む形で書かれている（`plainText`。
`internal/ui/detail_test.go`）。スクロール位置は `View` に出る行で判定できるので、viewport の内部状態を
直接読むテストは書かない。60 段落の `Body` を使うスクロールのテストが既にあるので、同じ組み立てを使う。
`View` の行で「末尾まで動いた」を判定する以上、Card の中身が結果を決める。カード詳細の Scenario は
`Comments` を nil（`コメント: 取得失敗` の 1 行）に固定し、PR 詳細の Scenario は `Comments` と
`ReviewThreads` の件数を固定する。実装が無くても通ってしまう形（押す前と後の `View` が等しいことだけを
見る Scenario）は書かない。

## Risks / Trade-offs

- **issue が書いた `gg` とは違うキーになる** → この点を Q1 として PR 上で人へ問う。回答が B / C なら D1 と D2 を
  差し替え、`g` を前置キーにする実装へ移る。そのとき `g` の今のジャンプは `Enter` と `Esc` が担うので、
  行き来する道は残る
- **`G` を知らないまま使い続ける人がいる** → ヘルプ（`?`）の一覧に 1 行出す。フッタのヒントには出さないので、
  ヘルプを開かない人には見つからない。スクロール系のキーを既にそう扱っているので、規則は揃う
- **`s33-pr-status-sections` と `internal/ui/detail.go` を共有する** → 触る関数が違う（s33 は
  `prBodyLines` / `cardBodyLines` / `checkLines`、この change は `updateDetailKey`）。先に merge された方に
  もう一方が追随する形で足り、spec の Requirement も重ならない
- **末尾へ飛んだ後に元の位置へ戻る道が無い** → Q2 として人へ問う。推奨案では足さないので、`PgUp` を
  押し続けるか、`Esc` と `Enter` で開き直すことになる
- **review thread を持つ PR では、`G` が最後のコメントを通り越す** → D5 のとおり受け入れ、spec の Scenario で
  見えるようにする。`s33-pr-status-sections` が `── review thread` の見出しを足すと 1 行遠ざかる
- **Q1 で B / C を採ると、s33 と同じ Requirement を MODIFIED することになる** → `g` の意味は
  `card-detail`「PR 詳細は 1 行目判定…」に書かれており、s33 がそれを MODIFIED する。merge 順で
  どちらかが追随する。既存テスト 2 本（`TestGoesBackAndForthBetweenIssueAndPR` /
  `TestGDoesNothingOnPROnlyCard`）と `queue-screen` の未実装キーの一覧も書き換えになる

## Open Questions

proposal.md の「未確定の判断」が 3 件を挙げている（Q1 はキーの割り当て、Q2 は最上部へ戻るキー、Q3 は適用範囲）。
この design.md はどれも推奨案を前提に書いており、回答が違えば D1 / D2 / D4 を差し替える。
