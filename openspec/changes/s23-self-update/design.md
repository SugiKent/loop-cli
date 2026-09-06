# 設計

## 全体像

```
cmd/sugi-loop/main.go   run(args, stdout, stderr) int
  ├ 引数なし        → 今までの TUI 起動（ui.Options.CheckUpdate に updateChecker(client) を渡す）
  ├ version         → client.Current() の版を 1 行出す
  ├ update          → runUpdate(ctx, client, …): Latest → 違えば Install → 結果を出す
  └ それ以外        → unknown command: <arg> と使い方（終了コード 1）

internal/version
  version.Client（gh.Client と同じく実行関数を構造体の欄に持つ）
    Current() (path, version string, local bool)     debug.ReadBuildInfo()
    Latest(ctx, path) (string, error)                go list -m -json <path>@latest
    Install(ctx, path, stdout, stderr) error         go install <path>/cmd/sugi-loop@latest
```

`internal/version` は `go` をサブプロセスとして呼ぶ。`internal/gh` の `Client` と同じく、実行する関数を構造体の欄に持ち（`gh.Client.run` / `openBrowser` と同じ形）、テストは自前の関数を入れた `Client` を組み立てる。`cmd/sugi-loop` 側は `ensureConfig(path, isTerminal, form)` と同じ引数注入で、`runUpdate` / `updateChecker` にこの `Client`（テストではスタブ）を渡す。テストで本物の `go` とネットワークを叩かない。

## 最新版の調べ方

`go list -m -json <module>@latest` を使う。理由は次の 3 つ。

- 利用者の `GOPROXY` / `GOPRIVATE` / `GOFLAGS` をそのまま尊重する。proxy.golang.org へ直接 HTTP を投げると、社内 proxy 環境で `go install` は通るのに確認だけ落ちる、という食い違いが起きる
- `go install ...@latest` が実際に解決する版と同じ答えが返る。比較の対象が揃う
- 標準ライブラリの HTTP クライアントも JSON の取り扱いも書かずに済む

`go` は利用者のカレントディレクトリで走るので、`cmd.Dir` を `os.TempDir()`（module の外）に固定する。main module のチェックアウトの中では `@latest` は解決できたが、**vendor ディレクトリを持つプロジェクトの中では `-mod=vendor` が自動で効き、`go list` も `go install` も `cannot query module due to -mod=vendor` で落ちる**（実測）。開発中のプロジェクトのディレクトリで打つ道具なので、この状況は普通に起きる。`cmd.Dir` を移すと解消することも実測した。

戻り値の JSON は `{"Path":…,"Version":"v0.0.0-20260905143706-3ce8a526b4ec",…}` の形で、必要なのは `Version` だけ。タグが 1 つも無い現状では擬似バージョンが返る。

## 手元 build の見分け方

Go 1.26 は作業ツリーでの `go build` にも VCS 由来の擬似バージョン（`v0.0.0-<日時>-<hash>+dirty`）を刻む。`(devel)` になるのは `-buildvcs=false` のときだけなので、版の文字列では `go install <module>@<version>` と区別できない（手元で確認済み）。一方 `vcs.revision` などの build setting は module cache から入れたバイナリには付かない。これを手元 build の目印にする。

## 版の比較

`現在 != 最新` の文字列比較だけで判定する。タグが無く擬似バージョン（`v0.0.0-<日時>-<hash>`）が返る間は辞書順が時系列順と一致し、タグを打った後も「違えば更新できる」で困らない。`golang.org/x/mod/semver` は依存を増やすので入れない。

## 未決事項

docs/mvp に無い機能なので、実装者が選ぶ既定値を各項に 1 つ置く。

| 項目 | 既定値 | 理由 |
| --- | --- | --- |
| 手元 build のときの起動時チェック | 何もしない（ヘッダに出さない） | 開発中のチェックアウトに「更新あり」を出しても意味が無い |
| 手元 build のときの `update` | 比較せず `go install` を実行する | 「明示的に打った以上は入れ直す」が素直 |
| 起動時チェックの時間制限 | 10 秒。超えたら諦める | 画面はチェックを待たない。失敗は無視するので上限だけ決める |
| 起動時チェックの回数 | 起動につき 1 度だけ（自動更新の tick では行わない） | 版はセッション中にまず変わらない |
| チェック失敗時の表示 | 何も出さない | ネットワークが無い場所で赤いエラーが常駐するのは邪魔 |
| ヘッダの表示 | `↑ update`（8 列）を `↻ HH:MM` の左に空白 2 列を空けて出す | フッタのヒント（幅 71 列）に足すと既定幅 80 で押し出される。`↑ sugi-loop update`（18 列）だと既定幅 80 でタブ名が 3 つとも落ちるので短くした。打つコマンドは README に書く |
| 狭い端末での優先順 | ヘッダが幅を超えるとき、タブ名の短縮の後・`↻ HH:MM` の省略の前に `↑ update` を落とす | 時刻の方が毎回見る情報 |
| `update` の実行中の出力 | `go install` の標準出力・標準エラーをそのまま流す | ダウンロードの進捗と失敗理由が見える |
| `go` が PATH に無いとき | 起動時チェックは黙って諦める。`update` は `go が見つかりません` と `https://go.dev/dl/` を出して終了コード 1 | TUI は `go` 無しでも動く |

## やらないこと

- GitHub Releases へのバイナリ配布とダウンロード。リリースのワークフローが無く、Go は README の前提
- TUI の中から更新を実行するキー。動作中のバイナリを置き換えることになり、`update` は shell から打つものとする
- 版の固定（`sugi-loop update v1.2.3`）。タグがまだ 1 つも無い
