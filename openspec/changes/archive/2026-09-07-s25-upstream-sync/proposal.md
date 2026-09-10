## Why

上流 issue-driven-sdd が `2b1b791`（2026-09-07）で routine 群を作り直した。1 日 170 session のうち半分が無駄起動・重複実装だった運用監査と、Routines / GitHub コネクタの実挙動の検証がもとで、loop-cli の分類器が前提にしていた状態機械が 3 点変わった。

1. **PR の AI リスク評価が `ai-assess:requested` ラベルで起動するようになった。** worker は未確定 0 件になった時点でこのラベルを付け、評価を終えた assess が外す。loop-cli は `question` が無く未確定 0 件・checks 緑の PR を局面 C（merge する）に出すので、**AI 評価がまだ走っていない PR を人に「merge して」と見せる**。上流は評価前の merge を想定していない。
2. **ラベルの書き込みが集合置換になり、ブロック解除と死んだ worker の再起動が「`[]` を書いてから `[stage:X]` を書く」2 回書きになった。** 1 回目のあとで死ぬと段階ラベルの無い issue が残り、sweep が `release:` / `restart:` / `advance:` コメントを目印に続きを書く。loop-cli はこの issue を局面 E（着手を承認する）としてバックログに出すので、**人が `stage:todo` を付けて段階を巻き戻す**おそれがある。
3. **`blocked-by: human` に `unblock-when:` が加わった。** 何をすれば解けるか（`comment` / `docs` / `#m`）が正本コメントに書かれるようになったが、カード詳細は `blocked-by:` 行しか要約に出さず、質問がパースできた場合は `unblock-when:` を捨てている。

あわせて `docs/domain/issue-driven-sdd/human-turn-signals.md` の同期点が `d8db3842` のままで、書き込みの不変条件 1 が根拠にしている「dispatcher は付与イベントの時刻で判定する」も上流では session id 判定に変わっている。

## What Changes

- `model.LabelAIAssess`（`ai-assess:requested`）を足す
- 分類器に進行中の規則を 2 つ足す
  - PR: `ai-assess:requested` が付いている open PR は「AI 評価待ち」。行 A の後・行 C の前に評価する（質問が残っている PR は人の番が先）
  - issue: 段階ラベルが無く `blocked` も無く、最新の routine コメントが `release:` / `restart:` / `advance:` の issue は「段階ラベルの書き直し中」。行 E より先に評価する
- `model.IsMidRelabel` と `model.UnblockWhen` を足す
- カード詳細の `blocked-by: human` の要約に `unblock-when: <値>` の 1 行を出し、質問がパースできないときの本文からもその行を除く
- `action.CheckMerge` の警告に `ai-assess:requested が付いています（AI 評価が未完了）` を足す。分類で C に出なくなっても、カード詳細や PR 詳細から `m` を押す経路が残るため
- `docs/domain/issue-driven-sdd/human-turn-signals.md` を上流 `2b1b791` に同期する（判定表・キューに入れないもの・不変条件 1・変更履歴）
- `CheckMerge` の未実装だった拒否条件「`State` が空文字列でも `OPEN` でもなければ拒否」を実装する。`merge-action` の Requirement を書き直すにあたって、メイン spec の記述と実装が食い違っているのを見つけた（s14 の取りこぼし）。カード詳細の PR 一覧は merged / closed の PR も選べるので、実害のある取りこぼしである

`ai-assess:requested` を TUI から付ける操作（上流の「PR をもう一度 AI に評価させる」）は入れない。人の出番を並べるのがこのツールの役目で、AI の再起動は今のところ GitHub 側で足りる。

## Capabilities

### Modified Capabilities

- `human-turn-classify`: 進行中の規則を 5 つから 7 つに増やし、要約表と評価順の記述を追随させる
- `card-detail`: 本文領域の `blocked-by: human` の要約に `unblock-when:` を足す
- `merge-action`: `CheckMerge` の警告に `ai-assess:requested` を足す
