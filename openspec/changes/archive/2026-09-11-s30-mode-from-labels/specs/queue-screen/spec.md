## MODIFIED Requirements

### Requirement: 取得が成功して Card が 0 件のときは表の領域にヒントを出す
キュー画面は、直近の取得が成功して `Cards` が 0 件（全タブが空）のとき、表の領域の代わりに次の 2 行を MUST 出す（mvp.md「初回起動（onboarding）」の文言。Markdown のバッククォートは付けない）。2 方式のどちらの利用者も直せるように、両方の入口を示す。方式の判定材料はリポジトリのラベル一覧なので、2 行目はラベルを作る手立てを案内する。
```
stage:* / To Do ラベルの無いリポジトリは何も出ません。
issue-driven-sdd の routines-setup を回すか、issue-label-driven の To Do ラベルを作ってください
```
2 行は表の領域（表の高さ）の縦中央に置き、各行を端末幅の横中央に置く（左に空白を置き、右には足さない）。端末幅より長い行は幅で切る。プレビューの領域は変えない。
次のときは出さない。
- 取得中（`fetching` が true。初回のスピナーを含む）: `Cards` が空でも表は 0 行のまま（「取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する」の初回取得前の表示を変えない）
- 取得失敗（フッタに error の文字列を出しているとき）: 前回結果の維持と誤解させないため。初回取得の失敗で `Cards` が空でも出さない
- 部分失敗（`Result.Errors` が 1 件以上）で `Cards` が 0 件: search は成功しているので出す（フッタの `詳細取得の失敗 …` はそのまま）
ヒントはキュー画面だけに出す。他の画面（s09 の詳細）は変えない。

#### Scenario: 取得成功で 0 件ならヒントが出る
- **WHEN** `Cards` が空で `Errors` も空の `Result` を取得完了として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `stage:* / To Do ラベルの無いリポジトリは何も出ません。` と `issue-driven-sdd の routines-setup を回すか、issue-label-driven の To Do ラベルを作ってください` の 2 行が含まれ、ヘッダの `[1]今やる 0` とフッタは従来どおり出る

#### Scenario: 取得中はヒントを出さない
- **WHEN** `New` 直後（初回取得前）の `Model` の `View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含まず、フッタに `取得中` が含まれる

#### Scenario: 取得失敗ではヒントを出さない
- **WHEN** `New` 直後に error `search issues: gh search issues: exit 1: rate limited` を取得失敗として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含まず、フッタに `rate limited` が含まれる

#### Scenario: Card があればヒントを出さない
- **WHEN** `example` の `Result` を取得完了として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含まず、表に `PR131` の行がある

#### Scenario: ヒントは横中央に置かれる
- **WHEN** `Cards` が空の `Result` を渡した `Model` に幅 120・高さ 40 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含む行の左側の空白の数は、（120 − その行の表示幅）÷ 2 の切り捨てに等しい

#### Scenario: 幅 60 でもヒントは表の領域に出てプレビューも描かれる
- **WHEN** `Cards` が空の `Result` を渡した `Model` に幅 60・高さ 40 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含む行と、プレビュー領域の `（このタブにはカードがありません）` の両方が含まれる

## REMOVED Requirements

### Requirement: リポジトリごとの運用方式を設定から受け取る

**Reason**: 方式の正本が設定ファイルからリポジトリのラベル一覧に変わり、`Options.Modes` という受け取り口そのものが無くなる。この Requirement は「`cmd/loop-cli` が `Config.Repos` から表を作って渡す」「表に無いリポジトリはゼロ値の `sdd`」を定めており、前者は経路ごと消え、後者は「まだ判定できていない」という新しい状態と区別できないので、Requirement 名から書き直す。

**Migration**: 表示（カード詳細の段階行・バッジ・PR 一覧）が方式の表を引くこと、`Cards` の中の値やスナップショットから方式を決めないことは、ADDED の Requirement「運用方式は取得のたびに判定した結果を持つ」がそのまま引き継ぐ。設定ファイルからの移行は `config-loading` の REMOVED が定める（`repos[].mode` を削除する）。利用者の設定に `mode` が残っていると起動しないので、振る舞いの移行は「設定ファイルから 1 行消す」だけである。

### Requirement: 方式の取り違えらしき状態をフッタで知らせる

**Reason**: この知らせは `mode: label` の書き忘れを検知するためのもので、方式をラベル一覧から判定する以上、書き忘れという状態が存在しない。`sdd` と判定したリポジトリ（`stage:todo` がある）に `To Do` の issue があるのは、2 つのラベルを併用している状態にすぎず、判定はラベル一覧のほうを見て正しく `sdd` に落ちている。人が直すものが残らないので、知らせる意味も無い。

**Migration**: 振る舞いの移行は要らない。フッタのステータスからこの 1 種類が減るだけで、他のステータス（取得中・取得失敗・書き込み・部分失敗）の優先順位は変わらない。書き忘れが招いていた「`t` が間違ったラベルを書く」は、ADDED の Requirement「運用方式は取得のたびに判定した結果を持つ」と `todo-toggle`「t は画面の Card の Issue に対して、判定した方式で確認なしに承認ラベルを切り替える」が防ぐ。

## ADDED Requirements

### Requirement: 運用方式は取得のたびに判定した結果を持つ
`internal/ui` の `Model` は「リポジトリ名 → 運用方式」の表を状態として MUST 持つ。表の中身は `internal/fetch` の `Result.Modes`（`card-fetch`「Fetch はリポジトリごとにラベル一覧を取り運用方式を判定する」）で、取得が成功したメッセージを受け取るたびに丸ごと差し替える。取得が失敗したメッセージでは触らず、前回の表を残す（D-002「失敗時は前回結果を維持する」）。`Options` は方式を受け取らず、`cmd/loop-cli` は方式を作らない。

表は「そのリポジトリの方式が分かっているか」を区別できる形で持つ。次の 3 つはいずれも「分からない」であり、表に入らない。

- 初回の取得がまだ終わっていない（スナップショットだけを描いている）
- そのリポジトリの `ListLabels` が失敗した
- そのリポジトリが `stage:todo` も `To Do` も持たない

表示（カード詳細の段階行・バッジ・PR 一覧の段階）は表を引き、分からないリポジトリはゼロ値の `sdd` の語彙で描く。分類も同じで、`Card.Result` は必ず 1 つ決まる（`card-fetch`）。書き込み（`t`）だけは分からないリポジトリで止める（`todo-toggle`「t は画面の Card の Issue に対して、判定した方式で確認なしに承認ラベルを切り替える」）。`Cards` の中の値や前回のスナップショットから方式を決めない。スナップショットは前回の実行時の派生データなので、そこから方式を決めると、ラベルを変えた後も古い方式でラベルを書く経路ができる。

#### Scenario: 取得の結果で方式の表が入れ替わる
- **WHEN** `org/board` を `label` と判定した `Result` を取得完了として渡した `Model` に、次の取得で `org/board` を `sdd` と判定した `Result` を渡し、`org/board` の `Labels` が `stage:propose` の issue の Card の詳細を開いて `View` を読む
- **THEN** `段階: stage:propose` が含まれる（前の取得の `label` は残らない）

#### Scenario: 取得が失敗したら前回の表を残す
- **WHEN** `org/board` を `label` と判定した `Result` を渡した `Model` に、取得失敗のメッセージを渡し、`org/board` の `Labels` が `In Progress` の issue の Card の詳細を開いて `View` を読む
- **THEN** `段階: In Progress` が含まれる

#### Scenario: 初回取得の前は方式が分からない
- **WHEN** `Options.Snapshot` に `org/board` の `In Progress` の issue の Card を持たせた `Model` を初回取得の前に描き、その Card の詳細を開いて `View` を読む
- **THEN** `段階なし` が含まれる（方式が分からないので `sdd` の語彙で描く）
