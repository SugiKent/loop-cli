# tasks: s30-other-grace

Refs #43

## 1. ドメイン文書（分類器の正本を先に直す）

- [ ] 1.1 `docs/domain/issue-driven-sdd/human-turn-signals.md` の「その他」バケットの段落（判定表の直後）に、猶予の規則を足す。sdd の open PR で `UpdatedAt` から `other_grace_min` 分（既定 30）未満の `other` は進行中タブに出し、以上なら今やるタブに出す。`0` で猶予なし。label の PR と issue には当てない。猶予の判断は `Card()` だけが持ち、判定表の行とフォールバックの意味は変えない。変更履歴に 1 行足す（design.md D6）。`docs/mvp` は触らない（CLAUDE.md）

## 2. 設定

- [ ] 2.1 `internal/config/config.go` の `Config` に `OtherGraceMin int`（yaml `other_grace_min`）を足し、`Load` の既定値に 30 を入れ、`validate` で 0 未満を `other_grace_min` を含むエラーにする（design.md D4）
- [ ] 2.2 `internal/config/config_test.go` に Scenario「repos だけのファイル」（`OtherGraceMin` 30）、「other_grace_min を 0 と明示する」、「明示した値が既定値を上書きする」（`other_grace_min: 5`）、「other_grace_min が負数」のテストを足し、既存の mvp.md の例のテストに `other_grace_min: 30` を含める

## 3. 分類

- [ ] 3.1 `internal/classify/card.go` の `Card` を `Card(c model.Card, mode model.Mode, now time.Time, grace time.Duration) model.Card` にし、`mode` が `sdd` のとき、`PR()` で埋めた open PR の `Result` に対して、`other` かつ `grace > 0` かつ `UpdatedAt` がゼロ値以外かつ `now.Sub(UpdatedAt) < grace` なら `in-progress` に置き換える（design.md D1 / D3。要約は `PR #<n> はどの局面にも当たらない（更新から <M>m は様子見）`、`<M>` は `int(grace.Minutes())`）。`Issue.Result` と `label` 方式の PR は触らない。置き換えは `Card.Result` を決める前に行う。`classify.go` は変えない
- [ ] 3.2 `internal/classify` の既存の `Card` 呼び出し（`card_test.go`、`fixture_test.go`、`label_test.go` 等）に `0` を足し、Requirement「Card は猶予内のその他の PR を進行中に置き換える」の 8 Scenario（猶予未満 / ちょうど猶予 / `grace` 0 / `UpdatedAt` ゼロ値 / label の PR は対象外 / `question` + `blocked` の issue は対象外 / 他の局面を隠さない / `other` 以外は対象外）と、Requirement「Card は…最上位の局面を 1 行目に出す」の Scenario「wip の issue と猶予中の PR のカードは先頭 open PR の要約で進行中」のテストを足す。`classify_test.go` と fixture の期待値表は変えない

## 4. 取得

- [ ] 4.1 `internal/fetch/fetch.go` の `Fetch` を `Fetch(ctx, client, repos, now time.Time, grace time.Duration)` にし、`buildCards` 経由で `classify.Card` の呼び出しに `grace` を渡す（design.md D2）
- [ ] 4.2 `internal/fetch/fetch_test.go` と `stale_test.go` の既存の `Fetch` 呼び出しに `0` を足し、Scenario「grace が分類に渡る」（`link` fixture の PR 61、`now` `2026-09-04T10:10:00Z`・`grace` 30 分で `in-progress` と猶予の要約）と「猶予以上経った now ではその他に戻る」（`now` `2026-09-04T10:31:00Z` で `other`）のテストを足す。同じ fixture の PR 62 / 132 も `other` で `updatedAt` が同じなので、既存の期待値表のテストとは別の `Fetch` 呼び出しにする

## 5. 起動の配線

- [ ] 5.1 `cmd/loop-cli/main.go` の `Fetcher` の閉包で `fetch.Fetch(ctx, client, repos, time.Now(), time.Duration(cfg.OtherGraceMin)*time.Minute)` を呼ぶ（本番コードの `fetch.Fetch` の呼び出しはここ 1 か所だけ）
- [ ] 5.2 `fetch.Fetch` を呼ぶ既存テスト `cmd/loop-cli/main_test.go`、`internal/ui/testdata_test.go`、`internal/snapshot/snapshot_test.go` の呼び出しに `0` を足し、3 パッケージがビルドできることを確認する

## 6. 通知

- [ ] 6.1 `internal/ui/notify_test.go` に Scenario「猶予が明けて今やるに現れたその他は増えたと数える」（`prev` が PR 61 単独の `in-progress` Card、`next` が同じ PR 61 の `other` Card で `addedNow` が 1 枚返す）のテストを足す。`notify.go` は変えない（design.md D5）

## 7. README

- [ ] 7.1 `README.md` の設定ファイルの例と設定表に `other_grace_min`（その他の PR を進行中に置く猶予（分）、既定 `30`、`0` 以上の整数。`0` で無効。label 方式の PR には当たらない）を足す

## 8. 仕上げ

- [ ] 8.1 `openspec validate s30-other-grace --strict` が通ることを確認する
- [ ] 8.2 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
