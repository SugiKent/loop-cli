---
name: assess-pr-risk
description: loop-cli の open PR 1 本のリスクを AI が評価し、結果をコメントして ai-assess:requested を外すスキル。`ai-assess:requested` で起動する assess Routine から呼ばれる。「PR のリスク評価をして」「この PR は人間レビューが要るか」「PR リスク」等のときにも使う。
---

`ai-assess:requested` が付いた PR 1 本を評価し、人が merge する前に見ておくべき点を 1 コメントに
まとめて、ラベルを外す。**merge はしない。** merge するかは人の判断で、loop-cli の今やるタブは
その判断点を人に届けるためにある。assess が merge すると loop-cli を使う意味が消える。

# 対象

`CCR_TRIGGER_PR_NUMBER` の PR だけを見る。他の open PR を見に行かない。評価するコミットは
`CCR_TRIGGER_HEAD_SHA`。環境変数が無い場合は、人に PR 番号を渡されたときだけ動く。

`gh` は Routine のセッションに無い。PR の閲覧・コメント・ラベルの付け外しは GitHub コネクタで行う。

# 手順

1. PR の差分・本文・ラベル・checks を読む。
2. 下の 3 点を評価する。
3. `<!-- routine -->` で始まる 1 コメントを PR に投稿する。
4. **`ai-assess:requested` を外す**（段階ラベルなど他のラベルはそのまま残す）。

**4 は必ず行う。** 評価が途中で失敗しても、失敗したことをコメントしてラベルを外す。外さずに終えると、
loop-cli がその PR を「AI 評価待ち」として今やるタブから外し続け、人の出番が見えなくなる。

# 評価する 3 点

| 点 | 見るもの |
| --- | --- |
| 規約違反 | 本文 1 行目の `未確定の判断: N 件` とラベル（`question`）が一致しているか。title が `[<段階>] #<issue番号> <要約>` か。`propose` / `apply` に `Closes #n` が無いか |
| 仕様との距離 | propose PR なら proposal が対応 issue の求めたものに収まっているか。apply PR なら change の `tasks.md` の範囲に収まっているか |
| 壊しやすさ | CLAUDE.md の「シンプルさを最優先にする」から外れていないか（投機的な機能・一度しか使わないヘルパー・起こり得ない状態への防御）。分類器（`internal/classify`）と判定表の docs がずれていないか |

行数は見ない。loop-cli の PR は openspec の change 単位で切られていて、行数はリスクの目安にならない。
テストの緑は CI が見ており、局面 C が `statusCheckRollup` で確認するので、ここでは重ねて確認しない。

# コメントの形式

```
<!-- routine -->
## PR リスク評価

対象コミットは <SHA>。

| 点 | 判定 | 根拠 |
| --- | --- | --- |
| 規約違反 | <無し / 内容> | |
| 仕様との距離 | <収まっている / 外れている点> | |
| 壊しやすさ | <低い / 気になる点> | |

**人が見るべき点: <箇条書き / 無し>**
```

指摘が無ければ「人が見るべき点: 無し」と書く。書くことが無いのに長く書かない。
