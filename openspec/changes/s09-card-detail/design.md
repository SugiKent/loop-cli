## Context

s08-queue-screen が `internal/ui` に `Model`（`New(fetcher)` / `Init` / `Update` / `View`、`fetchedMsg{res, err, at}`、`cards` とタブ別の行、`cursor`、`width` / `height`、1 ペイン用の `showPreview`、spinner）と `Subject`、`Elapsed`、プレビュー（Glamour で本文とコメント。コメントは routine マーカー行を除き、AI は各行に `▌`）を置く。キー処理は `tea.KeyPressMsg` の `String()` で判定し、`Enter` / `g` / 表に無いキーは何もしない。s08 はレビュー中で、この change は s08 の specs / design を前提にする。
s07 の `Fetch` が Card に入れる詳細は、`question` の issue のコメント、全 open PR のコメント、merge 候補 PR の `MergeState`、`apply` PR の `ReviewThreads` に限られる。それ以外は nil のままで、s05 は nil を「未取得」と定めている。
s05 の `internal/model` に `IsAI`（`Comment.AI`）、`ParseUndecided`、`LatestBlockedBy`、`ParseQuestions`、`IssueStages` / `PRStages` / `HasLabel` があり、s07 の `internal/fetch` に `LinkedIssue` がある。詳細画面に要るパーサはこれで揃っており、この change は新しいパーサを `depends on #m` の 1 つしか足さない。
mvp.md「カード詳細（Enter）」が画面の正本で、キーバインド表に `Enter`（詳細を開く。内部処理 `gh issue view` / `gh pr view --json`）、`x`（routine コメント展開）、`g`（PR ↔ issue 相互ジャンプ）がある。戻るキーとスクロールキーは表に無い。

## Goals / Non-Goals

**Goals:**
- キュー画面の `Enter` で、Issue と紐づく PR 群を 1 枚で読める。主要フロー A / B で「何を聞かれているか」（`## Q1.` と選択肢、`blocked-by: human` の要約）を読む場所を用意する
- AI / 人のコメントを s05 の判定（`Comment.AI`）だけで区別し、routine コメントを折りたたんで同趣旨の連投を見やすくする
- s07 が取っていない詳細を「未取得」と明示し、黙って空に見せない
- Model の振る舞い（画面遷移・折りたたみ・PR 選択・スクロール）と `View` の文字列を `internal/ui` のテストで、`example` fixture と手書きの Card で検証できる

**Non-Goals:**
- `a` 回答（s10。詳細画面でも `a` を効かせるかは s10 が決める）、`o` / `R` / `?`（s12）、自動更新中の詳細の追従（s13）、merge ガードの理由表示（s14）、review thread の選択と `A` 返信（s16）、段階の変遷タイムライン・「この段階に入ってからの経過時間」・cross-reference による merge 済み PR の取り込み（s17）
- 詳細を開いたときの `gh` の追加取得（未決事項。既定は取らない）
- `depends on #m` の先の issue を引くこと（番号を出すだけ）

## Decisions

### ファイル構成

```
internal/ui/detail.go        # 画面の状態の型、詳細の状態（対象 Card / PR 選択 / 展開フラグ / viewport）、詳細のキー処理、詳細の View（ヘッダ領域・本文領域・フッタ）、depends on のパース
internal/ui/detail_test.go   # 画面遷移・PR 選択・折りたたみ・スクロール・View の文字列
internal/ui/model.go         # 画面の状態を Model に足し、Update をキュー / 詳細で振り分ける（変更）
internal/ui/view.go          # View を画面の状態で振り分け、キューのフッタに Enter 開く を足す（変更）
internal/ui/preview.go       # s08 のコメント 1 件を行にする関数に折りたたみの引数を足す（変更。プレビューは常に展開で呼ぶ）
internal/ui/model_test.go / view_test.go  # Enter を「何も変えない」と見ていた検証を直し、x / Esc を足す（変更）
```

### 画面の状態は Model の 1 フィールド

`Model` に `screen`（キュー / カード詳細 / PR 詳細の 3 値）と `detail` 構造体（`card model.Card`、`prIdx int`、`expanded bool`、`fromDetail bool`（PR 詳細にカード詳細から入ったか）、本文領域の viewport）を足す。画面ごとに別の Bubble Tea Model を持って `Update` を委譲する形にしない。3 画面が同じ `cards` / `at` / spinner / エラー表示を共有し、詳細は「開いている Card のコピーと少数のフラグ」でしかないので、1 つの `Model` の中で `switch screen` する方が短い。
`Update` の先頭で `q` / `Ctrl+C` / サイズ / `fetchedMsg` / spinner を画面に関わらず処理し、キーはその後に `screen` で振り分ける。

### 詳細の対象は Card のコピー

`Enter` で `rows[tab][cursor]` の `model.Card` を値でコピーして `detail.card` に持つ。`fetchedMsg` で `cards` が差し替わっても詳細は開いたときのままにする（spec のとおり）。参照（添字）で持つと差し替えで別の Card を指すか範囲外になり、追従の規則を今決める必要が出る。追従は s13 の自動更新が要るときに s13 が決める。

### ヘッダ領域は固定、本文領域は viewport

Issue 本文 + コメント時系列は端末の高さを超えるのが普通で、スクロール無しでは読めない。s08 はプレビューにスクロールを持たなかったが、詳細では必須になる。D-003 が Bubbles の viewport を挙げているのでそれを使う。
PR 一覧（選択の対象）を viewport の外の固定領域に置く。選択中の PR が常に見え、`Tab` の効果がスクロール位置に依存しない。本文領域の高さは `height - ヘッダ領域の行数 - 区切り 1 - フッタ 1` で、下限は 1 行。1 行を割るときは PR 一覧を末尾から落としてヘッダ領域を縮める（PR 一覧は本文より優先度が低く、`Tab` の対象が減るだけで済む）。ヘッダ領域の各行（タイトル行・Summary 行・段階行・depends on 行・PR 一覧行、PR 詳細の 2 行）は表示幅が `width` を超えれば Lip Gloss の切り詰めで `…` を付ける（s08 の表のタイトル列と同じ方法）。折り返すと行数が変わり、本文領域の高さの計算が崩れる。本文領域の内容（レンダリング済み文字列）は、詳細を開いたとき・`x`・`g` / `Enter` / `Esc` で画面が変わったとき・サイズが変わったときに作り直して viewport に渡し、`View` では作り直さない（Glamour を毎フレーム呼ばない）。

### スクロールと PR 選択のキー

`j` / `k` / `↑` / `↓` はスクロール、`Tab` は PR 選択に使う。mvp.md の表で `j` / `k` は「行移動」で、詳細画面での意味は書かれていない。PR 一覧は数件しか無く、本文は数十行あるので、頻度の高い方（スクロール）を `j` / `k` に当てる。`Tab` は詳細画面ではタブ切替の用途が無く、「次の候補へ」という意味がキュー画面と揃う。`h` / `l` / `←` / `→` はカンバンの列移動（s17）に予約されているので使わない。

### 戻るキー

mvp.md に「戻る」キーが無い。`q` は終了に固定されている（mvp.md、s01）ので、`Esc` を戻るに当てる。`Esc` は表に無いキーで、後続 change が使う予定も無い。

### 折りたたみは画面単位の 1 フラグ

`x` は「画面内の AI コメント全部」を切り替える。コメントごとのフラグと選択カーソルを持つと、コメントの選択キーが要り、mvp.md にそのキーが無い。同趣旨の連投（human-turn-signals.md「同一イベントから複数セッションが起動し、同趣旨のコメントが 3 連続で付く」）を畳む目的は全体切替で足りる。折りたたみの見出しに要約（マーカー行を除いた最初の行）と `(+n 行)` を出し、畳んだまま「何のコメントか」が分かるようにする。
人のコメントは畳まない。人の発言は短く、A / B の流れで「自分が何と答えたか」を確認する対象だからである。

### コメントの行を作る関数を s08 と共有する

s08 `preview.go` のコメント整形（routine マーカー行の除去 = `strings.TrimSpace` 後の行全体が `<!-- routine -->` / `&lt;!-- routine --&gt;` に一致する行だけを落とす → 本文を Glamour で幅 − 1 でレンダリング → 見出し `AI  HH:MM` / `<Author>  HH:MM`、AI は全行に `▌`）に折りたたみの引数を足し、プレビューは常に展開で、詳細は `expanded` で呼ぶ。同じ書式を 2 か所で持つと見出しの書式がずれる。
展開形は s08 の出力そのもの（マーカー行を除き、Glamour を通し、`▌` 付きの全行）。折りたたみの要約と `(+n 行)` はマーカー行を除いた生の `Body`（Glamour を通さない）から取る。Glamour の出力は余白や折り返しで行数が環境に依存し、要約の 1 行にも装飾が混じるためである。review thread のコメントは `gh.ReviewComment`（`model.Comment` ではない）なので、`model.IsAI(Body)` で AI を決めてから同じ関数に渡せる形（`Author` / `Body` / `CreatedAt` / `AI` を引数にする）にする。

### PR 一覧の 3 段階と `なし`

mvp.md の例 `[propose] PR#131 merged` `[propose] PR#140 merged（最新・正本）` `[apply] なし` に合わせ、`propose` / `apply` / `archive` の 3 段階を常に出し、無い段階は `なし` にする。段階ラベルの無い PR（s07 が末尾に並べる）は `[-]` で 3 段階の後に出す。P1 では search が open PR しか返さないので `merged` と `（最新・正本）` は出ない。`Canonical` は s05 が立てる印をそのまま読む。

### 未取得の表示

`Comments` / `MergeState` / `ReviewThreads` の nil は「未取得」、長さ 0 は「なし」として別の文言で出す（s05 が区別を表示に委ねている）。`example` では issue 140 のコメントと PR 131 の checks / review thread が「未取得」になる（s07 の取得範囲）。s07 の取得範囲どおりの状態なので、通常の色で出す（赤はエラーに使う）。

### 追加取得はしない

この決定は docs からの逸脱である。mvp.md のキーバインド表は `Enter` の内部処理に `gh issue view` / `gh pr view --json` を挙げ、human-turn-signals.md は「merge 可否は表示時に `gh pr view --json mergeable,mergeStateStatus,statusCheckRollup` で取り直す」と書いている。この change はどちらも行わず、s07 の `Fetch` が入れた値だけを出し、無いものは `未取得` と出す。
理由: s07 が全 open PR のコメントと `question` issue のコメントを既に取っている。主要フロー A / B に要るものは揃っており、残りは `question` 無し issue のコメント（E / 進行中）と、merge 候補以外の checks（C 以外では merge しない）である。開いたときに取る形にすると、`Model` が `GHClient` を持つか `Fetcher` に 1 件取得を足す必要があり、s08 の「`Model` は取得関数だけを受け取る」を崩す。
申し送り: mvp.md のキーバインド表（`Enter` の内部処理）と human-turn-signals.md の「表示時に取り直す」は、この change の archive 時に「取得は `Fetch` が担い、詳細画面は取り直さない」へ更新が要る（docs の更新はこの change の範囲外）。詳細を開いたときの 1 件再取得を s18 の 1 件再取得に含めるかは s18 が決める。s09 では決めない。

### `depends on #m` のパース

mvp.md は `depends on #m` としか書いていない。`(?i)depends on #(\d+)` で本文全体から出現順に拾い、ヘッダに番号を並べる。行頭に限定しない（書式が定まっていないため）。関数は `detail.go` に置き、`model` には足さない（分類に使わない）。

## Risks / Trade-offs

- [Bubbles v2 の viewport の API 名が想定と違う] → spec は振る舞い（1 行 / ページ、先頭と末尾で止まる）で書いてある。実装時に README で確認する
- [Glamour の出力の行数が環境で変わり、スクロールの Scenario が壊れる] → Scenario は「60 段落のうち `行30` が初期表示に無い」「5 行スクロールで `行01` が消え `行06` が残る」と、余白の行数に依存しない範囲で書いてある
- [s08 の specs がレビューで変わり、MODIFIED で写した本文とずれる] → archive の順は s08 → s09。s09 の実装前に s08 の最終版と `specs/queue-screen/spec.md` を突き合わせる（tasks 1.1）
- [`Tab` の意味がキュー（タブ切替）と詳細（PR 選択）で違う] → フッタのヒントに `Tab PR 選択` を出す。画面が違えば混同しない
- [折りたたみが全体切替なので、1 件だけ読みたいときも全部開く] → AI コメントは連投が多く、まとめて開く方が速い。個別展開が要るならコメントの選択キーと一緒に後で足す
- [詳細を開いている間に自動更新（s13）が走っても表示が古いまま] → spec に明記。s13 が追従を決める
- [`Issue` が nil のカードでは PR 詳細が直接開き、`Esc` の戻り先がキューになる] → `fromDetail` で区別する。状態は 1 つ増えるだけ

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| 詳細を開いたときの追加取得 | しない。s07 の `Fetch` が入れた詳細だけを出し、nil は `未取得` | mvp.md の `Enter` 内部処理と human-turn-signals「表示時に取り直す」からの逸脱（Decisions「追加取得はしない」。docs 更新の申し送りあり）。s18 に含めるかは s18 が決める |
| 戻るキー | `Esc` | mvp.md に戻るキーが無い。`q` は終了に固定 |
| 詳細のスクロールキー | `j` / `k` / `↑` / `↓` で 1 行、`PgUp` / `PgDn` でページ。Bubbles の viewport | mvp.md にスクロールキーが無い。本文は数十行になる |
| PR 選択のキー | `Tab`（次の PR へ。末尾の次は先頭） | `j` / `k` はスクロールに使う。`h` / `l` は s17 のカンバンに予約 |
| `g` の意味 | カード詳細 → 選択中の PR の PR 詳細（`Enter` と同じ）、PR 詳細 → カード詳細。`Issue` nil なら PR 詳細で何もしない | mvp.md「PR ↔ issue を相互ジャンプ」。P1 では PR は Card の中にあるので `LinkedIssue` で引き直さず、表示（`紐づく issue: #n`）にだけ使う |
| `Issue` が nil のカードの `Enter` | PR 詳細を直接開く。`Esc` でキューに戻る | カード詳細に出すものが PR 一覧 1 件しか無い |
| `x` の単位 | 画面内の AI コメント全部を 1 フラグで切り替え。`Enter` で開くたびに折りたたみ。カード詳細 ↔ PR 詳細の遷移（`g` / `Enter` / `Esc`）では展開状態を維持する | コメントの選択キーが無い。連投を畳む目的は全体切替で足りる |
| 折りたたみの見出し | `▌AI  HH:MM  <要約>  (+n 行)`。要約はマーカー行を除いた生の `Body` の空行でない最初の行、`n` はその後の行数（空行を含む） | 畳んだまま何のコメントか分かる。生の行数なら Glamour の環境差に依らない |
| 人のコメントの折りたたみ | しない | 短く、自分の回答を確認する対象 |
| review thread 内のコメントの折りたたみ | しない | 1 thread は短い。s16 が thread の選択を足す |
| コメント本文の Markdown レンダリング | する。s08 と同じく Glamour で幅 − 1、マーカー行を除いてから。AI は各行に `▌` | s08 の既定値（D-003）を引き継ぐ。折りたたみの要約だけ生の行 |
| ヘッダ領域の構成 | カード詳細: `<Repo> #<n>  <Title>` / `Card.Result.Summary`（空なら省く）/ `段階: …` + バッジ / `depends on: …`（あれば）/ PR 一覧（3 段階 + `[-]`）。順序だけ定め、行番号は固定しない。PR 詳細: `<Repo> PR#<n>  <Title>` / `[<段階>] <状態>  labels: …` | mvp.md の項目を 1 行ずつ。`Summary` は mvp.md「いま人が何をすべきか」で s08 が s09 に委ねた。段階の経過時間とタイムラインは s17 が ADDED で行を足す |
| ヘッダ領域の行の切り詰め | 表示幅が `width` を超えれば切り詰めて末尾 `…`。折り返さない | 折り返すとヘッダの行数が変わり本文領域の高さが崩れる。s08 のタイトル列と同じ |
| 本文領域の高さの下限 | 1 行。割るときはカード詳細の PR 一覧を末尾から落とす。PR 一覧を使い切っても本文 1 行を確保し、はみ出しは切る | 本文が 0 行だと読めない。PR 一覧は `Tab` の対象が減るだけ |
| 段階ラベルが 2 つ以上のヘッダ | 全部並べる。色は付けない | 異常を隠さない。色は mvp.md に無い |
| `depends on #m` の書式 | `(?i)depends on #(\d+)` を本文全体から出現順に。行頭に限定しない | mvp.md は `depends on #m` としか書いていない |
| PR 一覧の各項目の表記 | `未確定 <n> 件` / `1 行目なし`、`labels: …`、`checks 緑` / `checks 緑以外` / `checks 未取得`、`mergeable <値>` / `mergeable 未取得` | mvp.md「1 行目判定・ラベル・checks・merge 状態」を短く。`ChecksGreen` は bool なので緑かどうかだけ |
| 段階ラベルの無い PR の段階名 | `[-]`。3 段階の後に出す | s07 が段階無しを末尾に並べる |
| PR 詳細の checks の各行 | `CheckRun` は `<Name>: <Conclusion>`（空なら `<Status>`）、`StatusContext` は `<Context>: <State>` | s03 の型のフィールドをそのまま |
| 未取得と空の文言 | nil は `未取得`、長さ 0 は `なし` | s05 が区別を表示に委ねた |
| 未取得の色 | 付けない | 設計どおりの状態で、エラーではない |
| 詳細を開いている間の `fetchedMsg` | `cards` / `at` は更新し、詳細の対象は差し替えない | 追従の規則は s13 |
| 詳細画面の 2 ペイン / 1 ペイン | 常に 1 ペインで全幅・全高。`p` は何もしない | 詳細は 1 つの内容しか無い |
| 本文領域の作り直しのタイミング | 開いたとき・`x`・画面遷移・サイズ変更。`View` では作らない | Glamour を毎フレーム呼ばない |
| 詳細画面の時刻のタイムゾーン | s08 と同じく `fetchedMsg.at` の `Location` | 一貫性 |
| フッタのヒント | カード詳細 `Esc 戻る  Tab PR 選択  Enter PR を開く  x 展開  g PR へ  j/k スクロール  q 終了`（`PRs` が空なら `Tab PR 選択  Enter PR を開く  g PR へ` を省く）、PR 詳細 `Esc 戻る  x 展開  g issue へ  j/k スクロール  q 終了`。展開中も `x 展開` のまま。右は s08 のステータス | 動くキーだけを出す |
| `Subject` の利用 | 詳細では使わない。ヘッダは `Card.Issue`（nil なら PR）で決める | mvp.md のヘッダは「リポジトリ、Issue 番号」。s08 の `Subject` は変えない |
