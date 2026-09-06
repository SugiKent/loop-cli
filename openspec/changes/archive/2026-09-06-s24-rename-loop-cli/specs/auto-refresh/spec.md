## MODIFIED Requirements

### Requirement: refresh_interval_sec ごとに取得を開始し、取得中と書き込み中の tick は次まで待つ
`Model` は `New` で受け取った自動更新の間隔（`Options.RefreshInterval`。`cmd/loop-cli` は `Config.RefreshIntervalSec` 秒を渡す。既定 120 秒は s02 が埋める）で MUST 次のとおり動く（D-002「自動更新は既定 120 秒（設定可）」）。
- 間隔が 0 より大きいとき、`Init` は初回取得の開始（s12 `manual-refresh` の `startFetch()`）に加えて、間隔の経過後に `internal/ui` 内の tick メッセージを返すコマンドを返す
- tick メッセージが届いたら、取得中（`fetching` が true）または書き込み中（s10 / s11 の投稿中フラグが true）なら取得を開始せず、そうでなければ `startFetch()` で取得を開始する（`R` と同じ経路。`errText` / `partial` / 書き込みステータスが消える。s10 / s11 の「次の取得が始まったとき消える」のとおり）。どちらの場合も次の tick のコマンドを返し、tick は止まらない
- tick はどの画面（キュー / カード詳細 / PR 詳細 / 確認 / ヘルプ）でも同じに扱う。詳細画面を開いている間の取得完了は s09「詳細を開いている間の取得完了は対象の Card を差し替えない」のとおりで、`Cards` と最終更新時刻は更新し、詳細の対象・確認画面の下書きは閉じるまで旧 Card のまま保持する。キュー画面の選択行は s08 の丸め規則だけで先頭に戻さない
- `R`（s12）で手動取得しても tick の周期は変えない（次の tick は前の tick から間隔後。design.md 未決事項の既定値）
- 間隔が 0 のとき自動更新は無い（`Init` は tick のコマンドを返さない。`config.Load` は 1 以上を保証するので本番では起きず、テストが tick を待たずに `Model` を作るための値）
tick の間隔は `Model` が壁時計を読まずにコマンドの遅延で作る。`Model` は tick の時刻を保持しない。

#### Scenario: tick で取得が始まる
- **WHEN** 呼ばれた回数を数える `Fetcher` と間隔 1 秒で `New` し、`example` の `Result` を取得完了として渡した `Model`（`Fetcher` の呼び出し回数 0）に tick メッセージを `Update` で渡し、返ったコマンドを実行し、得たメッセージが複数のコマンドの束ならその各コマンドも実行する
- **THEN** tick の直後に `fetching` は true で `View` に `取得中` があり、実行して得たメッセージに `fetchedMsg` が含まれ、`Fetcher` は 1 回呼ばれる

#### Scenario: 取得中の tick は取得を開始しない
- **WHEN** `New` 直後（初回取得中）の `Model` に tick メッセージを渡す
- **THEN** コマンドが返り（次の tick）、`Fetcher` の呼び出し回数は 0 のままで、返ったコマンドを実行して得たメッセージに `fetchedMsg` は含まれない

#### Scenario: 書き込み中の tick は取得を開始しない
- **WHEN** `example` の `Result` を渡してバックログの issue 140 を選んだ `Model` に s11 の手順で `t` を与えた直後（切り替えのコマンドを実行する前）に tick メッセージを渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `fetching` は false のままで、フッタに `切り替え中` が残り `取得中` は無く、`Fetcher` の呼び出し回数は 0 である

#### Scenario: 詳細を開いている間の自動更新は対象を差し替えない
- **WHEN** `example` の `Result` で issue 108 のカード詳細を開いた `Model` に tick メッセージを渡し、次に issue 108 の Card を含まない `Result` を取得完了として渡す
- **THEN** 画面はカード詳細のままで、詳細の対象は issue 108 の Card のままである。`Esc` でキューに戻ると今やるタブの行は新しい `Result` のものである

#### Scenario: Init は取得と tick の両方を返す
- **WHEN** 間隔 1 秒で `New` した `Model` の `Init` を呼ぶ
- **THEN** コマンドが返る（`fetching` は true）。間隔 0 で `New` した `Model` の `Init` もコマンドを返し、`fetching` は true である（tick の有無は Scenario「tick で取得が始まる」の経路で検証し、`Init` の束の中身は数えない）

#### Scenario: 手動確認で 2 分ごとに更新される
- **WHEN** 稼働リポジトリ 1 件を設定し `refresh_interval_sec` を省略した状態で起動し、2 分以上待つ
- **THEN** フッタに `取得中` のスピナーが出た後、ヘッダの `↻ HH:MM` が更新される
