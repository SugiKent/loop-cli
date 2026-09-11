# s30-other-grace

Refs #43

## Why

「その他」（局面 `other`）は今やるタブに出るのに、開いてみると routine の状態機械の途中（PR を作った直後でラベルが無い、checks が pending、`未確定の判断: N 件` が N > 0 のまま grill 中、`ai-assess:requested` が付く前）であることが多く、人が何をすべきか判断できない。
一方で `other` を進行中へ黙って落とすと、CI が赤のまま止まった routine PR や、外部から来て誰も触らない PR が今やるタブから消える（human-turn-signals.md「消えて見えなくなる項目を作らない」）。
「その他になってすぐ」は routine に任せ、「その他のまま一定時間誰も触っていない」だけを人に出す。猶予中も進行中タブには出るので、項目そのものは消えない。

## What Changes

- `classify.Card` が、open PR の分類結果が `other` で、かつその PR の `UpdatedAt` が取得時刻から猶予（既定 30 分）未満のものを `in-progress`（進行中タブ、優先度 7）に置き換える。要約は `PR #<n> はどの局面にも当たらない（更新から 30m は様子見）`
- issue の `other`（`question` と `blocked` があるのに `Comments` が nil）は猶予の対象にしない。この `other` は詳細取得の失敗で B（人待ち）を取りこぼしたもので、routine の途中ではない。隠す根拠が無い
- 更新から猶予以上経った `other` は今までどおり今やるタブに出る。今やるタブに現れた瞬間は、既存の差分通知（`addedNow`）が「増えた」と数えるので、デスクトップ通知が飛ぶ（追加実装なし）
- `Issue()` / `PR()` は変えない。判定表とフォールバックの意味（「どの行にも当たらない」）はそのままで、時間の判断は `Card()` だけが持つ
- `fetch.Fetch` が取得時刻と猶予を受け取り、`classify.Card` に渡す
- `config.yml` に `other_grace_min`（整数・分）を足す。省略時 30、`0` で猶予なし（今までの挙動）、負数はエラー。onboarding のフォームでは聞かず、書き出しもしない（`refresh_interval_sec` と同じく既定に任せる）
- `docs/domain/issue-driven-sdd/human-turn-signals.md` の「その他」バケットの段落に猶予の規則を足す。`docs/mvp/mvp.md` の設定ファイル例と README の設定表に `other_grace_min` を足す

経過時間の起点は主体の `UpdatedAt` にする。「その他になった時刻」を起点にするには「いつその他になったか」をローカルに記録する必要があるが、loop-cli は GitHub の状態だけで判定し、snapshot も前回結果の保存であって履歴ではない。`UpdatedAt` からの経過は「誰も触っていない時間」であり、測りたいゾンビ性に最も近い。routine や bot が触るたびに猶予が延びるので、bot が定期的に触る PR（自動 rebase される Dependabot PR など）は今やるタブに出ないことがある。その PR も進行中タブには出ており、消えるわけではない（design.md Risks）。

「stage ラベル付き PR で checks 待ちなら進行中」のような行の追加で `other` を減らす案は採らない。checks が赤のまま止まった PR を進行中に埋めるとゾンビになるので、結局は時間の条件が要る。一律の時間猶予だけで「一過性のその他を隠す」と「ゾンビを出す」の両方を満たす。

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `human-turn-classify`: `Card()` が取得時刻と猶予を受け取り、猶予内の open PR の `other` を `in-progress` に置き換える。優先度・タブ・種別・要約の表に猶予中の要約を足す
- `card-fetch`: `Fetch` が取得時刻と猶予を受け取って `classify.Card` に渡す
- `config-loading`: `Config` に `other_grace_min` を足す（既定 30、0 で無効、負数はエラー）
- `desktop-notify`: 猶予が明けて今やるタブに現れた `other` の Card は「増えた」と数える（既存の比較規則の帰結を Scenario で固定する）
- `onboarding`: `Marshal` は `other_grace_min` を書かず、`Load` の既定に任せる（mvp.md の例に 6 つ目のキーが入るので、「例と同じ形式」の意味を明示する）

## Impact

- `internal/classify/card.go`: `Card(c, mode, now, grace)` に猶予の置き換えを足す。`classify.go` は変えない
- `internal/fetch/fetch.go`: `Fetch(ctx, client, repos, modes, now, grace)`。`classify.Card` の呼び出しに `now` / `grace` を渡す
- `internal/config/config.go`: `Config.OtherGraceMin`、既定値、負数の検証
- `cmd/loop-cli/main.go`: `Fetcher` の閉包で `time.Now()` と `cfg.OtherGraceMin` 分を `fetch.Fetch` に渡す
- `internal/ui`: 変更なし（進行中タブの表示・`addedNow` はそのまま）
- `internal/onboarding`: コードは変更なし（`Marshal` は `other_grace_min` を書かず、`Load` の既定に任せる）。spec の文言だけ直す
- テスト: `card_test.go`（猶予内 / ちょうど猶予 / `grace` 0 / `UpdatedAt` ゼロ値 / issue は対象外）、`fetch_test.go`（`now` / `grace` が `Card` に届く）、`config_test.go`（既定・明示・負数）、`notify_test.go`（猶予明けで増えたと数える）。`Fetch` を呼ぶ既存テスト（`cmd/loop-cli/main_test.go`、`internal/ui/testdata_test.go`、`internal/snapshot/snapshot_test.go`）は引数を足すだけ。`classify_test.go` と fixture の期待値表は変えない
- docs: `human-turn-signals.md` の「その他」段落と変更履歴、`mvp.md` の設定ファイル例と変更履歴、`decisions.md` の D-005、`README.md` の設定表を更新する

## 先行 change との関係

`s26-close-issue-pr`（未 archive）は `card-detail` / `queue-screen` / `help-screen` / action 層を触り、この change が触る 5 つの spec とは重ならない。順序の制約は無い。
