## Purpose

AI agent が TUI を開かずに「今やる」の状態を読めるようにする。`loop-cli now` は設定したリポジトリを取得し、
人の出番と判定されたカードだけを機械が読める形で標準出力に出して終わる。

## ADDED Requirements

### Requirement: now サブコマンドは今やるのカードだけを JSON で出す

`loop-cli now` は、設定ファイルの `repos` を取得し、`[1]今やる` に入るカードだけを 1 つの JSON オブジェクトとして
標準出力に MUST 書き、終了コード 0 で終わる。TUI は起動しない。

判定と並び順は TUI と同じものを使い、`now` は独自の判定も並べ替えも行わない。カードは優先度の昇順、
同じ優先度では主体の最終更新時刻が新しいものを先に、それも同じならリポジトリ名の昇順、
最後に番号の昇順で並ぶ（`queue-screen` の並び順と同じ）。「今やる」に 1 件も無いときは `items` を空配列にして
終了コード 0 で終わる。

オブジェクトの最上位は次の 4 つのキーを持つ。

| キー | 中身 |
| --- | --- |
| `fetched_at` | 取得を始めた時刻（RFC 3339） |
| `count` | `items` の件数 |
| `items` | カードの配列。空でも `[]` を出す |
| `errors` | 取得の部分失敗を 1 件 1 要素にした文字列の配列。空でも `[]` を出す |

`items` の 1 要素は 1 枚のカードで、次のキーを持つ。

| キー | 中身 |
| --- | --- |
| `situation` | 局面の記号。「今やる」に入るのは `A` / `B` / `C` / `D` / `G` / `other` の 6 つ |
| `kind` | 画面の表に出す種別。`質問` / `方針` / `merge` / `その他` のいずれか |
| `priority` | 優先度の整数。小さいほど先 |
| `summary` | 「いま人が何をすべきか」の 1 行 |
| `repo` | `owner/name`。1 枚のカードは 1 リポジトリ分しか持たない |
| `subject` | その局面を出した issue または PR を指す `type`（`issue` または `pr`）と `number` |
| `issue` | カードの issue の `number`・`title`・`url`・`labels`・`updated_at`。PR 単独のカードでは `null` |
| `prs` | カードの PR の配列。無ければ `[]` |

`subject` は番号だけを持ち、内容は `issue` か `prs` の該当する要素から引く（同じ値を二重に出さない）。
issue の段階（`stage:propose` など）は `issue.labels` に入るので、別の欄を作らない。

`prs` の 1 件は次のキーを持つ。`mergeable` から `unresolved_threads` までは PR 詳細画面が出している状態で、
詳細の取得に失敗した PR では `null` になる。値が空である場合と、取得そのものが失敗した場合を区別するためである。

| キー | 中身 |
| --- | --- |
| `number` / `title` / `url` / `labels` / `draft` / `updated_at` | 検索結果から引く PR 自身の値 |
| `undecided` | 本文 1 行目の `未確定の判断: N 件` の N。1 行目に書かれていなければ `null` |
| `mergeable` | `gh` の `mergeable`（`MERGEABLE` / `CONFLICTING` など） |
| `merge_state_status` | `gh` の `mergeStateStatus`（`CLEAN` / `BLOCKED` など） |
| `review_decision` | `gh` の `reviewDecision`。レビューが無ければ空文字 |
| `checks_green` | checks が merge を妨げない状態かどうかの真偽値。判定は merge のガードと同じものを使う |
| `checks` | チェックごとの `name` と `state` の配列。`state` は CheckRun なら結論（空なら進行状況）、StatusContext なら state。0 件なら `[]` |
| `unresolved_threads` | 未解決の review thread の件数 |

#### Scenario: 今やるのカードが JSON で出る

- **WHEN** 「今やる」に 2 件、他のタブに 3 件のカードがある取得結果で `now` の出力を組み立てる
- **THEN** 出力は 1 つの JSON オブジェクトで、`count` は 2、`items` の長さは 2 であり、他のタブのカードは含まれない

#### Scenario: 優先度順に並ぶ

- **WHEN** 優先度 3 のカードと優先度 1 のカードをこの順で含む取得結果で `now` の出力を組み立てる
- **THEN** `items[0]` は優先度 1 だったカード、`items[1]` は優先度 3 だったカードである

#### Scenario: 優先度が同じなら更新の新しいものが先

- **WHEN** 同じ優先度で最終更新時刻だけが違う 2 枚のカードで `now` の出力を組み立てる
- **THEN** `items[0]` は最終更新時刻が新しい方のカードである

#### Scenario: 今やるが空でも成功する

- **WHEN** 「今やる」に 1 件も無い取得結果で `now` を実行する
- **THEN** `count` は 0、`items` は `[]`、`errors` は `[]` で、終了コードは 0 である

#### Scenario: PR が主体のカード

- **WHEN** issue #108 と PR #131 を持つカードが PR #131 の局面で「今やる」に入っている取得結果で出力を組み立てる
- **THEN** その要素の `subject` は `type` が `pr` で `number` が 131、`issue.number` は 108、`prs` は番号 131 の要素を含む

#### Scenario: PR 単独のカード

- **WHEN** issue に紐づかない PR 1 本だけのカードが「今やる」に入っている取得結果で出力を組み立てる
- **THEN** その要素の `issue` は `null` で、`subject.type` は `pr` である

#### Scenario: PR の状態が出る

- **WHEN** 本文 1 行目が `未確定の判断: 0 件`、`mergeable` が `MERGEABLE`、`mergeStateStatus` が `CLEAN`、
  checks が成功した CheckRun 1 件、未解決の review thread が 1 件の PR を含む取得結果で出力を組み立てる
- **THEN** その PR の `undecided` は 0、`mergeable` は `MERGEABLE`、`merge_state_status` は `CLEAN`、
  `checks_green` は `true`、`checks` はその 1 件を含み、`unresolved_threads` は 1 である

#### Scenario: PR の詳細が取れなかったとき

- **WHEN** merge 状態と review thread の取得に失敗した PR を含む取得結果で出力を組み立てる
- **THEN** その PR は状態の 6 つの欄（`mergeable`、`merge_state_status`、`review_decision`、`checks_green`、`checks`、`unresolved_threads`）をすべて `null` にして、`number` と `title` と `url` と `labels` を出す

### Requirement: now は設定ファイルを読み gh を呼ぶ

`loop-cli now` は、TUI と同じ設定ファイル（`~/.config/loop-cli/config.yml`）の `repos` を対象として MUST 使い、
取得の前に `gh` の存在と認証を MUST 確認する。設定ファイルが無いときに onboarding のフォームは出さない
（`now` は端末とは限らない場所で走るため）。

次の失敗は標準エラーに書いて終了コード 1 で終わり、標準出力には何も書かない。文言は `tui-entrypoint` の
Requirement「起動失敗は標準エラーに出て終了コード 1 になる」と同じものを使い、ここでは定義し直さない。

- `config.DefaultPath` の失敗
- 設定ファイルが無い: `設定ファイルがありません: <パス>` の 1 行（端末かどうかによらず、フォームには入らない）
- 設定ファイルの読み込みの失敗: パスと原因を含む 1 行
- `gh` の確認の失敗: `gh` が PATH に無いときと未認証のときの 2 行、それ以外は 1 行
- issue / PR を検索する呼び出し自体の失敗: そのエラーの 1 行

`loop-cli now` は引数を取らない。`now` の後ろに引数やフラグがあれば、標準エラーにそれを含む 1 行を書いて
終了コード 1 で MUST 終わる（黙って捨てない）。

#### Scenario: 設定ファイルが無い

- **WHEN** `~/.config/loop-cli/config.yml` が無い状態で `now` を実行する
- **THEN** 標準エラーに `設定ファイルがありません` とそのパスが出て、onboarding のフォームは出ず、終了コードは 1 である

#### Scenario: gh が無い

- **WHEN** `gh` の確認が `exec.ErrNotFound` を包んだエラーを返す状態で `now` を実行する
- **THEN** 標準エラーは 2 行で、1 行目に `gh が見つかりません`、2 行目に `https://cli.github.com/` を含み、終了コードは 1 である

#### Scenario: 検索の失敗

- **WHEN** 取得が失敗する状態で `now` を実行する
- **THEN** 標準出力は空で、標準エラーにそのエラーが出て、終了コードは 1 である

#### Scenario: 余分な引数

- **WHEN** 引数 `now --repo org/app` で実行する
- **THEN** 標準出力は空で、標準エラーに `--repo` を含む 1 行が出て、終了コードは 1 である

### Requirement: now の部分失敗は結果を捨てずに errors へ入れる

一部のリポジトリ・issue・PR の取得だけが失敗した場合、`loop-cli now` は取得できた範囲のカードを MUST 出力し、
失敗を `errors` の要素として MUST 入れ、終了コード 0 で終わる。TUI がフッタに `詳細取得の失敗 N 件` を出して
一覧を残すのと同じ扱いにする（`queue-screen`）。

`errors` には、issue / PR 1 件ごとの詳細取得の失敗に加えて、リポジトリのラベル一覧の取得の失敗も入る
（`card-fetch`）。ラベル一覧が取れなかったリポジトリは運用方式を判定できず、既定の方式として分類されるため、
`errors` が空でないときの `items` は不完全であり得る。

#### Scenario: 詳細取得の一部が失敗する

- **WHEN** 1 件の PR の詳細取得だけが失敗した取得結果で `now` を実行する
- **THEN** 標準出力の `items` には取得できたカードが並び、`errors` はその失敗 1 件を含み、終了コードは 0 である

### Requirement: now はスナップショットを読み書きしない

`loop-cli now` は、実行のたびに `gh` から取得し、`~/.cache/loop-cli/snapshot.json` を MUST 読まず MUST 書かない。
スナップショットは TUI の起動直後の表示のためのもので、TUI を起動していない環境では古いか存在しないため、
`now` の結果としては使わない。

#### Scenario: スナップショットがあっても取得する

- **WHEN** `~/.cache/loop-cli/snapshot.json` がある状態で `now` を実行する
- **THEN** 出力は取得した内容から組み立てられ、そのファイルの中身と更新時刻は実行の前後で同じである
