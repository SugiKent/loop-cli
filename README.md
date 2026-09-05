# sugi-loop

複数の GitHub リポジトリを横断して、**人が手を動かすべき issue / PR だけ**を 1 本のキューに優先度順で並べる TUI です。

[issue-driven-sdd](https://github.com/SugiKent/sugiken-dev-plugin-public/tree/main/plugins/issue-driven-sdd) のように、AI の routine が GitHub Issue のラベルで propose → apply → archive を回す運用では、AI は「人の判断待ち」で止まります。リポジトリが増えるほど、どこで何が待っているかを GitHub の通知やタブから拾い直す手間が増えます。sugi-loop はその待ちだけを集めて、先頭から捌けるようにします。

データ層は `gh` CLI です。サーバーもトークン管理も持たず、手元で動きます。

## 前提

- Go 1.26 以上（`go.mod` の `go 1.26.6`）
- [GitHub CLI](https://cli.github.com/) が PATH にあり、認証済みであること。起動時に `gh auth status` を実行して確認します
- 対象リポジトリの issue に `stage:todo` / `stage:propose` / `stage:apply` / `stage:archive` などのラベル運用があること。ラベルが無いリポジトリからは何も出ません
- 回答を書くためのエディタ（`$EDITOR` または設定ファイルの `editor`）

## インストールと起動

```sh
go install github.com/SugiKent/sugi-loop/cmd/sugi-loop@latest
sugi-loop
```

リポジトリを clone して直接動かす場合:

```sh
go run ./cmd/sugi-loop
```

## 初回起動

設定ファイル `~/.config/sugi-loop/config.yml` が無いとき、起動すると入力フォームが出ます。順に `repos`（owner/name を 1 行に 1 つ）、`merge_method`、`notify`、`editor` を聞き、回答を設定ファイルに書き出してからキュー画面に進みます。ディレクトリは `0700`、ファイルは `0600` で作られます。

途中で中止（Ctrl+C）した場合、設定ファイルは書かれません。標準入力が端末でない場合はフォームを出さず、`設定ファイルがありません: <path>` で終了します。

設定ファイルが既にあるときはフォームを出しません（壊れた設定を上書きしないため）。読み込みに失敗した場合はその内容をエラーとして表示します。

## 設定ファイル

`~/.config/sugi-loop/config.yml`:

```yaml
repos:
  - org/app
  - org/web
refresh_interval_sec: 120
merge_method: squash
editor: $EDITOR
notify: true
```

| キー | 意味 | 既定値 | 取り得る値 |
| --- | --- | --- | --- |
| `repos` | 監視するリポジトリ。1 件以上必須 | なし（必須） | `owner/name` の文字列、または `{name: owner/name, merge_method: ...}` のマッピング |
| `refresh_interval_sec` | 再取得の間隔（秒） | `120` | 1 以上の整数 |
| `merge_method` | merge の方式 | `squash` | `squash` / `merge` / `rebase` |
| `editor` | 回答の下書きを開くコマンド。環境変数を展開してから空白で分割し、シェルを通さずに実行する | `$EDITOR` | 例: `vim`、`code --wait` |
| `notify` | デスクトップ通知を出すか | `true` | `true` / `false` |

`repos` の要素をマッピングで書くと、そのリポジトリだけ `merge_method` を上書きできます。

```yaml
repos:
  - name: org/app
    merge_method: merge
  - org/web
```

現時点で TUI が実際に使うのは `repos` と `editor` だけです。`refresh_interval_sec` / `merge_method` / `notify` は読み込みと検証の対象ですが、まだ動作には影響しません（自動再取得・merge 操作・通知が未実装のため）。

未知のキーはエラーになります。`repos` が空、`owner/name` 形式でない、`merge_method` が 3 つ以外、`refresh_interval_sec` が 0 以下のときも起動に失敗します。

## 画面とキー操作

起動すると全リポジトリの open issue / open PR を取得し、issue 1 件とそれに紐づく PR 群を 1 枚の**カード**にまとめて 4 つのタブに振り分けます。PR は、タイトルが `[propose] #140` のような形か、本文に `Refs #140` / `Closes #140` があるとき、その issue のカードに入ります。紐づかない PR は単独の行になります。

| タブ | 入るもの |
| --- | --- |
| `[1]今やる` | 質問への回答、方針の決定、merge 待ちなど、人の出番があるもの |
| `[2]バックログ` | 段階ラベルが無く、着手を承認すればよい issue |
| `[3]進行中` | AI が作業中で、人は待っていればよいもの |
| `[4]異常` | 段階ラベルが 2 つ以上付いた壊れた状態 |

行の列は 優先 / 種別 / リポジトリ / 番号 / タイトル / 経過 です。種別は `質問` / `方針` / `merge` / `todo 候補` / `異常` / `進行中` / `その他`。

端末が 80 桁 20 行以上あるときは、上に一覧・下に選択行のプレビュー（本文と会話コメント）を同時に出します。それより狭いときは一覧だけを出し、`p` で一覧とプレビューを切り替えます。

### キュー画面

| キー | 動作 |
| --- | --- |
| `j` / `↓`、`k` / `↑` | 行を移動する |
| `1` `2` `3` `4` | タブを選ぶ |
| `Tab` | 次のタブへ |
| `Enter` | 選択行のカード詳細を開く |
| `a` | 選択行に回答する |
| `p` | 一覧とプレビューを切り替える（2 ペイン表示のときは何もしない） |
| `q` / `Ctrl+C` | 終了する |

### カード詳細（`Enter`）

issue のタイトル・現在の局面・段階ラベルとバッジ・`depends on:` の参照、紐づく PR の一覧をヘッダに出し、本文と会話コメントを下に並べます。

| キー | 動作 |
| --- | --- |
| `j` / `↓`、`k` / `↑` | 1 行スクロールする |
| `PgDn` / `PgUp` | 1 画面スクロールする |
| `x` | 折りたたまれている AI コメントの全文を出す / 戻す |
| `Tab` | 紐づく PR が複数あるとき、対象の PR を切り替える |
| `Enter` / `g` | 対象の PR 詳細を開く |
| `a` | この issue に回答する |
| `Esc` | キュー画面へ戻る |

### PR 詳細

`未確定の判断` の件数、紐づく issue、checks の結果、本文、会話コメント、review thread を並べます。

| キー | 動作 |
| --- | --- |
| `j` / `↓`、`k` / `↑`、`PgDn` / `PgUp` | スクロールする |
| `x` | 折りたたまれている AI コメントの全文を出す / 戻す |
| `g` | 紐づく issue のカード詳細へ移る |
| `a` | この PR に回答する |
| `Esc` | カード詳細から来ていればそこへ、そうでなければキュー画面へ戻る |

### 回答（`a`）

`a` を押すと、対象の最新の AI コメントに含まれる質問（`## Q1.` 形式）から `Q1: A` のような回答テンプレートを作り、`editor` で開きます。推奨と書かれた選択肢があればその記号が既定値として入ります。

エディタを閉じると、本文に応じて次のように進みます。

- 空（空白のみ）: 投稿せずに中止する
- `<!-- routine -->` を含む: 確認画面に入る。TUI からの投稿は人の発言でなければならないため、この本文は投稿できない（`e` で編集に戻るか `Esc` で中止する）
- `blocked-by:` で始まる行がある: 確認画面に入る。該当行と下書き全文を確認してから `y` で投稿、`e` で編集に戻る、`Esc` で中止する
- それ以外: そのまま issue / PR にコメントとして投稿する

投稿の結果はフッタに出ます。投稿しても一覧は取り直しません。

### 再取得について

一覧を取得するのは起動時の 1 回だけです。最新の状態を見るには再起動してください（キー操作による更新と自動更新は未実装です）。

## sugi-loop-cli（開発補助）

fixture を採取したり、分類の結果や通知を手元で確認するための CLI です。`fixture capture` と `classify` はリポジトリのルートで実行します。

```sh
go run ./cmd/sugi-loop-cli help

# 指定リポジトリの open issue / PR を採取し、伏せ字にして
# internal/gh/testdata/fixtures/<alias>/ に保存する
go run ./cmd/sugi-loop-cli fixture capture --repo org/app --alias example

# 採取済み fixture を分類し、4 タブ別にタブ区切りで出力する（gh は呼ばない）
go run ./cmd/sugi-loop-cli classify --fixture example

# デスクトップ通知を 1 件出す
go run ./cmd/sugi-loop-cli notify test
```

## うまく動かないとき

**`gh が見つかりません`**
GitHub CLI をインストールして PATH に通してください。

**`gh の認証に失敗しました`**
`gh auth login` を実行してください。sugi-loop は起動のたびに `gh auth status` で確認します。

**`設定ファイルがありません: ~/.config/sugi-loop/config.yml`**
標準入力が端末でないため初回フォームを出せませんでした。端末から起動するか、設定ファイルを手で置いてください。

**キューが空で「stage:\* ラベルの無いリポジトリは何も出ません。」と出る**
設定した `repos` の issue に段階ラベルが付いていません。issue-driven-sdd の `routines-setup` を回したリポジトリを設定してください。

**フッタに `詳細取得の失敗 N 件` が出る**
一部の issue / PR の詳細取得に失敗しています。一覧そのものは表示されています。`gh` のレート制限や権限を確認してください。

**`a` を押すと `editor が設定されていません` と出る**
設定ファイルの `editor` が空で、環境変数 `EDITOR` も設定されていません。どちらかを設定してください。

**デスクトップ通知が出ない**
`go run ./cmd/sugi-loop-cli notify test` で単体で確認できます。OS 側の通知許可設定を確認してください（TUI 自体はまだ通知を出しません）。
