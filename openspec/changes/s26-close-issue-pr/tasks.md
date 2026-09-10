## 1. close の呼び出し（internal/gh）

- [ ] 1.1 `internal/gh/gh.go` の `GHClient` に `CloseIssue(ctx, repo, number)` と `ClosePR(ctx, repo, number)` を足す
- [ ] 1.2 `internal/gh/client.go` に `Client.CloseIssue` / `Client.ClosePR` を実装する（引数は `issue close <n> -R <repo>` と
      `pr close <n> -R <repo>`、標準出力は読み捨て、失敗は既存の `gh` エラーに載せる）
- [ ] 1.3 `internal/gh/fake.go` の `Fake.CloseIssue` / `Fake.ClosePR` で呼び出しを `Calls` に記録する
- [ ] 1.4 実行する引数と失敗時のエラー文字列のテストを `internal/gh/client_test.go` に書き、`Fake` が呼び出しを記録することの
      テストを `internal/gh/fake_test.go` に書く

## 2. close の判定と実行（internal/action）

- [ ] 2.1 `internal/action/close.go` に、拒否を返す判定関数（PR の `State` が空でも `OPEN` でもなければ `<State> の PR です`、
      issue は拒否なし）と、`action.Target` の `IsPR` で `ClosePR` / `CloseIssue` を呼び分ける close 関数を足す
- [ ] 2.2 テストを `internal/action/close_test.go` に書く。検証するのは拒否の判定（MERGED / CLOSED / OPEN / 空文字列 / issue）と、
      書き先の分かれ方（`Calls` が `ClosePR` か `CloseIssue` のちょうど 1 件）と、ラベルもコメントも書かないこと

## 3. c と確認画面（internal/ui）

- [ ] 3.1 `internal/ui/close.go` を作り、close の状態（リポジトリ・番号・種別・表示名・タイトル・ラベル・PR の `State`・拒否・
      戻り先）と、画面ごとに対象を決める処理と、`c` のハンドラを置く（書き込み中は何もしない。`gh` は呼ばない）
- [ ] 3.2 `Model` の画面の状態に close の確認画面を足し、`internal/ui/model.go` のキー振り分けに確認画面の分岐を merge の
      確認画面の分岐の直後（`a` / `t` / `m` / `o` / `?` / `u` より先）、`c` のハンドラを `m` のハンドラの直後に置く
- [ ] 3.3 確認画面のキー（`y` は拒否が空のときだけ close のコマンドを返して戻り先へ、`Esc` は中止してステータスを出す、
      他のキーは何もしない）と、close のコマンドと結果のメッセージ、`m.writing` による二重防止を実装する
- [ ] 3.4 確認画面の描画を実装する。出す行は `close の確認: <表示名>  <タイトル>` と `種別:` と `labels:` と拒否の赤行で、
      幅で切り詰め、フッタ左は拒否の有無で変える。`internal/ui/view.go` の画面分岐にも 1 行足す
- [ ] 3.5 `internal/ui/help.go` の `helpKeys` に `c  issue / PR を close する（確認あり）` の行を `m` の直後で足す

## 4. テスト（internal/ui）

- [ ] 4.1 画面遷移のテストを書く（3 画面で `c` が確認画面を開き対象が画面どおりに決まる・0 行のタブとヘルプ画面と各確認画面では
      何もしない・`Esc` で戻り先に戻る・確認中に取得が完了しても対象が変わらない）
- [ ] 4.2 確認画面の描画のテストを書く（issue と PR の行の形・`labels: なし`・`State` が `CLOSED` の PR で
      `close できません:` が出て `y close` が消える・その画面で `y` が何もしない）
- [ ] 4.3 close の実行のテストを書く（`Fake.Calls` の `CloseIssue` / `ClosePR` がちょうど 1 件・成功と失敗のフッタ・
      close 中の `t` / `a` / `m` / `c` が効かず `o` は効く・`Cards` が変わらない・ラベルとコメントを書かない）
- [ ] 4.4 既存テストのうちヘルプ画面のキーの行数を見ているものを新しい値（17 行）に直す

## 5. 正本ドキュメントと検証

- [ ] 5.1 `docs/mvp/mvp.md` のキーバインド表に `c`（issue / PR を close する。内部処理は `gh issue close` / `gh pr close`）の行を
      `m` の直後で足し、変更履歴に 1 行足す
- [ ] 5.2 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
