## 1. モジュールパスの是正

- [ ] 1.1 `go.mod` の `module` を `github.com/SugiKent/loop-cli` に変え、リポジトリ内の全 import パスを置換して `go build ./... && go test ./...` が通ることを確認する
- [ ] 1.2 README のインストール手順（`go install`）を新しいパスに直す

## 2. 版の取得（internal/version）

- [ ] 2.1 `internal/version` に、現在のモジュールパスと版を `debug.ReadBuildInfo()` から返す関数を書く（読めない / 空は `(devel)`）
- [ ] 2.2 最新の版を `go list -m -json <module>@latest` から取る関数を書く（`go` の実行は差し替え可能な関数を通す。失敗・非 JSON・空 `Version` はエラー）
- [ ] 2.3 `go install <module>/cmd/sugi-loop@latest` を実行する関数を書く（標準出力・標準エラーを呼び出し側の Writer に流す）
- [ ] 2.4 `internal/version` のテストを書く（コマンドの引数・JSON の解釈・失敗のエラー文言。本物の `go` とネットワークは使わない）

## 3. サブコマンド（cmd/sugi-loop）

- [ ] 3.1 `main` を `run(args []string, stdout, stderr io.Writer) int` に組み替える（引数なしは今までの TUI 起動、`os.Exit` は `main` だけが行う）
- [ ] 3.2 `version` サブコマンドを実装する
- [ ] 3.3 `update` サブコマンドを実装する（最新取得 → 同版なら `最新版です:` → 違えば install → `更新しました: <現在> → <最新>`。`go` 不在は 2 行の案内）
- [ ] 3.4 未知の第 1 引数で `unknown command:` と使い方を出し、終了コード 1 で終わるようにする
- [ ] 3.5 `cmd/sugi-loop` のテストを書く（振り分け・`version` の出力・`update` の 3 経路・未知のサブコマンド。`gh` と設定ファイルを読まない）

## 4. 起動時チェック（internal/ui）

- [ ] 4.1 `ui.Options` に更新確認の関数の欄と、結果のメッセージ・`Model` の印を足し、`Init` で 1 度だけ確認を開始する（`nil` なら何もしない）
- [ ] 4.2 ヘッダに `↑ update` を `↻ HH:MM` の左へ出し、幅が足りないときは `↻ HH:MM` より先に落とすように `header` を直す
- [ ] 4.3 `cmd/sugi-loop` から渡す確認関数を書く（`(devel)` は調べない、10 秒の時間制限、失敗は「新しい版は無い」と同じ扱い）
- [ ] 4.4 `internal/ui` のテストを書く（更新ありでヘッダに出る・無し / 失敗では何も出ない・幅 60 で印が先に落ちる・`nil` では呼ばない・`R` で再確認しない）

## 5. 正本ドキュメント

- [ ] 5.1 README に `sugi-loop update` / `sugi-loop version` の節を足す
- [ ] 5.2 `docs/mvp/mvp.md` の画面構成のヘッダ例に `↑ update` の説明を足し、`docs/mvp/implementation-tasks.md` に自己更新の項目と変更履歴を 1 行ずつ足す
- [ ] 5.3 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
