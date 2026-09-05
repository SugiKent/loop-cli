# snapshot-cache Specification

## Purpose
TBD - created by archiving change s13-auto-refresh-notify. Update Purpose after archive.
## Requirements

### Requirement: スナップショットのパスは ~/.cache/sugi-loop/snapshot.json に決まる
`internal/snapshot` は `DefaultPath() (string, error)` を MUST 公開し、`$HOME/.cache/sugi-loop/snapshot.json`（D-002）を返す。`$HOME` は `os.UserHomeDir()` で決め、`os.UserCacheDir()` は使わない（macOS では `~/Library/Caches` になり D-002 のパスと一致しないため。s02 `config.DefaultPath` と同じ決め方）。`os.UserHomeDir()` のエラーはそのまま返す。

#### Scenario: 既定のパス
- **WHEN** 環境変数 `HOME` が `/tmp/h` の状態で `DefaultPath()` を呼ぶ
- **THEN** `/tmp/h/.cache/sugi-loop/snapshot.json` が返る

### Requirement: スナップショットは Card 群と保存時刻を JSON で往復する
`internal/snapshot` は型 `Snapshot{Cards []model.Card; At time.Time}` と、`Save(path string, s Snapshot) error` / `Load(path string) (Snapshot, error)` を MUST 公開する。
- `Save` はディレクトリが無ければ `os.MkdirAll(filepath.Dir(path), 0o700)` で作り、`encoding/json` で `Snapshot` を書き、`os.WriteFile(path, data, 0o600)` で保存する。中身は GitHub から作り直せる派生データだけ（D-002）で、認証情報を含まない
- `Load` は `path` を読んで `Snapshot` にデコードして返す。ファイルが無い・読めない・JSON として壊れている（デコードのエラー）のいずれもエラーを返し、区別しない。`Load` はファイルを消さない
- `model.Card` は `Issue *model.Issue` / `PRs []model.PR` / `Result model.Result` からなり、その中の型（`model.Comment` / `gh.PRMergeState` / `gh.ReviewThread` / `gh.ReviewComment` / `time.Time` / `string` / `bool` / `int`）は `encoding/json` で往復できる。関数・チャネル・interface 値を含まない（s05 / s03 の型定義で確認済み）。`nil` のスライス（s09 が `未取得` と表示する `Comments` / `MergeState` / `ReviewThreads`）は `null` として書かれ `nil` に戻り、長さ 0 のスライス（`なし`）は `[]` として書かれ長さ 0 の非 `nil` に戻る。往復後も `Card.Issue.Result == Card.Result`（s08 `subjectOf` の主体判定）は保たれる

#### Scenario: example の Card が往復する
- **WHEN** s07 の `fetch.Fetch` に `gh.NewFake("../gh/testdata/fixtures/example")` を渡して得た `Result.Cards` と `At` `2026-09-05T03:04:00Z`（UTC。JSON から戻した `time.Time` の場所情報が一致するよう UTC で与える）の `Snapshot` を、一時ディレクトリの下の `sugi-loop/snapshot.json`（ディレクトリは未作成）に `Save` し、同じパスを `Load` する
- **THEN** `Save` はエラーを返さずファイルを作り、`Load` の結果は `reflect.DeepEqual` で元の `Snapshot` と等しい。issue 140 の Card の `Issue.Comments` は `nil` のまま、issue 108 の Card の `PRs[0].ReviewThreads` は `Save` 前と同じ `nil` / 非 `nil` である

#### Scenario: nil と長さ 0 のスライスが区別されたまま戻る
- **WHEN** `Comments` が `nil` の Issue の Card と、`Comments` が長さ 0 の非 `nil` スライスの Issue の Card の 2 枚を `Save` して `Load` する
- **THEN** 1 枚目の `Issue.Comments` は `nil` であり、2 枚目の `Issue.Comments` は長さ 0 の非 `nil` スライスである

#### Scenario: ファイルが無ければエラー
- **WHEN** 存在しないパスを `Load` する
- **THEN** エラーが返る

#### Scenario: 壊れた JSON はエラーで、ファイルは残る
- **WHEN** 内容が `{"cards": [` のファイルを `Load` する
- **THEN** エラーが返り、ファイルは元の内容のまま残る

### Requirement: 取得が成功するたびにスナップショットを保存する
`cmd/sugi-loop` は `ui.New` に渡す `Fetcher` を、s07 の `fetch.Fetch` を呼んだ後に MUST 次を行う関数にする（保存は `internal/ui` の外で行い、`Model` はファイルを触らない）。
- `fetch.Fetch` がエラーを返さなかった（部分失敗 `Result.Errors` が 1 件以上でも含む。search は成功しており `Cards` は表示に使うもの）: `snapshot.Save(path, Snapshot{Cards: res.Cards, At: time.Now()})` を呼ぶ。`Save` のエラーは無視し、`Result` をそのまま返す（派生データの保存失敗で画面を止めない。design.md 未決事項の既定値）
- `fetch.Fetch` がエラーを返した: 保存せず、そのエラーを返す（前回のスナップショットは残る）
`path` は `snapshot.DefaultPath()` で起動時に 1 回決める。`DefaultPath()` のエラーは起動失敗にせず、スナップショットの読み込みと保存を行わない（design.md 未決事項の既定値）。この包みは `cmd/sugi-loop` の非公開関数 `savingFetcher(fetcher ui.Fetcher, path string) ui.Fetcher` とし、テストは固定の `Result` を返す `Fetcher` と一時ディレクトリのパスで検証する。

#### Scenario: 成功で保存される
- **WHEN** `example` の `Result` を返す `Fetcher` を一時ディレクトリの下の未作成のパスで包み、`context.Background()` で呼ぶ
- **THEN** 返り値はその `Result` とエラー nil で、パスに `Load` できるファイルがあり、その `Cards` は `Result.Cards` と `reflect.DeepEqual` で等しく、`At` は呼ぶ前後の時刻の間にある

#### Scenario: 失敗では保存されない
- **WHEN** error `search issues: gh search issues: exit 1: rate limited` を返す `Fetcher` を存在しないパスで包み、呼ぶ
- **THEN** そのエラーが返り、パスにファイルは作られない

#### Scenario: 保存の失敗は取得の結果を変えない
- **WHEN** `example` の `Result` を返す `Fetcher` を、書き込めないパス（既存のディレクトリと同じパス）で包み、呼ぶ
- **THEN** 返り値はその `Result` とエラー nil である

### Requirement: 起動時にスナップショットがあれば stale 表示から始める
`ui.New` は初期 Card と保存時刻（s13 `queue-screen`「Model は Card をタブ別に並べ、選択行を 1 つ持つ」の MODIFIED の `Options.Snapshot`）を受け取り、渡されたときは MUST 次の状態から始める（D-002「起動直後は stale 表示 → 背景で再取得」）。
- `Cards` はスナップショットの `Cards` で、4 タブの振り分けと並びは通常の取得完了と同じ
- 最終更新時刻はスナップショットの `At`。ヘッダの `↻ HH:MM` と各行の経過（s08 `Elapsed`）はこの時刻を基準にする
- `fetching` は true で、`Init` はスナップショットが無いときと同じく初回取得を開始する（フッタに `取得中` のスピナー。stale 表示中であることをスピナー以外で示さない。design.md 未決事項の既定値）
- 初回取得の完了で `Cards` と最終更新時刻は s08 の規則どおり差し替わる。失敗ならスナップショットの表示を維持し、エラーを赤で出す
- スナップショットの `Cards` が 0 件でも、初回取得の完了まで s08a の空キューのヒントは出ない（`fetching` が true のため。s08a の条件は変えない）
`cmd/sugi-loop` は起動時に `snapshot.Load(path)` を 1 回呼び、エラーなら `Options.Snapshot` を渡さない（壊れたファイルは消さず、次の取得成功で上書きされる。design.md 未決事項の既定値）。読み込みは `Check` の後、`ui.New` の前に行う。

#### Scenario: スナップショットの Card と時刻で始まる
- **WHEN** `example` の `Result.Cards` と `At` `2026-09-05T11:50:00+09:00` のスナップショットを渡して `New` した `Model` の `View` から ANSI エスケープを除いて読む
- **THEN** 今やるタブに PR 131 の行、ヘッダに `[1]今やる 1` と `↻ 11:50`、フッタに `取得中` が含まれ、`routines-setup` は含まれない

#### Scenario: 初回取得の完了で差し替わる
- **WHEN** 上の `Model` に、issue 108 の Card を含まず issue 140 の Card だけを持つ `Result` を完了時刻 `12:04` の取得完了として渡す
- **THEN** 今やるタブは 0 行、バックログに issue 140 の行があり、ヘッダは `↻ 12:04` で、`取得中` は無い

#### Scenario: 初回取得の失敗でもスナップショットが残る
- **WHEN** スナップショットを渡して `New` した `Model` に error `search issues: gh search issues: exit 1: rate limited` を取得失敗として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** 今やるタブに PR 131 の行が残り、ヘッダの `↻ 11:50` は変わらず、フッタに `rate limited` が含まれる

#### Scenario: スナップショット無しは従来どおり
- **WHEN** `Options` の `Snapshot` を渡さずに `New` した `Model` の `View` を読む
- **THEN** 表は 0 行、ヘッダに `↻ --:--`、フッタに `取得中` が含まれる
