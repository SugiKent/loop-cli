## Why

s10-answer-question を終えた時点では、人の役割 3 つ（`stage:todo` を付ける / `question` に答える / merge する）のうち「答える」だけが TUI からできる。主要フロー E（着手を承認する）は、バックログタブに並んだ issue（局面 E: open issue、`stage:*` 無し、`blocked` 無し）に `stage:todo` を付けて dispatcher を起動させる操作であり、上流が人に許した唯一のラベル操作である。この change は docs/mvp/implementation-tasks.md §3 (P1) の項目「**フロー E**: `t` で stage:todo を付ける / 外す（1 操作 1 ラベルで呼ぶ）」を実装する。

## What Changes

- `internal/action` に `stage:todo` の切り替え `ToggleTodo` を足す。human-turn-signals.md「人が書き込むときの不変条件」のうちラベルに関わる 2 つを Requirement として持つ: 1（1 操作 1 ラベル。`AddLabel` / `RemoveLabel` を `stage:todo` 1 つだけで 1 回呼ぶ。ラベル集合の置換をしない）、2（TUI が書くラベルは `stage:todo` と `s` の `stage:propose` の 2 つに限る。この change は `stage:todo` しか書かない）
- `ToggleTodo` は判定の前に `client.ViewIssue` で現在のラベルを読み直す。画面の Card のラベルは最後の取得時点のもので、`t` を続けて 2 回押したときに 2 回目が「もう一度付ける」になってしまう（mvp.md「付け直しは dispatcher への『もう一度評価しろ』の合図」は付いていないものを付ける操作であり、付いているものを外すのが取り消し）。読み直した結果で `stage:todo` が付いていれば外し、付いていなければ付ける。別の段階ラベルが付いている・段階ラベルが 2 つ以上ある・`blocked` が付いている issue は拒否して理由を返す（docs に記述が無い。design.md 未決事項）
- `internal/ui` に `t` を足す: キュー画面（選択行の Card の `Issue`）と カード詳細画面（詳細の対象の `Issue`）で `t` を押すと、確認なしで `ToggleTodo` をコマンドとして実行し、結果をフッタのステータスに出す。`Issue` が nil の PR 単独カードでは何もしない。PR 詳細画面では `t` は何もしない
- 書き込み中（この change のラベル書き込み、s10 のコメント投稿）は `t` と `a` を受け付けない
- 書き込み後の再取得はしない（s10 と同じ。対象 1 件の再取得は s18、全件は s12 の `R` と s13 の自動更新）。画面の Card のラベルも書き換えない（`ToggleTodo` が毎回読み直すので、取り消しの `t` は画面の状態に依存しない）
- キュー画面とカード詳細画面のフッタのヒントに `t todo` を足す

## Capabilities

### New Capabilities
- `todo-action`: `internal/action` の `stage:todo` 切り替え。ラベルの読み直し、付ける / 外すの判定、拒否の条件、1 操作 1 ラベル（不変条件 1）、書くラベルは `stage:todo` だけ（不変条件 2）を定める
- `todo-toggle`: `internal/ui` の `t` のフロー。対象の決定（画面の Card の `Issue`）、確認なしの実行、書き込み中の排他、結果のステータス表示、再取得しないことを定める

### Modified Capabilities
- `queue-screen`: Requirement「j / k / ↑ / ↓ で行を移動し、1–4 / Tab でタブを切り替え、他のキーは何もしない」（s10 が MODIFIED した最新版）の `t` を「何もしない」から外す。Requirement「ヘッダはタブ名と件数と最終更新時刻、フッタはキーヒントとステータスを出す」（同）のフッタに `t todo` を足す
- `card-detail`: Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」（s10 が MODIFIED した最新版）のカード詳細のフッタのヒントに `t todo` を足す（PR 詳細には足さない）
- `answer-question`: Requirement「a は画面の対象を決めて回答テンプレートを入れたエディタを開く」の「前の投稿の結果を待っている間は `a` を押しても何もしない」を「書き込み（投稿・ラベル切り替え）の結果を待っている間は `a` / `t` を無視」に、Requirement「投稿の結果をステータスに出し、再取得しない」の投稿中の `a` とステータスの消滅条件を「`a` / `t`」「次に `t` か `a` を押したとき」に改める（s10 の ADDED を MODIFIED で写す）

## Impact

- この change が新しく作るファイルは次のとおり。`internal/action/todo.go` は `ToggleTodo` とエラー値を持つ。`internal/action/todo_test.go` はそのテストである。`internal/action/testdata/todo/issue-<n>.json` はラベルの組み合わせ別の `gh issue view` fixture である。`internal/ui/todo.go` は `t` のキー処理とコマンドと結果メッセージを持つ。`internal/ui/todo_test.go` はそのテストである
- 変更: `internal/ui/model.go`（`t` の振り分け、書き込み中フラグの共有）/ `internal/ui/view.go`（キュー画面とカード詳細のフッタに `t todo`）/ `internal/ui/model_test.go` / `view_test.go` / `detail_test.go`（`t` を「何も変えない」と見ていた検証とフッタの検証を直す）
- 依存: `internal/gh`（`GHClient` の `ViewIssue` / `AddLabel` / `RemoveLabel`、`Fake.Calls`。**interface / Client / Fake は変えない**。s03 が 3 メソッドと記録を定義済み）、`internal/model`（`LabelStageTodo` / `LabelBlocked` / `IssueStages` / `HasLabel`）、s10 の `ui.New(fetcher, client, editor)` の `client` とフッタ右のステータス。`New` の署名と `cmd/sugi-loop/main.go` は変えない
- 前提: s10-answer-question が実装済みであること（`internal/action` パッケージ、`New` の `client`、フッタ右のステータスと書き込み中フラグ）。archive の順は s08 → s09 → s10 → s11
- 後続 change への影響: s15 の `s`（`stage:todo` を外し → `ViewIssue` で読み直し → `stage:propose` を付ける）は、この change の `ToggleTodo` を再利用せず、不変条件 3 の手順を自分で持つ（`ToggleTodo` は 1 操作 1 ラベルで完結する関数であり、2 ラベルの手順を混ぜない）。s18 が書き込み後の 1 件再取得を足す
- 検証: validation-plan.md「TUI からの書き込みを検証する手段は未定」のとおり、live でのラベル付与確認は tasks に入れない（design.md 未決事項）。`Fake.Calls` で「`ViewIssue` → `AddLabel` / `RemoveLabel` が 1 回・`stage:todo` だけ・他のラベル無し」を検証する
