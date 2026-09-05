## 1. 前提と依存

- [x] 1.1 s11-todo-toggle が実装済みで、`internal/ui` に画面の状態 4 つ（キュー / カード詳細 / PR 詳細 / 確認）、`New(fetcher, client, editor)`、フッタ右の書き込みステータス、書き込み中フラグがあり `go build ./... && go test ./internal/ui/` が通ることを確認する。未実装ならこの change を始めない。s10 / s11 の最終版の specs と、この change の MODIFIED（`queue-screen` 2 件、`card-detail` 2 件）を突き合わせ、本文がレビューで変わっていれば写し直す。s11 のカード詳細の `t` が `Update` で画面判定前に振り分けられている（`a` と同じ形）か `updateDetailKey` 側で扱われているかを確認し、この change の `o` は 3.1 のとおり `Update` 側に置いたうえで、`t` が `updateDetailKey` 側にあれば `t` も `Update` 側に寄せて揃える
- [x] 1.2 `internal/gh/fake.go` の `Fake.Browse` が `Calls` に `Call{Method: "Browse", Repo, Number}` を記録すること、`internal/gh/client.go` の `Client.Browse` が `gh browse <n> -R <repo>` を実行することを確認する。`internal/gh` は変更しない

## 2. R（取得開始の共通経路）

- [x] 2.1 `internal/ui/model.go` に非公開の `startFetch() tea.Cmd`（`Model` のポインタレシーバ）を実装する。この関数は `fetching` を true にし、`errText` と `partial` と書き込みステータス（s10 / s11 の共有フィールド）を空にし、`tea.Batch(m.spinner.Tick, fetchCmd(m.fetcher))` を返す。`startFetch` は `fetching` のガードを持たない。`Init` は現状のまま（`New` が `fetching` を立て、`Init` は tick + `fetchCmd` を返す）で `startFetch` を呼ばない。`startFetch` は `R` と s13 用である。キュー画面の `updateKey` に `R` を足す: `fetching` なら `m, nil` を返し、そうでなければ `startFetch()` の返り値を返す。キーの判定は既存のキーと同じく `tea.KeyPressMsg` の `String()` で行う（Bubble Tea v2 の `Key.String()` は `Text` が空でなければ `Text` を返し、端末で Shift + r を押すと `Text` は `R` になる。`shift+r` は見ない）。カード詳細 / PR 詳細 / 確認 / ヘルプの各画面では `R` を何もしないままにする（`updateDetailKey` と確認画面のキー処理に `R` を足さない）
- [x] 2.2 `internal/ui/model_test.go` で `R` を検証する。準備として、呼ばれた回数 `n` を数える `Fetcher`（`example` の `Result` を返す）で `New` し、`fetchedMsg` を直接 `Update` に渡して取得完了にする（この時点で `n` は 0）。検証する内容は次のとおり
  - この `Model` に `R` を与えると、コマンドが返り、`fetching` が true になり、`View` に `取得中` が出る。返ったコマンドを実行し、得たメッセージが複数コマンドの束なら各要素も実行して、`fetchedMsg` を `Update` に渡すと、`n` は 1 になり、`cards` が入り、`fetching` は false になる
  - `R` の直後（コマンド未実行）にもう一度 `R` を与えると nil が返り `n` は変わらない。`New` 直後の `Model` に `R` を与えても nil が返る
  - `example` の `Result` の後に `rate limited` の失敗を渡し、さらに s11 の手順で `t` の結果メッセージまで渡した `Model` に `R` を与えると、`View` に `取得中` があり、`rate limited` と `付けました` は無く、今やるタブの issue 108 の行とヘッダの `↻ 12:04` は残る
  - `付けました` まで出した `Model` に `R` を与えた後に失敗の `fetchedMsg` を渡すと、issue 108 の行と `↻ 12:04` が残り、`rate limited` が出て、`付けました` と `取得中` は無い
  - 3 行のタブで `cursor` を 2 にしてから `R` を与え、同じ 3 枚の `fetchedMsg` を渡すと `cursor` は 2 のまま
  - カード詳細 / PR 詳細 / 確認画面（s10 の手順）/ ヘルプ画面の各 `Model` に `R` を与えると nil が返り、`fetching` は false のまま

## 3. o（ブラウザで開く）

- [x] 3.1 `internal/ui/browse.go` に `o` のキー処理 `browseKey() (tea.Model, tea.Cmd)`（非公開。`answerKey` と同じ形）を実装する。対象は、キュー画面なら選択行の `Subject`（行が無ければ `m, nil`）、カード詳細なら `detail.card.Issue`、PR 詳細なら `currentPR()`、確認画面とヘルプ画面なら `m, nil` とする。表示名は s10 と同じ（`<Repo> PR#<n>` / `<Repo> #<n>`）。書き込みステータスは触らずに、非公開の `browseCmd(client, label, repo, number) tea.Cmd`（`context.WithTimeout(context.Background(), 30*time.Second)` で `client.Browse` を呼び `browsedMsg{label, err}` を返す）を返す。書き込み中フラグは見ない。`browsedMsg` は `Update` の先頭で画面に関わらず処理する: `err == nil` なら何もせず、`err != nil` なら書き込みステータスに `<label> をブラウザで開けません: <err>` を赤で入れる。`cards` / `at` / `detail` は触らず、取得コマンドを返さない。`o` の振り分けは `model.go` の `Update` に置く: `key == "a"` の分岐の直後（画面判定 `m.screen != screenQueue` より前）に `key == "o"` を足し、`a` と同じ形で画面に関わらず `browseKey()` を呼ぶ（`updateDetailKey` は `Model` だけを返して Cmd を返せないので、`updateDetailKey` にも `updateKey` にも `o` を足さない）
- [x] 3.2 `internal/ui/browse_test.go` で検証する。共通の準備は s10 / s11 と同じにする（`example` の `Result` は `fetch.Fetch` で作り、`New` に渡す `Fake` は `gh.NewFake` で別に作る。コマンドは `cmd()` で実行してメッセージを `Update` に渡す。`View` は ANSI エスケープを除いて読む）。検証する内容は次のとおり
  - 今やるタブ（主体 PR 131）の `Model` に `o` を与えると、直後の `fake.Calls` は空で、コマンドの実行後は `Browse` / `org/app` / 131 の 1 件になり、画面はキューのまま。バックログ（issue 140）で `o` を与えると `Browse` / 140 になる
  - issue 108 のカード詳細で `o` を与えると `Browse` / 108 になり画面はカード詳細のまま。そこから `Enter` で PR 詳細に移って `o` を与えると `Browse` / 131 になり画面は PR 詳細のまま
  - 異常タブ（0 行）、s10 の手順で移った blocked-by の確認画面、`?` で開いたヘルプ画面の各 `Model` に `o` を与えると nil が返り、`Calls` は空で、画面は変わらない
  - issue 140 で `t` を与えた直後（コマンド未実行）に `o` を与えるとコマンドが返り、実行後に `Calls` に `Browse` / 140 が入る
  - `*gh.Fake` を埋め込み `Browse` が `errors.New("gh browse 131 -R org/app: exit 1: no browser")` を返す型を `client` にして PR 131 で `o` を与え、コマンドを実行すると、`View` に `org/app PR#131 をブラウザで開けません:` と `no browser` が出て、`cards` は変わらず、画面はキューのまま。続けて `R` を与えると `View` から `開けません` が消え `取得中` が出る
  - 成功の手順の後の `View` のフッタに `開けません` は無い

## 4. ?（ヘルプ画面）

- [x] 4.1 `internal/ui/help.go` にヘルプ画面を実装する。`screen` に `screenHelp` を足し、`Model` に戻り先 `helpFrom screen` を持たせる。`?` のキー処理は、キュー / カード詳細 / PR 詳細では `helpFrom` に現在の画面を入れて `screen` を `screenHelp` にし、確認画面では何もしない（s10 の確認画面のキー処理に `?` を足さない）。ヘルプ画面のキー処理 `updateHelpKey(key)` は、`?` と `esc` で `screen` を `helpFrom` に戻し、`helpFrom` がカード詳細 / PR 詳細なら `refreshDetail()` を呼ぶ（ヘルプ中の `WindowSizeMsg` は詳細の寸法を作り直さないため。`GotoTop` は呼ばない）。他のキーでは何もしない（`q` / `ctrl+c` は `Update` の先頭の共通処理が扱う）。`helpLines() []string` は help-screen spec の 15 行を、キーの表記を `pad(key, 16)`（`view.go` の `pad`）で揃えた固定の列として返す。`renderHelp()` は 1 行目 `キーバインド`、空行、`helpLines()`、高さ − 1 まで空行で埋め（超える分は `cut` で切る）、最終行に `m.footer("? / Esc 閉じる  q 終了")` を置き、各行を `ansi.Truncate(l, m.width, "…")` で切る。`view.go` の `render()` の `screen` の振り分けに `screenHelp` を足す。`model.go` の `Update` では `KeyPressMsg` の振り分けに `screenHelp` → `updateHelpKey` を足す。この分岐は `screenConfirm` の分岐の直後、`key == "a"` の共通処理（s11 の `t`、この change の `o` を含む）より前に置く（`Update` は `a` を画面判定より先に処理するので、後ろに置くとヘルプ画面で `a` / `t` / `o` が効いてしまう）。`WindowSizeMsg` で `refreshDetail()` を呼ぶ条件は現状（`screen` がカード詳細または PR 詳細のとき）のままで良いことを確認する（ヘルプをキューから開いた状態では `detail.card.Issue` が nil なので、条件を広げてはいけない）。`fetchedMsg` / `postedMsg` / `toggledMsg` / `browsedMsg` はヘルプ画面でも共通処理のままにする
- [x] 4.2 `internal/ui/help_test.go` で検証する。検証する内容は次のとおり
  - `example` のバックログを選んだ `Model` に `?` を与えると画面はヘルプになり、もう一度 `?` を与えると画面はキューに戻り、`tab` はバックログ、`cursor` は 0 のまま
  - issue 108 のカード詳細で `x` を与えてから `?` → `Esc` を与えると、画面はカード詳細に戻り、`detail.card.Issue.Number` は 108 で、`View` に `(+1 行)` は無い（展開状態が残る）
  - PR 詳細で `?` → `Esc` を与えると画面は PR 詳細に戻り `currentPR().Number` は 131
  - s10 の確認画面で `?` を与えると画面は確認画面のままで nil が返る
  - ヘルプ画面で `j` / `2` / `Enter` / `a` / `t` / `o` / `R` / `x` / `g` / `p` を 1 つずつ与えると、どれも nil が返り、画面はヘルプのままで、`tab` と `cursor` は変わらず、`Calls` は空で、スタブ `Editor` は呼ばれない
  - ヘルプ画面で `q` を与えると `tea.QuitMsg` を生むコマンドが返る
  - 今やるタブでヘルプを開いた `Model` に issue 108 を含まない `fetchedMsg` を渡すと画面はヘルプのままで、`?` で戻ると今やるタブの行は新しい `Result` のもの
  - 幅 80・高さ 24 の `View`: 1 行目は `キーバインド`。`j / k / ↑ / ↓` と `行移動（キュー）/ スクロール（詳細）` を含む行、`o` で始まり `ブラウザで開く` を含む行、`R` で始まり `全件再取得（キュー）` を含む行、`?` で始まり `ヘルプを開く / 閉じる` を含む行、`p` で始まり `表とプレビューの切替` を含む行がこの順にある。キーの行（2 行目の空行の次から次の空行まで）は 15 行。`merge` / `新規` / `即着手` / `カンバン` / `絞り込み` / `review thread` を含まない。最終行に `? / Esc 閉じる` と `q 終了` があり、`Enter 開く` と `1-4/Tab タブ` は無い
  - 幅 80・高さ 10 の `View`: 行数は 10、1 行目は `キーバインド`、最終行に `? / Esc 閉じる` があり、`p` で始まる行は無い
  - `New` 直後に `?` を与えた `View`: 1 行目は `キーバインド`、最終行に `取得中` がある

## 5. フッタと MODIFIED の反映

- [x] 5.1 `internal/ui/view.go` の `queueHint()` を `Enter 開く  a 回答  t todo  o ブラウザ  R 更新  ? ヘルプ  q 終了`（1 ペインでの ` p プレビュー` / ` p 一覧` の追加は s08 どおり）に直す。`detail.go` の `detailHint()` を、カード詳細は `Esc 戻る  Tab PR 選択  Enter PR を開く  x 展開  g PR へ  ? ヘルプ  a 回答  t todo  o ブラウザ  q 終了`（`PRs` が空なら `Tab PR 選択  Enter PR を開く  g PR へ` を省く）、PR 詳細は `Esc 戻る  x 展開  g issue へ  ? ヘルプ  a 回答  o ブラウザ  q 終了` に直す（`? ヘルプ` は `a 回答` の前。カード詳細で幅 80 でも `? ヘルプ` が見える）。`ansi.StringWidth` でキューのヒントが 64 列、PR 詳細が 66 列、カード詳細が 101 列であることをテストで確かめる
- [x] 5.2 既存テストを直す。`view_test.go` の `TestFooterShowsOnlyImplementedKeys` は、`Enter 開く` / `a 回答` / `t todo` / `o ブラウザ` / `R 更新` / `? ヘルプ` / `q 終了` を含み `j/k 移動` / `1-4/Tab タブ` / `m merge` / `Esc 戻る` を含まない検証にし、`New` 直後（幅 80）の `View` の最終行に `Enter 開く` と `q 終了` と `取得中` が同時にある検証を足す。`model_test.go` の「未実装のキーは何も変えない」から `o` / `R` / `?` を外す。`detail_test.go` の `TestDetailFooters` は幅 120 にし、カード詳細のフッタに `o ブラウザ` / `? ヘルプ` / `q 終了` があり `R 更新` / `j/k スクロール` が無いこと、PR 詳細のフッタに `o ブラウザ` / `? ヘルプ` / `q 終了` があり `R 更新` が無いことを足す。`TestFooterWithoutPRs` に `o ブラウザ` を足す。s09 の「詳細画面ではタブ切替キーが効かない」の検証に `R` を足す。s10 の `tui-entrypoint`「初期フレームに `sugi-loop` と `q 終了`」が幅 80 で通ることを確認する

## 6. 手動確認と最終確認

- [ ] 6.1 手動確認: 稼働リポジトリ 1 件を設定した状態（V-2）で起動し、次を確認する。キュー画面で `o` を押すと選択行の issue / PR がブラウザで開く。カード詳細の `o` は Issue を、PR 詳細の `o` は PR を開く。`R` でフッタにスピナーと `取得中` が出て、完了後にヘッダの `↻ HH:MM` が更新される。取得中に `R` を連打しても取得は 1 回で終わる。`?` でヘルプが開き、`?` と `Esc` で元の画面に戻る。幅 80 の端末でキュー画面のフッタに `q 終了` まで見える
- [x] 6.2 `gofmt -l .` が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する（`golangci-lint run ./...` も通す）。`internal/fetch` / `internal/classify` / `internal/gh` / `internal/model` / `internal/action` / `cmd/sugi-loop-cli` の既存テストが変更なしで通ることを確認する
