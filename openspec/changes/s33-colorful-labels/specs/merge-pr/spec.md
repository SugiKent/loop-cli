## ADDED Requirements

### Requirement: merge の確認画面はラベル名と merge 状態に色を付ける

merge の確認画面の `labels: <Labels を空白区切り>` の各ラベル名は、`queue-screen`「ラベル名は GitHub のラベル色を背景に、輝度で選んだ黒か白を文字にして描く」のとおり MUST 色を付ける。色は対象 PR のリポジトリのラベル色の表から引く。ラベルが 1 件も無いときの `なし` はラベル名ではないので色を付けない。

`mergeable: <Mergeable> <MergeStateStatus>` の 2 つの値と `checks: <緑 | 緑以外>` の値には、`queue-screen`「状態を表す語は固定の 4 色を文字色にして描く」のとおり MUST 色を付ける。`labels:` / `mergeable:` / `checks:` / `方式:` の見出しには色を付けない。`merge できません: <理由>` と `注意: <内容>` の行は今までどおりで、拒否の行は行全体が赤のまま（`errorStyle`）とする。

ANSI エスケープを除いた表示は、この change の前と MUST 一致する。

#### Scenario: 確認画面のラベルと merge 状態に色が付く
- **WHEN** `propose`（`0e8a16`）と `question`（`d876e3`）が付き、`mergeable` が `MERGEABLE`、`mergeStateStatus` が `CLEAN`、checks が緑の PR で merge の確認画面を開いた `Model` の `View` を読む
- **THEN** `propose` は背景色 `0e8a16`、`question` は背景色 `d876e3` で描かれ、`MERGEABLE` と `CLEAN` と `緑` は文字色 `#0E8A16` で描かれ、`labels:` / `mergeable:` / `checks:` の見出しに色は付かない

#### Scenario: 悪い状態は赤と黄で出る
- **WHEN** `mergeable` が `CONFLICTING`、`mergeStateStatus` が `DIRTY`、checks が緑以外の PR で merge の確認画面を開いた `Model` の `View` を読む
- **THEN** `CONFLICTING` と `DIRTY` と `緑以外` は文字色 `#B60205` で描かれる

#### Scenario: ANSI を除いた表示は変わらない
- **WHEN** 上の 2 つの `Model` の `View` から ANSI エスケープを除いて読む
- **THEN** `merge-pr`「merge の確認画面は判断材料を出し、y で merge して Esc で中止する」の既存の Scenario が定める行がそのまま得られる
