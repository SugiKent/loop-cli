## MODIFIED Requirements

### Requirement: fixture はリポジトリ別名ごとのディレクトリに決まった名前で置く
fixture は `internal/gh/testdata/fixtures/<repo-alias>/` 配下に MUST 置く。`<repo-alias>` は個人・組織情報を伏せた別名（例 `example`）で、ディレクトリ 1 つが稼働リポジトリ 1 件に対応する。ファイル名は次の規則に MUST 従い、内容は `Client` が発行するコマンドの標準出力をそのまま保存したものである（採取は s04 が担当する。`pr-<n>.json` だけは `ViewPR` と `ViewPRMergeState` の合成フィールド列で採る）。
- `search-issues.json`: `gh search issues …` の出力（配列）
- `search-prs.json`: `gh search prs …` の出力（配列）
- `labels.json`: `gh label list …` の出力（配列）
- `issue-<n>.json`: `gh issue view <n> …` の出力
- `pr-<n>.json`: `gh pr view <n> …` の出力（`ViewPR` と `ViewPRMergeState` の `--json` フィールドを合わせた 1 回の出力。`Fake` は両メソッドでこの 1 ファイルを読む）
- `pr-<n>-review-threads.json`: `ReviewThreads` の GraphQL 応答（`data` から始まる生の応答）
- `issue-<n>-cross-refs.json`: `CrossReferencedPRs` の GraphQL 応答（同上）
- `issue-<n>-timeline.json`: `LabelTimeline` の `--jq` 出力（JSON オブジェクトの連続）

`<n>` は issue / PR 番号をそのまま 10 進で書く（ゼロ埋めしない）。

#### Scenario: 番号付きファイル名が決まる
- **WHEN** 別名 `example` の PR 131 の `gh pr view` 出力を保存する
- **THEN** パスは `internal/gh/testdata/fixtures/example/pr-131.json` である

#### Scenario: ラベル一覧のファイル名は番号を持たない
- **WHEN** 別名 `example` の `gh label list` 出力を保存する
- **THEN** パスは `internal/gh/testdata/fixtures/example/labels.json` である

### Requirement: Fake は fixture を読んで GHClient と同じ型を返す
`internal/gh` は `GHClient` を満たす型 `Fake` と、fixture ディレクトリのパスを受け取るコンストラクタ `NewFake(dir string) *Fake` を MUST 提供する。読み取りメソッドは上記の命名規則でファイルを探し、`Client` と同じデコード関数で型に写して返す。`repo` 引数はファイルの探索に使わない（ディレクトリ 1 つが 1 リポジトリに対応するため）。対応するファイルが無ければ、そのパスを含むエラーを返す。空の結果を返して黙って通さない。`Fake` の `ViewPRMergeState` は再取得も待ちも行わず、`pr-<n>.json` の merge 関連フィールドをそのまま返す。`Fake` の `ListLabels` は `labels.json` を読み、`Calls` には記録しない（読み取りのうち記録するのは `ViewIssue` だけである）。

#### Scenario: search-issues.json を読む
- **WHEN** `internal/gh/testdata/fixtures/example/search-issues.json` に issue 2 件の配列がある状態で `NewFake("testdata/fixtures/example").SearchIssues(ctx, []string{"org/app"})` を呼ぶ
- **THEN** `SearchIssue` 2 件が返り、`Number` / `Title` / `Labels` / `Repository.NameWithOwner` がファイルの内容と一致する

#### Scenario: pr-<n>.json を読む
- **WHEN** `pr-131.json` がある状態で `ViewPR(ctx, "org/app", 131)` を呼ぶ
- **THEN** `PRDetail` が返り、`IsDraft` / `Comments` がファイルの内容と一致する

#### Scenario: pr-<n>.json から merge ガード用の状態を読む
- **WHEN** `pr-131.json` が `mergeable: "UNKNOWN"` の状態で `ViewPRMergeState(ctx, "org/app", 131)` を呼ぶ
- **THEN** `PRMergeState` が返り、`Mergeable` は `UNKNOWN` のまま、`StatusCheckRollup` がファイルの内容と一致する。再取得も待ちも行わない

#### Scenario: labels.json を読む
- **WHEN** `labels.json` に 3 件の配列がある状態で `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** `Label` 3 件がファイルの並びのまま返り、`Calls` は空である

#### Scenario: fixture が無い
- **WHEN** `issue-999.json` が無い状態で `ViewIssue(ctx, "org/app", 999)` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `issue-999.json` を含む

#### Scenario: ラベルの fixture が無い
- **WHEN** `labels.json` が無いディレクトリで `ListLabels(ctx, "org/app")` を呼ぶ
- **THEN** エラーが返り、エラー文字列に `labels.json` を含む

#### Scenario: Client と同じデコードを通る
- **WHEN** `pr-131-review-threads.json` に GraphQL の生の応答がある状態で `ReviewThreads(ctx, "org/app", 131)` を呼ぶ
- **THEN** 実行関数を差し替えて同じバイト列を返す `Client` の `ReviewThreads` の結果と `reflect.DeepEqual` で一致する
