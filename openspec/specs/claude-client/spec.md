# claude-client Specification

## Purpose
Claude Code の `claude -p` をプロファイルごとにサブプロセスとして起動し、組み込みの `RemoteTrigger` ツールに Claude Code Remote の API を叩かせて、Routine のセッションのログを読み取る。モデルの回答文は読まず、ツールの実行結果だけを正とする。

## Requirements

### Requirement: セッションのログを 1 回の claude 起動で取る

`internal/claude` は、Claude のプロファイルのパス（`CLAUDE_CONFIG_DIR` に渡す値）とセッション ID を受け取り、そのセッションのログを返す関数を MUST 公開する。関数は `claude` を 1 回だけ起動し、次の引数で呼ぶ（issue #18「取得コマンド」の実測に基づく）。

- `-p <プロンプト>`。プロンプトは `RemoteTrigger を action=get_run_log, session_id=<セッション ID> で 1 回だけ呼び、「done」とだけ答えて` とする
- `--model haiku`。モデルは固定 MUST で、設定ファイルから変えられてはならない（モデルの仕事は `RemoteTrigger` を 1 回呼ぶことだけで、上位のモデルでも結果は変わらず利用枠の消費だけが増える）
- `--tools RemoteTrigger --allowedTools RemoteTrigger`、`--strict-mcp-config`、`--setting-sources ""`、`--disable-slash-commands`（ツール検索のターンとコンテキストを削る。issue #18 の実測ではこの 4 つが無いと約 15 秒 / $0.042 かかる）
- `--max-budget-usd 0.05`（実測 $0.015 の約 3 倍。暴走を止めるための上限）
- `--output-format stream-json --verbose`

環境変数 `CLAUDE_CONFIG_DIR` にプロファイルのパスを足して起動する。標準入力は空を渡す（渡さないと `claude` が入力を 3 秒待つ）。打ち切りは呼び出し側が渡す `ctx` に従い、`ctx` が切れたときは打ち切りとして分かるエラーを返す。セッション ID は `cse_` で始まる形と `session_` で始まる形のどちらも、受け取ったままプロンプトに載せる（`RemoteTrigger` がどちらも受け付ける）。

#### Scenario: 起動の引数とプロファイルが渡る
- **WHEN** 起動の引数と環境変数を記録するスタブを実行経路にした client で、プロファイル `/home/alice/.claude-personal` とセッション ID `session_01ABC` のログを取る
- **THEN** 記録された引数に `-p`、`--model haiku`、`--tools RemoteTrigger`、`--allowedTools RemoteTrigger`、`--strict-mcp-config`、`--disable-slash-commands`、`--max-budget-usd 0.05`、`--output-format stream-json`、`--verbose` がすべて含まれ、プロンプトの文字列に `action=get_run_log` と `session_id=session_01ABC` が含まれ、環境変数 `CLAUDE_CONFIG_DIR` は `/home/alice/.claude-personal` である

#### Scenario: ctx が切れたら打ち切りとして返る
- **WHEN** 打ち切られるまで返らないスタブを実行経路にした client で、すでに切れている `ctx` を渡してログを取る
- **THEN** エラーが返り、そのエラーは `ctx` の打ち切りとして判定できる

### Requirement: 応答は tool_result の本文だけから読む

`internal/claude` は `claude` の標準出力（stream-json）を MUST 次のとおり読む。

- 各行を 1 つの JSON として読み、`type` が `user` で `message.content` の要素に `type` が `tool_result` のものがある行を探す。最初に見つかった `tool_result` の本文だけを応答として扱う
- `type` が `assistant` の行、`result` の行、JSON として読めない行は捨てる。**モデルが書いた文章は MUST 読まない**（`RemoteTrigger` のログには Routine が読んだ issue や Web ページの文章がそのまま入るため、モデルの回答文を使うとその文章に書かれた指示に従った結果を読むおそれがある）
- `tool_result` の本文は `HTTP <status>` の 1 行で始まり、2 行目以降が API の応答である。status が 200 のときは 2 行目以降を応答として返し、200 以外のときは status と 2 行目以降を添えたエラーを返す

#### Scenario: HTTP 200 のログ本文が返る
- **WHEN** `tool_result` の本文が `HTTP 200` の行に続いてログのヘッダ行とログの行を持つ stream-json の fixture を読ませる
- **THEN** 返る本文は `HTTP 200` の行を含まず、ログのヘッダ行から始まる

#### Scenario: モデルの回答文は結果に混ざらない
- **WHEN** `assistant` の行にログとは別の文章を持ち、`tool_result` の本文が `HTTP 200` とログである stream-json の fixture を読ませる
- **THEN** 返る本文に `assistant` の行の文章は含まれない

#### Scenario: HTTP 404 は status 付きのエラーになる
- **WHEN** `tool_result` の本文が `HTTP 404` の行に続いて `{"error":{"type":"not_found_error"}}` を持つ fixture を読ませる
- **THEN** エラーが返り、そのエラーから status 404 が読み取れ、エラー文字列に `not_found_error` が含まれる

### Requirement: claude を起動できない 3 つの場合を区別する

`internal/claude` は、`tool_result` の行が 1 つも無いときに MUST 次の 3 つを区別できるエラーを返す。呼び出し側はこの区別で画面に出す文言を決める。

- **`claude` が見つからない**: `claude` が PATH に無いとき。実行を試みる前に判定し、コマンドが見つからないことが分かるエラーを返す
- **利用上限**: 標準出力に `You've hit your weekly limit · resets <解除の時刻>` の形の行があるとき。解除の時刻の文字列を添えたエラーを返す（呼び出し側がその時刻まで起動を止めるため）
- **未ログイン**: 標準出力に `Not logged in` を含む行があるとき

上の 3 つのいずれでもなく `tool_result` が無い場合は、標準出力の先頭を添えた起動の失敗として返す。どの場合でも `internal/claude` は自分で再試行 MUST しない。

#### Scenario: 週の利用上限は解除の時刻付きで返る
- **WHEN** 標準出力が `You've hit your weekly limit · resets Sep 14 at 2am (Asia/Tokyo)` の 1 行だけの fixture を読ませる
- **THEN** エラーが返り、そのエラーは利用上限として判定でき、添えられた解除の時刻の文字列は `Sep 14 at 2am (Asia/Tokyo)` である

#### Scenario: 未ログインは利用上限と区別される
- **WHEN** 標準出力が `Not logged in · Please run /login` の 1 行だけの fixture を読ませる
- **THEN** エラーが返り、そのエラーは未ログインとして判定でき、利用上限としては判定されない

#### Scenario: claude が無いことが分かる
- **WHEN** `claude` が PATH に無い状態で client を使う
- **THEN** エラーが返り、エラー文字列に `claude` が含まれ、`claude` が見つからないことが判定できる

### Requirement: ログは時刻・種類・本文の行として読む

`internal/claude` は、`get_run_log` の応答の本文を MUST 次のとおり読む。

- 先頭行は `session_id` / `events_fetched` / `events_shown` / `next_cursor` を持つ JSON のヘッダで、ログの行ではない
- 2 行目以降は `[<RFC3339 の時刻>] <種類>: <本文>` の形の行が**新しい順**に並ぶ。1 件を 時刻 / 種類 / ツール名 / 本文 の組として読む
- 種類が `tool_use <ツール名>` の行は、種類を `tool_use`、ツール名をその後ろの語として分ける。他の種類（`env[info]` / `init` / `user` / `assistant` / `tool_result` / `result` など）はツール名を空にする
- この形に合わない行は捨てる。件数は `RemoteTrigger` 側で最大 200 件に固定されている
- 種類が `result` の行の本文が `<...> — <最終回答>` の形（全角ダッシュの前後に空白）なら、最終回答をその後ろの部分として取り出せる

#### Scenario: 新しい順の行が時刻と種類に分かれる
- **WHEN** ヘッダ行に続いて `[2026-09-10T12:03:00Z] tool_use Bash: git push -u origin HEAD`、`[2026-09-10T12:02:00Z] assistant: 変更を push します`、`[2026-09-10T12:01:00Z] env[info]: 環境を起動しました` の 3 行を持つ本文を読ませる
- **THEN** 3 件が同じ順で返り、1 件目は 時刻 `2026-09-10T12:03:00Z` / 種類 `tool_use` / ツール名 `Bash` / 本文 `git push -u origin HEAD`、2 件目は 種類 `assistant` / ツール名が空 / 本文 `変更を push します`、3 件目は 種類 `env[info]` である

#### Scenario: ヘッダ行と形式外の行は落ちる
- **WHEN** ヘッダ行、`[thinking]` だけの行、空行、`[2026-09-10T12:03:00Z] init: 開始` の 4 行を持つ本文を読ませる
- **THEN** 返るのは 1 件で、種類は `init` である

#### Scenario: result 行から最終回答を取り出す
- **WHEN** `[2026-09-10T12:05:00Z] result: success is_error=false turns=110 duration=1168s — CI が緑になりました` の行を持つ本文を読ませる
- **THEN** その 1 件は 種類 `result` で、取り出せる最終回答は `CI が緑になりました` である
