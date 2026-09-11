issue: #18

## Why

Routine が Claude Code on the web で issue / PR を進めている間、その進み具合は GitHub のラベルとコメントにしか現れない。セッションが動いているのか、止まっているのか、人の入力を待っているのかは、ブラウザで `https://claude.ai/code/session_...` を開くまで分からない。loop-cli は「人の出番だけ」を並べる TUI なのに、出番かどうかを決める材料の半分が画面の外にある。

セッションの情報は `claude -p` を起動して Claude Code 組み込みの `RemoteTrigger` ツールに API を叩かせる方式でしか取れず（issue #18「既存の CLI コマンドでは取れない」）、1 回の取得でモデルが 2 ターン動く（実測 約 4.7 秒 / $0.015、haiku、2026-09-10）。消費するのは Routine を回しているプロファイルと同じ利用枠なので、取得の回数そのものが設計の制約になる。

## What Changes

- 設定ファイルの `repos` の要素に `claude_config_dir` を足す。値は `CLAUDE_CONFIG_DIR` に渡す Claude のプロファイルのパスで、`~` と環境変数を loop-cli 側で展開する。`claude_config_dir` の無いリポジトリではセッションを取得しない
- `claude -p` をサブプロセスとして起動し、`RemoteTrigger` の `tool_result` から API の応答を読み取る client を足す（`internal/gh` と同じ形）。モデルは `--model haiku` に固定し、設定で変えられるようにしない
- カード詳細画面と PR 詳細画面を左右 2 ペインにする。左は今までの詳細、右は紐づく Claude Code セッションの 状態 / 最終更新 / 直近の動き / 最終回答 / セッション URL / 取得の状態
- 端末が狭いときは右ペインを出さず、今までどおりの全幅 1 ペインで描く
- セッション情報の取得は、詳細画面を開いたときと、詳細画面で `R` を押したときだけ行う。自動更新（`refresh_interval_sec`）とキュー画面の `R` では `claude` を起動しない
- 同じプロファイルの `claude -p` は同時 1 本までとし、1 回の実行を 30 秒で打ち切る。利用上限に達したプロファイルは `resets` の時刻まで起動せず、右ペインに「制限中」と前回の内容を出す
- 詳細画面の `R` が「何もしない」から「セッションを取得する」に変わるので、ヘルプ画面のキーの一覧と README の「再取得（`R`）」を書き換える

## Capabilities

### New Capabilities
- `claude-client`: `claude -p` をプロファイルごとに起動し、`RemoteTrigger` の 3 つの用途（`list` / `list_runs` / `get_run_log`）を呼んで `tool_result` の本文を返す。stream-json の読み方、`HTTP <status>` の判定、利用上限・未ログイン・`claude` が見つからない場合のエラーを含む
- `session-pane`: 詳細画面の右ペインに出す内容と、セッション情報を取得するタイミング（開いたとき / `R`）、プロファイル単位の同時実行と利用上限の扱い

### Modified Capabilities
- `card-detail`: 「詳細の本文領域はスクロールし、ヘッダ領域は固定する」が定める 1 ペインを、左右 2 ペイン（狭い端末では 1 ペイン）に変える
- `config-loading`: `repos` の要素に `claude_config_dir` を足し、`~` と環境変数を展開する
- `manual-refresh`: 「カード詳細画面 / PR 詳細画面では `R` は何もしない」を、詳細画面ではセッションを取得するに変える
- `help-screen`: キーの一覧の `R` と `u` の説明を書き換える
- `url-picker`: URL 一覧の出典に `session` を足し、右ペインのセッション URL を `u` から開けるようにする

## Impact

- `internal/config/config.go`: `Repo` に `ClaudeConfigDir`、`UnmarshalYAML` に `claude_config_dir`、パスの展開
- `internal/claude/`（新規）に client と stream-json のデコードを置き、fixture とそのテストを同じパッケージに持たせる
- `internal/model/parse.go` に、PR 本文と issue コメントからセッション ID を取り出す関数を足す
- `internal/ui/detail.go` と `internal/ui/view.go` が、2 ペインの割り付けと右ペインの描画を受け持つ
- `internal/ui/model.go` に、セッション取得のコマンドとメッセージ、取得結果のキャッシュ、プロファイル単位の同時実行と利用上限の状態を持たせる
- `internal/ui/help.go` と `README.md`（設定ファイルの表・再取得の節・詳細画面の節）の記述を書き換える
- 直下の `CLAUDE.md` に「`docs/mvp` はこれ以降更新しない」のルールを足す（この PR で済ませる）
- 外部依存として `claude` コマンドを使う（Claude Code 2.1.267 で確認）。コマンドが無い場合は右ペインにその旨を出し、他の機能は動き続ける

## 確定した判断

- **`claude -p` の起動オプションは issue #18「取得コマンド」のまま採る。** `--model haiku` 固定、`--tools RemoteTrigger --allowedTools RemoteTrigger`、`--strict-mcp-config`、`--setting-sources ""`、`--disable-slash-commands`、`--max-budget-usd 0.05`、`--output-format stream-json --verbose`、標準入力に `/dev/null`。いずれも issue に実測の根拠（この 4 つを外すと約 15 秒 / $0.042、`< /dev/null` が無いと 3 秒待つ）がある
- **モデルの回答文は読まず、`type == "user"` の行の `tool_result` だけを正とする。** `RemoteTrigger` のログには Routine が読んだ issue や Web ページの文章がそのまま入るため（issue #18「出力の読み方」）
- **右ペインは両方の詳細画面に出す。** issue #18 の完了条件が「カード詳細と PR 詳細の右ペインに」と書いており、2026-09-11T04:07:25Z のコメントは取得のタイミングを変えるものでセッションを出す画面を減らすものではない
- **出すセッションの決め方も issue #18 のまま。** PR 詳細はその PR 本文の `https://claude.ai/code/session_...`、カード詳細は issue の routine コメントのうち `session:` を持つ最新のもの（`internal/model` が `<!-- routine -->` と `&lt;!-- routine --&gt;` の両方を routine と見なす既存の規則を使う。`internal/model/parse.go:11`）
- **取得結果はプロセス内のメモリにだけ持つ。** `internal/snapshot` に混ぜない。スナップショットは D-002 が「起動直後の stale 表示」のために持つ Card 群で、利用枠を使う取得の結果を跨いで残す根拠が無い。「制限中」の状態も同じくメモリだけに持つ（制限中の `claude -p` は即座に失敗して利用枠を使わないので、再起動後に 1 回空振りしても失う物が無い）
- **fixture は issue #18 に記録された出力の形式から手書きする。** このセッションには `claude` の OAuth も Claude のプロファイルも無く、実出力を採取できない。`internal/gh/testdata/fixtures/example`（s03 の手書き fixture）と同じ扱いで、`cmd/loop-cli-dev fixture capture` の採取経路は作らない（`claude -p` の採取は利用枠を使い、`--alias` で伏せ字にする既存の仕組みとも別物になる）。残るリスクは「手書きの形式が実際とずれていればパースが外れる」ことで、そのときは右ペインに取得の状態として現れる
- **`SugiKent/loop-cli` の Routine のプロファイルは未検証のまま残す。** issue #18 は `~/.claude-personal` と推定しつつ「まだ確認していない」と書いており、2026-09-10 時点でそのプロファイルは週の利用上限に達していた。完了条件の「ccp の制限が解けたあと（9/14 以降）`session_01N9YWTYcwFdwhLD38CdASgA` で `get_run_log` が 200 を返すことを確認する」は `tasks.md` に置かない（tasks は全行 `[x]` で archive が通る決まりで、セッション内に完了できない行を置くと archive が止まる）。プロファイルが違えば右ペインに `HTTP 404`（プロファイル違いの可能性がある）と出るので、設定を直す手掛かりは画面に出る
- **詳細画面を開いたときの取得は、そのセッションの結果をまだ持っていないときだけ行う。** 持っていれば前回の内容と取得時刻を出し、取り直しは `R` に任せる。カード詳細と PR 詳細を `Enter` / `g` / `Esc` で行き来しても、同じセッションについて 2 回目の `claude` は起動しない（PR #31 のレビューコメント 2026-09-11T06:47:10Z で人が推奨案を選んだ）
- **右ペインの「状態」はログの最新行から決め、1 回の取得で `claude` を 1 回だけ起動する。** `get_run_log` だけを呼び、`list` と `list_runs` は実装しない。キュー画面の `R` では `claude` を起動しない。代わりに `requires_action`（入力待ち）と `archived`（終了）は区別できず、ログが `result:` で終わっていなければ「実行中」に見える（同じレビューコメント）
- **セッション URL は `u` の URL 一覧に出典 `session` の 1 件として足す。** `o` の意味は変えず、専用のキーも足さない（同じレビューコメント）
- **`docs/mvp` はこれ以降更新しない。** 直下の `CLAUDE.md` にルールとして書き、この change の tasks からも `docs/mvp` を直す行を落とした。仕様の正本は `openspec/specs/` に寄せる（PR #31 のレビューコメント 2026-09-11T06:47:00Z）
