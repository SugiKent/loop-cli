## 1. 1 ペイン経路の削除

- [x] 1.1 `internal/ui/model.go` から `twoPane()` と `Model.showPreview` を削除し、`updateKey` の `case "p"` を落として `p` を何もしないキーに戻す
- [x] 1.2 `internal/ui/view.go` の `render()` からキュー画面の 1 ペイン分岐を削除し、常に ヘッダ / 表 / 区切り線 / プレビュー / フッタ を描く（按分は現行の `tableH = (rest + 1) / 2` のまま）
- [x] 1.3 `internal/ui/view.go` の `queueHint()` から `p プレビュー` / `p 一覧` の分岐を削除し、固定のヒント文字列を返す

## 2. ヘルプ

- [x] 2.1 `internal/ui/help.go` の `helpKeys` から `p` の行を削除する

## 3. テスト

- [x] 3.1 `internal/ui/view_test.go` の `TestNarrowTerminalTogglesWithP` と `TestShortTerminalIsOnePane` を、幅 79 / 高さ 15 でも表とプレビューの両方が出ることを検証するテストに書き換える
- [x] 3.2 `internal/ui/view_test.go` の `TestPDoesNothingInTwoPane` を、狭い端末（幅 79）でも `p` が View を変えずコマンドも返さないことを含む検証にする
- [x] 3.3 `internal/ui/view_test.go` の `TestPreviewPaneShowsNoHint` を、幅 60 で空タブのヒントが表の領域に出て、同時にプレビュー領域の `（このタブにはカードがありません）` も出ることを検証するテストに置き換える
- [x] 3.4 `internal/ui/help_test.go` のキーの行数を 15 から 14 に直し、`p` の行を期待する検証を `PgUp / PgDn` に読み替え、`表とプレビューの切替` が出ないことを検証する
- [x] 3.5 `internal/ui/help_test.go` の `TestHelpCutsTailOnShortTerminal` で、切られる対象を `p` の行から `PgUp / PgDn` の行に読み替える
- [x] 3.6 `internal/ui/model_test.go` の `TestUnimplementedKeysDoNothing` のキー一覧に `p` を足す
- [x] 3.7 `internal/ui/detail_test.go` の `TestNarrowTerminalKeepsOnePane` から、消えた `p プレビュー` のヒントを見る検証を外す（詳細が 1 ペインであること自体の検証は残す）

## 4. ドキュメント

- [x] 4.1 `docs/mvp/mvp.md` の「狭い端末では上下を切り替える 1 ペイン表示にフォールバックする」を、端末サイズによらず 2 ペインで出す記述に置き換える
- [x] 4.2 `docs/mvp/implementation-tasks.md` の同じフォールバックに触れている行を直す
- [x] 4.3 `README.md` の「端末が 80 桁 20 行以上あるときは…`p` で一覧とプレビューを切り替えます」の段落を常に 2 ペインの記述に直し、キュー画面のキーバインド表から `p` の行を削除する

## 5. 確認

- [x] 5.1 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [x] 5.2 `sugi-loop` を幅 60・高さ 12 程度に狭めた端末で起動し、表とプレビューの両方が出ること、`p` で何も起きないこと、フッタに `p` のヒントが無いことを目視で確認する
- [x] 5.3 行が 15 件以上あるタブを幅 79・高さ 24 で開き、表が減った行数で読めること、`j` で選択行が表の高さを超えたときの見え方を目視で確認する（design.md の Risks に記録した劣化の確認）
