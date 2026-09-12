# tasks: s36-classify-reflected-pr

Refs #46

## 1. ドメイン文書（分類器の正本を先に直す）

- [ ] 1.1 `docs/domain/issue-driven-sdd/human-turn-signals.md` の「キューに入れないもの（進行中タブに出す）」の 2 つ目の項
  （`question` も `ai-assess:requested` も無い PR）に、worker が反映を終えた印がある PR を除くことを書き足す。
  印は「本文 1 行目が `未確定の判断: 0 件`」かつ「`updatedAt` が人の最新コメントより後」の 2 つで、印がある PR は判定表へ流す。
  時間切れを説明する項にも、印がある PR は時間切れの `その他` にならないことを足す。最終更新日を更新する。
  `docs/mvp` は凍結（CLAUDE.md）なので触らない。書き足した段落が spec delta の Requirement 本文と食い違わないことを読み合わせる

## 2. 分類

- [ ] 2.1 `internal/classify/classify.go` の `PR()` の規則 2 / 3 の分岐に、worker が反映を終えた PR を外す条件を足す
  （design.md D1 / D2）。`mode != model.ModeLabel && !assess && hasComments && !aiLatest` に加えて、
  「`question` が無く、`model.ParseUndecided(pr.Body)` が `(0, true)` で、`pr.UpdatedAt` が `pr.Comments` の末尾の
  `CreatedAt` より後」なら分岐に入らず判定表へ進む。`isC` / `isD` / `Issue` / `Card` と関数シグネチャは変えない。
  `go build ./...` が通ることを確認する
- [ ] 2.2 テストを `internal/classify/classify_test.go` の `TestInProgressRules` へ足す。spec delta で足した 6 Scenario
  （反映を終えた印があれば判定表へ流す / 人のコメントの後に PR が動いていなければ進行中のまま / 未確定が残る PR は印にならない /
  1 行目が未確定の判断で始まらない PR は印にならない / 印があり判定表のどの行にも当たらない PR はその他 /
  question PR は本文が未確定 0 件でも規則 3 のまま）のテストを足す。既存の
  「規則 2: 最新コメントが人のまま 3 時間動かなければ応答なしのその他」のテストに、人の最新コメントの `CreatedAt` を
  `UpdatedAt` と同時刻にする指定を足し、印が付かないまま時間切れになることを固定する。
  `go test ./internal/classify/...` が通ることを確認する
- [ ] 2.3 `internal/classify/fixture_test.go` の期待値表（`example` / `board`）が変わらないことを確認する。
  変える必要が出た場合は、その fixture の PR が規則 2 に当たるかを判定表から手で当て直してから書く（分類器の出力を写さない）

## 3. 仕上げ

- [ ] 3.1 `openspec validate s36-classify-reflected-pr --strict` が通ることを確認する
- [ ] 3.2 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
