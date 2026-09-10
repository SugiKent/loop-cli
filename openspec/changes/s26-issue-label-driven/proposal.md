## Why

loop-cli の分類器は issue-driven-sdd（`stage:*` + `wip` + PR 段階ラベル）の語彙に固定されている（`internal/model/model.go` 11-25 行の定数、`internal/classify/classify.go` 33 行の規則 1、110-121 行の `isC`）。
OpenSpec を挟まない **issue-label-driven**（以下 ILD。`To Do` / `In Progress` / `Done` の 3 ラベルだけで進む構成）のリポジトリを `repos` に入れても、キューにはほぼ何も出ない。

局面 A–G の意味（質問 / 方針 / merge / todo 候補 / 異常 / 進行中）は 2 つの方式で共通で、違うのはラベルの語彙と遷移の細部だけである。
1 本のキューに両方式を共存させ、同じ 4 タブ・同じキー操作・同じ優先度で捌けるようにする（#5）。

## What Changes

- `internal/config` の `repos` 要素に `mode: sdd | label` を足す（既定 `sdd`。既存の設定ファイルはそのまま動く）
- `internal/model` に `Mode` 型と ILD の語彙（`To Do` / `In Progress` / `Done`）を足し、`IssueStages` / `PRStages` を方式で切り替える。ゼロ値の `Mode` は `sdd` として扱う
- `model.Issue` / `model.PR` に `Mode` を持たせ、`internal/fetch` がリポジトリ名から引いて埋める。`classify.Issue` / `classify.PR` のシグネチャは変えない
- 分類器を方式ごとに分岐させる
  - 規則 1（進行中）: sdd は「段階 1 件 + `wip`」、ILD は「`In Progress`」
  - 規則 6（2 回書きの途中）: sdd は `release:` / `restart:` / `advance:`、ILD は `restart:` のみ
  - 規則 7（`ai-assess:requested`）と行 G（`docs`）: sdd のみ。ILD には対応するラベルが無い
  - 行 C（merge する）: ILD は PR に段階ラベルが無いので、別の条件で対象を絞る（未確定の判断 Q1 / Q2）
  - 行 D（レビュー質問）: ILD には `apply` ラベルが無い（未確定の判断 Q3）
  - 行 B / E / F と、行 A・規則 2〜5 は語彙が変わるだけで規則は共通
- `action.ToggleTodo` が方式に応じて `stage:todo` / `To Do` を付け外しする。拒否の条件（段階ラベル 2 つ以上 / 別の段階ラベル / `blocked`）は語彙を差し替えて共通
- カード詳細の「段階」行・バッジ・PR 一覧、ヘルプの `t` の文言、キュー 0 件のときの空表示メッセージを方式に合わせる
- `cmd/loop-cli-dev classify` に `--mode` を足し、ILD リポジトリの fixture（手書き）を `internal/gh/testdata/fixtures/` に足して期待値表で分類を固定する
- `docs/domain/issue-driven-sdd/human-turn-signals.md` に ILD 方式の列（またはシグナル対応表）を足し、判定表の正本を 2 方式ぶんにする
- `docs/mvp/mvp.md` の「設定ファイル」節に `mode` を、「初回起動」節の空キューのヒントに ILD 方式を書き足す

**既存の sdd リポジトリの分類結果は 1 件も変えない。** `Mode` のゼロ値が `sdd` なので、既存の fixture の期待値表・分類器のテストはそのまま通ることを回帰の判定に使う（#5 成功条件 6）。

ラベル名そのものを設定で変えられるようにはしない（mvp.md「ラベル名と routine マーカーはプラグインの規約に固定し、設定で変えられるようにしない」を維持し、方式 2 つ分の語彙だけを持つ）。

## Capabilities

### New Capabilities

（無し。既存 capability の要求を方式で切り替える）

### Modified Capabilities

- `config-loading`: `repos` 要素に `mode` キーを足し、`Repo` に解決済みの方式を持たせる
- `card-model`: `Mode` 型と ILD の語彙を足し、`Issue` と `PR` に `Mode` フィールドを持たせ、`IssueStages` と `PRStages` が方式ごとの語彙を返すようにする
- `card-fetch`: `Fetch` がリポジトリごとの方式を受け取り、`Issue` / `PR` の `Mode` を埋める
- `human-turn-classify`: 規則 1 / 6 / 7、行 C / D / G を方式で切り替える。fixture の期待値表に ILD の alias を足す
- `todo-action`: `ToggleTodo` が方式に応じて `stage:todo` / `To Do` を書く
- `card-detail`: 段階行・バッジ・PR 一覧を方式に合わせる
- `queue-screen`: キュー 0 件のときの空表示メッセージ
- `help-screen`: `t` の説明
- `dev-cli`: `classify` の `--mode`

## Impact

- コード: `internal/config`、`internal/model`、`internal/fetch`、`internal/classify`、`internal/action/todo.go`、`internal/ui/{detail,help,view,todo}.go`、`cmd/loop-cli/main.go`、`cmd/loop-cli-dev/classify.go`
- テストデータ: `internal/gh/testdata/fixtures/` に ILD の alias を 1 つ足す（実リポジトリが無いので手書き）
- ドキュメント: `docs/domain/issue-driven-sdd/human-turn-signals.md`、`docs/mvp/mvp.md`
- 設定ファイル: 後方互換。`mode` を書かないリポジトリは `sdd` のまま
- 依存・外部 API: 変更なし（`gh` の呼び出しは増えない。方式は設定から決まり、ラベル一覧の取得はしない）

## 確定した判断

1. **方式はリポジトリ設定で持つ（自動判定しない）。** `repos` の要素に `mode: sdd | label`、既定 `sdd`。`internal/config/config.go` 50-77 行の `Repo.UnmarshalYAML` が既に `{name, merge_method}` のマッピングを受けるので、キーを 1 つ足すだけで済む。ラベル集合からの自動判定は `gh` 呼び出しが増え、両方式のラベルを持つリポジトリで曖昧になる（#5「未決」節の推奨に従う）。
2. **`classify.Issue(is)` / `classify.PR(pr)` のシグネチャは変えない。** 方式は `model.Issue` / `model.PR` の `Mode` フィールドで運ぶ。`human-turn-classify` の「分類は純粋関数」Requirement と、`internal/classify` を呼ぶ 3 経路（`internal/fetch`、`cmd/loop-cli-dev classify`、fixture テスト）の形をそのまま保てる。
3. **`Mode` のゼロ値は `sdd`。** 既存の分類器テスト・fixture 期待値表・`model.Issue` のリテラルを書き換えずに済み、回帰（#5 成功条件 6）が「既存テストが無改変で通ること」で検証できる。
4. **`docs/mvp/mvp.md` を改訂する。** #5 が `internal/ui/view.go:146` の空表示メッセージの変更を明示しており、その文言と設定ファイルの正本が mvp.md にある（135 行・143 行）。製品の目的地そのもの（何を解く TUI か）は変えず、設定キーと文言の追随に限る。
5. **fixture は手書きで足す。** `internal/gh/capture.go` の採取は実在の ILD リポジトリを要求するが、今は存在しない。fixture は `gh` の出力形式の JSON なので手で書ける。`internal/classify/fixture_test.go` の期待値表は判定表を手で当てて書く規約（V-1）なので、手書き fixture でも規約に反しない。
6. **`action.CheckMerge` は変えない。** ILD の PR に `ai-assess:requested` は付かず、`model.ParseUndecided` は 1 行目が無ければ `false` を返して警告を出さない。方式の分岐を足す必要がない。

## 未確定の判断

### Q1. ILD で「merge していい PR」をどう見分けるか（局面 C の対象）

sdd は `propose` / `apply` / `archive` のどれかが付いた PR だけを merge 行に出している（`classify.go:110` の `isC`）。ILD の PR には段階ラベルが付かないので、この絞り込みが無くなる。

- 選択肢 A（推奨）: `Closes #n` で open issue に紐づく PR だけを候補にする。Dependabot や外部からの PR は今までどおり `[1]今やる` の「その他」行に出る（消えはしない）
- 選択肢 B: open PR すべてを候補にする。`question` が無く mergeable で checks 緑なら merge 行に出るので、Dependabot の PR も「merge する」として並ぶ
- 依存: なし

### Q2. ILD の PR 本文 1 行目に `未確定の判断: N 件` の規約はあるか

sdd では 1 行目が `未確定の判断: 0 件` でない PR を merge 行に出さない（`isC` の `ParseUndecided`）。ILD の worker がこの行を書くかどうかが、この change の外（未 push の plugin）でしか分からない。

- 選択肢 A（推奨）: ILD では 1 行目を読まない。merge 禁止の印は `question` ラベルだけにする。ILD の worker がこの行を書いていても、`question` と二重にはならないので実害は無い
- 選択肢 B: sdd と同じ規約がある前提にする。1 行目が無い PR は merge 行に出ない。ILD の worker がこの行を書いていない場合、ILD リポジトリの merge 行は**常に空**になる
- 依存: なし。Q1 でどちらを選んでも、この問いは行 C に「1 行目を読む」条件を足すかどうかを別に決める

### Q3. ILD で局面 D（レビュー質問に答える）を出す対象

sdd は `apply` ラベルの PR に限って、未 resolve の review thread の最終コメントが AI なら D（優先度 1）に出す（`classify.go:124`）。ILD にはこのラベルが無い。

- 選択肢 A（推奨）: `Closes #n` で紐づく open PR すべてを対象にする。AI が review thread で聞いてきた質問が、sdd と同じ優先度で `[1]今やる` に浮上する
- 選択肢 B: ILD では D を出さない。未解決の review thread は「その他」に落ち、レビュー質問だけが人の出番として見えなくなる
- 依存: なし

## 関連

- issue: #5
- 進行中の change: 無し（`openspec/changes/` 直下は空）
