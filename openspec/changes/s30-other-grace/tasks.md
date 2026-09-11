# tasks: s30-other-grace

Refs #43

## 1. ドメイン文書（分類器の正本を先に直す）

proposal / design が挙げていた `docs/mvp/mvp.md`（設定ファイルの例）と `docs/mvp/decisions.md`（D-005）の更新は行わない。
`CLAUDE.md`「docs/mvp はこれ以降更新しない」が「設定ファイルが変わっても `docs/mvp` に書き戻さず、変更履歴の行も足さない」と
定めており、仕様の正本は `openspec/specs/`、利用者向けの説明は `README.md` で保つと決まっているため（この判断は PR 本文にも書く）。

- [x] 1.1 `docs/domain/issue-driven-sdd/human-turn-signals.md` の「その他」バケットの段落（判定表の直後）に、猶予の規則を足す。主体の `UpdatedAt` から `other_grace_min` 分（既定 30）未満の `other` は進行中タブに出し、超えたら今やるタブに出す。`0` で猶予なし。時間の判断は `Card()` だけが持ち、判定表の行とフォールバックの意味は変えない。変更履歴に 1 行足す（design.md D6）

## 2. 設定

- [x] 2.1 `internal/config/config.go` の `Config` に `OtherGraceMin int`（yaml `other_grace_min`）を足し、`Load` の既定値に 30 を入れ、`validate` で 0 未満を `other_grace_min` を含むエラーにする（design.md D4）
- [x] 2.2 `internal/config/config_test.go` に Scenario「repos だけのファイル」（`OtherGraceMin` 30）、「other_grace_min を 0 と明示する」、「明示した値が既定値を上書きする」（`other_grace_min: 5`）、「other_grace_min が負数」のテストを足し、既存の mvp.md の例のテスト（例は 5 キーのまま）に `OtherGraceMin` が既定の 30 になる検証を足す

## 3. 分類

- [x] 3.1 `internal/classify/card.go` の `Card` を `Card(c model.Card, mode model.Mode, now time.Time, grace time.Duration) model.Card` にし、`PR()` で埋めた open PR の `Result` に対して、`other` かつ `grace > 0` かつ `UpdatedAt` がゼロ値以外かつ `now.Sub(UpdatedAt) < grace` なら `in-progress` に置き換える（design.md D1 / D3。要約は `PR #<n> はどの局面にも当たらない（更新から <M>m は様子見）`、`<M>` は `int(grace.Minutes())`）。`Issue.Result` は触らない。置き換えは `Card.Result` を決める前に行う。`classify.go` は変えない
- [x] 3.2 `internal/classify/card_test.go` の既存の `Card` 呼び出しに `time.Time{}` と `0` を足し、Requirement「Card は猶予内のその他の PR を進行中に置き換える」の 7 Scenario（猶予未満 / ちょうど猶予 / `grace` 0 / `UpdatedAt` ゼロ値 / `question` + `blocked` の issue は対象外 / 他の局面を隠さない / `other` 以外は対象外）と、Requirement「Card は…最上位の局面を 1 行目に出す」の Scenario「wip の issue と猶予中の PR のカードは先頭 open PR の要約で進行中」のテストを足す。`classify_test.go` と fixture の期待値表は変えない

## 4. 取得

- [x] 4.1 `internal/fetch/fetch.go` の `Fetch` を `Fetch(ctx, client, repos, now time.Time, grace time.Duration)` にし、`classify.Card` の呼び出しに `now` / `grace` を渡す。`Fetch` は `time.Now()` を読まない（design.md D2。`modes` は s30-mode-from-labels で引数から消え、`now` は既にあるので、足すのは `grace` だけ）
- [x] 4.2 `internal/fetch/fetch_test.go` の既存の `Fetch` 呼び出しに `0` を足し、Scenario「grace が分類に渡る」（PR 61 の `updatedAt` の 10 分後・`grace` 30 分で `in-progress` と猶予の要約）と「猶予以上経った now ではその他に戻る」（31 分後で `other`）のテストを足す。PR 61 の fixture の `updatedAt`（`internal/fetch/testdata/link/search-prs.json`）を読んで `now` を作る。同じ fixture の PR 62 / 132 も `other` で `updatedAt` が同じなので、既存の期待値表のテストとは別の `Fetch` 呼び出しにする

## 5. 起動の配線

- [x] 5.1 `cmd/loop-cli/main.go` の `Fetcher` の閉包で `fetch.Fetch(ctx, client, repos, time.Now(), time.Duration(cfg.OtherGraceMin)*time.Minute)` を呼ぶ（本番コードの `fetch.Fetch` の呼び出しはここ 1 か所だけ）
- [x] 5.2 `fetch.Fetch` を呼ぶ既存テスト `cmd/loop-cli/main_test.go`、`internal/ui/testdata_test.go`、`internal/snapshot/snapshot_test.go` の呼び出しに `0` を足し、3 パッケージがビルドできることを確認する

## 6. 通知

- [x] 6.1 `internal/ui/notify_test.go` に Scenario「猶予が明けて今やるに現れたその他は増えたと数える」（`prev` が PR 61 単独の `in-progress` Card、`next` が同じ PR 61 の `other` Card で `addedNow` が 1 枚返す）のテストを足す。`notify.go` は変えない（design.md D5）

## 7. README と onboarding spec

- [x] 7.1 `README.md` の設定ファイルの例と設定表に `other_grace_min`（その他の PR を進行中に置く猶予（分）、既定 `30`、`0` 以上の整数。`0` で無効）を足す
- [x] 7.2 `internal/onboarding/onboarding_test.go` の `config.Load` 往復のテストに `OtherGraceMin` が 30 になる検証を足す（`Marshal` の出力は変えない）

## 8. 仕上げ

- [x] 8.1 `openspec validate s30-other-grace --strict` が通ることを確認する
- [x] 8.2 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
