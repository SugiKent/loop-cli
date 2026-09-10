## Why

今の TUI が書けるラベルは `t` の `stage:todo` だけである（`internal/ui/todo.go`）。それ以外のラベル——プロジェクト固有の分類ラベル、`docs` を付け忘れた PR、`ai-assess:requested` の付け直し——を触るには、ブラウザで GitHub を開くか別の端末で `gh` を打つしかない。loop-cli は「人の出番」を捌く手数を減らす道具なので、ラベル 1 つのために画面を離れさせるのは目的を損なう。

キーを 1 つ（`L`）で「使えるラベルの一覧を出して選ぶ」形にすれば、ラベルの種類が増えてもキーは増えない。`u`（s22 `url-picker`）が URL に対して採った形と同じである。

Refs #3

## What Changes

- `L`（Shift + L）を押すと**ラベル一覧画面**（`internal/ui/detail.go:22-30` の既存 7 状態に続く 8 つ目の画面の状態）に移り、対象リポジトリのラベルを並べる
- 一覧では、対象に今付いているラベルと、送信したら付く / 外れるラベルを行の印で区別する。`j` / `k` / `↑` / `↓` で行を移動し、選んだ行の印を切り替えると「変更予定」になる。この時点では GitHub に何も書かない
- **送信のキーを押すと、変更予定をまとめて 1 回の `gh` 呼び出しで反映する。** 複数のラベルを付け外ししても書き込みは 1 回である（回答 Q4）。送信のキーと印の切り替えのキーの割り当てだけが未確定（Q6）
- `Esc` は変更予定を捨てて元の画面に戻る。変更予定が 0 件のまま送信すると、何も書かずに閉じてフッタに `ラベルの変更がありません` を出す
- 送信に成功したら元の画面へ戻り、フッタに `<repo> #<n> のラベルを更新しました（+docs -wip）` の形で結果を出す。`Cards` は書き換えず、反映は `R` / 自動更新に任せる（`t` と同じ）
- **触れるラベルに制限を設けない**（回答 Q1: 選択肢 C）。`stage:*` / `wip` / `blocked` / `question` / `ai-assess:requested` を含む、そのリポジトリのすべてのラベルを付け外しできる
- 対象は `a` / `u` と同じ規則（画面が見せているもの）で決まる。キュー画面は選択行の主体、カード詳細は `Card.Issue`、PR 詳細は選択中の PR。`t` と違い PR も対象になる。対象は `L` を押した時点で `action.Target` に写し取り、一覧を出している間に自動更新で `Cards` が入れ替わっても変えない（s14 `merge-pr` が確認画面の対象を押した時点で固定するのと同じ）
- 書き込み中（`m.writing`）は `L` を受け付けない。ラベル一覧の取得中と送信中も書き込み中として扱う。取得中に画面が変わっていたら一覧を開かず、`m` と同じく「画面が変わりました」をステータスに出す（`openspec/specs/merge-pr/spec.md:38`）
- `GHClient` に `ListLabels(ctx, repo)` を足す。`Client` は `gh label list -R <repo> --json name,description,color --sort name --order asc --limit 100` を実行する。取得したラベルは `Model` がリポジトリ名をキーにプロセス内で保持し、同じリポジトリの 2 回目以降は `gh` を呼ばない
- `GHClient` に `EditIssueLabels` / `EditPRLabels`（付ける名前の並びと外す名前の並びを受け取る）を足す。`Client` は `gh issue edit <n> -R <repo> --add-label … --remove-label …` と `gh pr edit <n> -R <repo> --add-label … --remove-label …` を 1 回だけ実行する。既存の `AddLabel` / `RemoveLabel` は `t` の担当のまま残す
- ヘルプ画面のキー一覧に `L` の行を足し、キュー / カード詳細 / PR 詳細の 4 つのフッタすべてに `L ラベル` を足す（回答 Q5: 選択肢 A）
- `docs/mvp/mvp.md` のキーバインド表に `L` の行を足す。`l`（カンバンの列移動）はそのまま残す

## Capabilities

### New Capabilities

- `label-picker`: `L` でラベル一覧画面を開き、対象リポジトリのラベルを（付いているか・変更予定かの印つきで）並べ、選んだ変更をまとめて 1 回の書き込みで反映する。対象の決め方と固定、送信直前の読み直し、取得の失敗と書き込みの失敗のステータス表示、リポジトリ単位のキャッシュを含む

### Modified Capabilities

- `gh-client`: `GHClient` interface に `ListLabels` / `EditIssueLabels` / `EditPRLabels` を足し、`Client` が `gh label list` / `gh issue edit` / `gh pr edit` として実行することを定める
- `gh-fake`: `Fake` が `labels.json` から `ListLabels` を返し、その呼び出しとラベルの一括編集を `Calls` に記録することを定める
- `fixture-capture`: `Capture` が `labels.json` を採り、自己検査が `ListLabels` を含むことを定める
- `queue-screen`: キュー画面のキーの扱いに `L` を足し、フッタのキーヒントに `L ラベル` を足す
- `card-detail`: カード詳細 / PR 詳細のキーの扱いに `L` を足し、3 つのフッタのキーヒントに `L ラベル` を足す
- `help-screen`: ヘルプ画面のキー一覧に `L` の行を足す

## Impact

- `internal/gh/gh.go` / `client.go` / `fake.go` / `capture.go`: `ListLabels` と `Label` 型、`EditIssueLabels` / `EditPRLabels`、`labels.json` fixture
- `internal/gh/testdata/fixtures/example/labels.json`: 新規 fixture
- `internal/ui`: ラベル一覧画面を新しいファイルに足し、`Update` のキー振り分け（URL 一覧と同じく `a` / `t` / `m` / `o` / `?` より前）と `View` の画面分岐を広げ、4 つのフッタのヒントとヘルプのキー一覧に `L` を加える
- `internal/action`: `Target` と「付ける名前の並び・外す名前の並び」を受け取って 1 回で書き込む関数（`ToggleTodo` の隣）
- `docs/mvp/mvp.md`: キーバインド表に `L` の行と、変更履歴に 1 行
- `docs/domain/issue-driven-sdd/human-turn-signals.md`: 不変条件 1 と 2、および冒頭の「`ai-assess:requested` は この TUI からは行わない」の 3 か所の改訂（後述）
- `l` はカンバンの列移動（s17、未着手）の予約のまま残るので、`docs/mvp/mvp.md` の既存行は変えない
- 並行して走っている propose との関係: `openspec/changes/` 直下に進行中の change は無いが、issue #2 / #4 / #5 / #6 が同時に propose 段階にある（それぞれ別セッション）。#4（`c` で close）と #6（`n` で issue 作成）は同じく画面とキーを足すので `queue-screen` / `card-detail` / `help-screen` / `gh-client` / `gh-fake` の同じ Requirement を MODIFIED する。#2（タイトルの折り返し）は `card-detail` のヘッダを MODIFIED する。s22 が s21 に対して行ったのと同じく、**先に merge された change の delta を土台に写して MODIFIED を書き、実装と archive は merge 順に行う**

## 確定した判断

### 人の回答で決まったこと

- **触れるラベルに制限を設けない（Q1: C）。** `docs/domain/issue-driven-sdd/human-turn-signals.md` の不変条件 2「TUI が書くラベルは `stage:todo` と `stage:propose` の 2 つに限る」と、冒頭の「`ai-assess:requested` を付け直すと AI 評価をもう一度走らせられるが、この TUI からは行わない」を撤廃し、「人が `L` で明示的に選んだラベルは書く」に書き換える。誤って `stage:apply` を付けると worker が 1 本起動し、`question` を外すと AI 評価前の PR を merge できるようになるが、どちらも受け入れる
- **キーは `L`（Shift + L）（Q3）。** `l` はカンバンの列移動（s17）の予約のまま残す。フッタとヘルプの表記は `L` とする（`R 更新` が既に大文字キーを `Shift+R` と書かずに `R` と書いている前例に合わせる）
- **書き込みは「選んで、まとめて送信」（Q4）。** 印の切り替えは GitHub に触らず、送信で 1 回だけ書く
- **フッタのキーヒント 4 つすべてに `L ラベル` を足す（Q5: A）。** キュー画面は 80 → 90 列になり、既定幅 80 の端末では末尾の `q 終了` が切れる。`queue-screen` spec:150 と `card-detail` spec:194 が既に 2 回記録している「キーを隠すよりキーを出すことを採った」を続ける
- Q2（触れないラベルを一覧に出すか）は、Q1 が C になったことで問い自体が消えた

### 回答から導いたこと

- **不変条件 1 を「ラベル集合の置換は行わない」に狭める。** 現在の文面は「`--add-label` / `--remove-label` を 1 ラベルずつ**別プロセスで**呼ぶ」だが、Q4 の「1 度のリクエストで更新が走る」と両立しない。`gh issue edit <n> --add-label a --add-label b --remove-label c` は 1 プロセスで、`gh` の内部でも `addLabelsToLabelable` と `removeLabelsFromLabelable` の 2 つの mutation に分かれる（cli/cli の `pkg/cmd/pr/shared/editable_http.go` が、ラベル一覧の丸ごとの置き換えと競合を避けるためにこの 2 つへ分けると注記している）。**ラベル集合の置換は起きない**ので、不変条件 1 が本当に守りたかった「他の人の付けたラベルを消さない」は保たれる。1 回の送信で増えたラベルの数だけ `labeled` イベントが出る点は変わるが、`t` の経路は 1 ラベルずつのまま残す
- **送信後は元の画面に戻る。** 送信は 1 回で完結する操作であり、送信後の一覧は印が実物と食い違う（`Cards` を書き換えないため）。残す理由が無い
- **確認画面は出さない。** issue #3 の「トグルなので確認画面は出さない」を保つ。送信前の「変更予定の印が付いた一覧」がそのまま確認の場になっており、`Esc` で捨てられる。確認画面を挟むと同じ内容を 2 画面で見せることになる
- **送信の直前に対象のラベルを読み直す。** 対象が issue なら `ViewIssue`、PR なら `ViewPR` で現在のラベルを取り、印が表す「送信後にこうなっていてほしい状態」との差分から、付ける名前の並びと外す名前の並びを作る。印の元になる画面の Card のラベルは最終取得時点のもので、その間に他の経路で変わっていると差分がずれる（`action.ToggleTodo` が `ViewIssue` で読み直しているのと同じ理由）

### 調査で確定したこと

- **PR へのラベル書き込みは `gh pr edit` の経路を新設する。** `gh issue edit` は `issueOrPullRequest(number:)` で対象を解決するので PR 番号でも通るが、この経路は採らない。理由は 3 つ。(1) `internal/gh/client.go` は PR の操作をすべて `pr` サブコマンドで書いており（`CommentPR` は `gh pr comment` client.go:278、`MergePR` は `gh pr merge` client.go:293）、ラベルだけ型を跨ぐと唯一の例外になる。(2) `internal/action` は `Target{Repo, Number, IsPR}`（action.go:29-33）を持ち、不変条件 4「回答の書き先は 3 種類を混同しない」を層の責務にしている。(3) `gh issue edit` が PR を受けることは `gh` のヘルプに書かれておらず、`Fake` は `gh` を起動しないので、この挙動が将来変わってもテストが気づけない
- **一覧は 8 つ目の画面の状態として全画面で描く。** 依頼の言葉は「モーダル」だが、このコードベースに重ね合わせの前例は無く、確認画面・merge 確認画面・ヘルプ画面・URL 一覧はすべて全画面である（s22 design.md D-1 と同じ判断）
- **一覧画面の分岐は `a` / `t` / `m` / `o` / `?` より前に置く。** `internal/ui/model.go:220-260` が確認画面・merge 確認画面・ヘルプ画面・URL 一覧をこの位置で分岐しており、後ろに置くと一覧の裏の対象へ書き込みが起きる
- **ラベルが 0 件のリポジトリでは画面を変えず、ステータスに `ラベルがありません` を出す。** s22 の「URL がありません」（`openspec/specs/url-picker/spec.md:45`）と同型にする
- **一覧の並びは `gh` に固定させる。** `gh label list` の既定は作成順（cli/cli `pkg/cmd/label/http.go` の `defaultSort = "created"`）で、リポジトリごとに並びが変わると画面も fixture も非決定になる。`--sort name --order asc` を渡す

## 未確定の判断

### Q6. 一覧画面のキー割り当て（印の切り替えと送信）

TUI にボタンは無いので、Q4 の「サブミットボタン」をキーに置き換える必要がある。この画面で要るのは 4 つ——行の移動 / 印の切り替え / 送信 / 破棄——で、行の移動（`j` / `k` / `↑` / `↓`）と破棄（`Esc`）は他の画面と同じにする。残る 2 つの割り当てを決める。

- 選択肢 A（推奨）: **`Space` で印の切り替え、`Enter` で送信。** 一覧から複数を選ぶ UI の一般的な形で、`Space` が「選ぶ」、`Enter` が「決定」という対応が素直。既存の画面で `Space` に割り当てはない
- 選択肢 B: **`Enter` で印の切り替え、`Ctrl+S` で送信。** `Enter` の意味が URL 一覧（開く）や merge 確認（使わない）とずれないが、`Ctrl+S` はこのコードベースで初めての修飾キーになる
- 選択肢 C: **`Enter` で印の切り替え、`w`（write）で送信。** 修飾キーを増やさずに済むが、`w` に「送信」を読み取れるかどうかは慣れ次第で、押し間違えると意図せず書き込みが走る
- 依存: なし
