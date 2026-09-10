## 0. 回帰の基準を取る

- [x] 0.1 実装を始める前に `go run ./cmd/loop-cli-dev classify --fixture example` の出力を控える（#5 成功条件 6 の突き合わせに使う。8.3 で一致を確認する）

## 1. 設定に mode を足す

- [x] 1.1 `internal/config/config.go` の `Repo` に方式のフィールドを足し、`UnmarshalYAML` で `mode` キーを読み、`sdd` / `label` 以外を検証で弾き、未指定を `sdd` に解決する
- [x] 1.2 `internal/config/config_test.go` に「mode 未指定は sdd」「mode: label が読める」「mode と merge_method を同じ要素に書ける」「mode が不正ならリポジトリ名と値を含むエラー」のテストを足す

## 2. model に方式と語彙を足す

- [x] 2.1 `internal/model/model.go` に `Mode` 型（`ModeSDD` / `ModeLabel`、ゼロ値は sdd）と ILD のラベル定数（`LabelToDo` / `LabelInProgress` / `LabelDone`）を足す
- [x] 2.2 `IssueStages` / `PRStages` を `(mode Mode, labels []string)` に変え、`TodoLabel(mode Mode) string` を足す
- [x] 2.3 `internal/model/parse.go` に `ClosesIssue(body string) (int, bool)`（`Closes #n` だけを採り、`Refs` は採らない）を足し、`IsMidRelabel` を `(mode Mode, comments []Comment)` に変えて `label` の目印を `restart:` だけにする
- [x] 2.4 `internal/model/model_test.go` / `parse_test.go` に 2.1〜2.3 のテストを足す（ゼロ値が sdd、label の段階ラベル、`TodoLabel`、`ClosesIssue` が `Refs` を採らない、label で `release:` を目印にしない）

## 3. 分類器を方式で分岐させる

- [x] 3.1 `classify.Issue` / `classify.PR` / `classify.Card` に `mode model.Mode` を足し、`Card` が中の issue / PR に同じ方式を渡すようにする（`internal/classify/card.go` の `markCanonical` の `PRStages` 呼び出しも直す）
- [x] 3.2 `classify.Issue` の規則 1 を方式で分け（sdd は段階 1 件 + `wip`、label は `In Progress` 1 件かつ `question` 無し）、規則 6 を `model.IsMidRelabel(mode, …)` にする
- [x] 3.3 `classify.PR` の規則 7（`ai-assess:requested`）と行 G（`docs`）を sdd のときだけ評価し、label では行 D を行 C より先に評価する。`isC` / `isD` を方式で分ける（label は `model.ClosesIssue(Body)` が ok の open PR を対象にし、C は `question` 無し・`MERGEABLE`・checks 緑だけを見て本文 1 行目を読まない）
- [x] 3.4 `internal/classify/classify_test.go` / `card_test.go` に `human-turn-classify` の ADDED Requirement と MODIFIED した Scenario をすべて足す（既存の sdd のテストは書き換えずに通ることを確認する）

## 4. 取得と画面に方式を流す

- [x] 4.1 `fetch.Fetch` に `modes map[string]model.Mode` を足し、カードごとの方式で `classify.Card` を呼ぶ（`stageRank` の `PRStages` 呼び出しも直す）
- [x] 4.2 `internal/ui` の `Options` に `Modes map[string]model.Mode` を足し、`Model` にリポジトリ名から方式を引くメソッドを持たせる（`mergeMethod` と同じ形）
- [x] 4.3 `cmd/loop-cli/main.go` で `cfg.Repos` から方式の表を作り、`Fetch` と `ui.Options` に渡す
- [x] 4.4 `internal/fetch/fetch_test.go` に「modes の方式で分類される」「表に無いリポジトリは sdd」のテストを足し、`internal/snapshot/snapshot_test.go` / `internal/ui/testdata_test.go` / `cmd/loop-cli/main_test.go` の `Fetch` 呼び出しを新しいシグネチャに直す

## 5. t キーを方式に合わせる

- [x] 5.1 `action.ToggleTodo` に `mode model.Mode` を足し、書くラベルを `model.TodoLabel(mode)` にする
- [x] 5.2 `internal/action/testdata/todo` に label 方式の issue（ラベル無し / `To Do` / `In Progress` / `stage:propose`）の JSON を足し、`internal/action/todo_test.go` に label 方式の 4 ケースと 1 操作 1 ラベルのテストを足す
- [x] 5.3 `internal/ui/todo.go` が `Options.Modes` から引いた方式を `ToggleTodo` に渡し、フッタの文言を書いたラベル名にする（`stage:todo を切り替え中` を固定で書かない）
- [x] 5.4 `internal/ui/todo_test.go` に `todo-toggle` の label 方式の Scenario（`To Do` を書く、スナップショット表示中でも設定の方式で書く、フッタの文言）を足す

## 6. 画面を方式に合わせる

- [x] 6.1 `internal/ui/detail.go` の段階行を `model.IssueStages(mode, Labels)` にし、バッジを方式で切り替える（label は `wip` を出さない）
- [x] 6.2 `internal/ui/detail.go` の PR 一覧を方式で切り替え（label は 3 段階の見出しを出さず `[-]` で並べる）、`prHeaderLines` を含む `PRStages` の呼び出しを新しいシグネチャに直す
- [x] 6.3 `internal/ui/view.go` のキュー 0 件のヒント 2 行を 2 方式ぶんの文言にし、方式の書き忘れの知らせ（sdd 扱いのリポジトリに `To Do` / `In Progress` の issue があればフッタに出す。取得中・失敗・書き込み・部分失敗より後ろ）を足す
- [x] 6.4 `internal/ui/help.go` の `t` の説明を `stage:todo / To Do を付ける / 外す` にする
- [x] 6.5 `internal/ui/detail_test.go` / `view_test.go` / `help_test.go` を `card-detail` / `queue-screen` / `help-screen` の Scenario に合わせて足す・直す

## 7. fixture と dev CLI

- [x] 7.1 `internal/gh/testdata/fixtures/` に ILD の alias を手書きで足す（`To Do` / `In Progress` / `Done` / `blocked` + `question` の issue と、`Closes #n` を持つ緑の PR・`question` の PR。issue ごとに `issue-<n>.json`、PR ごとに `pr-<n>.json` と `pr-<n>-review-threads.json` を揃え、`login` は `user-N` 形式にする）
- [x] 7.2 `cmd/loop-cli-dev/classify.go` に `--mode` フラグ（既定 `sdd`、不正値はエラー）を足し、`cmd/loop-cli-dev/main.go` の使い方の 1 行にも書く
- [x] 7.3 `cmd/loop-cli-dev/classify_test.go` に「label の fixture が方式どおりに 4 タブへ振り分けられる」「不正な mode はエラー」のテストを足す
- [x] 7.4 `internal/classify/fixture_test.go` の期待値表を alias ごとに方式を持つ形に変え、7.1 の alias の期待値を判定表から手で当てて書く

## 8. ドキュメントと仕上げ

- [x] 8.1 `docs/domain/issue-driven-sdd/human-turn-signals.md` に ILD 方式のシグナル対応表（局面 A〜G と「キューに入れないもの」の ILD 版、`In Progress` と `question` の関係）と変更履歴の行を足す
- [x] 8.2 `docs/mvp/mvp.md` の「設定ファイル」節に `mode` を、「初回起動（onboarding）」節の空キューのヒントに新しい文言を、「前提と未決事項」の「`routines-setup` を回していないリポジトリは空」に ILD 方式の扱いを反映する
- [x] 8.3 `go build ./... && go vet ./... && go test ./...` が通ることを確認し、続けて `go run ./cmd/loop-cli-dev classify --fixture example` の出力が 0.1 で控えたものと一致することを確認する
