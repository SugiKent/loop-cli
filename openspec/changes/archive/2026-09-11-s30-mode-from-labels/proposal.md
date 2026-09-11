issue: #24

## Why

リポジトリの運用方式（issue-driven-sdd か issue-label-driven か）は、いま設定ファイルの `repos[].mode` が正本である
（`internal/config/config.go` の `Repo.Mode`、`cmd/loop-cli/main.go:176` の `repoModes`）。これには 2 つの問題がある。

1. **方式を手で書く必要があり、書き忘れると間違ったラベルを書く。** `mode: label` を書き忘れた issue-label-driven（以下 ILD）の
   リポジトリは `sdd` として扱われ、`t` が `stage:todo` を書く。今は `internal/ui/view.go:251` の
   `mode: label の設定漏れかもしれません` が知らせるだけで、書き込みは止まらない
2. **ILD でも「今やる」に出ない open PR がある。** `internal/classify/classify.go` の `PR` は、最新コメントが人の PR を
   `question` の有無にかかわらず `[3]進行中` に入れ（規則 2 / 3）、行 C（merge する）は `Closes #n` を必須にしている。
   ILD の PR は 1 issue = 1 PR で人が捌く前提なので、キューから外れると人の出番が見えなくなる

方式はリポジトリのラベル一覧から機械的に分かる（`stage:todo` があるかどうか）。設定に書かせるのをやめれば、書き忘れという
状態そのものが消える。s28 で `GHClient.ListLabels` は既に存在し（`internal/gh/client.go:280`）、判定に必要な材料は揃っている。

## What Changes

### 1. 方式を取得のたびにラベル一覧から判定する

- `internal/model` に `ModeFromLabels(labels []gh.RepoLabel) (Mode, bool)` を足す。`stage:todo` があれば `sdd`、
  `To Do` があれば `label`、どちらも無ければ `ok` が偽（判定できない）
- `fetch.Fetch` は `modes` 引数をやめ、起動時・`R`・自動更新の取得のたびに、リポジトリごとに `ListLabels` を詳細取得と同じ
  並行プールで呼ぶ。判定結果は `Result.Modes map[string]model.Mode` に入れ、その表でカードを分類する
- `ListLabels` が失敗したリポジトリと、どちらのラベルも持たないリポジトリは `Modes` に入れない。失敗は `Result.Errors` に積む
- `internal/ui` は `Options.Modes` をやめ、取得が成功したら `Result.Modes` で方式の表を丸ごと差し替える。取得が失敗したら
  前回の表を残す（D-002 と同じ扱い）
- **方式が分からないリポジトリ（初回取得の前、`ListLabels` の失敗、どちらのラベルも無い）では `t` は何も書かず、
  フッタに理由を赤で出す。** 推測したラベルは書かない
- 方式は設定ファイルにもスナップショットにも保存しない。`repos[].mode` と `repoModes` と `view.go:251` の設定漏れの警告を消す

s26-issue-label-driven の proposal「確定した判断 1・2」（方式は設定で持ち、ラベル集合からは判定しない）を覆す。
理由と、当時の懸念への答えは `docs/mvp/decisions.md` の D-005 に書く。

### 2. label 方式では open PR をすべて「今やる」に出す

`classify.PR` の `label` の分岐を次の順に変える（`sdd` の分類結果は 1 件も変えない）。

1. `question` あり → 行 A（質問）
2. 未 resolve の review thread があり、その thread の最終コメントが AI → 行 D（レビュー質問）
3. `question` が無く、`Mergeable` が `MERGEABLE` で checks が緑 → 行 C（merge）
4. それ以外 → その他

`label` では進行中の規則 2 / 3（最新コメントが人 → 進行中）を適用しない。行 C と行 D の対象の絞り込みから `Closes #n` を外す。
`その他` の `Tab` は `今やる` なので、これで ILD の open PR は全件が `[1]今やる` に並ぶ。

### 3. `ListLabels` の取得上限

`gh label list` の `--limit` を 100 から 1000 に上げる。100 のままだと、ラベルが 100 件を超えるリポジトリで名前昇順の後ろにある
`stage:todo` が切れ、`label` と誤判定して `To Do` を書きに行く経路ができる。

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `card-model`: 方式の正本が「設定」から「リポジトリのラベル一覧」に変わる。ラベル一覧から方式を導く `ModeFromLabels` を持つ
- `card-fetch`: `Fetch` から `modes` 引数が消え、リポジトリごとの `ListLabels` と `Result.Modes` が加わる。`ListLabels` の失敗が部分失敗に加わる
- `card-detail`: ヘッダの段階行・バッジと PR 一覧が引く方式が、`Options.Modes` から取得結果の表に変わる
- `gh-client`: `ListLabels` の `--limit` が 100 から 1000 になる
- `human-turn-classify`: `label` の差分から `Closes #n` の絞り込みが外れ、進行中の規則 2 / 3 を適用しない差分が加わる
- `queue-screen`: `Options.Modes`（設定由来の表）と「方式の取り違えらしき状態をフッタで知らせる」が無くなり、取得結果から方式の表を持つ Requirement に置き換わる。0 件ヒントの 2 行目が変わる
- `todo-toggle`: `t` が引く方式が取得結果の表になり、方式が分からないリポジトリでは書き込まずに理由を出す
- `config-loading`: `repos[].mode` が無くなる
- `label-picker`: `L` のラベルの表を、取得が成功するたびに捨てる

## Impact

- コードは 7 パッケージを触る。`internal/model/model.go` に `ModeFromLabels` を足し、`internal/fetch/fetch.go` に
  `ListLabels` の並行取得と `Result.Modes` とエラーの並びを入れ、`internal/classify/classify.go` の `PR` にある `label` の
  分岐を書き換える。`internal/ui` では `model.go` から `Options.Modes` を落として取得結果の表を持ち、`todo.go` に拒否の枝を足し、
  `view.go` から設定漏れの警告を消して 0 件ヒントの文言を直す。`internal/config/config.go` から `mode` を落とし、
  `cmd/loop-cli/main.go` から `repoModes` を消して `Fetch` の呼び出しを直し、`internal/gh/client.go` の `--limit` を上げる
- テストデータには `labels.json` を足す。`Fetch` を通す fixture がすべて対象で、`internal/gh/testdata/fixtures/board` には
  ILD の語彙を、`internal/fetch/testdata` の `link` / `multirepo` / `partial` / `samestage` には `stage:todo` を含む sdd の
  語彙を置く。`internal/fetch/fetch_test.go` は全 alias について `Errors` が空であることを要求しているので、
  足さないとテストが落ちる
- `internal/gh` の `Fake` は変えない。「ディレクトリ 1 つが 1 リポジトリ」の不変条件を保ち、リポジトリごとに違うラベルを返す
  必要があるテスト（`Fetch` の方式の振り分け）は `fetch_test.go` のスタブ `GHClient` で書く
- ドキュメントは 3 つを直す。`docs/mvp/decisions.md` に D-005 を足し、`docs/mvp/mvp.md` では設定ファイルの節と 0 件ヒントと
  「前提と未決事項」を書き換え、`docs/domain/issue-driven-sdd/human-turn-signals.md` の ILD の節では方式の正本・行 C と行 D の
  条件・進行中の規則・方式の書き忘れの段落を書き換える
- 人から見た変化: `config.yml` から `mode` を消す（消さないと起動せず、専用のエラー文で理由が出る）。ILD のリポジトリで `t` が
  `To Do` を書くのに設定が要らなくなる。GitHub 側でラベルを足すと再起動なしで方式が変わる。ILD の open PR が全件キューに出る
- `gh` 呼び出しは 1 回の更新でリポジトリ数ぶん増える（`search` の枠は使わない）。`loop-cli-dev classify` は `--mode` を
  引数で受けるので変わらない

## 確定した判断

- **判定規則は「`stage:todo` があれば `sdd`、無くて `To Do` があれば `label`」とする。** 両方あるリポジトリは `sdd` に倒す。
  `stage:todo` を先に見るのは、issue-driven-sdd の routines-setup が作るラベルであり、ILD 側に `stage:todo` を作る理由が無いためである。
  `To Do` は Projects のカンバンでよく使う名前なので、issue-driven-sdd のリポジトリが別の目的で持っていることがある。
  両方あるときに `label` へ倒すと、そのリポジトリを ILD と誤判定して `To Do` を書いてしまう
- **`t` が `label` 方式で書くラベル名は `To Do` とする。** 上流の issue-label-driven（`routine-common`）が `To Do` を定め、
  work の Routine は `IN [To Do]` で起動する。現行の `model.TodoLabel(ModeLabel)` も `To Do` を返す（`internal/model/model.go:229`）。
  小文字の `todo` は worker を起動しないので採らない
- **方式が分からないリポジトリでも、分類と表示は `sdd` として行う。** 止めるのは書き込みだけである。`Card.Result` は必ず
  1 つ決まらないとカードが並べられず、`sdd` の語彙で見た結果は「段階ラベルが無い」= 行 E になるだけで、誤ったラベルを
  書く経路にはならない。`推測したラベルは書かない` は `t` に対する規則として実装する
- **部分失敗のときも表は差し替える。** `ListLabels` が失敗したリポジトリは取得成功のたびに表から落ち、`t` が拒否される。
  「前回の値を残す」は取得そのものが失敗したとき（`Fetch` がエラーを返したとき）だけに適用する。失敗しているリポジトリに
  古い方式でラベルを書くのは、この change が消したい経路そのものである
- **`Result.Errors` の並びは、リポジトリのエラー（リポジトリ名順）→ issue のエラー → PR のエラー とする。**
  `ListLabels` の失敗は特定の issue / PR に属さないので、番号を持つエラーより前に置く
- **`Fetch` は `Result.Modes` に方式だけを入れ、`Card` にも `Issue` / `PR` にも方式を持たせない。**
  `card-model` の「方式は `model.Issue` / `model.PR` のフィールドとしては持たない」は変えない。スナップショットにも入れない
  ので、前回の実行時の方式で書き込む経路は生まれない
- **`Fake`（`internal/gh`）は変えない。** 「`repo` 引数はファイルの探索に使わない（ディレクトリ 1 つが 1 リポジトリに対応する）」は
  `gh-fake` の不変条件である。リポジトリごとに違うラベルを返すテストは `fetch_test.go` のスタブで書く（`fetch_test.go` は既に
  呼び出し回数を数えるスタブ `GHClient` を持つ）
- **`--limit` は 1000 にする。** 100 件で切れると `stage:todo`（名前昇順で後ろ）が落ちて `label` と誤判定し、`To Do` を
  書きに行く。ページングは入れない（1000 件を超えるラベルを持つリポジトリは想定しない）
- **（人の回答 Q1: A）`L` のラベルの表は、取得が成功するたびに捨てる。** 次に `L` を押したときに取り直すので、GitHub 側で
  ラベルを作れば次の更新のあとの `L` に出る。方式は毎回の取得で変わるのに `L` の一覧だけが起動時のまま古く残る、という
  食い違いをここで消す。ラベル一覧そのものを `fetch.Result` に載せる形（`gh` 呼び出しは増えないが `label-picker` の Scenario を
  10 本前後書き直す）は、得られるものに対して書き直しが大きいので採らなかった
- **（人の回答 Q2: A）`stage:todo` も `To Do` も持たないリポジトリは「判定できない」として扱い、`t` を拒否する。**
  フッタに赤で `<Repo> の運用方式が分かりません（stage:todo / To Do のラベルがありません）` を出す。`label` に倒して
  `gh` の「そんなラベルは無い」を人に見せる形は、`t` を押すまで理由が分からず、文言も `gh` 任せになるので採らなかった。
  0 件ヒントの 2 行目も、設定を促す文からラベルを作ることを促す文へ直す
- **（人の回答 Q3: A）既存の `config.yml` の `mode:` は専用のエラー文で起動を止める。**
  `repos の mode は廃止しました。運用方式はリポジトリのラベル一覧から判定するので、この行を削除してください` を返す。
  汎用の未知キーのエラーだけでは、値が正しいのになぜ止まるのかが読み取れない。読み捨てて起動する形は、書いた値が黙って
  無視されるので採らなかった
- **（人の回答 Q4: A）`label` 方式の行 D からも `Closes #n` の絞り込みを外す。** 行 C とそろい、`Closes` を書き忘れた PR や
  外部から来た PR でも、未 resolve の AI のレビュー質問が残っていれば「質問」（優先度 1）として上に出る

## 延期した判断と残るリスク

- **`sdd` 側の PR 分類は変えない。** 進行中の規則 2 / 3・`ai-assess:requested`・`docs` の扱いはそのままで、
  `loop-cli-dev classify --fixture example` の出力は変更の前後で一致する
- **ラベル一覧の取得をリポジトリ単位でキャッシュしない。** 毎回の更新で全リポジトリぶん取る。監視対象は数リポジトリで、
  `gh label list` は search の枠を使わない。取得が重くなったら別 issue で扱う
- **判定に使うのはラベルの存在だけで、Routines の設定や `openspec/` の有無は見ない。** `stage:todo` ラベルだけ作って
  Routines を回していないリポジトリは `sdd` と判定される。ラベルより確かな材料は `gh` から安く取れない
- **初回取得の前は、スナップショットから描いたカード詳細の段階行が `sdd` の語彙になる。** いまは設定から方式が分かるので
  ILD のカードは `段階: In Progress` と出るが、この change の後は最初の取得が終わるまで `段階なし` になる。取得は起動直後に
  走るので数秒で解消する。方式をスナップショットに持たせないという判断（上記）の代償として受け入れる
- **`t` の拒否は「押したときに分かる」形にとどめる。** 方式が分からないリポジトリのカードに印を付けることはしない
  （表の 1 行に載せる情報を増やすと、s08 が決めた行の構成を作り直すことになる）
