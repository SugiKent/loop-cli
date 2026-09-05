## Context

s04-fixture-capture が `cmd/sugi-loop-cli`（`run(args, stdout, stderr) int` による振り分け、usage 定数、`fixture capture`）を、s05-classify が `internal/model`（`Issue` / `PR` / `Comment` / `Result` / `Situation.Kind()`）と `internal/classify`（`Issue()` / `PR()` / `Card()`）を定義している。どちらも書き上げ済みで未 archive。この change は両方が実装された後に着手する。
implementation-tasks.md §2 の「動作確認用 CLI」は 3 用途を挙げており、fixture 保存は s04 が済ませた。残りは「fixture に分類器をかけてキューをプレーンテキスト出力」と「テスト通知の発火」。85-create-cli スキルは参照できないので、s04 と同じく標準ライブラリ + サブコマンド方式で最小に書く。

## Goals / Non-Goals

**Goals:**
- `classify --fixture <alias>` 1 コマンドで、fixture の全 issue / PR がどのタブにどの種別で出るかを端末で読める
- `notify test` 1 コマンドで、この環境でデスクトップ通知が出ることを確かめられる。s13 が同じ依存を使う
- s04 の骨組みを変えずにケースを足す

**Non-Goals:**
- この CLI は Issue と PR を紐づけず（Card 化しない）、`classify.Card` を呼ばない。紐づけは s07 が担当する
- live の `gh` からの分類（`classify --live`）。未決事項に既定値を書く
- 色・列幅の揃え・端末幅への追従。s08 のキュー画面が担当する
- 通知の差分比較、設定ファイル `notify:` の参照、通知クリック。s13 / s19 が担当する
- 出力形式の golden ファイル。テストは行数と各行に含まれる文字列だけを見る

## Decisions

### ファイル構成

```
cmd/sugi-loop-cli/main.go            # 変更: usage に 2 行、run に classify / notify のケース
cmd/sugi-loop-cli/main_test.go       # 変更: help の 2 行、notify 単独 / notify foo、classify のフラグエラー
cmd/sugi-loop-cli/classify.go        # classifyFixture(args, stdout, stderr) error（s04 の fixtureCapture と同形）、elapsed(now, t time.Time) string
cmd/sugi-loop-cli/classify_test.go   # example の 7 行と経過の表記
cmd/sugi-loop-cli/notify.go          # notifyTest(stdout) error
go.mod / go.sum                      # github.com/gen2brain/beeep
```

### 行は issue / PR 単位。Card は組み立てない

s05 は Card の組み立て（PR title `[<段階>] #<n>` / `Refs #n` のパース）を s07 に委ね、`classify.Card` の呼び出し側が組み立て済みの `Card` を渡す前提にしている。この CLI が紐づけを自前で書くと s07 の仕様を先取りして 2 か所に持つことになる。mvp.md の画面例も PR を独立した行（`PR131 [propose] #108 …`）で出しているので、issue 1 件 / PR 1 件を 1 行にして `classify.Issue` / `classify.PR` を直接呼ぶ。
入力の組み立ては s05 の `fixture_test.go`（tasks 5.1）と同じ手順を写す。テストコードは import できないので重複するが、数十行で、s07 の取得層ができればどちらも s07 の関数に置き換わる。

### 出力は 4 節・タブ区切り

- 節の見出しは mvp.md のタブバー `[1]今やる 7  [2]バックログ 12  [3]進行中 5  [4]異常 1` の 1 項目ずつを 1 行にしたもの。列の見出し行は出さない（列の順は usage に書く）。テストの行数が「4 + 行数」で数えやすい
- 列の区切りはタブ文字 1 つ。`todo 候補` のように種別に空白が入るので空白区切りでは列が曖昧になる。列幅を揃えない（日本語の表示幅の計算は s08 の画面が持つ。ここで runewidth 系の依存を足さない）
- 優先の列は `Result.Priority` の数値。mvp.md の画面例は `!!` / `!` / `●` の記号だが 9 局面のうち 3 つ分しか例が無く、残りを決めるのは発明になる。記号への変換は s08 が画面で決める
- 種別は s05 の `Situation.Kind()` をそのまま使う（mvp.md の種別列に D / G / その他 / 進行中を足したもの。s05 の未決事項で確定済み）
- 経過は `time.Now()` と `UpdatedAt` の差。`now` を引数で注入しない（`run` の引数は s04 で固定。テストは `elapsed(now, t)` を直接呼ぶ）

### notify test は beeep を直接呼ぶ

`internal/notify` のような薄い包みは作らない。使うのはこの CLI と s13 の 2 か所で、どちらも「タイトルと本文で 1 件出す」だけなので、beeep の関数をそれぞれが直接呼べば足りる。s13 が「再利用」するのは依存（go.mod の 1 行）であり、関数ではない。s13 で共通化が必要になればそのとき s13 が置き場所を決める。
通知の発火は `go test` で検証しない（CI で通知が出る・環境によって失敗する）。差し替え用の変数も置かず、tasks で利用者が手で実行して確認する。テストが見るのは振り分け（`notify` 単独 / `notify foo`）だけ。

### classify は fixture 専用

live の横断取得（search 2 回 + 遅延詳細取得 + Card 化）は s07 の担当で、その動作確認は TUI 本体（V-2）で行う。`classify` に live を足すと s07 の取得層と TUI の 2 か所から同じ関数を呼ぶ形になり、CLI 側の表示（Card 無し・記号無し）が TUI と食い違ったまま残る。fixture 専用にして、live の確認は TUI に任せる。

### エラーの扱い

- `--fixture` 無し・未知のフラグ: s04 と同じく `flag.NewFlagSet("classify", flag.ContinueOnError)` + `SetOutput(stderr)`。`--fixture` が空ならエラー文字列に `--fixture` を含める
- `internal/gh/testdata/fixtures` 自体の不在: s04 の `fixtureCapture` と同じく、リポジトリのルートで実行するよう促すエラーを返す（`go run ./cmd/sugi-loop-cli …` をルートで実行する前提）
- `<alias>` ディレクトリの不在: `os.Stat` で先に確認し、パスを含むエラーを返す。`Fake` の「対応するファイルが無ければそのパスを含むエラー」（s03）に任せると `search-issues.json` の不在として出て、alias の綴り間違いと区別しにくい
- `Fake` の読み取り失敗: そのまま返す。出力は全件の分類が終わってから書き始めるので、途中失敗で半端な出力が残らない

## Risks / Trade-offs

- [s05 の `fixture_test.go` と入力組み立てが重複する] → s07 の取得層に置き換える。それまでは数十行の重複を許容する
- [`beeep` が環境によって動かない（Linux で通知デーモンが無い等）] → `notify test` がそのエラーをそのまま出す。この change が依存を足す目的は「この環境で出るかを s13 の前に確かめる」ことなので、失敗が見えれば足りる
- [`beeep` の依存が `cmd/sugi-loop` のバイナリに入るのは s13 から] → この change では `cmd/sugi-loop-cli` だけが import する。バイナリサイズの変化は s13 で見る
- [列幅を揃えないので端末で読みにくい] → dev 専用 CLI。`column -t -s $'\t'` に通せば揃う。画面は s08 が持つ
- [s03 の `example` に `issue-140.json` が無い] → s05 が tasks 5.0 で `example/issue-140.json` を追加している前提。s06 は fixture を変更しない。`fs.ErrNotExist` を「詳細なし」と読み替える案は採らない（s03 `gh-fake` の「空の結果を返して黙って通さない」に反する）
- [`example` の期待値が s05 と二重に書かれる] → s05 の期待値表が正本で、この CLI のテストはそれを写す。s05 の期待値が変われば両方直す（fixture が変わるのは意図的な変更のときだけ）

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| `classify --live` を s07 完了後に足すか | 足さない | 上記 Decisions。live の確認は TUI 本体（V-2）で行う。必要になったら s07 以降の change が「取得層の関数を呼んで同じ出力を出す」形で足す |
| 優先の列の表記 | `Result.Priority` の数値（0〜7） | mvp.md の記号は 3 局面分しか例が無い。記号への変換は s08 |
| 節内の並び | 第 1 キー `Priority` 昇順、第 2 キー `Repo` 昇順、第 3 キー issue → PR、第 4 キー番号昇順 | s05 が「第 1 キーは `Priority`、第 2 キー以降は s08」と定めている。CLI は決定的な並びであれば足りる |
| 列の見出し行 | 出さない | 行数のテストを単純にする。列の順は usage に書く |
| 列の区切り | タブ文字 1 つ | 種別に空白を含む値（`todo 候補`）がある |
| 経過の表記 | 1 時間未満 `<m>m`、24 時間未満 `<h>h`、それ以上 `<d>d`。切り捨て。負なら `0m` | mvp.md の画面例（`12m` / `1h` / `3h` / `2d`）に合わせる |
| 経過の基準時刻 | `time.Now()`。注入しない | `run` の引数は s04 で固定。テストは `elapsed(now, t)` を直接呼ぶ |
| 件数 0 の節 | 見出し行だけ出す | 4 タブが常にあることを見せる |
| 通知のタイトルと本文 | `sugi-loop` / `テスト通知` | 最小。アイコンは指定しない |
| 通知の包み | 作らない。beeep を直接呼ぶ | 上記 Decisions |
| 通知発火のテスト | `go test` ではしない。利用者が手で実行する | CI で通知を出さない。環境依存 |
| `classify` / `notify test` の余分な位置引数 | エラー（フラグ解析後に引数が残っていれば、その引数を含むエラー） | `classify example` のような書き間違いを黙って通さない |
