## 1. 色の規則

- [x] 1.1 `internal/ui/color.go`（新規）: 16 進 6 桁から相対輝度を求めて黒か白の文字色を選ぶ関数、ラベル名を背景色 + その文字色で描く関数、状態語を良し悪しの 4 色の文字色で描く関数と「語 → 色」の表を置く。4 色は暗い端末用（緑 `#3FB950` / 赤 `#F85149` / 黄 `#D29922` / 紫 `#A371F7`）と明るい端末用（緑 `#1A7F37` / 赤 `#CF222E` / 黄 `#9A6700` / 紫 `#8250DF`）の 2 組を持ち、引数で受けた「背景が暗いか」で選ぶ。色が空・16 進 6 桁として読めない・表に無い語は、色を付けずにそのまま返す
- [x] 1.2 `internal/ui/color_test.go`（新規）: `fbca04` と `d876e3` で黒、`b60205` と `0075ca` で白の文字色が選ばれること、8 つの状態色が自分の側の背景（黒 / 白）に対してコントラスト比 4.5 以上であること、色が引けない入力と表に無い語が ANSI エスケープを含まない文字列で返ること、描いた文字列から ANSI を除くと入力と一致することを検証する
- [x] 1.3 `internal/ui/model.go`: 端末の背景色を伝えるメッセージを `Update` で受け、「背景が暗いか」を `Model` に持つ（受け取るまでは暗いものとして扱う）。`internal/ui/model_test.go` で、明るい背景を伝えた後に状態語の色が明るい端末用の組に変わることを検証する

## 2. ラベル色の配り方

- [x] 2.1 `internal/fetch/fetch.go`: `Result` に `LabelColors map[string]map[string]string` を足す。`fetchDetails` の戻り値に色の表を足し、`modes` と同じ場所で事前に初期化して同じ `mu` で保護する。色は `ModeFromLabels` の判定より前に書き（方式を判定できないリポジトリも表に入れる）、`ListLabels` が失敗したリポジトリは書かない。`ListLabels` の呼び出しは増やさない
- [x] 2.2 `internal/fetch/fetch_test.go`: `example` の fixture で `LabelColors` が埋まること、方式を判定できないリポジトリにも色が返ること、`ListLabels` が失敗したリポジトリが `LabelColors` に入らないことを検証する
- [x] 2.3 `internal/ui/model.go`: `Model` にリポジトリごとのラベル色の表を足し、取得完了のメッセージで丸ごと差し替え、取得失敗では触らない（運用方式の表と同じ場所・同じ規則）。リポジトリ名とラベル名から色を引くヘルパを置く
- [x] 2.4 `internal/ui/model_test.go`: 取得成功で表が入れ替わること、取得失敗で前回の表が残ること、スナップショットだけで起動した直後は色が付かないことを検証する

## 3. 描画箇所

- [x] 3.1 `internal/ui/preview.go`: プレビュー 1 行目の `labels: ` の各ラベル名を 1.1 の関数で描く
- [x] 3.2 `internal/ui/detail.go` の `cardHeaderLines`: `段階: ` の段階ラベル名と `[blocked]` / `[wip]` / `[question]` の badge の名前を描く（角括弧と `段階なし` は塗らない）
- [x] 3.3 `internal/ui/detail.go` の `prListRow` / `prHeaderLines`: `[<段階>]` の段階ラベル名と `labels: ` の各ラベル名、PR の状態、`checks` と `mergeable` に続く値を描く（`[-]` と `なし` と見出しは塗らない）
- [x] 3.4 `internal/ui/detail.go` の `checkLines` / `commentSection` / `reviewThreadLines`: `mergeable:` の 2 つの値、checks の各行のチェック名に続く値、`取得失敗` を描く
- [x] 3.5 `internal/ui/labels.go` の `labelLine`: 一覧の各行の名前を、その行の `gh.RepoLabel.Color` で描く（`Model` の表は引かない）
- [x] 3.6 `internal/ui/merge.go` の `renderMergeConfirm`: `labels: ` の各ラベル名、`mergeable:` の 2 つの値、`checks:` の値を描く（`なし` と見出しは塗らない）
- [x] 3.7 `internal/ui/close.go` の `renderCloseConfirm`: `labels: ` の各ラベル名を描く（`なし` と見出しは塗らない）
- [x] 3.8 `internal/ui` の表示テスト: 各画面で狙ったラベル名・状態語に期待した色が付くこと、色が引けないラベルに色が付かないこと、ANSI エスケープを除いた表示が既存 Scenario の期待値と一致することを検証する。色の検証は「色のエスケープ + 語 + リセット」が連続して現れることで書く（エスケープの有無だけを見ると、切り詰めで消えた語について偽の合格が出る）。PR 一覧行の末尾（`mergeable` の値）を見るケースは幅 100 を使う。色が引けない経路は、手書きの Card と手書きの `labels.json` を持つ fixture ディレクトリで作る（`internal/gh/testdata/fixtures/` には足さない。あそこは `internal/classify` / `internal/fetch` / `internal/gh` の全件走査テストが期待値表と突き合わせている）

## 4. 確認

- [x] 4.1 `go build ./... && go vet ./... && go test ./...` が通る
- [x] 4.2 `golangci-lint run` が通る
- [x] 4.3 `openspec validate s33-colorful-labels --strict` が通る
