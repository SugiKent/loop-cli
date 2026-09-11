## 1. 色の規則

- [ ] 1.1 `internal/ui/color.go`（新規）: 16 進 6 桁から相対輝度を求めて黒か白の文字色を選ぶ関数、ラベル名を背景色 + その文字色で描く関数、状態語を固定の 4 色（緑 `#0E8A16` / 赤 `#B60205` / 黄 `#FBCA04` / 紫 `#5319E7`）の文字色で描く関数と「語 → 色」の表を置く。色が空・16 進 6 桁として読めない・表に無い語は、色を付けずにそのまま返す
- [ ] 1.2 `internal/ui/color_test.go`（新規）: `fbca04` と `d876e3` で黒、`b60205` と `0075ca` で白の文字色が選ばれること、色が引けない入力と表に無い語が ANSI エスケープを含まない文字列で返ること、描いた文字列から ANSI を除くと入力と一致することを検証する

## 2. ラベル色の配り方

- [ ] 2.1 `internal/fetch/fetch.go`: `Result` に `LabelColors map[string]map[string]string` を足し、`ListLabels` の応答から「ラベル名 → 色」の表を作って入れる（`ListLabels` の呼び出しは増やさない。方式を判定できなかったリポジトリも表に入れ、`ListLabels` が失敗したリポジトリは入れない）
- [ ] 2.2 `internal/fetch/fetch_test.go`: `example` の fixture で `LabelColors` が埋まること、方式を判定できないリポジトリにも色が返ること、`ListLabels` が失敗したリポジトリが `LabelColors` に入らないことを検証する
- [ ] 2.3 `internal/ui/model.go`: `Model` にリポジトリごとのラベル色の表を足し、取得完了のメッセージで丸ごと差し替え、取得失敗では触らない（運用方式の表と同じ場所・同じ規則）。リポジトリ名とラベル名から色を引くヘルパを置く
- [ ] 2.4 `internal/ui/model_test.go`: 取得成功で表が入れ替わること、取得失敗で前回の表が残ることを検証する

## 3. 描画箇所

- [ ] 3.1 `internal/gh/testdata/fixtures/` に、色が空のラベルと、`labels.json` に無いラベルが付いた issue を持つ fixture を足す（色が引けない経路を手元で再現できるようにする）
- [ ] 3.2 `internal/ui/preview.go`: プレビュー 1 行目の `labels: ` の各ラベル名を 1.1 の関数で描く
- [ ] 3.3 `internal/ui/detail.go` の `cardHeaderLines`: `段階: ` の段階ラベル名と `[blocked]` / `[wip]` / `[question]` の badge の名前を描く（角括弧と `段階なし` は塗らない）
- [ ] 3.4 `internal/ui/detail.go` の `prListRow` / `prHeaderLines`: `[<段階>]` の段階ラベル名と `labels: ` の各ラベル名、PR の状態、`checks` と `mergeable` に続く値を描く（`[-]` と `なし` と見出しは塗らない）
- [ ] 3.5 `internal/ui/detail.go` の `checkLines` / `commentSection` / `reviewThreadLines`: `mergeable:` の 2 つの値、checks の各行のチェック名に続く値、`取得失敗` を描く
- [ ] 3.6 `internal/ui/labels.go` の `labelLine`: 一覧の各行の名前を、その行の `gh.RepoLabel.Color` で描く（`Model` の表は引かない）
- [ ] 3.7 `internal/ui/merge.go` の `renderMergeConfirm`: `labels: ` の各ラベル名、`mergeable:` の 2 つの値、`checks:` の値を描く（`なし` と見出しは塗らない）
- [ ] 3.8 `internal/ui/close.go` の `renderCloseConfirm`: `labels: ` の各ラベル名を描く（`なし` と見出しは塗らない）
- [ ] 3.9 `internal/ui` の表示テスト: 各画面で狙ったラベル名・状態語に期待した色が付くこと、色が引けないラベルに色が付かないこと、ANSI エスケープを除いた表示が既存 Scenario の期待値と一致することを検証する

## 4. 確認

- [ ] 4.1 `go build ./... && go vet ./... && go test ./...` が通る
- [ ] 4.2 `golangci-lint run` が通る
- [ ] 4.3 `openspec validate s33-colorful-labels --strict` が通る
