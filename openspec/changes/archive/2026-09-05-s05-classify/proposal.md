## Why

sugi-loop の価値の本体は「人の出番だけ」を GitHub の状態から判定する分類器であり、V-1（validation-plan.md）は UI より先にこれを fixture でテストすることを求めている。
s03-gh-client の生の型と `Fake`、s04-fixture-capture の fixture は揃ったが、human-turn-signals.md の判定表（局面 A〜G・「進行中」・「その他」・AI/人コメント判定）を実装したコードはまだ無く、`internal/model`（Card / Comment / 分類結果）も無い。
この change は docs/mvp/implementation-tasks.md §2「`internal/classify`: 局面 A〜G と「その他」バケットを純粋関数で実装し、fixture でテストする」に対応し、共通ルールどおり `internal/model` もここで導入する。

## What Changes

- `internal/model` を新設する。`Issue` / `PR` / `Comment`（AI/人の判定付き）/ `Card`（Issue 1 件 + 紐づく PR 群 + 分類結果）/ `Result`（局面・優先度・タブ・「いま人が何をすべきか」の 1 行）/ `Situation` / `Tab` の型と、s03 の生の型からの変換関数（`IssueFromSearch` / `PRFromSearch` / `CommentFrom`）を置く
- `internal/model` に本文パーサを置く。AI コメント判定 `IsAI`（`<!-- routine -->` 始まり、HTML エスケープ済み `&lt;!-- routine --&gt;` 始まり、`## PR リスク評価` 見出し）、PR 本文 1 行目の `未確定の判断: N 件` パース、`blocked-by:` 行を含む最新コメントの検出、`## Q1.` 見出しと `選択肢 A（推奨）: …` のパース（s10 の `a` 事前入力用）
- `internal/classify` を新設する。Issue 単体 / PR 単体の純粋関数と、Card 単位で最上位の局面を 1 行目に出す集約の 2 層。判定表 A〜G を表の順に評価して最初に当たった行で止め、「キューに入れないもの」を進行中に、どの行にも当たらない open PR を「その他」バケットに入れる。4 タブ（今やる = A/B/C/D/G/その他、バックログ = E、異常 = F、進行中）への振り分けと優先度（F=0, A=1, D=1, B=2, C=3, E=4, G=5, その他=6）もここで定義する
- 「同段階の merge 済み PR が複数あれば最新が正本。古い PR の `question` を異常扱いしない」を Card 集約の Requirement にする
- テストは s03 の `Fake` で `internal/gh/testdata/fixtures/<alias>/` を読み、fixture ごとの期待値表（issue / PR 番号 → 局面）と突き合わせる。期待値表は全 issue / 全 PR を網羅していなければ失敗する。s04 の実採取後に `<alias>` の期待値を埋めるタスクを含める
- Issue と PR の紐づけ（PR title `[<段階>] #<n>` / 本文 `Refs #n` `Closes #n` のパース、search 結果からの Card 組み立て、D-001 の遅延取得）は s07 が担当する。この change は「組み立て済みの `Card` と、詳細取得済みの `Issue` / `PR` を受け取る」前提で書く

## Capabilities

### New Capabilities
- `card-model`: `internal/model` が Issue / PR / Comment / Card / Result / Situation / Tab の型を定義し、s03 の生の型を変換し、AI コメントを判定し、`未確定の判断` / `blocked-by:` / `## Q1.` をパースする
- `human-turn-classify`: `internal/classify` が局面 A〜G・進行中・その他を評価順どおりに判定し、優先度・タブ・1 行要約を決め、Card を集約して merge 済み PR の正本を決め、fixture と期待値表でテストする（V-1）

### Modified Capabilities
（無し。`gh-client` / `gh-fake` / `fixture-capture` / `dev-cli` / `config-loading` は変更しない）

## Impact

- 新規: `internal/model/model.go` / `parse.go` / `parse_test.go` / `model_test.go`、`internal/classify/classify.go` / `card.go` / `classify_test.go` / `card_test.go` / `fixture_test.go`
- 依存: `internal/model` は `internal/gh` の型（`gh.SearchIssue` / `gh.SearchPR` / `gh.Comment` / `gh.PRMergeState` / `gh.ReviewThread`）を import する。`internal/classify` は `internal/model` と `internal/gh` を import する。逆方向の依存は無い。外部依存は追加しない（標準ライブラリの `regexp` / `strings` のみ）
- fixture: `internal/gh/testdata/fixtures/*/` を読み取り専用で使う。fixture は変更しない。期待値表は `internal/classify/classify_test.go` 内の Go のマップとして持つ
- 検証計画との対応: V-1 をこの change が満たす。書き込み系の検証（validation-plan.md「未定」）は扱わない
- 後続 change への影響: s06 の `classify` サブコマンドは `classify.Issue` / `classify.PR` の結果を issue / PR 単位でプレーンテキストに出す（Card 化は s07 の紐づけ後）。s07 は `model.IssueFromSearch` / `model.PRFromSearch` / `model.CommentFrom` で `Card` を組み立て、D-001 の遅延取得結果を `Issue.Comments` / `PR.Comments` / `PR.MergeState` / `PR.ReviewThreads` に入れてから `classify.Card` を呼ぶ。s08 は `Result.Tab` / `Result.Priority` / `Result.Summary` で並べる。s09 は `model.LatestBlockedBy` と `Comment.AI` を表示に使う。s10 は `model.ParseQuestions` で回答テンプレートを作る。s14 は `classify.ChecksGreen` と `model.ParseUndecided` を merge ガードに使う。s17 は cross-reference で得た merge 済み PR を `PR.State` 付きで `Card.PRs` に加える
