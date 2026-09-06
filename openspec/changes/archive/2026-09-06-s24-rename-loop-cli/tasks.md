## 1. コマンドの改名

- [x] 1.1 `cmd/sugi-loop` を `cmd/loop-cli` に、`cmd/sugi-loop-cli` を `cmd/loop-cli-dev` に `git mv` する
- [x] 1.2 `cmd/loop-cli` の使い方・doc コメント・`version` の出力を `loop-cli` に、`cmd/loop-cli-dev` の使い方・doc コメントを `loop-cli-dev` に直す
- [x] 1.3 `internal/version` の `Install` が `<module>/cmd/loop-cli@latest` を実行するようにする

## 2. パスと表示名

- [x] 2.1 `internal/config.DefaultPath` を `~/.config/loop-cli/config.yml`、`internal/snapshot.DefaultPath` を `~/.cache/loop-cli/snapshot.json` に変える
- [x] 2.2 `internal/ui` のヘッダのアプリ名・通知タイトル・回答用の一時ファイル名を `loop-cli` にする
- [x] 2.3 対応するテスト（`internal/config` / `internal/snapshot` / `internal/onboarding` / `internal/ui` / `internal/version` / `cmd/loop-cli`）の期待値を直す

## 3. ヘッダ幅の Scenario

- [x] 3.1 幅 80 のテストを「短縮せずに収まる」に変え、`[4]` のタブ名が落ちる検証を幅 79 のテストとして足す
- [x] 3.2 印を落とす検証の幅を 60 から 59 に変える
- [x] 3.3 印なしで右から短縮する検証（幅 60）の期待値を `[2]バックログ 1` が残る形に直す

## 4. ドキュメント

- [x] 4.1 README の見出し・インストール・起動・`update` / `version`・設定パス・`loop-cli-dev` の節を直す
- [x] 4.2 `docs/mvp/*`・`docs/domain/*`・`docs/agent-memory/*`・`openspec/config.yaml`・`.gitignore`・`.serena/project.yml` を直す
- [x] 4.3 未 archive の `openspec/changes/s14-merge-pr` の記述を追随させる（archive 済みの change とメイン spec は触らない）
- [x] 4.4 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
