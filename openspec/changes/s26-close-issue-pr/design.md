## Context

動機は proposal.md の Why にある（issue #4）。既にある 4 つの書き込みのキーが土台になる。

- `a`（s10 `answer-question`）: `action.Target{Repo, Number, IsPR}` で書き先を分け、`m.writing` で二重投稿を防ぎ、結果を
  フッタ右のステータスに出す。`internal/ui/answer.go`
- `t`（s11 `todo-toggle`）: 確認なしで即書き込み。`internal/ui/todo.go`
- `m`（s14 `merge-pr`）: 押下時に `gh` を呼んで判断材料を取り直し、確認画面（`y` / `Esc`）を出し、`y` で書き込む。
  確認画面の状態（対象・取り直した値・拒否・警告・戻り先）を `Model` の 1 フィールドに持つ。`internal/ui/merge.go`
- `o`（s12 `browse-open`）: 書き込み中でも受け付ける

`c` は `m` の形（確認画面つき）に最も近い。issue #4 も「`internal/ui/merge.go` の一式をなぞって `internal/ui/close.go` として
作るのが素直」と書いている。異なるのは、対象が issue でも PR でもよい点（`a` と同じ規則）と、押下時に `gh` を呼ばない点である。

## Goals / Non-Goals

**Goals:**

- `c` → 確認画面 → `y` で `gh issue close` / `gh pr close` までを 1 本の経路で通し、対象は画面が見せているものに決める
- 拒否の判定を `internal/action` に置き、`internal/ui` は判定を持たない（`action.CheckMerge` と同じ層の分け方）
- 確認画面のキー振り分けを、裏の画面のキーより先に置く

**Non-Goals:**

- close した対象を 1 件だけ再取得すること（D-002 を s18 が担当する）
- close にコメントを添える・ブランチを消す・段階ラベルを外す
- close された issue / PR を画面に残す工夫（次の取得で `--state open` の検索から消えるのに任せる）
- reopen（`gh issue reopen`）。取り消しはブラウザで行う

## Decisions

### `c` の押下では `gh` を呼ばない

`m` は「merge 可否は表示時に取り直す」（human-turn-signals.md）に従って `ViewPR` と `ViewPRMergeState` を呼ぶが、close の
判断材料は「open かどうか」だけで、画面に出ている issue と PR は取得時点では必ず open である（`SearchIssues` / `SearchPRs` が
`--state open`。`internal/gh/client.go`）。取り直しを入れると `c` にレイテンシと失敗経路（取り直し中の画面移動、部分失敗）が
増える。取らなかった案は「`m` と同じく取り直す」で、これは proposal.md の Q1 として人に問う。Q1 が B に決まれば、
`gh issue view` / `gh pr view` の `--json` に `state` を足し、`c` の押下を `m` と同じ 2 段（取り直し → 確認画面）にする。

### 拒否は `internal/action` で判定し、警告は持たない

`action.CheckMerge` は拒否と警告の 2 列を返すが、close の判断材料は 1 つ（PR が open か）しかないので、close の判定関数は
拒否の 1 列だけを返す。`question` / 未確定の判断の件数 / checks / draft は close を止める理由にならない。代わりに確認画面へ
`labels:` の行を出し、`wip`（AI が作業中）や段階ラベルが人の目に入る形にする。判断は人に委ねる（`m` の警告と同じ思想）。

### 確認画面の状態は押下時の値を凍結する

確認画面を読んでいる間に自動更新（s13）が走ると `Cards` が入れ替わり選択行が動くので、`y` の時点で対象を導き直すと
別の issue / PR を close しうる。`m` と同じく、押下時にリポジトリ・番号・種別・表示名・タイトル・ラベル・PR の `State`・戻り先を
1 つの構造体に写して持つ。

### キー振り分けの位置

`internal/ui/model.go` の `tea.KeyPressMsg` の分岐で、close の確認画面の分岐を merge の確認画面の分岐と同じ位置（`a` / `t` /
`m` / `o` / `?` / `u` より先、`screenConfirm` の後）に置く。後ろだと確認中に裏の対象へ書き込みが起きる。`c` 自身のハンドラは
`m` のハンドラの次に置く。

### 書き込み中フラグは既存の 1 つを共有する

`m.writing` を close にも使う（s11 design の「投稿中フラグと 1 つで共有してよい」に従う）。対象ごとの排他は持たない。

## Risks / Trade-offs

- **取得後に人が GitHub 側で close していると、`y` が失敗する** → フッタに赤で `<表示名> の close に失敗: <gh のエラー>` を出す。
  `gh` のエラーには実行した引数と stderr が入るので、原因は読める（s03 `gh-client`）。Q1 が B に決まればこの窓は縮む
- **close は TUI から取り消せない** → 確認画面を必須にし、拒否があるときは `y` を出さない。reopen はブラウザ（`o`）に委ねる
- **`wip` が付いた issue（AI が作業中）を close できる** → 確認画面の `labels:` で見せるが、止めはしない。止めると
  「取り下げる・止める」（mvp.md 前提と未決事項）ができなくなり、この change の目的を外す
- **キューに残った行から `c` を 2 回押すと 2 回 close される** → 1 回目の書き込み中は `c` を受け付けない。2 回目の `gh` は
  すでに closed なものへの close なので、GitHub 側は状態を変えない
