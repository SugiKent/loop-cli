## MODIFIED Requirements

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
