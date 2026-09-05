## 1. URL の抽出（internal/model）

- [x] 1.1 `internal/model/parse.go` に、本文から URL とリンクテキストを出現順に取り出す関数と、その戻り値の型を足す（Markdown リンクと裸の URL、末尾の句読点と閉じ括弧の削り）
- [x] 1.2 `internal/model/parse_test.go` に抽出のテストを書く（Markdown リンクと裸 URL の混在・末尾記号・参照記法と相対リンクを取らない・同じ URL が 2 回）

## 2. URL を開く経路（internal/gh）

- [x] 2.1 `internal/gh/gh.go` の `GHClient` に `OpenURL(ctx, url)` を足し、`Call` に `URL` の欄を足す
- [x] 2.2 `internal/gh/client.go` に `Client.OpenURL` を実装する（ブラウザ起動コマンド用の差し替え可能な関数を持ち、既定は `darwin` で `open`、他は `xdg-open`。失敗はコマンドと URL と stderr を含むエラー）
- [x] 2.3 `internal/gh/fake.go` の `Fake.OpenURL` で呼び出しを `Calls` に記録する
- [x] 2.4 `internal/gh/client_test.go` と `internal/gh/fake_test.go` に、渡すコマンド名と引数・OS 別のコマンド名・失敗時のエラー・`Fake` の記録のテストを書く

## 3. URL 一覧画面（internal/ui）

- [x] 3.1 `internal/ui` に URL 一覧の状態（一覧の項目・選択位置・戻り先）と、対象から URL を集めて重複を除く処理を新しいファイルで足す
- [x] 3.2 `Model` の画面の状態に URL 一覧を足し、`Update` のキー振り分けに「URL 一覧画面の分岐 → `u` のハンドラ」の順で置く（どちらも `a` / `t` / `o` / `?` の判定より前）。`u` は画面ごとに対象を決め、0 件ならステータスを出す
- [x] 3.3 一覧画面のキー（`j` / `k` / `↑` / `↓` / `Enter` / `Esc` / `q`、他は何もしない）と、`Esc` で戻るときの詳細の作り直しを実装する
- [x] 3.4 `OpenURL` を呼ぶコマンドと結果のメッセージ、失敗時のフッタ表示（赤）を実装する
- [x] 3.5 一覧画面の `View`（見出し・URL の行・空行・フッタ、幅での切り詰め、選択に追従する表示位置）を実装する

## 4. ヒントとヘルプ

- [x] 4.1 キュー・カード詳細・PR 詳細のフッタのヒントに `u URL` を `? ヘルプ` の直後で足す
- [x] 4.2 ヘルプ画面のキー一覧に `u  URL 一覧を開く` の行を `o` の直後で足す

## 5. テストと正本ドキュメント

- [x] 5.1 `internal/ui` に画面遷移のテストを書く（3 画面で `u` が開く・0 件はステータス・0 行 / 確認 / ヘルプでは何もしない・`Esc` で戻る・他のキーは何もしない）
- [x] 5.2 `internal/ui` に一覧の内容のテストを書く（本文 → コメント → thread の順と出典・重複の除去・`nil` の収集元を飛ばす）
- [x] 5.3 `internal/ui` に描画と `Enter` のテストを書く（行の形・選択の印・件数が高さを超えるときの追従・`Fake.Calls` の `OpenURL`・失敗の赤表示）
- [x] 5.4 既存テストのうち、フッタのヒントとヘルプの行数を見ているものを新しい期待値に直す（s21-always-two-pane が直した後の値を土台にする）
- [x] 5.5 `docs/mvp/mvp.md` のキーバインド表に `u`（URL 一覧を開く / 内部処理は `open <url>`）の行を足し、変更履歴に 1 行足す
- [x] 5.6 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
