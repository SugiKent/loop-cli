# tasks: s30-mode-from-labels

Refs #24

## 0. 前提

- [ ] 0.1 着手時に `origin/main` の `openspec/specs/` を見て、この change が MODIFIED する Requirement（`card-model`「運用方式の型と方式ごとのラベル語彙を model が持つ」、`card-fetch` の「gh 呼び出しには…」「search の失敗と…」、`card-detail` の「ヘッダは…」「紐づく PR 一覧は…」、`gh-client`「ListLabels は…」、`human-turn-classify` の「局面 C は…」「局面 D は…」、`queue-screen`「取得が成功して Card が 0 件のとき…」、`config-loading`「不正な設定は…」、`label-picker`「L は画面の対象のラベル一覧画面を開く」）を、その時点の最新の版から写し直す。MODIFIED は Requirement ブロック全体を置き換えるので、写し忘れると先に archive された change の変更が消える。`s26-close-issue-pr` は `queue-screen` / `todo-toggle` / `card-detail` の**別の** Requirement（キーの割り当て・フッタ・書き込み中・詳細の開閉とスクロール）を触るので、写し直しの対象が重ならないことも確かめる
- [ ] 0.2 REMOVED + ADDED で作り直す 4 つの Requirement（`card-fetch`「Fetch は設定リポジトリに…」→「Fetch は search を 2 回だけ実行し、方式を判定して…」、`human-turn-classify` の label の 1 本、`todo-toggle` の `t` の 1 本、`queue-screen` の「リポジトリごとの運用方式を設定から受け取る」と「方式の取り違えらしき状態をフッタで知らせる」）について、`origin/main` に同名の Requirement が残っていることを確認する。既に別の change が消していたら、archive が REMOVED を拒むので proposal を直す

## 1. ラベル一覧から方式を判定する（internal/model）

- [ ] 1.1 `internal/model/model.go` に `ModeFromLabels(labels []gh.RepoLabel) (Mode, bool)` を足す。`stage:todo` があれば `ModeSDD`、無くて `To Do` があれば `ModeLabel`、どちらも無ければゼロ値と `false` を返す
- [ ] 1.2 `internal/model/model_test.go` に `card-model`「リポジトリのラベル一覧から運用方式を判定する」の 5 つの Scenario（`stage:todo` だけ / `To Do` だけ / 両方 / どちらも無い（小文字の `todo` を含む） / 空の一覧）を足す

## 2. Fetch がラベル一覧を取って方式の表を返す（internal/fetch）

- [ ] 2.1 `internal/fetch/fetch.go` の `Result` に `Modes map[string]model.Mode` を足し、`Fetch` の引数から `modes` を落とす。`buildCards` は `Result.Modes` を引く
- [ ] 2.2 `fetchDetails` の並行プールに、リポジトリごとの `ListLabels` を投入する。成功したものを `model.ModeFromLabels` に通し、判定できたリポジトリだけを表に入れる。表の書き込みは `mu` で守る
- [ ] 2.3 `detailError` に「どの層の失敗か」（リポジトリ / issue / PR）を表す順位を足し、`sortErrors` をリポジトリ → issue → PR の順に直す。`ListLabels` の失敗は `ListLabels <owner>/<name>: %w` の形にする
- [ ] 2.4 `internal/fetch/fetch_test.go` に、`card-fetch` の新しい Scenario を足す。リポジトリごとに違うラベル一覧を返すスタブ `GHClient`（既存の呼び出し回数を数えるスタブと同じ形）を書き、「リポジトリごとに 1 回ずつ呼ぶ」「リポジトリごとに違う方式を判定する」「判定できないリポジトリは表に入らない」「ラベル一覧の取得に失敗したリポジトリは方式の表に入らない」「リポジトリのエラーは番号を持つエラーより前に並ぶ」を検証する。`modes` 引数を渡していた既存のケースは、fixture の `labels.json` で方式が決まる形に書き換える

## 3. label 方式の PR 分類（internal/classify）

- [ ] 3.1 `internal/classify/classify.go` の `PR` で、進行中の規則 2 / 3（最新コメントが人）を `mode != model.ModeLabel` のときだけ評価するようにする
- [ ] 3.2 `isC` と `isD` から `label` の `model.ClosesIssue` の絞り込みを外す。`isC` の `label` 側は「`question` が無い」「mergeable」「checks 緑」の 3 つだけ、`isD` の `label` 側は thread の条件だけになる
- [ ] 3.3 `internal/classify/classify_test.go` の `label` のケースを、`human-turn-classify` の新しい Scenario（`Closes` の無い緑の PR が C / `Closes` の無い PR の未 resolve thread が D / 最新コメントが人でも C / `question` に人が答えても A / merge できない PR は その他）に合わせて書き換える。`sdd` のケースは 1 本も変えない
- [ ] 3.4 `internal/gh/testdata/fixtures/board` を使う期待値表（`human-turn-classify`「fixture と期待値表で分類器をテストする」の対象）を、新しい規則での値に直す

## 4. UI が取得結果の方式を持ち、t を拒否する（internal/ui）

- [ ] 4.1 `internal/ui/model.go` の `Options` から `Modes` を落とし、`Model` の `modes` を「取得のたびに差し替える状態」にする。取得完了のメッセージを扱う箇所で `Result.Modes` を丸ごと代入し、取得失敗では触らない
- [ ] 4.2 `internal/ui/merge.go` の `repoMode` に、方式が分かっているかを返す形（`(model.Mode, bool)` を返すか、判定用の関数を別に持つ）を足す。表示側（`detail.go` の 3 か所と `view.go`）は今までどおりゼロ値で描く
- [ ] 4.3 `internal/ui/todo.go` の `todoKey` に、方式が分からないリポジトリでコマンドを返さずフッタへ `<Repo> の運用方式が分かりません（stage:todo / To Do のラベルがありません）` を赤で出す枝を足す
- [ ] 4.4 `internal/ui/view.go` から `missingLabelMode` と、フッタの `mode: label の設定漏れかもしれません` の枝を消す。0 件ヒントの 2 行目を `issue-driven-sdd の routines-setup を回すか、issue-label-driven の To Do ラベルを作ってください` に直す
- [ ] 4.5 `internal/ui/labels.go` で、取得が成功したメッセージを扱うときに `repoLabels` を捨てる（`label-picker` の「取得のたびに一覧を取り直す」）。取得失敗では捨てない
- [ ] 4.6 `internal/ui` のテストを直す。`Options.Modes` を渡していたケース（`todo_test.go` / `detail_test.go` / `view_test.go` など）を `Result.Modes` で渡す形に書き換え、`todo-toggle` の新しい Scenario 3 本（初回取得の前 / `ListLabels` の失敗 / 他のリポジトリは書ける）、`queue-screen` の新しい Scenario 3 本（表の入れ替え / 取得失敗で前回を残す / 初回取得の前）、`label-picker` の新しい Scenario 2 本（取得のたびに取り直す / 取得失敗では捨てない）を足す。設定漏れの警告のテストは消す

## 5. 設定から mode を落とす（internal/config, cmd/loop-cli）

- [ ] 5.1 `internal/config/config.go` から `Repo.Mode` と `validate` の `mode` の検査を消し、`UnmarshalYAML` の `case "mode":` を「廃止した」と伝えるエラーに置き換える
- [ ] 5.2 `internal/config/config_test.go` の `mode` のケースを、廃止のエラーを検証する 1 本に置き換える
- [ ] 5.3 `cmd/loop-cli/main.go` から `repoModes` を消し、`fetch.Fetch` の呼び出しと `ui.Options` から `modes` を落とす
- [ ] 5.4 `internal/gh/client.go` の `argsListLabels` の `--limit` を 1000 にし、`internal/gh/client_test.go` の期待する引数を直す

## 6. fixture に labels.json を足す

- [ ] 6.1 `internal/gh/testdata/fixtures/board/labels.json` を作る。ILD の語彙（`To Do` / `In Progress` / `Done` と `blocked` / `question`）で、`stage:todo` を含めない。これで `board` は `Fetch` から `label` と判定される
- [ ] 6.2 `internal/fetch/testdata` の `link` / `multirepo` / `partial` / `samestage` に `labels.json` を足す。`stage:todo` を含む sdd の語彙にして、既存のケースの期待値を変えない。`nosearch` は search で失敗して `ListLabels` まで進まないので足さない
- [ ] 6.3 `internal/fetch/fetch_test.go` の「全 alias について `Errors` が空」のケースが通ることを確認する

## 7. docs

- [ ] 7.1 `docs/mvp/decisions.md` に D-005（運用方式はリポジトリのラベル一覧から取得のたびに判定する）を足す。s26 の判断 1・2 を覆す理由と、当時の 3 つの懸念（`gh` 呼び出しが増える / 両方のラベルを持つリポジトリ / スナップショットに古い方式が残る）への答えを書き、変更履歴に 1 行足す
- [ ] 7.2 `docs/mvp/mvp.md` の設定ファイルの節から `mode` の行と説明を消し、0 件ヒントの文言を直し、「前提と未決事項」の `mode: label` に関する記述を「ラベル一覧から判定する」形に書き換え、変更履歴に 1 行足す
- [ ] 7.3 `docs/domain/issue-driven-sdd/human-turn-signals.md` の ILD の節を直す。方式の正本の文（「リポジトリ設定が正本で、ラベル集合からは判定しない」）、行 C と行 D の `Closes #n`、「ILD でキューに入れないもの」の PR の 2 規則、「方式の書き忘れ」の段落を書き換え、変更履歴に 1 行足す

## 8. 仕上げ

- [ ] 8.1 `loop-cli-dev classify --fixture example` の出力が変更の前後で一致することを確認する（`--mode` を引数で受けるので `Fetch` の変更の影響を受けない。成功条件 5）
- [ ] 8.2 `gofmt -l .` が空で、`go build ./... && go vet ./... && go test ./...` と `golangci-lint run ./...` が通ることを確認する
- [ ] 8.3 `openspec validate s30-mode-from-labels --strict` が通ることを確認する
