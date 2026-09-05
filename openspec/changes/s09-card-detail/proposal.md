## Why

s08-queue-screen が終わると、キュー画面の選択行のプレビューで 1 行目・本文・コメントは読めるが、Issue と紐づく PR 群を 1 枚にまとめて読む画面が無く、`Enter` は何もしない。mvp.md「カード詳細（Enter）」が定めるヘッダ・紐づく PR 一覧・Issue 本文・最新 `blocked-by:` の要約・コメント時系列（routine コメントの折りたたみ）・PR 側の会話と review thread と checks を作り、主要フロー A / B の「何を聞かれているか」を読む場所を用意するのがこの change である。
この change は docs/mvp/implementation-tasks.md §3 (P1) の項目「カード詳細（Enter）: Glamour 表示、AI / 人のコメント区別（エスケープ済みマーカー・`## PR リスク評価` 見出しも AI 扱い）、紐づく PR 一覧」を実装する。AI / 人の判定は s05 の `model.IsAI`（`Comment.AI`）をそのまま使い、再実装しない。

## What Changes

- `internal/ui` の `Model` に画面の状態（キュー / カード詳細 / PR 詳細）を足す。キュー画面で `Enter` を押すと選択行の Card の詳細画面を開き、`Esc` で戻る
- カード詳細画面: 固定のヘッダ領域（リポジトリ、Issue 番号、`Card.Result.Summary`「いま人が何をすべきか」、現在の段階、バッジ blocked / wip / question、`depends on #m`、紐づく PR 一覧）と、スクロールする本文領域（Issue 本文を Glamour でレンダリング、最新 `blocked-by:` の要約（`human` なら `## Q1.` の見出しと選択肢・推奨）、コメント時系列）を描く
- 紐づく PR 一覧は s07 が段階順に並べた `Card.PRs` を `[propose] PR#131 open` の形で出し、各 PR の 1 行目判定（`model.ParseUndecided`）・ラベル・checks・merge 状態を添える。同段階の merge 済み PR が複数あれば s05 が立てた `Canonical` の PR に `（最新・正本）` を付ける。P1 では search が open PR しか返さないので merge 済み PR は出ず、印は s17 の cross-reference で merge 済み PR が入って初めて立つ
- routine（AI）コメントは既定で折りたたみ、`x` で展開 / 折りたたみを切り替える。AI コメントは s08 のプレビューと同じ左バー `▌` で区別する
- `Tab` で紐づく PR の選択を移し、`Enter` で PR 詳細画面（1 行目の判定結果、`Refs #n` / `Closes #n`（s07 の `fetch.LinkedIssue`）、PR 本文、会話コメント、review thread（未 resolve を先頭）、checks 状態）に入る。`g` でカード詳細（issue）と PR 詳細を相互に行き来する
- 詳細を開いても `gh` を追加で呼ばない。s07 の `Fetch` が入れた詳細だけを表示し、取っていない詳細（`Comments` / `MergeState` / `ReviewThreads` が nil）は `未取得` と出す。これは mvp.md のキーバインド表が `Enter` の内部処理として挙げる `gh issue view` と `gh pr view --json` を呼ばず、human-turn-signals.md「merge 可否は表示時に取り直す」も満たさないという逸脱なので、design.md「追加取得はしない」にこの逸脱と docs 更新の申し送りを記す。詳細を開いたときの 1 件再取得を s18 に含めるかは s18 が決める
- 段階の変遷タイムラインと「この段階に入ってからの経過時間」（データ源は REST timeline）は s17 が担当する。この change は枠を確保せず、s17 が ADDED で足す
- s08 `queue-screen` の 2 つの Requirement を MODIFIED する: キュー画面のキー処理（`Enter` を「何もしない」から外し、`g` / `x` / `Esc` の担当を s09 に移す。キューではいずれも何もしない）と、フッタのキーヒント（`Enter 開く` を足す）

## Capabilities

### New Capabilities
- `card-detail`: `internal/ui` のカード詳細画面と PR 詳細画面。画面の状態遷移（`Enter` / `Esc` / `Tab` / `g`）、ヘッダと紐づく PR 一覧、Issue 本文と `blocked-by:` 要約、コメントの折りたたみ（`x`）と AI の左バー、PR 詳細（1 行目判定・紐づけ・会話・review thread・checks）、未取得の表示、スクロールを定める

### Modified Capabilities
- `queue-screen`: Requirement「j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、他のキーは何もしない」を、キュー画面に限定したうえで `Enter`（詳細を開く）を「何もしない」から外し、`g` / `x` / `Esc` の担当を s09 に移す（キュー画面では何もしない）形に改める。Requirement「ヘッダはタブ名と件数と最終更新時刻、フッタはキーヒントとステータスを出す」を、キュー画面に限定したうえでフッタのヒントに `Enter 開く` を足す形に改める。他の Requirement は変えない

## Impact

- 新規: `internal/ui/detail.go`（詳細の状態・キー・描画）/ `internal/ui/detail_test.go`
- 変更: `internal/ui/model.go`（画面の状態と `Enter` / `Esc` / `Tab` / `x` / `g` の振り分け）/ `internal/ui/view.go`（画面の状態で `View` を切り替え、フッタのヒントに `Enter 開く` を足す）/ `internal/ui/preview.go`（s08 のコメントの行を作る関数に折りたたみの引数を足し、プレビューと詳細の両方から呼ぶ）/ `internal/ui/model_test.go` / `view_test.go`（`Enter` を「何も変えない」と見ていた検証を s09 の振る舞いに合わせ、`x` / `Esc` を「何も変えない」に足す）
- 依存: この change は `internal/model` から `Card` と `Issue` と `PR` の型、`Comment.AI`、`IssueStages` と `PRStages` と `HasLabel`、`ParseUndecided` と `LatestBlockedBy` と `ParseQuestions` を使う。`internal/classify` からは `ChecksGreen` を、`internal/fetch` からは `LinkedIssue` を使う（`internal/fetch` は s08 が `Fetcher` のために既に import している）。`internal/gh` からは `PRMergeState` と `ReviewThread` の型を読む。スクロールには Bubbles v2 の viewport を使う（Bubbles は s08 が依存に加えている）。新しい外部依存は無い
- 前提: s08-queue-screen が実装済みであること（`internal/ui` の `Model` / `New` / `Subject` / プレビュー / `fetchedMsg`）。s08 はレビュー中なので、MODIFIED で写した Requirement の本文は s08 の最終版に合わせる。archive の順は s08 → s09
- 後続 change への影響: s10 が `a`（詳細画面でも回答できるかは s10 が決める）、s12 が `o` / `R` / `?`、s13 が自動更新（詳細を開いている間の `fetchedMsg` の扱いは design.md の既定値を引き継ぐか s13 が変える）、s14 が merge ガードの理由表示、s16 が review thread の選択と `A` 返信、s17 がタイムライン・段階の経過時間・cross-reference による merge 済み PR の取り込み（`Canonical` の印がここで立つ）を、この change の画面にキーと行を足す形で担当する
