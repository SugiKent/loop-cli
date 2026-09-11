## 1. close の呼び出し（internal/gh）

- [x] 1.0 `origin/main` の `openspec/specs/` と `openspec/changes/`（そのときに残っている未 archive の change の delta）を
      読み、この change が MODIFIED / REMOVED している Requirement のうち他の change も触っているものを洗い出し、
      delta を最新の版から写し直す（proposal の Impact の一覧は 2026-09-10 09:15 時点の断面。`help-screen` /
      `queue-screen` 2 本 / `card-detail` 2 本 / `todo-toggle` / `gh-fake` が既知）。他の change の追記（`n` の行、
      タイトル行の折り返し、`ListLabels` の記録など）を消していないことを、`openspec validate --strict` の後に diff で確かめる

- [x] 1.1 `internal/gh/gh.go` の `GHClient` に `CloseIssue(ctx, repo, number)` と `ClosePR(ctx, repo, number)` を足す
- [x] 1.2 `internal/gh/client.go` に `Client.CloseIssue` / `Client.ClosePR` を実装する（引数は `issue close <n> -R <repo>` と
      `pr close <n> -R <repo>`、標準出力は読み捨て、失敗は既存の `gh` エラーに載せる）
- [x] 1.3 `internal/gh/fake.go` の `Fake.CloseIssue` / `Fake.ClosePR` で呼び出しを `Calls` に記録する
- [x] 1.4 実行する引数と失敗時のエラー文字列のテストを `internal/gh/client_test.go` に書き、`Fake` が呼び出しを記録することの
      テストを `internal/gh/fake_test.go` に書く

## 2. close の判定と実行（internal/action）

- [x] 2.1 `internal/action/close.go` を作り、`action.Target` の `IsPR` で `ClosePR` / `CloseIssue` を呼び分ける close 関数と、
      `propose` / `apply` ラベルの PR に却下の注意を返す判定関数を置く（拒否は持たない。理由は design.md）
- [x] 2.2 テストを `internal/action/close_test.go` に書く。検証するのは書き先の分かれ方（`Calls` が `ClosePR` か
      `CloseIssue` のちょうど 1 件）と、`gh` の失敗がそのまま返ること、ラベルもコメントも書かないこと
- [x] 2.3 注意の判定のテストを同じファイルに書く（`propose` / `apply` の PR で 1 件、`archive` / `docs` / ラベル無しの PR と
      issue では空）

## 3. c と確認画面（internal/ui）

- [x] 3.1 `internal/ui/close.go` を作り、close の状態（リポジトリ・番号・種別・表示名・タイトル・ラベル・注意・戻り先）と、
      画面ごとに対象を決める処理と、`c` のハンドラを置く（書き込み中は何もしない。押下時にステータスを空にする。`gh` は呼ばない）
- [x] 3.2 `Model` の画面の状態に close の確認画面を足し、`internal/ui/model.go` のキー振り分けに確認画面の分岐を merge の
      確認画面の分岐の直後（`a` / `t` / `m` / `o` / `?` / `u` より先）、`c` のハンドラを `m` のハンドラの直後に置く
- [x] 3.3 確認画面のキー（`y` は close のコマンドを返して戻り先へ、`Esc` は中止してステータスを出す、他のキーは何もしない。
      どちらも戻り先が詳細画面なら本文領域を作り直す）と、close のコマンドと結果のメッセージ、`m.writing` による二重防止を実装する
- [x] 3.4 確認画面の描画を実装する。出す行は `close の確認: <表示名>  <タイトル>` と `種別:` と `labels:` と `注意:` で、
      幅で切り詰め、フッタ左は `y close  Esc 中止  q 終了` とする。`internal/ui/view.go` の画面分岐にも 1 行足す
- [x] 3.5 `internal/ui/help.go` の `helpKeys` に `c  issue / PR を close する（確認あり）` の行を `m` の直後で足す
- [x] 3.6 フッタのヒントに `c close` を足す。`internal/ui/view.go` のキュー画面のヒント（`m merge` の次）と
      `internal/ui/detail.go` の `detailHint` の 3 分岐（PR 詳細と PR ありカード詳細は `m merge` の次、PR なしカード詳細は
      `m merge` が無いので `L ラベル` の次。s28 が `t todo` と `m merge` の間に `L ラベル` を入れたので、3 画面で
      `c close` の位置がそろう側を採った）を直す

## 4. テスト（internal/ui）

- [x] 4.1 画面遷移のテストを書く（3 画面で `c` が確認画面を開き対象が画面どおりに決まる・0 行のタブとヘルプ画面と各確認画面では
      何もしない・`Esc` で戻り先に戻る・確認中に取得が完了しても対象が変わらない・`c` の押下で直前のステータスが消える）
- [x] 4.2 確認画面の描画のテストを書く（issue と PR の行の形・`labels: なし`・`propose` PR で `注意:` の行が出て issue では
      出ない・確認画面で `a` / `t` / `m` / `c` / `o` / `?` / `u` が何もしない）
- [x] 4.3 close の実行のテストを書く（`Fake.Calls` の `CloseIssue` / `ClosePR` がちょうど 1 件・成功と失敗のフッタ・
      close 中の `t` / `a` / `m` / `c` が効かず `o` は効く・`Cards` が変わらない・ラベルとコメントを書かない）
- [x] 4.4 確認画面でリサイズしてから `Esc` で詳細に戻ると本文領域が新しい高さになることのテストを書く
- [x] 4.5 既存テストのうちヘルプ画面のキーの行数を見ているものを新しい値（19 行。`L` と `n` が入った後の値）に直す
- [x] 4.6 フッタのヒントを見ている既存テストを新しい期待値に直す（`L` と `n` が入った後の値で、キュー 99 列 /
      カード詳細 144 列 / `PRs` 空 96 列 / PR 詳細 109 列。幅を指定している既存の Scenario は、幅 100 → 幅 108 と、
      幅 140 → 幅 150 に合わせる）

## 5. 正本ドキュメントと検証

- [x] 5.1 `docs/mvp/mvp.md` のキーバインド表に `c`（issue / PR を close する。内部処理は `gh issue close` / `gh pr close`）の行を
      `m` の直後で足し、変更履歴に 1 行足す（「前提と未決事項」の行は触らない。理由は proposal.md）
- [x] 5.2 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
