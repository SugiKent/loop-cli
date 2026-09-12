issue: #58

## Why

詳細画面の本文領域を送るキーは、1 行ずつの `j` / `k` / `↓` / `↑` と 1 画面ずつの `PgDn` / `PgUp` の
4 つだけで、末尾へ一気に飛ぶ手立てを持たない（`internal/ui/detail.go:88-99`）。issue #58 が挙げた
「PR のコメント一覧を一気に最下部へ」は、本文 → コメント → review thread → checks の順に並ぶ PR 詳細の
本文領域（`internal/ui/detail.go:371-391`）を `PgDn` で何度も送るしかない。コメントが数十件付いた PR では、
キーを押す回数が中身を読む時間より長くなる。

本文領域を描く Bubbles の viewport は末尾へ飛ぶ手立てを持ち、先頭へ飛ぶ側をこの TUI は既に使っている
（`openDetail` と画面遷移でスクロール位置を先頭に戻す。`internal/ui/detail.go:82`, `:106`, `:129`）。
足りないのはキーの割り当てだけである。

## What Changes

- カード詳細画面と PR 詳細画面に、本文領域を一度に末尾まで送る `G`（Shift+G）と `End`、先頭まで戻す `Home` を足す
- ヘルプ画面のキー一覧に 2 行足し、キーの行を 19 行から 21 行にする
- キュー画面の「他のキーは何もしない」の台帳に、新しい 3 つのキーを足す。キュー画面は詳細画面のキーを
  名指しで列挙して「何もしない」ことを Scenario で検証しており、その一覧から漏らさない
- 詳細画面のフッタのヒントは変えない。スクロール系のキーをヒントに出さずヘルプへ委ねる既存の規則に従う
- キュー画面・URL 一覧・ラベル一覧・確認画面は変えない。これらはスクロールを持たないので、新しいキーは何もしない
- 取得（`internal/gh` / `internal/fetch`）・分類（`internal/classify`）・書き込み（`internal/action`）には触れない

## Capabilities

### New Capabilities

（無し）

### Modified Capabilities

- `card-detail`: Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」に、本文領域を一度に
  最下部まで送るキーを足す。同じ Requirement が定める他の規則（ヘッダを固定すること、高さの計算、
  フッタのヒント、スクロール位置が先頭に戻る条件）は変えない
- `help-screen`: Requirement「ヘルプ画面は実装済みのキーだけを一覧する」のキーの行を 2 行足し、
  行数の記述（キーの行 19 → 21、画面全体 21 → 23）を書き換える
- `queue-screen`: Requirement「j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、他のキーは何もしない」に
  `G` / `Home` / `End` を足し、Scenario「未実装のキーは何も変えない」の一覧にも足す

## Impact

- `internal/ui/detail.go`: `updateDetailKey` の `switch` に分岐を 2 つ足す（`G` / `End` と `Home`）
- `internal/ui/help.go`: `helpKeys` に 2 行足す
- `internal/ui/detail_test.go` と `internal/ui/help_test.go`: 新しい Scenario の期待値を足す
- `internal/ui/model_test.go`: `TestUnimplementedKeysDoNothing` の一覧に `G` / `Home` / `End` を足す
- `README.md`: 「画面とキー操作」のカード詳細 / PR 詳細の表に、新しいキーの行を 2 本ずつ足す
- `openspec/specs/card-detail/spec.md`・`openspec/specs/help-screen/spec.md`・`openspec/specs/queue-screen/spec.md`:
  上の 3 Requirement を書き換える
- 依存の追加は無い。`internal/model` / `internal/classify` / `internal/action` / `internal/gh` は変更しない

## 確定した判断

- **`g` は詳細画面で既に塞がっている。** `updateDetailKey` の `case "enter", "g"` が PR ↔ issue の相互
  ジャンプを担い、1 打鍵目でその場で画面が移る（`internal/ui/detail.go:114-128`）。この意味を書いている
  のは 4 か所で、mvp.md のキーバインド表（`docs/mvp/mvp.md:122`）、ヘルプの一覧（`internal/ui/help.go:23`）、
  詳細のフッタ（`internal/ui/detail.go:443`, `:450`）、`card-detail` spec がそれに当たる。したがって
  「`g` を 2 回」は、1 打鍵目で何もせず次の打鍵を待つ前置キーに `g` を変えない限り実現しない。
- **前置キーにすると画面の状態が 1 つ増える。** `detailState`（`internal/ui/detail.go:43` 付近）に
  「`g` を待っている」フラグを持たせ、他のキー・画面遷移・ヘルプの開閉のすべてで落とす必要がある。
  落とし忘れると、後から押した `g` が意図せず最下部へ飛ぶ。CLAUDE.md の「シンプルさを最優先にする」に
  照らし、キー 1 つで済むなら前置キーを採らない。
- **`g` の今のジャンプは `Enter` / `Esc` と重複している。** カード詳細の `g` は `Enter` と同じ分岐に
  入っており、完全に同じ動きをする（`internal/ui/detail.go:114`）。PR 詳細の `g` は `Issue` が non-nil の
  ときだけカード詳細へ戻るが、`Issue` が non-nil の Card は必ずカード詳細を経由して PR 詳細へ来る
  （`openDetail` は `Issue` が nil のときだけ PR 詳細を直接開く。`internal/ui/detail.go:66-83`）ため、
  そのとき `fromDetail` が真であり `Esc` も同じ戻り先を持つ（同 `:102-108`）。**つまり `g` を空けても、
  行き来する道は無くならない。** ただし戻り先が同じでも返すコマンドは違い、`g` は右ペインのセッションの取得を
  始める（`internal/ui/detail.go:130` の `sessionOnOpenCmd`）のに対し `Esc` は何も返さない。Q1 の回答は
  `g` を残す案だったので、この取得の起点は減らない。
- **`G`（Shift+G）はどのキーとも当たらない。** mvp.md のキーバインド表（`docs/mvp/mvp.md:107-126`）に
  `G` は無い。表にあって未実装の予約キーを `help-screen` spec が `v` / `h` / `l` / `←` / `→` / `A` /
  `s` / `/` と列挙しており、`G` はどれでもない。大文字はキーの文字列にそのまま載る（`L` が
  `internal/ui/model.go:313`、`R` が同 `:347` と `:379` で既にその形）。
- **スクロールのキーはフッタのヒントに出さず、ヘルプに出す。** `card-detail` の Requirement
  「詳細の本文領域はスクロールし、ヘッダ領域は固定する」が「s11 までにあった `j/k スクロール` は…
  ヒントから外す」と決めており、`PgUp` / `PgDn` もヘルプにだけ出ている（`internal/ui/help.go:30`）。
  この change の新しいキーも同じ扱いにするので、フッタのヒントの表示幅（PR 詳細 109 列、
  カード詳細 144 列）は変わらない。
- **ヘルプは行数を spec が固定しているので MODIFIED になる。** `help-screen` の Requirement が
  「キーの行は 19 行」「行数は 21 で、`Model` の既定の高さ 24 に収まる」と書き、Scenario も 19 行を
  検証している。2 行足すと 21 行と 23 行になり、フッタと合わせて既定の高さ 24 をちょうど使い切る。
  **次にキーの行が 1 本増えた時点で、ヘルプは末尾から黙って切り詰め始める**（`internal/ui/help.go:56-65`）。
- **viewport で送る画面は詳細の本文領域だけである。** URL 一覧（`internal/ui/urls.go:115-127`）と
  ラベル一覧（`internal/ui/labels.go:136-158`）はカーソルで動く。この 2 画面は表示の窓が
  `start := max(cursor-h+1, 0)` で選択行に追従する（`internal/ui/urls.go:157`, `internal/ui/labels.go:253`）。
  キュー画面の表だけは窓の追従を持たない（`s35-queue-grouping` が「表にスクロールを足さない」と明記）ので、
  キュー画面で選択行を末尾へ飛ばすと、選択行だけが画面の外へ出る。Q3 の回答はこの 3 画面を対象から外した。
- **`Home` / `End` はどのキーとも当たらない。** mvp.md のキーバインド表に `Home` と `End` は無く、
  キュー画面・確認画面・一覧画面・ヘルプ画面のどの分岐も `home` / `end` を持たない。キーの文字列は
  `home` / `end` で届く（`PgDn` / `PgUp` が `pgdown` / `pgup` で届くのと同じ経路）。
- **`G` が着く先は本文領域の末尾であり、コメントの末尾ではない。** PR 詳細の本文領域は checks → 本文 →
  コメント → review thread の順に並ぶ（`prBodyLines`。`internal/ui/detail.go:371-391`）。review thread を持つ
  PR で `G` を押すと、最後のコメントを通り越して thread の末尾に着き、issue #58 が挙げた「コメントの末尾」は
  画面の外に出ることがある（`k` / `PgUp` で戻れる）。spec の Scenario がこの振る舞いを固定する。
  セクション単位で送るキーは、この change では扱わない。
- **進行中の 2 change とは触る Requirement が重ならない。** `s33-pr-status-sections` は `card-detail` の
  「本文領域は Issue 本文…」と「PR 詳細は 1 行目判定…」を MODIFIED し、`s35-queue-grouping` は
  `queue-screen` を触る。この change が触るのは `card-detail` の「詳細の本文領域はスクロールし…」と
  `help-screen` と `queue-screen` の 3 本で、どれとも別である（`queue-screen` の「他のキーは何もしない」を
  s35 は触らない）。`internal/ui/detail.go` を s33 と共有するが、s33 が触る関数は `prBodyLines` /
  `cardBodyLines` / `checkLines` で、この change が触るのは `updateDetailKey` である。
  `g` を空ける案を採っていたらこの判断は崩れていた。`g` の意味は s33 が MODIFIED する Requirement
  「PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す」の中にあり
  （`openspec/specs/card-detail/spec.md:220`, `:224` の Scenario 2 本）、同じ Requirement を s37 も
  MODIFIED することになっていた。Q1 の回答が `G` だったので、その二重 MODIFIED は起きない。
- **スクロール位置が先頭に戻る条件は変えない。** キュー画面から開いたとき、およびカード詳細と PR 詳細を
  行き来したときに先頭へ戻る規則（`card-detail` の同じ Requirement）をそのまま残す。

## 回答で確定した判断（PR #63 のコメント、2026-09-12）

- **末尾へ送るキーは `G`（Shift+G）の 1 打鍵にする**（Q1 の回答「G で」）。`g` は PR ↔ issue の相互ジャンプの
  ままにする。`g` を前置キーに変える案（`gg`）は採らない。これにより `s33-pr-status-sections` が MODIFIED する
  Requirement「PR 詳細は 1 行目判定…」に手を入れずに済み、既存テストの `TestGoesBackAndForthBetweenIssueAndPR` と
  `TestGDoesNothingOnPROnlyCard` もそのまま通る。
- **先頭へ戻るキーも足す**（Q2 の回答「足す」）。`Home` を先頭へ、`End` を末尾へ割り当て、`G` と併存させる。
  `gg` が空かないので `Home` / `End` の組を使う。末尾へ送る手立てが `G` と `End` の 2 つになるので、
  ヘルプでは 1 行にまとめて `G / End` と出し、`Home` を次の行に置く。
- **適用する画面はカード詳細と PR 詳細の本文領域だけにする**（Q3 の回答「A」）。キュー画面・URL 一覧・
  ラベル一覧では、この 3 つのキーは何もしない。キュー画面の表は選択行に追従する窓を持たないので、
  選択行だけが画面の外へ出る形になるのを避ける。

## 明示的に延期した判断と残るリスク

- **`G` は本文領域の末尾へ着くので、review thread を持つ PR では最後のコメントを通り越す。** issue #58 が
  挙げた「コメントの末尾」で止めるにはセクションと行の対応を viewport の外に持つことになり、
  `refreshDetail` が本文を作り直すたびに同期が要る。`k` / `PgUp` で数行戻れるので、この change では
  末尾へ着く形のままにする。セクション単位で送るキーが要るかは、`G` を使ってみてから別 issue で決める。
- **ヘルプ画面の余白が無くなる。** キーの行が 21 行になり、見出し・空行・フッタと合わせて既定の高さ 24 を
  ちょうど使い切る。次にキーの行が 1 本増えた時点で、ヘルプは末尾から黙って切り詰め始める
  （`internal/ui/help.go:56-65`）。未実装の予約キー（`v` / `h` / `l` / `←` / `→` / `A` / `s` / `/`）が入ると
  必ずこの境目に当たるので、そのときヘルプの畳み方かスクロールを決めることになる。
- **`End` は `G` と同じ動きの 2 つ目のキーになる。** vim の手（`G`）と、そうでない手（`End`）の両方に道を
  用意する形で、ヘルプでは 1 行にまとめる。キーの総数を抑えるなら `End` を落とせるが、Q2 の回答が
  `Home` / `End` の組を選んだので対称のまま残す。
- **`Home` で戻るのは本文の先頭であり、`G` を押す前の位置ではない。** 元の位置へ戻す仕組みは、
  スクロール位置の履歴を持つことになるので足さない。
- **東アジア幅やキーボードプロトコルの差は見ていない。** `G` は文字として、`Home` / `End` はキー名として
  届く経路で、`L` / `R` と `PgDn` / `PgUp` が既に同じ経路で動いている。
