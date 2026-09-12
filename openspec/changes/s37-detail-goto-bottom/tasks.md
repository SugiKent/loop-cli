## 1. 末尾と先頭へ送るキー

- [ ] 1.1 `updateDetailKey`（`internal/ui/detail.go:88-131`）の `switch` に `case "G", "end"`（本文領域の末尾へ）と
  `case "home"`（本文領域の先頭へ）を足す。どちらも `pgup` の次に置き、`refreshDetail` は呼ばない
- [ ] 1.2 `card-detail` の MODIFIED Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」の
  Scenario「G でカード詳細の本文が末尾まで動く」を `internal/ui/detail_test.go` に足して通す。
  60 段落の `Body` と `Comments` が nil の Card は既存のスクロールのテストから借りる
- [ ] 1.3 同 Requirement の Scenario「Home で本文の先頭へ戻る」と「End は G と同じ末尾へ動く」を足して通す
- [ ] 1.4 同 Requirement の Scenario「review thread の無い PR では G でコメントの末尾が出る」を足して通す。
  `Comments` が `コメント01` から `コメント30` までの 30 件で `ReviewThreads` が長さ 0 の open PR を
  手書きの Card で作る
- [ ] 1.5 同 Requirement の Scenario「review thread を持つ PR では G が最後のコメントを通り越す」を足して通す。
  1.4 の PR の `ReviewThreads` を、`返信01` から `返信20` までの 20 件を持つ未 resolve の thread 1 本に差し替える

## 2. ヘルプの一覧

- [ ] 2.1 `helpKeys`（`internal/ui/help.go:11-31`）の末尾に `G / End`（`本文の末尾へ飛ぶ（詳細）`）と
  `Home`（`本文の先頭へ飛ぶ（詳細）`）の 2 行を足す
- [ ] 2.2 `help-screen` の MODIFIED Requirement「ヘルプ画面は実装済みのキーだけを一覧する」の Scenario
  「実装済みのキーの行が順に出る」を `internal/ui/help_test.go` で更新し、キーの行が 21 行になること、
  新しい 2 行が末尾に順に来ることの 2 つを検証して通す
- [ ] 2.3 同 Requirement の Scenario「低い端末では末尾の行を切る」が、行が 2 本増えた後も通ることを確かめる。
  あわせて既定の高さ 24 で全 21 行が出る（末尾が切れない）ことを 2.2 の Scenario で見る

## 3. キュー画面の台帳

- [ ] 3.1 `queue-screen` の MODIFIED Requirement「j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、
  他のキーは何もしない」の Scenario「未実装のキーは何も変えない」に `G` / `Home` / `End` を足し、
  `TestUnimplementedKeysDoNothing`（`internal/ui/model_test.go:87-110`）を更新して通す

## 4. ドキュメント

- [ ] 4.1 README の「画面とキー操作」にある 2 つの表を直す。カード詳細の表（`README.md:164` 付近）と
  PR 詳細の表（同 `:184` 付近）に、`G` / `End` で本文領域の末尾へ、`Home` で先頭へ飛ぶ行を 2 本ずつ足す

## 5. 通し確認

- [ ] 5.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [ ] 5.2 `openspec validate --strict` が緑であることを確認する
