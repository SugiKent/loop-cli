# card-detail Specification

## Purpose
TBD - created by archiving change s09-card-detail. Update Purpose after archive.
## Requirements
### Requirement: Enter でカード詳細を開き、Esc で 1 つ前の画面に戻る
`internal/ui` の `Model` は画面の状態として キュー / カード詳細 / PR 詳細 / 確認 / ヘルプ の 5 つを MUST 持ち、初期状態はキューである。確認画面は s10 `answer-question`「確認画面では投稿・編集に戻る・中止を選ぶ」が、ヘルプ画面は s12 `help-screen`「? はヘルプ画面を開き、? か Esc で開いた画面に戻る」が定める。`Update` はキー入力を画面の状態ごとに次のとおり扱う。
- キュー画面で `Enter`: 選択行があれば、その行の `model.Card` のコピーを詳細の対象として保持し、カード詳細画面に移る。`Card.Issue` が nil（PR 単独のカード）なら、カード詳細を挟まず `PRs[0]` の PR 詳細画面に直接移る（Issue の情報が無く、カード詳細に出すものが PR 一覧 1 件しか無いため。design.md 未決事項の既定値）。選択行が無い（タブが 0 行）なら何もしない。開いたときに routine コメントの展開状態は折りたたみに戻り、スクロール位置は先頭に戻る
- カード詳細画面で `Esc`: キュー画面に戻る。キュー画面の現在のタブと選択行は開く前のまま（戻るキーは mvp.md に無く、`Esc` は design.md 未決事項の既定値）
- PR 詳細画面で `Esc`: カード詳細画面から入ったならカード詳細に戻り、キュー画面から直接入った（`Issue` が nil）ならキュー画面に戻る
- `q` / `Ctrl+C`: どの画面でも終了コマンドを返す（s01 `tui-entrypoint`「q で終了する」）
- カード詳細画面と PR 詳細画面では、キュー画面のキー `1`〜`4`（タブ切替）と `R`（全件再取得。s12 `manual-refresh`）は何もしない。`p` はキュー画面でも詳細画面でも何もしない（s21 `queue-screen` が 2 ペインを常設にして切替を廃止した）。`j` / `k` / `↑` / `↓` はスクロール（Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」）に使う。`Tab` はカード詳細画面でだけ PR の選択（Requirement「紐づく PR 一覧は段階順に 1 行ずつ出し、選択中の PR に印を付ける」）に使い、PR 詳細画面では何もしない。詳細画面での `j` / `k` / `↑` / `↓` と `Tab` の意味は mvp.md に無く、design.md 未決事項の既定値である
- カード詳細画面と PR 詳細画面で `?`: 戻り先を保持してヘルプ画面に移る。`o` はその画面の対象をブラウザで開く（s12 `browse-open`）
- 詳細を開いている間に取得完了のメッセージ（s08 の `fetchedMsg`）が届いたら、キュー画面の `Cards` と最終更新時刻は s08 の規則どおり更新するが、詳細の対象として保持している Card のコピーは差し替えない。`Esc` でキューに戻った後の `Enter` で新しい Card を開く（自動更新中の追従は s13 が決める）

詳細を開いても `gh` を呼ばない。表示するのは s07 の `Fetch` が Card に入れた値だけである（design.md 未決事項「詳細を開いたときの追加取得」。mvp.md の `Enter` 内部処理 `gh issue view` / `gh pr view --json` と human-turn-signals.md「merge 可否は表示時に取り直す」からの逸脱で、design.md「追加取得はしない」に理由と申し送りがある）。

#### Scenario: Enter でカード詳細が開き Esc で戻る
- **WHEN** `example` の `Result`（issue 108 + PR 131 の Card が今やるタブ）を渡した `Model` に `Enter` を与え、次に `Esc` を与える
- **THEN** `Enter` の後の画面はカード詳細で、詳細の対象は issue 108 の Card である。`Esc` の後の画面はキューで、現在のタブは今やる、選択行の添字は 0 のままである

#### Scenario: PR 単独のカードは PR 詳細が直接開く
- **WHEN** `Issue` が nil で `PRs` が `docs` ラベルの open PR 60 の 1 件の Card を今やるタブに持つ `Model` に `Enter` を与え、次に `Esc` を与える
- **THEN** `Enter` の後の画面は PR 詳細で対象は PR 60、`Esc` の後の画面はキューである

#### Scenario: 0 行のタブで Enter は何もしない
- **WHEN** `example` の `Result` を渡して異常タブ（0 行）に切り替えた `Model` に `Enter` を与える
- **THEN** 画面はキューのままで、コマンドは返らない

#### Scenario: 詳細画面ではタブ切替キーと R が効かない
- **WHEN** カード詳細を開いた `Model` に `2`、`4`、`p`、`R` を 1 つずつ与える
- **THEN** 画面はカード詳細のままで、キュー画面の現在のタブは今やるのままで、どのキーでもコマンドは返らない

#### Scenario: 詳細を開いている間の取得完了は対象の Card を差し替えない
- **WHEN** `example` の `Result` で issue 108 のカード詳細を開いた `Model` に、issue 108 の Card を含まない `Result` を取得完了として渡す
- **THEN** 画面はカード詳細のままで、詳細の対象は issue 108 の Card のままである。`Esc` でキューに戻ると今やるタブの行は新しい `Result` のものになっている

#### Scenario: どの画面でも q で終了する
- **WHEN** カード詳細を開いた `Model` に `q` を与える
- **THEN** 終了コマンドが返る

### Requirement: ヘッダはリポジトリ・番号・いま人が何をすべきか・現在の段階・バッジ・depends on を出す
カード詳細画面の `View` はヘッダ領域の先頭に、詳細の対象の Card について MUST 次の行をこの順で出す（mvp.md「カード詳細」のヘッダ）。行番号は固定せず順序だけを定める（s17 が ADDED で行を足せるようにするため）。
- `<Repo> #<Number>  <Title>`（`Card.Issue` の値）
- `Card.Result.Summary`（空なら出さない。s05 が定めた「いま人が何をすべきか」。mvp.md「カードは『いま人が何をすべきか』を 1 行目に出す」。s08 のプレビューはこれを出さず、この change のヘッダに委ねている）
- `段階: ` に続けて `model.IssueStages(Issue.Labels)` の段階ラベルを空白区切りで並べる。1 件も無ければ `段階なし`。2 件以上あればすべて並べる（異常の状態を隠さない）。続けてバッジを `[blocked]` / `[wip]` / `[question]` の順で、`Labels` にあるものだけ出す
- `Issue.Body` の中に `depends on #<n>`（大文字小文字を区別しない。`<n>` は 10 進整数）が 1 つ以上あれば `depends on: #<n> #<m> …` を出現順に出す。無ければこの行を出さない（書式は design.md 未決事項の既定値）

ヘッダ領域の各行（上の各行と Requirement「紐づく PR 一覧は段階順に 1 行ずつ出し、選択中の PR に印を付ける」の PR 一覧の行、PR 詳細のヘッダ領域の行）は、表示幅が端末の幅を超えれば端末の幅に切り詰めて末尾を `…` にする（design.md 未決事項の既定値）。
「この段階に入ってからの経過時間」と段階の変遷タイムラインはデータ源が REST の timeline（D-001「ラベル変遷」。`GHClient.LabelTimeline`）であり、s17 が担当する。この change はヘッダに枠を確保せず、s17 が ADDED で行を足す。

#### Scenario: issue 108 のヘッダ
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、`example` の issue 108 の Card（`Labels` が `stage:propose` と `question`、`Card.Result.Summary` が `PR #131 の質問に答える`）の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `org/app #108`、issue 108 の `Title`、`PR #131 の質問に答える`、`段階: stage:propose`、`[question]` がこの順で含まれ、`[blocked]` と `[wip]` と `depends on:` は含まれない

#### Scenario: 長いヘッダ行は幅で切り詰める
- **WHEN** 幅 40・高さ 40 のサイズメッセージを与えた後、`Title` が表示幅 60 の issue の Card の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `org/app #` で始まる行の表示幅は 40 以下で、末尾が `…` である

#### Scenario: 段階ラベルが無い issue と depends on
- **WHEN** `Labels` が空、`Body` が `集計が遅い。\n\ndepends on #12\nDepends on #34` の issue の Card の詳細を開き、`View` を読む
- **THEN** `段階なし` と `depends on: #12 #34` が含まれる

#### Scenario: 段階ラベルが 2 つある issue は両方出す
- **WHEN** `Labels` が `stage:propose` と `stage:apply` と `blocked` と `wip` の issue の Card の詳細を開き、`View` を読む
- **THEN** `段階: stage:propose stage:apply` と `[blocked]` と `[wip]` がこの順で含まれる

### Requirement: 紐づく PR 一覧は段階順に 1 行ずつ出し、選択中の PR に印を付ける
カード詳細画面の `View` はヘッダ領域のヘッダ行の下に、`Card.PRs`（s07 が段階順 → 番号順に並べたもの）を 1 件 1 行で MUST 出す。`propose` / `apply` / `archive` の各段階について、`model.PRStages(Labels)` の先頭がその段階である PR の行を並べ、その段階の PR が 1 件も無ければ `[<段階>] なし` の 1 行を出す（mvp.md「`[apply] なし`」）。段階ラベルの無い PR は 3 段階の後に `[-]` を段階名の代わりにして並べる。
各行の書式は `[<段階>] PR#<Number> <状態>` に続けて次を空白区切りで並べる。
- 状態: `State` が `OPEN` なら `open`、`MERGED` なら `merged`、`CLOSED` なら `closed`。`Canonical` が true なら状態の直後に `（最新・正本）`（s05「同段階の merge 済み PR は最新が正本」。search は open PR しか返さないので P1 では立たず、s17 の cross-reference で merge 済み PR が入って初めて付く）
- 1 行目判定: `model.ParseUndecided(Body)` が `n, true` なら `未確定 <n> 件`、false なら `1 行目なし`
- `labels: <Labels を空白区切り>`
- checks: `MergeState` が nil なら `checks 取得失敗`（s20 は全 PR の merge 状態を取るので、nil は取得に失敗したことを意味する）、non-nil で `classify.ChecksGreen(MergeState)` が true なら `checks 緑`、false なら `checks 緑以外`
- merge 状態: `MergeState` が nil なら `mergeable 取得失敗`、non-nil なら `mergeable <Mergeable>`

`Model` は詳細の中で選択中の PR の添字（初期値 0）を持ち、その PR の行の先頭に `▶` を置く。他の PR の行と `なし` の行は同じ幅の空白で始める。`Tab` で選択を次の PR（`PRs` の並び順）に移し、末尾の次は先頭に戻る。`PRs` が空なら `Tab` は何もしない。
このヘッダ領域はスクロールせず、常に表示される（Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」）。各行は端末の幅で切り詰める（Requirement「ヘッダはリポジトリ・番号・いま人が何をすべきか・現在の段階・バッジ・depends on を出す」）。

#### Scenario: issue 108 の PR 一覧
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、`example` を `Fetch` して得た issue 108 の Card（`PRs` は `propose` + `question` の open PR 131 のみ。本文 1 行目は `issue #108 の提案。`。s20 で全 PR の merge 状態を取るので `MergeState` は non-nil で `Mergeable` が `UNKNOWN`、`StatusCheckRollup` に `StatusContext ci/legacy PENDING` を含む）の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `▶` で始まり `[propose] PR#131 open`、`1 行目なし`、`labels: propose question`、`checks 緑以外`、`mergeable UNKNOWN` をこの順で含む行があり、`[apply] なし` と `[archive] なし` の行がある

#### Scenario: merge 状態の取得に失敗した PR
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、`PRs` が `[propose, OPEN, #131, MergeState nil]` の 1 件だけである手書きの Card の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `[propose] PR#131 open` の行に `checks 取得失敗` と `mergeable 取得失敗` がこの順で含まれる

#### Scenario: 同段階の merge 済み PR は最新に印が付く
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、`PRs` が `[propose, MERGED, #131]`、`[propose, MERGED, #140, Canonical true]`、`[apply, OPEN, #151, Body 1 行目 未確定の判断: 0 件, MergeState{Mergeable: MERGEABLE, StatusCheckRollup: [CheckRun test SUCCESS]}]` の順である Card の詳細を開き、`View` を読む
- **THEN** `[propose] PR#131 merged` の行、`[propose] PR#140 merged（最新・正本）` の行、`[apply] PR#151 open` と `未確定 0 件` と `checks 緑` と `mergeable MERGEABLE` を含む行、`[archive] なし` の行がこの順で含まれ、`PR#131` の行に `（最新・正本）` は無い

#### Scenario: Tab で PR の選択が移る
- **WHEN** 上の Card（PR 3 件）の詳細を開いた `Model` に `Tab` を 3 回与え、各回の後に `View` を読む
- **THEN** `▶` が付く行は順に `PR#140`、`PR#151`、`PR#131` の行である

#### Scenario: 段階ラベルの無い PR は末尾に出る
- **WHEN** `PRs` が `[apply, OPEN, #90]`、`[ラベル無し, OPEN, #61]` の Card の詳細を開き、`View` を読む
- **THEN** `[propose] なし`、`[apply] PR#90 open`、`[archive] なし`、`[-] PR#61 open` の行がこの順で含まれる

### Requirement: 本文領域は Issue 本文・最新 blocked-by の要約・コメント時系列を出す
カード詳細画面の `View` は本文領域に、詳細の対象の `Card.Issue` について MUST 次を上から順に出す。
1. `Issue.Body` を Glamour で Markdown レンダリングした文字列（幅は端末の幅。s08 のプレビューと同じ方法。レンダリングが失敗したら `Body` をそのまま出す。空なら何も出さない）
2. `model.LatestBlockedBy(Issue.Comments)` が見つかれば `blocked-by: <value>` の 1 行。`value` が `human` なら続けて `model.ParseQuestions(そのコメントの Body)` の各質問を `Q<Number>. <Title>` の行と、その選択肢を `  <Letter>: <Text>`（`Recommended` なら `  <Letter>（推奨）: <Text>`）の行で出す（mvp.md「`human` なら『人に何を決めてほしいか』の選択肢と推奨がここにある」）。質問が 1 件もパースできなければ、そのコメントの `Body` から routine マーカー行と `blocked-by:` 行を除いた本文をそのまま出す。見つからなければこの部分を出さない
3. コメント時系列: `Issue.Comments` が nil なら `コメント: 取得失敗`（s20 は全 issue のコメントを取るので、nil は取得に失敗したことを意味する）、長さ 0 なら `コメント: なし`、1 件以上なら Requirement「routine コメントは折りたたみ、x で展開する」の書式で並び順（末尾が最新）に出す

#### Scenario: issue 108 の本文とコメント
- **WHEN** `example` の issue 108 の Card の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** 本文の `認証まわりの仕様を決めたい` が含まれ、`blocked-by:` は含まれず（`example` に `blocked-by:` 行が無い）、`コメント: 取得失敗` と `コメント: なし` は含まれない

#### Scenario: blocked-by human の要約に選択肢と推奨が出る
- **WHEN** `Labels` が `stage:propose` と `blocked` と `question`、`Comments` が AI の 1 件（`Body` が `<!-- routine -->\nblocked-by: human\n## Q1. 名前での絞り込みを含めるか\n- 選択肢 A（推奨）: 含めない\n- 選択肢 B: 含める`）の issue の Card の詳細を開き、`View` を読む
- **THEN** `blocked-by: human`、`Q1. 名前での絞り込みを含めるか`、`A（推奨）: 含めない`、`B: 含める` がこの順で含まれる

#### Scenario: 見出しの無い blocked-by コメントは本文をそのまま出す
- **WHEN** 上と同じで `Comments` の `Body` が `<!-- routine -->\nblocked-by: human\n次の方針をコメントで教えてください` の Card の詳細を開き、`View` を読む
- **THEN** `blocked-by: human` の行と `次の方針をコメントで教えてください` が含まれる

#### Scenario: コメントが 0 件の issue
- **WHEN** `example` を `Fetch` して得た issue 140 の Card の詳細を開き、`View` を読む（s20 で全 issue のコメントを取るので、`issue-140.json` のコメント 0 件が長さ 0 の非 nil で入る）
- **THEN** 本文の `起動時に設定ファイルが無いと落ちる` と `コメント: なし` が含まれ、`コメント: 取得失敗` と `▌` は含まれない

#### Scenario: コメントの取得に失敗した issue
- **WHEN** `Comments` が nil の手書きの issue の Card の詳細を開き、`View` を読む
- **THEN** `コメント: 取得失敗` が含まれ、`コメント: なし` と `▌` は含まれない

### Requirement: routine コメントは折りたたみ、x で展開する
カード詳細画面と PR 詳細画面のコメント時系列は、各コメントを MUST 次の書式で出す。時刻は `CreatedAt` を最終更新時刻（s08 の `fetchedMsg.at`）と同じタイムゾーンに直した `HH:MM`。
コメント本文はまず s08 `queue-screen`「プレビューは選択行の 1 行目・本文・コメントを出す」と同じ規則で routine マーカー行を取り除く（`strings.TrimSpace` 後の行全体が `<!-- routine -->` または `&lt;!-- routine --&gt;` に一致する行だけを落とす。部分一致は触らない）。以下の「`Body`」はマーカー行を除いた後の本文を指す。
- `AI` が true のコメント（s05 `model.IsAI` が本文で判定したもの。エスケープ済みマーカーと `## PR リスク評価` 見出しを含む）:
  - 折りたたみ時（既定）: 見出し 1 行だけ。`▌AI  HH:MM  <要約>  (+<n> 行)`。`<要約>` は `Body` の生の行（Glamour を通さない）のうち空行でない最初の行。`<n>` は `Body` の行数から要約の行までを除いた残りの行数（空行を含む。要約が最終行なら `(+0 行)`）
  - 展開時: 見出し `▌AI  HH:MM` と、`Body` を s08 のプレビューと同じく Glamour で幅 = 端末の幅 − 1 でレンダリングした全行の左端に `▌` を付けたもの（レンダリングが失敗したら `Body` をそのまま出す）
- `AI` が false のコメント: 常に見出し `<Author>  HH:MM` と、`Body` を同じ幅で Glamour でレンダリングした全行を、`▌` を付けずに出す。折りたたまない

`x` はカード詳細画面と PR 詳細画面で routine コメントの展開 / 折りたたみを切り替える。1 つのフラグで画面内の全 AI コメントを同時に切り替え、コメントごとには持たない。キュー画面から `Enter` で開くたびに折りたたみに戻る。キュー画面で `x` は何もしない。
review thread 内のコメント（Requirement「PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す」）は折りたたまない。

#### Scenario: 折りたたみ時は AI コメントの見出しだけ出る
- **WHEN** `example` の issue 108 の Card（コメント 1 件目が AI で `Body` が `<!-- routine -->\nQ1: セッションの寿命は何日にしますか。\nQ2: 失効時はログイン画面へ戻しますか。`、`CreatedAt` `2026-09-04T09:00:00Z`。2 件目が `user-2` の `寿命は 30 日で。`、`2026-09-04T09:12:00Z`）の詳細を、最終更新時刻 `2026-09-05T12:04:00+09:00` で開き、`View` から ANSI エスケープを除いて読む
- **THEN** `▌AI  18:00  Q1: セッションの寿命は何日にしますか。  (+1 行)` の行があり、`Q2: 失効時は` は含まれず、`user-2  18:12` の行と `寿命は 30 日で。` の行があってどちらも `▌` で始まらない

#### Scenario: x で展開すると全行が出る
- **WHEN** 上の `Model` に `x` を与えて `View` を読み、もう一度 `x` を与えて `View` を読む
- **THEN** 1 回目は `▌AI  18:00` の行と、`▌` で始まり `Q2: 失効時はログイン画面へ戻しますか。` を含む行があり、`<!-- routine -->` と `(+1 行)` は無い。2 回目は `(+1 行)` があり `Q2: 失効時は` は無い

#### Scenario: 開き直すと折りたたみに戻る
- **WHEN** 上の `Model` に `x`（展開）、`Esc`、`Enter` の順で与えて `View` を読む
- **THEN** `(+1 行)` があり `Q2: 失効時は` は無い

#### Scenario: PR リスク評価の見出しを持つコメントも AI として折りたたまれる
- **WHEN** `Comments` が `Author` `user-2`、`Body` `PR #131 の評価です。\n\n## PR リスク評価\n\n- 影響範囲: 小` のコメント（`CommentFrom` で `AI` true）1 件の PR の詳細を開き、`View` を読む
- **THEN** `▌AI` で始まり `PR #131 の評価です。` と `(+4 行)` を含む行があり、`影響範囲: 小` は含まれない

### Requirement: PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す
PR 詳細画面の `View` は、対象の `model.PR` について MUST 次を上から順に出す。ヘッダ領域は 1〜2、本文領域は 3〜8（Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」）。
1. `<Repo> PR#<Number>  <Title>`
2. `[<段階>] <状態>  labels: <Labels を空白区切り>`（段階と状態は Requirement「紐づく PR 一覧は段階順に 1 行ずつ出し、選択中の PR に印を付ける」と同じ表記）
3. 1 行目判定: `model.ParseUndecided(Body)` が `n, true` なら `未確定の判断: <n> 件`、false なら `1 行目に未確定の判断が無い`
4. 紐づけ: s07 の `fetch.LinkedIssue(Title, Body)` が `n, true` なら `紐づく issue: #<n>`、false なら `紐づく issue: なし`（mvp.md「`Refs #n` / `Closes #n`」）
5. checks と merge 状態: `MergeState` が nil なら `checks: 取得失敗` の 1 行（s20 は全 PR の merge 状態を取るので、nil は取得に失敗したことを意味する）。non-nil なら `mergeable: <Mergeable> <MergeStateStatus>` の 1 行に続けて、`StatusCheckRollup` の各要素を 1 行ずつ（`Typename` が `CheckRun` なら `  <Name>: <Conclusion>`（`Conclusion` が空なら `<Status>`）、`StatusContext` なら `  <Context>: <State>`）。要素が 0 件なら `  checks: なし`
6. `Body` を Glamour でレンダリングした文字列（Issue 本文と同じ扱い）
7. 会話コメント: `Comments` が nil なら `コメント: 取得失敗`、長さ 0 なら `コメント: なし`、1 件以上なら Requirement「routine コメントは折りたたみ、x で展開する」の書式
8. review thread: `ReviewThreads` が nil なら `review thread: 取得失敗`（s20 は全 PR の review thread を取るので、nil は取得に失敗したことを意味する）、長さ 0 なら `review thread: なし`、1 件以上なら `IsResolved` が false の thread を先に、true の thread を後に（それぞれ元の順を保つ）並べ、各 thread を見出し `thread 未 resolve` / `thread resolved` の 1 行と、その `Comments` の各件（Requirement「routine コメントは折りたたみ、x で展開する」の展開時の書式。`model.IsAI(Body)` が true なら `▌AI  HH:MM` と `▌` 付きの全行、false なら `<Author.Login>  HH:MM` と全行。折りたたまない）で出す。thread の選択と `A` 返信は s16 が担当する

PR 詳細画面で `g` は、カードに `Issue` があればカード詳細画面に移る（`Issue` が nil なら何もしない）。カード詳細画面で `g` は、選択中の PR の PR 詳細画面に移る（`Enter` と同じ。`PRs` が空なら何もしない）。カード詳細画面で `Enter` は選択中の PR の PR 詳細画面に移る（mvp.md「PR を選んで Enter で PR の会話・review thread へ入る」。`PRs` が空なら何もしない）。

#### Scenario: PR 131 の詳細
- **WHEN** `example` の issue 108 のカード詳細で `Enter` を与え（PR 131 が選択中）、`View` から ANSI エスケープを除いて読む
- **THEN** `org/app PR#131`、`[propose] open  labels: propose question`、`1 行目に未確定の判断が無い`、`紐づく issue: #108`、`mergeable: UNKNOWN BLOCKED`、`test: SUCCESS`、`ci/legacy: PENDING`、本文の `issue #108 の提案`、`▌AI  19:31  Q1: マイグレーションを分けますか。  (+0 行)`、`thread 未 resolve` がこの順で含まれ、`checks: 取得失敗` と `review thread: 取得失敗` は含まれない（s20 で全 PR の merge 状態と review thread を取るので、`example` の PR 131 はどちらも埋まる）

#### Scenario: 未 resolve の thread が先頭に出る
- **WHEN** `Labels` が `apply`、`ReviewThreads` が `[{IsResolved: true, Comments: [{user-3, "直しました"}]}, {IsResolved: false, Comments: [{user-1, "<!-- routine -->\nこの分岐は残しますか"}]}]` の順、`MergeState` が `{Mergeable: MERGEABLE, MergeStateStatus: CLEAN, StatusCheckRollup: [CheckRun test SUCCESS, StatusContext ci/legacy PENDING]}` の open PR の詳細を開き、`View` を読む
- **THEN** `mergeable: MERGEABLE CLEAN`、`test: SUCCESS`、`ci/legacy: PENDING` がこの順で含まれ、`thread 未 resolve` の行が `thread resolved` の行より前にあり、`この分岐は残しますか` の行は `▌` で始まり、`直しました` の行は `▌` で始まらない

#### Scenario: checks が無い merge 状態
- **WHEN** `MergeState` が `{Mergeable: UNKNOWN, StatusCheckRollup: []}`、`ReviewThreads` が空（長さ 0）、`Comments` が空の PR の詳細を開き、`View` を読む
- **THEN** `mergeable: UNKNOWN`、`checks: なし`、`コメント: なし`、`review thread: なし` が含まれる

#### Scenario: 詳細の取得に失敗した PR
- **WHEN** `MergeState` と `Comments` と `ReviewThreads` がいずれも nil である手書きの open PR の詳細を開き、`View` を読む
- **THEN** `checks: 取得失敗`、`コメント: 取得失敗`、`review thread: 取得失敗` がこの順で含まれ、`checks: なし` と `コメント: なし` と `review thread: なし` は含まれない

#### Scenario: g で issue と PR を行き来する
- **WHEN** `example` の issue 108 のカード詳細を開いた `Model` に `g`、`g`、`Enter`、`Esc` の順で与える
- **THEN** 画面は順に PR 詳細（PR 131）、カード詳細、PR 詳細、カード詳細になる

#### Scenario: PR 単独のカードで g は何もしない
- **WHEN** `Issue` が nil の Card から直接開いた PR 詳細の `Model` に `g` を与える
- **THEN** 画面は PR 詳細のままで、コマンドは返らない

### Requirement: 詳細の本文領域はスクロールし、ヘッダ領域は固定する
カード詳細画面と PR 詳細画面は、端末の幅と高さ（s08 の `Model` が保持する値）の全体を使う 1 ペインで MUST 描く（キュー画面の表とプレビューの 2 ペインは詳細に適用しない）。上からヘッダ領域（カード詳細: ヘッダ行と PR 一覧。PR 詳細: 1〜2 行目）、区切り線、本文領域、フッタの順で、ヘッダ領域と区切り線とフッタは常に表示され、本文領域だけが残りの高さに収まらない分をスクロールする。
本文領域は Bubbles の viewport で描き、`j` / `↓` で 1 行下へ、`k` / `↑` で 1 行上へ、`PgDn` / `PgUp` で本文領域の高さぶん動く（スクロールキーは mvp.md に無く、design.md 未決事項の既定値）。先頭と末尾で止まる。キュー画面から開いたとき、および PR 詳細とカード詳細を行き来したとき、スクロール位置は先頭に戻る。
本文領域の高さは端末の高さからヘッダ領域の行数と区切り線 1 行とフッタ 1 行を引いた値で、下限は 1 行とする。1 行を下回るときはカード詳細の PR 一覧を末尾から落として（`なし` の行を含む）本文領域 1 行を確保する（design.md 未決事項の既定値）。
フッタの左はその画面で動く操作キーのヒント（カード詳細: `Esc 戻る  Tab PR 選択  Enter PR を開く  x 展開  g PR へ  ? ヘルプ  a 回答  t todo  o ブラウザ  q 終了`（表示幅 101 列）。`PRs` が空なら `Tab PR 選択  Enter PR を開く  g PR へ` を省く。PR 詳細: `Esc 戻る  x 展開  g issue へ  ? ヘルプ  a 回答  o ブラウザ  q 終了`（表示幅 66 列）。`? ヘルプ` を `a 回答` の前に置くのは、カード詳細で末尾に置くと幅 80 で切れてヘルプの入口が見えないため（`? ヘルプ` は 65 列目で終わる。design.md 未決事項の既定値）。`t` の振る舞いは s11 `todo-toggle` が定め、PR 詳細では `t` が何もしないのでヒントを出さない。`o` / `?` は s12 `browse-open` / `help-screen` が定める。s11 までにあった `j/k スクロール` は、キュー画面のフッタ（s12 の `queue-screen` MODIFIED）と同じく移動系のキーとして `?` のヘルプに委ね、ヒントから外す）、右は s08 と同じステータス（スピナー / エラー）とする。ヒントが端末幅に収まらないときの切り詰めは s08 のフッタの規則のままである（カード詳細のヒントは幅 100 以下の端末で末尾から切れる。design.md）。

#### Scenario: 本文が高さを超えると j でスクロールする
- **WHEN** `Body` が `行01` から `行60` までの 60 段落（各段落を空行で区切る）である issue の Card の詳細を、幅 100・高さ 20 の `Model` で開き、`View` を読んでから `j` を 5 回与えて `View` を読む
- **THEN** 1 回目はヘッダ行 `org/app #<n>` と `行01` を含み `行30` を含まない。2 回目はヘッダ行を含んだまま `行01` を含まず `行06` を含む

#### Scenario: PR 詳細に移るとスクロール位置が先頭に戻る
- **WHEN** 上の `Model`（`j` を 5 回与えた後。Card の `PRs` に open PR が 1 件ある）に `Enter` を与えて PR 詳細に移り、`Esc` で戻って `View` を読む
- **THEN** `行01` を含む

#### Scenario: 詳細画面のフッタ
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えてカード詳細を開いた `Model` の `View` から ANSI エスケープを除いて読み、`Enter` で PR 詳細に移って再び読む
- **THEN** 1 回目の最終行に `Esc 戻る`、`Tab PR 選択`、`x 展開`、`a 回答`、`t todo`、`o ブラウザ`、`? ヘルプ`、`q 終了` が含まれ `1-4/Tab タブ` と `R 更新` と `j/k スクロール` は含まれない。2 回目の最終行に `Esc 戻る` と `g issue へ` と `a 回答` と `o ブラウザ` と `? ヘルプ` と `q 終了` が含まれ `Tab PR 選択` と `t todo` と `R 更新` は含まれない

#### Scenario: PR の無いカードのフッタには PR のキーを出さない
- **WHEN** `example` の issue 140 の Card（`PRs` 空）の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** 最終行に `Esc 戻る` と `x 展開` と `t todo` と `o ブラウザ` が含まれ、`Tab PR 選択`、`Enter PR を開く`、`g PR へ` は含まれない

#### Scenario: 低い端末では PR 一覧を削って本文 1 行を残す
- **WHEN** `Body` が `行01` の 1 行で、`PRs` が `propose` / `apply` / `archive` の open PR 3 件の issue の Card の詳細を、幅 120・高さ 7 の `Model` で開き、`View` から ANSI エスケープを除いて読む
- **THEN** ヘッダ行 `org/app #<n>` と本文の `行01` とフッタの `q 終了` が含まれ、`[archive] PR#` の行は含まれない

#### Scenario: 狭い端末でも詳細は 1 ペインで全部出す
- **WHEN** 幅 60・高さ 40 の `Model` でカード詳細を開き、`View` を読む
- **THEN** ヘッダ行 `org/app #108` と PR 一覧の `PR#131` と本文の `認証まわりの仕様を決めたい` がすべて含まれ、区切り線はちょうど 1 本で、`org/app #108` と `PR#131` はその上、`認証まわりの仕様を決めたい` はその下にある（ヘッダ領域 / 区切り線 / 本文領域 の 1 ペイン）

