## Why

今の TUI が書けるラベルは `t` の `stage:todo` だけである（`internal/ui/todo.go`）。それ以外のラベル——プロジェクト固有の分類ラベル、`docs` を付け忘れた PR——を触るには、ブラウザで GitHub を開くか別の端末で `gh` を打つしかない。loop-cli は「人の出番」を捌く手数を減らす道具なので、ラベル 1 つのために画面を離れさせるのは目的を損なう。

キーを 1 つ（`l`）で「使えるラベルの一覧を出して選ぶ」形にすれば、ラベルの種類が増えてもキーは増えない。`u`（s22 `url-picker`）が URL に対して採った形と同じである。

ただし、このプロジェクトでラベルは routine の状態機械の入力そのものであり、`docs` を含むどこまでを人に触らせるかは仕様の判断である（未確定の判断 Q1）。

Refs #3

## What Changes

- `l` を押すと**ラベル一覧画面**（`internal/ui/detail.go:22-30` の既存 7 状態に続く 8 つ目の画面の状態）に移り、対象リポジトリのラベルを並べる。`j` / `k` / `↑` / `↓` で選び、`Enter` で決定、`Esc` で元の画面に戻る
- 対象は `a` / `u` と同じ規則（画面が見せているもの）で決まる。キュー画面は選択行の主体、カード詳細は `Card.Issue`、PR 詳細は選択中の PR。`t` と違い PR も対象になる。対象は `l` を押した時点で `action.Target` に写し取り、一覧を出している間に自動更新で `Cards` が入れ替わっても変えない（s14 `merge-pr` が確認画面の対象を押した時点で固定するのと同じ）
- `Enter` で決定したラベルは、対象に付いていれば外し、付いていなければ付ける（`t` と同じトグル。確認画面は出さない）。付けるか外すかは書き込みの直前に GitHub から読み直したラベルで決める（`action.ToggleTodo` と同じ理由。画面の Card のラベルは最終取得時点のもの）
- 一覧では、対象に今付いているラベルに印を出す。印の元は画面の Card のラベルで、トグルに成功したラベルの印だけは結果で上書きする
- 結果はフッタのステータスに出す。`Cards` は書き換えず、反映は `R` / 自動更新に任せる（`t` と同じ）
- 書き込み中（`m.writing`）は `l` を受け付けない。ラベル一覧の取得中も書き込み中として扱う。取得中に画面が変わっていたら一覧を開かず、`m` と同じく「画面が変わりました」をステータスに出す（`openspec/specs/merge-pr/spec.md:38`。ヘルプ画面や URL 一覧をラベル一覧で踏み潰さないため）
- `GHClient` に `ListLabels(ctx, repo)` を足す。`Client` は `gh label list -R <repo> --json name,description,color --sort name --order asc --limit 100` を実行する。取得したラベルは `Model` がリポジトリ名をキーにプロセス内で保持し、同じリポジトリの 2 回目以降は `gh` を呼ばない
- `GHClient` に `AddLabelPR` / `RemoveLabelPR` を足す。PR へのラベル書き込みは `gh pr edit -R <repo> <n> --add-label` / `--remove-label` で行う（後述「確定した判断」）
- ヘルプ画面のキー一覧に `l` の行を足す
- `docs/mvp/mvp.md` のキーバインド表で `l` に当たっている「カンバンの列移動」との衝突を解く（Q3）

## Capabilities

### New Capabilities

- `label-picker`: `l` でラベル一覧画面を開き、対象リポジトリのラベルを（対象に付いているかの印つきで）並べ、選んだ 1 つを対象に対してトグルする。対象の決め方と固定、書き込み直前の読み直し、取得の失敗と書き込みの失敗のステータス表示、リポジトリ単位のキャッシュ、触れないラベルの扱いを含む

### Modified Capabilities

- `gh-client`: `GHClient` interface に `ListLabels` / `AddLabelPR` / `RemoveLabelPR` を足し、`Client` が `gh label list` / `gh pr edit` として実行することを定める
- `gh-fake`: `Fake` が `labels.json` から `ListLabels` を返し、その呼び出しと PR へのラベル書き込みを `Calls` に記録することを定める
- `fixture-capture`: `Capture` が `labels.json` を採り、自己検査が `ListLabels` を含むことを定める
- `queue-screen`: キュー画面のキーの扱いで、`l` を「s17 が担当する未実装のキー」からラベル一覧画面を開くキーに変える。フッタのキーヒントの扱いは Q5 の結論に依る
- `card-detail`: カード詳細 / PR 詳細のキーの扱いに、`l` がラベル一覧画面を開く行を書き足す。フッタのキーヒントの扱いは Q5 の結論に依る
- `help-screen`: ヘルプ画面のキー一覧に `l` の行を足す

## Impact

- `internal/gh/gh.go` / `client.go` / `fake.go` / `capture.go`: `ListLabels` と `Label` 型、`AddLabelPR` / `RemoveLabelPR`、`labels.json` fixture
- `internal/gh/testdata/fixtures/example/labels.json`: 新規 fixture
- `internal/ui`: ラベル一覧画面を新しいファイルに足し、`Update` のキー振り分け（URL 一覧と同じく `a` / `t` / `m` / `o` / `?` より前）と `View` の画面分岐を広げる
- `internal/action`: `Target` を受け取ってラベル 1 つをトグルする関数（`ToggleTodo` の隣）と、触れないラベルの判定（`internal/action` の package doc が「不変条件の判定をここ 1 か所に置き、`internal/ui` は判定を持たずにこの層の関数を呼ぶ」と定めているため）
- `docs/mvp/mvp.md`: キーバインド表（`l` の行、Q3 の結論）
- `docs/domain/issue-driven-sdd/human-turn-signals.md`: 不変条件 2「TUI が書くラベルは `stage:todo` と `stage:propose` の 2 つに限る」と、冒頭の「`ai-assess:requested` を付け直すと AI 評価を走らせられるが、この TUI からは行わない」の 2 か所の改訂（Q1 の結論）
- 先行 change との関係: 進行中の change は無く（`openspec/changes/` 直下は archive のみ）、open PR も無い。`l` は s17（カンバンビュー `v` / `h` / `l` / `←` / `→`）が予約しているキーで、s17 は未着手

## 確定した判断

- **PR へのラベル書き込みは `gh pr edit` の経路を新設する。** `gh issue edit` は `issueOrPullRequest(number:)` で対象を解決するので PR 番号でも通るが、この経路は採らない。理由は 3 つ。(1) `internal/gh/client.go` は PR の操作をすべて `pr` サブコマンドで書いており（`CommentPR` は `gh pr comment` client.go:278、`MergePR` は `gh pr merge` client.go:293）、ラベルだけ型を跨ぐと唯一の例外になる。(2) `internal/action` は `Target{Repo, Number, IsPR}`（action.go:29-33）を持ち、不変条件 4「回答の書き先は 3 種類を混同しない」を層の責務にしている。(3) `gh issue edit` が PR を受けるのは文書化されていない挙動で、`Fake` は `gh` を起動しないため壊れてもテストで検出できない。新設は `Client` に 2 メソッド（各 2 行）で済む
- **`gh` はラベルを 1 つずつの mutation で足し引きする。** cli/cli の `pkg/cmd/pr/shared/editable_http.go` は「Labels are updated through discrete mutations to avoid having to replace the entire list of labels and risking race conditions」として `addLabelsToLabelable` / `removeLabelsFromLabelable` を呼ぶ。CLI の引数と API の呼び方の両方で、不変条件 1「ラベル集合の置換は行わない」が守られる
- **routine が読むラベルの集合は `internal/model/model.go:13-24` の 12 定数で既に定義されている**（`stage:todo` / `stage:propose` / `stage:apply` / `stage:archive` / `wip` / `blocked` / `question` / `propose` / `apply` / `archive` / `docs` / `ai-assess:requested`）。Q1 はこの集合をどう扱うかの問いであって、集合そのものを決め直す問いではない
- **触れるかどうかの判定は `internal/action` に置く。** package doc（action.go:1-3）が「不変条件の判定をここ 1 か所に置き、`internal/ui` は判定を持たずにこの層の関数を呼ぶ」と定めている。UI は判定結果（触れる / 触れない）を受け取って印と拒否のメッセージを描くだけにする
- **`stage:todo` は `l` から触らない。** `t` が担当し、`action.ToggleTodo` の段階ラベルの検査（`ErrOtherStage` / `ErrMultipleStages` / `ErrBlocked`。todo.go:36-47）を通す。`l` の汎用トグルにこの検査を持ち込むと、`stage:todo` 以外のラベルにも段階ラベルの検査が走る（Q1 で選択肢 C を採る場合だけ、`l` から選ばれた `stage:todo` を `ToggleTodo` に流す）
- **一覧は 8 つ目の画面の状態として全画面で描く。** 依頼の言葉は「モーダル」だが、このコードベースに重ね合わせの前例は無く、確認画面・merge 確認画面・ヘルプ画面・URL 一覧はすべて全画面である（s22 design.md D-1 と同じ判断）
- **一覧画面の分岐は `a` / `t` / `m` / `o` / `?` より前に置く。** `internal/ui/model.go:220-260` が確認画面・merge 確認画面・ヘルプ画面・URL 一覧をこの位置で分岐しており、後ろに置くと一覧の裏の対象へ書き込みが起きる
- **トグルの判定は書き込み直前に GitHub から読み直す。** `action.ToggleTodo`（todo.go:22-49）が `ViewIssue` で読み直しているのと同じ理由で、画面のラベルは最終取得時点のものであり、続けて 2 回押した 2 回目が「もう一度付ける」になってしまう。対象が PR のときは `ViewPR` で読む
- **ラベルが 0 件のリポジトリでは画面を変えず、ステータスに `ラベルがありません` を出す。** s22 の「URL がありません」（`openspec/specs/url-picker/spec.md:45`）と同型にする

## 未確定の判断

### Q1. `l` から書けるラベルの範囲

`docs/domain/issue-driven-sdd/human-turn-signals.md` は 2 か所でこれを禁じている。不変条件 2「**TUI が書くラベルは `stage:todo` と、`s` の強制操作で付ける `stage:propose` の 2 つに限る**」と、冒頭の「`ai-assess:requested` を付け直すと PR の AI 評価をもう一度走らせられるが、**この TUI からは行わない**」。任意のラベルをトグルできる `l` はこの 2 つと正面から衝突する。

12 定数（確定した判断を参照）は、人が触ったときに何が起きるかで 3 群に分かれる。

| 群 | ラベル | 人が触ると何が起きるか |
| --- | --- | --- |
| ア | `stage:propose` / `stage:apply` / `stage:archive` / `wip` / `blocked` / `question`（issue） | worker が起動する・段階ラベルが 2 つになる（局面 F）・dispatcher の再起動判定が狂う |
| イ | `question`（PR） / `ai-assess:requested` | loop-cli 自身の merge ガード（不変条件 5）を外側から無効化できる。付ければ merge できなくなり、外せば AI 評価前の PR を merge できてしまう |
| ウ | `docs` / `propose` / `apply` / `archive`（PR の段階ラベル） | `docs` は merge しても段階が進まないが、loop-cli の分類が局面 G に変わる。`propose` / `apply` は merge で dispatcher が issue の段階を進める |

Why が挙げた「`docs` を付け忘れた PR」は群ウにある。ここを禁じると、この change の動機の 1 つがそのまま消える。

- 選択肢 A（推奨）: **群ウのうち `docs` だけを触れるようにし、ア・イと `propose` / `apply` / `archive` は触れない。** 触れるのは `docs` とプロジェクト固有のラベル。`docs` は merge しても段階が進まず、間違えても loop-cli の分類が変わるだけで元に戻せる。不変条件 2 を「TUI が書くラベルは `stage:todo`、`s` の `stage:propose`、`docs`、および routine が読まないラベル」に書き換える。冒頭の `ai-assess:requested` の一文は残す
- 選択肢 B: **12 定数すべてを触れないようにする。** 触れるのはプロジェクト固有のラベルだけ。不変条件 2 の改訂は「routine が読まないラベルは書いてよい」の 1 行で済み、merge ガードも段階も絶対に崩れない。代わりに Why の `docs` の動機は満たせないので、Why を書き直す
- 選択肢 C: **全部触れる。** 不変条件 2 と `ai-assess:requested` の一文を撤廃し、「人が明示的に選んだラベルは書く」に変える。`l` は最も強力になるが、`stage:apply` を誤って付けると worker が 1 本起動し、`question` を外すと AI 評価前の PR を merge できる。どちらも TUI からは取り消せない
- 依存: なし。この答えが Q2 の前提になる

### Q2. 触れないラベルを一覧に出すか

Q1 で A か B を採った場合、触れないラベルの見せ方を決める必要がある。

- 選択肢 A（推奨）: **一覧に出すが選べない。** 行に「触れない」印を出し、`Enter` を押すとフッタに赤で `<ラベル> は routine が管理するラベルです` を出して何も書かない。リポジトリに何があるかが分かり、なぜ触れないかも画面で伝わる
- 選択肢 B: **一覧から除外する。** 画面は短くなるが、「なぜ `wip` が出ないのか」が画面から分からず、利用者が「ラベル一覧の取得が壊れている」と読む余地が残る
- 依存: Q1 で C（全部触れる）を選ぶとこの問いは消える

### Q3. `l` のキー割り当て（カンバンの列移動と衝突する）

依頼は「`l` は現状どこにも割り当てが無いので衝突しない」と書いているが、これは実装済みのキーの話である。`docs/mvp/mvp.md:105` のキーバインド表は `h` / `l` / `←→` を「カンバンの列移動」に割り当てており、`openspec/specs/queue-screen/spec.md:117` も「`v` / `h` / `l` / `←` / `→` は s17 が担当」と書いている。カンバンビュー（`v`）は未実装だが、正本ドキュメントの上では予約済みである（なお同 spec の Scenario「未実装のキーは何も変えない」の列挙には `h` / `l` が入っていないので、直すのは散文とキーの箇条書きだけで足りる）。

- 選択肢 A（推奨）: **`l` だけをラベル一覧に付け替え、カンバンの列移動は `h` / `←` / `→` にする。** `h` は残すので vim 風の移動が片側だけ生き、`l` の代わりは `→` が担う。mvp.md の表と変更履歴を直す
- 選択肢 B: **`h` も一緒に落とし、カンバンの列移動は `←` / `→` だけにする。** 片側だけ残る不揃いを避けられるが、`h` を落とす理由は `l` の衝突と無関係である
- 選択肢 C: **ラベル一覧を `L`（大文字）にし、`l` はカンバン用に温存する。** mvp.md を変えずに済むが、依頼の「`l` キーで」と食い違い、`t` / `a` / `m` / `o` / `u` がすべて小文字の中で 1 つだけ大文字になる
- 依存: なし

### Q4. `Enter` でトグルした後、一覧を閉じるか残すか

前例が割れている。URL 一覧は `Enter` の後も一覧に留まる（`internal/ui/urls.go:114-115` は `screen` を触らない。続けて別の URL を開けるようにするため）。`m` の merge は確認画面から抜ける。

- 選択肢 A（推奨）: **残す。** 続けて 2 つ目のラベルを触れる。トグルに成功したラベルの印はその場で書き換わるので（確定した判断）、「付けたのに印が変わらない」を見て二度押しする事故は起きない
- 選択肢 B: **閉じて元の画面に戻る。** 1 回のトグルで完了する使い方に最短で、押しっぱなしの誤操作も起きにくい。2 つ目を触るには `l` をもう一度押す
- 依存: なし

### Q5. フッタのキーヒントに `l` を出すか

`l ラベル` は空白 2 列込みで表示幅 10 列。現在のフッタは次のとおりで、既に 2 つが `Model` の既定幅 80 を超えている。

| フッタ | 現在 | `l ラベル` を足すと |
| --- | --- | --- |
| キュー（`view.go:83-86`） | 80 | 90 |
| PR 詳細（`detail.go:396-398`） | 82 | 92 |
| カード詳細・PR あり（`detail.go:402`） | 117 | 127 |
| カード詳細・PR なし（`detail.go:400`） | 69 | 79（収まる） |

このプロジェクトは同じ判断を 2 回している。`openspec/specs/queue-screen/spec.md:150` と `openspec/specs/card-detail/spec.md:194` はどちらも「既定幅 80 の端末では末尾の `q 終了` が切れる（`q` は切れても動く。**キーを隠すよりキーを出すことを採った**）」と記録している。`detail.go:394` のコメントも「動くキーだけを出す」である。

- 選択肢 A（推奨）: **4 つすべてに足す。** 前例をそのまま続ける。キュー画面は幅 80 で `q 終了` が切れるようになる（PR 詳細とカード詳細は既に切れている）
- 選択肢 B: **どれにも足さず、`?` のヘルプにだけ載せる。** 移動系のキー（`j` / `k` / `1`–`4` / `Tab`）と同じ扱いにする。前例を 2 回ぶん反転させ、「動くのに出さないキー」を初めて作る
- 選択肢 C: **79 列に収まるカード詳細・PR なしのフッタにだけ `l ラベル` を足す。** 幅で切れる場所を増やさずに済むが、同じキーがフッタに出たり出なかったりする
- 依存: なし
