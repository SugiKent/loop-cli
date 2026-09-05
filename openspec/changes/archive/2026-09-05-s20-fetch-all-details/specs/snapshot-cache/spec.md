## MODIFIED Requirements

### Requirement: スナップショットは Card 群と保存時刻を JSON で往復する
`internal/snapshot` は型 `Snapshot{Cards []model.Card; At time.Time}` と、`Save(path string, s Snapshot) error` / `Load(path string) (Snapshot, error)` を MUST 公開する。
- `Save` はディレクトリが無ければ `os.MkdirAll(filepath.Dir(path), 0o700)` で作り、`encoding/json` で `Snapshot` を書き、`os.WriteFile(path, data, 0o600)` で保存する。中身は GitHub から作り直せる派生データだけ（D-002）で、認証情報を含まない
- `Load` は `path` を読んで `Snapshot` にデコードして返す。ファイルが無い・読めない・JSON として壊れている（デコードのエラー）のいずれもエラーを返し、区別しない。`Load` はファイルを消さない
- `model.Card` は `Issue *model.Issue` / `PRs []model.PR` / `Result model.Result` からなり、その中の型（`model.Comment` / `gh.PRMergeState` / `gh.ReviewThread` / `gh.ReviewComment` / `time.Time` / `string` / `bool` / `int`）は `encoding/json` で往復できる。関数・チャネル・interface 値を含まない（s05 / s03 の型定義で確認済み）。`nil` のスライス（s09 が `取得失敗` と表示する `Comments` / `MergeState` / `ReviewThreads`）は `null` として書かれ `nil` に戻り、長さ 0 のスライス（`なし`）は `[]` として書かれ長さ 0 の非 `nil` に戻る。往復後も `Card.Issue.Result == Card.Result`（s08 `subjectOf` の主体判定）は保たれる

#### Scenario: example の Card が往復する
- **WHEN** s07 の `fetch.Fetch` に `gh.NewFake("../gh/testdata/fixtures/example")` を渡して得た `Result.Cards` と `At` `2026-09-05T03:04:00Z`（UTC。JSON から戻した `time.Time` の場所情報が一致するよう UTC で与える）の `Snapshot` を、一時ディレクトリの下の `sugi-loop/snapshot.json`（ディレクトリは未作成）に `Save` し、同じパスを `Load` する
- **THEN** `Save` はエラーを返さずファイルを作り、`Load` の結果は `reflect.DeepEqual` で元の `Snapshot` と等しい。issue 140 の Card の `Issue.Comments` は長さ 0 の非 `nil` のまま（s20 で全 issue のコメントを取るため）、issue 108 の Card の `PRs[0].ReviewThreads` は `Save` 前と同じ `nil` / 非 `nil` である

#### Scenario: nil と長さ 0 のスライスが区別されたまま戻る
- **WHEN** `Comments` が `nil` の Issue の Card と、`Comments` が長さ 0 の非 `nil` スライスの Issue の Card の 2 枚を `Save` して `Load` する
- **THEN** 1 枚目の `Issue.Comments` は `nil` であり、2 枚目の `Issue.Comments` は長さ 0 の非 `nil` スライスである

#### Scenario: ファイルが無ければエラー
- **WHEN** 存在しないパスを `Load` する
- **THEN** エラーが返る

#### Scenario: 壊れた JSON はエラーで、ファイルは残る
- **WHEN** 内容が `{"cards": [` のファイルを `Load` する
- **THEN** エラーが返り、ファイルは元の内容のまま残る
