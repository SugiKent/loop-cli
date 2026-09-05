## Context

リポジトリには docs と openspec しか無く、Go のソースもモジュールも無い。これが最初の change で、後続 change（s02〜s21）は
すべて「`go build ./... && go vet ./... && go test ./...` が通る」ことを完了条件にする。D-003 の技術選定（Go 1.26 / Bubble Tea v2 /
Bubbles v2 / Lip Gloss v2 / Glamour / huh、配布は go install）を go.mod に写し、CI で守る。

現状の事実:
- 手元の Go は `go1.26.6 darwin/arm64`
- `git remote` は未設定
- golangci-lint は手元に未インストール
- `go mod init` は `go 1.26.6` を書き出す（1.26 ではない）
- Bubble Tea v2.0.9 を使い捨てモジュールで実行して確認: `/dev/tty` を開けない環境では `Run` がエラー
  `bubbletea: error opening TTY: ...` を返す（panic しない）。対話シェルで `< /dev/null` にしただけでは `/dev/tty` が開けるので再現しない
- Go モジュールプロキシで確認した各ライブラリの正規モジュールパス（go.mod の `module` 宣言）は `charm.land/bubbletea/v2` /
  `charm.land/bubbles/v2` / `charm.land/lipgloss/v2` / `charm.land/glamour/v2` / `charm.land/huh/v2`。
  `github.com/charmbracelet/<name>/v2` でも取得できるが go.mod には `charm.land` が書かれるので、最初から `charm.land` を使う

## Goals / Non-Goals

**Goals:**
- `go build ./...` が通る Go モジュールと、D-003 の依存 5 つの固定
- gofmt / `go vet` / golangci-lint（既定 + gofmt 相当）の導入
- `cmd/sugi-loop` の hello world（1 フレーム描画、`q` で終了）
- GitHub Actions による build / vet / test / lint

**Non-Goals:**
- `internal/*` と `cmd/sugi-loop-cli` の作成（s02〜s06 が担当）
- キュー画面・設定読み込み・gh 呼び出し（s02 以降）
- beeep の依存追加（s13 が担当）
- GoReleaser / gh extension 配布（D-003 で候補扱い。この change では触れない）
- `go mod tidy` の差分チェックを CI に入れること（後述の Decisions を見る）

## Decisions

### ファイル構成

```
go.mod / go.sum
.gitignore                    # /bin/ /sugi-loop /cmd/sugi-loop/sugi-loop（先頭 / 必須。cmd/sugi-loop/ を無視しないため）
.golangci.yml
.github/workflows/ci.yml
cmd/sugi-loop/main.go         # model 型（Init / Update / View）と main()
cmd/sugi-loop/main_test.go    # View の内容と q / Ctrl+C / 他キーの Update を検証
```

`internal/ui` は作らない（s08 が作る）。hello world の model は `cmd/sugi-loop/main.go` に package main で置く。s08 がキュー画面を
`internal/ui` に作ったら、main.go はそれを起動するだけに置き換わる。

### hello world の責務

- `model` 型: 状態を持たない（フィールド無し）。`View` は Lip Gloss で装飾した 2 行（アプリ名 `sugi-loop` と `q で終了` の案内）を返す。
- `Update`: `q` と `Ctrl+C` のキー入力で終了コマンドを返す。他のメッセージは無視して model をそのまま返す。
- `main()`: Bubble Tea のプログラムを起動し、エラーが返ったら `fmt.Fprintln(os.Stderr, err)` の後 `os.Exit(1)`。エラーの代表例は上記 Context の TTY エラー。
- キーの判定は Bubble Tea v2 のキーメッセージが持つ文字列表現（`q` / `ctrl+c`）で行う。Bubbles の `key` / `help` は hello world では使わない
  （キーが 2 つしかない段階でキーマップ抽象を入れない。CLAUDE.md「一度しか使わない処理のためにヘルパーを増やさない」）。

テストは package main に置き、以下を検証する。Bubble Tea のメッセージ型名は実装時に v2 のドキュメントで確認する。
- 初期 model の `View()` 文字列に `sugi-loop` と `q` が含まれる
- `q` のキー入力メッセージを `Update` に渡すと非 nil の終了コマンドが返る（返ったコマンドを実行して得たメッセージが終了メッセージであることを確認する）
- `Ctrl+C` も同様
- `j` を渡すと nil のコマンドが返る

### 依存 5 つの固定と `go mod tidy` の関係

hello world が import するのは Bubble Tea v2 と Lip Gloss v2 だけで、Bubbles v2 / Glamour v2 / huh v2 はこの change のコードから
import されない。`go mod tidy` は import されていないモジュールを require から落とすため、次のように扱う。

- 5 つすべてを `go get charm.land/<name>/v2@latest` で go.mod に require する（implementation-tasks.md §1 の「依存追加」を満たす）
- CI では `go mod tidy` の差分チェックを**行わない**。行うと import されていない 3 つが落ちて失敗する
- 実装者が手元で `go mod tidy` を実行して 3 つが落ちた場合は `go get` で戻す。落ちたまま commit しても後続 change（s08 で Bubbles、
  s09 で Glamour、s15 で huh）が import 時に再追加するので、動作は壊れない
- `tools.go` のような blank import ファイルは作らない（投機的な抽象になる）

代替案として「import するものだけ require する」も検討したが、implementation-tasks.md の項目と一致しないため採らない。

### golangci-lint

- v2 系を使う。設定ファイルは `.golangci.yml`。v2 の設定形式では linter と formatter が別セクションになるので、linter は既定集合のまま、
  formatter として gofmt を有効化する。それ以外は書かない
- 手元に golangci-lint が無いので、tasks に公式手順でのインストールを含める。CI では `golangci/golangci-lint-action` を使う。
  action 自体のメジャータグ（`@v8` 系。実装時に README で確認）と、`version:` 入力で指定する golangci-lint のバージョン（v2.x）は別物。
  後者は具体リリースを明示固定し `latest` を使わない（CI が突然壊れるのを避ける）

### CI

- トリガー: `main` への push と `main` 向け pull_request
- runner: `ubuntu-latest`
- Go のセットアップは `actions/setup-go` で `go-version-file: go.mod` を指定し、go.mod の `go` ディレクティブと二重管理しない
- ステップは順に `go build ./...` → `go vet ./...` → `go test ./...` → golangci-lint。1 job で直列に流す（並列化するほどの実行時間ではない）
- remote の作成と push はユーザーの作業。実装者は指示なしに remote を作らず push しない

## Risks / Trade-offs

- [Bubble Tea v2 のキーメッセージ表現が想定と違う] → 実装時に v2 のドキュメント（context7 または公式 README）で確認する。spec は振る舞いで書いてあるので spec 側の修正は不要
- [`go mod tidy` で未 import の 3 依存が落ちる] → 上記 Decisions のとおり CI で tidy チェックをしない。後続 change が再追加する
- [golangci-lint v2 の設定キー名を誤る] → `golangci-lint config verify` で設定ファイルを検証するタスクを入れる
- [端末が無い環境での起動エラーの挙動が Bubble Tea のバージョンで変わる] → spec は「プログラム実行がエラーを返した場合」に限定しており、エラーを返さず動く場合は該当しない

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| モジュールパス | `github.com/SugiKent/sugi-loop` | git remote が未設定。GitHub ユーザー名 SugiKent とリポジトリ名 sugi-loop（mvp.md「ツール名 `sugi-loop`」）から。remote を設定するときはこのパスに合わせる |
| Glamour / huh のメジャーバージョン | それぞれ v2（`charm.land/glamour/v2` / `charm.land/huh/v2`） | docs は「Glamour」「huh」とだけ書く。Bubble Tea v2 と同じ世代の v2 が公開されているので揃える |
| 各ライブラリの具体バージョン | `go get` 時点の最新安定版を go.mod に固定 | spec にバージョン番号を書かない |
| 初期フレームの内容 | 1 行目 `sugi-loop`、2 行目 `q で終了` | テストで assert できる最小 |
| `Ctrl+C` での終了 | 終了する | 端末ツールの慣習 |
| golangci-lint のメジャーバージョンと設定ファイル名 | v2 系、`.golangci.yml` | 現行系。設定形式が v1 と異なるので明示 |
| golangci-lint の固定バージョン | v2.13.2（実装時点のリリース一覧で確認した最新安定版。手元と CI で同一にする） | 手元と CI で結果が一致するように 1 つに固定する。`latest` は CI が突然壊れるので使わない |
| CI で golangci-lint を回すか | 回す | lint 導入が範囲内で、CI が通ることが観測手段 |
| CI のトリガーと runner | `main` への push / `main` 向け pull_request、`ubuntu-latest` | 最小構成 |
| `go` ディレクティブ | `go mod init` が書き出す `1.26.6` のまま | 手で `1.26` に丸めない。toolchain 解決を Go に任せる |
| `.gitignore` の 3 パターン | `/bin/` / `/sugi-loop` / `/cmd/sugi-loop/sugi-loop` | docs に記述が無い。`go build -o bin/sugi-loop ./cmd/sugi-loop` の出力先が `bin/`、`go build ./cmd/sugi-loop` の出力先がカレントディレクトリ（ルートで実行すれば `/sugi-loop`、`cmd/sugi-loop/` 内で実行すれば `/cmd/sugi-loop/sugi-loop`）。先頭 `/` は `cmd/sugi-loop/` ディレクトリを無視しないため |
