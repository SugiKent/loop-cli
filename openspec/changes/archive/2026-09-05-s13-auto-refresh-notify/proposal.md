## Why

s12-open-refresh-help までの `sugi-loop` は、起動のたびに空の画面から取得を待ち、その後は `R` を押さない限りキューが古いまま止まる。人の出番が新しく増えても画面を見ていなければ気付けない。D-002（スナップショットキャッシュで起動直後は stale 表示、自動更新は既定 120 秒）と mvp.md の設定 `refresh_interval_sec` / `notify`（人の出番が新しく増えたらデスクトップ通知）はどれも未実装である。この change は docs/mvp/implementation-tasks.md §3 (P1)「自動更新（既定 120 秒）+ スナップショットキャッシュ + デスクトップ通知（更新前後で「今やる」を差分比較し、増えたカードを 1 件 1 通知。beeep）」を実装する。

## What Changes

- スナップショットキャッシュ: 新しい package `internal/snapshot` を足す。取得が成功するたびに `[]model.Card` と保存時刻を `~/.cache/sugi-loop/snapshot.json` に JSON で書き、起動時に読めれば `ui.New` に渡して stale 表示（表は前回の Card、ヘッダの `↻ HH:MM` は保存時刻）から始め、背景で通常どおり初回取得を行う。読めない・壊れているときは無視して従来どおり空の画面から始める（消さない）。保存は `cmd/sugi-loop` が `Fetcher` を包んで行い、`internal/ui` はファイルを触らない
- 自動更新: `refresh_interval_sec` ごとに s12 の `startFetch()` で取得を開始する。取得中（`fetching`）と書き込み中（s10 / s11 の投稿中フラグ）の tick は取得を開始せず次の tick を待つ。tick はどの画面でも届き、詳細画面を開いている間も `Cards` は差し替えるが、開いている詳細の対象は s09 の規則どおり閉じるまで旧 Card を保持する
- デスクトップ通知: 設定 `notify` が true のとき、取得成功後に前回の「今やる」タブのカード集合（キー: リポジトリ + 番号 + 主体が Issue か PR か）と今回を比較し、増えたカード 1 件につき beeep で 1 通知（タイトル `sugi-loop`、本文は 1 行要約 + `<Repo> #<n>` / `<Repo> PR#<n>`）を出す。前回が無い（スナップショット無しの初回取得）ときは通知しない。スナップショットがあればそれと比較して通知する。通知の失敗は無視する。beeep の呼び出しは s06 `notify test` と同じ `Notify(title, msg, "")`（icon は空文字列）
- `ui.New` の引数を拡張する（**BREAKING**: `internal/ui` 内部の署名変更。既存テストの `New` 呼び出しを直す）: スナップショットの初期 Card と保存時刻、自動更新の間隔、通知関数を受け取る
- `cmd/sugi-loop/main.go` の起動手順を拡張する: スナップショットの読み込み、`Fetcher` の保存の包み、`refresh_interval_sec` と `notify` の `ui.New` への配線
- 新しいキー・フッタのヒント・ヘルプ行は足さない。`internal/gh` / `internal/fetch` / `internal/classify` / `internal/model` / `internal/action` は変えない

## Capabilities

### New Capabilities
- `snapshot-cache`: `internal/snapshot` がスナップショットのパスを決めて JSON を往復し、`cmd/sugi-loop` が取得成功時に保存し、`internal/ui` が起動時に stale 表示を出すことを定める
- `auto-refresh`: `internal/ui` の自動更新。間隔・tick の処理・スキップする条件・詳細画面を開いている間の扱い
- `desktop-notify`: `internal/ui` の通知。「今やる」タブの差分を求める純粋関数、通知関数の呼び方、通知しない条件、失敗の扱い

### Modified Capabilities
- `queue-screen`: Requirement「Model は Card をタブ別に並べ、選択行を 1 つ持つ」（s10 が MODIFIED した最新版。`openspec/specs/queue-screen/spec.md`）の `New` の引数を拡張する。Requirement「取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する」（s08。同）の「初回取得前は `Cards` が空」をスナップショット無しのときに限定し、末尾の「`R` は s12、自動更新は s13、stale 表示は s13」の申し送りを実際の参照に直す
- `tui-entrypoint`: Requirement「sugi-loop バイナリが起動して 1 フレーム描画する」（s08a が MODIFIED した最新版。`openspec/changes/s08a-onboarding/specs/tui-entrypoint/spec.md`）の起動手順 3 にスナップショットの読み込み・保存の包み・間隔・通知関数を足し、Scenario「起動直後に取得が始まる」をスナップショット無しに限定して stale 起動の Scenario を 1 件足す。「起動失敗」は変えない（スナップショットの読み書きの失敗は起動失敗にしない）

## Impact

- この change が新しく作るファイル: `internal/snapshot/snapshot.go`（`Snapshot` / `DefaultPath` / `Load` / `Save`）と `internal/snapshot/snapshot_test.go`、`internal/ui/refresh.go`（自動更新の tick）と `internal/ui/refresh_test.go`、`internal/ui/notify.go`（差分の純粋関数・`Notifier`・通知コマンド）と `internal/ui/notify_test.go`
- 変更: `internal/ui/model.go`（`New` の引数、`Options`、`Init` の tick、`fetchedMsg` 成功時の通知）、`internal/ui/model_test.go` / `view_test.go` / `testdata_test.go` ほか `New` を呼ぶテスト、`cmd/sugi-loop/main.go`（`run()` の起動手順）、`cmd/sugi-loop/main_test.go`（s08a が作る。`Fetcher` の包みの検証を足す）
- 依存: `github.com/gen2brain/beeep`（s06 が追加済み。`cmd/sugi-loop` から呼ぶ）。新しい外部依存は無い
- 前提: s12-open-refresh-help が実装済みであること（非公開 `startFetch()`。取得開始の経路をこの change が再利用する）。s08a-onboarding が実装済みであること（`run()` の `ensureConfig` / `checkError`、`cmd/sugi-loop/main_test.go`）。archive の順は s08a → s12 → s13
- 後続 change への影響: s18 の 1 件再取得は `Cards` を差し替えるので、スナップショットの保存と通知の差分をどう扱うかを s18 が決める。s19 の通知クリックで該当カードを開くはこの change の通知本文の形式を前提にする
