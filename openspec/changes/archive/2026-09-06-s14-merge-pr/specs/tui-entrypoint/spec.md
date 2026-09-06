## MODIFIED Requirements

### Requirement: loop-cli バイナリが起動して 1 フレーム描画する
`go build ./cmd/loop-cli` は `loop-cli` バイナリを MUST 生成し、端末で起動すると Bubble Tea のプログラムとして 1 フレームを MUST 描画する。
`main()` は `run(args []string, stdout, stderr io.Writer) int` を呼び、その返り値で `os.Exit` する（テストがプロセスを起動せずに検証するため。`cmd/loop-cli-dev` と同じ形）。`run` は `args` が空のときだけ TUI を起動し、空でなければ Requirement「引数はサブコマンドに振り分ける」に従う。
TUI は次の順で起動する。
1. `config.DefaultPath()` で設定ファイルのパスを決め、そのパスにファイルが無く標準入力が端末なら s08a `onboarding` のフォームで設定ファイルを作ってから（「設定ファイルが無いときだけ onboarding に入る」）、`config.Load` で `Config` を読む（s02）
2. `gh.NewClient()` を作り、`Check(ctx)` で `gh` の存在と認証を確認する（s03。`ctx` は `context.Background()`）
3. `snapshot.DefaultPath()` でスナップショットのパスを決め、`snapshot.Load(path)` を 1 回呼ぶ（s13 `snapshot-cache`）。読めたら `ui.Options.Snapshot` に渡し、エラーなら渡さない（消さない）。`DefaultPath()` のエラーは起動失敗にせず、読み込みも保存もしない
4. `Config.Repos[].Name` を並べた `repos` と `client` で `fetch.Fetch`（s07）を閉じた `ui.Fetcher` を、`snapshot.Save` で包んだもの（s13 `snapshot-cache`「取得が成功するたびにスナップショットを保存する」）と、同じ `client`、`ui.ExternalEditor(Config.Editor)`（s10 `answer-question`。`Config.Editor` は s02 が `$EDITOR` を展開済みで、空でもここでは失敗させない）、`ui.Options{Snapshot, RefreshInterval: Config.RefreshIntervalSec 秒, Notify, CheckUpdate, MergeMethods}`（`MergeMethods` は `Config.Repos[]` の `Name` から `string(MergeMethod)` を引く対応表で、s14 `merge-pr`「merge 方式はリポジトリ名から引く」が使う。`internal/ui` が `internal/config` を import しないために `cmd/loop-cli` が作る）（`CheckUpdate` は s23 `self-update`「TUI は起動時に 1 度だけ更新を調べ、あればヘッダに出す」が定める確認の関数。設定では切り替えない）（`Notify` は `Config.Notify` が true なら `beeep.Notify(title, body, "")` を呼ぶ関数、false なら `nil`。s13 `desktop-notify`）を `ui.New` に渡し、Bubble Tea のプログラムとして実行する
最初のフレームは s08 `queue-screen` のキュー画面であり、スナップショットが無ければ `Init` が返す取得コマンドが完了するまでは `Cards` が空の表と `取得中` のスピナーを描く。スナップショットがあれば、その `Cards` と保存時刻を表示したまま `取得中` のスピナーを描く（D-002「起動直後は stale 表示 → 背景で再取得」）。取得の完了で表が埋まる。hello world の画面（s01）は無くなる。
描画内容にはアプリ名 `loop-cli` を含める。`q` で終了できることを示すフッタのヒントは、s08 `queue-screen` のフッタの規則（ヒントとステータスの両方が端末幅に収まらなければステータスを優先する）に従って出る。s14 `merge-pr` がキュー画面のヒントに `m merge` を足して 80 列になったため、既定幅 80 の端末では初回取得中のスピナーに押されてヒントが出ず、取得の完了後、または幅 89 列以上の端末で `q 終了` が見える（s14 design.md。キーを隠すよりキーを出すことを採った結果の逸脱で、`q` は初回取得中でも動く）。

#### Scenario: 初期フレームにアプリ名が出る
- **WHEN** `ui.New` で作った Model（既定幅 80、初回取得中）の初期状態から描画文字列を得る
- **THEN** 文字列に `loop-cli` と `取得中` が含まれる

#### Scenario: 幅 90 の端末なら初期フレームに終了案内も出る
- **WHEN** `ui.New` で作った Model に幅 90・高さ 24 のサイズメッセージを与えて描画文字列を得る
- **THEN** 文字列に `loop-cli` と `q 終了` と `取得中` が含まれる

#### Scenario: バイナリが生成される
- **WHEN** `go build -o bin/loop-cli ./cmd/loop-cli` を実行する
- **THEN** 終了コード 0 で `bin/loop-cli` が生成される

#### Scenario: 起動直後に取得が始まる
- **WHEN** 設定ファイルがあり `gh auth status` が通り、`$HOME/.cache/loop-cli/snapshot.json` が無い端末で `loop-cli` を起動する
- **THEN** 最初のフレームはヘッダ `[1]今やる 0 …` と `↻ --:--`、フッタの `取得中` を含むキュー画面で、`fetch.Fetch` の完了後に設定リポジトリの Card がタブに並び、`$HOME/.cache/loop-cli/snapshot.json` が作られる

#### Scenario: スナップショットがあれば stale 表示から始まる
- **WHEN** 前回の起動で作られた `$HOME/.cache/loop-cli/snapshot.json` がある端末で `loop-cli` を起動する
- **THEN** 最初のフレームは前回の Card が並んだ表とヘッダの `↻ <保存時刻>`、フッタの `取得中` を含むキュー画面で、`fetch.Fetch` の完了後にヘッダの時刻が現在時刻に更新される

#### Scenario: editor が空でも起動する
- **WHEN** 環境変数 `EDITOR` が未設定で `editor` を省略した設定ファイルがあり、`gh auth status` が通る端末で `loop-cli` を起動する
- **THEN** キュー画面が描画される（`a` を押したときに初めて `editor が設定されていません` のエラーがフッタに出る）

#### Scenario: 設定ファイルが無い端末ではフォームの後にキュー画面が出る
- **WHEN** `$HOME/.config/loop-cli/config.yml` が無く、`gh auth status` が通る端末で `loop-cli` を起動し、フォームに `org/app` を入力して完了する
- **THEN** `$HOME/.config/loop-cli/config.yml` が `repos:\n  - org/app\nrefresh_interval_sec: 120\nmerge_method: squash\neditor: $EDITOR\nnotify: true\n` の内容で作られ、続けてキュー画面が描画される
