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
- `help-screen`: 実装済みキーの一覧に `c` が入る。この delta は先行 change `s26-issue-label-driven`（`origin/main` の
  `openspec/changes/` に未 archive で存在し、同じ Requirement を MODIFIED している）の版を土台に写して `c` の行を足した。
  `t` の行の「stage:todo / To Do を付ける / 外す」はその change の変更で、打ち消さない。archive はその change を先に行う
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
- `gh` は `y` の後に `gh issue close` または `gh pr close` を 1 回呼ぶだけで、読み取りは呼ばない
- close された issue / PR は次の取得（`R` / 自動更新）で `gh search --state open` の結果から消え、キューの行としては落ちる
- 書き込み後の対象 1 件再取得（D-002）は s18 の担当で、この change では行わない
- 順序依存と衝突: `origin/main` の `openspec/changes/` にある未 archive の change が、この change と同じ Requirement を
  MODIFIED している。`openspec archive` は MODIFIED をその時点のメイン spec に上書きするので、**apply / archive の直前に
  下の Requirement を、そのときのメイン spec と未 archive の先行 change の delta から写し直す**（tasks 1.0）。
  写し直さずに archive すると、先に archive された change の変更が消える。下の一覧は 2026-09-10 09:15 時点の断面で、
  この repo は propose PR が次々 merge されるので、tasks 1.0 はそのときに存在する change を見て写し直す（一覧の更新は要らない）。
  - `help-screen`「ヘルプ画面は実装済みのキーだけを一覧する」: `s26-issue-label-driven`（`t` の行を
    `stage:todo / To Do を付ける / 外す` に変更）と `s15-new-issue`（`n` の行を `m` の次に追加）。この change の delta は
    `s26-issue-label-driven` の版を土台にしているので、`s15-new-issue` の `n` の行を足す作業が残る（3 本すべてが入ると
    キーの行は 18 行になる）
  - `queue-screen`「j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、他のキーは何もしない」: `s15-new-issue`（`n` を追加）
  - `queue-screen`「ヘッダはタブ名と件数と最終更新時刻、フッタはキーヒントとステータスを出す」: `s15-new-issue`
    （ステータスの出所に `new-issue` を追加し、`n` をヒントに出さない理由を書き足す）。この change はこの Requirement を
    REMOVED + ADDED で作り直すので、`s15-new-issue` が先に archive されるとその追記が消える
  - `card-detail`「Enter でカード詳細を開き、Esc で 1 つ前の画面に戻る」: `s15-new-issue`（作成の確認画面を追加）
  - `card-detail`「詳細の本文領域はスクロールし、ヘッダ領域は固定する」: `s15-new-issue`（詳細のフッタに `n` を追加）と
    `2026-09-10-s27-wrap-titles`（タイトル行の折り返しと切り詰めを追加）
  - `todo-toggle`「書き込み中は t と a を受け付けない」: `s15-new-issue`（`n` を追加）
  - `gh-fake`「Fake は書き込み呼び出しと ViewIssue を記録する」: `s28-label-picker`（`ListLabels` とラベルの一括編集の
    記録を追加）。この change は同じ Requirement に `CloseIssue` / `ClosePR` を足す

  `s28-label-picker` は上のうち `help-screen` 1 本・`queue-screen` 2 本・`card-detail` 2 本・`gh-fake` 1 本を触る
  （`L` のラベル一覧画面を足す change）。`2026-09-10-s27-wrap-titles` は `card-detail` 1 本、`s15-new-issue` は 5 本、
  `s26-issue-label-driven` は `help-screen` 1 本である。
- フッタの budget の食い違い: `s15-new-issue` は「キュー画面のヒントは 80 列で埋まっているので `n 新規` は足さず、
  入口は `?` のヘルプと詳細画面のフッタが見せる」と決めている。この change は人の判断（PR #10 のコメント）で
  `c close` を足して 89 列にするので、両方が入ると「`c` はヒントにあって `n` は無い」状態になる。判断は人のもので、
  この change はそれに従う

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
- **`c` の押下では `gh` を呼ばない（対象のラベルと状態を取り直さない）**。人の回答（PR #10 のコメント）で決まった。
  `c` は待ち時間なしで確認画面を出し、`labels:` は最終取得時点の値を出す。`y` は常に出す。
  「すでに closed なものは対象にしない」（issue #4）に対応する拒否は入れない。画面に出る issue と PR は取得時点で
  必ず open で（`SearchIssues` / `SearchPRs` が `--state open`、`internal/fetch` は検索結果からしか Card を作らない）、
  closed / merged の PR がカード詳細に並ぶ経路は cross-reference 紐づけ（mvp.md §3 の P2）が入るまで無いので、
  拒否を書いても発火しない防御コードになる（CLAUDE.md「起こり得ない状態への防御コードを追加しない」）
- **mvp.md のキーバインド表に `c` の行を足す**。`u` を足した s22（`openspec/changes/archive/2026-09-06-s22-url-picker/tasks.md` 5.5）と
  `↑ update` を足した s23 の前例に従う。`help-screen` の行順は「前半は mvp.md キーバインド表の順」なので、表に足さないと
  ヘルプの行を置く位置が決まらない。mvp.md「前提と未決事項」の「取り下げる・止める（段階ラベルを外す）」は `c` では
  解消しない。その行が指すのは段階ラベルを外す操作であり、不変条件 2 がそれを TUI に置くことを禁じているので、この行は触らない

## 明示的に延期した判断と残るリスク

- **close の入口に「open でないものを止める」ガードを入れない。** 上記のとおり今は発火しないため。P2 の
  cross-reference 紐づけが入って closed / merged の PR がカード詳細の PR 一覧に並ぶようになったら、その回に
  ガードを足す（`action.CheckMerge` の `State` の拒否を実際に働かせる作業と同じ回にまとめるのが安い）
- **確認画面の `labels:` は古いことがある。** 自動更新の既定は 120 秒、スナップショットから起動した直後は前回
  セッションの値なので、直前に付いた `wip`（AI が作業中）は確認画面に出ない。`t` / `a` が画面の値で書き込むのと
  同じ性質で、この change では受け入れる
- **すでに closed の対象を close したときの表示は未実測。** `gh issue close` / `gh pr close` がその場合に終了コード 0 を
  返すなら、TUI は `… を close しました` と出す（この環境に `gh` が無く確認できなかった）。状態は変わらないので
  影響は表示だけで、実装時に手元の `gh` で確かめて spec の文言をそろえる
- **`internal/ui/merge.go` が `model.PR.State` を写していない既存の spec 違反は直さない。** s14 の取りこぼしで、
  `action.CheckMerge` の `State` の拒否が本番で発火しない。今は closed / merged の PR が画面に出ないので実害は無く、
  P2 の cross-reference 紐づけの回にまとめて直すのが安い
- **カード詳細では `m` が選択中の PR、`c` が `Issue` を対象にする。** ▶ を PR に合わせている人が `c` を押すと issue が
  閉じるので、確認画面の `種別:` と番号とタイトルで気づける形にしてある（issue #4 の指定どおり）
