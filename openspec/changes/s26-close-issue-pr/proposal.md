## Why

issue #4。上流の「人が持つ操作」にある「取り下げる・止める」に対応するキーが TUI に無い（mvp.md「前提と未決事項」の 3 項目目
「必要になったら追加する」）。人の出番を捌いていて「これはもう要らない」と判断したとき、`o` でブラウザへ出るしか道がなく、
キーボードだけでループを閉じられない。close は取り消しに手間がかかる操作なので、`m`（merge）と同じく確認画面を挟む。

## What Changes

- キュー画面（選択行の主体。issue でも PR でもよい）・カード詳細画面（その Card の Issue）・PR 詳細画面（その PR）で `c` を押すと
  close の確認画面を出す。`y` で実行、`Esc` で中止（`m` の確認画面と同じ規則）
- 確認画面は対象（`<repo> #<n>` / `<repo> PR#<n>` とタイトル）・種別（`種別: issue` / `種別: PR`）・`labels:` を出す。本文は出さない
- 拒否は「PR の `State` が `OPEN` でない」の 1 つ。拒否があるときは `y` を出さない（`action.CheckMerge` と同じ形）
- 確認画面で `y` を押すと `gh issue close <n> -R <repo>` または `gh pr close <n> -R <repo>` を実行する
- close の結果はフッタ右のステータスに出す。`Cards` は書き換えず、反映は `R` / 自動更新に任せる（`a` / `t` / `m` と同じ）
- 書き込み中（コメント投稿 / ラベル切り替え / merge / close）は `c` を受け付けない。close の手続き中も `t` / `a` / `m` / `c` を受け付けない
- `?` ヘルプのキー一覧に `c` の行を足す（3 画面のフッタのヒントに足すかどうかは下記 Q3 が未確定）
- `docs/mvp/mvp.md` のキーバインド表に `c` の行を足し、変更履歴に 1 行足す（`c` は表に無いキーなので、`u` を足した s22 と同じ扱い）

close にコメントを添える機能は入れない（コメントは `a` がある）。段階ラベルは触らない（close は段階ラベルの操作ではない。
human-turn-signals.md 不変条件 2「TUI が書くラベルは `stage:todo` と `s` の `stage:propose` の 2 つに限る」を守る）。

## Capabilities

### New Capabilities
- `close-action`: `internal/action` の close。拒否の判定（`State` が空でも `OPEN` でもない PR は拒否）と、
  `action.Target` の `IsPR` による `gh.GHClient.CloseIssue` / `ClosePR` の書き分け。ラベル・コメントは書かない
- `close-issue-pr`: `internal/ui` の `c` を定める。`Model` が画面ごとに対象を決め、close の確認画面（`y` / `Esc`）を出し、
  結果をステータスに出す

### Modified Capabilities
- `gh-client`: `GHClient` に `CloseIssue` / `ClosePR` を足す（s03 の「後続 change が ADDED で Requirement を足して
  interface・`Client`・`Fake` を同時に拡張する」に従う ADDED）
- `gh-fake`: `Fake` が記録する書き込みメソッドに `CloseIssue` / `ClosePR` が加わる
- `queue-screen`: `c` が「何もしないキー」から「主体の close の確認画面を開く」に変わる
- `card-detail`: 画面の状態が 7 つから 8 つ（close の確認画面を追加）になり、詳細画面の `c` の対象が決まる
- `help-screen`: 実装済みキーの一覧に `c` が入る
- `todo-toggle`: 「書き込み中は `t` と `a` を受け付けない」の対象に `c` が加わる（Requirement 名は変えない）

Q3 が選択肢 B に決まったときは、`queue-screen` と `card-detail` のフッタの Requirement も Modified に加わる。

## Impact

- コードは 3 か所を触る。`internal/gh` に `CloseIssue` / `ClosePR` を足し、`internal/action` に close のファイルを足し、
  `internal/ui` に `c` のキー処理と確認画面とステータスを足す。`cmd/loop-cli` の配線は変わらない（設定項目を増やさない）
- `gh` は `y` の後に `gh issue close` または `gh pr close` を 1 回呼ぶ。読み取りは呼ばない（下記 Q1 が未確定）
- close された issue / PR は次の取得（`R` / 自動更新）で `gh search --state open` の結果から消え、キューから落ちる
- 書き込み後の対象 1 件再取得（D-002）は s18 の担当で、この change では行わない

## 確定した判断

- **対象の決め方は `a` と同じ「画面が見せているものを対象にする」**。キュー画面は選択行の主体（`internal/ui/rows.go` の
  `Subject`。issue でも PR でもよい）、カード詳細は `Card.Issue`、PR 詳細はその PR。issue #4 の指定どおり
- **`c` は現状どこにも割り当てが無い**。`internal/ui` に `case "c"` / `key == "c"` は 1 つも無く（`internal/ui/model.go`、
  `internal/ui/detail.go:84-107`、`internal/ui/urls.go:106-116`）、mvp.md キーバインド表（`docs/mvp/mvp.md:100-119`）にも無い
- **確認画面のキー振り分けは `a` / `t` / `m` / `o` / `?` / `u` より先に置く**。`internal/ui/model.go` の merge 確認画面と同じ理由で、
  後ろだと確認中に裏の対象を触れてしまう
- **確認画面に本文は出さない**。issue #4 が求めるのは「対象と、issue か PR かが分かる表示」。代わりに `labels:` を出して、
  `wip`（AI が作業中）や段階ラベルが人の目に入るようにする（判断は人に委ねる。`m` の確認画面が警告を並べるのと同じ思想）
- **拒否は PR の `State` だけ**。`model.Issue` は `State` を持たず（`internal/model/model.go:126-136`）、`model.PR` は持つ
  （同:154）。`action.CheckMerge` の拒否条件と同じ形にそろえる
- **mvp.md のキーバインド表に `c` の行を足す**。`u` を足した s22（`openspec/changes/archive/2026-09-06-s22-url-picker/tasks.md` 5.5）と
  `↑ update` を足した s23 の前例に従う。`help-screen` の行順は「前半は mvp.md キーバインド表の順」なので、表に足さないと
  ヘルプの行を置く位置が決まらない

## 未確定の判断

### Q1. 「すでに closed なもの」の判定を、GitHub から状態を取り直して行うか

issue #4 は「すでに closed なものは対象にしない（確認画面でその旨を出して実行させない）」と書いているが、今の TUI に
closed のものは出てこない。`internal/gh/client.go:87-103` の search は `--state open` で、`internal/fetch/fetch.go` は
search の結果からしか Card を作らないので、画面の issue も PR も取得時点では必ず open である（`CrossReferencedPRs` による
紐づけ補完は未実装＝ mvp.md §3 の P2）。

- 選択肢 A（推奨）: **取り直さない。** `c` は即座に確認画面を出し、拒否は `model.PR.State` が `OPEN` でないときだけ
  （`action.CheckMerge` と同じ形。issue には状態が無いので拒否条件を持たない）。取得後に人が GitHub 側で close していた場合は
  `gh issue close` が失敗し、フッタに赤で `<表示名> の close に失敗: …` が出る。実装は最小で、`c` にレイテンシが乗らない
- 選択肢 B: **`m` と同じく `c` の押下時に取り直す。** `gh issue view` / `gh pr view` の `--json` に `state` を足して
  `IssueDetail` / `PRDetail` に `State` を持たせ、取り直してから確認画面を出す。`OPEN` でなければ `close できません: <State> です` を
  出して `y` を出さない。issue #4 の記述に忠実で、cross-reference 紐づけ（P2）が入って closed PR がカード詳細の一覧に並ぶように
  なったとき、拒否がそのまま働く。代わりに `gh-client` spec と fixture（`internal/gh/testdata`）と型が変わり、
  `c` に 1 往復ぶんの待ちが乗る
- 依存: なし

### Q2. `gh issue close` に `--reason` を渡すか

`gh issue close` は close の理由を `--reason completed` / `--reason "not planned"` で選べ、GitHub の画面では
「Closed as completed」と「Closed as not planned」が別の印で表示される。`gh pr close` に理由は無い。issue #4 は理由に触れていない。

- 選択肢 A（推奨）: **渡さない。** `gh` の既定（`completed`）になる。フラグも設定項目も増えない
- 選択肢 B: **issue には `--reason "not planned"` を固定で渡す。** mvp.md「前提と未決事項」がこの操作を「取り下げる・止める」と
  位置づけているので、GitHub 側にも取り下げとして記録する。完了として記録する道は残らない。確認画面に `理由: not planned` の行を出す
- 依存: なし

### Q3. 3 画面のフッタのヒントに `c close` を足すか

issue #4 はヘルプ（`internal/ui/help.go`）だけを挙げている。フッタは画面ごとに動くキーを並べる場所で、書き込みのキー
`a` / `t` / `m` はすべて出ている。ただしキュー画面のヒントは既に表示幅 80 列で、`Model` の既定幅 80 の端末では取得中や
書き込み中にヒント全体が消える（フッタは両方入らないときステータスを優先する）。`c close` を足すとキュー 89 列 /
カード詳細 126 列 / PR 詳細 91 列になる。

- 選択肢 A（推奨）: **フッタは変えず、ヘルプにだけ出す。** 幅 90〜97 の端末で「取得中でもヒントとスピナーが両方出る」
  という今の振る舞いを保てる（`queue-screen` の Scenario「幅 90 では取得中でもヒントとスピナーが両方出る」がそれを固定している）。
  spec の変更は `help-screen` の 1 本で済む
- 選択肢 B: **3 画面とも `m merge` の次に `c close` を足す。** s14 がフッタを 80 列に伸ばすときの「キーを隠すよりキーを出すことを
  採った」（queue-screen spec 132 行目、card-detail spec 194 行目）に忠実で、`c` の入口がキューの画面上に見える。代わりに
  幅 90〜97 の端末では取得中にヒントが丸ごと消えるようになり、上記の Scenario が成り立たなくなる。openspec 1.13 の MODIFIED は
  既存 Scenario を落とせないので、`queue-screen` のフッタの Requirement を REMOVED + ADDED で作り直す作業が加わる
- 依存: なし
