# dev-cli Specification

## Purpose
TBD - created by archiving change s06-dev-cli. Update Purpose after archive.
## Requirements
### Requirement: help は classify と notify test の使い方も出す
`help` / `-h` / `--help` / 引数なしで出す使い方には、s04 の 2 行に加えて `classify --fixture <alias>` と `notify test` の 1 行説明を MUST 含める。`classify` の説明には、出力する列の順（優先 / 種別 / リポジトリ / 番号 / タイトル / 経過）を書く。
第 1 引数が `classify` なら `classify` サブコマンド、`notify` なら第 2 引数が `test` のときだけ `notify test` サブコマンドに振り分ける。`notify` の後に `test` 以外（無しを含む）が続く場合は、s04 の `fixture` と同じく `unknown command: notify` と使い方を標準エラーに書き、終了コード 1 で終わる。振り分けは s04 の `run(args []string, stdout, stderr io.Writer) int` にケースを足して行い、サブコマンドの失敗は s04 の Requirement「サブコマンドの失敗は標準エラーに出て終了コード 1 になる」に従う。

#### Scenario: help に classify と notify test が載る
- **WHEN** 引数 `help` で実行する
- **THEN** 標準出力に `classify --fixture` と `notify test` を含む使い方が出て、終了コードは 0 である

#### Scenario: notify の後にサブコマンドが無い
- **WHEN** 引数 `notify` だけで実行する
- **THEN** 標準エラーに `unknown command: notify` と使い方が出て、終了コードは 1 である

#### Scenario: notify の後が test 以外
- **WHEN** 引数 `notify foo` で実行する
- **THEN** 標準エラーに `unknown command: notify` と使い方が出て、終了コードは 1 である

#### Scenario: classify の必須フラグが無い
- **WHEN** 引数 `classify`（`--fixture` 無し）で実行する
- **THEN** 標準エラーに `--fixture` を含むエラーが出て、終了コードは 1 である

#### Scenario: classify の未知のフラグ
- **WHEN** 引数 `classify --fixture example --live` で実行する
- **THEN** 標準エラーに `live` を含むエラーが出て、終了コードは 1 である

### Requirement: classify は fixture を分類してキューを 4 タブ別にプレーンテキストで出す
`loop-cli-dev classify --fixture <alias> [--mode sdd|label]` は、カレントディレクトリからの相対パス `internal/gh/testdata/fixtures/<alias>` を s03 の `gh.NewFake` で読み、全 open issue / open PR を s05 の `classify.Issue` / `classify.PR` で分類し、結果を標準出力に MUST 書く。live の `gh` は実行しない。
- `--mode` は分類に使う運用方式で、既定は `sdd`。`sdd` / `label` 以外の値は、その値を含むエラーを返し、出力を書かない。`classify.Issue` / `classify.PR` の呼び出しにこの値を渡す（fixture は 1 リポジトリ 1 方式で採る）
- `classify.PR` の `now` には、経過の列の計算に使うのと同じ現在時刻を渡す（画面の分類と同じ見え方にする。採取から 3 時間を超えた fixture の PR は、進行中の規則 2 / 3 / 7 の時間切れとして分類される）
- `internal/gh/testdata/fixtures` が存在しなければ、s04 `fixture capture` と同じくリポジトリのルートで実行するよう促すエラーを返す。`<alias>` ディレクトリが無ければ、そのパスを含むエラーを返す（どちらも `Fake` を呼ぶ前に確認する）
- 入力の組み立ては s05 `human-turn-classify`「fixture と期待値表で分類器をテストする」と同じ: 各 issue は `model.IssueFromSearch` に `ViewIssue` の `Comments` を `model.CommentFrom` で入れ、各 PR は `model.PRFromSearch` に `ViewPR` の `Comments`、`ViewPRMergeState`、`ReviewThreads` を入れる。D-001 の遅延取得による絞り込みはしない（fixture は全詳細を持つ）。`Fake` の読み取りが 1 つでも失敗したら、そのエラーを返し、出力を書かない
- Issue と PR の紐づけ（`classify.Card`）は行わない。行は issue 1 件または PR 1 件である
- 出力は mvp.md「画面構成（今やるキュー）」のタブ順に 4 つの節を並べる。各節は見出し行 `[1]今やる <件数>` / `[2]バックログ <件数>` / `[3]進行中 <件数>` / `[4]異常 <件数>`（`<件数>` はその節の行数）の後に、その節の行を 1 件 1 行で続ける。件数 0 の節は見出し行だけを出す。節の間に空行を入れない
- 行の列は順に 優先 / 種別 / リポジトリ / 番号 / タイトル / 経過 で、タブ文字 1 つで区切る。優先は `Result.Priority` の 10 進表記、種別は `Situation.Kind()`、リポジトリは `Issue.Repo` / `PR.Repo`、番号は issue が `#<n>`、PR が `PR<n>`（mvp.md の画面例と同じ）、タイトルは `Title` そのまま、経過は現在時刻と `UpdatedAt` の差を `<m>m`（1 時間未満）/ `<h>h`（24 時間未満）/ `<d>d`（24 時間以上）で書き、`UpdatedAt` が現在時刻より後なら `0m` と書く
- 節内の並びは 第 1 キー `Result.Priority` 昇順、第 2 キー `Repo` 昇順、第 3 キー issue → PR、第 4 キー番号昇順。タブ内の並び順の第 2 キー以降は s08 が画面で決めるので、この CLI の並びを画面の仕様にしない

#### Scenario: example を分類する
- **WHEN** リポジトリのルートで `classify --fixture example` を実行する（s03 の `example` は issue 108 / issue 140 / PR 131 を持ち、s05 の期待値は順に `in-progress` / `E` / `A`）
- **THEN** 標準出力は 7 行で、順に `[1]今やる 1`、`PR131` と `質問` と `org/app` を含む行、`[2]バックログ 1`、`#140` と `todo 候補` を含む行、`[3]進行中 1`、`#108` と `進行中` を含む行、`[4]異常 0` であり、終了コードは 0 である

#### Scenario: label 方式の fixture を分類する
- **WHEN** リポジトリのルートで issue-label-driven の `<alias>` に対して `classify --fixture <alias> --mode label` を実行する
- **THEN** `In Progress` の issue は `[3]進行中` の節、ラベルの無い issue は `[2]バックログ` の節、`To Do` と `In Progress` が同時に付いた issue は `[4]異常` の節、`Closes #n` を持ち checks 緑で mergeable な PR は `[1]今やる` の節に `merge` の種別で出て、終了コードは 0 である

#### Scenario: mode の値が不正
- **WHEN** `classify --fixture example --mode kanban` を実行する
- **THEN** 標準エラーに `kanban` を含むエラーが出て、終了コードは 1 で、標準出力は空である

#### Scenario: リポジトリのルート以外で実行する
- **WHEN** `internal/gh/testdata/fixtures` が無いディレクトリで `classify --fixture example` を実行する
- **THEN** 標準エラーに `internal/gh/testdata/fixtures` を含むエラーが出て、終了コードは 1 で、標準出力は空である

#### Scenario: fixture が無い
- **WHEN** `classify --fixture nonexistent` を実行する
- **THEN** 標準エラーに `internal/gh/testdata/fixtures/nonexistent` を含むエラーが出て、終了コードは 1 で、標準出力は空である

#### Scenario: 経過の表記
- **WHEN** `elapsed(now, t)` に `t` が `now` の 12 分前・3 時間前・2 日前・未来（`now` より後）の値を渡す
- **THEN** 戻り値はそれぞれ `12m` / `3h` / `2d` / `0m` である

### Requirement: notify test はデスクトップ通知を 1 件出す
`loop-cli-dev notify test` は、タイトル `loop-cli`、本文 `テスト通知` のデスクトップ通知を 1 件 MUST 出し、成功したら標準出力に `通知を送りました` と書いて終了コード 0 で終わる。通知の発行に失敗したら、そのエラーを返し、終了コード 1 で終わる（s04 の失敗時の Requirement に従う）。通知ライブラリは `github.com/gen2brain/beeep` を使い、この change で依存に加える。s13 の自動更新通知はこの依存を再利用する。設定ファイル（`notify:` 項目）は読まない。

#### Scenario: 通知が出る
- **WHEN** デスクトップ通知が使える環境で `notify test` を実行する
- **THEN** タイトル `loop-cli`、本文 `テスト通知` の通知が 1 件表示され、標準出力に `通知を送りました` が出て、終了コードは 0 である

#### Scenario: 通知の発行に失敗する
- **WHEN** 通知ライブラリがエラーを返す環境で `notify test` を実行する
- **THEN** 標準エラーにそのエラーが出て、終了コードは 1 である

### Requirement: サブコマンドの失敗は標準エラーに出て終了コード 1 になる
サブコマンドがエラーを返した場合、CLI はエラー文字列を標準エラーに 1 行書き、終了コード 1 で MUST 終わる。panic しない。フラグの解析エラー（未知のフラグ・必須フラグの欠落）も同じ扱いにする。成功時は終了コード 0 である。

#### Scenario: 必須フラグの欠落
- **WHEN** 引数 `fixture capture --repo org/app`（`--alias` 無し）で実行する
- **THEN** 標準エラーに `--alias` を含むエラーが出て、終了コードは 1 である

#### Scenario: 未知のフラグ
- **WHEN** 引数 `fixture capture --repo org/app --alias app --out x` で実行する
- **THEN** 標準エラーに `out` を含むエラーが出て、終了コードは 1 である

### Requirement: loop-cli-dev はサブコマンドを振り分け help で使い方を出す
`go build ./cmd/loop-cli-dev` は `loop-cli-dev` バイナリを MUST 生成する。`cmd/loop-cli-dev` は動作確認用の CLI（implementation-tasks.md §2）で、標準ライブラリの `flag` と `os.Args` だけで書き、CLI フレームワークを依存に加えない。
第 1 引数をサブコマンド名として振り分ける。`help` / `-h` / `--help`、または引数なしのときは使い方を標準出力に書き、終了コード 0 で終わる。使い方には `help` と `fixture capture --repo owner/name --alias <alias>` の 1 行説明を含める。
未知のサブコマンド、または `fixture` の後に `capture` 以外（無しを含む）が続く場合は、`unknown command: <args[0]>`（第 1 引数のみ。`fixture foo` でも `fixture` 単独でも `unknown command: fixture`）と使い方を標準エラーに書き、終了コード 1 で終わる。
振り分けは `run(args []string, stdout, stderr io.Writer) int` のような関数で行い、`main` はその返り値で `os.Exit` する（テストがプロセスを起動せずに検証するため）。この change で定義するサブコマンドは `help` と `fixture capture` だけである。`classify` / `notify test` は s06 が ADDED で足す。

#### Scenario: help が使い方を出す
- **WHEN** 引数 `help` で実行する
- **THEN** 標準出力に `fixture capture` と `--repo` と `--alias` を含む使い方が出て、終了コードは 0 である

#### Scenario: 引数なしも使い方を出す
- **WHEN** 引数なしで実行する
- **THEN** 標準出力に使い方が出て、終了コードは 0 である

#### Scenario: 未知のサブコマンド
- **WHEN** 引数 `frobnicate` で実行する
- **THEN** 標準エラーに `unknown command: frobnicate` と使い方が出て、終了コードは 1 である

#### Scenario: fixture の後にサブコマンドが無い
- **WHEN** 引数 `fixture` だけで実行する
- **THEN** 標準エラーに `unknown command: fixture` と使い方が出て、終了コードは 1 である

#### Scenario: fixture の後が capture 以外
- **WHEN** 引数 `fixture foo` で実行する
- **THEN** 標準エラーに `unknown command: fixture` と使い方が出て、終了コードは 1 である

