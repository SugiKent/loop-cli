issue: #34

## Why

mvp.md は「行の色は種別で固定。GitHub 側のラベル色（`question` D876E3、`blocked` B60205、`propose` 0E8A16 など）と揃える」と書いているが、実際に色が付いているのはキューの表の行だけで、ラベル名そのものは全画面で無色のまま並んでいる（`internal/ui/labels.go` の `labelLine` は「ラベルの色（`gh label list` の `color`）は使わない」と明記し、`label-picker` spec もそう定めている）。

その結果、次の 2 つが読めない。

- **ラベルがどれも同じ色に見える**。PR 詳細のヘッダは `[propose] open  labels: propose question ai-assess:requested` の 1 行で、どれが段階でどれが人待ちかを語の意味から読み直さないと分からない。ラベル一覧画面（`L`）は 14 行が同じ色で並ぶ
- **`MERGEABLE` / `CONFLICTING` のような状態語が、良い状態か悪い状態か一目で分からない**。PR 一覧の行は `checks 緑以外 mergeable UNKNOWN` のように、良し悪しが逆の語が同じ色で 1 行に混ざる

ラベルの色は `gh label list` の `color` として既に手元にある。`internal/fetch` は運用方式の判定のためにリポジトリごとに `ListLabels` を毎回 1 回呼んでおり（`card-fetch`「Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する」）、色はその応答に含まれたまま捨てられている。ラベル一覧画面に至っては `gh.RepoLabel.Color` を手元に持ったうえで使っていない。**追加の API 呼び出しは要らない。**

## What Changes

- ラベル名を、背景色 = GitHub のラベル色、文字色 = その背景色の相対輝度から選んだ黒または白、で描く。文字色を輝度から決めることが「視認性を維持する」の中身で、任意の 16 進色に対して本文とラベルのコントラストを保証する唯一の方法である（GitHub 自身のラベル表示と同じ規則）
- 色が付くのはラベル名の文字の範囲だけで、その前後に空白を足さない。**画面から ANSI エスケープを除いた文字列は 1 文字も変わらない**（既存の spec の Scenario・幅の計算・折り返し・切り詰めがそのまま通る）
- ラベル名が出る 6 か所すべてを同じ規則で塗る: キューのプレビュー 1 行目 / カード詳細のヘッダ（`段階:` の段階ラベルと `[blocked]` `[wip]` `[question]` の badge）/ カード詳細の PR 一覧行 / PR 詳細のヘッダ / merge の確認画面 / close の確認画面 / ラベル一覧画面（`L`）
- 状態を表す短い語に固定の 3 色（緑 `#0E8A16` / 赤 `#B60205` / 黄 `#FBCA04`。`propose` / `blocked` / `stage:todo` のラベル色で、mvp.md の行の色と同じ出典）を文字色として付ける。対象は `mergeable <値>` / `mergeStateStatus` / `StatusCheckRollup` の各状態 / `checks 緑` と `checks 緑以外` / `取得失敗` / PR の状態（`open` / `merged` / `closed`）
- 状態語には背景色を付けず文字色だけにする。ラベル（背景色あり）と状態語（文字色だけ）が見た目で分かれ、`[propose] open  labels: propose` の 1 行でラベルと状態を取り違えない
- ラベル色の配り方: `internal/fetch` の `Result` に「リポジトリ名 → ラベル名 → 色」の表を足し、`internal/ui` の `Model` が運用方式の表と同じ規則（取得成功で丸ごと差し替え、失敗では触らない）で持つ。ラベル一覧画面は自分が取った `gh.RepoLabel.Color` をそのまま使い、この表を引かない
- 色が引けないラベル（`ListLabels` が失敗したリポジトリ、取得後に GitHub 側で色が変わったラベル）は、今までどおり色を付けずに描く。色が無いことでラベルが消えたり文字が変わったりはしない

キューの表そのものは変えない。表にはラベル列が無く、ラベルを出すのは列を足す別の話になる。行の色は今までどおり種別で固定する（mvp.md）。

## Capabilities

### New Capabilities

（無し）

### Modified Capabilities

- `card-fetch`: `Result` に「リポジトリ名 → ラベル名 → 色」の表を足す。「ラベル一覧そのものは `Result` に持たせず、判定した方式だけを返す」を、色だけは返すように改める
- `queue-screen`: `Model` がラベル色の表を運用方式の表と同じ規則で持つ。プレビューの 1 行目の `labels: ` のラベル名に色を付ける
- `card-detail`: カード詳細のヘッダの段階ラベルと badge、PR 一覧行の `labels:` と `mergeable` / `checks` / PR の状態、PR 詳細のヘッダの `labels:` と段階と状態、本文の `mergeable:` の行と checks の各行に色を付ける
- `label-picker`: 「ラベルの色（`gh label list` の `color`）は使わない」を、一覧の各行のラベル名に色を付けるように改める
- `merge-pr`: merge の確認画面の `labels:` / `mergeable:` / `checks:` に色を付ける
- `close-issue-pr`: close の確認画面の `labels:` に色を付ける

## Impact

- `internal/ui`（新規ファイル 1 本）: 16 進色から相対輝度で黒白を選ぶ関数と、ラベル名・状態語を描く 2 つの関数
- `internal/ui/preview.go` / `detail.go` / `labels.go` / `merge.go` / `close.go` / `model.go`: 上の関数を通して描くように書き換え、`Model` にラベル色の表を足す
- `internal/fetch/fetch.go`: `ListLabels` の応答から色の表を作り `Result` に入れる（呼び出しは増えない）
- テスト: `internal/ui` の既存の表示テストは ANSI を除いて読むので文言の期待値は変わらない。色の有無は ANSI を除く前の文字列で検証する Scenario を足す
- 依存の追加は無い（Lip Gloss v2 の `Style` と、その端末ごとのダウンサンプリングは D-003 のとおり既存）

## 確定した判断

- **ラベル色の出どころは既存の `ListLabels` 応答**。`internal/fetch/fetch.go:115-133` がリポジトリごとに毎回 1 回呼び、`model.ModeFromLabels` に渡した後で色ごと捨てている。`gh.RepoLabel` は既に `Color` を持つ（`internal/gh/types.go`）。gh の引数（`internal/gh/client.go:118` の `--json name,description,color`）も変えない
- **ラベル一覧画面は新しい表を要らない**。`labelPickerState.labels` が `[]gh.RepoLabel` で、色は既にその場にある（`internal/ui/labels.go:20-28`）
- **`model.Issue.Labels` / `model.PR.Labels` は `[]string` のまま**にし、色はリポジトリ単位の表から引く。構造体にすると `internal/classify` / `internal/action` / `internal/snapshot` まで波及するが、色は表示の都合でしかない
- **文字色は背景色の相対輝度から黒か白を選ぶ**。ラベル色は利用者のリポジトリごとに任意の 16 進値なので、固定の文字色ではコントラストを保証できない
- **表示テキストを変えない**。色は ANSI エスケープとしてだけ増え、`ansi.Strip` した結果は今と同一にする。`ansi.Truncate` / `ansi.StringWidth` は ANSI を数えないので、幅・折り返し・切り詰めの既存の規則をどこも直さずに済む

## 未確定の判断

### Q1. 色を付ける範囲を、ラベルと `mergeable` の値だけに絞るか、状態を表す語まで広げるか

- 選択肢 A（推奨）: 状態語まで広げる。`mergeable MERGEABLE` / `mergeStateStatus` / checks の各状態（`SUCCESS` / `FAILURE` / `PENDING` …）/ `checks 緑` と `checks 緑以外` / `取得失敗` / PR の状態（`open` / `merged` / `closed`）を、緑・赤・黄の 3 色のどれかで塗る。PR 詳細を開いたとき「赤が 1 つでもあるか」を読む前に見つけられる。塗る語が増えても仕組みは「語 → 色」の表 1 つで、実装は増えない
- 選択肢 B: ラベル名と `mergeable` の値だけにする。issue の字面に最も近く、画面の色数が最小になる。checks の行（`test: SUCCESS` / `ci/legacy: PENDING`）は無色のまま残る
- 依存: なし

### Q2. ラベルの背景色を、名前の文字の範囲だけに付けるか、左右に空白を 1 つずつ足した帯にするか

- 選択肢 A（推奨）: 文字の範囲だけ。`labels: propose question` の表示テキストが 1 文字も変わらず、既存の spec の Scenario と幅の計算をどこも直さない。背景色の帯が隣の語と接するので、GitHub のラベルよりは窮屈に見える
- 選択肢 B: 左右に空白を足して ` propose ` の帯にする。GitHub のラベルの見た目に近く読みやすいが、表示テキストが変わるので `card-detail` / `merge-pr` / `close-issue-pr` / `queue-screen` の既存 Scenario（`labels: propose question` のような期待値）と、1 行に収まるかの判定を全部書き直す
- 依存: なし
