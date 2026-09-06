## 1. merge のアクション（internal/action）

- [x] 1.1 `internal/action/merge.go` を作成し、`CheckMerge(pr model.PR) (blocked []string, warnings []string)` を実装する。`IsDraft` なら `blocked` に `draft の PR です`、`State` が空でも `OPEN` でもなければ `<State> の PR です`、`model.HasLabel(pr.Labels, model.LabelQuestion)` なら `question ラベルが付いています`、`model.ParseUndecided(pr.Body)` が `ok` かつ `n >= 1` なら `本文 1 行目が「未確定の判断: <n> 件」です`、`classify.ChecksGreen(pr.MergeState)` が false なら `checks が緑ではありません` を、この順で `warnings` に足す
- [x] 1.2 同ファイルに `Merge(ctx, client, repo, number, method) error` と `ErrBadMergeMethod` を実装する。`method` が `squash` / `merge` / `rebase` のいずれかなら `client.MergePR` を 1 回呼んでその戻り値を返し、それ以外は `client` を呼ばず `ErrBadMergeMethod`（`Error()` に `method` を含む）を返す
- [x] 1.3 `internal/action/merge_test.go` を作成し、`merge-action` の Scenario を検証する。`CheckMerge` は draft / merged / 警告 3 種 / 警告なし / 1 行目が未確定の形でない / `MergeState` が nil の 6 ケース、`Merge` は `squash` / `merge` / `rebase` の `MergeMethod` 記録、不正な方式 2 種で `Calls` が空、`MergePR` がエラーを返す `Fake` の埋め込み型でエラーが素通りすること、全ケースを通して `Calls` に `AddLabel` / `RemoveLabel` / `CommentIssue` / `CommentPR` / `CreateIssue` / `ReplyReviewThread` / `Browse` / `OpenURL` が無いこと

## 2. merge 方式の配線（internal/ui・cmd/loop-cli）

- [x] 2.1 `internal/ui/model.go` の `Options` に `MergeMethods map[string]string` を足し、`Model` に保持する。リポジトリ名で引き、見つからなければ `squash` を返す小さな関数を `internal/ui/merge.go` に置く
- [x] 2.2 `cmd/loop-cli/main.go` で `cfg.Repos[]` から `Name` → `string(MergeMethod)` の対応表を作り、`ui.Options` に渡す。`internal/ui` は `internal/config` を import しない
- [x] 2.3 `cmd/loop-cli/main_test.go` に、対応表がリポジトリ別の `merge_method` を反映することの検証を足す

## 3. m の押下と状態の取り直し（internal/ui）

- [x] 3.1 `internal/ui/merge.go` に、対象の PR（リポジトリ・番号・表示名）を画面の状態から決める処理と、`ViewPR` / `ViewPRMergeState` を並行に呼んで両方の完了を待ち、結果（リポジトリ名 / 番号 / 表示名 / `*gh.PRDetail` / `*gh.PRMergeState` / エラー / `m` を押した画面）を運ぶメッセージを返すコマンドを実装する。`ctx` は 30 秒
- [x] 3.2 `internal/ui/model.go` の `Model.Update` に `m` の処理を足す。キュー画面は主体が PR のときだけ、カード詳細は選択中の PR、PR 詳細はその PR を対象にし、他の画面では何もしない。取り直し中は書き込み中フラグを立て、ステータスに `<表示名> の状態を取得中` を出す
- [x] 3.3 取り直しの結果のメッセージを処理する。片方でもエラーなら `<表示名> の状態を取得できません: <エラー>` を赤で出して画面を変えない。届いた時点の画面が `m` を押した画面と違えば `<表示名> の merge を中止しました（画面が変わりました）` を出して捨てる。両方成功して画面も同じなら、`State` を画面の Card から写した `model.PR` を組み立てて `action.CheckMerge` を呼び、リポジトリ名・番号・表示名・方式・戻り先を確認状態に保持して merge の確認画面に移る
- [x] 3.4 `internal/ui/testdata/merge/pr-131.json`（`labels` は `propose` のみ・`isDraft` false・`body` の 1 行目が `未確定の判断: 0 件`・`mergeable` `MERGEABLE`・`mergeStateStatus` `CLEAN`・`statusCheckRollup` は `CheckRun` の `test` `SUCCESS` 1 件）と `internal/ui/testdata/merge-draft/pr-131.json`（`isDraft` true）を作る
- [x] 3.5 `internal/ui/merge_test.go` を作成し、`merge-pr` の Requirement「m は画面の対象 PR を決めて GitHub から状態を取り直す」と「取り直しに成功したら確認画面へ移り、失敗したら赤で出す」の Scenario を検証する

## 4. merge の確認画面と実行（internal/ui）

- [x] 4.1 画面の状態に merge の確認を足し、`View` を実装する。`View` は上から順に、見出しの `merge の確認: <表示名>` を出し、続けて `方式:`、`labels:`、`mergeable:`、`checks:` の各行を出し、次に `merge できません:` の行を赤で出し、その後に `注意:` の行を出し、区切り線を挟んで本文を出す。収まらない分は残りの高さで切る（スクロールしない）。フッタ左は `blocked` が空なら `y merge  Esc 中止  q 終了` を出し、空でなければ `Esc 中止  q 終了` を出す
- [x] 4.2 確認画面のキーを実装する。`y` は `blocked` が空のときだけ merge のコマンドを返して戻り先に戻る。`Esc` は戻り先に戻して `merge を中止しました`。`q` / `Ctrl+C` は終了。他のキーは何もしない。取得完了のメッセージは `Cards` と時刻だけ更新する。詳細画面へ戻るときは `updateHelpKey` と同じく詳細の寸法を作り直す
- [x] 4.3 merge のコマンド（`action.Merge` を 30 秒の `ctx` で呼ぶ）と結果の処理を実装する。引数は確認状態が保持しているリポジトリ名・番号・方式だけを使い、選択行や詳細の対象から導かない。merge 中 / 成功 / 失敗のステータスを出し、`Cards` も詳細の対象も変えず、取得のコマンドを返さない
- [x] 4.4 `internal/ui/merge_test.go` に、確認画面の表示・キー・merge の結果の Scenario を足す（警告あり / draft で `y` が出ない / `Esc` で中止 / 詳細画面へ戻る / 確認画面での取得完了 / 取得完了で選択行が動いても対象が変わらない / 取り直し中に画面を変えると確認画面に移らない / `y` で `MergePR` が 1 回 / merge 中のキー / 失敗が赤 / ラベルとコメントを書かない / 方式が `rebase` と既定の `squash`）

## 5. 既存画面の追従

- [x] 5.1 書き込み中フラグの判定に `m` を加える（`t` / `a` / `m` を受け付けない）。`internal/ui/todo_test.go` の該当テストを `todo-toggle` の MODIFIED に合わせて更新し、merge 中に `t` / `a` が効かない検証を足す
- [x] 5.2 `tui-entrypoint` の MODIFIED（初期フレームの `q 終了` は取得完了後または幅 89 列以上で出る）に合わせて初期フレームのテストを更新する。キュー画面のフッタのヒントを `Enter 開く  a 回答  t todo  m merge  o ブラウザ  R 更新  ? ヘルプ  u URL  q 終了`（80 列）に変え、`internal/ui/view_test.go` の該当テスト（既定幅 80 で取得中はステータスだけになる / 幅 90 では両方出る）を更新する
- [x] 5.3 詳細画面のフッタのヒントに `m merge` を足す（カード詳細は `PRs` が空なら省く。PR 詳細は `a 回答` の次で 82 列になる）。`internal/ui/detail_test.go` の該当テストを更新する
- [x] 5.4 `internal/ui/help.go` の `helpKeys` に `m` の行（`PR を merge する（確認あり）`）を `t` の次に足し、`internal/ui/help_test.go` の行数と順序の検証を更新する

## 6. 仕上げ

- [x] 6.1 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [x] 6.2 `openspec validate s14-merge-pr --strict` が通ることを確認する
