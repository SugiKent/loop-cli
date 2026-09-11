## ADDED Requirements

### Requirement: カード詳細と PR 詳細はラベル名と状態語に色を付ける

カード詳細画面と PR 詳細画面は、次の位置に出るラベル名を `queue-screen`「ラベル名は GitHub のラベル色を背景に、輝度で選んだ黒か白を文字にして描く」のとおり MUST 色を付ける。色はその Issue / PR のリポジトリのラベル色の表から引く。

- ヘッダの `段階: <段階ラベル>` の段階ラベル名（`段階なし` には色を付けない）
- ヘッダの `[blocked]` / `[wip]` / `[question]` の badge。角括弧は塗らず、中の名前だけを塗る
- PR 一覧行の `[<段階>]` の段階ラベル名。段階ラベルが無い PR の `[-]` と、その段階の PR が無い行の `[<段階>] なし` の `なし` には色を付けない（`なし` はラベル名ではない。`[<段階>]` の中の段階ラベル名は塗る）
- PR 一覧行の `labels: <Labels を空白区切り>` の各ラベル名
- PR 詳細のヘッダの `[<段階>]` の段階ラベル名と `labels: <Labels を空白区切り>` の各ラベル名

続けて、次の位置に出る状態語を `queue-screen`「状態を表す語は良し悪しの 4 色を文字色にして描き、色は端末の背景に追随する」のとおり MUST 色を付ける。見出しの語（`mergeable`、`checks`、`labels:`、チェック名）には色を付けない。

- PR 一覧行では、PR の状態（`open` / `merged` / `closed`）と、`checks` に続く値（`緑` / `緑以外` / `取得失敗`）と、`mergeable` に続く値を塗る
- PR 詳細のヘッダでは、PR の状態を塗る
- PR 詳細の本文では、`mergeable: <Mergeable> <MergeStateStatus>` の 2 つの値と、`checks: 取得失敗` の `取得失敗` を塗る
- PR 詳細の本文の checks の各行では、チェック名に続く値を塗る（`CheckRun` なら `Conclusion`、空なら `Status`。`StatusContext` なら `State`）
- 本文では、`コメント: 取得失敗` と `review thread: 取得失敗` の `取得失敗` を塗る

ANSI エスケープを除いた表示は、この change の前と MUST 一致する。ヘッダ領域の行数・本文領域の高さ・折り返し・切り詰めの規則は変わらない。

色が付いたことは、「色を指定するエスケープ・その語・リセット」がこの順で**連続して**現れることで確かめる。エスケープが行のどこかにあることだけを見ると、幅で切り詰められて 1 文字も描かれていない語について偽の合格が出る（切り詰めは切り捨てた範囲のエスケープを行末に残すため）。PR 一覧行は表示幅 91 列に達するので、末尾の `mergeable` の値を見る Scenario は 1 ペインで幅 92 以上（端末幅 120 以上は 2 ペインに分かれて左ペインが 79 列になるので、120 未満）を使う。

#### Scenario: PR 一覧行のラベルと状態に色が付く
- **WHEN** `example` の fixture（PR 131 は `propose` `0e8a16` と `question` `d876e3` が付き、`mergeable` は `UNKNOWN`）でカード詳細を開いた `Model`（幅 100・高さ 24。1 ペインで PR 一覧行の末尾まで入る）の `View` を読む
- **THEN** PR 131 の行の `[propose]` の `propose` は背景色 `0e8a16`、`question` は背景色 `d876e3` で描かれ、`open` は暗い端末の緑、`UNKNOWN` は暗い端末の黄の文字色で描かれ、角括弧・`labels:` の見出し・`checks` と `mergeable` の見出しに色は付かない

#### Scenario: ヘッダの段階ラベルと badge に色が付く
- **WHEN** `stage:propose`（`0e8a16`）と `question`（`d876e3`）が付いた issue のカード詳細を開いた `Model`（幅 100・高さ 24）の `View` を読む
- **THEN** `段階: ` の後の `stage:propose` は背景色 `0e8a16`、`[question]` の中の `question` は背景色 `d876e3` で描かれ、`段階: ` と角括弧に色は付かない

#### Scenario: checks の各行の状態に色が付く
- **WHEN** `test` が `SUCCESS`、`ci/legacy` が `PENDING` の PR 詳細を開いた `Model`（幅 100・高さ 24）の `View` を読む
- **THEN** `SUCCESS` は暗い端末の緑、`PENDING` は暗い端末の黄の文字色で描かれ、`test` と `ci/legacy` のチェック名に色は付かない

#### Scenario: 取得失敗は赤で出る
- **WHEN** `MergeState` が nil の PR を持つカード詳細を開いた `Model`（幅 100・高さ 24）の `View` を読む
- **THEN** PR 一覧行の `checks 取得失敗` と `mergeable 取得失敗` の `取得失敗` は暗い端末の赤の文字色で描かれ、`checks` と `mergeable` の見出しに色は付かない

#### Scenario: ANSI を除いた表示は変わらない
- **WHEN** `example` の fixture でカード詳細と PR 詳細を開いた `Model`（既存の Scenario と同じ幅 80 と幅 120）の `View` から ANSI エスケープを除いて読む
- **THEN** `card-detail` の既存の Scenario が定める行（`[propose] PR#131 open`、`labels: propose question`、`mergeable: UNKNOWN BLOCKED`、`test: SUCCESS` など）がそのまま得られる
