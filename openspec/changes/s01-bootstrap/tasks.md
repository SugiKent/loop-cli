## 1. Go モジュールと依存

- [ ] 1.1 `go mod init github.com/SugiKent/sugi-loop` を実行し、`go.mod` の `go` ディレクティブが 1.26 系（手元の go1.26.6 なら `1.26.6`）で書き出されたことを確認する。手で書き換えない
- [ ] 1.2 `go get charm.land/bubbletea/v2@latest charm.land/bubbles/v2@latest charm.land/lipgloss/v2@latest charm.land/glamour/v2@latest charm.land/huh/v2@latest` で 5 つを require し、`go list -m all` に 5 つが出ることと beeep が無いことを確認する
- [ ] 1.3 `.gitignore` を作成し、`/bin/` / `/sugi-loop` / `/cmd/sugi-loop/sugi-loop` の 3 行だけを書く（先頭 `/` 必須。`sugi-loop` だけだと `cmd/sugi-loop/` ディレクトリごと無視される）。`git status --porcelain` で `cmd/sugi-loop/` 配下のソースが無視されていないことを確認する

## 2. hello world TUI

- [ ] 2.1 `cmd/sugi-loop/main.go` を作成する。フィールド無しの `model` 型に Init / Update / View を実装し、View は Lip Gloss で装飾した `sugi-loop` と `q で終了` の 2 行を返す。Update は `q` と `ctrl+c` で終了コマンドを返し、他は model をそのまま返す。`main()` はプログラム実行のエラーを標準エラーに出して終了コード 1 にする
- [ ] 2.2 `cmd/sugi-loop/main_test.go` を作成する。View に `sugi-loop` と `q` が含まれること、`q` / `ctrl+c` で終了コマンドが返ること（返ったコマンドを実行して終了メッセージになることまで確認）、`j` では nil が返ることを検証する
- [ ] 2.3 `go build -o bin/sugi-loop ./cmd/sugi-loop` でバイナリが生成され、端末で起動して 1 フレーム描画され `q` で終了コード 0 になることを手元で確認する。`/dev/tty` を開けない環境（CI runner、または `setsid` 等で制御端末を切り離した実行）で実行し、標準エラーに `error opening TTY` を含むエラーが出て終了コード 1 になることを確認する（Bubble Tea v2.0.9 で確認済みの挙動。手元の対話シェルから `< /dev/null` にしても `/dev/tty` が開けるので再現しない）

## 3. lint

- [ ] 3.1 golangci-lint を公式手順でインストールし（バージョンは design.md 未決事項「golangci-lint の固定バージョン」に書いた v2.x.y と同一にする）、`.golangci.yml` を作成する（`version: "2"`、linter は既定集合のまま、formatter に gofmt を有効化。それ以外は書かない）。`golangci-lint config verify` で設定を検証する
- [ ] 3.2 `gofmt -l .` が空、`go vet ./...` と `golangci-lint run ./...` が終了コード 0 であることを確認する。gofmt に反する一時ファイルを置いて `golangci-lint run ./...` が失敗することを確認し、一時ファイルを削除する

## 4. CI

- [ ] 4.1 `.github/workflows/ci.yml` を作成する。トリガーは `main` への push と `main` 向け pull_request、runner は `ubuntu-latest`、`actions/setup-go` に `go-version-file: go.mod` を指定し、`go build ./...` → `go vet ./...` → `go test ./...` → `golangci/golangci-lint-action` の順で 1 job で実行する。action は現行のメジャータグ（`@v8` 系。実装時に README で確認）を使い、`version:` 入力に design.md 未決事項「golangci-lint の固定バージョン」と同一の v2.x.y を明示固定する（`latest` 不可。action のタグと golangci-lint のバージョンは別物なので混同しない）
- [ ] 4.2 ワークフローの各ステップと同じコマンド列（`go build ./... && go vet ./... && go test ./... && golangci-lint run ./...`）を手元で実行して通ることを確認する。remote の作成と push、Actions が success になる確認はユーザーの作業とし、実装者は行わない（ユーザーから push の指示があった場合のみ実行し、結果を報告する）

## 5. 最終確認

- [ ] 5.1 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
