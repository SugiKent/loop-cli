## Context

s03-gh-client が `GHClient`（読み取り 8 メソッド）と `Fake`、s05-classify が `internal/model`（`Issue` / `PR` / `Card`、変換関数 `IssueFromSearch` / `PRFromSearch` / `CommentFrom`、`HasLabel` / `PRStages` / `ParseUndecided`）と `internal/classify`（`Issue()` / `PR()` / `Card()`）を定めている。live の GitHub から Card を作る層はまだ無い。
D-003 の内部構成に `fetch` は無い。`gh/` は生の型と `gh` 実行、`model/` は型、`classify/` は純粋関数、`ui/` は画面、`action/` は書き込みであり、「search → 遅延取得 → 紐づけ」はどれにも収まらないので `internal/fetch` を足す。

先行 change から引き取る前提:
- s03: `Client` はタイムアウトを持たず、s07 が `context.WithTimeout` の値（既定 30 秒）を決める。`ViewPRMergeState` の UNKNOWN 再取得は `internal/gh` が持ち、取得層は意識しない。`--limit 200` を超えた分の扱いは s07 の判断。デコード失敗と `*gh.Error` は `errors.As` で区別できる
- s05: `Comments` / `MergeState` / `ReviewThreads` が nil のとき、その詳細を要する条件は不成立。`Card.PRs` の順序は s07 が決め、`classify.Card` の同点判定はその順序を使う。`question` + `blocked` でコメント nil の issue は `other` に出る（取り忘れが見える）。D-001 の取得範囲と「キューに入れないもの」の規則 2（`question` 無し PR で最新コメントが人）の不整合は s07 が決める
- s02 / s03 の design は `cmd/sugi-loop` への配線（`config.Load` / `Check`）を s07 と書いているが、この change は配線をせず s08 に渡す。s08 が画面と一緒に配線する方が、hello world（s01）を置き換える差分が 1 か所に収まる
- s06 の `classify` サブコマンドと s05 の `fixture_test.go` は「全詳細を入れる」組み立てを持つ。この change の `Fetch` は D-001 の部分集合しか取らないので、同じ関数に置き換えると意味が変わる

D-001 の遅延取得の表は「`question` の issue → comments」「`question` の PR → comments」「merge 候補 PR → merge 状態」「`apply` PR → review threads」の 4 行で、「merge 候補」の定義は無い。human-turn-signals.md の除外規則 2 は `question` 無し PR のコメントを見るが、表にその行は無い。

## Goals / Non-Goals

**Goals:**
- `Fetch` 1 回で、設定した全リポジトリの open issue / open PR が分類済みの `model.Card` 群になり、s08 がタブ別に並べるだけで済む
- 遅延取得の対象を s05 の分類条件から機械的に導き、余分な `gh` 実行をしない
- 失敗を 2 段（全体失敗 / 部分失敗）に分け、部分失敗で issue / PR が消えない
- s03 の `Fake` だけでテストできる（live を叩かない、`gh` を起動しない）

**Non-Goals:**
- s08 が `cmd/sugi-loop` への配線と `Check` の呼び出しを担当する。この change の完了時点で TUI の表示は変わらない。`openspec/specs/gh-client/spec.md` の Requirement「起動前に gh の存在と認証を確認する」にある「`Check` を呼ぶのは起動時の s07 であり」は main spec に残っている。s08 がこの Requirement ブロックを MODIFIED して s08 に書き換える。この change は `gh-client` の spec に触らない
- s13 がスナップショットの保存・前回結果の維持・定期実行・通知の差分比較を担当する
- s17 が cross-reference による紐づけ補完と merge 済み PR の取り込みを担当する。この change は `CrossReferencedPRs` / `LabelTimeline` を呼ばない
- s18 が書き込み後の 1 件再取得とレートリミット表示を、s19 が GraphQL への統合を担当する
- `sugi-loop-cli classify --live` は足さない（s06 が決めている）

## Decisions

### ファイル構成

```
internal/fetch/fetch.go        # Result / CallTimeout / Fetch、詳細取得の対象判定（非公開の述語 3 つ。PR のコメントは全件）、Card 組み立て、PRs の並び
internal/fetch/link.go         # LinkedIssue（title / body のパース）
internal/fetch/fetch_test.go   # example と testdata/* を Fake で読む Scenario、呼び出し回数・期限・キャンセルの stub
internal/fetch/link_test.go    # LinkedIssue の Scenario
internal/fetch/testdata/<name>/  # s07 専用の手書き fixture（s03 の命名規則。下記）
```

### 詳細取得の対象は分類条件から導く

D-001 の表の 4 行を、s05 の局面の条件のうち「search 結果だけで決められる部分」を前提条件にして述語にする。PR のコメントだけは D-001 の行（`question` の PR）より広く、全 open PR で取る（次節）。

| D-001 の行 | 述語（search 結果だけで決める） | 取得 | 必要とする局面 |
| --- | --- | --- | --- |
| `question` の issue | `HasLabel(question)` | `ViewIssue` → `Comments` | B、進行中の規則 4 / 5 |
| 全 open PR（D-001 の表は `question` の PR） | 無条件（ラベルを見ない） | `ViewPR` → `Comments` | A、規則 2 / 3 |
| merge 候補 PR | `len(PRStages) >= 1 && !HasLabel(question) && ParseUndecided(Body) == (0, true)` | `ViewPRMergeState` → `MergeState` | C |
| `apply` PR | `HasLabel(apply)` | `ReviewThreads` → `ReviewThreads` | D |

- issue の `question` は `blocked` の有無で絞らない。規則 4（回答済み。sweep 待ち）と規則 5（`question` のみ。dispatcher の回収待ち）は `Summary` が違い、コメントが無いと 4 を出せない。s05 の `example` の期待値（issue 108 → 規則 4）もコメントを前提にしている
- merge 候補の定義は C の条件から `MergeState` に依存しない部分を全部取ったもの。`IsDraft` を含めない理由は spec に書いたとおり（s05 の C は draft を見ない。draft を除くと分類結果が s05 と食い違う）。`ParseUndecided` が `0, true` でなければ C は成立しないので、`ViewPRMergeState`（UNKNOWN なら 2 秒待つ）を呼ばずに済む
- `apply` PR は `question` があっても `ReviewThreads` を取る。`question` + `apply` でコメントが空の PR は A にも規則 3 にも当たらず D が評価されるため

### 全 open PR のコメントを取る（除外規則 2 を評価する）

human-turn-signals.md「キューに入れないもの」の 2 番目「`question` の付いていない open PR で最新コメントが人のもの」は s05 の進行中の規則 2 として実装済みで、全 open PR のコメントが無いと live で成立しない。D-001 の遅延取得表は `question` の PR のコメントしか挙げておらず、docs の 2 か所が噛み合っていない。この change は分類条件（human-turn-signals.md）を正本にし、全 open PR に対して `ViewPR`（comments）を遅延取得する。
根拠: 規則 2 を評価しないと、auto-fix が作業中の段階 PR（人がレビューコメントを書いた直後の PR）が C（mergeable なら）か `other` として今やるタブに merge 候補で出る。人の出番でないものが今やるに混ざる方が、取得 1 回の節約より害が大きい。コストは open PR 1 件につき `gh pr view` が +1 回（PR 10 件で +10 回 / 更新）で、REST の通常枠 5,000 req/時（D-001）に対して余裕がある。
これは D-001 の遅延取得表からの逸脱なので、表に「全 open PR → `gh pr view --json comments`」の行を足す docs 更新が要る（未決事項に記す。この change は docs を編集しない）。

### 紐づけは PR 側のパースだけ。同一リポジトリの open issue に限る

`LinkedIssue` は D-001 の 1（PR 側）をそのまま実装する。title の `[<段階>]` は `propose` / `apply` / `archive` の 3 語に限る。mvp.md の画面例がこの 3 語で、`[WIP]` 等の無関係な角括弧を段階と誤認しないため。本文は `Refs` / `Closes` の 2 語（GitHub のクローズキーワード全部（`Fixes` / `Resolves` 等）に広げない。docs に無く、プラグインの規約は `Refs` / `Closes` で固定されている）。
紐づけ先は `map[repo]map[number]*card` で引く。search は open issue しか返さないので、閉じた issue に紐づく PR や、`#n` が PR 番号の PR は PR 単独カードになる。落とすと「消えて見えなくなる項目」になるため。
D-001 の 2 番目の根拠（Issue 側の cross-reference で、`docs` PR の除外規則を含む）は s17 が足す。s17 は `CrossReferencedPRs` で得た merged PR を `model.PR`（`State` が `MERGED`）に写して `PRs` に加える形で `internal/fetch` を拡張する想定だが、この change では定義しない。

### PRs の並びは段階順 → 番号順

mvp.md「紐づく PR: `[propose] PR#131 merged` `[propose] PR#140 merged（最新・正本）` `[apply] なし`」の段階順に合わせる。段階は `PRStages(Labels)` の先頭（`propose` → `apply` → `archive`）、段階ラベル無しは末尾、同段階は番号昇順。番号は単調増加なので「同段階の複数 PR は古い順」になる。
`classify.Card` は `Priority` が同じ候補のうち `PRs` で先にあるものを `Card.Result` に採るので、この並びがカードの 1 行目を決める。並行取得しても並びが変わらないよう、詳細取得の後に組み立てと並び替えを行う。

### Fetch は分類まで済ませる

`Fetch` の最後で `classify.Card` を呼ぶ。s05 の Goals「s07 が Card を組み立てて `classify.Card` を呼ぶだけで 4 タブに振り分けられる」に従う。s08 が分類を呼ぶ案は、詳細取得の範囲（この change）と分類（s05）の整合をとる場所が s08 に移り、テストが `internal/fetch` で閉じない。

### 失敗は 2 段

- search の失敗、`ctx` の中断 → `(nil, error)`。Card が 1 枚も無い、または途中までしか無い結果で s13 のスナップショットを上書きさせない
- 詳細取得 1 件の失敗（`*gh.Error`、デコード失敗、`Fake` のファイル不在等） → その issue / PR の詳細を nil のままにして続行し、`Result.Errors` に積む。s05 の「詳細 nil は条件不成立」により、`question` PR は `other`、`question` + `blocked` issue は `other` として今やるに出る。s08 はステータスバーに `Errors` の件数か先頭を赤で出す（表示は s08）
- `ctx` の中断は「詳細取得 1 件の失敗」と区別する。全ゴルーチン終了後に親 `ctx.Err() != nil` なら全体失敗にする。`context.WithTimeout` で派生させた子 `ctx` の期限切れ（1 件が 30 秒を超えた）は親の `ctx` が生きていれば部分失敗

`Result.Errors` の要素は `fmt.Errorf("ViewPR org/app#131: %w", err)` の形で、メソッド名・リポジトリ・番号を含める。並びは issue のエラーを (repo, number) 順、続けて PR のエラーを (repo, number) 順に整列してから返し、並行実行の完了順に依存させない（テストと表示の両方で決定的にする）。

### 並行度は既定 4、search は直列（既定値）

spec の MUST は「詳細取得は並行してよい。並行しても `Cards` / `PRs` の並びは決定的」までで、並行度の上限と search の直列はこの節と未決事項の既定値である。
詳細取得は issue 30 件（うち `question` 5 件）・PR 10 件で 25 回前後の `gh` 実行になり、1 回 0.5〜1 秒として直列だと 15 秒を超える。上限 4 の並行で 5 秒程度に収める。実装は `sync.WaitGroup` + 容量 4 のチャネル（セマフォ）で、各ゴルーチンは自分の issue / PR のインデックスに書くので排他は要らない（`Errors` への追記だけ `sync.Mutex`）。`golang.org/x/sync/errgroup` は間接依存にあるが、最初のエラーでキャンセルする挙動が部分失敗と合わないので使わない。
search 2 回は直列にする。search API は 30 req/分で、2 回を並行にしても 1 秒しか変わらない。
`gh` の並行実行の上限を 4 にするのは、REST / GraphQL の secondary rate limit（同時リクエスト数）に当たらない範囲として。docs に値は無いので未決事項に記す。

### タイムアウトは呼び出し 1 回ごと

`context.WithTimeout(ctx, CallTimeout)` を `client` の各呼び出しの直前で作り、呼び出し後に `cancel` する。全体に 1 つの期限を付ける案は、`question` issue が多いときに search の結果ごと捨てることになる。1 回ごとなら遅い 1 件だけが部分失敗になる。`CallTimeout` は `const`（30 秒。s03 の未決事項の既定値）で、テストは親 `ctx` の期限で代用する（「親の期限が伝わる」Scenario）。

### テストは Fake と手書き fixture

- `example`（s03）で Card 2 枚の Scenario、`ViewIssue` の呼び出し（`Fake.Calls`）、`question` PR の詳細を検証する。`example` は s05 / s06 が期待値を持つので内容を変えない（`issue-140.json` は存在するが、この change は issue 140 の詳細を取らないので s07 は読まない）
- s07 専用の fixture は `internal/fetch/testdata/<name>/` に置く。`internal/gh/testdata/fixtures/` に置くと s05 の `fixture_test.go` が期待値表に無い alias として失敗し、s04 の `fixtures_test.go` が `login` の形式を検査する。ファイル名は s03 の命名規則（`Fake` がそれで探す）。1 ディレクトリを `Fake` に渡すと `repo` 引数は無視されるので、2 リポジトリのケースは `search-issues.json` の `repository.nameWithOwner` を 2 種類にして表現する
- 全 open PR が `ViewPR` を呼ぶので、fixture の `search-prs.json` にある PR には全部 `pr-<n>.json` を置く（`partial` の PR 131 だけは意図して置かない）。`Fake` は `ViewPR` と `ViewPRMergeState` で同じ `pr-<n>.json` を読むので、「merge 状態を取らない」ことは `pr-<n>.json` が `mergeable: MERGEABLE` でも `MergeState` が nil のままであることで検証する。「review threads を取らない」ことは `pr-<n>-review-threads.json` を置かず `Errors` が空であることで検証する（`Fake` は `ViewIssue` しか記録しない）。「取る」ことは詳細が埋まっていることで検証する。各 `pr-<n>.json` の `comments` は明示する（末尾が人なら s05 の規則 2 で進行中になる。PR 90 以外は空か末尾が AI）
- 呼び出し回数・期限・キャンセルは `gh.GHClient` を埋め込んだ stub（`type stub struct { gh.GHClient; … }`）で必要なメソッドだけ上書きする。`Fake` にエラー注入を足さない（s03 の決定）
- `Fake.Calls` は s03 が直列呼び出し前提で `append` しており、`Fetch` の並行取得で `ViewIssue` が同時に走ると競合する（`go test -race` で検出される）。`internal/gh/fake.go` の `Calls` への追記を `sync.Mutex` で守る。記録する内容と直列呼び出し時の順序の規則（`gh-fake` の既存 Requirement）は変えず、「並行呼び出しでも `Calls` の内容と件数を壊さない」を `gh-fake` に ADDED Requirement として足す（`specs/gh-fake/spec.md`）。並行した `ViewIssue` どうしの `Calls` の順序は決定的でないので、`Fetch` 側の `Calls` の検証は `example`（`ViewIssue` 1 回）に限る
- `internal/gh/testdata/fixtures/` 直下の全 alias に対する不変条件テスト（エラー無し・`Errors` 空・全番号がちょうど 1 枚に現れる）を置く。`Situation` の期待値は見ない。s05 の期待値表は全詳細を入れた分類で、この change は取得範囲が狭い（merge 状態・review threads は条件付き）ので食い違い得る。現在の alias は `example` だけで、s04 で採取した alias を足すときは全 open PR の `pr-<n>.json` が要る

fixture の内容（`testdata/`）:

| ディレクトリ | 内容 | 検証する Scenario |
| --- | --- | --- |
| `link` | `org/app` の issue 108（`stage:propose`。`question` 無し）。`search-prs.json` は 151（`archive`、title `[archive] #108`、`未確定の判断: 0 件`、`isDraft: true`）/ 131（`propose`、`Closes #108`）/ 140（`apply`、title `[apply] #108`、`未確定の判断: 1 件`）/ 60（`docs`）/ 61（ラベル無し）/ 62（ラベル無し、title `[propose] #999`）/ 132（`propose`、`未確定の判断: 2 件`、紐づけ無し）/ 90（`apply`、`未確定の判断: 1 件`、紐づけ無し）の順。全 PR に `pr-<n>.json` を置く: `pr-151.json`（`MERGEABLE`、`CheckRun/SUCCESS`、コメント無し）、`pr-132.json`（`MERGEABLE`、AI コメント 1 件）、`pr-90.json`（コメント 2 件、末尾が人）、他は `mergeable: UNKNOWN`・コメント無し。`pr-140-review-threads.json` と `pr-90-review-threads.json`（どちらも resolve 済み 1 件） | 段階順（search 順のソート）、merge 候補、未確定 1 件以上は merge 状態を取らない（PR 132）、docs / ラベル無し / 紐づけ先無しの PR 単独、`question` 無し PR で最新コメントが人 → 進行中（PR 90） |
| `multirepo` | `org/app` の issue 12 と `org/web` の issue 12（ラベル無し）、`org/web` の PR 30（title `[propose] #12`、`propose`、`未確定の判断: 1 件`）と `pr-30.json`（コメント無し） | 複数リポジトリで同じ番号 |
| `samestage` | issue 108（`stage:propose` + `question`）と `issue-108.json`（コメント 2 件、末尾が人）。`search-prs.json` は PR 140（`propose`、`未確定の判断: 1 件`、本文 `Closes #108`）、PR 131（`propose` + `question`、本文 `Closes #108`）、PR 88（`apply` + `question`）をこの順で。詳細は `pr-140.json`（コメント無し）、`pr-131.json`（`<!-- routine -->` 始まりのコメント 1 件）、`pr-88.json`（AI のコメント 1 件）、`pr-88-review-threads.json`（末尾が AI の未 resolve thread 1 件） | 同段階の open PR 複数、`apply` + `question` は `question` があっても review threads を取る（PR 88） |
| `partial` | issue 108（`question`、`issue-108.json` あり）、PR 131（`propose` + `question`、`Closes #108`、`pr-131.json` 無し） | 詳細取得 1 件の失敗 |
| `nosearch` | `search-prs.json` だけ（`search-issues.json` 無し） | search の失敗 |

fixture の `login` は s04 の規約に合わせて `user-N` にする（`internal/gh` の検査対象ではないが、形式を揃えておく）。

## Risks / Trade-offs

- [全 open PR の `ViewPR` で更新 1 回あたり PR 件数分の `gh` 実行が増える] → REST 5,000 req/時に対し +1 回 / PR で余裕がある。D-001 の表に行を足す docs 更新が要る（未決事項）
- [`--limit 200` を超える open issue / PR は取れない] → s03 と同じく D-001 の値を採る。`len(issues) == 200` で警告を出すかは未決事項（既定: 出さない）。P3 の GraphQL 統合（s19）でページングを考える
- [紐づけが PR 側のパースだけなので、title / body に番号を書かない PR は単独カードになる] → プラグインの規約は title `[<段階>] #<n>` を書く。取りこぼしは s17 の cross-reference で補う
- [並行実行で `gh` の secondary rate limit に当たる] → 並行度を 4 に抑える。当たった場合は `*gh.Error` として部分失敗に出る。値の調整は未決事項に記す
- [1 件 30 秒の期限が詳細取得の合計を長くする（最悪 25 件 / 4 並行 × 30 秒）] → 通常は 1 秒未満で返る。遅い 1 件が全体を巻き込まない方を優先する
- [`Fetch` が `classify` を呼ぶので `internal/fetch` は `gh` / `model` / `classify` の 3 つに依存する] → 依存の向きは一方向（`fetch` → `classify` → `model` → `gh`）で循環しない
- [`stub` を `gh.GHClient` の埋め込みで作ると、上書きしていないメソッドの呼び出しが nil パニックになる] → テストは上書きしたメソッドだけを呼ぶ経路（search 失敗、search 成功 + `ViewIssue` 待ち）に限る。パニックはテスト失敗として見える
- [s05 / s06 の組み立てと `Fetch` が重複する] → この change は s05 / s06 の組み立てを置き換えない（未決事項に記す）。s05 / s06 は全詳細を入れ、`Fetch` は D-001 の部分集合を入れるので、意味が違う

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| パッケージ名と場所 | `internal/fetch` | D-003 の一覧に無い。`gh` / `model` / `classify` / `ui` のどれにも収まらない |
| 「merge 候補 PR」の定義 | `PRStages` が 1 件以上 + `question` 無し + `ParseUndecided(Body) == (0, true)`。`IsDraft` は見ない | s05 の C の条件のうち search 結果で決められる部分。draft を除くと s05 と食い違う |
| 除外規則 2（`question` 無し PR で最新コメントが人）の扱い | 評価する（全 open PR に `ViewPR` を呼ぶ） | s05 の進行中の規則 2 を live で成立させる。auto-fix 作業中の段階 PR が merge 候補として今やるに出るのを防ぐ。REST 5,000 req/時に対し +1 回 / PR で余裕。D-001 の遅延取得表に「全 open PR → `ViewPR`（comments）」の行を足す docs 更新が必要（この change は docs を編集しない） |
| issue の `question` で `blocked` 無しの詳細取得 | 取る | 規則 4 と 5 の `Summary` を区別するのにコメントが要る |
| title の `[<段階>]` の語 | `propose` / `apply` / `archive` の 3 語のみ | mvp.md の画面例。`[WIP]` 等を段階と誤認しない |
| 本文のキーワード | `Refs` / `Closes` の 2 語。大文字小文字を区別しない。`Fixes` / `Resolves` 等は含めない | D-001 の記述どおり。規約は 2 語で固定 |
| title と本文が違う番号を指すとき | title を採る | title は規約で機械的に書かれ、本文は自由記述 |
| 本文に複数の `Refs` / `Closes` があるとき | 本文の先頭から最初のもの | 1 PR は 1 Card に入る。複数 issue への所属は作らない |
| 紐づけ先の open issue が無い PR | PR 単独カード | 落とすと見えなくなる |
| PR 単独カードをラベルで選別するか | しない（全部カードにする） | mvp.md の「`docs` PR とその他だけ」は定常状態の説明 |
| `PRs` の並び | 段階順（propose → apply → archive → 無し）→ 番号昇順 | mvp.md「段階順に並べる」 |
| `Cards` の並び | issue カードを `SearchIssues` の順、続けて PR 単独カードを `SearchPRs` の順 | 決定的であれば足りる。タブ内の並びは s08 |
| `Fetch` が分類まで行うか | 行う（`classify.Card` を呼んで返す） | s05 の Goals。s08 は並べるだけ |
| `Result` の形 | `Result { Cards []model.Card; Errors []error }`。全体失敗は `(nil, error)` | 部分失敗と全体失敗を型で分ける |
| `Errors` の文言 | `<メソッド名> <owner/name>#<n>: <元のエラー>`（`%w`） | s08 が赤で出すときに何が失敗したか読める |
| `Errors` の並び | issue のエラーを (repo, number) 順、続けて PR のエラーを (repo, number) 順 | 並行実行の完了順に依存させない。複数リポジトリで同じ番号があっても決定的 |
| `ctx` 中断の判定 | 全ゴルーチン終了後に親 `ctx.Err()` を見る。子 `ctx`（30 秒）の期限切れは部分失敗 | 途中までの Card でスナップショットを上書きしない |
| タイムアウトの単位と値 | `gh` 呼び出し 1 回ごとに 30 秒（`const CallTimeout`） | s03 の未決事項の既定値。遅い 1 件を部分失敗に閉じ込める |
| 並行度 | 詳細取得 4、search は直列（spec の MUST は「並行してよい」まで） | secondary rate limit に当たらない範囲。docs に値は無い |
| 並行の実装 | `sync.WaitGroup` + 容量 4 のチャネル。`errgroup` は使わない | 最初のエラーでキャンセルする挙動が部分失敗と合わない |
| search 結果の `Labels` / `Body` を詳細で上書きするか | しない | 同じ更新の中で取っており、差は無視できる。s18 の 1 件再取得が別途扱う |
| `len(issues) == 200` の警告 | 出さない | docs に無い。s19 でページングを考える |
| s05 `fixture_test.go` / s06 `classify` の組み立てを `Fetch` に置き換えるか | 置き換えない | 全詳細 vs D-001 の部分集合で意味が違う。s05 の期待値表は全詳細前提 |
| s07 専用 fixture の場所 | `internal/fetch/testdata/<name>/`（s03 の命名規則） | `internal/gh/testdata/fixtures/` に置くと s05 の期待値表と s04 の検査に引っかかる |
| `example` fixture の変更 | しない | s05 / s06 が期待値を持つ |
| `cmd/sugi-loop` への配線 | しない（s08） | 画面の置き換えと同じ change に置く。s02 / s03 の記述を上書きする |
