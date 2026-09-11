## MODIFIED Requirements

### Requirement: 本文領域は Issue 本文・最新 blocked-by の要約・コメント時系列を出す
カード詳細画面の `View` は本文領域に、詳細の対象の `Card.Issue` について MUST 次を上から順に出す。
1. `Issue.Body` を Glamour で Markdown レンダリングした文字列（幅は端末の幅。s08 のプレビューと同じ方法。レンダリングが失敗したら `Body` をそのまま出す。空なら何も出さない）
2. `model.LatestBlockedBy(Issue.Comments)` が見つかれば、セクションの見出し行 `blocked-by` に続けて `blocked-by: <value>` の 1 行。`value` が `human` なら、そのコメントに `model.UnblockWhen` の値があれば続けて `unblock-when: <値>` の 1 行を出し（上流 `routine-common`「解除条件を `unblock-when:` の 1 行で明示する」。`comment` は答えのコメントで解け、`docs` は方針文書の更新が要り、`#m` はその issue / PR の完了が要る。人は何をすれば動き出すかをこの行で知る）、さらに `model.ParseQuestions(そのコメントの Body)` の各質問を `Q<Number>. <Title>` の行と、その選択肢を `  <Letter>: <Text>`（`Recommended` なら `  <Letter>（推奨）: <Text>`）の行で出す（mvp.md「`human` なら『人に何を決めてほしいか』の選択肢と推奨がここにある」）。質問が 1 件もパースできなければ、そのコメントの `Body` から routine マーカー行と `blocked-by:` 行と `unblock-when:` 行を除いた本文をそのまま出す。見つからなければこの部分（見出し行を含む）を出さない
3. セクションの見出し行 `コメント` に続けてコメント時系列: `Issue.Comments` が nil なら `コメント: 取得失敗`（s20 は全 issue のコメントを取るので、nil は取得に失敗したことを意味する）、長さ 0 なら `コメント: なし`、1 件以上なら Requirement「routine コメントは折りたたみ、x で展開する」の書式で並び順（末尾が最新）に出す

セクションの見出し行の書式と出す条件は、Requirement「本文領域のセクションは見出し行で区切る」が定める。

#### Scenario: issue 108 の本文とコメント
- **WHEN** `example` の issue 108 の Card の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** 本文の `認証まわりの仕様を決めたい` が含まれ、`blocked-by:` は含まれず（`example` に `blocked-by:` 行が無い）、`コメント: 取得失敗` と `コメント: なし` は含まれない

#### Scenario: blocked-by human の要約に選択肢と推奨が出る
- **WHEN** `Labels` が `stage:propose` と `blocked` と `question`、`Comments` が AI の 1 件（`Body` が `<!-- routine -->\nblocked-by: human\n## Q1. 名前での絞り込みを含めるか\n- 選択肢 A（推奨）: 含めない\n- 選択肢 B: 含める`）の issue の Card の詳細を開き、`View` を読む
- **THEN** `blocked-by: human`、`Q1. 名前での絞り込みを含めるか`、`A（推奨）: 含めない`、`B: 含める` がこの順で含まれる

#### Scenario: unblock-when が blocked-by の次の行に出る
- **WHEN** 上と同じで `Comments` の `Body` が `<!-- routine -->\nblocked-by: human\nunblock-when: docs\n## Q1. 認可の方針をどこに書くか\n- 選択肢 A（推奨）: docs/policy.md` の Card の詳細を開き、`View` を読む
- **THEN** `blocked-by: human`、`unblock-when: docs`、`Q1. 認可の方針をどこに書くか` がこの順で含まれる

#### Scenario: 見出しの無い blocked-by コメントは本文をそのまま出す
- **WHEN** 上と同じで `Comments` の `Body` が `<!-- routine -->\nblocked-by: human\nunblock-when: comment\n次の方針をコメントで教えてください` の Card の詳細を開き、`View` を読む
- **THEN** `blocked-by: human`、`unblock-when: comment`、`次の方針をコメントで教えてください` がこの順で含まれる（要約に出した `unblock-when:` 行は、続けて出す本文からは除く）

#### Scenario: コメントが 0 件の issue
- **WHEN** `example` を `Fetch` して得た issue 140 の Card の詳細を開き、`View` を読む（s20 で全 issue のコメントを取るので、`issue-140.json` のコメント 0 件が長さ 0 の非 nil で入る）
- **THEN** 本文の `起動時に設定ファイルが無いと落ちる` と `コメント: なし` が含まれ、`コメント: 取得失敗` と `▌` は含まれない

#### Scenario: blocked-by とコメントの手前に見出し行が出る
- **WHEN** 幅 100・高さ 40 のサイズメッセージを与えた後、`Comments` が AI の 1 件（`Body` が `<!-- routine -->\nblocked-by: human\nunblock-when: comment\n方針を教えてください`）である issue の Card の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `── blocked-by ` で始まる行、`blocked-by: human` の行、`── コメント ` で始まる行がこの順で含まれ、`── blocked-by ` の行と `── コメント ` の行の表示幅はどちらも 100 である

### Requirement: PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す
PR 詳細画面の `View` は、対象の `model.PR` について MUST 次を上から順に出す。ヘッダ領域は 1〜2、本文領域は 3〜8（Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」）。
1. タイトル行。接頭辞 `<Repo> PR#<Number>  ` に `Title` を続け、Requirement「ヘッダはリポジトリ・番号・いま人が何をすべきか・現在の段階・バッジ・depends on を出す」が定めたタイトル行の規則（端末の幅で折り返して全文出し、継続行を接頭辞の表示幅ぶん字下げする）に MUST 従う
2. `[<段階>] <状態>  labels: <Labels を空白区切り>`（段階と状態は Requirement「紐づく PR 一覧は段階順に 1 行ずつ出し、選択中の PR に印を付ける」と同じ表記）
3. 1 行目判定: `model.ParseUndecided(Body)` が `n, true` なら `未確定の判断: <n> 件`、false なら `1 行目に未確定の判断が無い`
4. 紐づけ: s07 の `fetch.LinkedIssue(Title, Body)` が `n, true` なら `紐づく issue: #<n>`、false なら `紐づく issue: なし`（mvp.md「`Refs #n` / `Closes #n`」）
5. checks と merge 状態: `MergeState` が nil なら `checks: 取得失敗` の 1 行（s20 は全 PR の merge 状態を取るので、nil は取得に失敗したことを意味する）。non-nil なら `mergeable: <Mergeable> <MergeStateStatus>` の 1 行に続けて、字下げの無い checks の見出し 1 行を出す。見出しは `StatusCheckRollup` が 0 件なら `checks: なし`、1 件以上なら `classify.ChecksGreen(MergeState)` が true で `checks: 緑`、false で `checks: 緑以外`（s02 `human-turn-classify` が定める関数をそのまま使う。merge の確認画面 s14 と同じ語にする）。1 件以上なら見出しに続けて `StatusCheckRollup` の各要素を 1 行ずつ（`Typename` が `CheckRun` なら `  <Name>: <Conclusion>`（`Conclusion` が空なら `<Status>`）、`StatusContext` なら `  <Context>: <State>`）。各要素の行には Requirement「checks の行は成功を緑・失敗を赤で出す」が定める色を付ける
6. セクションの見出し行 `本文` に続けて、`Body` を Glamour でレンダリングした文字列（Issue 本文と同じ扱い）
7. セクションの見出し行 `コメント` に続けて会話コメント: `Comments` が nil なら `コメント: 取得失敗`、長さ 0 なら `コメント: なし`、1 件以上なら Requirement「routine コメントは折りたたみ、x で展開する」の書式
8. セクションの見出し行 `review thread` に続けて review thread: `ReviewThreads` が nil なら `review thread: 取得失敗`（s20 は全 PR の review thread を取るので、nil は取得に失敗したことを意味する）、長さ 0 なら `review thread: なし`、1 件以上なら `IsResolved` が false の thread を先に、true の thread を後に（それぞれ元の順を保つ）並べ、各 thread を見出し `thread 未 resolve` / `thread resolved` の 1 行と、その `Comments` の各件（Requirement「routine コメントは折りたたみ、x で展開する」の展開時の書式。`model.IsAI(Body)` が true なら `▌AI  HH:MM` と `▌` 付きの全行、false なら `<Author.Login>  HH:MM` と全行。折りたたまない）で出す。thread の選択と `A` 返信は s16 が担当する

セクションの見出し行の書式と出す条件は、Requirement「本文領域のセクションは見出し行で区切る」が定める。

PR 詳細画面で `g` は、カードに `Issue` があればカード詳細画面に移る（`Issue` が nil なら何もしない）。カード詳細画面で `g` は、選択中の PR の PR 詳細画面に移る（`Enter` と同じ。`PRs` が空なら何もしない）。カード詳細画面で `Enter` は選択中の PR の PR 詳細画面に移る（mvp.md「PR を選んで Enter で PR の会話・review thread へ入る」。`PRs` が空なら何もしない）。

#### Scenario: PR 131 の詳細
- **WHEN** `example` の issue 108 のカード詳細で `Enter` を与え（PR 131 が選択中）、`View` から ANSI エスケープを除いて読む
- **THEN** `org/app PR#131`、`[propose] open  labels: propose question`、`1 行目に未確定の判断が無い`、`紐づく issue: #108`、`mergeable: UNKNOWN BLOCKED`、`checks: 緑以外`、`test: SUCCESS`、`ci/legacy: PENDING`、`── 本文 `、本文の `issue #108 の提案`、`── コメント `、`▌AI  19:31  Q1: マイグレーションを分けますか。  (+0 行)`、`── review thread `、`thread 未 resolve` がこの順で含まれ、`checks: 取得失敗` と `review thread: 取得失敗` は含まれない（s20 で全 PR の merge 状態と review thread を取るので、`example` の PR 131 はどちらも埋まる）

#### Scenario: 長い PR タイトルは折り返して全文出す
- **WHEN** 幅 40・高さ 40 のサイズメッセージを与えた後、`Repo` が `org/app`、`Number` が 131、`Title` が表示幅 60 の PR の詳細を開き、`View` から ANSI エスケープを除いて読む。タイトル行は 1 行目と、それに続く行頭が `org/app PR#131  ` と同じ表示幅の空白である行とする
- **THEN** タイトル行は 2 行以上あり、各行から接頭辞と行頭の空白を取り除いて連結した文字列が `Title` と一致し、どのタイトル行の表示幅も 40 以下で、`…` で終わるタイトル行は無い

#### Scenario: 全角文字の PR タイトルは表示幅で折り返す
- **WHEN** 幅 40・高さ 40 のサイズメッセージを与えた後、`Repo` が `org/app`、`Number` が 131、`Title` が全角文字 40 字の PR の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** 上の Scenario と同じ手順で取り出したタイトル行の連結が `Title` と一致し、どのタイトル行の表示幅も 40 を超えない

#### Scenario: labels 行は今までどおり幅で切り詰める
- **WHEN** 幅 40・高さ 40 のサイズメッセージを与えた後、`Labels` を連ねた labels 行の表示幅が 60 になる PR の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `[` で始まる labels 行の表示幅は 40 以下で、末尾が `…` である

#### Scenario: 未 resolve の thread が先頭に出る
- **WHEN** `Labels` が `apply`、`ReviewThreads` が `[{IsResolved: true, Comments: [{user-3, "直しました"}]}, {IsResolved: false, Comments: [{user-1, "<!-- routine -->\nこの分岐は残しますか"}]}]` の順、`MergeState` が `{Mergeable: MERGEABLE, MergeStateStatus: CLEAN, StatusCheckRollup: [CheckRun test SUCCESS, StatusContext ci/legacy PENDING]}` の open PR の詳細を開き、`View` を読む
- **THEN** `mergeable: MERGEABLE CLEAN`、`checks: 緑以外`、`test: SUCCESS`、`ci/legacy: PENDING` がこの順で含まれ、`thread 未 resolve` の行が `thread resolved` の行より前にあり、`この分岐は残しますか` の行は `▌` で始まり、`直しました` の行は `▌` で始まらない

#### Scenario: checks が無い merge 状態
- **WHEN** `MergeState` が `{Mergeable: UNKNOWN, StatusCheckRollup: []}`、`ReviewThreads` が空（長さ 0）、`Comments` が空の PR の詳細を開き、`View` を読む
- **THEN** `mergeable: UNKNOWN`、`checks: なし`、`コメント: なし`、`review thread: なし` が含まれ、`checks: 緑` と `checks: 緑以外` は含まれない

#### Scenario: 全部の checks が成功なら見出しは緑になる
- **WHEN** `MergeState` が `{Mergeable: MERGEABLE, MergeStateStatus: CLEAN, StatusCheckRollup: [CheckRun test SUCCESS, CheckRun lint SKIPPED]}` の PR の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `checks: 緑` が含まれ、`checks: 緑以外` は含まれない

#### Scenario: 詳細の取得に失敗した PR
- **WHEN** `MergeState` と `Comments` と `ReviewThreads` がいずれも nil である手書きの open PR の詳細を開き、`View` を読む
- **THEN** `checks: 取得失敗`、`コメント: 取得失敗`、`review thread: 取得失敗` がこの順で含まれ、`checks: なし` と `コメント: なし` と `review thread: なし` は含まれない

#### Scenario: g で issue と PR を行き来する
- **WHEN** `example` の issue 108 のカード詳細を開いた `Model` に `g`、`g`、`Enter`、`Esc` の順で与える
- **THEN** 画面は順に PR 詳細（PR 131）、カード詳細、PR 詳細、カード詳細になる

#### Scenario: PR 単独のカードで g は何もしない
- **WHEN** `Issue` が nil の Card から直接開いた PR 詳細の `Model` に `g` を与える
- **THEN** 画面は PR 詳細のままで、コマンドは返らない

## ADDED Requirements

### Requirement: 本文領域のセクションは見出し行で区切る
カード詳細画面と PR 詳細画面の本文領域は、セクションの切り替わりに MUST 見出し行を 1 行置く。見出し行は
`── <名前> ` に、行の表示幅が本文領域の幅（Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」が
定める左ペインの幅）に達するまで `─` を継ぎ足した 1 行とする。`── <名前> ` がその幅に収まらないときは、
表示幅で切って `…` を付けない（切った跡が線の一部に見えるようにする）。

見出し行は、そのセクションより上に本文領域の行が 1 行以上あるときにだけ出す。本文領域の 1 行目が見出し行になると、
直上にあるヘッダとの区切り線と 2 行続いて、どちらがどの境目か読めなくなるためである。

セクションの名前は、カード詳細が `blocked-by` / `コメント`、PR 詳細が `本文` / `コメント` / `review thread` とする
（各セクションの中身は上の 2 つの Requirement が定める）。1 つ目のセクションには名前を与えない。1 つ目のセクションとは、
カード詳細では Issue 本文を指し、PR 詳細では 1 行目判定から checks までを指す。見出し行に色は付けない。

#### Scenario: PR 詳細の見出し行は本文領域の幅いっぱいに引かれる
- **WHEN** 幅 100・高さ 40 のサイズメッセージを与えた後（幅 100 は 120 未満なので 1 ペインである）、`example` の issue 108 のカード詳細から PR 131 の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `── 本文 ` で始まる行、`── コメント ` で始まる行、`── review thread ` で始まる行がこの順で含まれ、どの行も表示幅が 100 で、`── ` と名前の後は `─` だけが並ぶ

#### Scenario: 2 ペインの見出し行は左ペインの幅に収まる
- **WHEN** 幅 140・高さ 40 のサイズメッセージを与えた後（幅 140 は 120 以上なので 2 ペインで、左ペインの幅は 99 である）、`example` の issue 108 のカード詳細から PR 131 の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `── 本文 ` で始まる行の、縦の区切り線 `│` より左の部分の表示幅は 99 である

#### Scenario: 本文領域の 1 行目には見出し行を出さない
- **WHEN** 幅 100・高さ 40 のサイズメッセージを与えた後、`Body` が空文字列で `Comments` が長さ 0、`blocked-by:` を持つコメントが無い issue の Card の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** 本文領域の 1 行目は `コメント: なし` であり、`── コメント ` で始まる行は含まれない

### Requirement: checks の行は成功を緑・失敗を赤で出す
PR 詳細画面の本文領域が出す `StatusCheckRollup` の各要素の行は、その状態（`CheckRun` なら `Conclusion`、
`Conclusion` が空なら `Status`。`StatusContext` なら `State`）に応じて MUST 行全体に次の色を付ける。
キューの表が種別ごとの色を行全体に付けているのと同じ形にする（mvp.md「行の色は種別で固定」）。

| 区分 | 状態 | 色 |
| --- | --- | --- |
| 成功 | `SUCCESS` / `SKIPPED` / `NEUTRAL` | 緑 `#0E8A16` |
| 失敗 | `FAILURE` / `TIMED_OUT` / `CANCELLED` / `ACTION_REQUIRED` / `STARTUP_FAILURE` / `ERROR` | 赤 `#B60205` |
| それ以外 | 上のどちらにも無い状態（`QUEUED` / `IN_PROGRESS` / `PENDING` / `EXPECTED` / 空 など） | 色を付けない |

成功の 3 つは s02 `human-turn-classify` の `ChecksGreen` が緑とみなす集合と同じにする。表に無い状態に色を付けないのは、
GitHub が状態を増やしたときに、その状態を黙って成功や失敗に見せないためである。色は状態の文字を読まなくても
赤を見つけられるようにする補助であり、状態の語は今までどおり文字で出す。

#### Scenario: 失敗した check の行だけが赤になる
- **WHEN** `MergeState` が `{Mergeable: MERGEABLE, MergeStateStatus: UNSTABLE, StatusCheckRollup: [CheckRun static SUCCESS, CheckRun tests FAILURE, CheckRun image (Conclusion 空、Status IN_PROGRESS)]}` の PR の詳細を開き、`View` を ANSI エスケープごと読む
- **THEN** `static: SUCCESS` の行には緑 `#0E8A16` の指定があり、`tests: FAILURE` の行には赤 `#B60205` の指定があり、`image: IN_PROGRESS` の行には色の指定が無い

#### Scenario: 色を落としても状態の語は読める
- **WHEN** 上と同じ PR の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `static: SUCCESS`、`tests: FAILURE`、`image: IN_PROGRESS` がこの順で含まれる
