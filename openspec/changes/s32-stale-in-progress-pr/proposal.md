## Why

2026-09-12 時点で X-Mile/ca-ai-role-play の open PR 5 件（#822 / #824 / #825 / #826 / #827）が、1 件も「今やる」に出ていない。すべて「進行中」に入っている。

- #822 / #824 / #827 は進行中の規則 7（`ai-assess:requested` が付いている）に当たる。このラベルは評価を終えた assess が外す約束だが、このリポジトリの assess Routine は 2026-09-06 を最後に起動しておらず、`assess-pr-risk` skill にもラベルを外す手順が無い。merge 済みの PR（#798 以降）にもラベルが残ったままで、付いたら外れないラベルになっている
- #825 / #826 は進行中の規則 2（`question` 無しで最新コメントが人）に当たる。worker は人の回答を反映して `question` を外し `ai-assess:requested` を付けたが、コメントを残していないので、最新コメントが人のまま止まっている

規則 2 / 3 / 7 は「次に AI が動く」ことを前提に PR を人の目から外すが、AI が来なかったときに戻す仕組みが無い。上流の Routine が止まる・書き忘れるだけで、人の出番の PR が恒久的に見えなくなる。

## What Changes

- PR の進行中の規則 2 / 3 / 7 を、PR が最後に動いてから 3 時間以内のときだけ当てる。3 時間は、上流の sweep が「最新コメントが人のまま routine の返信が無い PR」を止まったとみなす時間に揃える。「最後に動いた時刻」には search が返す `UpdatedAt` を使う。`UpdatedAt` はコメント・ラベルの付け外し・push で更新されるので、追加の API 呼び出しは要らない
- 規則 2 / 3（最新コメントが人）のまま 3 時間を過ぎた PR は、その他の局面で `PR #<n> は人のコメントに AI が応答していない` と出す。判定表に流すと、反映されていない人の依頼が「merge する」に見えるためである
- 規則 7（`ai-assess:requested`）のまま 3 時間を過ぎた PR は、判定表（C / D / F / G / その他）で分類する
- `ai-assess:requested` が付いた PR には規則 2 / 3 を当てない。worker は人の回答を反映し終えてからこのラベルを付けるので、最新コメントが人でも回答は済んでいる
- 上の結果、#822 / #824 / #825 / #826 は C（merge する）、checks が赤の #827 はその他として「今やる」に出る
- 分類を入力だけから決める性質を保つため、`classify.PR` / `classify.Card` / `fetch.Fetch` は現在時刻 `now` を引数で受け取る。`loop-cli` は更新のたびに `time.Now()` を渡す
- issue の規則 1 / 4 / 5 / 6 と、`label` 方式の分類は変えない
- `merge-action` の `ai-assess:requested` の警告はそのまま残す（評価前の PR を merge しようとしたときに気付ける）

## Capabilities

### New Capabilities

（無し）

### Modified Capabilities

- `human-turn-classify`: `PR()` / `Card()` が `now` を受け取る。進行中の規則 2 / 3 / 7 に時間切れを足し、規則 2 / 3 の時間切れをその他の要約として定める。fixture テストは各 PR の `UpdatedAt` を `now` にする
- `card-fetch`: `Fetch` が `now` を受け取り、`classify.Card` に渡す
- `dev-cli`: `classify` サブコマンドが、経過の列に使う現在時刻を `classify.PR` にも渡す

## Impact

- `internal/classify/classify.go` / `card.go`: `now` を引数に足し、しきい値の定数を置き、規則 2 / 3 / 7 に時間切れの分岐を足す
- `internal/fetch/fetch.go`: `Fetch` と `buildCards` の引数に `now` を足す
- `cmd/loop-cli/main.go`: fetcher で `time.Now()` を渡す
- `cmd/loop-cli-dev/classify.go`: `classifyRows` に `now` を渡す
- テスト: `internal/classify`、`internal/fetch`、`internal/snapshot`、`internal/ui`、`cmd/loop-cli` の `Fetch` / `PR` / `Card` 呼び出し
- `docs/domain/issue-driven-sdd/human-turn-signals.md`: 行 C の条件と「キューに入れないもの」に時間切れを書き足し、変更履歴に 1 行足す
- 上流（issue-driven-sdd plugin と ca-ai-role-play の assess Routine / `assess-pr-risk` skill）が止まっている件そのものは、この change では直さない
