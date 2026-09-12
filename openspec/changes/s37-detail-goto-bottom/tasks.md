## 0. 回答の反映

- [ ] 0.1 PR の「未確定の判断」への回答を読み、Q1 / Q2 / Q3 の採用案を確かめる。回答が無いまま merge された
  場合は推奨案（Q1 = `G` の 1 打鍵、Q2 = 最上部のキーは足さない、Q3 = 詳細画面の本文領域だけ）を採る。
  推奨案と違う案を採るなら、先に `proposal.md` の「確定した判断」・`design.md` の D1 / D2 / D4・
  spec delta の文面を書き換えてから 1 章へ進む

## 1. 末尾へ送るキー

- [ ] 1.1 `updateDetailKey`（`internal/ui/detail.go:88-131`）の `switch` に `case "G"` を足し、
  本文領域のスクロール位置を末尾へ動かす。`pgup` の次に置き、`refreshDetail` は呼ばない
- [ ] 1.2 `card-detail` の MODIFIED Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」の
  Scenario「G でカード詳細の本文が末尾まで動く」を `internal/ui/detail_test.go` に足して通す。
  60 段落の `Body` を作る組み立ては既存のスクロールのテストから借りる
- [ ] 1.3 同 Requirement の Scenario「G で PR 詳細のコメントの末尾が出る」を足して通す。`Comments` が
  `コメント01` から `コメント30` までの 30 件で `ReviewThreads` が長さ 0 の open PR を手書きの Card で作る
- [ ] 1.4 同 Requirement の Scenario「本文が収まっているときの G はスクロール位置を動かさない」と
  「G の後にカード詳細へ戻るとスクロール位置は先頭に戻る」を足して通す

## 2. ヘルプの一覧

- [ ] 2.1 `helpKeys`（`internal/ui/help.go:11-31`）の末尾に `G` の行（`本文の末尾へ飛ぶ（詳細）`）を足す
- [ ] 2.2 `help-screen` の MODIFIED Requirement「ヘルプ画面は実装済みのキーだけを一覧する」の Scenario
  「実装済みのキーの行が順に出る」を `internal/ui/help_test.go` で更新し、キーの行が 20 行になること、
  `G` の行が末尾に来ることの 2 つを検証して通す
- [ ] 2.3 同 Requirement の Scenario「低い端末では末尾の行を切る」が、行が 1 本増えた後も通ることを確かめる

## 3. ドキュメント

- [ ] 3.1 README の「画面とキー操作」にある 2 つの表を直す。カード詳細の表（`README.md:164` 付近）と
  PR 詳細の表（同 `:183` 付近）に、`G` で本文の末尾へ飛ぶ行を 1 本ずつ足す

## 4. 通し確認

- [ ] 4.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [ ] 4.2 `openspec validate --strict` が緑であることを確認する
