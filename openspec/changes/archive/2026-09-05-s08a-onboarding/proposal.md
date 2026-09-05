## Why

s08-queue-screen までの `sugi-loop` は `~/.config/sugi-loop/config.yml` が無いと `config.Load` のエラーで終了し、`gh` が無い・未認証のときも s03 のエラー文字列を 1 行出すだけで、利用者が次に何をすればよいかを画面から読めない。設定したリポジトリに `stage:*` ラベルが無ければキューは空のまま何の説明も出ない。最初の起動は必ず設定ファイルが無い状態から始まるので、mvp.md「初回起動（onboarding）」（2026-09-05 追記）が定める「設定ファイルが無ければフォームで作る / gh 未認証の案内 / 空キューのヒント」をこの change で実装する。この change は docs/mvp/implementation-tasks.md §3 (P1)「初回起動 onboarding」の項目を実装する。

## What Changes

- 起動時の判定順を「設定ファイルの存在確認 → 無ければ onboarding フォーム → 書き出し → `config.Load` → `gh.Client.Check` → TUI」にする。存在確認で「無い」と分かったときだけフォームに入り、壊れた設定（YAML エラー・検証エラー）は従来どおり `config.Load` のエラーで終了する（上書きしない）
- 新しい package `internal/onboarding` を足す。huh のフォームで `repos`（改行区切りの複数行入力。各行を `owner/name` 規則で検証）、`merge_method`（squash / merge / rebase の select、既定 squash）、`notify`（confirm、既定 true）、`editor`（input、既定 `$EDITOR`）を聞き、mvp.md の設定例と同じ形式の YAML（`refresh_interval_sec: 120` を含む）を `~/.config/sugi-loop/config.yml` に書く。ディレクトリが無ければ作る。書いた後は通常の `config.Load` で読み直し、自前で `Config` を組まない
- `internal/config` に `owner/name` 規則の検証を公開する関数を足し、`Load` の検証と onboarding のフォームが同じ関数を使う（規則を 2 か所に書かない）
- フォームの中止（huh の既定の中止操作 Ctrl+C）は設定ファイルを書かず終了コード 1。標準入力が端末でない場合はフォームを出さず、「設定ファイルが無い」と期待するパスを標準エラーに出して終了コード 1
- `gh.Client.Check` の失敗を、原因と次の一手の 2 行にして標準エラーに出す。`gh` が無い → インストール先 `https://cli.github.com/`、`gh auth status` の失敗 → `gh auth login`。終了コード 1
- キュー画面で取得が成功して Card が 0 件（全タブ空）のとき、表の領域の中央に mvp.md のヒント「`stage:*` ラベルの無いリポジトリは何も出ません。issue-driven-sdd の `routines-setup` を回したリポジトリを設定してください」を出す。取得中（初回スピナー）と取得失敗時には出さない
- `charm.land/huh/v2` の依存を加える（s08 design の「`huh` は s15 まで入らない」を前倒し）

## Capabilities

### New Capabilities
- `onboarding`: 設定ファイルが無いときの判定順、huh フォームの項目と検証、YAML の書き出し、中止・非端末での振る舞い

### Modified Capabilities
- `tui-entrypoint`: Requirement「起動失敗は標準エラーに出て終了コード 1 になる」を MODIFIED。「設定ファイルが無い」を「端末なら onboarding、非端末なら exit 1」に分け、`Check` の失敗を原因 + 次の一手の 2 行にし、フォームの中止と書き出し失敗を起動失敗に加える。Requirement「sugi-loop バイナリが起動して 1 フレーム描画する」も MODIFIED し、変更は起動手順 1 を「存在確認 → 無ければフォーム → `config.Load`」に変えることと、Scenario「設定ファイルが無い端末ではフォームの後にキュー画面が出る」を 1 件足すことの 2 点だけ（写し元は s10 の現行版。archive の順は s10 → s08a）。「q で終了する」は触らない
- `queue-screen`: ADDED で Requirement「取得が成功して Card が 0 件のときは表の領域にヒントを出す」を足す。既存の Requirement は触らない（キー / ヘッダ・フッタは s09 / s10 / s11 が MODIFIED 中）
- `config-loading`: ADDED で Requirement「リポジトリ名の検証規則を公開する」を足す（`Load` の検証と onboarding が共有する関数）

## Impact

- この change は `internal/onboarding/`（フォーム・YAML 生成・書き出し）とそのテスト、`cmd/sugi-loop/main_test.go`（判定順と `Check` 失敗の文言を検証する）を新しく作る
- この change は `cmd/sugi-loop/main.go`（判定順・非端末判定・`Check` 失敗の文言）、`internal/config/config.go`（検証関数の公開）、`internal/ui/view.go`（空キューのヒント）、`internal/ui/view_test.go` を変更する
- 依存: `charm.land/huh/v2` を追加。標準入力が端末かの判定は `github.com/charmbracelet/x/term`（Bubble Tea 経由で go.sum に既にある）
- 先行 change との関係: s09 / s10 / s11 が `internal/ui` の `Model` / `New` の署名 / フッタ / キー処理を変更中。この change が `internal/ui` で触るのは表の領域の描画（0 件のとき）だけで、`New` の署名変更（s10: `New(fetcher, client, editor)`）が先に入っていればテストの呼び出しをそれに合わせる
