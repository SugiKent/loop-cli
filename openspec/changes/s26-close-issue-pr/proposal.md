## Why

issue #4。キューを捌いていて「これはもう要らない」と判断したとき、close する手立てが TUI に無く、`o` でブラウザへ出るしか道がない。
close は取り消しに手間がかかる操作なので、`m`（merge）と同じく確認画面を挟む。

## What Changes

- キュー画面（選択行の主体。issue でも PR でもよい）・カード詳細画面（その Card の Issue）・PR 詳細画面（その PR）で `c` を押すと
  close の確認画面を出す。`y` で実行、`Esc` で中止（`m` の確認画面と同じ規則）
- 確認画面は対象（`<repo> #<n>` / `<repo> PR#<n>` とタイトル）・種別（`種別: issue` / `種別: PR`）・`labels:` を出す。本文は出さない
- 対象が `propose` / `apply` ラベルを持つ PR のときは、確認画面に `注意: merge せずに close した PR は却下として扱われ、
  issue に blocked-by: human が書き戻されます` を出す（下記 Impact の 1 項目目）
- 確認画面で `y` を押すと `gh issue close <n> -R <repo>` または `gh pr close <n> -R <repo>` を実行する
- close の結果はフッタ右のステータスに出す。`Cards` は書き換えず、反映は `R` / 自動更新に任せる（`a` / `t` / `m` と同じ）
- 書き込み中（コメント投稿 / ラベル切り替え / merge / close）は `c` を受け付けない。close の書き込み中も `t` / `a` / `m` / `c` を受け付けない
- `?` ヘルプのキー一覧に `c` の行を足し、キュー・カード詳細・PR 詳細のフッタのヒントに `c close` を `m merge` の次
  （`PRs` が空のカード詳細では `t todo` の次）で足す
- `docs/mvp/mvp.md` のキーバインド表に `c` の行を足し、変更履歴に 1 行足す（`c` は表に無いキーなので、`u` を足した s22 と同じ扱い）

close にコメントを添える機能は入れない（コメントは `a` がある）。段階ラベルは触らない（human-turn-signals.md 不変条件 2
「TUI が書くラベルは `stage:todo` と `s` の `stage:propose` の 2 つに限る」を守る）。

## Capabilities

### New Capabilities
- `close-action`: `internal/action` の close。`action.Target` の `IsPR` による `gh.GHClient.CloseIssue` / `ClosePR` の書き分けと、
  却下として扱われる PR の注意の判定。ラベル・コメントは書かない
- `close-issue-pr`: `internal/ui` の `c` を定める。`Model` が画面ごとに対象を決め、close の確認画面（`y` / `Esc`）を出し、
  結果をステータスに出す

### Modified Capabilities
- `gh-client`: `GHClient` に `CloseIssue` / `ClosePR` を足す（s03 の「後続 change が ADDED で Requirement を足して
  interface・`Client`・`Fake` を同時に拡張する」に従う ADDED）
- `gh-fake`: `Fake` が記録する書き込みメソッドに `CloseIssue` / `ClosePR` が加わる
- `queue-screen`: `c` が「何もしないキー」から「主体の close の確認画面を開く」に変わる。フッタのヒントが 80 列から
  89 列になり、右のステータスの出所に close の中止・実行が加わる。ヒントの幅が変わって Scenario
  「幅 90 では取得中でもヒントとスピナーが両方出る」が成り立たなくなるので、フッタの Requirement は REMOVED + ADDED で
  作り直す（Scenario 名に幅が入っており MODIFIED では書き直せない）
- `card-detail`: 画面の状態が 7 つから 8 つ（close の確認画面を追加）になり、詳細画面の `c` の対象が決まる。
  カード詳細・PR 詳細のフッタのヒントに `c close` が入る（126 列 / `PRs` 空で 78 列 / PR 詳細 91 列）
- `help-screen`: 実装済みキーの一覧に `c` が入る
- `todo-toggle`: 「書き込み中は `t` と `a` を受け付けない」の対象に `c` が加わる（Requirement 名は変えない）

## Impact

- **close は上流のループを閉じない。人の出番を 1 つ作る。** merge せずに close された `propose` / `apply` PR は、上流の
  dispatcher / sweep が「人が却下した」とみなして issue に `blocked-by: human`（`unblock-when: comment`）を書き戻し、
  ラベルを `[stage:X, blocked, question]` にする（human-turn-signals.md の局面 F の説明、`routine-sweep` 手順 1、
  `routine-dispatch` 手順 D）。その issue は次の取得で局面 B（方針を決める・優先度 2）としてキューに戻る。
  issue 自体を close した場合は、`routine-dispatch` 手順 C が `depends on #n` でその issue を待っていた issue を放出するので、
  別の issue の AI 作業が動き出すことがある。merge 済みの `propose` / `apply` PR がある issue を close すると、
  `routine-sweep` 手順 6 が「孤児 proposal」として人に reopen か取り下げかを問う（sweep は reopen しない）
- コードは 3 か所を触る。`internal/gh` に `CloseIssue` / `ClosePR` を足し、`internal/action` に close のファイルを足し、
  `internal/ui` に `c` のキー処理と確認画面とステータスを足す。`cmd/loop-cli` の配線は変わらない（設定項目を増やさない）
- `gh` は `y` の後に `gh issue close` または `gh pr close` を 1 回呼ぶ。読み取りを呼ぶかどうかは下記 Q1 が未確定
- close された issue / PR は次の取得（`R` / 自動更新）で `gh search --state open` の結果から消え、キューの行としては落ちる
- 書き込み後の対象 1 件再取得（D-002）は s18 の担当で、この change では行わない

## 確定した判断

- **対象の決め方は issue #4 の指定どおり**。キュー画面は選択行の主体（`internal/ui/rows.go` の `Subject`。issue でも PR でもよい）、
  カード詳細は `Card.Issue`、PR 詳細はその PR。カード詳細では `m` が ▶ で選択中の PR を対象にするのに対し `c` は `Issue` を
  対象にするので対象がずれるが、確認画面が `種別:` と番号を出すので、閉じる前に読める（design.md にトレードオフを書く）
- **`c` は現状どこにも割り当てが無い**。`internal/ui` に `case "c"` / `key == "c"` は 1 つも無く（`internal/ui/model.go`、
  `internal/ui/detail.go:84-107`、`internal/ui/urls.go:106-116`）、mvp.md キーバインド表（`docs/mvp/mvp.md:100-119`）にも無い。
  既存テストで `c` を押すものも無く、壊れるのはヘルプの行数を見ているテスト 1 件（16 行 → 17 行）だけ
- **確認画面のキー振り分けは `a` / `t` / `m` / `o` / `?` / `u` より先に置く**。`internal/ui/model.go` の merge 確認画面と同じ理由で、
  後ろだと確認中に裏の対象を触れてしまう
- **確認画面に本文は出さない**。issue #4 が求めるのは「対象と、issue か PR かが分かる表示」。代わりに `labels:` を出して、
  `wip`（AI が作業中）や段階ラベルが人の目に入るようにする（判断は人に委ねる。`m` の確認画面が警告を並べるのと同じ思想）
- **`gh issue close` に `--reason` は渡さない**（`gh` の既定の `completed` になる）。上流の dispatcher / sweep は close の理由を
  読まず、`blocked-by: human` を書くかどうかは「merge されずに close されたか」だけで決まる。理由を選ばせても上流の挙動は
  変わらないので、フラグも設定項目も増やさない。`gh pr close` に理由の指定は無い
- **`c` を押した時点でフッタのステータスを空にする**（`internal/ui/answer.go` の `a` と同じ）。そうしないと直前の赤字が
  確認画面と、`Esc` で戻った先に残る
- **`internal/ui/rows.go` の `row` は変えない**。`row` は close の対象に必要な値をすべて持っている（リポジトリ・番号・
  issue か PR か・タイトル・ラベル）。`row` にフィールドを足すと s08 `queue-screen` の spec に波及する
- **フッタのヒントに `c close` を足す**（キュー 89 列 / カード詳細 126 列・`PRs` 空で 78 列 / PR 詳細 91 列）。
  人の回答（PR #10 のコメント）で決まった。位置は `m merge` の次で、`PRs` が空のカード詳細では `t todo` の次に置く。
  幅 90〜97 の端末では取得中や書き込み中にヒントが丸ごと消えるようになる（s14 がフッタを 80 列に伸ばしたときの
  「キーを隠すよりキーを出すことを採った」と同じ引き換え）
- **mvp.md のキーバインド表に `c` の行を足す**。`u` を足した s22（`openspec/changes/archive/2026-09-06-s22-url-picker/tasks.md` 5.5）と
  `↑ update` を足した s23 の前例に従う。`help-screen` の行順は「前半は mvp.md キーバインド表の順」なので、表に足さないと
  ヘルプの行を置く位置が決まらない。mvp.md「前提と未決事項」の「取り下げる・止める（段階ラベルを外す）」は `c` では
  解消しない。その行が指すのは段階ラベルを外す操作であり、不変条件 2 がそれを TUI に置くことを禁じているので、この行は触らない

## 未確定の判断

### Q1. `c` を押した時点で、対象のラベルと状態を GitHub から取り直すか

issue #4 は「すでに closed なものは対象にしない（確認画面でその旨を出して実行させない）」と書いているが、画面に出ている
issue と PR に closed のものは無い。`internal/gh/client.go:87-103` の search は `--state open` で、`internal/fetch/fetch.go` は
search の結果からしか Card を作らず、`model.PRFromSearch` は `State` に必ず `OPEN` を入れる（`internal/model/model.go:222-235`）。
`CrossReferencedPRs` による紐づけ補完は未実装（mvp.md §3 の P2）なので、closed / merged の PR がカード詳細の一覧に並ぶ経路も
今は無い。つまり `State` を見る拒否を書いても、本番では発火しない防御コードになる（CLAUDE.md「起こり得ない状態への防御コードを
追加しない」）。同じことは `action.CheckMerge` の `State` の拒否にも起きている（`internal/ui/merge.go:145-153` は
`merge-pr` spec が要求する「画面の Card の `State` を写す」を行っておらず、この拒否は発火しない。s26 では直さず申し送る）。

古くなるのは `State` より**ラベル**である。確認画面の `labels:` は最終取得時点の値で、自動更新の既定は 120 秒、
スナップショットから起動した直後（`internal/snapshot`）は前回セッションの値なので、直前に付いた `wip`（AI が作業中）は
確認画面に出ない。初回取得が終わる前でも `c` は動く（取得中かどうかを見るキーは `R` だけである。`internal/ui/model.go`）。

- 選択肢 A（推奨）: **取り直さない。拒否も持たない。** `c` は即座に確認画面を出し、`labels:` は取得時点の値を出す。`y` は常に出す。
  取得後に人が GitHub 側で close していた場合、`gh issue close` はすでに closed なものへの close なので状態を変えない。
  `t` / `a` が画面の値で書き込むのと同じ扱いで、実装は最小。**リスク**: `wip` が付いた直後の issue を、人が気づかずに close しうる。
  また `gh issue close` / `gh pr close` がすでに closed の対象に対して終了コード 0 を返す場合、TUI は
  `… を close しました` と表示する（この環境に `gh` が無いため未実測。merged PR の close は失敗するはず）
- 選択肢 B: **`m` と同じく押下時に取り直す。** issue は `gh issue view` の `--json` に `state` を足し、PR は `ViewPR` の結果を使う
  （`gh pr view --json state` が使えるかは要確認。`merge-pr` spec:40 は「`gh pr view` の `--json` に `state` が無い」と書いており、
  実機で確かめる）。取り直した `labels` と `state` で確認画面を作り、open でなければ `close できません: <State> です` を出して
  `y` を出さない。`wip` も最新が出る。代わりに `gh-client` spec と fixture（`internal/gh/testdata`）と型が変わり、
  `c` に 1 往復ぶんの待ちと失敗経路（取り直し中の画面移動）が増える
- 依存: なし
