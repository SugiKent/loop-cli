## 1. 既定を展開にする

- [ ] 1.1 `detailState`（`internal/ui/detail.go:38-44`）のフィールド `expanded bool` を `collapsed bool` に
  変える。`case "x"`（同 `:103-105`）は `collapsed` を反転して `refreshDetail` を呼び、コメントを描く箇所
  （同 `:375`）は `commentBlock` の最後の引数へ `!m.detail.collapsed` を渡す。`openDetail` はゼロ値で
  `detailState` を作るままにし、初期化を書き足さない
- [ ] 1.2 `card-detail` の MODIFIED Requirement「routine コメントは既定で展開し、x で折りたたむ」の Scenario
  「開いた直後は AI コメントの全行が出る」を `internal/ui/detail_test.go` の
  `TestAICommentIsCollapsedAndExpandedByX`（`:549`）を作り直す形で書き、テスト名も既定に合わせて直して通す
- [ ] 1.3 同 Requirement の Scenario「x で折りたたむと見出しだけになる」を 1.2 と同じテストの後半として書き、
  `x` の 1 回目で `▌AI  18:00  Q1: セッションの寿命は何日にしますか。  (+1 行)` に畳まれ、2 回目で
  全行に戻ることを検証して通す
- [ ] 1.4 同 Requirement の Scenario「開き直すと展開に戻る」を `TestReopenCollapsesAgain`（`:594`）の
  書き換えで通す。`x`（折りたたみ）→ `Esc` → `Enter` の後に全行が出ることを見る
- [ ] 1.5 同 Requirement の Scenario「PR リスク評価の見出しを持つコメントも AI として扱う」を
  `TestRiskHeadingCommentIsCollapsedAsAI`（`:840`）の書き換えで通す。開いた直後は `影響範囲: 小` が出て、
  `x` の後に `(+4 行)` の見出し 1 行になることを見る
- [ ] 1.6 `TestPRDetailOfPR131`（`:749`）の期待順から `▌AI  19:31  Q1: マイグレーションを分けますか。  (+0 行)`
  を外し、展開時の見出し `▌AI  19:31` に差し替える。高さ 40 の端末で `── review thread ` と
  `thread 未 resolve` がまだ本文領域に見えるかを実際に走らせて確かめ、見えなくなっていれば期待順を
  そこまでに切って、切った理由をテストのコメントに 1 行書く
- [ ] 1.7 `go test ./internal/ui/...` を走らせ、この節で挙げていないテストが落ちていないことを確かめる。
  落ちていたら、既定が展開になったことで期待値が変わったのか実装の取りこぼしかを判定して直す

## 2. フッタのヒント

- [ ] 2.1 `internal/ui/detail.go` のフッタのヒント 3 本（`:477`, `:482`, `:484`）の `x 展開` を `x 畳む` に
  変える。他のキーの並びと文言は変えない
- [ ] 2.2 `card-detail` の MODIFIED Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」の
  Scenario「詳細画面のフッタ」と「PR の無いカードのフッタには PR のキーを出さない」に合わせて、
  `TestDetailFooters`（`:998`）と `TestFooterWithoutPRs`（`:1029`）の期待文字列を `x 畳む` に直して通す

## 3. ドキュメント

- [ ] 3.1 `README.md` の「画面とキー操作」にある `x` の行を 2 か所（カード詳細 `:172`、PR 詳細 `:192`）
  書き換え、「AI コメントを畳む / 戻す」の意味にする

## 4. 通し確認

- [ ] 4.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [ ] 4.2 `openspec validate --strict` が緑であることを確認する
