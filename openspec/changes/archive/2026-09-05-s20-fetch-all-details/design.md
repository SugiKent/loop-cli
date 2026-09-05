## Context

`internal/fetch` の `fetchDetails` は、search で得た issue / PR のうち「分類に本文以外が要る」ものだけに詳細を取りに行く。条件は 4 つで、いずれも search 結果（ラベル・本文・`IsDraft`）だけで決まる。

1. `question` ラベルのある issue → `ViewIssue`（`Issue.Comments`）
2. すべての open PR → `ViewPR`（`PR.Comments`）
3. merge 候補 PR（段階ラベル 1 件以上・`question` 無し・本文が「未確定の判断: 0 件」）→ `ViewPRMergeState`（`PR.MergeState`）
4. `apply` ラベルのある PR → `ReviewThreads`（`PR.ReviewThreads`）

条件から外れた対象は `Comments` / `MergeState` / `ReviewThreads` が `nil` のままになり、s09 `card-detail` がそれを `未取得` と表示する。この設計は D-001「データ取得は『広く 2 回検索してクライアント側で分類する』」の遅延取得表に沿ったもので、search API の 30 req/分を避けつつ詳細の呼び出しを絞る意図だった。

実際に稼働リポジトリで使うと、キューに並ぶカードの多くに `未取得` が出る。例えば `propose` + `question` が付き本文が「未確定の判断: 2 件」の PR は 3 にも 4 にも当たらないので、checks・mergeable・review thread がすべて `未取得` になる。利用者から見ると「調べていないから分からない」が並び、結局 GitHub を開くことになって確認の手数が減らない。

## Goals / Non-Goals

**Goals:**

- カード詳細・PR 詳細に「取りに行っていないから分からない」という状態が出ないようにする
- `gh` の失敗やタイムアウトで値が得られなかった状態と、コメント 0 件・thread 0 件で値が無い状態を、画面で区別する
- キューの並び・タブ・`Situation` を一切変えない

**Non-Goals:**

- 取得の高速化・呼び出し回数の削減。この change は逆に呼び出しを増やす
- 詳細画面を開いたときの追加取得（遅延取得）の仕組みを作ること。全件先取りにするので不要
- D-001 の「ラベル変遷」と Issue 側 cross-reference を取得すること（s17 が担当する）
- 取得件数の上限やレート制限のガード（後述の見積もりを記録するに留める）

## Decisions

### D-1: 詳細は Fetch で全件先取りする（遅延取得しない）

`fetchDetails` の 4 つの条件分岐を外し、すべての open issue に `ViewIssue`、すべての open PR に `ViewPR` / `ViewPRMergeState` / `ReviewThreads` を呼ぶ。

**なぜ「詳細を開いたときに補完する」ではないか。** 遅延補完でも `未取得` は最終的に消えるが、開いた直後に `取得中` が出て値が入れ替わる。利用者が減らしたいのは「一覧を見たときに未読・未確認が並ぶこと」なので、開く前に揃っている必要がある。また遅延補完は UI 側に「この 1 件だけ取得中」という状態と、その結果を Card に差し込む経路を足すことになり、`Model` の状態が増える。全件先取りは `fetchDetails` の条件を消すだけで済む。

**分類結果は変わらない。** これはこの決定の前提であり、実装で確認する不変条件でもある。

- `classify.Issue` が `Issue.Comments` を読む分岐は 3 か所とも `question` で守られている（規則 4 / 規則 5 / 局面 B）。`question` の無い issue のコメントが埋まっても、そこに入らない
- `classify.isC` は「段階ラベル 1 件以上・`question` 無し・未確定 0 件」を先に判定してから `MergeState` を読む。これは現在の merge 候補の条件そのもので、今まで `nil` だった PR は `MergeState` を読む前に false で返る
- `classify.isD` は `apply` ラベルを先に判定してから `ReviewThreads` を読む。これも現在の取得条件と一致する

つまり `fetch` 側の 4 つの条件は、分類が実際に詳細を読む条件をもう一度書いたものである。条件を外しても、分類が受け取る入力は変わらない。

### D-2: `nil` は「取得失敗」だけを意味するようにする

全件取得にすると、`Comments` / `MergeState` / `ReviewThreads` が `nil` になるのは詳細取得が失敗したときだけになる。表示側に「失敗したかどうか」を伝える新しいフィールドや構造体を足す必要はなく、`nil` の文言を `未取得` から `取得失敗` に変えるだけでよい。

**なぜ `Result.Errors` を UI に渡さないか。** `Errors` は現在 `[]error` で、どの issue / PR のどのメソッドが失敗したかは文字列に畳まれている。表示のために構造を戻すと、`fetch` と `ui` の間に新しい型が要る。`nil` が失敗と一対一で対応するなら、その型は情報を増やさない。`Errors` はフッタのエラー表示（s08）の役割のまま変えない。

`長さ 0` の非 `nil`（コメント 0 件・thread 0 件・checks 0 件）が `なし` を意味するのは今までどおりで、`snapshot-cache` の JSON 往復もこの区別を保つ。`fetch.comments` と `decodeReviewThreads` はどちらも `make` で長さ 0 の非 `nil` を返すので、取得に成功して中身が空なら `なし` になる。

1 つの PR で 3 つの詳細が同時に失敗し得るようになるため、`Errors` の並びの規則に「同じ (repo, number) の中はメソッド順（`ViewPR` → `ViewPRMergeState` → `ReviewThreads`）」を足す。足さないと並行実行の完了順が残り、`card-fetch` の「並行実行の完了順に依存しない」が偽になる。フッタが出す `Errors[0]` も実行ごとに変わってしまう。

### D-3: 失敗の文言は `取得失敗` に統一する

`未取得` を出している 4 か所をすべて `取得失敗` に置き換える。

| 場所 | 現在 | 変更後 |
| --- | --- | --- |
| カード詳細の PR 行 | `checks 未取得 mergeable 未取得` | `checks 取得失敗 mergeable 取得失敗` |
| カード詳細の issue コメント | `コメント: 未取得` | `コメント: 取得失敗` |
| PR 詳細の checks | `checks: 未取得` | `checks: 取得失敗` |
| PR 詳細の会話コメント | `コメント: 未取得` | `コメント: 取得失敗` |
| PR 詳細の review thread | `review thread: 未取得` | `review thread: 取得失敗` |

`なし` の表示は変えない。

**文言の置換だけでは済まない。** `internal/ui/detail_test.go` の該当ケースは `example` を実際に `Fetch` した結果を使っており、全件取得にすると同じ入力から出る値そのものが変わる。`example` の PR 131 は `mergeable: UNKNOWN`・`StatusCheckRollup` に `ci/legacy PENDING` を持ち、`pr-131-review-threads.json` に未 resolve の thread が 1 件ある。`issue-140.json` はコメント 0 件である。したがって:

| 場所 | 変更前の期待 | 全件取得後の実際 |
| --- | --- | --- |
| カード詳細の PR 131 の行 | `checks 未取得 mergeable 未取得` | `checks 緑以外 mergeable UNKNOWN` |
| カード詳細の issue 140 | `コメント: 未取得` | `コメント: なし` |
| PR 詳細の PR 131 | `checks: 未取得` / `review thread: 未取得` | `mergeable: UNKNOWN BLOCKED` と checks 2 行 / `thread 未 resolve` |

`example` 由来のケースは実値に直し、`取得失敗` は `MergeState` / `Comments` / `ReviewThreads` を `nil` にした手書きの Card で別に検証する。

### D-4: テスト fixture の不足分を足す（`Fake` は変えない）

`Fake` の読み取りは fixture ファイルが無ければエラーを返す。全件取得にすると、これまで呼ばれなかった対象のファイルが要る。

- `internal/gh/testdata/fixtures/example` は issue 108 / 140 と PR 131 の全ファイルが揃っており、追加は不要
- `internal/fetch/testdata/` の 5 ディレクトリには不足がある（`link` は issue の `issue-*.json` と一部 PR の `pr-*-review-threads.json`、`multirepo` / `samestage` / `partial` も同様）

**なぜ `Fake` を「ファイルが無ければ空を返す」に変えないか。** `Fake` は fixture が実際の `gh` 出力の写しであることを担保する道具で、ファイルの欠落を黙って空で埋めると、採取漏れがテストで見えなくなる。実際の `gh` はコメント 0 件の issue にも review が無い PR にも空の結果を返すので、fixture 側にその「空の結果」を置くのが実態に合う。`gh-fake` の spec も変えずに済む。

### D-5: 並行度は 4 のまま、呼び出し回数の見積もりを記録する

open が issue N 件 / PR M 件のとき、1 回の更新の `gh` 呼び出しは次のとおり。

| | 現在 | 変更後 |
| --- | --- | --- |
| search | 2 | 2 |
| issue 詳細 | `question` の件数 | N |
| PR コメント | M | M |
| PR merge 状態 | merge 候補の件数 | M |
| PR review thread | `apply` の件数 | M |
| 合計 | 2 + α + M | **2 + N + 3M** |

消費する枠は 2 つに分かれる。`gh search issues` / `gh search prs` は REST の search 枠（30 req/分）で、1 更新あたり 2 回のままなので影響しない。`gh issue view` / `gh pr view --json` / `gh api graphql` はいずれも GraphQL の枠（5,000 point/時。REST の 5,000 req/時とは別勘定）で、増えるのはこちらだけである。

GraphQL は呼び出し回数ではなく point で数えるので、1 呼び出しの単価は実測しないと分からない。**仮に 1 呼び出し 1 point とすると**、`refresh_interval_sec` の既定値 120 秒（30 回/時）で 1 更新あたり **166 回**までが上限に収まる境目になり、`N + 3M ≤ 164` から issue 100 件 + PR 20 件（160）までは余裕、issue 200 件 + PR 50 件（350）では超える。単価が 1 を超えるなら上限はその分下がる。実際の単価は `gh api graphql -f query='{rateLimit{cost remaining}}'` と取得中の消費で測る（tasks 6.2）。

この枠は sugi-loop の専有ではない。手動の `R` と、利用者自身の他の `gh` 利用が同じトークンの枠を使う。30 回/時は自動更新だけの数である。

`detailConcurrency` は 4 のまま変えない。上げると secondary rate limit に近づき、この change の目的（表示から `未取得` を消す）とは別の判断になる。

**取得にかかる時間は伸びる。** 並行度 4 なら、28 回で 7 波、上限付近の 164 回では 41 波になる。1 呼び出し 1 秒でも 41 秒を超え、`ViewPRMergeState` の再取得が混ざればさらに伸びる。`refresh_interval_sec` の既定値 120 秒に収まるかは規模しだいで、tasks 6.2 の目視で確かめる。s08 `queue-screen` の「取得中は前回結果を維持する」により、待っている間は前の表とスピナーが出るので画面は止まらない。

### D-6: 詳細が埋まることで、表示だけが変わる箇所を受け入れる

分類は変わらないが、詳細を読んでいる表示は変わる。いずれも既存の spec が無条件に書いている振る舞いなので仕様違反ではなく、この change が意図する方向（見えるようになる）でもある。tasks では実装対象にせず、変化として記録する。

- **カード詳細の blocked-by 節が `question` 無しの issue にも出る。** `model.LatestBlockedBy(Issue.Comments)` は今まで `question` 無しの issue では常に nil を見ていた。今後は `blocked-by:` 行を含むコメントがあれば、質問と選択肢が描画される
- **キュー画面のプレビューに、`question` 無し issue のコメントが並ぶ。** `queue-screen` の「主体の `Comments` を並び順に出す」は無条件の記述であり、入力が増える
- **`a` の回答テンプレートが `question` 無し issue でも埋まる。** `action.AnswerTemplate` は最新の AI コメントから雛形を作るので、コメントがあれば空でなくなる
- **フッタの `詳細取得の失敗 N 件` の N が最大 3 倍になる。** 1 つの PR の失敗が 3 件のエラーを生むため。D-2 のメソッド順で `Errors[0]` は決定的に保つ

## Risks / Trade-offs

- **[呼び出し回数が open 件数に比例して増える]** → D-5 の見積もりを decisions.md（D-001 の追記）に残し、上限に近づく規模が分かるようにする。実際に超える運用が出たら、そのときに上限や間隔の調整を別 change で扱う
- **[`ViewPRMergeState` の UNKNOWN 再取得が全 PR に広がる]** → `mergeable` は base ブランチが進むたびに全 open PR で無効化されるため、routine が PR を merge し続けるリポジトリでは毎回の更新で一定数が `UNKNOWN` を返す。「新しい PR のときだけ」ではなく定常的に起きると見るべきで、最悪ケースの呼び出しは `2 + N + 4M` になる。さらに `RetryWait`（2 秒）は並行度の枠を握ったまま待つので、波あたりの実時間も伸びる。この change では再取得の仕組み自体は変えず、tasks 6.2 で実際の待ち時間を見る
- **[この change より前に作られたスナップショットは `nil` を含む]** → 旧スナップショットの `Comments` / `MergeState` / `ReviewThreads` の `nil` は「取りに行っていない」の意味だが、新しい表示では `取得失敗` と出る。この表示は初回の取得が完了するまで続く。s20 は取得を長くするので数十秒に及び、初回の search が失敗すれば（オフライン等）s08 / s13 が前回結果を維持するため、復旧するまで残る。誤りは表示だけで、書き込みや分類には影響しない。スナップショットにバージョンを持たせるほどの問題ではないと判断する
- **[取得の待ち時間が伸びて、初回起動時にキューが出るまでが遅くなる]** → s13 のスナップショットにより 2 回目以降の起動は前回の表が即座に出る。初回起動だけは待つことになる
- **[詳細取得の部分失敗が増える可能性]** → 呼び出しが増えれば失敗も増え得るが、失敗は `Result.Errors` に積まれてフッタに出る仕組み（s08）が既にあり、1 件の失敗が他を巻き込まない構造も変わらない

## 未決事項

docs が沈黙している点と、実装者が採る既定値。

1. **失敗の文言** — docs に `取得失敗` という語は無い。既定値: D-3 の表のとおり `取得失敗`。行の構造・並び・`なし` の表示は変えない。テストの期待値は文言の置換ではなく、D-3 の後半のとおり実値に直したうえで、`取得失敗` は手書きの `nil` で別に検証する
2. **`detailConcurrency` の値** — D-001 は並行度に触れていない。既定値: 4 のまま変えない
3. **取得件数の上限** — D-001 は 5,000 req/時に対する余裕があるとしか書いておらず、上限に達したときの振る舞いを定めていない。既定値: この change ではガードを設けず、D-5 の見積もりを decisions.md に記録するに留める
4. **旧スナップショットの `nil`** — s13 `snapshot-cache` はスナップショットのバージョンを持たない。既定値: 変換もバージョンも足さず、初回取得で置き換わるのに任せる
5. **`internal/fetch/testdata` に足す fixture の中身** — 既定値: コメント 0 件・thread 0 件の最小の JSON を置く。既存のテストが見ている分類結果が変わらない値にする（`Situation` の期待値を変えない）。**review thread のファイルは裸の `[]` ではなく GraphQL の入れ子**（`{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[]}}}}}`）で書く。`decodeReviewThreads` はこの形を前提にしており、裸の配列は unmarshal に失敗して「thread 0 件」ではなく「取得失敗」になる
