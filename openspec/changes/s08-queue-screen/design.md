## Context

s01 の `cmd/sugi-loop/main.go` は状態を持たない hello world の `model` 型と、`tea.NewProgram(...).Run()` のエラーを標準エラーに出す `main()` だけを持つ。s02（`config.DefaultPath` / `Load`）、s03（`gh.NewClient` / `(*Client).Check`）、s05（`internal/model` の `Card` / `Result` / `Situation.Tab()` / `Kind()` / `Priority()`、`Comment.AI`）、s07（`internal/fetch` の `Fetch(ctx, client, repos) (*Result, error)`、`Result{Cards, Errors}`）は揃う前提で、まだ画面が無い。
s02 / s03 の design は配線を s07 と書き、s07 の design はそれを s08 に渡した。s07 は「`Fetch` が分類まで済ませ、s08 は並べるだけ」と定めている。s07 はレビュー修正中で、遅延取得の範囲が「全 open PR に `ViewPR`」に変わる見込みだが、この change は `Result{Cards, Errors}` の形にしか依存しない。
mvp.md「画面構成（今やるキュー）」が画面の正本で、ヘッダ（タブ名と件数、↻ 最終更新時刻）、上段の表（優先 / 種別 / リポジトリ / # / タイトル / 経過。行の色は種別で固定）、下段のプレビュー（本文の Markdown レンダリング、AI コメントの左バー、1 行目）、フッタのキーヒント、狭い端末の 1 ペインを定めている。D-002 が「取得中はステータスバーにスピナー、失敗時は前回結果を維持してエラーを赤で表示」を定めている。
go.mod には現在 Bubble Tea v2 と Lip Gloss v2 しか無い（s01 が入れた Bubbles / Glamour / huh は `go mod tidy` で落ちている。s01 design がそれを見越して「後続 change が import 時に再追加する」と書いている）。

## Goals / Non-Goals

**Goals:**
- `sugi-loop` を起動すると、設定リポジトリの Card が 4 タブに優先度順で並び、選択行のプレビューが読める
- 起動 → `config.Load` → `Check` → 非同期取得 → 表示が 1 本につながり、設定と `gh` の失敗は起動前に標準エラーで分かる
- Model の振る舞い（並び・選択・タブ・取得状態）を `internal/ui` のテストで、Fake + `example` fixture と手書きの Card で検証できる

**Non-Goals:**
- この change は `Enter` の詳細画面（s09）、`a` / `t` の書き込み（s10 / s11）、`o` / `R` / `?`（s12）、自動更新・スナップショット・通知（s13）、それ以降の change が担当するキーを実装しない
- プレビューのスクロール（キーが mvp.md に無い）
- レートリミット表示（s18）、絞り込み `/`（s19）
- `sugi-loop-cli classify` の表示を画面に合わせること（s06 が「画面の仕様にしない」と決めている）

## Decisions

### ファイル構成

```
internal/ui/model.go        # Fetcher / Model / New / Init / Update（キー・サイズ・取得メッセージ）/ 取得コマンド
internal/ui/rows.go         # Subject、タブ振り分けと並び（rows(tab) []row）
internal/ui/view.go         # View、ヘッダ / 表の 1 行 / 区切り線 / フッタ、色のスタイル、優先記号
internal/ui/preview.go      # プレビュー（1 行目 / Glamour / コメント）
internal/ui/elapsed.go      # Elapsed(now, t)
internal/ui/*_test.go       # 各ファイルに対応
cmd/sugi-loop/main.go       # 配線だけ（model 型を削除）
cmd/sugi-loop/main_test.go  # 削除（キーのテストは internal/ui に移る。ビルド確認は tasks の go build）
go.mod / go.sum             # charm.land/bubbles/v2 / charm.land/glamour/v2 を追加
```

この配置は D-003 が定めた `internal/ui`（queue / detail / reply / form / help）に従う。s09 以降の change が詳細画面などのファイルを足す。

### Model は取得関数だけを受け取る

`New(fetcher Fetcher)` の `Fetcher` は `func(ctx context.Context) (*fetch.Result, error)`。`cmd/sugi-loop` が `fetch.Fetch` を `client` と `repos` で閉じて渡す。`Model` が `gh.GHClient` や `config.Config` を持つ案は、`internal/ui` が s02 / s03 に依存し、テストで `Fake` と設定を組む手間が増えるだけで、この change の画面には `repos` も `client` も要らない。s12 の `R` はこの `Fetcher` をもう 1 回呼ぶだけで済み、s13 の自動更新も同じ。
取得の完了は非公開の `fetchedMsg{res *fetch.Result; err error; at time.Time}` で届く。`New` は `fetching` を true にした `Model` を返し（初期状態は常に「これから取得する」。`Init` はコマンドしか返せない）、取得コマンドは非公開関数 `fetchCmd(fetcher Fetcher) tea.Cmd` に分け、`func() tea.Msg { res, err := fetcher(context.Background()); return fetchedMsg{res, err, time.Now()} }` を返す。`Init` はスピナーの tick と `fetchCmd(m.fetcher)` をまとめて（`tea.Batch`）返す。まとめたコマンドはテストで分解できないので、取得だけを実行したいテストは `fetchCmd` を直接実行する（Bubble Tea v2 の API 名は実装時に確認する。振る舞いは spec のとおり）。`ctx` にキャンセルを付けない: この change には取得を止める操作が無い。`Fetch` 自体が呼び出し 1 回ごとに 30 秒の期限を持つ（s07）。
テストは `fetchCmd(fetcher)()` を実行して `fetchedMsg` を得るか、`fetchedMsg` を直接組んで `Update` に渡す。`Init` の返り値は実行しない。
s13 は D-002 の stale 表示（起動直後にスナップショットの Card を出し、背景で再取得）のため `New` の引数を拡張し、初期 `cards` と `at` を受け取れるようにする。この change の `New(fetcher)` は「取得前の空状態」しか作らない。`Fake` + `example` の `Result` は `fetch.Fetch(context.Background(), gh.NewFake("../gh/testdata/fixtures/example"), []string{"org/app"})` で作る。

### 「今」は取得完了時刻

`Model` は壁時計を読まない。経過（`Elapsed`）とヘッダの `↻ HH:MM` は `fetchedMsg.at` を使う。`time.Now` を注入する案（`Model` に `Now func() time.Time` を持たせる）は、経過が描画ごとに変わり、テストで注入の手間が増える。取得完了時刻を基準にすれば、s13 の自動更新（既定 120 秒）で経過が更新され、テストは `at` を固定するだけで済む。表示のずれは最大で更新間隔ぶん。

### 行の主体は `Result` の一致で決める

`Card.Result` は s05 `classify.Card` が Issue / PR の `Result` から選んだコピーなので、`model.Result`（string と int だけの struct）の `==` で「どの要素が 1 行目を決めたか」を復元できる。Issue → PRs の順に見るのは `classify.Card` の同点判定と同じ。候補が無い（進行中）場合も `Card.Result.Summary` は先頭の open PR か Issue の `Summary` のコピーなので同じ規則で当たる。当たらないケース（`Issue` も open PR も無く `Summary` が `進行中`）は s07 の `Fetch` では作られないが、`Issue` → `PRs[0]` のフォールバックを置いてパニックを避ける。
`Subject` を公開関数にするのは s09 のカード詳細ヘッダが同じ主体を使うため。s09 が別の決め方をするなら s09 が変える。

### タブ内の並び

第 1 キーは s05 が定めた `Priority` 昇順。第 2 キーは主体の `UpdatedAt` 降順にする。mvp.md の画面例は `!!` の 2 行が 12m → 1h、`!` の 2 行が 3h → 2d、`●` の 3 行が 5h → 9h → 1d と、どの優先度でも新しいものが上に並んでいる。s06 の CLI は `Repo` 昇順を第 2 キーにしたが、s06 自身が「画面の仕様にしない」と書いている。第 3 / 第 4 キー（`Repo` 昇順、番号昇順）は同時刻のときの決定性のためだけにある。並び替えは `sort.SliceStable` で `fetchedMsg` を受けたときに 1 回行い、`View` では並び替えない。

### 選択行は画面全体で 1 つ

タブごとに添字を持たない。「先頭から捌く」体験ではタブを切り替えたら先頭を見るのが自然で、状態も少ない。`Cards` の差し替えで選択行が指す Card が変わることは許容する（s13 の自動更新で「選択していた Card を追い続ける」が要るなら s13 が決める）。

### 表の 1 行

列幅は固定（未決事項）: `▶` 2 / 優先 4 / 種別 10 / リポジトリ 20 / 番号 7 / 経過 5（計 48 列）で、残り（幅 − 48）をタイトルに充てる。残りが負ならタイトル幅は 0 でタイトルを出さず、行は端末幅で切る（幅 48 列未満は 1 ペインの閾値 80 列を大きく下回る端末で、表として読める状態を保証しない）。日本語の表示幅は Lip Gloss（`charmbracelet/x/ansi`）の幅計算と切り詰めを使い、自前で計算しない。`todo 候補` のように空白を含む種別があるので、列は固定幅で埋め、区切り文字は使わない。
ヘッダが端末幅を超えるときは、件数付きタブ名を `[4]` から左へ順に `[n] <件数>`（タブ名を落とす）に短縮し、収まった時点で止める。`[1]` は短縮しない（`tui-entrypoint` の初期フレームが `[1]今やる 0` を期待する）。`[2]`〜`[4]` を全部短縮しても超えれば `↻ HH:MM` を省き、それでも超えれば行を幅で切る。空白は `sugi-loop` の後に 2、タブ間に 2、`↻ HH:MM` の左に最低 1（spec に固定。切り詰めの計算を決定的にするため）。1 ペインの閾値 80 列でも `sugi-loop  [1]今やる 1  [2]バックログ 1  [3]進行中 0  [4]異常 0 ↻ 12:04` は 71 列で収まるが、60 列では `[4]` → 67、`[3]` → 61 でまだ超え、`[2]` の短縮で 51 列になる。
行の色は `Kind()` で引く `map[string]lipgloss.Style`。選択行は色を変えず `▶` だけで示す（色を反転すると種別の色が読めなくなる）。その他 / 進行中は `lipgloss.NewStyle()`（装飾なし）。
優先記号は `Priority` から引く。mvp.md にある 1 / 2 / 3 の 3 つを固定し、残りは未決事項の既定値。s06 は数値のまま出しているが、s06 は「記号への変換は s08」と書いている。

### プレビュー

1 行目は主体の `Body` の 1 行目（先頭の空行を除いた最初の行）+ `labels: …`。mvp.md の画面例がそのまま PR 本文の 1 行目 `未確定の判断: 2 件 — merge しないでください` と `labels: propose question` を並べており、それに合わせる。`Labels` が空なら `labels:` を省き、`Body` も空なら 1 行目を出さない。`Card.Result.Summary`（s05「いま人が何をすべきか」）はこの画面には出さない。表の種別と優先記号が同じ情報を担っており、`Summary` は s09 のカード詳細ヘッダで使う。
本文は Glamour v2 で幅をプレビュー幅に合わせてレンダリングする（D-003「Glamour: Issue / PR 本文とコメントを色付きで端末表示」）。スタイルは自動判定（端末の明暗）に任せ、テストは ANSI を除いた文字列に本文の語が含まれることだけを見る（Glamour の出力の空白や折り返しに依存させない）。レンダリング失敗（Glamour がエラーを返す）は `Body` をそのまま出す。空の `Body` は何も出さない。
コメントは `model.Comment.AI` で分け、AI は見出しと本文の全行の左端に `▌` を付ける（mvp.md の画面例の形。`▌AI  12:04  …`）。コメント本文も Glamour でレンダリングする。幅は `▌` の 1 列を引いたプレビュー幅 − 1 で、レンダリング後に行ごとに `▌` を前置する（人のコメントも同じ幅でレンダリングし、`▌` は付けない）。本文と同じくレンダリング失敗は `Body` をそのまま出す。
`<!-- routine -->` のマーカー行（エスケープ済みの `&lt;!-- routine --&gt;` も）はレンダリング前に取り除く。Glamour は HTML コメントを出さないが、エスケープ済み形は文字列として描画され、レンダリング失敗のフォールバックでは生の形が出るので、レンダラに頼らず `internal/ui` が落とす。取り除くのはマーカー行だけで、`IsAI` の判定に使う `## PR リスク評価` は本文なので残す。s09 の詳細画面がコメントの折りたたみと `blocked-by:` 要約を持つ。
高さを超えた分は捨てる。スクロールは持たない。

### 狭い端末

閾値は幅 80 列・高さ 20 行（未決事項）。2 ペインの高さ配分は、ヘッダ 1 + 区切り 1 + フッタ 1 を引いた残りを表とプレビューで半分ずつ（奇数なら表に 1 行多く）。1 ペインでは残り全部を表かプレビューのどちらかに充てる。
切替キーは mvp.md の表に無いので発明になる。表に無い文字で、後続 change が使う予定の無い `p`（preview）を既定値にする。`Tab` はタブ切替に使われており、`Enter` は s09 の詳細に使われる。
Bubbles の `viewport` は使わない。スクロールしないので、行を高さで切るだけで足りる（Bubbles から使うのは `spinner` だけ）。

### 取得状態はフッタの右側

D-002 の「ステータスバー」はフッタと同じ行にする。画面例のフッタはキーヒントだけだが、ステータス専用の行を増やすと表の行が 1 つ減る。左にキーヒント、右にスピナー / エラー。エラーは赤（`lipgloss` の `#B60205`。`blocked` の色と同じ）。部分失敗は `詳細取得の失敗 <n> 件: <Errors[0]>` で先頭だけ出す（全部出す場所が無い。s07 は `Errors` にメソッド名・リポジトリ・番号を入れている）。
失敗時に `Cards` を保持するのは `fetchedMsg.err != nil` のときに `Cards` と `at` を触らないだけ。スナップショットからの復元は s13（上記「Model は取得関数だけを受け取る」のとおり `New` の引数を拡張する）。

### cmd/sugi-loop の配線

`main()` は `run() error` を呼んで、エラーなら標準エラーに 1 行 + `os.Exit(1)`。`run` の順は spec のとおり `DefaultPath` → `Load` → `NewClient` → `Check` → `ui.New` → プログラム実行。`Check` の `ctx` は `context.Background()`（`Check` は `gh auth status` を 1 回実行するだけ）。`repos` は `Config.Repos[i].Name` を並べた `[]string`。
`main_test.go` は削除する。s01 のテストは hello world の `model` に対するもので、置き換え後は `internal/ui` のテストが同じ Scenario（`q` / `Ctrl+C` / 他キー / 初期フレーム）を持つ。`main()` 自体のテスト（設定無し → exit 1）は `os.Exit` を伴うので書かず、tasks の手動確認にする。

### 依存の追加

`charm.land/bubbles/v2`（spinner）と `charm.land/glamour/v2` を `go get` で加える。s01 design の「Glamour は s09 で import」はプレビューの Markdown レンダリング（mvp.md）を s08 が持つので前倒しになる。`huh` は s15 まで入らない。

## Risks / Trade-offs

- [Bubble Tea v2 / Bubbles v2 / Glamour v2 の API 名が想定と違う] → spec は振る舞いで書いてある。実装時に各ライブラリの README で確認する。s01 の `main.go` / `main_test.go` に v2 のキーメッセージ（`tea.KeyPressMsg`、`Key.String()` が `q` / `ctrl+c`）の使い方がある
- [Glamour の出力が環境（端末の明暗判定）で変わる] → テストは ANSI を除いた文字列に語が含まれることだけを見る。幅や空白は見ない
- [経過が取得完了時刻基準なので、次の取得まで表示が古くなる] → 上記 Decisions。s13 の自動更新が既定 120 秒
- [1 ペインの切替キー `p` が docs に無い] → 未決事項に書く。docs にキーが足されたら合わせる
- [`Result` の `==` 比較が `Summary` の文言に依存する] → `Card.Result` は Issue / PR の `Result` のコピーなので同じ文言。s05 が `Summary` の文言を変えても比較は壊れない（両方変わる）
- [`fetchedMsg` を待つ間に `q` を押しても取得ゴルーチンは動き続ける] → Bubble Tea の終了で process が終わる。`gh` サブプロセスは `ctx` を持たないので最長 30 秒残るが、`Fetch` の `CallTimeout` で終わる
- [`internal/fetch` が未実装だと `internal/ui` がビルドできない] → tasks の前提に s07 の実装完了を書く
- [表の列幅が固定なので、長いリポジトリ名（20 列超）は切れる] → 切り詰めて `…`。設定は少数のリポジトリなので許容する

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| タブ内の並びの第 2 キー以降 | 主体の `UpdatedAt` 降順 → `Repo` 昇順 → 番号昇順 | mvp.md の画面例がどの優先度でも新しい順。s05 が第 1 キーだけ定め、s06 は「画面の仕様にしない」 |
| 優先記号（1 / 2 / 3 以外） | 0 → `!!!`、5 → `●`、4 / 6 / 7 → `-` | mvp.md は 1 → `!!`、2 → `!`、3 → `●` だけ。F（0）は最上位なので `!` を 3 つ、G（5）は C と同じ merge 系なので `●`、E / その他 / 進行中は記号なし相当 |
| 番号の表記 | `#<n>` / `PR<n>`（空白なし） | s06 と同じ。画面例は `PR131` と `PR 88` が混在しており空白なしを採る |
| シアンと黄の具体値 | todo 候補 = ANSI 6（シアン）、異常 = 背景 ANSI 3（黄）+ 前景 ANSI 0（黒） | mvp.md は色名だけ。GitHub のラベル色に対応が無いので端末の基本色を使う |
| その他 / 進行中の色 | 付けない | mvp.md に指定が無い |
| 選択行の表現 | 先頭に `▶`。色は変えない | 画面例。反転すると種別の色が読めない |
| 列幅 | `▶` 2 / 優先 4 / 種別 10 / リポジトリ 20 / 番号 7 / 経過 5（計 48）、残り（幅 − 48）をタイトル。負なら 0 でタイトルを出さず、行は幅で切る | 画面例の幅に近い固定値。48 列未満は閾値 80 列を大きく下回る |
| ヘッダの切り詰め | `[4]` から左へ `[n] <件数>` に短縮（`[1]` は残す）→ `↻ HH:MM` を省く → 行を幅で切る | 件数は捌く量の把握に要り、タブ名は番号で分かる。時刻は最後 |
| 経過とヘッダ時刻の基準 | 取得完了時刻（`fetchedMsg.at`）。壁時計を読まない | テストが決定的。s13 の自動更新で追従 |
| 時刻の表記 | `HH:MM` 24 時間表記、完了時刻のタイムゾーンのまま。コメントの `CreatedAt` も同じタイムゾーンに直す | 画面例の `12:04` |
| プレビューの 1 行目 | 主体の `Body` の 1 行目（先頭の空行を除く）+ `labels: <主体の Labels>`。`Labels` が空なら `labels:` を省く | mvp.md の画面例のとおり |
| `Card.Result.Summary` の表示 | この画面には出さない | 種別と優先記号が同じ情報を担う。s09 の詳細ヘッダで使う |
| コメント本文の Markdown レンダリング | する。幅 = プレビュー幅 − 1、AI は各行に `▌` を前置 | D-003「Glamour: Issue / PR 本文とコメントを色付きで端末表示」 |
| routine マーカー行 | `<!-- routine -->` / `&lt;!-- routine --&gt;` の行をレンダリング前に落とす | レンダラやフォールバックに関わらずマーカーを見せない |
| Glamour のスタイル | 自動判定 | docs に無い。テストは語の有無だけを見る |
| プレビューのスクロール | 持たない。高さで切る | キーが mvp.md に無い |
| 狭い端末の閾値 | 幅 80 列未満または高さ 20 行未満で 1 ペイン | docs に値が無い。80 列は端末の慣習的な最小幅 |
| サイズ受信前の想定サイズ | 幅 80・高さ 24 | 2 ペインの下限を満たす値 |
| 2 ペインの高さ配分 | ヘッダ 1 + 区切り 1 + フッタ 1 を引いた残りを半分ずつ（奇数は表に +1） | 画面例は表とプレビューがほぼ同じ高さ |
| 1 ペインの切替キー | `p` | mvp.md の表に無い文字。`Tab` / `Enter` は他の用途 |
| 1 ペインの初期表示 | 表 | 先頭から捌くのが目的 |
| ステータスの場所 | フッタの右側 | 専用行を増やすと表の行が減る |
| スピナーの文言 | スピナー + `取得中` | テストで文字列を見る |
| 部分失敗の表示 | `詳細取得の失敗 <n> 件: <Errors[0]>` を赤 | 全部出す場所が無い |
| エラーの赤 | `#B60205`（`blocked` の色） | mvp.md「エラーを赤で表示」。GitHub ラベル色と揃える方針 |
| フッタのキーヒント | `j/k 移動  1-4/Tab タブ  q 終了`（1 ペインでは `p プレビュー` / `p 一覧` を追加） | 動かないキーを出さない。後続 change が足す |
| 0 行のタブのプレビュー | `（このタブにはカードがありません）` | 空であることを明示する |
| `Elapsed` の置き場 | `internal/ui`。s06 の `elapsed` は残す | ブリーフの既定 |
| 取得の `ctx` | `context.Background()`。キャンセルしない | 止める操作が無い。`Fetch` が呼び出しごとに 30 秒の期限を持つ |
| `main_test.go` | 削除 | テスト対象の `model` 型が無くなる。同じ Scenario は `internal/ui` が持つ |
