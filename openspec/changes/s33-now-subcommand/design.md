## Context

動機は proposal.md の「Why」を見る。

現状の `cmd/loop-cli/main.go` は `run` が第 1 引数で振り分け、`version` / `update` は
設定ファイルも `gh` も使わない軽い経路、引数なしは `runTUI` の重い経路という 2 極になっている。
`now` は「設定と `gh` は使うが Bubble Tea は起動しない」という 3 番目の経路になる。

`runTUI` が組み立てている依存のうち、`now` に要るのは設定の読み込み・`gh.Client.Check`・`fetch.Fetch` だけである。
エディタ・通知・スナップショット・更新確認・`claude` のセッション取得は要らない。

## Goals / Non-Goals

**Goals:**

- `fetch.Fetch` の結果を「今やる」だけに絞って JSON にする層を、`cmd/loop-cli` の中に薄く作る
- 出力の形（キー名と型）を単体テストで固定し、agent から見た契約が実装の都合で動かないようにする

**Non-Goals:**

- 分類・並び順・局面の定義を変えること。`internal/classify` と `internal/model` と `internal/ui` は読むだけで触らない
- TUI 側に新しい表示を足すこと
- `now` から issue / PR に書き込むこと（回答・ラベル・merge は TUI の担当のまま）

## Decisions

### D1. 出力の組み立ては `cmd/loop-cli` に置き、`internal` に新しいパッケージを作らない

`now` の出力は `model.Card` から JSON 用の構造体に写すだけで、他から使う予定は無い。
実装は `cmd/loop-cli/now.go` に出力の組み立てと `run` から呼ぶ関数を置き、`cmd/loop-cli/now_test.go` に
そのテストを置けば足りる。
`internal/nowjson` のようなパッケージを作ると、1 箇所からしか呼ばれない公開 API を増やすことになる
（CLAUDE.md「一度しか使う予定のない処理のために抽象化を増やさない」）。

代案として `internal/ui` の `buildRows` を公開して使い回すことも考えたが、`buildRows` は 4 タブ分の
`row`（本文・コメント・カードを抱えた画面用の型）を返すので、JSON に要らない依存を持ち込む。
`now` 側で `Card.Result.Tab == model.TabNow` に絞り、同じキーで `sort.SliceStable` する方が短い。
並び順の規則は `now-command` の Requirement に書いてあるので、両者がずれればテストで落ちる。

### D2. 主体は `ui.Subject` で引く

どの issue / PR がカードの分類結果を出したかは `internal/ui` の `Subject`（既に公開されている）が返す。
`now` はこれを呼んで `subject` を作る。同じ判定を書き写すと、TUI と CLI で主体が食い違う余地が生まれる。

### D3. `errors` と `items` は必ず配列として出す

Go の `nil` スライスは `null` になる。agent 側で `null` と `[]` の分岐を書かせないため、
JSON へ写す時点で長さ 0 のスライスを割り当てる。テストで `"items":[]` と `"errors":[]` を確認する。

### D4. 時刻は RFC 3339、`fetched_at` は取得を始めた時刻

`updated_at` は GitHub が返した値をそのまま `time.Time` として出す。`fetched_at` は `time.Now()` を
1 回だけ取り、`fetch.Fetch` の `now` 引数にも同じ値を渡す（`classify` の時間切れ判定と出力の時刻をそろえる）。
経過時間（`12m` / `3h` / `2d`）は出さない。agent は `fetched_at` と `updated_at` の差を自分で計算できる。

### D5. 部分失敗は文字列の配列にする

`fetch.Result.Errors` は `error` の配列で、構造化された情報を持たない（`ViewPR org/app#131: ...` の形）。
JSON でも `err.Error()` の文字列をそのまま並べる。`gh` の失敗を型として設計し直すのは、この change の範囲を超える。

### D6. `now` は 30 秒などの追加の時間制限を持たない

`fetch.Fetch` は 1 回の `gh` 呼び出しごとに 30 秒（`fetch.CallTimeout`）を掛け、詳細取得は 4 並行で走る。
`now` 全体の制限は掛けず、agent 側のタイムアウトに任せる。TUI と同じ取得の重さになる。

## Risks / Trade-offs

- **[人が `loop-cli now` を打つと JSON が流れる]** → 未確定の判断 Q1 で人に選んでもらう。
  推奨案（JSON だけ）を採る場合、README に「人が読む画面は `loop-cli`、agent が読むのは `loop-cli now`」と書き分ける
- **[出力のキー名が後から変わると agent 側が壊れる]** → 単体テストで JSON のキーを固定する。
  変えるときは spec の Requirement を MODIFIED することになるので、気付かずに変わることはない
- **[agent が短い間隔で叩くと `gh` のレート制限に当たる]** → `now` は 1 回の実行で TUI の 1 回の取得と同じだけ
  `gh` を呼ぶ。呼ぶ間隔は agent 側の判断に任せ、CLI ではキャッシュも制限も持たない（スナップショットを
  読まないのは proposal.md の「確定した判断」4 のとおり）
- **[「今やる」が空のときに agent が失敗と読む]** → 終了コード 0 と `count: 0` を Requirement とテストで固定する
