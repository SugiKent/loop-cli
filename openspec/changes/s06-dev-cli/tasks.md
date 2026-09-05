## 1. 依存と振り分け

- [x] 1.1 `go get github.com/gen2brain/beeep` で依存を追加し、`go mod tidy` で `go.mod` / `go.sum` を固定する
- [x] 1.2 `cmd/sugi-loop-cli/main.go` を変更する。usage 定数に `classify --fixture <alias>`（列の順「優先 / 種別 / リポジトリ / 番号 / タイトル / 経過」を添える）と `notify test` の 1 行説明を足し、`run` に `classify` → `classifyFixture(args[1:], stdout, stderr)`、`notify` → `len(args) >= 2 && args[1] == "test"` なら `notifyTest(stdout)`、それ以外は `unknown command: notify` のケースを足す。s04 の `fixture` / `help` / unknown の振る舞いは変えない
- [x] 1.3 `cmd/sugi-loop-cli/main_test.go` に追加する: `help` の標準出力に `classify --fixture` と `notify test` を含む / `notify` 単独と `notify foo` が標準エラーに `unknown command: notify` で 1 / `classify`（フラグ無し）が標準エラーに `--fixture` で 1 / `classify --fixture example --live` が標準エラーに `live` で 1。s04 の既存ケースが変更なしで通ることを確認する

## 2. classify サブコマンド

- [x] 2.1 `cmd/sugi-loop-cli/classify.go` を作成し、`classifyFixture(args []string, stdout, stderr io.Writer) error` を実装する（s04 の `fixtureCapture` と同形）。`flag.NewFlagSet("classify", flag.ContinueOnError)` + `SetOutput(stderr)` で `--fixture` を解析（空ならエラーに `--fixture`、余分な位置引数はその引数を含むエラー）→ `internal/gh/testdata/fixtures` を `os.Stat` で確認（無ければ s04 と同じく、リポジトリのルートで実行するよう促すエラー）→ `internal/gh/testdata/fixtures/<alias>` を `os.Stat` で確認（無ければパスを含むエラー）→ `gh.NewFake(dir)` で `SearchIssues` / `SearchPRs` を読み、各 issue は `model.IssueFromSearch` + `ViewIssue` の `Comments`（`model.CommentFrom`）、各 PR は `model.PRFromSearch` + `ViewPR` の `Comments` + `ViewPRMergeState` + `ReviewThreads` を入れて `classify.Issue` / `classify.PR` を呼ぶ → 4 タブ（`model.TabNow` / `TabBacklog` / `TabInProgress` / `TabAbnormal` の順）に振り分け、各タブ内を `Priority` 昇順 → `Repo` 昇順 → issue → PR → 番号昇順に並べる → 見出し行 `[1]今やる <件数>` … `[4]異常 <件数>` と行（優先 / 種別 / リポジトリ / 番号 / タイトル / 経過をタブ区切り。番号は `#<n>` / `PR<n>`、種別は `Situation.Kind()`）を stdout に書く。読み取りが 1 つでも失敗したら出力を書かずにエラーを返す。経過は `elapsed(now, t time.Time) string`（1 時間未満 `<m>m`、24 時間未満 `<h>h`、以上 `<d>d`、負なら `0m`）で `time.Now()` を渡す
- [x] 2.2 `cmd/sugi-loop-cli/classify_test.go` を作成する。`t.Chdir("../..")` でリポジトリのルートに移り、`run([]string{"classify", "--fixture", "example"}, …)` の標準出力が 7 行で、順に `[1]今やる 1` / `PR131` と `質問` と `org/app` を含む / `[2]バックログ 1` / `#140` と `todo 候補` を含む / `[3]進行中 1` / `#108` と `進行中` を含む / `[4]異常 0`、終了コード 0 であることを検証する。さらに `--fixture nonexistent` を渡したとき、標準エラーが `internal/gh/testdata/fixtures/nonexistent` を含み、終了コードが 1 で、標準出力が空であることを検証する。`elapsed(now, t)` は 12 分前 / 3 時間前 / 2 日前 / 未来の入力に対して `12m` / `3h` / `2d` / `0m` を返すことを検証する。golden ファイルは作らない

## 3. notify test サブコマンド

- [x] 3.1 `cmd/sugi-loop-cli/notify.go` を作成し、`notifyTest(stdout io.Writer) error` を実装する。beeep でタイトル `sugi-loop`、本文 `テスト通知` の通知を 1 件出し（アイコン指定なし）、エラーならそのまま返し、成功なら stdout に `通知を送りました` を 1 行書く。`go test` では発火させない（差し替え用の変数も置かない）
- [x] 3.2 利用者が `go run ./cmd/sugi-loop-cli notify test` を実行し、デスクトップ通知が 1 件出て標準出力に `通知を送りました` が出ることを確認する。出なければエラー文言を控え、s13 の着手前に環境側（通知の許可設定等）を直す

## 4. 最終確認

- [x] 4.1 `gofmt -l .` が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する（`golangci-lint run ./...` も s01 の CI が回すので通す）
