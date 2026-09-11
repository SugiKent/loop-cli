## Context

動機は proposal.md の Why にある（issue #4）。既にある 4 つの書き込みのキーが土台になる。

- `a`（s10 `answer-question`）: `action.Target{Repo, Number, IsPR}` で書き先を分け、`m.writing` で二重投稿を防ぎ、押下時に
  ステータスを空にし、結果をフッタ右に出す。`internal/ui/answer.go`
- `t`（s11 `todo-toggle`）: 確認なしで即書き込み。`internal/ui/todo.go`
- `m`（s14 `merge-pr`）: 押下時に `gh` を呼んで判断材料を取り直し、確認画面（`y` / `Esc`）を出し、`y` で書き込む。
  確認画面の状態（対象・取り直した値・拒否・警告・戻り先）を `Model` の 1 フィールドに持つ。`internal/ui/merge.go`
- `o`（s12 `browse-open`）: 書き込み中でも受け付ける

`c` は `m` の形（確認画面つき）に最も近い。issue #4 も「`internal/ui/merge.go` の一式をなぞって `internal/ui/close.go` として
作るのが素直」と書いている。異なるのは、対象が issue でも PR でもよい点（`a` と同じ規則）と、押下時に `gh` を呼ばない点である。

## Goals / Non-Goals

**Goals:**

- `c` → 確認画面 → `y` で `gh issue close` / `gh pr close` までを 1 本の経路で通し、対象は画面が見せているものに決める
- close が上流の routine に却下として伝わる PR を、押す前に人へ知らせる
- 判定を `internal/action` に置き、`internal/ui` は判定を持たない（`merge-action` と同じ層の分け方）
- 確認画面のキー振り分けを、裏の画面のキーより先に置く

**Non-Goals:**

- close した対象を 1 件だけ再取得すること（D-002 を s18 が担当する）
- close にコメントを添える・ブランチを消す・段階ラベルを外す・close の理由を選ぶ
- close された issue / PR を画面に残す工夫（次の取得で `--state open` の検索から消えるのに任せる）
- reopen（`gh issue reopen`）。取り消しはブラウザ（`o`）で行う
- `internal/ui/merge.go` が `model.PR.State` を写していない既存の spec 違反を直すこと（下記「申し送り」）

## Decisions

### `c` の押下では `gh` を呼ばない

`m` は「merge 可否は表示時に取り直す」（human-turn-signals.md）に従って `ViewPR` と `ViewPRMergeState` を呼ぶが、close は
「open か」だけで決まり、画面に出ている issue と PR は取得時点で必ず open である（`SearchIssues` / `SearchPRs` が `--state open`。
`internal/gh/client.go`。`internal/fetch/fetch.go` は検索の結果からしか Card を作らない）。取り直しを入れると `c` に
レイテンシと失敗経路（取り直し中の画面移動、部分失敗）が増える。取らなかった案は「`m` と同じく取り直す」で、`gh issue view` の `--json` に `state` を足し、`c` の押下を `m` と同じ
2 段（取り直し → 確認画面）にするもの。`c` に 1 往復の待ちが乗るのと、`gh-client` spec と fixture と型が増えるのを
避けて、人の判断（PR #10 のコメント）で取り直さない側を採った。

### 拒否を持たない

`action.CheckMerge` は「`State` が空でも `OPEN` でもなければ拒否」を持つが、この拒否は本番で発火しない。
`model.PRFromSearch` は常に `State: "OPEN"` を入れ（`internal/model/model.go`）、`CrossReferencedPRs` による紐づけ補完は
未実装（mvp.md §3 の P2）なので、closed / merged の PR がカード詳細の PR 一覧に並ぶ経路が無い。同じ形を close に写すと、
CLAUDE.md「起こり得ない状態への防御コードを追加しない」に反する死んだ分岐が増えるので、close は拒否を持たず `y` を常に出す。
代わりに、取得後に GitHub 側で close されていた場合は `gh` の結果をそのままフッタに出す。

**残余リスク**: P2 の cross-reference 紐づけが入って closed / merged の PR がカード詳細に並ぶようになったら、close の入口に
ガードが無い状態になる。そのときに拒否を足す change を切る（`merge-action` の `State` の拒否を実際に働かせる作業と同じ回で行う）。

### 注意は 1 つだけ持つ

`propose` / `apply` ラベルの PR を merge せずに close すると、上流の dispatcher / sweep が「人が却下した」とみなして issue に
`blocked-by: human` を書き戻す（human-turn-signals.md、`routine-sweep` 手順 1、`routine-dispatch` 手順 D）。close が
キューを減らすどころか局面 B の行を作るので、押す前に確認画面で知らせる。それ以外の注意（`wip` が付いている、`question` が
残っている等）は足さない。ラベルは `labels:` の行に出るので、人が読んで判断できる。

### 確認画面の状態は押下時の値を凍結する

確認画面を読んでいる間に自動更新（s13）が走ると `Cards` が入れ替わり選択行が動くので、`y` の時点で対象を導き直すと
別の issue / PR を close しうる。`m` と同じく、押下時にリポジトリ・番号・種別・表示名・タイトル・ラベル・注意・戻り先を
1 つの構造体に写して持つ。`internal/ui/rows.go` の `row` はこれらをすべて持っているので、`row` は変えない
（フィールドを足すと s08 `queue-screen` の spec に波及する）。

### カード詳細の `c` は `Issue` を対象にする

issue #4 の指定に従う。同じ画面で `m` は ▶ で選択中の PR を対象にするので、▶ を PR に合わせている人が `c` を押すと issue が
閉じる。確認画面が `種別: issue` と番号とタイトルを出すので、`y` の前に読める。PR を閉じたい人は `Enter` で PR 詳細へ入る。
取らなかった案は「カード詳細では選択中の PR を対象にする」（`m` とそろうが、issue を閉じる道がカード詳細から消える）と
「カード詳細では `c` を無効にする」（キーの効かない画面が増える）。

### キー振り分けの位置

`internal/ui/model.go` の `tea.KeyPressMsg` の分岐で、close の確認画面の分岐を merge の確認画面の分岐の直後（`a` / `t` /
`m` / `o` / `?` / `u` より先）に置く。後ろだと確認中に裏の対象へ書き込みが起きる。`c` 自身のハンドラは `m` のハンドラの次に置く。

### 書き込み中フラグは既存の 1 つを共有する

`m.writing` を close にも使う（s11 design の「投稿中フラグと 1 つで共有してよい」に従う）。対象ごとの排他は持たない。
`c` の押下では `gh` を呼ばないので、確認画面へ移るところは書き込み中に入らない。

### 申し送り

`openspec/specs/merge-pr/spec.md` の Requirement「取り直しに成功したら確認画面へ移り、失敗したら赤で出す」は
「`State` には画面の Card が持つその PR の `State` を写し」と定めているが、`internal/ui/merge.go` の実装は `State` を写して
おらず、`action.CheckMerge` の `State` の拒否は発火しない。これは s14 の取りこぼしであり、s26 はこの欠落を直さない。
`State` を持つ PR が画面に出るようになる P2（cross-reference 紐づけ）の回にまとめて直すのが安い。

## Risks / Trade-offs

- **`gh` がすでに closed の対象に対して終了コード 0 を返すと、失敗として見えない** → その場合 TUI は `… を close しました` と
  出す。propose の環境にも apply の環境にも `gh` が無く未実測のままなので、spec は「`gh` の結果をそのままフッタに出す」
  とだけ定め、文言を分岐させない。状態は変わらないので実害は表示だけである
- **確認画面の `labels:` が古い** → 自動更新の既定は 120 秒、スナップショットから起動した直後（`internal/snapshot`）は
  前回セッションの値なので、直前に付いた `wip` は出ない。`t` / `a` が画面の値で書き込むのと同じ性質で、この change では受け入れる
- **close は TUI から取り消せない** → 確認画面を必須にする。reopen はブラウザ（`o`）に委ねる
- **`wip` が付いた issue（AI が作業中）を close できる** → 確認画面の `labels:` で見せるが、止めはしない。止めると
  「取り下げる・止める」ができなくなり、この change の目的を外す。走っている worker は close では止まらず、
  PR を作ろうとして失敗するか、sweep が後で片付ける
- **キューに残った行から `c` を 2 回押すと 2 回 close される** → 1 回目の書き込み中は `c` を受け付けない。2 回目の `gh` は
  すでに closed なものへの close なので状態を変えない
