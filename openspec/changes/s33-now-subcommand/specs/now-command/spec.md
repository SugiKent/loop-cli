## Purpose

AI agent が TUI を開かずに「今やる」の状態を読めるようにする。`loop-cli now` は設定したリポジトリを取得し、
人の出番と判定されたカードだけを機械が読める形で標準出力に出して終わる。

## ADDED Requirements

### Requirement: now サブコマンドは今やるのカードだけを JSON で出す

`loop-cli now` は、設定ファイルの `repos` を取得し、`[1]今やる` に入るカードだけを 1 つの JSON オブジェクトとして
標準出力に MUST 書き、終了コード 0 で終わる。TUI は起動しない。

判定と並び順は TUI と同じものを使い、`now` は独自の判定も並べ替えも行わない。カードは優先度の昇順、
同じ優先度では主体の最終更新時刻が新しいものを先に、それも同じならリポジトリ名の昇順、
最後に番号の昇順で並ぶ。「今やる」に 1 件も無いときは `items` を空配列にして終了コード 0 で終わる。

オブジェクトの最上位は次の 4 つのキーを持つ。

| キー | 中身 |
| --- | --- |
| `fetched_at` | 取得を始めた時刻（RFC 3339） |
| `count` | `items` の件数 |
| `items` | カードの配列。空でも `[]` を出す |
| `errors` | 詳細取得の部分失敗を 1 件 1 要素にした文字列の配列。空でも `[]` を出す |

`items` の 1 要素は 1 枚のカードで、次のキーを持つ。`priority` / `situation` / `kind` / `summary` は
カードの分類結果、`subject` はその分類結果を出した issue または PR である。

| キー | 中身 |
| --- | --- |
| `priority` | 優先度の整数。小さいほど先 |
| `situation` | 局面の記号（`A` 〜 `G` と `other`） |
| `kind` | 種別の日本語（`質問` / `方針` / `merge` / `その他` など） |
| `summary` | 「いま人が何をすべきか」の 1 行 |
| `repo` | `owner/name` |
| `subject` | `type`（`issue` または `pr`）・`number`・`title`・`url`・`labels`・`updated_at` を持つオブジェクト |
| `issue` | カードの issue の `number`・`title`・`url`・`labels`。PR 単独のカードでは `null` |
| `prs` | カードの PR の配列。1 件は `number`・`title`・`url`・`labels`・`state`・`draft` を持つ。無ければ `[]` |

#### Scenario: 今やるのカードが JSON で出る

- **WHEN** 「今やる」に 2 件、他のタブに 3 件のカードがある取得結果で `now` の出力を組み立てる
- **THEN** 標準出力は 1 つの JSON オブジェクトで、`count` は 2、`items` の長さは 2 であり、他のタブのカードは含まれない

#### Scenario: 優先度順に並ぶ

- **WHEN** 優先度 3 のカードと優先度 1 のカードを含む取得結果で `now` の出力を組み立てる
- **THEN** `items[0]` は優先度 1 のカードで、`items[1]` は優先度 3 のカードである

#### Scenario: 今やるが空でも成功する

- **WHEN** 「今やる」に 1 件も無い取得結果で `now` を実行する
- **THEN** `count` は 0、`items` は `[]`、`errors` は `[]` で、終了コードは 0 である

#### Scenario: PR が主体のカード

- **WHEN** issue #108 と PR #131 を持つカードが PR #131 の局面 A で「今やる」に入っている取得結果で出力を組み立てる
- **THEN** その要素の `subject.type` は `pr`、`subject.number` は 131、`issue.number` は 108、`prs` は PR #131 を含む配列である

#### Scenario: PR 単独のカード

- **WHEN** issue に紐づかない `docs` PR 1 本だけのカードが「今やる」に入っている取得結果で出力を組み立てる
- **THEN** その要素の `issue` は `null` で、`subject.type` は `pr` である

### Requirement: now は設定ファイルを読み gh を呼ぶ

`loop-cli now` は、TUI と同じ設定ファイル（`~/.config/loop-cli/config.yml`）の `repos` を対象として MUST 使い、
取得の前に `gh` の存在と認証を MUST 確認する。設定ファイルが無いときに onboarding のフォームは出さない
（`now` は端末とは限らない場所で走るため）。

次の失敗は標準エラーに書いて終了コード 1 で終わり、標準出力には何も書かない。

- 設定ファイルが無い: `設定ファイルがありません: <パス>` の 1 行
- 設定ファイルの読み込みの失敗: `config-loading` のエラー文字列の 1 行
- `gh` が PATH に無い: `gh が見つかりません` と `https://cli.github.com/ から GitHub CLI をインストールしてください` の 2 行
- `gh` が未認証: `gh の認証に失敗しました: <gh の標準エラー>` と `gh auth login を実行してください` の 2 行
- issue / PR を検索する呼び出し自体の失敗: そのエラーの 1 行

#### Scenario: 設定ファイルが無い

- **WHEN** `~/.config/loop-cli/config.yml` が無い状態で `now` を実行する
- **THEN** 標準エラーに `設定ファイルがありません` とそのパスが出て、onboarding のフォームは出ず、終了コードは 1 である

#### Scenario: gh が無い

- **WHEN** `gh` が PATH に無い状態で `now` を実行する
- **THEN** 標準エラーに `gh が見つかりません` と `https://cli.github.com/` の 2 行が出て、終了コードは 1 である

#### Scenario: 検索の失敗

- **WHEN** issue の検索が失敗する状態で `now` を実行する
- **THEN** 標準出力は空で、標準エラーにそのエラーが出て、終了コードは 1 である

### Requirement: now の部分失敗は結果を捨てずに errors へ入れる

一部の issue / PR の詳細取得だけが失敗した場合、`loop-cli now` は取得できた範囲のカードを MUST 出力し、
失敗を `errors` の要素として MUST 入れ、終了コード 0 で終わる。TUI がフッタに `詳細取得の失敗 N 件` を出して
一覧を残すのと同じ扱いにする（`queue-screen`）。

#### Scenario: 詳細取得の一部が失敗する

- **WHEN** 1 件の PR の詳細取得だけが失敗した取得結果で `now` の出力を組み立てる
- **THEN** `items` には取得できたカードが並び、`errors` はその失敗 1 件を含み、終了コードは 0 である

### Requirement: now はスナップショットを読み書きしない

`loop-cli now` は、実行のたびに `gh` から取得し、`~/.cache/loop-cli/snapshot.json` を MUST 読まず MUST 書かない。
スナップショットは TUI の起動直後の表示のためのもので、TUI を起動していない環境では古いか存在しないため、
`now` の結果としては使わない。

#### Scenario: スナップショットがあっても取得する

- **WHEN** `~/.cache/loop-cli/snapshot.json` がある状態で `now` を実行する
- **THEN** 出力は `gh` から取り直した内容で、そのファイルの中身と更新時刻は実行の前後で同じである
