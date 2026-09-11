## ADDED Requirements

### Requirement: ラベル名は GitHub のラベル色を背景に、輝度で選んだ黒か白を文字にして描く

`internal/ui` は、画面に出すラベル名を MUST 次の規則で描く。この規則はラベル名が出るすべての画面（キューのプレビュー、カード詳細、PR 詳細、merge の確認、close の確認、ラベル一覧）に等しく当てる。

- **背景色**はそのラベルの GitHub 上の色（`gh label list` の `color`。`#` の無い 16 進 6 桁）
- **文字色**は背景色の相対輝度から決める。sRGB の各成分を 0–1 に正規化し、`c ≤ 0.03928` なら `c / 12.92`、そうでなければ `((c + 0.055) / 1.055)^2.4` を掛け、`0.2126R + 0.7152G + 0.0722B` を相対輝度 `L` とする。`L > 0.179` なら黒、そうでなければ白を文字色にする（`0.179` は黒と白のコントラスト比が等しくなる点で、このしきい値は常にコントラスト比の高い方を選ぶ）
- 色が引けないラベル（色を持たないリポジトリの表に無い、値が空、16 進 6 桁として読めない）は、色を付けずに名前を描く

色を付けるのはラベル名の文字の範囲だけで、その前後に空白や記号を MUST 足さない。ラベル名を囲む角括弧（`[propose]` や `[question]`）には色を付けず、括弧の中の名前だけを塗る。この規則により、**画面から ANSI エスケープを除いた文字列は色を付ける前と一致する**。

#### Scenario: 明るいラベル色には黒、暗いラベル色には白の文字色を選ぶ
- **WHEN** `fbca04` / `b60205` / `d876e3` を背景色としてラベル名を描く
- **THEN** `fbca04` と `d876e3` の文字色は黒、`b60205` の文字色は白である

#### Scenario: 色を付けても表示テキストは変わらない
- **WHEN** ラベルの色が引ける状態と引けない状態のそれぞれで `View` を読み、ANSI エスケープを除く
- **THEN** 2 つの文字列は一致する

#### Scenario: 色が引けないラベルは色を付けずに描く
- **WHEN** リポジトリのラベル色の表に無いラベル名を持つ Card を選んで `View` を読む
- **THEN** そのラベル名は ANSI エスケープを除く前後で同じ形で現れる（背景色を指定するエスケープを伴わない）

### Requirement: Model はリポジトリごとのラベル色を持つ

`internal/ui` の `Model` は「リポジトリ名 → ラベル名 → 色」の表を状態として MUST 持つ。表の中身は `internal/fetch` の `Result.LabelColors`（`card-fetch`「Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する」）で、取得が成功したメッセージを受け取るたびに丸ごと差し替える。取得が失敗したメッセージでは触らず、前回の表を残す（D-002「失敗時は前回結果を維持する」。運用方式の表と同じ規則）。

ラベルの色を引くときは、そのラベルが付いている issue / PR のリポジトリで引く。`Options` はラベル色を受け取らず、`cmd/loop-cli` はラベル色を作らない。

#### Scenario: 取得の成功で表が入れ替わる
- **WHEN** `stage:propose` を `0e8a16` とする `LabelColors` を持つ取得完了のメッセージを渡し、続けて同じラベルを `1d76db` とする `LabelColors` を持つ取得完了のメッセージを渡す
- **THEN** `stage:propose` の背景色は 1 回目が `0e8a16`、2 回目が `1d76db` になる

#### Scenario: 取得の失敗では前回の表が残る
- **WHEN** 上の 1 回目の後に取得失敗のメッセージを渡して `View` を読む
- **THEN** `stage:propose` は引き続き `0e8a16` の背景色で描かれる

### Requirement: 状態を表す語は固定の 4 色を文字色にして描く

`internal/ui` は、merge 状態・checks・PR の状態を表す語に MUST 次の色を文字色として付ける。背景色は付けない（ラベルは背景色の帯、状態語は文字色だけ、という見た目の違いで両者を区別する）。色を付けるのは値の語だけで、`mergeable ` や `checks ` のような見出しには付けない。

| 色 | 語 |
| --- | --- |
| 緑 `#0E8A16` | `MERGEABLE`、`CLEAN`、`SUCCESS`、`緑`、`open` |
| 赤 `#B60205` | `CONFLICTING`、`DIRTY`、`BLOCKED`、`FAILURE`、`ERROR`、`TIMED_OUT`、`STARTUP_FAILURE`、`緑以外`、`closed`、`取得失敗` |
| 黄 `#FBCA04` | `UNKNOWN`、`BEHIND`、`UNSTABLE`、`DRAFT`、`HAS_HOOKS`、`PENDING`、`QUEUED`、`IN_PROGRESS`、`WAITING`、`ACTION_REQUIRED`、`EXPECTED` |
| 紫 `#5319E7` | `merged` |

表に無い語（`SKIPPED`、`NEUTRAL`、`CANCELLED` を含む）には色を付けない。4 色は既存のラベル色（`propose` / `blocked` / `stage:todo` / `archive`）から採っており、mvp.md「GitHub 側のラベル色と揃える」に従う。ラベル名と同じく、色を付けても表示テキストは変わらない。

#### Scenario: 良し悪しで色が分かれる
- **WHEN** `MERGEABLE` / `CONFLICTING` / `UNKNOWN` / `merged` を状態語として描く
- **THEN** 文字色はそれぞれ `#0E8A16` / `#B60205` / `#FBCA04` / `#5319E7` で、どれにも背景色が付かない

#### Scenario: 表に無い語には色を付けない
- **WHEN** `SKIPPED` を状態語として描く
- **THEN** 色を指定する ANSI エスケープを伴わずに現れる

### Requirement: プレビューの 1 行目のラベル名に色を付ける

キュー画面のプレビューの 1 行目に出る `labels: <Labels を空白区切り>` の各ラベル名は、Requirement「ラベル名は GitHub のラベル色を背景に、輝度で選んだ黒か白を文字にして描く」のとおり MUST 色を付ける。`labels: ` の見出しと名前の間の空白には色を付けない。キューの表の行は変えない（行の色は種別で固定のまま。表にラベル列は無い）。

#### Scenario: プレビューのラベルに色が付き、表の行の色は変わらない
- **WHEN** `stage:propose`（`0e8a16`）と `question`（`d876e3`）が付いた issue の Card を選んで `View` を読む
- **THEN** プレビューの 1 行目の `stage:propose` は背景色 `0e8a16`、`question` は背景色 `d876e3` で描かれ、ANSI エスケープを除いた 1 行目は `labels: stage:propose question` を含み、表の選択行の色は種別の色のままである
