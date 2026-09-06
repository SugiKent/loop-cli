## Why

s23 で `go.mod` のモジュールパスを公開リポジトリと同じ `github.com/SugiKent/loop-cli` に直したが、直したのはモジュールパスだけで、名乗りは `sugi-loop` のまま残った。バイナリ名・起動コマンド・ヘッダのアプリ名・設定ファイルとキャッシュのパス・開発補助 CLI の名前・README と docs の全記述が `sugi-loop` を指しており、公開リポジトリ名と食い違う。

`go install github.com/SugiKent/loop-cli/cmd/sugi-loop@latest` のように、1 つのコマンドの中でリポジトリ名とコマンド名が違うのは読み手を迷わせる。利用者は `loop-cli` で起動することを求めている。

## What Changes

- `cmd/sugi-loop` を `cmd/loop-cli` に改名する。バイナリ名と起動コマンドは `loop-cli` になり、`version` / `update` のサブコマンドと使い方の文言もこれに従う
- 動作確認用の CLI `cmd/sugi-loop-cli` を `cmd/loop-cli-dev` に改名する（本体が `loop-cli` になり名前が衝突するため。利用者の決定）
- 設定ファイルを `~/.config/loop-cli/config.yml`、スナップショットを `~/.cache/loop-cli/snapshot.json` に変える（利用者の決定）。旧パスからの移行処理は書かない。旧パスを持つ利用者は onboarding のフォームがもう一度出るので、そこで入れ直す
- TUI のヘッダのアプリ名、デスクトップ通知のタイトル、回答用の一時ファイル名を `loop-cli` にする
- README・`docs/mvp/*`・`docs/domain/*`・`docs/agent-memory/*`・`openspec/config.yaml`・`.gitignore`・`.serena/project.yml` の記述をすべて追随させる
- アプリ名が 1 列狭くなる（`sugi-loop` 9 列 → `loop-cli` 8 列）ので、ヘッダの短縮が起きる端末幅が 1 列ずれる。`queue-screen` の該当 Scenario の幅と期待値を実際の挙動に合わせて直す
- `openspec/changes/archive/**` と、この change の delta が更新する前の `openspec/specs/**` は書き換えない（`.claude/rules/openspec-immutable.md`）。過去の change に残る `sugi-loop` は当時の記述として残す

この change を archive した後でなければ `s14-merge-pr` は archive できない。s14 の delta は `tui-entrypoint` の Requirement を新しい名前（`loop-cli バイナリが起動して 1 フレーム描画する`）で参照している。

## Capabilities

### Modified Capabilities

- `tui-entrypoint`: バイナリ名・起動コマンド・Requirement 名
- `dev-cli`: 開発補助 CLI の名前と Requirement 名
- `queue-screen`: ヘッダのアプリ名と、幅による短縮の Scenario
- `config-loading`: 設定ファイルの既定パス
- `snapshot-cache`: スナップショットの既定パスと Requirement 名
- `self-update`: `version` / `update` の出力と `go install` の対象
- `go-module`: `go install` の対象と `.gitignore` のパターン
- `onboarding` / `desktop-notify` / `answer-question` / `auto-refresh` / `gh-client` / `fixture-capture`: 文中の `cmd/sugi-loop` / 通知タイトル / 一時ファイル名の追随
