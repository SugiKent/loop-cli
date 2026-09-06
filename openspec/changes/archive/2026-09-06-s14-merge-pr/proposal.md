## Why

人の役割 3 つ（`stage:todo` を付ける / `question` に答える / PR を merge する）のうち、merge だけが TUI に無い。
局面 C（merge する）の行はキューに並ぶのに、そこから `o` でブラウザへ出るしか道が無く、TUI が「人の出番を 1 本のキューで捌く」という目的を満たしていない。
`gh` 側の `MergePR` は s03 で実装済みで、キー `m` は s14 の担当として各 spec に予約されている。

## What Changes

- キュー画面（選択行の主体が PR のとき）・カード詳細画面（選択中の PR）・PR 詳細画面で `m` を押すと、対象 PR の最新状態を GitHub から取り直し、merge の確認画面を出す
- 確認画面で `y` を押すと `gh pr merge -R <repo> <n> --<method>` を実行する。`method` は設定（リポジトリ別上書き込み）の `merge_method`。`Esc` で中止する
- ガードは **draft と、すでに merged / closed になっている PR** が merge を止める。この 2 つでは確認画面に merge の選択肢（`y`）を出さない
- `question` ラベル / 本文 1 行目の `未確定の判断: N 件`（N > 0）/ checks 非緑は、確認画面に**警告として並べる**が `y` で merge できる（人の判断に委ねる。docs 逸脱。下記 Impact 参照）
- merge の結果はフッタ右のステータスに出す。`Cards` は変えず、再取得もしない（s10 / s11 と同じ）
- 書き込み中（コメント投稿 / ラベル切り替え / merge）は `m` を受け付けない
- キュー画面と詳細画面のフッタ、`?` ヘルプのキー一覧に `m merge` を足す

## Capabilities

### New Capabilities
- `merge-action`: `internal/action` の merge。ガード判定（draft と open でない PR は拒否、`question` / 未確定 N > 0 / checks 非緑 は警告）と `gh.GHClient.MergePR` の呼び出し。ラベル・コメントは書かない
- `merge-pr`: `internal/ui` の `m` を定める。`Model` が対象の PR を決め、押下時に最新状態を取り直し、merge の確認画面（`y` / `Esc`）を出し、結果をステータスに出す。merge 方式は `Options` が供給する

### Modified Capabilities
- `queue-screen`: キュー画面の `m` が「何もしないキー」から「主体が PR なら merge の確認画面を開く」に変わる。フッタのキーヒントに `m merge` が入り 80 列になる
- `card-detail`: 画面の状態が 6 つから 7 つ（merge 確認画面を追加）になる。カード詳細・PR 詳細のフッタのキーヒントに `m merge` が入る
- `help-screen`: 実装済みキーの一覧に `m` が入る
- `todo-toggle`: 「書き込み中は `t` と `a` を受け付けない」に `m` が加わり、merge 中も `t` / `a` / `m` を受け付けない
- `tui-entrypoint`: `ui.New` に渡す `ui.Options` に、リポジトリ別の merge 方式（`Config.Repos[].MergeMethod`）を渡す配線が加わる

## Impact

- コードは 3 か所を触る。`internal/action` に merge のファイルを足し、`internal/ui` に `m` のキー処理と確認画面とステータスと `Options` の項目を足し、`cmd/loop-cli` で merge 方式を配線する
- `gh` は、`m` の押下時に読み取りの `ViewPR` と `ViewPRMergeState` を呼び、`y` の後に書き込みの `MergePR` を呼ぶ。いずれも s03 が実装済みで、`GHClient` インターフェースは変えない
- docs 逸脱: human-turn-signals.md 不変条件 5 は「`question` あり / 1 行目 N > 0 / checks 失敗 / `isDraft` なら merge を拒否し理由を表示する」と定めるが、この change は draft 以外を警告に緩める。理由（人の判断に委ねる）と申し送りを design.md に書く
- 逸脱の回収: human-turn-signals.md「merge 可否は表示時に取り直す」に対して s09 は「詳細を開いても `gh` を呼ばない」と逸脱していた。`m` の押下時の再取得で、merge の判断に使う値についてはこれを回収する
- 書き込み後の対象 1 件再取得（D-002）は s18 の担当で、この change では行わない
