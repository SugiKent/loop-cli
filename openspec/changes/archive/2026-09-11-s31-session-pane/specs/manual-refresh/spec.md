## MODIFIED Requirements

### Requirement: R はキュー画面で全件を再取得し、取得中は無視する
`Model` の `Update` は、画面の状態がキューのとき `R` を MUST 次のとおり扱う（mvp.md キーバインド表の `R`「全件再取得」。D-002「手動 `R`」）。
- 取得中（s08 の `fetching` が true。初回取得を含む）: 何もしない（コマンドを返さず、`Model` を変えない）。取得を多重に発行しない
- 取得中でない: 取得を開始する。取得の開始とは、`fetching` を true にし、取得の失敗と部分失敗の表示（s08 の `errText` / `partial`）を空にし、フッタ右側の書き込みステータス（s10 の回答、s11 の切り替え、s12 `browse-open` の失敗）を消し、スピナーの tick と s08 の `fetchCmd(fetcher)`（`New` で受け取った同じ `Fetcher` を使う）をまとめたコマンドを返すことである。`Init`（s08）が返す初回取得のコマンドと同じ tick + `fetchCmd` の束であり、`Init` と `New` 直後の状態は s08 のまま変えない
取得の完了（`fetchedMsg`）の扱いは s08「取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する」のとおりで、この change は変えない（成功なら `Cards` を差し替えて最終更新時刻を更新し、失敗なら前回結果を維持してエラーを赤で出す）。`R` で再取得しても、選択行の添字は s08 の丸め規則（行数を超えていれば末尾）に従うだけで、先頭に戻さない。キュー画面の `R` は `claude` を MUST 起動しない（s31 `session-pane`「セッションの取得は詳細画面を開いたときと詳細の R だけで起きる」）。
カード詳細画面 / PR 詳細画面では、`R` はその画面に紐づく Claude Code セッションの取得を始める（s31 `session-pane` が条件と結果を定める）。GitHub の内容（`Cards`・最終更新時刻・詳細の対象）は MUST 取り直さない（詳細は開いた時点のコピーであり、反映は `Esc` でキューに戻ってから `R` を押す。s31 design.md）。確認画面 / ヘルプ画面 / URL 一覧画面 / ラベル一覧画面では `R` は何もしない（design.md 未決事項の既定値）。自動更新は s13 が担当し、キュー画面の経路を再利用する。書き込み後の対象 1 件だけの再取得は s18 が担当する。

#### Scenario: R で Fetcher がもう一度呼ばれる
- **WHEN** 呼ばれた回数を数える `Fetcher` で `New` した `Model` に `example` の `Result` を取得完了として渡し（この時点で `Fetcher` の呼び出し回数は 0。`Init` のコマンドは実行していない）、`R` を与え、返ったコマンドを実行し、得たメッセージが複数のコマンドの束ならその各コマンドも実行して、`fetchedMsg` を `Update` に渡す
- **THEN** `R` の直後にコマンドが返り `fetching` が true である。実行後に `Fetcher` はちょうど 1 回呼ばれ、`Update` 後の `Cards` はその `Result.Cards` で `fetching` は false である

#### Scenario: 取得中の R は無視される
- **WHEN** 上の手順で `R` を与えた直後（返ったコマンドを実行する前）にもう一度 `R` を与え、別に `New` 直後（初回取得中）の `Model` に `R` を与える
- **THEN** どちらもコマンドは返らず、`Fetcher` の呼び出し回数は増えない

#### Scenario: R で前回のエラーとステータスが消え、スピナーが出る
- **WHEN** `example` の `Result` を取得完了として渡した後に error `search issues: gh search issues: exit 1: rate limited` を取得失敗として渡し、さらに s11 の手順でバックログの issue 140 に `t` を与えて結果のメッセージまで `Update` に渡した（フッタに `org/app #140 に stage:todo を付けました`）`Model` に `R` を与え、`View` から ANSI エスケープを除いて読む
- **THEN** フッタに `取得中` が含まれ、`rate limited` と `付けました` は含まれない。今やるタブに issue 108 の Card の行とヘッダの `↻ 12:04` は残っている（前回結果は取得の完了まで維持する）

#### Scenario: 再取得の失敗は前回結果を残す
- **WHEN** `example` の `Result` を渡し、s11 の手順でバックログの issue 140 に `t` を与えて結果のメッセージまで `Update` に渡した（フッタに `org/app #140 に stage:todo を付けました`）`Model` に `R` を与え、返ったコマンドを実行せずに error `search issues: gh search issues: exit 1: rate limited` を運ぶ `fetchedMsg` を `Update` に渡し、`View` から ANSI エスケープを除いて読む
- **THEN** 今やるタブに issue 108 の Card の行が残り、ヘッダの `↻ 12:04` は変わらず、フッタに `rate limited` が含まれ、`付けました` と `取得中` は含まれない

#### Scenario: 選択行は再取得後も維持される
- **WHEN** 今やるタブに 3 行ある `Model` で選択行を 2 に動かしてから `R` を与え、同じ 3 枚の Card の `Result` を `fetchedMsg` で渡す
- **THEN** 選択行の添字は 2 のままである

#### Scenario: 詳細画面の R は GitHub を取り直さない
- **WHEN** 呼ばれた回数を数える `Fetcher` と、呼ばれた回数を数える `claude` のスタブで `New` し `example` の `Result` を渡した `Model`（幅 120。`org/app` に `claude_config_dir` を設定する）について、issue 108 のカード詳細を開いて開いたときの取得を終えてから `R` を与え、返ったコマンドを実行する
- **THEN** `Fetcher` の呼び出し回数は 0 のままで、`fetching` は false、`Cards` と最終更新時刻は変わらず、画面はカード詳細のままである。`claude` のスタブの呼び出し回数は 2 回になっている

#### Scenario: 詳細画面・確認画面・ヘルプ画面で R は何もしない
- **WHEN** 呼ばれた回数を数える `Fetcher` と `claude` のスタブで `New` し `example` の `Result` を渡した `Model`（幅 80。この幅では右ペインを出さないので、詳細を開いてもセッションの取得は起きない）について、issue 108 のカード詳細を開いて `R` を与え、そこから `Enter` で PR 詳細を開いて `R` を与え、別に s10 の手順で blocked-by の確認画面に移って `R` を与え、別に `?` でヘルプ画面を開いて `R` を与える
- **THEN** どれもコマンドは返らず、画面は変わらず、`fetching` は false のままで、`Fetcher` と `claude` のスタブはどちらも 1 度も呼ばれない
