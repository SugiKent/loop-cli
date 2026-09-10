## 1. issue 作成のアクション（internal/action）

- [ ] 1.1 `internal/action/newissue.go` を作成し、`SplitNewIssue(text string) (title, body string)` を実装する。最初の改行で 2 つに分け、それぞれ `strings.TrimSpace` した結果を返す。改行が無ければ `body` は空
- [ ] 1.2 同ファイルに `CreateIssue(ctx, client, repo, title, body) (string, error)` と `ErrEmptyTitle` を実装する。タイトルが空 → `ErrEmptyTitle`、本文が空 → `ErrEmptyBody`、`HasRoutineMarker` がタイトルか本文に対して真 → `ErrRoutineMarker` の順に検査し、通ったときだけ `client.CreateIssue` を 1 回呼んでその戻り値を返す
- [ ] 1.3 `internal/action/newissue_test.go` を作成し、`new-issue-action` の Scenario を検証する。`SplitNewIssue` は 3 ケース（空行を挟む / 1 行だけ / 空行から始まる）、`CreateIssue` は成功時の `Calls` と URL、拒否 4 種で `Calls` が空、`CreateIssue` がエラーを返す `Fake` の埋め込み型でエラーが素通りすること、全ケースを通して `Calls` に `AddLabel` / `RemoveLabel` / `CommentIssue` / `CommentPR` / `MergePR` / `ReplyReviewThread` が無いこと

## 2. n の押下とエディタ（internal/ui）

- [ ] 2.1 `internal/ui/editor.go` の一時ファイルの接頭辞を `loop-cli-draft-` に変え、`internal/ui/editor_test.go` に「一時ファイル名が `loop-cli-draft-` で始まる」ことの検証を足す（`answer-question` の MODIFIED。今のテストは `.md` で終わることしか見ていない）
- [ ] 2.2 `internal/ui/new.go` を作成し、作成の状態（作成先・下書き・タイトル・本文・マーカーの有無・戻り先の画面）と、画面から作成先のリポジトリを決める処理を実装する。キュー画面は選択行の主体、カード詳細は `Card.Issue`、PR 詳細はその PR から `Repo` を取り、選択行が無ければ `新規作成の対象がありません` をステータスに出す
- [ ] 2.3 `internal/ui/model.go` の `Model` に「今の編集が `a` と `n` のどちらで始まったか」を持たせ、エディタを開く 4 か所（`a` / `n` / 回答の確認画面の `e` / 作成の確認画面の `e`）すべてで記録し、`updateEdited` をその値で振り分ける。作成・中止・エラーでは消さない。`Update` の `tea.KeyPressMsg` の分岐に `n` を足す（作成の確認画面の振り分けは回答 / merge の確認画面と並べて `a` / `t` / `m` / `o` / `?` / `u` より先に置く）。書き込み中は `n` を受け付けない
- [ ] 2.4 `new.go` に `n` で始めた編集完了の処理を実装する。エラー → `エディタ: <エラー>` を赤、タイトルが空 → `作成を中止しました（タイトルが空）`、本文が空 → `作成を中止しました（本文が空）`、それ以外 → 下書きとタイトルと本文とマーカーの有無を保持して作成の確認画面へ
- [ ] 2.5 `internal/ui/new_test.go` を作成し、`new-issue` の Requirement「n は画面の対象のリポジトリを作成先にしてエディタを開く」と「編集結果を検査して作成の確認画面に移る」の Scenario を検証する

## 3. 作成の確認画面と実行（internal/ui）

- [ ] 3.1 画面の状態に作成の確認を足し、`View` を実装する。上から `新規 issue の確認: <repo>`、`タイトル: <タイトル>`、マーカーがあるときだけ警告行、区切り線、本文の順で描き、残りの高さで切る。フッタ左はマーカーが無ければ `y 作成  e 編集に戻る  Esc 中止  q 終了`、あれば `y 作成` を省く
- [ ] 3.2 確認画面のキーを実装する。`y` はマーカーが無いときだけ作成のコマンドを返して戻り先へ、`e` は下書き全体を `initial` にして `Editor` を呼び戻り先へ、`Esc` は `作成を中止しました` を出して戻り先へ、`q` / `Ctrl+C` は終了、他のキーは何もしない。取得完了は `Cards` と時刻だけ更新する。詳細画面へ戻るときは詳細の寸法を作り直す
- [ ] 3.3 作成のコマンド（`action.CreateIssue` を 30 秒の `ctx` で呼ぶ）と結果の処理を実装する。引数は確認状態が保持している作成先・タイトル・本文だけを使う。作成中 / 成功（`<repo> に issue を作成しました: <URL>`）/ 失敗（赤）のステータスを出し、`Cards` を変えず、取得のコマンドを返さない
- [ ] 3.4 `internal/ui/new_test.go` に、確認画面の表示・キー・作成の結果の Scenario を足す（`y` で `CreateIssue` が 1 回 / `e` で下書き全体が戻る / **`e` から戻った編集完了が再び作成の確認画面になり `CommentPR` を呼ばない** / `Esc` で中止 / マーカーで `y` が出ない / 詳細画面へ戻る / 確認中の取得完了で作成先が変わらない / 確認画面で他のキーが効かない / **成功時に `issue を作成しました: <URL>` がフッタに出て、長いリポジトリ名でも幅 80 で URL が切れない** / 作成中のキー / 失敗が赤 / ラベルを書かない）
- [ ] 3.5 `internal/ui/answer_test.go` に検証を 2 本足す（`answer-question` の MODIFIED。今のテストは `e` の後の編集完了を `Update` に戻していない）。1 本目は `n` で始めた編集が回答として扱われないこと、2 本目は回答の確認画面の `e` から戻った編集完了が `CommentPR` になること

## 4. 既存画面の追従

- [ ] 4.1 書き込み中フラグの判定に `n` を加える（`t` / `a` / `m` / `n` を受け付けない）。`internal/ui/todo_test.go` の該当テストを `todo-toggle` の MODIFIED に合わせて更新し、issue 作成中に `t` / `a` / `m` が効かない検証を足す
- [ ] 4.2 キュー画面のフッタのヒント（80 列）は変えず、`internal/ui/view_test.go` の「フッタのキーヒント」のテストに `n 新規` を含まないことの検証を足す（`queue-screen` の MODIFIED）
- [ ] 4.3 詳細画面のフッタのヒントに `n 新規` を `m merge` の次に足す（カード詳細は `PRs` が空でも出す。PR 詳細は 90 列になる）。`internal/ui/detail_test.go` の該当テストを幅 130 に合わせて更新する
- [ ] 4.4 `internal/ui/help.go` の `helpKeys` に `n` の行（`選択中の repo に issue を作る（確認あり）`）を `m` の次に足し、`internal/ui/help_test.go` の行数（17 行）と順序と「未実装のキーは出ない」の検証を更新する
- [ ] 4.5 `internal/ui/model_test.go` の「未実装のキーは何も変えない」から `n` を外す（`queue-screen` の MODIFIED）

## 5. docs と仕上げ

- [ ] 5.1 `docs/mvp/mvp.md` キーバインド表の `n` の行を、選択中の対象の repo に `$EDITOR` で作る形に書き換え、内部処理の欄を `gh issue create -R <選択中の repo>`（ラベルは付けない）に直し、変更履歴に 1 行足す
- [ ] 5.2 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [ ] 5.3 `openspec validate s15-new-issue --strict` が通ることを確認する
