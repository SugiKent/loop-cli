## MODIFIED Requirements

### Requirement: 取得成功後に増えた今やるカードを 1 件 1 通知で出す
`internal/ui` は通知関数の型 `Notifier`（`func(title, body string) error`）を MUST 公開し、`Model` は `New` の `Options.Notify` で受け取る。`cmd/loop-cli` は `Config.Notify` が true のとき `beeep.Notify(title, body, "")`（s06 `notify test` と同じ呼び方。icon は空文字列。`nil` を渡すと beeep v0.11.2 はエラーになる）を呼ぶ関数を渡し、false のとき `nil` を渡す。`nil` なら通知しない。
`Model` は取得完了（`fetchedMsg`）が成功のとき、`Cards` を差し替える前の `Cards` を `prev`、`Result.Cards` を `next` として `addedNow` を呼び、MUST 次のとおり通知する（implementation-tasks.md「更新前後で「今やる」を差分比較し、増えたカードを 1 件 1 通知」。mvp.md `notify`「人の出番が新しく増えたらデスクトップ通知」）。
- 最終更新時刻がゼロ値のとき（スナップショット無しで起動した直後の初回取得で、比較する前回が無い）は `Model` は通知を出さない。スナップショットがあれば最終更新時刻はその `At` で、スナップショットの `Cards` と比較して通知する（design.md 未決事項の既定値。前回の起動から増えた分を知らせる）
- 増えた Card 1 枚につき `Notifier` を 1 回、`addedNow` の並び順で呼ぶ。タイトルは `loop-cli`、本文は 1 行目に `Card.Result.Summary`、2 行目に主体の表示名（s10 と同じ `<Repo> PR#<n>` / `<Repo> #<n>`）を改行で結んだ 2 行
- `Notifier` の呼び出しはコマンド（別ゴルーチン）で行い、`Model` は `Update` の中で `Notifier` を呼ばない（beeep は OS の通知が終わるまで待つことがある）。1 回の取得完了で増えた分は 1 つのコマンドで順に呼ぶ。コマンドは `Model` に戻すメッセージを持たず `nil` を返す
- `Notifier` のエラーは無視し、ステータスに出さず、残りの通知も続ける（design.md 未決事項の既定値）
- 取得の失敗（error）と `Notifier` が `nil` のときは `addedNow` を呼ばない。部分失敗（`Result.Errors` が 1 件以上）は成功として扱い通知する
通知のクリックで該当カードを開く（P3）は s19 が担当し、ここでは定義しない。実際の beeep の呼び出しはテストせず、手動確認だけにする（validation-plan.md「デスクトップ通知の差分比較の検証手段」は未定であり、この change は `Notifier` の差し替えで差分比較を検証し、OS の通知は `notify test` と手動で見る）。

#### Scenario: 増えたカードが 1 件 1 通知になる
- **WHEN** 呼び出しの `title` / `body` を順に記録するスタブ `Notifier` で `New` し、issue 140 の Card（バックログ）だけの `Result` を完了時刻 `12:00` の取得完了として渡し（この時点で記録は空）、次に `example` の `Result`（今やるに issue 108 + PR 131 の Card、`Summary` は s05 の局面 A の要約）を `12:04` の取得完了として渡し、返ったコマンドを実行する
- **THEN** 記録は 1 件で、`title` は `loop-cli`、`body` は 1 行目が今やるタブの行の `Summary` と同じ文字列、2 行目が `org/app PR#131` である

#### Scenario: 2 件増えれば 2 回呼ぶ
- **WHEN** 同じスタブで、`Cards` が空の `Result` を取得完了として渡してから、今やるタブの Card 2 枚（`org/app PR#131` と `org/web PR#88`。`Priority` はどちらも 1）とバックログの Card 1 枚を持つ `Result` を渡し、コマンドを実行する
- **THEN** 記録は 2 件で、`body` の 2 行目はそれぞれ今やるタブの行の並び順の表示名であり、バックログの Card の表示名は無い

#### Scenario: 初回取得では通知しない
- **WHEN** スナップショット無しでスタブ `Notifier` を渡して `New` した `Model` に `example` の `Result` を取得完了として渡す
- **THEN** コマンドは返らず（または実行しても記録は空で）、記録は空である

#### Scenario: スナップショットからの起動では比較して通知する
- **WHEN** issue 140 の Card だけを持つスナップショット（`At` `11:50`）とスタブ `Notifier` で `New` した `Model` に `example` の `Result` を取得完了として渡し、コマンドを実行する
- **THEN** 記録は 1 件で `body` の 2 行目は `org/app PR#131` である

#### Scenario: 減っただけ・同じなら通知しない
- **WHEN** `example` の `Result` を 2 回続けて取得完了として渡し、次に `Cards` が空の `Result` を渡す
- **THEN** 2 回目以降どの取得完了でも記録は増えない

#### Scenario: Notifier が nil なら何もしない
- **WHEN** `Options.Notify` を渡さずに `New` した `Model` に、issue 140 だけの `Result`、次に `example` の `Result` を取得完了として渡す
- **THEN** どちらもコマンドは返らず、`Cards` は `example` のものになっている

#### Scenario: 通知の失敗は無視して続ける
- **WHEN** 1 回目の呼び出しで error `notification failed` を返し 2 回目以降は nil を返すスタブ `Notifier` で、Scenario「2 件増えれば 2 回呼ぶ」の手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** 記録は 2 件で、フッタに `notification failed` は無い

#### Scenario: 取得の失敗では比較しない
- **WHEN** `example` の `Result` を渡した後に error `search issues: gh search issues: exit 1: rate limited` を取得失敗として渡す
- **THEN** 記録は増えず、`Cards` は `example` のままである
