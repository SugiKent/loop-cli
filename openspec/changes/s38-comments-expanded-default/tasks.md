## 1. 折りたたみを廃止する

- [x] 1.1 `commentBlock`（`internal/ui/preview.go:50-65`）から引数 `expanded` と折りたたみの枝を消し、
  AI かどうかで `▌` を付けるかだけを分けて常に全行を返す形にする。`summarize`（同 `:69-78`）を関数ごと消し、
  `commentLines`（同 `:43`）の呼び出しから引数を 1 つ落とす
- [x] 1.2 `detailState`（`internal/ui/detail.go:38-44`）からフィールド `expanded` を消し、`case "x"`
  （同 `:103-105`）を消し、コメントを描く箇所（同 `:375`）の呼び出しから引数を 1 つ落とす
- [x] 1.3 `card-detail` の ADDED Requirement「コメントは AI も人も常に全文を出す」の Scenario
  「AI コメントも人のコメントも全文が出る」を `internal/ui/detail_test.go` の
  `TestAICommentIsCollapsedAndExpandedByX`（`:549`）の書き換えで書き、テスト名も新しい振る舞いに直して通す
- [x] 1.4 同 Requirement の Scenario「x を押しても表示は変わらない」を `TestReopenCollapsesAgain`（`:594`）の
  書き換えで書き、`x` でコマンドが返らず画面と行が変わらないこと、`Esc` → `Enter` で開き直しても
  同じ全文が出ることを検証して通す
- [x] 1.5 同 Requirement の Scenario「PR リスク評価の見出しを持つコメントも AI として全文が出る」を
  `TestRiskHeadingCommentIsCollapsedAsAI`（`:840`）の書き換えで通す
- [x] 1.6 `TestPRDetailOfPR131`（`:749`）の期待順から `▌AI  19:31  Q1: マイグレーションを分けますか。  (+0 行)`
  を外し、展開時の見出し `▌AI  19:31` に差し替える。高さ 40 の端末で `── review thread ` と
  `thread 未 resolve` がまだ本文領域に見えるかを実際に走らせて確かめ、見えなくなっていれば期待順を
  そこまでに切って、切った理由をテストのコメントに 1 行書く

## 2. フッタのヒント

- [x] 2.1 フッタのヒント 3 本から `x 展開` とその区切りの空白を落とす（`internal/ui/detail.go:477`, `:482`, `:484`）。
  他のキーの並びと文言はそのまま残す
- [x] 2.2 `card-detail` の MODIFIED Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」に
  合わせて、`TestDetailFooters`（`:998`）と `TestFooterWithoutPRs`（`:1029`）から `x 展開` の期待を外して通す。
  3 つのヒントの表示幅は既存の `TestHintWidths`（`internal/ui/view_test.go:538`）が `ansi.StringWidth` で
  見ているので、その期待値を 136 / 101 / 88 と `? ヘルプ` 57 列目 / `u URL` 64 列目に引き直す
  （幅の検証を別テストで重複させない）

## 3. ヘルプ画面

- [x] 3.1 `helpKeys`（`internal/ui/help.go:11-33`）から `x` の行を落とす
- [x] 3.2 `help-screen` の MODIFIED Requirement「ヘルプ画面は実装済みのキーだけを一覧する」の Scenario
  「実装済みのキーの行が順に出る」に合わせて、`TestHelpListsImplementedKeys`（`internal/ui/help_test.go:186`）の
  行数の期待を 21 から 20 に直し、`x` で始まる行が無いことの検証を足して通す
- [x] 3.3 ヘルプから戻ったときの検証を直す。`TestHelpReturnsToCardDetail`（`internal/ui/help_test.go:66`）から
  `x` を押す操作と `(+1 行)` の検証を外し、戻った後もコメントの全文が出ていることを見る形にして通す。
  これは MODIFIED Requirement「? はヘルプ画面を開き、? か Esc で開いた画面に戻る」の Scenario
  「カード詳細から開いて Esc で戻る」に対応する

## 4. キュー画面の台帳

- [x] 4.1 `queue-screen` の MODIFIED Requirement「j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、
  他のキーは何もしない」に合わせて、`TestUnimplementedKeysDoNothing`（`internal/ui/model_test.go:87-110`）が
  `x` を含んだまま通ることを確かめる。キュー画面の扱いは変わらないので、テストの変更は要らない見込みである

## 5. ドキュメント

- [x] 5.1 `README.md` の「画面とキー操作」にある `x` の行を 2 か所（カード詳細 `:172`、PR 詳細 `:192`）
  消す。表の他の行は変えない
- [x] 5.2 `docs/domain/issue-driven-sdd/human-turn-signals.md` の「詳細画面ではコメントを畳み、…」の
  1 行を、全文表示を前提にした文へ直す（`docs/mvp` は凍結だが `docs/domain` は現在の振る舞いを述べる）

## 6. 通し確認

- [x] 6.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する。
  この change で挙げていないテストが落ちたら、折りたたみを消したことで期待値が変わったのか実装の
  取りこぼしかを判定して直す
- [x] 6.2 `openspec validate --strict` が緑であることを確認する
