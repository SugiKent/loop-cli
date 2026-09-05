## Why

PR の説明文には、設計ドキュメント・仕様の参照先・CI の実行ページなど「読まないと判断できない URL」が貼られる。今の TUI で開けるのは `o`（`gh browse`）の issue / PR そのものだけで、本文の中の URL は目で読んで手で写すか、ブラウザで PR を開いてから辿るしかない。sugi-loop は「人の出番」を捌く手数を減らす道具なので、判断に要るリンクへ到達するのに GitHub を経由させるのは目的を損なう。

本文中のすべての URL を個別のキーに割り当てるのは現実的ではない。`u` で「その PR / issue に含まれる URL の一覧」を 1 画面出し、選んで開く形にすれば、キーは 1 つで済み、URL の件数にも依存しない。

## What Changes

- `u` を押すと URL 一覧画面（キュー / カード詳細 / PR 詳細に続く新しい画面の状態）に移り、対象に含まれる URL を出典付きで並べる。`j` / `k` / `↑` / `↓` で選び、`Enter` で開き、`Esc` で元の画面に戻る。`Enter` で開いても一覧に留まり、続けて別の URL を開ける
- 対象は `o` と同じ規則（画面が見せているもの）で決まる。キュー画面は選択行の主体、カード詳細は `Card.Issue`、PR 詳細は選択中の PR
- URL の収集元は、対象の本文・コメント本文・（PR の）review thread のコメント本文。`https://` / `http://` で始まる裸の URL と Markdown の `[テキスト](URL)` を拾う。`#123` のような GitHub の参照記法と、checks の詳細 URL（`statusCheckRollup` の `detailsUrl` / `targetUrl`）は対象外
- 一覧の 1 行は `<印><出典> <リンクテキスト>  <URL>` の形（例: `▶ [本文] 設計メモ  https://example.com/design`）。同じ URL が複数回出てきたら初出の 1 件だけを残す
- URL が 1 件も無いときは一覧画面に移らず、フッタのステータスに `URL がありません` を出す
- 任意の URL を開く経路として `GHClient` に `OpenURL(ctx, url)` を足す。`Client` は macOS の `open <url>`（それ以外は `xdg-open <url>`）を実行する。`gh browse` は番号しか受け取れないので `o` の経路は使えない。失敗は `o` と同じくフッタに赤で出す
- ヘルプ画面のキー一覧に `u` の行を足し（s21 適用後の 14 行が 15 行になる）、キュー / カード詳細 / PR 詳細のフッタのキーヒントに `u URL` を `? ヘルプ` の直後で足す
- mvp.md のキーバインド表に `u` の行を足す。`u` は表に無いキーなので、実装と正本ドキュメントを揃える

## Capabilities

### New Capabilities

- `url-picker`: `u` で URL 一覧画面を開き、対象の本文・コメント・review thread から URL を集めて出典付きで並べ、選んだ URL を `OpenURL` で開く。0 件のときの扱いと、開けなかったときのステータス表示を含む

### Modified Capabilities

- `gh-client`: `GHClient` interface に `OpenURL(ctx, url)` を足し、`Client` がそれを OS のブラウザ起動コマンドとして実行することを定める
- `gh-fake`: `Fake` が `OpenURL` の呼び出し（URL）を `Calls` に記録することを定める。`Call` に URL の欄が増える
- `queue-screen`: キュー画面のキーの扱いで、`u` を「表に無いキーなので何もしない」から「URL 一覧画面を開く」に変える。フッタのキーヒントに `u URL` を足す
- `card-detail`: カード詳細 / PR 詳細のフッタのキーヒントに `u URL` を足す
- `help-screen`: ヘルプ画面のキー一覧に `u` の行を足す（s21 適用後のキーの行 14 行が 15 行になる）

## Impact

- `internal/gh/gh.go` / `internal/gh/client.go` / `internal/gh/fake.go`: `OpenURL` の追加と `Call` の URL 欄
- `internal/model/parse.go`: 本文から URL とリンクテキストを取り出す純粋な関数を足す（`ParseUndecided` / `ParseQuestions` と同じ場所）
- `internal/ui`: URL 一覧画面を新しいファイルに足し、`Update` のキー振り分けと `View` の画面分岐を広げ、フッタのヒントとヘルプのキー一覧に `u` を加える
- `docs/mvp/mvp.md`: キーバインド表に `u` の行
- テスト: `internal/model` で URL の抽出を、`internal/ui` で画面遷移と一覧の内容と 0 件の扱いと失敗表示を、`internal/gh` で `OpenURL` が渡す引数をそれぞれ検証する
- 先行 change との関係: `s20-fetch-all-details` は archive 済みで、全 issue / PR のコメントと review thread が取得されるため、URL の収集元が揃っている。未 archive の `s21-always-two-pane`（`p` 廃止・常時 2 ペイン）とは `queue-screen` のキー・`card-detail` の 2 つ・`help-screen` のキー一覧という同じ Requirement を書き換えるので、この change の MODIFIED は s21 の delta を土台に写している。実装と archive は s21 → s22 の順で行う
- `OpenURL` は OS のブラウザ起動コマンドを実行するため、`gh` の呼び出し回数は増えない
