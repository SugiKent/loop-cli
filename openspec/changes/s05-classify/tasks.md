## 1. internal/model の型と変換

- [x] 1.1 `internal/model/model.go` を作成し、spec `card-model`「model は Issue / PR / Comment / Card と分類結果の型を定義する」のとおり `Comment` / `Issue` / `PR` / `Card` / `Result` / `Situation`（定数 `SituationA`〜`SituationG` / `SituationOther` / `SituationInProgress`。ゼロ値 `""` は未分類）/ `Tab`（`TabNow` / `TabBacklog` / `TabInProgress` / `TabAbnormal`）、ラベル定数 11 個、`HasLabel` / `IssueStages` / `PRStages` を実装する。`Situation` の `Priority()` / `Tab()` / `Kind()` は spec `human-turn-classify`「局面ごとの優先度・タブ・種別・1 行要約が決まる」の表どおり（`""` は 8 / 空文字列 / 空文字列）
- [x] 1.2 同ファイルに `IssueFromSearch(gh.SearchIssue) Issue` / `PRFromSearch(gh.SearchPR) PR`（`State` は `OPEN`、詳細は nil）/ `CommentFrom(gh.Comment) Comment`（`AI` は `IsAI(Body)`）を実装する
- [x] 1.3 `internal/model/model_test.go` を作成し、`IssueStages` の段階順（`question, stage:apply, blocked, stage:propose` → `stage:propose, stage:apply`）/ `PRStages`（`question, archive, docs` → `archive`）/ `IssueFromSearch` の `Repo` / `Number` / `Labels` / `Comments` nil / `PRFromSearch` の `State` `OPEN` と詳細 nil / `CommentFrom` の `AI` true / `Situation` 9 値 + ゼロ値の `Priority()` と `Tab()` と `Kind()`（A と D が `質問`、`""` が空文字列）が表どおりであることを検証する

## 2. internal/model のパーサ

- [x] 2.1 `internal/model/parse.go` を作成し、`IsAI(body string) bool`（本文が字面どおり `<!-- routine -->` または `&lt;!-- routine --&gt;` で始まる（`strings.HasPrefix`。TrimSpace しない）、または行頭から `## PR リスク評価` で始まる行がどこかにある）、`ParseUndecided(body string) (int, bool)`（先頭の空行を除いた 1 行目を `^未確定の判断:\s*(\d+)\s*件` で見る）、`LatestBlockedBy(comments []Comment) (*Comment, string, bool)`（末尾から先頭へ、先頭の空白を除いて `blocked-by:` で始まる行（各行を `strings.TrimSpace` してから `strings.HasPrefix`）を含む最初のコメント。`value` は最初の該当行の `blocked-by:` より後ろを `strings.TrimSpace`）、`ParseQuestions(body string) []Question`（見出し `^##\s*Q(\d+)\.\s*(.*)$`、選択肢 `^[-*]\s*選択肢\s*([A-Z])\s*(（推奨）|\(推奨\))?\s*[:：]\s*(.*)$`）を実装する。`regexp` はパッケージ変数にコンパイルする
- [x] 2.2 `internal/model/parse_test.go` を作成し、spec `card-model` の Scenario をそのまま検証する。`IsAI` 6 ケース（マーカー始まり / エスケープ済み / 3 行目の `## PR リスク評価` / 人 / 途中引用 / 先頭空白の後のマーカーは人）、`ParseUndecided` 3 ケース（0 件 / `2 件 — merge しないでください` / 1 行目に無い）と先頭が空行のケース、`LatestBlockedBy` 3 ケース（最新が正本 / 同趣旨 3 連続で末尾 / 無し）、`ParseQuestions` 2 ケース（2 問と推奨 / 見出し無しで空）と全角 `：` の選択肢

## 3. internal/classify の単体判定

- [x] 3.1 `internal/classify/classify.go` を作成し、`Issue(model.Issue) model.Result` / `PR(model.PR) model.Result` / `ChecksGreen(*gh.PRMergeState) bool` を実装する。評価順は spec「分類は純粋関数で、進行中の除外・判定表の順・フォールバックの順に評価する」のとおり: 進行中の 5 規則（issue: `IssueStages` が `stage:propose` / `stage:apply` / `stage:archive` の 1 件だけ + `wip` / PR: `question` 無し + 最新コメントが人 / PR: `question` + 最新コメントが人 / issue: `question` + 最新コメントが人 / issue: `question` + `blocked` 無し）→ A〜G を表の順（issue は B → E → F、PR は A → C → D → F → G）→ フォールバック（PR は `other`、issue は `question` + `blocked` なら `other`、それ以外は `in-progress`）。`PR` の `State` が `OPEN` 以外ならゼロ値の `Result` を返す。`Result` は `Situation` から `Priority()` / `Tab()` を埋め、`Summary` は spec の表の文言（進行中は規則ごとに 6 種、`other` は PR / issue で 2 種）
- [x] 3.2 `internal/classify/classify_test.go` を作成し、手書きの `model.Issue` / `model.PR` で spec の各 Requirement の Scenario を検証する。評価順 3 ケース（`wip` が先 / 詳細 nil で `other` / `MERGED` でゼロ値）、A 2 ケース（最新が人なら `in-progress`）、B 2 ケース、C 6 ケース（成立 / N=2 / 1 行目無し / checks 失敗 / UNKNOWN / `ChecksGreen` 空で true。`IsDraft` は見ない）、D 3 ケース、E 3 ケース、F 4 ケース（`wip` でも段階ラベル 2 つなら F を含む）、G 2 ケース、進行中 6 ケース（`question` PR で最新が人を含む。`Summary` の文字列一致を含む）、その他 3 ケース（ラベル無し PR / `retro` PR / `question` + `blocked` でコメント nil の issue）、優先度とタブの表。`Comment` の AI / 人は `AI` フィールドで直接与える

## 4. internal/classify の Card 集約

- [x] 4.1 `internal/classify/card.go` を作成し、`Card(model.Card) model.Card` を実装する。入力をコピー（`PRs` は新しいスライス、`Issue` は新しいポインタ）→ `Issue.Result` と open PR の `Result` を埋める → 同段階の `MERGED` PR のうち番号最大に `Canonical` → 候補（`in-progress` 以外、ゼロ値以外）の `Priority` 最小を `Card.Result` に（同点は Issue、次に `PRs` の順）→ 候補が無ければ `in-progress`（`Summary` は先頭の open PR → Issue → `進行中` の順で採る）
- [x] 4.2 `internal/classify/card_test.go` を作成し、spec「Card は Issue と open PR 群のうち最上位の局面を 1 行目に出す」と「同段階の merge 済み PR は最新が正本で、古い PR の question を異常扱いしない」の Scenario を検証する。PR の A が issue の進行中に勝つ / 同点で issue 優先 / すべて進行中（`Summary` は open PR のもの）/ open PR が無ければ issue の `Summary` / Issue nil の docs PR / 入力不変（呼び出し後に入力の `Result` がゼロ値、`PRs` の `Canonical` が false のまま）/ 古い merge 済み propose PR の `question` が無視され `Canonical` が番号最大に付く / 段階が違えば両方 `Canonical`

## 5. fixture と期待値表

- [x] 5.0 `internal/gh/testdata/fixtures/example/issue-140.json` を手書きで追加する（s03 の `example` には無く、`Fake.ViewIssue(140)` が失敗する）。内容は `search-issues.json` の 140 と整合する最小の `IssueDetail`（open、`labels` 空、`comments` 空、`body` / `title` / `url` / `number` は search と同じ。login を書く場合は `user-N` 形式）。`go test ./internal/gh/...`（s04 の個人情報検査を含む）が通ることを確認する
- [x] 5.1 `internal/classify/fixture_test.go` を作成する。期待値表 `var expected = map[string]map[string]model.Situation{"example": {"issue-108": model.SituationInProgress, "issue-140": model.SituationE, "pr-131": model.SituationA}}` を置き、`../gh/testdata/fixtures/` 直下の全ディレクトリについて `gh.NewFake(dir)` で `SearchIssues` / `SearchPRs` を読み、各 issue は `IssueFromSearch` + `ViewIssue` の `Comments`（`CommentFrom`）、各 PR は `PRFromSearch` + `ViewPR` の `Comments` + `ViewPRMergeState` + `ReviewThreads` を入れて `Issue()` / `PR()` を呼び、`Situation` を突き合わせる。表に無い alias、表に無い `issue-<n>` / `pr-<n>`、fixture に無い表の項目はそれぞれ名前を含むメッセージで `t.Errorf` する（黙って通さない）
- [ ] 5.2 s04 で採取した `<alias>` の期待値を `expected` に追加する（s04 の 5.2〜5.4 が済んでいる場合。済んでいなければ、この項目を s04 完了後に行う旨を PR / コミットメッセージに書き、`example` だけで 5.1 を通す）。各 issue / PR について human-turn-signals.md の判定表と「キューに入れないもの」を fixture の JSON（ラベル・本文 1 行目・コメント末尾のマーカー・`mergeable` / `statusCheckRollup`・review thread）に手で当てて局面を書く。分類器の出力を写さない。表と実装が食い違ったら、文書を読み直してどちらが誤りかを判断し、実装の誤りなら直す

## 6. 最終確認

- [x] 6.1 `gofmt -l .` が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する（`golangci-lint run ./...` も s01 の CI が回すので通す）
