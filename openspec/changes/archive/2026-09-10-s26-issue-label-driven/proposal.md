## Why

loop-cli の分類器は issue-driven-sdd（`stage:*` + `wip` + PR 段階ラベル）の語彙に固定されている（`internal/model/model.go` 11-25 行の定数、`internal/classify/classify.go` 33 行の規則 1、110-121 行の `isC`）。
OpenSpec を挟まない **issue-label-driven**（以下 ILD。`To Do` / `In Progress` / `Done` の 3 ラベルだけで進む構成）のリポジトリを `repos` に入れても、キューにはほぼ何も出ない。

局面 A–G の意味（質問 / 方針 / merge / todo 候補 / 異常 / 進行中）は 2 つの方式で共通で、違うのはラベルの語彙と遷移の細部だけである。
1 本のキューに両方式を共存させ、同じ 4 タブ・同じキー操作・同じ優先度で捌けるようにする（#5）。

## What Changes

- `internal/config` の `repos` 要素に `mode: sdd | label` を足す（既定 `sdd`。既存の設定ファイルはそのまま動く）
- `internal/model` に `Mode` 型と ILD の語彙（`To Do` / `In Progress` / `Done`）を足し、`IssueStages` / `PRStages` / `IsMidRelabel` を方式で切り替える。`TodoLabel(mode)` と `ClosesIssue(body)` を足す
- **方式は値として持ち回らず、設定を正本にする。** `classify.Issue(is, mode)` / `classify.PR(pr, mode)` / `classify.Card(c, mode)` が引数で受け取り、`internal/fetch` と `internal/ui` は設定から作った `map[string]model.Mode` を引く。`model.Issue` / `model.PR` に方式のフィールドは足さない（スナップショットに古い方式が残り、設定と食い違ったラベルを書く経路を作らないため）
- 分類器を方式ごとに分岐させる
  - 規則 1（進行中）: sdd は「段階 1 件 + `wip`」、ILD は「`In Progress` 1 件かつ `question` 無し」（ILD は `In Progress` が段階と作業中の印を兼ねるので、`question` を除外しないと行 B に到達しない）
  - 規則 6（2 回書きの途中）: sdd は `release:` / `restart:` / `advance:`、ILD は `restart:` のみ
  - 規則 7（`ai-assess:requested`）と行 G（`docs`）: sdd のみ。ILD には対応するラベルが無い
  - 行 C（merge する）: ILD は「`Closes #n` を持つ open PR で、`question` 無し・mergeable・checks 緑」。本文 1 行目は読まない
  - 行 D（レビュー質問）: ILD は「`Closes #n` を持つ open PR の未 resolve thread」。**ILD では行 C より先に評価する**（C から 1 行目のゲートが外れる以上、C を先に見るとレビュー質問が緑の PR に埋もれる）
  - 行 A / B / E / F と規則 2〜5 は語彙が変わるだけで規則は共通
- `action.ToggleTodo` が方式に応じて `stage:todo` / `To Do` を付け外しする。拒否の条件（段階ラベル 2 つ以上 / 別の段階ラベル / `blocked`）は語彙を差し替えて共通
- カード詳細の段階行・バッジ・PR 一覧、ヘルプの `t` の文言、キュー 0 件のときの空表示メッセージを方式に合わせる
- **方式の書き忘れをフッタで知らせる。** sdd として扱っているリポジトリに `To Do` / `In Progress` の issue があれば `mode: label の設定漏れかもしれません` を出す。書き忘れると ILD の issue が局面 E に並び、`t` が `stage:todo` を書いてしまうが、0 件ヒントはカードが 1 件でもあれば出ないので他に伝える経路が無い
- `cmd/loop-cli-dev classify` に `--mode` を足し、ILD リポジトリの fixture（手書き）を `internal/gh/testdata/fixtures/` に足して期待値表で分類を固定する
- `docs/domain/issue-driven-sdd/human-turn-signals.md` に ILD 方式のシグナル対応表を足し、判定表の正本を 2 方式ぶんにする
- `docs/mvp/mvp.md` の「設定ファイル」節に `mode` を、「初回起動」節の空キューのヒントと「前提と未決事項」の「`routines-setup` を回していないリポジトリは空」に ILD 方式を反映する

**既存の sdd リポジトリの分類結果は 1 件も変えない。** `Mode` のゼロ値が `sdd` で、既存の分類器テスト（`internal/classify/classify_test.go`）は書き換えずに通る。加えて `loop-cli-dev classify --fixture example` の出力を変更の前後で突き合わせて、回帰が無いことを機械的に確かめる（#5 成功条件 6）。

ラベル名そのものを設定で変えられるようにはしない（mvp.md「ラベル名と routine マーカーはプラグインの規約に固定し、設定で変えられるようにしない」を維持し、方式 2 つ分の語彙だけを持つ）。

## Capabilities

### New Capabilities

（無し。既存 capability の要求を方式で切り替える）

### Modified Capabilities

- `config-loading`: `repos` 要素に `mode` キーを足し、`Repo` に解決済みの方式を持たせる
- `card-model`: `Mode` 型・ILD の語彙・`TodoLabel` / `ClosesIssue` を足し、`IssueStages` / `PRStages` が方式ごとの語彙を返すようにする
- `card-fetch`: `Fetch` がリポジトリごとの方式の表を受け取り、カードごとの方式で分類する
- `human-turn-classify`: `Issue` / `PR` / `Card` が方式を引数で受け取り、規則 1 / 6 / 7、行 C / D / E / F / G を方式で切り替える。fixture の期待値表に方式を持たせる
- `todo-action`: `ToggleTodo` が方式に応じて `stage:todo` / `To Do` を書く
- `todo-toggle`: `t` が設定由来の方式で `ToggleTodo` を呼び、フッタの文言をそのラベル名にする
- `card-detail`: 段階行・バッジ・PR 一覧を方式に合わせる
- `queue-screen`: 方式の表を `Options` で受け取り、空表示メッセージを 2 方式ぶんにし、方式の書き忘れをフッタで知らせる
- `help-screen`: `t` の説明
- `dev-cli`: `classify` の `--mode`

## Impact

- コード: `internal/config`、`internal/model`、`internal/fetch`、`internal/classify`、`internal/action/todo.go`、`internal/ui/{model,detail,help,view,todo}.go`、`cmd/loop-cli/main.go`、`cmd/loop-cli-dev/classify.go`
- シグネチャが変わる関数: `model.IssueStages` / `model.PRStages` / `model.IsMidRelabel` / `classify.Issue` / `classify.PR` / `classify.Card` / `fetch.Fetch` / `action.ToggleTodo`。いずれも内部 API で、呼び出し箇所はこの change の中で全部書き換える
- テストデータ: `internal/gh/testdata/fixtures/` に ILD の alias を 1 つ、`internal/action/testdata/todo` に ILD の issue を 4 件足す（実リポジトリが無いので手書き）
- ドキュメント: `docs/domain/issue-driven-sdd/human-turn-signals.md`、`docs/mvp/mvp.md`
- 設定ファイル: 後方互換。`mode` を書かないリポジトリは `sdd` のまま
- 依存・外部 API: 変更なし。方式の判定は設定から決まるので、リポジトリ 1 件あたりの `gh` 呼び出しは増えない。ILD リポジトリを `repos` に足すと、そのリポジトリの open issue / open PR の件数だけ詳細取得が増える（design.md の Risks を参照）

## 確定した判断

1. **方式はリポジトリ設定で持つ（自動判定しない）。** `repos` の要素に `mode: sdd | label`、既定 `sdd`。`internal/config/config.go` 50-77 行の `Repo.UnmarshalYAML` が既に `{name, merge_method}` のマッピングを受けるので、キーを 1 つ足すだけで済む。ラベル集合からの自動判定は `gh` 呼び出しが増え、両方式のラベルを持つリポジトリで曖昧になる（#5「未決」節の推奨に従う）。
2. **方式は引数で配り、データに持たせない。** `model.Issue` / `model.PR` にフィールドとして持たせると、スナップショット（前回起動時の派生データ）に古い方式が残り、設定を変える前の値で `t` がラベルを書く経路ができる。設定を唯一の正本にし、`internal/ui` は `Options.Modes`（`MergeMethods` と同じ経路）を引く。
3. **`Mode` のゼロ値は `sdd`。** 既存の分類器テストと `model.Issue` のリテラルを書き換えずに済み、回帰の判定が軽くなる。
4. **ILD の merge 禁止の印は `question` だけで、PR 本文 1 行目は読まない。** #5 のラベル表が `question` を「PR では merge 禁止の印」と定めており、成功条件 3 も `Closes #n` と checks 緑と mergeable だけを条件にしている。sdd の `未確定の判断: N 件` に相当する規約を ILD に仮定しない。
5. **ILD の「routine が作った PR」の印は `Closes #n` だけ。** #5 は「issue との紐付けは `Closes #n` だけ。issue 1 件 = PR 1 本」と定める。`internal/fetch` の `LinkedIssue`（title の段階記法と `Refs` も採る）はカード組み立て用の別判定なので、分類には使わず `model.ClosesIssue` を足す。紐づけ先の issue が実在するか・open かは見ない（分類は純粋関数であり、閉じた issue を閉じる PR も merge 対象として正しい）。
6. **ILD でも局面 D（レビュー質問）を出す。** 対象は `Closes #n` を持つ open PR。ILD の worker が review thread を使わなければ発火しないだけで、害が無い。出さない選択は、使っていた場合にレビュー質問が「その他」に埋もれる。
7. **`IsDraft` は sdd と同じく分類で見ない。** draft の merge 拒否は `action.CheckMerge`（`internal/action/merge.go:26`）が持つ。方式で扱いを変えると、同じ「draft を merge させない」規則が 2 か所に分かれる。
8. **`docs/mvp/mvp.md` を改訂する。** #5 が `internal/ui/view.go:146` の空表示メッセージの変更を明示しており、その文言と設定ファイルの正本が mvp.md にある（135 行・143 行・150 行）。製品の目的地そのもの（何を解く TUI か）は変えず、設定キーと文言の追随に限る。
9. **fixture は手書きで足す。** `internal/gh/capture.go` の採取は実在の ILD リポジトリを要求するが、今は存在しない。fixture は `gh` の出力形式の JSON なので手で書ける。`internal/classify/fixture_test.go` の期待値表は判定表を手で当てて書く規約（V-1）なので、手書き fixture でも規約に反しない。
10. **`action.CheckMerge` は変えない。** ILD の PR に `ai-assess:requested` は付かず、`model.ParseUndecided` は 1 行目が無ければ警告を出さない（`internal/action/merge.go:33`）。方式の分岐を足す必要がない。
11. **onboarding のフォームには `mode` を足さない。** フォームは `repos` を改行区切りの文字列で受けており、リポジトリごとの値を聞く形になっていない（`internal/onboarding/form.go:22`。`merge_method` も全体で 1 つ聞いてリポジトリ別は設定ファイルの手編集に委ねている）。ILD 利用者の初回起動は「設定漏れをフッタで知らせる」で拾い、フォームの作り替えはこの change に含めない。

判断 4〜6 は ILD の plugin（`sugiken-dev-plugins-public` の `issue-label-driven`）が未 push で読めないため、#5 の本文だけを根拠にしている。規約が違っていた場合に直す箇所は `internal/classify/classify.go` の 2 つの述語だけで、spec の該当 Requirement もそこに閉じている。

## 関連

- issue: #5
- 進行中の change: 無し（`openspec/changes/` 直下は空）
