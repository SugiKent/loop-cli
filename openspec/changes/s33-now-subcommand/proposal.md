issue: #36

## Why

`loop-cli` のサブコマンドは `version` と `update` の 2 つだけで（`cmd/loop-cli/main.go:49`）、
「今やる」に何が並んでいるかは TUI を人が開かないと分からない。AI agent（Claude Code など）は
TUI を操作できないので、同じ判定結果を使えず、人の出番を自分で拾えない。

分類そのものは既にライブラリとして揃っている（`internal/fetch` + `internal/classify`）。
足りないのは、それを 1 回の実行で読める形に出す入口だけである。

## What Changes

- `loop-cli now` サブコマンドを足す。設定ファイルの `repos` を全件取得し、`[1]今やる` に入るカードだけを
  TUI と同じ優先度順で標準出力に出して終わる（TUI は起動しない）
- 分類は `fetch.Fetch`（`internal/fetch/fetch.go:50`）と `classify.Card` をそのまま呼ぶ。判定は 1 行も書かない
- サブコマンドが設定ファイルと `gh` を使うのは `now` が初めてなので、`tui-entrypoint` の
  Requirement「引数はサブコマンドに振り分ける」を書き換える（`version` / `update` は今までどおり
  設定ファイルを読まず `gh` も呼ばない）
- 使い方（`usage`）と README にも `now` の 1 行を足す

## 確定した判断

1. **分類は既存のものをそのまま使う。** `fetch.Fetch`（`internal/fetch/fetch.go:50`）が open issue / PR を
   検索して詳細を並行取得し、`classify.Card` が 4 タブへ振り分ける。「今やる」は `model.TabNow`
   （`internal/model/model.go:62`）。`now` はこの結果を `Card.Result.Tab == TabNow` で絞るだけで、
   新しい判定ルールを持たない。`docs/domain/issue-driven-sdd/human-turn-signals.md` が正本のまま変わらない。
2. **並び順は TUI と同じ。** `internal/ui/rows.go` の `buildRows` は、カードを優先度の昇順で並べ、
   同じ優先度なら主体の `UpdatedAt` が新しいものを先に、それも同じならリポジトリ名の昇順、
   最後に番号の昇順で並べる。`now` はこの順をそのまま使い、並べ替えのキーを新しく決めない。
3. **「いま人が何をすべきか」の 1 行は既にある。** `model.Result.Summary`（`internal/model/model.go:131`）に
   `PR #131 の質問に答える` のような文字列が入っている（`internal/classify/classify.go`）。これをそのまま出す。
4. **スナップショットは読まないし書かない。** `~/.cache/loop-cli/snapshot.json` は TUI が取得に成功するたびに
   書くもので（`internal/snapshot/snapshot.go`）、TUI を起動していない環境では存在しないか古い。
   agent が読む値としては当てにできないので、`now` は毎回 `gh` で取得する。TUI の起動直後表示を
   壊さないよう、`now` の結果で上書きもしない。
5. **部分失敗は結果を捨てない。** `fetch.Fetch` は詳細取得 1 件の失敗を `Result.Errors`
   （`internal/fetch/fetch.go:26`）に積んで一覧は返す。TUI がフッタに `詳細取得の失敗 N 件` を出すのと同じ扱いで、
   `now` も一覧を出したうえで失敗を伝え、終了コードは 0 にする。検索そのものの失敗（`Fetch` が error を返す）と
   `gh` の未認証・不在は終了コード 1 で終わる。
6. **`gh` の確認は TUI と同じ。** 実行の先頭で `gh.Client.Check` を呼び、`gh が見つかりません` /
   `gh の認証に失敗しました` の 2 行を標準エラーに出して終了コード 1 で終わる
   （`tui-entrypoint` の Requirement「起動失敗は標準エラーに出て終了コード 1 になる」と同じ文言）。
7. **onboarding のフォームは出さない。** `now` は端末とは限らない場所（agent のサブプロセス）で走る。
   設定ファイルが無ければフォームに入らず、その場で終わる。

## 未確定の判断

### Q1. 出力の形式をどれにするか
- 選択肢 A（推奨）: **JSON だけを出す。** `loop-cli now` は常に 1 つの JSON オブジェクト（取得時刻・件数・
  カードの配列・部分失敗の配列）を標準出力に出す。agent は `loop-cli now | jq` でそのまま読める。
  人が直接打つと JSON が流れる
- 選択肢 B: 既定は人が読める表（TUI の今やるタブと同じ列）にし、`--json` を付けたときだけ JSON にする。
  人も agent も自然に使えるが、出力の実装とテストが 2 系統になる
- 選択肢 C: `loop-cli-dev classify` と同じタブ区切り 1 行 1 件にする。実装は最小だが、
  タイトルにタブや改行が入ると agent 側で壊れる
- 依存: なし

### Q2. サブコマンドの名前と、出す範囲
- 選択肢 A（推奨）: **`loop-cli now`。今やるタブだけを出す。** 他のタブ（バックログ / 進行中 / 異常）が
  必要になったら別 issue で足す
- 選択肢 B: `loop-cli status --tab now|backlog|in-progress|abnormal`（既定は `now`）。4 タブすべてを
  今回出せるようにする。実装は分岐 1 つ分増える
- 依存: なし

### Q3. 設定ファイルが無い環境で動かせるようにするか
- 選択肢 A（推奨）: **設定ファイル必須。** `~/.config/loop-cli/config.yml` が無ければ
  `設定ファイルがありません: <パス>` を標準エラーに出して終了コード 1。agent は人と同じ設定を使う
- 選択肢 B: `--repo owner/name` を繰り返し指定でき、指定があれば設定ファイルを読まずにそのリポジトリだけを見る。
  設定ファイルを置いていない CI や別マシンの agent からも呼べるようになる
- 依存: なし

### Q4. 1 件あたりどこまで出すか
- 選択肢 A（推奨）: **一覧の 1 行分までを出す。** 1 件につき、優先度と種別（`質問` / `方針` / `merge` など）、局面（A〜G）、リポジトリ名と番号、issue か PR かの別、タイトルと URL、要約、最終更新時刻、ラベル、紐づく PR の番号とラベルを出す。
  本文とコメントは出さない。agent は URL を見れば `gh` で本文を取れる
- 選択肢 B: A に加えて、本文と会話コメントの全文も含める。agent が 1 回の実行で答えまで作れるが、
  出力が数百 KB になり、`gh` から取り直せる内容を二重に持つ
- 依存: なし

## Capabilities

### New Capabilities

- `now-command`: `loop-cli now` が設定のリポジトリを取得し、「今やる」のカードだけを機械が読める形で
  標準出力に出して終わる。取得の失敗・部分失敗の扱いと終了コードを含む

### Modified Capabilities

- `tui-entrypoint`: Requirement「引数はサブコマンドに振り分ける」に `now` を足し、
  「サブコマンドは設定ファイルを読まず `gh` も呼ばない」を `version` / `update` だけの規則に直す。
  使い方（`unknown command` のときに出す文面）にも `now` の 1 行を足す

## Impact

- `cmd/loop-cli/main.go`: `run` の `switch` に `now` を足し、設定の読み込み → `Check` → `Fetch` → 出力を行う関数を足す。
  `usage` に 1 行足す
- `cmd/loop-cli` の新しいファイル（出力の組み立てとその単体テスト）
- `internal/fetch` / `internal/classify` / `internal/model` / `internal/ui`: 変更しない（読むだけ）
- `README.md`: 「更新」の節の近くに `now` の説明を足す
- `docs/mvp` と `docs/domain/issue-driven-sdd/human-turn-signals.md`: 変更しない（判定は変わらない）
