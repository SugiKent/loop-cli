## Context

分類器の語彙は `internal/model/model.go` の定数と `IssueStages` / `PRStages` に集約されているが、**規則そのもの**も sdd の状態機械に結びついている（`classify.go:33` の「段階 1 件 + `wip`」、`classify.go:110` の `isC` が段階ラベルを要求する、`classify.go:125` の `isD` が `apply` を要求する）。issue-label-driven（以下 ILD）は別のラベル名を使い、`wip` を持たず、PR に段階ラベルを付けない。後ろの 2 点は規則の差なので、語彙の差し替えだけでは足りない。

制約は 4 つある。

- 既存の sdd リポジトリの分類結果を 1 件も変えられない（#5 成功条件 6）
- 分類は純粋関数である（`human-turn-classify`「分類は純粋関数で…」）。方式を知るために I/O を足せない
- 書き込み（`t`）は設定と食い違ってはいけない。TUI は前回起動時のスナップショット（`internal/snapshot`）を初回取得の前に表示し、その間も `t` を受け付ける（`internal/ui/model.go:104`、`internal/ui/todo.go:24`）
- ILD の plugin（`sugiken-dev-plugins-public` の `issue-label-driven`）は未 push で、規約は #5 の本文に写された範囲しか読めない

背景と目的は proposal.md「Why」を参照。

## Goals / Non-Goals

**Goals:**
- 2 方式のリポジトリを 1 本のキューに混ぜても、タブ・優先度・キー操作が方式によらず同じに見える
- 方式の分岐点をコードの少数の箇所に閉じ込め、どちらの方式でも同じ関数を通す
- 方式の取り違えが「静かに間違ったラベルを書く」形で終わらないようにする

**Non-Goals:**
- ラベル名を設定で変えられるようにすること（mvp.md の固定方針を維持する）
- リポジトリのラベル集合から方式を自動判定すること
- ILD 向けに新しいキー操作や画面を足すこと。既存の 4 タブとキーをそのまま使う
- onboarding のフォームをリポジトリごとの設定を聞ける形に作り替えること（proposal.md 確定した判断 11）
- 実在の ILD リポジトリでの実測。fixture と単体テストで閉じる

## Decisions

### 方式は設定を正本にし、引数で配る

`classify.Issue(is, mode)` / `classify.PR(pr, mode)` / `classify.Card(c, mode)`、`fetch.Fetch(ctx, client, repos, modes)`、`action.ToggleTodo(ctx, client, repo, number, mode)` の形にする。`internal/ui` は `Options.Modes map[string]model.Mode` を持ち、表示にも書き込みにも `m.modes[repo]` を使う（`MergeMethods` と `internal/ui/merge.go:90` の `mergeMethod` が先例）。

代替案は `model.Issue` / `model.PR` に `Mode` フィールドを持たせて取得時に埋める設計で、分類器のシグネチャを変えずに済む。これを採らない理由は 2 つある。第 1 に、スナップショットは前回の実行時の `Cards` をそのまま復元するので、`mode: label` を書く前に保存された Card は方式が空（= sdd）のまま画面に出る。初回取得が終わる前に `t` を押すと ILD リポジトリへ `stage:todo` を書く。第 2 に、`Card` は 1 リポジトリ 1 枚なので、`classify.Card` が受け取って中の要素に配れば済み、引数を増やす負担が小さい。

`Mode` のゼロ値 `""` は `sdd` として扱う。既存のテストが書く `model.Issue{Labels: …}` のリテラルと `classify_test.go` は無改変で通り、回帰の確認が軽くなる。

### 語彙は `model` の関数が方式で切り替え、規則の分岐は `classify` に置く

`IssueStages(mode, labels)` / `PRStages(mode, labels)` / `TodoLabel(mode)` / `IsMidRelabel(mode, comments)` が語彙を持ち、呼び出し側は方式ごとの定数を直接参照しない。規則の差（進行中の規則 1、行 C / D の対象と評価順、行 G と規則 7 の有無）は `classify.Issue` / `classify.PR` の中の分岐で書く。

方式ごとに分類器を 2 本作る案は採らない。局面 A / B / E / F と進行中の規則 2〜5 は完全に共通で、2 本にすると共通部分を二重に持つことになる。

### ILD の規則 1 は `question` の付いた issue を除外する

sdd の規則 1（AI が作業中）は段階ラベルに加えて `wip` を要求し、worker がブロックしたときに `wip` が外れるので、`blocked` + `question` の issue は行 B（方針を決める）に届く。ILD には `wip` が無く `In Progress` が段階と作業中の印を兼ねるため、そのまま移すと `In Progress` + `blocked` + `question` の issue が常に進行中へ落ち、**ILD リポジトリの「方針を決める」が恒久的に空になる**。規則 1 に「`question` が無い」を足して、行 B と進行中の規則 4（回答済み）に届くようにする。

「ブロック中に `In Progress` が残るか」は #5 に書かれていないが、この形ならどちらでも正しく動く（残るなら `question` の除外が効き、外れるなら規則 1 に当たらない）。

### ILD では行 D を行 C より先に評価する

sdd の行 C は本文 1 行目 `未確定の判断: 0 件` をゲートに使うので、レビュー質問が残っている PR は C に当たらず D に届く。ILD にはこの 1 行目の規約が無く（proposal.md 確定した判断 4）、C の条件が「紐づき + `question` 無し + 緑」だけになる。C を先に評価すると、未 resolve の AI レビュー質問がある緑の PR が常に「merge する」になり、D が checks の赤い PR にしか出なくなる。優先度は D が 1、C が 3 なので、評価順を入れ替える方が表とも整合する。

### ILD の「routine が作った PR」は `Closes #n` で見分ける

`internal/fetch` の `LinkedIssue` は title の `[propose] #n` と本文の `Refs` / `Closes` を採る、カード組み立て用の判定である。ILD の紐づけは `Closes #n` だけなので（#5）、分類には `model.ClosesIssue(body)` を新しく足して使う。`LinkedIssue` を `internal/model` へ移して共用する案も検討したが、判定の意味が違う（片方は「どの issue のカードに置くか」、もう片方は「routine が作った PR か」）ので、名前を分けて別の関数にする。`internal/fetch` の API と `internal/ui/detail.go` の呼び出しも変えずに済む。

### 方式の書き忘れをフッタで知らせる

`mode: label` を書き忘れた ILD リポジトリは、issue が段階ラベル無しに見えて局面 E（着手を承認する）に並び、`t` が `stage:todo` を書く。0 件ヒントはカードが 1 件でもあれば出ないので、この状態は他の経路では人に伝わらない。取得済みの `Cards` のラベルだけで判定でき（追加の `gh` 呼び出しが要らない）、フッタのステータスは既にスピナー・エラー・部分失敗で共有されているので、そこに 1 つ足す。

### 画面が方式を見る箇所

`detail.go` の `cardHeaderLines` が段階行を組み立てるところ、同じ関数がバッジを並べるところ、`prListLines` が PR 一覧の見出しを出すところ、`help.go` が `t` の説明を書くところ、`view.go` がキュー 0 件のヒントと設定漏れの知らせを出すところ、`todo.go` が書くラベル名をフッタに出すところ。PR 詳細ヘッダの段階（`detail.go:316`）は `model.PRStages` が空を返すので `-` になり、分岐は要らない。
PR 一覧と PR 詳細の「1 行目なし」の列は ILD でも出す。ILD の PR に 1 行目の規約が無いことは事実であり、列を方式で落とすと同じ情報の出し方が 2 通りになる。

### ILD の fixture は手書きで置く

`gh.Capture` は実在のリポジトリを要求するが、ILD で運用中のリポジトリがまだ無い。fixture は `gh` の出力形式の JSON なので手で書ける。`internal/gh/fixtures_test.go:44` が `login` を `user-N` 形式に限り、`internal/fetch/fetch_test.go:423` が全 alias について `Errors` が空であることを要求するので、issue ごとに `issue-<n>.json`、PR ごとに `pr-<n>.json` と `pr-<n>-review-threads.json` を揃える。

## Risks / Trade-offs

- **ILD の plugin が未 push で、規約の一部を #5 の本文から読んでいる** → 影響は `classify` の 2 つの述語（行 C / D の対象条件）に閉じており、spec でもその Requirement に閉じている。規約が違っていたら、そこだけを直す
- **手書き fixture が実データとずれる** → 実リポジトリでの確認は別 issue に切り出す。この change の完了条件は「#5 に書かれた規約を分類器が再現すること」に閉じる
- **`To Do` にはスペースが含まれる** → `gh issue edit --add-label "To Do"` はサブプロセスの引数として渡るのでシェルのクォートは要らないが、`Fake` の記録と突き合わせるテストで実際の引数を確認する
- **ILD リポジトリを足すと 1 回の更新の `gh` 子プロセスが増える** → `internal/fetch/fetch.go:99` は open の全 issue / 全 PR に詳細取得を投げ、`internal/gh/client.go:87` の search は全リポジトリ合計 200 件で切る。カンバン運用の ILD リポジトリはバックログが大きくなりやすく、既存 sdd の項目が 200 件の上限で押し出されることがある。この change では上限も取得方法も変えず、実際に詰まったら別 issue で扱う
- **ILD の draft PR が merge 行に出ることがある** → 分類は `IsDraft` を見ない（proposal.md 確定した判断 7）。sdd と同じ振る舞いで、merge そのものは `CheckMerge` の拒否で止まる
- **シグネチャが変わる関数が 8 つある** → いずれも内部 API で、production の呼び出しは `IssueStages` 3 か所・`PRStages` 6 か所・`IsMidRelabel` 1 か所・`classify` 系 4 か所・`Fetch` 1 か所・`ToggleTodo` 1 か所である。テスト側（`model_test.go` / `parse_test.go` / `todo_test.go` / `fixture_test.go` / `snapshot_test.go` / `testdata_test.go` / `main_test.go`）も併せて書き換える。「既存テストを 1 行も書き換えずに通す」ことは成立しないので、回帰の根拠は `loop-cli-dev classify --fixture example` の出力を変更前後で突き合わせる形で置く

## Migration Plan

設定ファイルは後方互換で、`mode` を書かない既存の `config.yml` は `sdd` として動く。データの移行も再取得も要らない。

## Open Questions

無し。#5 に書かれていない ILD の規約（`In Progress` がブロック中に残るか、worker が review thread を使うか）は、どちらの答えでも同じ挙動になる形に設計を寄せた（Decisions の該当節）。
