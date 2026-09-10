## 0. 前提

- [ ] 0.1 proposal.md「未確定の判断」の Q1 / Q2 / Q3 が「確定した判断」へ移り、PR 本文 1 行目が `未確定の判断: 0 件` になっていることを確認する（回答で行 C / D の条件が変わるため、確定前に実装を始めない）

## 1. 設定に mode を足す

- [ ] 1.1 `internal/config/config.go` の `Repo` に方式のフィールドを足し、`UnmarshalYAML` で `mode` キーを読み、`sdd` / `label` 以外を検証で弾き、未指定を `sdd` に解決する
- [ ] 1.2 `internal/config/config_test.go` に「mode 未指定は sdd」「mode: label が読める」「mode と merge_method を同じ要素に書ける」「mode が不正ならリポジトリ名と値を含むエラー」のテストを足す

## 2. model に方式と語彙を足す

- [ ] 2.1 `internal/model/model.go` に `Mode` 型（`ModeSDD` / `ModeLabel`、ゼロ値は sdd）と ILD のラベル定数（`LabelToDo` / `LabelInProgress` / `LabelDone`）を足す
- [ ] 2.2 `IssueStages` / `PRStages` を `(mode Mode, labels []string)` に変え、`TodoLabel(mode Mode) string` を足す
- [ ] 2.3 `model.Issue` / `model.PR` に `Mode` フィールドを足す
- [ ] 2.4 `LinkedIssue(title, body string) (int, bool)` を `internal/fetch/link.go` から `internal/model` へ移し、`internal/fetch` からその関数を消して呼び出しを `model.LinkedIssue` に差し替える（テストも移す）
- [ ] 2.5 `internal/model/parse.go` の `IsMidRelabel` を `(mode Mode, comments []Comment)` に変え、`label` の目印を `restart:` だけにする
- [ ] 2.6 `internal/model/model_test.go` / `parse_test.go` に 2.1〜2.5 のテストを足す（ゼロ値が sdd、label の段階ラベル、`TodoLabel`、label で `release:` を目印にしない）

## 3. 分類器を方式で分岐させる

- [ ] 3.1 `classify.Issue` の規則 1 を方式で分ける（sdd は段階 1 件 + `wip`、label は `In Progress` 1 件）
- [ ] 3.2 `classify.Issue` の規則 6 を `model.IsMidRelabel(is.Mode, is.Comments)` に変える
- [ ] 3.3 `classify.PR` の規則 7（`ai-assess:requested`）と行 G（`docs`）を sdd のときだけ評価する
- [ ] 3.4 `isC` を方式で分ける（label は `model.LinkedIssue` が ok・`question` 無し・`MERGEABLE`・checks 緑。本文 1 行目を読まない）
- [ ] 3.5 `isD` を方式で分ける（label は `model.LinkedIssue` が ok の open PR を対象にする）
- [ ] 3.6 `internal/classify/classify_test.go` に `human-turn-classify` の ADDED Requirement の Scenario をすべて足す
- [ ] 3.7 既存の分類器テストを 1 行も書き換えずに通ることを確認する（#5 成功条件 6 の回帰）

## 4. 取得経路に方式を流す

- [ ] 4.1 `fetch.Fetch` に `modes map[string]model.Mode` を足し、各 issue / PR の `Mode` を埋める
- [ ] 4.2 `cmd/loop-cli/main.go` で `cfg.Repos` から `modes` を組み立てて `Fetch` に渡す
- [ ] 4.3 `internal/fetch/fetch_test.go` に「modes の方式が issue / PR に入る」「表に無いリポジトリは sdd」のテストを足す

## 5. t キーを方式に合わせる

- [ ] 5.1 `action.ToggleTodo` に `mode model.Mode` を足し、書くラベルを `model.TodoLabel(mode)` にする
- [ ] 5.2 `internal/action/testdata/todo` に label 方式の issue（ラベル無し / `To Do` / `In Progress` / `stage:propose`）の JSON を足す
- [ ] 5.3 `internal/action/todo_test.go` に label 方式の 4 ケースと、書き込みが 1 回 1 ラベルであるテストを足す
- [ ] 5.4 `internal/ui/todo.go` が対象 Issue の `Mode` を `ToggleTodo` に渡すようにし、`internal/ui/todo_test.go` を追随させる

## 6. 画面を方式に合わせる

- [ ] 6.1 `internal/ui/detail.go` の段階行を `model.IssueStages(Mode, Labels)` にし、バッジを方式で切り替える（label は `wip` を出さない）
- [ ] 6.2 `internal/ui/detail.go` の PR 一覧を方式で切り替える（label は 3 段階の見出しを出さず、`[-]` で並べる）
- [ ] 6.3 `internal/ui/view.go` のキュー 0 件のヒント 2 行を 2 方式ぶんの文言にする
- [ ] 6.4 `internal/ui/help.go` の `t` の説明を `stage:todo / To Do を付ける / 外す` にする
- [ ] 6.5 `internal/ui/detail_test.go` / `view_test.go` / `help_test.go` を `card-detail` / `queue-screen` / `help-screen` の Scenario に合わせて足す・直す

## 7. fixture と dev CLI

- [ ] 7.1 `internal/gh/testdata/fixtures/` に ILD の alias を手書きで足す（`To Do` / `In Progress` / `Done` / `blocked` + `question` の issue と、`Closes #n` を持つ緑の PR・`question` の PR。`login` は `user-N` 形式）
- [ ] 7.2 `cmd/loop-cli-dev/classify.go` に `--mode` フラグ（既定 `sdd`、不正値はエラー）を足し、読み込んだ issue / PR の `Mode` に入れる
- [ ] 7.3 `cmd/loop-cli-dev/classify_test.go` に「label の fixture が方式どおりに 4 タブへ振り分けられる」「不正な mode はエラー」のテストを足す
- [ ] 7.4 `internal/classify/fixture_test.go` の期待値表を alias ごとに方式を持つ形に変え、7.1 の alias の期待値を判定表から手で当てて書く

## 8. ドキュメントと仕上げ

- [ ] 8.1 `docs/domain/issue-driven-sdd/human-turn-signals.md` に ILD 方式のシグナル対応表（局面 A〜G と「キューに入れないもの」の ILD 版）と変更履歴の行を足す
- [ ] 8.2 `docs/mvp/mvp.md` の「設定ファイル」節に `mode` を、「初回起動（onboarding）」節の空キューのヒントに新しい文言を反映する
- [ ] 8.3 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
