## ADDED Requirements

### Requirement: close の確認画面はラベル名に色を付ける

close の確認画面の `labels: <Labels を空白区切り>` の各ラベル名は、`queue-screen`「ラベル名は GitHub のラベル色を背景に、輝度で選んだ黒か白を文字にして描く」のとおり MUST 色を付ける。色は対象のリポジトリのラベル色の表から引く。ラベルが 1 件も無いときの `なし` はラベル名ではないので色を付けない。`close の確認:` / `種別:` / `labels:` の見出しと `注意: <内容>` の行には色を付けない。

ANSI エスケープを除いた表示は、この change の前と MUST 一致する。

#### Scenario: 確認画面のラベルに色が付く
- **WHEN** `stage:propose`（`0e8a16`）と `question`（`d876e3`）が付いた issue で close の確認画面を開いた `Model` の `View` を読む
- **THEN** `stage:propose` は背景色 `0e8a16`、`question` は背景色 `d876e3` で描かれ、`labels:` の見出しに色は付かず、ANSI エスケープを除いた行は `labels: stage:propose question` である

#### Scenario: ラベルが無いときの なし に色は付かない
- **WHEN** ラベルの無い issue で close の確認画面を開いた `Model` の `View` を読む
- **THEN** `labels: なし` の行は色を指定する ANSI エスケープを 1 つも含まない
