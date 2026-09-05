## MODIFIED Requirements

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
