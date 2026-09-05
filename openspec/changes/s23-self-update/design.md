# 設計

## 全体像

```
cmd/sugi-loop/main.go   run(args, stdout, stderr) int
  ├ 引数なし        → 今までの TUI 起動（ui.Options.CheckUpdate に version.Check を渡す）
  ├ version         → 現在の版を 1 行出す
  ├ update          → version.Latest → 違えば go install → 結果を出す
  └ それ以外        → unknown command: <arg> と使い方（終了コード 1）

internal/version
  Current() (path, version string)   debug.ReadBuildInfo()
  Latest(ctx, path) (string, error)  go list -m -json <path>@latest
  Install(ctx, path) error           go install <path>/cmd/sugi-loop@latest
```

`internal/version` は `go` をサブプロセスとして呼ぶ。`internal/gh` の `Client` と同じく、コマンドを組み立てて実行する関数をパッケージ変数に持ち、テストで差し替える（s22 の `Client.OpenURL` と同じ書き方）。テストで本物の `go` とネットワークを叩かない。

## 最新版の調べ方

`go list -m -json <module>@latest` を使う。理由は次の 3 つ。

- 利用者の `GOPROXY` / `GOPRIVATE` / `GOFLAGS` をそのまま尊重する。proxy.golang.org へ直接 HTTP を投げると、社内 proxy 環境で `go install` は通るのに確認だけ落ちる、という食い違いが起きる
- `go install ...@latest` が実際に解決する版と同じ答えが返る。比較の対象が揃う
- 標準ライブラリの HTTP クライアントも JSON の取り扱いも書かずに済む

手元の Go 1.26.6 では、モジュールパスと同じ main module のチェックアウトの中で実行しても `@latest` は解決できた（リポジトリのルートで確認済み）。`cmd.Dir` を module の外に移す細工は入れない。

戻り値の JSON は `{"Path":…,"Version":"v0.0.0-20260905143706-3ce8a526b4ec",…}` の形で、必要なのは `Version` だけ。タグが 1 つも無い現状では擬似バージョンが返る。

## 版の比較

`現在 != 最新` の文字列比較だけで判定する。タグが無く擬似バージョン（`v0.0.0-<日時>-<hash>`）が返る間は辞書順が時系列順と一致し、タグを打った後も「違えば更新できる」で困らない。`golang.org/x/mod/semver` は依存を増やすので入れない。

## 未決事項

docs/mvp に無い機能なので、実装者が選ぶ既定値を各項に 1 つ置く。

| 項目 | 既定値 | 理由 |
| --- | --- | --- |
| `(devel)`（手元 build）のときの起動時チェック | 何もしない（ヘッダに出さない） | 開発中のチェックアウトに「更新あり」を出しても意味が無い |
| `(devel)` のときの `update` | 比較せず `go install` を実行する | 「明示的に打った以上は入れ直す」が素直 |
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
