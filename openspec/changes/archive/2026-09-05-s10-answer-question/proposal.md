## Why

s09-card-detail が終わると、キュー画面と詳細画面で「何を聞かれているか」（`## Q1.` の見出しと選択肢、issue の `blocked-by: human`）は読めるが、答える手段が無い。主要フロー A（PR の `question` に答える）と B（issue の `question` に方針を答える）は「`question` が付いたものにコメントで答える」で同じ実装であり、書き先が PR か issue かだけが違う（implementation-tasks.md）。`a` で `$EDITOR` を開き、回答テンプレートを事前入力し、投稿前に `blocked-by:` 行を検出して警告し、`gh pr comment` / `gh issue comment` で投稿する経路を作るのがこの change である。
この change は docs/mvp/implementation-tasks.md §3 (P1) の項目「**フロー A / B** `a` 回答: PR の `question` は `gh pr comment`、issue の `question` は `gh issue comment`。`## Q1.` と選択肢をパースして `Q1: A` テンプレートを事前入力、投稿前に `blocked-by:` 行を検出して警告。ラベルは触らない」を実装する。この change は sugi-loop で最初の**書き込み系**の change である。D-003 が定めた `internal/action`（ラベル・コメント・merge の書き込みを担い、不変条件を持つ層）を、この change が導入する。

## What Changes

- `internal/action` を新設する。human-turn-signals.md「人が書き込むときの不変条件」のうちコメント投稿に関わる 3 つを、それぞれ 1 つの Requirement として持つ: 4（書き先 3 種類を混同しない。主体が PR なら `CommentPR`、issue なら `CommentIssue`）、7（`<!-- routine -->` を書かない。含む本文は投稿を拒否する）、8（`blocked-by:` で始まる行を含めない。投稿前に検出して UI が警告する）。ラベルは一切触らない（不変条件 2）。空の本文は投稿しない
- `internal/action` に回答テンプレートの組み立てを置く: 対象の最新の routine コメントを s05 の `model.ParseQuestions` でパースし、`（推奨）` の選択肢を既定値にした `Q1: A\nQ2: B` を作る。見出しの無いコメント（issue の `blocked-by: human` は書式が固定でない）はパースできた分だけ、1 件も無ければ空
- `internal/ui` に `a` のフローを足す: キュー画面（選択行の主体）/ カード詳細画面（Issue）/ PR 詳細画面（その PR）で `a` を押すと、対象と回答テンプレートを決めて `$EDITOR`（config の `editor`）を外部プロセスとして開く。編集結果が空なら投稿しない。`blocked-by:` 行があれば確認画面（投稿 / 編集に戻る / 中止）に移る。`<!-- routine -->` を含めば投稿の選択肢の無い確認画面（編集に戻る / 中止）に移る。問題が無ければ s03 の `CommentPR` / `CommentIssue` で投稿し、フッタのステータスに結果を出す
- エディタ起動は差し替え可能にする（`Editor` 型。実機用の `ExternalEditor(command)` と、テストで固定文字列を返すスタブ）。投稿の呼び出しは s03 の `Fake` の `Calls` で引数（repo / number / body）と「ラベルの呼び出しが無いこと」を検証する
- `ui.New` の引数に `gh.GHClient` と `Editor` を足す（s08 の `New(fetcher)` は書き込みの手段を持たない）。`cmd/sugi-loop/main.go` が `client` と `ui.ExternalEditor(cfg.Editor)` を渡す
- キュー画面のフッタのヒントと詳細画面のフッタのヒントに `a 回答` を足す
- 書き込み後の対象 1 件再取得（D-002。s18 の担当）はしない。投稿後は何も再取得せず、s12 の `R` か s13 の自動更新で反映する

## Capabilities

### New Capabilities
- `answer-action`: `internal/action` のコメント投稿。書き先の決定（不変条件 4）、`<!-- routine -->` の拒否（7）、`blocked-by:` 行の検出（8）、空本文の拒否、回答テンプレートの組み立てを定める。ラベルは触らない（2）
- `answer-question`: `internal/ui` の `a` のフロー。対象の決定、エディタの起動と差し替え、編集結果の検査（空 / マーカー / `blocked-by:`）、確認画面のキー、投稿とステータス表示を定める

### Modified Capabilities
- `queue-screen`: Requirement「Model は Card をタブ別に並べ、選択行を 1 つ持つ」の `New` の引数に `gh.GHClient` と `Editor` を足す。Requirement「j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、他のキーは何もしない」（s09 が MODIFIED した最新版）の `a` を「何もしない」から外す。Requirement「ヘッダはタブ名と件数と最終更新時刻、フッタはキーヒントとステータスを出す」（同）のフッタに `a 回答` を足す
- `card-detail`: Requirement「Enter でカード詳細を開き、Esc で 1 つ前の画面に戻る」の画面の状態を 3 つから 4 つ（確認画面を追加）に改める。Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」のフッタのヒントに `a 回答` を足す（カード詳細・PR 詳細とも）
- `tui-entrypoint`: Requirement「sugi-loop バイナリが起動して 1 フレーム描画する」（s08 が MODIFIED した最新版）の起動手順 3 で `client` と `ui.ExternalEditor(Config.Editor)` を `ui.New` に渡す形に改める

## Impact

- この change が新しく作るファイルは次のとおり。`internal/action/action.go` は `Target` と `Comment` と `BlockedByLines` と routine マーカー判定関数 `HasRoutineMarker(body string) bool` とエラー値を持つ。`internal/action/template.go` は `AnswerTemplate` を持つ。`internal/action/action_test.go` と `template_test.go` はそのテストである。`internal/ui/editor.go` は `Editor` 型と `ExternalEditor` と編集完了メッセージを持ち、`internal/ui/editor_test.go` はそのテストである。`internal/ui/answer.go` は対象の決定、`a` のキー処理、編集結果の検査、確認画面、投稿コマンドと結果メッセージ、確認画面の描画を持つ。`internal/ui/answer_test.go` はそのテストである
- 変更: `internal/ui/model.go`（`New` の引数、`client` / `editor` / 回答の状態の保持、`a` と確認画面のキーの振り分け）/ `internal/ui/view.go`（キュー・詳細のフッタに `a 回答`、確認画面の `View`）/ `cmd/sugi-loop/main.go`（`ui.New` に `client` と `ui.ExternalEditor(cfg.Editor)` を渡す）/ `internal/ui/model_test.go` / `view_test.go` / `detail_test.go`（`a` を「何も変えない」と見ていた検証と `New` の呼び出しを直す）
- 依存: `internal/gh`（`GHClient` の `CommentPR` / `CommentIssue`、`Fake.Calls`。**interface / Client / Fake は変えない**。s03 が両メソッドと記録を定義済み）、`internal/model`（`Comment.AI` / `ParseQuestions` / `Issue` / `PR`）、`internal/config`（`Config.Editor`。空のときの扱いは s02 が s10 に委ねている）。Bubble Tea の外部プロセス実行（既存依存。新しい外部依存は無い）
- 前提: s08-queue-screen と s09-card-detail が実装済みであること（`internal/ui` の `Model` / `New` / `Subject` / 画面の状態 / 詳細の対象）。s08 / s09 はレビュー中なので、MODIFIED で写した Requirement の本文は両者の最終版に合わせる。archive の順は s08 → s09 → s10
- 後続 change への影響: s11 の `t` と s14 の `m` と s15 の `n` / `s` は、この change が `ui.New` に足した `client` と `internal/action` を再利用して書き込みの Requirement を ADDED で足す。s13 は `New` の引数をさらに拡張する。s16 の `A`（review thread 返信）は `ReplyReviewThread` を使う別の書き先（不変条件 4 の 3 種類目）で、この change の `Comment` には足さない。s18 が書き込み後の 1 件再取得を足す
- 検証: validation-plan.md「TUI からの書き込みを検証する手段は未定」のとおり、live での投稿確認は tasks に入れない（design.md の未決事項）。手動確認は「エディタが開いてテンプレートが入っている → 本文を空にして閉じる → 何も投稿されない」の範囲に限る
