## 0. 回答の反映

- [ ] 0.1 PR の「未確定の判断」への回答を読み、Q1 / Q2 / Q3 の採用案を確かめる。回答が無いまま merge された
  場合は推奨案（Q1 = `G` の 1 打鍵、Q2 = 最上部のキーは足さない、Q3 = 詳細画面の本文領域だけ）を採る。
  推奨案と違う案を採るなら、先に `proposal.md` の「確定した判断」・`design.md` の D1 / D2 / D4・
  spec delta の文面を書き換えてから 1 章へ進む
- [ ] 0.2 Q1 で B / C を採った場合だけ、`g` を空ける波及を先に片付ける。`card-detail` の Requirement
  「PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す」を MODIFIED に足して `g` の
  Scenario 2 本（`openspec/specs/card-detail/spec.md:220`, `:224`）を書き換え、`queue-screen` の未実装キーの
  一覧から `g` を外し、既存テストの `TestGoesBackAndForthBetweenIssueAndPR`（`internal/ui/detail_test.go:155-173`）と
  `TestGDoesNothingOnPROnlyCard`（同 `:176-188`）を新しい打ち方に合わせる。`s33-pr-status-sections` が
  同じ Requirement を MODIFIED しているので、先に merge された方へ追随する

## 1. 末尾へ送るキー

- [ ] 1.1 `updateDetailKey`（`internal/ui/detail.go:88-131`）の `switch` に `case "G"` を足し、
  本文領域のスクロール位置を末尾へ動かす。`pgup` の次に置き、`refreshDetail` は呼ばない
- [ ] 1.2 `card-detail` の MODIFIED Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」の
  Scenario「G でカード詳細の本文が末尾まで動く」を `internal/ui/detail_test.go` に足して通す。
  60 段落の `Body` を作る組み立ては既存のスクロールのテストから借りる
- [ ] 1.3 同 Requirement の Scenario「review thread の無い PR では G でコメントの末尾が出る」を足して通す。
  `Comments` が `コメント01` から `コメント30` までの 30 件で `ReviewThreads` が長さ 0 の open PR を
  手書きの Card で作る
- [ ] 1.4 同 Requirement の Scenario「review thread を持つ PR では G が最後のコメントを通り越す」を足して通す。
  1.3 の PR の `ReviewThreads` を、`返信01` から `返信20` までの 20 件を持つ未 resolve の thread 1 本に差し替える

## 2. ヘルプの一覧

- [ ] 2.1 `helpKeys`（`internal/ui/help.go:11-31`）の末尾に `G` の行（`本文の末尾へ飛ぶ（詳細）`）を足す
- [ ] 2.2 `help-screen` の MODIFIED Requirement「ヘルプ画面は実装済みのキーだけを一覧する」の Scenario
  「実装済みのキーの行が順に出る」を `internal/ui/help_test.go` で更新し、キーの行が 20 行になること、
  `G` の行が末尾に来ることの 2 つを検証して通す
- [ ] 2.3 同 Requirement の Scenario「低い端末では末尾の行を切る」が、行が 1 本増えた後も通ることを確かめる

## 3. キュー画面の台帳

- [ ] 3.1 `queue-screen` の MODIFIED Requirement「j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、
  他のキーは何もしない」の Scenario「未実装のキーは何も変えない」に `G` を足し、
  `TestUnimplementedKeysDoNothing`（`internal/ui/model_test.go:87-110`）を更新して通す

## 4. ドキュメント

- [ ] 4.1 README の「画面とキー操作」にある 2 つの表を直す。カード詳細の表（`README.md:164` 付近）と
  PR 詳細の表（同 `:184` 付近）に、`G` で本文領域の末尾へ飛ぶ行を 1 本ずつ足す

## 5. 通し確認

- [ ] 5.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [ ] 5.2 `openspec validate --strict` が緑であることを確認する
