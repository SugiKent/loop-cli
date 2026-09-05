## Context

s12 までの `internal/ui` は、`New(fetcher, client, editor)` で `fetching` を立て、`Init` が `startFetch()`（`fetching` true、`errText` / `partial` / 書き込みステータスを空にし、スピナーの tick と `fetchCmd` を束ねる）を呼ぶ。2 回目以降の取得は `R` だけが起こす。`fetchedMsg` の成功で `cards` / `rows` / `at` を差し替え、失敗なら前回結果を維持する。詳細画面は `detail.card` に Card のコピーを持ち、取得完了でも差し替えない（s09）。書き込み中は `posting` フラグ 1 つで `a` / `t` を止める（s10 / s11）。`cmd/sugi-loop/main.go` の `run()` は s08a の `ensureConfig` → `Load` → `NewClient` → `Check` → `ui.New` → プログラム実行の直列で、`Fetcher` は `fetch.Fetch` を `client` と `repos` で閉じた無名関数である。
`internal/model` の `Card` は `Issue *Issue` / `PRs []PR` / `Result` からなり、中身は文字列・整数・真偽値・`time.Time`・スライス・`*gh.PRMergeState` だけで、関数や interface 値を持たない（JSON で往復できる）。s06 が `github.com/gen2brain/beeep` v0.11.2 を `cmd/sugi-loop-cli` の `notify test` で使い、icon に `nil` を渡すとエラーになるので空文字列 `""` を渡している（tmp/spec-progress.md の申し送り）。
正本は D-002（永続 DB 無し。`~/.cache/sugi-loop/snapshot.json`。起動直後は stale 表示 → 背景で再取得。手動 `R`、自動更新既定 120 秒。取得中はスピナー、失敗時は前回結果を維持してエラー赤）、mvp.md の設定 `refresh_interval_sec` / `notify`（人の出番が新しく増えたらデスクトップ通知）、implementation-tasks.md の「更新前後で「今やる」を差分比較し、増えたカードを 1 件 1 通知。beeep」である。スナップショットの中身と形式、自動更新中の画面の扱い、通知の本文と通知しない条件、通知失敗の扱いは docs に無い。validation-plan.md は「デスクトップ通知の差分比較の検証手段」を未定としている。

## Goals / Non-Goals

**Goals:**
- 起動直後に前回の Card が見え、背景の取得で最新になる（起動のたびに空の画面で待たない）
- `refresh_interval_sec` ごとに取得が走り、`R` を押さなくてもキューが追従する。取得中・書き込み中には重ねない
- 「今やる」に増えた Card を 1 件 1 通知で知らせ、画面を見ていなくても人の出番に気付ける
- スナップショットの往復・差分の純粋関数・tick のスキップ条件・通知の回数と本文を、`gh` も OS の通知も動かさずにテストできる

**Non-Goals:**
- 書き込み後の対象 1 件だけの再取得（D-002。s18）、レートリミット表示（s18）
- 通知クリックで該当カードを開く（P3。s19）
- スナップショットの世代管理・有効期限・サイズ制限。派生データなので次の成功で上書きするだけ
- stale 表示中であることをスピナー以外で示す表示（docs に無い）
- 減った Card や変わった Card の通知。docs は「増えたカード」だけ

## Decisions

### ファイル構成

```
internal/snapshot/snapshot.go        # Snapshot / DefaultPath / Load / Save
internal/snapshot/snapshot_test.go   # 往復、nil と長さ 0、無い・壊れたファイル
internal/ui/refresh.go               # 自動更新: refreshTickMsg、tickCmd(interval)、tick の処理
internal/ui/refresh_test.go          # tick で取得が始まる / 取得中・書き込み中はスキップ / 詳細を保持
internal/ui/notify.go                # Notifier、addedNow、notifyCmd、本文の組み立て
internal/ui/notify_test.go           # addedNow の表、通知の回数と本文、通知しない条件
internal/ui/model.go                 # Options、New の引数、Init の束、fetchedMsg 成功時の通知（変更）
cmd/sugi-loop/main.go                # snapshot の読み込み、savingFetcher、Options の配線（変更）
cmd/sugi-loop/main_test.go           # savingFetcher の検証を足す（s08a が作るファイルに追記）
```

### スナップショットは `internal/snapshot` に置き、`cmd/sugi-loop` が読み書きする

D-003 の内部構成に snapshot の package は無いが、`internal/ui` に置くとテストがファイルシステムを持ち、`internal/fetch` に置くと取得と保存が結びつく。s07 が `internal/fetch` を、s08a が `internal/onboarding` を独立した package にした前例に倣い、この change も `internal/snapshot` を独立した package にする。依存は `internal/model`（と `model` 経由の `internal/gh` の型）だけで、`internal/ui` と `cmd/sugi-loop` から使う。
保存は `cmd/sugi-loop` が `Fetcher` を包む `savingFetcher` で行う。`Model` の `fetchedMsg` の処理で保存する案は、`Model` がパスを持ち `Update` の中で `os.WriteFile` を呼ぶことになり、`internal/ui` のテスト全部がファイルを書く。包みなら `internal/ui` は変えず、`cmd/sugi-loop/main_test.go`（s08a）で固定の `Result` を返す `Fetcher` と一時ディレクトリで検証できる。`At` は包みの中の `time.Now()` で、`fetchCmd` の完了時刻より数ミリ秒早いが、ヘッダの分表示に影響しない。
読み込みは `run()` が `Check` の後に 1 回行い、`Options.Snapshot` に渡す。`Load` は「無い」も「壊れている」も同じエラーで、`run()` はエラーなら渡さないだけにする。壊れたファイルは消さない（次の成功で上書きされる。消す処理を書く分だけコードが増え、消さなくても害が無い）。
`DefaultPath` は `os.UserHomeDir()` + `.cache/sugi-loop/snapshot.json`。`os.UserCacheDir()` は macOS で `~/Library/Caches` を返し、D-002 のパスと一致しない。s02 の `config.DefaultPath` と同じ決め方。
JSON は `encoding/json` の既定（フィールド名のまま）で書き、`json` タグを `internal/model` に足さない。`nil` スライスは `null`、長さ 0 は `[]` になり、往復で区別が残る（s09 の `未取得` / `なし` の表示が保たれる）。テストの `At` は UTC で与える（JSON から戻した `time.Time` は名前の無い固定オフセットの `Location` になり、`FixedZone("JST", …)` の値と `reflect.DeepEqual` で一致しない）。

### `New` は `Options` 構造体で拡張する

s10 が `New(fetcher, client, editor)` に引数を足した前例に倣うと、この change で 3 つ（スナップショット・間隔・通知）足して 6 引数になり、テストの `New` 呼び出しが読みにくい。`Options{Snapshot, RefreshInterval, Notify}` の 1 引数にまとめ、ゼロ値を「従来どおり」にする。既存テストの `New(f, c, e)` は `New(f, c, e, Options{})` に直すだけで振る舞いが変わらない。`Options` は起動時の値だけを持ち、実行中に変えない。フィールドの追加を見越した設計はしない。

### 自動更新は tick メッセージで `startFetch()` を呼ぶ

s12 が取得開始の経路を `startFetch()` に集めたので、tick の処理は「`fetching` でも `posting` でもなければ `startFetch()`」の 1 分岐で済む。tick は Bubble Tea の遅延コマンドで作り、tick の処理が必ず次の tick のコマンドを返すことで周期を保つ（スキップしても止まらない）。`Model` は tick の時刻を持たず壁時計も読まない（s08 の「壁時計を読まない」を保つ）。
取得中の tick をスキップするのは `R` と同じ理由（多重発行しない。30 req/分の search 枠）。書き込み中をスキップするのは、`startFetch()` が書き込みステータス（`投稿中` / `切り替え中`）を消すので、進行中の表示が取得中に置き換わり、結果のメッセージが届いても `posting` が false に戻るだけで何が起きたか読めなくなるため。次の tick（120 秒後）まで待てば書き込みは 30 秒のタイムアウトで終わっている。
tick はどの画面でも同じに処理する。詳細画面は `detail.card` のコピーを持ち（s09）、確認画面は `answer` に対象と下書きを持つ（s10）ので、`cards` が差し替わっても開いている画面は壊れない。詳細の対象を新しい Card に追従させる案は、Card が消えた（局面が変わってタブを移った・close された）ときの扱いを決める必要があり、s09 が「`Esc` で戻って開き直す」と決めているので、ここでは追従しない。
`R` で手動取得しても tick の周期はそのまま（前の tick から間隔後）。`R` の直後に tick が来れば `fetching` でスキップされ、`R` が周期をずらす処理を持たない。
間隔 0 で自動更新無しにするのは、`internal/ui` のテストが `Options{}` で tick を持たない `Model` を作るため。本番は `config.Load` が `refresh_interval_sec` を 1 以上に検証するので 0 にならない。

### 通知は `fetchedMsg` の成功時に差分を取り、コマンドで呼ぶ

差分は `addedNow(prev, next)` の純粋関数にし、キーは s08 `Subject` の（リポジトリ、番号、主体が PR か）。主体の種別を含めるのは、同じ Card（issue + PR 群）でも主体が変われば「人がすべきこと」が変わっている（PR への回答 → issue の方針決め）ため。`Result.Summary` や `Situation` をキーにすると、同じ局面で `Summary` の数字（未確定の判断の件数）が変わるたびに通知が出る。`UpdatedAt` はコメントが付くたびに変わるのでキーに入れない。
比較する「前回」は `fetchedMsg` を処理する直前の `m.cards`（スナップショットからの起動ならスナップショットの `Cards`）。「前回が無い」は `m.at.IsZero()` で判定する。スナップショットがあれば `at` は保存時刻なので比較に入り、無ければ初回取得の完了まで `at` はゼロ値で通知しない。フラグを 1 つ足さずに済む。スナップショットが 0 件（前回の取得で今やるが空だった）なら今やるの全部が「増えた」になるが、これは「前回の起動から増えた」の素直な読みで、意図どおり。
`Notifier` は `func(title, body string) error` の関数型で、`cmd/sugi-loop` が `beeep.Notify(title, body, "")` を包んで渡す。`internal/ui` は beeep を import しない（テストがスタブを渡すだけで済み、`internal/ui` のビルドが OS の通知ライブラリに依存しない）。s06 の `notifyTest` と `cmd/sugi-loop` の 2 か所で beeep を直接呼ぶが、共有のヘルパーは作らない（1 行の呼び出しを 2 か所に書くほうが短い）。
通知はコマンドで順に呼ぶ。beeep は OS の通知が完了するまで戻らないことがあり、`Update` の中で呼ぶと画面が止まる。1 回の取得完了で増えた分を 1 つのコマンドにまとめ、`Notifier` のエラーは無視して次へ進む。失敗をステータスに出さないのは、通知は画面の外への補助で、失敗しても画面に Card は出ており、ステータスは取得と書き込みの結果に使うため。
本文は `Summary` + 改行 + 表示名の 2 行。s10 / s11 の表示名（`org/app PR#131` / `org/app #140`）と揃える。s19 の通知クリックはこの本文からカードを引く前提になる。

### s08a の空キューヒントとの関係

s08a のヒントの条件は `len(cards)==0 && !fetching && errText==""` で、s08a の design はこの change が `errText` の扱いを変えるなら見直すよう申し送っている。この change は `errText` の扱いを変えない。stale 表示中は `fetching` が true なので、スナップショットが 0 件でもヒントは初回取得の完了まで出ない。条件は変えない。

### 先行 change との衝突

- `internal/ui/model.go`: s12 が `startFetch()` / `screenHelp` を、s08a が `tableLines` を変更中。この change は `New` の引数・`Options`・`Init` の束・`fetchedMsg` の成功時の通知だけを足す。`startFetch()` の名前と責務は s12 の specs / tasks のとおりであることを着手時に確認する（tasks 1.1）
- `cmd/sugi-loop/main.go`: s08a が `run()` の前半（`ensureConfig` / `checkError`）を、s10 が `ui.New` の引数を変えた。この change は `Check` の後から `ui.New` までを変える。同じ関数を触るので、後から入る方が手で合わせる（s08a と同じ扱い）
- `tui-entrypoint`「1 フレーム描画」の写し元は s08a の版。**archive の順は s08a → s12 → s13**
- `queue-screen` の 2 ブロック（`New` の署名、取得の非同期）の写し元は `openspec/specs/queue-screen/spec.md`（s10 / s08 の archive 済み版）。s11 / s12 はこの 2 ブロックを触っていない

## Risks / Trade-offs

- [スナップショットが古い（数日前）ままで stale 表示が誤解を招く] → ヘッダの `↻ HH:MM` が保存時刻で、取得中のスピナーが出る。時刻に日付を足す案は docs の `↻ 12:04` の形式に無く採らない
- [`Card` の構造が後続 change（s17 のタイムライン等）で変わり、古いスナップショットが読めない・欠ける] → `encoding/json` は未知フィールドを無視し欠けたフィールドをゼロ値にするので読める。読めなければエラーで無視し、次の成功で上書きされる
- [自動更新が search の枠（30 req/分）を使う] → 1 回 2 req を 120 秒ごとで、D-001 が「余裕がある」と評価済み。取得中はスキップし重ねない
- [自動更新のたびに書き込みステータスが消える] → s10 / s11 が「次の取得が始まったとき消える」と定めたとおり。書き込み中はスキップするので進行中の表示は消えない
- [`fetchedMsg` の処理で `addedNow` が毎回走る] → Card 数十〜数百枚のマップ比較で、120 秒に 1 回。無視できる
- [beeep が環境によって失敗する（通知の許可が無い・Linux で D-Bus が無い）] → エラーは無視し、画面は動き続ける。`notify test`（s06）で単体確認できる
- [`Notifier` のコマンドが `nil` メッセージを返す] → Bubble Tea は `nil` メッセージを捨てる。`Update` は `Notifier` の完了を知る必要が無い
- [tick のテストが実時間を待つ] → tick メッセージを直接 `Update` に渡す。`Init` の束の中身は数えず、間隔 0 のときに tick が無いことは `Init` の返り値を実行しないテストで間接的にしか見ない
- [`time.Time` の往復で `Location` が変わり `reflect.DeepEqual` が失敗する] → テストの `At` は UTC で与える。fixture の時刻は JSON から読んだものなので往復で変わらない
- [保存の失敗（ディスク満杯・権限）に気付けない] → 派生データで、次の起動が空から始まるだけ。ステータスに出す案は取得結果の表示と紛れるので採らない（未決事項）

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| スナップショットの内容 | `[]model.Card` と保存時刻の 2 つ。`Result.Errors` は入れない | D-002「派生データに限る」。表示に要るのは Card と時刻だけ |
| スナップショットの形式 | `encoding/json` の既定（フィールド名のまま、タグ無し） | `internal/model` を変えない。人が読める |
| パスの決め方 | `os.UserHomeDir()` + `.cache/sugi-loop/snapshot.json` | D-002 のパスそのまま。`UserCacheDir` は macOS で別の場所になる |
| ファイルのモード | ディレクトリ 0o700、ファイル 0o600 | s08a の設定ファイルと同じ |
| 保存のタイミング | 取得成功のたび（部分失敗を含む） | 表示した `Cards` と同じものを残す |
| 保存する場所 | `cmd/sugi-loop` の `savingFetcher`（`Fetcher` の包み） | `internal/ui` がファイルを触らない。`main_test.go` で検証できる |
| 保存の失敗 | 無視する（ステータスに出さない） | 派生データ。次の起動が空から始まるだけ |
| 壊れたスナップショット | 無視して通常起動。消さない | 次の成功で上書きされる。消す処理を書かない |
| `DefaultPath()` の失敗 | 起動失敗にせず、読み込みも保存もしない | `HOME` が無い環境は `config.DefaultPath` で先に失敗する。起動失敗の Requirement を変えない |
| stale 表示中の印 | スピナー以外に出さない。ヘッダの時刻が保存時刻 | docs に無い。スピナーと古い時刻で読み取れる |
| `New` の拡張の形 | `Options{Snapshot, RefreshInterval, Notify}` の 1 引数 | 6 引数を避ける。ゼロ値が従来どおり |
| 自動更新の周期の起点 | `Init` の時点。`R` で周期を変えない | 周期をずらす処理を持たない |
| 取得中の tick | スキップして次の tick を待つ | `R` と同じ。多重発行しない |
| 書き込み中の tick | スキップして次の tick を待つ | 進行中のステータスを取得が消さない |
| tick を受け付ける画面 | すべて | 詳細・確認はコピーを持ち、`cards` の差し替えで壊れない |
| 自動更新中の詳細の対象 | 差し替えない（s09 の規則のまま） | 消えた Card の扱いを決めなくて済む。`Esc` → `Enter` で開き直す |
| 間隔 0 | 自動更新無し | テスト用。本番は `config.Load` が 1 以上を保証 |
| 差分のキー | （リポジトリ、番号、主体が PR か） | 主体が変われば人がすべきことが変わる。`Summary` / `UpdatedAt` の変化では通知しない |
| 「前回が無い」の判定 | 最終更新時刻がゼロ値 | フラグを足さない。スナップショットがあれば比較する |
| スナップショットからの起動直後の通知 | スナップショットと比較して通知する | 前回の起動から増えた分を知らせる |
| 通知の本文 | `Summary` + 改行 + `<Repo> PR#<n>` / `<Repo> #<n>` | 「1 行要約 + repo#番号」。表示名は s10 / s11 と同じ |
| 通知のタイトル | `sugi-loop` | s06 `notify test` と同じ |
| 通知の呼び方 | コマンドで順に呼ぶ。1 回の取得完了で 1 コマンド | `Update` を止めない |
| 通知の失敗 | 無視して残りを続ける。ステータスに出さない | 補助機能。ステータスは取得と書き込みに使う |
| `Notifier` の型 | `func(title, body string) error`。`nil` で無効 | beeep を `internal/ui` に持ち込まない |
| 減った・変わった Card | 通知しない | docs は「増えたカード」だけ |
| live での確認 | 手動で、`notify: true` で起動して別の端末から `question` ラベルを付けた issue が増えたときに通知が出ること、2 分待って `↻` が進むこと、再起動で前回の表が即座に出ることを見る | validation-plan.md「通知の検証手段は未定」。この change は `Notifier` の差し替えで差分比較を検証する |
