## 1. 設定に claude_config_dir を足す

- [x] 1.1 `internal/config` の `Repo` に `claude_config_dir` を足し、`UnmarshalYAML` でキーを受け、`Load` で環境変数と `~` を展開する。`config-loading` の ADDED Requirement の 4 Scenario（ホーム展開 / 環境変数 / 省略は空 / 存在しないパスでも成功）を `internal/config/config_test.go` のテストにして通す
- [x] 1.2 既存の設定のテストが通ることを確認する（未知キーのエラー、`mode` 廃止の専用の文言、`merge_method` の解決が変わっていない）

## 2. セッション ID の取り出し

- [x] 2.1 `internal/model` に、本文から最初の `https://claude.ai/code/session_<ID>` を取り出す関数を足す。URL が無い本文では見つからず、複数あるときは最初を採ることをテストで通す
- [x] 2.2 `internal/model` に、コメント列から最新の routine コメントの `session: <ID>` を取り出す関数を足す。`session-pane`「出すセッションは PR 本文と issue の routine コメントから決める」の 3 Scenario（PR 本文 / 最新のコメント / 人のコメントは採らない）に対応するテストを通す

## 3. claude を起動してログを読む

- [x] 3.1 `internal/claude` に、実行経路を差し替えられる `Client`（`internal/gh/client.go:27` と同じ形）を作る。`claude-client`「セッションのログを 1 回の claude 起動で取る」の 2 Scenario（引数とプロファイルが渡る / `ctx` 打ち切り）をテストで通す
- [x] 3.2 stream-json から `tool_result` の本文を取り出し `HTTP <status>` を判定するデコードを書く。`claude-client`「応答は tool_result の本文だけから読む」の 3 Scenario（200 / `assistant` の文章が混ざらない / 404）を fixture でテストして通す
- [x] 3.3 `claude` 不在・週の利用上限・未ログイン・起動の失敗を呼び出し側が区別できるエラーを書く。`claude-client`「claude を起動できない 3 つの場合を区別する」の 3 Scenario を fixture でテストして通す
- [x] 3.4 ログ本文を 時刻 / 種類 / ツール名 / 本文 の並びに読み、`result` の行から最終回答を取り出す関数を書く。`claude-client`「ログは時刻・種類・本文の行として読む」の 3 Scenario をテストで通す
- [x] 3.5 `internal/claude/testdata` に手書きの fixture を置く（`get_run_log` の HTTP 200・HTTP 404・週の利用上限・未ログイン・`assistant` の行が混ざった 200 の 5 件）。issue #18「出力の読み方」の形式を写し、セッション ID とリポジトリ名は伏せ字にする。3.1〜3.4 のテストがこの fixture を読んでいることを確認する

## 4. 詳細画面を 2 ペインにする

- [x] 4.1 詳細画面の描画を左右 2 ペインに割る（端末幅 120 以上で右ペイン 40 列 + 縦の区切り 1 列、左ペインは残り。120 未満は今までの全幅 1 ペイン）。フッタは 1 行で端末幅の全体を使う。`card-detail` の MODIFIED の 3 Scenario（幅 120 で右ペインと縦の区切り / 幅 119 で出さない / 幅 60 は 1 ペイン）と既存の 6 Scenario をテストで通す
- [x] 4.2 右ペインの行を作る（見出し / `状態:` / `最終更新:` / セッション ID / `直近の動き` と最新 12 行 / `最終回答:` / `取得:`）。`session-pane`「右ペインは紐づくセッションの稼働状況を出す」の 4 Scenario をテストで通す
- [x] 4.3 取得できないときの行を作る（`claude_config_dir が未設定です` / `セッション ID が見つかりません` / `claude が見つかりません` / `未ログインです` / `HTTP 404 プロファイルが違う可能性があります` / `制限中 <解除の時刻>` / `30 秒で打ち切りました` / エラーの文字列）。`session-pane`「取得できない理由は右ペインに出し、画面の他の部分は動き続ける」の 4 Scenario をテストで通す

## 5. 取得のタイミングと抑制

- [x] 5.1 セッション取得のコマンドとメッセージ、セッション ID ごとの取得結果の保持を `internal/ui` に足し、詳細画面に移ったときに結果が無ければ 1 回だけ取得する。`session-pane`「セッションの取得は詳細画面を開いたときと詳細の R だけで起きる」の 4 Scenario をテストで通す
- [x] 5.2 詳細画面の `R` を取り直しに割り当てる（`o` / `?` と同じく `Update` で `tea.Cmd` を返せる位置に置く）。`manual-refresh` の MODIFIED の 7 Scenario をテストで通す
- [x] 5.3 プロファイル単位の同時 1 本と、週の利用上限に当たったプロファイルのブロックを足す。`session-pane`「同じプロファイルの claude は同時 1 本に限り、制限中は起動しない」の 3 Scenario をテストで通す

## 6. URL 一覧とヘルプ

- [x] 6.1 `u` の URL 一覧の先頭に出典 `session` の 1 件を置く（キュー画面では置かない）。`url-picker` の MODIFIED の 2 Requirement の Scenario をテストで通す
- [x] 6.2 `internal/ui/help.go` の `helpKeys` の `R` と `u` の行を書き換える。`help-screen` の MODIFIED の 6 Scenario をテストで通す

## 7. ドキュメント

- [x] 7.1 README の設定ファイルの表に `claude_config_dir` を足し、設定例に 1 行足す。「再取得（`R`）」の節を詳細画面の振る舞いで書き直し、カード詳細 / PR 詳細の節に右ペインの説明を足す。前の版に戻すときは `claude_config_dir` の行を消す必要があることも書く
- [x] 7.2 `docs/mvp` を触っていないことを `git diff origin/main --stat` で確認する（CLAUDE.md「`docs/mvp` はこれ以降更新しない」。キーバインド表と設定ファイルの記述は README と `openspec/specs/` だけで保つ）

## 8. 通し確認

- [x] 8.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [x] 8.2 `openspec validate --strict` が緑であることを確認する
