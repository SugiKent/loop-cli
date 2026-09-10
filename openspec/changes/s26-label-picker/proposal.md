## Why

今の TUI が書けるラベルは `t` の `stage:todo` だけである（`internal/ui/todo.go`）。それ以外のラベル—— `docs` を付け忘れた PR、プロジェクト固有の分類ラベル、`ai-assess:requested` の付け直し——を触るには、ブラウザで GitHub を開くか別の端末で `gh` を打つしかない。loop-cli は「人の出番」を捌く手数を減らす道具なので、ラベル 1 つのために画面を離れさせるのは目的を損なう。

キーを 1 つ（`l`）で「使えるラベルの一覧を出して選ぶ」形にすれば、ラベルの種類が増えてもキーは増えない。`u`（s22 `url-picker`）が URL に対して採った形と同じである。

Refs #3

## What Changes

- `l` を押すと**ラベル一覧画面**（キュー / カード詳細 / PR 詳細 / 確認 / ヘルプ / URL 一覧 に続く 7 つ目の画面の状態）に移り、対象リポジトリのラベルを並べる。`j` / `k` / `↑` / `↓` で選び、`Enter` で決定、`Esc` で元の画面に戻る
- 対象は `a` / `u` と同じ規則（画面が見せているもの）で決まる。キュー画面は選択行の主体、カード詳細は `Card.Issue`、PR 詳細は選択中の PR。`t` と違い PR も対象になる
- `Enter` で決定したラベルは、対象に付いていれば外し、付いていなければ付ける（`t` と同じトグル。確認画面は出さない）。付けるか外すかは書き込みの直前に GitHub から読み直したラベルで決める（`action.ToggleTodo` と同じ理由。画面の Card のラベルは最終取得時点のもの）
- 一覧では、対象に今付いているラベル（画面の Card が持つラベル）に印を出す
- 結果はフッタのステータスに出す。`Cards` は書き換えず、反映は `R` / 自動更新に任せる（`t` と同じ）
- 書き込み中（`m.writing`）は `l` を受け付けない。ラベル一覧の取得中も書き込み中として扱い、`l` / `t` / `a` / `m` を重ねさせない（s14 `merge-pr` が「取り直し中」を書き込み中に含めたのと同じ）
- `GHClient` に `ListLabels(ctx, repo)` を足す。`Client` は `gh label list --repo <repo> --json name,description,color --limit 100` を実行する。取得したラベルは `Model` がリポジトリ名をキーにプロセス内で保持し、同じリポジトリの 2 回目以降は `gh` を呼ばない
- 付け外しは既存の `AddLabel` / `RemoveLabel`（`gh issue edit --add-label` / `--remove-label`）をそのまま使う。PR にもそのまま使える（後述「確定した判断」）
- ヘルプ画面のキー一覧に `l` の行を足す
- `docs/mvp/mvp.md` のキーバインド表で `l` に当たっている「カンバンの列移動」との衝突を解く（未確定の判断 Q2）

## Capabilities

### New Capabilities

- `label-picker`: `l` でラベル一覧画面を開き、対象リポジトリのラベルを（対象に付いているかの印つきで）並べ、選んだ 1 つを対象に対してトグルする。対象の決め方、書き込み直前の読み直し、取得の失敗と書き込みの失敗のステータス表示、リポジトリ単位のキャッシュを含む

### Modified Capabilities

- `gh-client`: `GHClient` interface に `ListLabels(ctx, repo)` を足し、`Client` が `gh label list` として実行することを定める
- `gh-fake`: `Fake` が fixture ディレクトリの `labels.json` から `ListLabels` を返すことを定める
- `fixture-capture`: `Capture` が `labels.json` を採ることを定める
- `queue-screen`: キュー画面のキーの扱いで、`l` を「s17 が担当する未実装のキー」からラベル一覧画面を開くキーに変える
- `card-detail`: カード詳細 / PR 詳細のキーの扱いに、`l` がラベル一覧画面を開く行を書き足す
- `help-screen`: ヘルプ画面のキー一覧に `l` の行を足す

（`queue-screen` / `card-detail` のフッタのキーヒントを変えるかどうかは未確定の判断 Q3 に依る。`docs` の扱いは Q1 に依る）

## Impact

- `internal/gh/gh.go` / `client.go` / `fake.go` / `capture.go`: `ListLabels` と `Label` 型、`labels.json` fixture
- `internal/gh/testdata/fixtures/example/labels.json`: 新規 fixture
- `internal/ui`: ラベル一覧画面を新しいファイルに足し、`Update` のキー振り分け（URL 一覧と同じく `a` / `t` / `m` / `o` / `?` より前）と `View` の画面分岐を広げる
- `internal/action`: ラベル 1 つを読み直してトグルする関数（`ToggleTodo` の隣）。段階ラベルの判定は持たせない / 持たせる は Q1 に依る
- `docs/mvp/mvp.md`: キーバインド表（`l` の行、Q2 の結論）
- `docs/domain/issue-driven-sdd/human-turn-signals.md`: 不変条件 2「TUI が書くラベルは `stage:todo` と `stage:propose` の 2 つに限る」の改訂（Q1 の結論）
- 先行 change との関係: 進行中の change は無く（`openspec/changes/` 直下は archive のみ）、open PR も無い。`l` は s17（カンバンビュー `v` / `h` / `l` / `←` / `→`）が予約しているキーで、s17 は未着手

## 確定した判断

- **`gh issue edit` は PR 番号でも動く。** `gh` の `issue edit` は `issueOrPullRequest(number:)` で対象を解決し、`UpdateIssue(..., issue.IsPullRequest(), ...)` を呼ぶ（cli/cli `pkg/cmd/issue/shared/lookup.go` の `FindIssueOrPR`、`pkg/cmd/issue/edit/edit.go:360,494`）。したがって PR にラベルを付けるために `gh pr edit` の経路を新設する必要はなく、既存の `AddLabel` / `RemoveLabel`（`internal/gh/client.go:283-291`）をそのまま使える
- **一覧は 7 つ目の画面の状態として全画面で描く。** 依頼の言葉は「モーダル」だが、このコードベースに重ね合わせの前例は無く、確認画面（`renderConfirm`）・ヘルプ画面（`renderHelp`）・URL 一覧（`renderURLs`）はすべて全画面である（s22 design.md D-1 と同じ判断）
- **一覧画面の分岐は `a` / `t` / `m` / `o` / `?` より前に置く。** `internal/ui/model.go:220-260` が確認画面・ヘルプ画面・URL 一覧をこの位置で分岐しており、後ろに置くと一覧の裏の対象へ書き込みが起きる
- **トグルの判定は書き込み直前に GitHub から読み直す。** `action.ToggleTodo`（`internal/action/todo.go:22-49`）が `ViewIssue` で読み直しているのと同じ理由で、画面のラベルは最終取得時点のものであり、続けて 2 回押した 2 回目が「もう一度付ける」になってしまう。対象が PR のときは `ViewPR` で読む
- **一覧の「付いている」印は画面の Card のラベルで出す。** 押した時点で読み直すと `l` の反応が鈍り、依頼の「キャッシュしてよい（毎回叩くと反応が鈍い）」に反する。印は古くなり得るが、実際の付け外しは読み直した結果で決まるので誤操作にはならない
- **不変条件 1（1 操作 1 ラベル）は守る。** `--add-label` / `--remove-label` を 1 ラベルずつ呼び、ラベル集合の置換は行わない

## 未確定の判断

### Q1. `l` で書けるラベルの範囲

`docs/domain/issue-driven-sdd/human-turn-signals.md` の不変条件 2 は「**TUI が書くラベルは `stage:todo` と、`s` の強制操作で付ける `stage:propose` の 2 つに限る**。`blocked` / `question` / `wip` / `stage:apply` / `stage:archive` は書かない」と定めている。理由は「人がラベルを触らない前提で dispatcher が状態機械を回している」ことである。任意のラベルをトグルできる `l` はこの不変条件と正面から衝突する。

routine が状態機械に使うラベルは `stage:todo` / `stage:propose` / `stage:apply` / `stage:archive` / `wip` / `blocked` / `question` / `propose` / `apply` / `archive` / `docs` / `ai-assess:requested`。これらを人が触ると、worker が起動する・段階が飛ぶ・merge で段階が進むといった副作用が起きる。

- 選択肢 A（推奨）: **一覧には全部出すが、routine が管理するラベルは選べない。** 一覧に「触れない」印を出し、`Enter` を押すとフッタに赤で `<ラベル> は routine が管理するラベルです` と出して何も書かない。`stage:todo` だけは例外で `t` と同じくトグルできる。不変条件 2 を「TUI が書くラベルは `stage:todo` と、routine が管理しないラベル」に書き換える。何が存在して何が触れないかが画面で分かる
- 選択肢 B: **routine が管理するラベルを一覧から除外する。** 出るのはプロジェクト固有のラベルだけ。画面は短くなるが、「なぜ `docs` が出ないのか」が画面から分からない
- 選択肢 C: **全部トグルできる。** 不変条件 2 を撤廃し、「人が明示的に選んだラベルは書く」に変える。`l` は最も強力になるが、`stage:apply` を誤って付けると worker が 1 本起動し、段階ラベルが 2 つになった issue（局面 F）を人が手で直すことになる
- 依存: なし

### Q2. `l` のキー割り当て（カンバンの列移動と衝突する）

依頼は「`l` は現状どこにも割り当てが無いので衝突しない」と書いているが、これは実装済みのキーの話である。`docs/mvp/mvp.md` のキーバインド表は `h` / `l` / `←→` を「カンバンの列移動」に割り当てており、`openspec/specs/queue-screen/spec.md:117` も「`v` / `h` / `l` / `←` / `→` は s17 が担当」と書いている。カンバンビュー（`v`）は未実装だが、正本ドキュメントの上では予約済みである。

- 選択肢 A（推奨）: **`l` をラベル一覧に付け替え、カンバンの列移動は `←` / `→` だけにする。** mvp.md のキーバインド表から `h` / `l` を落とし、変更履歴に理由を 1 行足す。s22 が `u` を表に足したのと同じく、実装と正本を今のうちに揃える。カンバンは列が 4 つ程度なので矢印だけで足りる
- 選択肢 B: **ラベル一覧を `L`（大文字）にし、`l` はカンバン用に温存する。** mvp.md を変えずに済むが、依頼の「`l` キーで」と食い違い、`t` / `a` / `m` / `o` / `u` がすべて小文字の中で 1 つだけ大文字になる
- 選択肢 C: **`l` にするが mvp.md は今は変えない。** s17 の着手時にカンバンの列移動キーを決め直す。決定を先送りするので、s17 の propose で同じ問いをもう一度出すことになる
- 依存: なし

### Q3. キュー / 詳細画面のフッタのキーヒントに `l` を出すか

キュー画面のフッタのヒント `Enter 開く  a 回答  t todo  m merge  o ブラウザ  R 更新  ? ヘルプ  u URL  q 終了` は表示幅ちょうど 80 列で、`Model` の既定幅 80 に収まっている（`openspec/specs/queue-screen/spec.md:150`）。ここに `l ラベル`（空白 2 列込みで 9 列）を足すと 89 列になり、**ステータスが何も出ていない通常時でも末尾の `q 終了` が切れる**。

- 選択肢 A（推奨）: **フッタには足さず、`?` のヘルプにだけ載せる。** 移動系のキー（`j` / `k` / `1`–`4` / `Tab`）と同じ扱いにする。既定幅 80 の見た目を壊さない
- 選択肢 B: **足して、幅 80 では末尾が切れるのを受け入れる。** s14 が `m merge` を足したとき「キーを隠すよりキーを出すことを採った」（同 spec）のと同じ判断を続ける。ただし s14 のときの代償は「ステータスが出ている間ヒントが消える」で、今回は「常時 `q 終了` が切れる」であり一段重い
- 選択肢 C: **足したうえで、既存のヒントから何かを落とす**（例: `R 更新` を落として `?` に委ねる）。80 列に収まるが、落とすキーを選ぶ判断が別に要る
- 依存: なし
