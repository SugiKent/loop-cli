## MODIFIED Requirements

### Requirement: ヘッダはリポジトリ・番号・いま人が何をすべきか・現在の段階・バッジ・depends on を出す
カード詳細画面の `View` はヘッダ領域の先頭に、詳細の対象の Card について MUST 次の行をこの順で出す（mvp.md「カード詳細」のヘッダ）。行番号は固定せず順序だけを定める（s17 が ADDED で行を足せるようにするため）。方式は `Options.Modes`（設定由来。`queue-screen`「リポジトリごとの運用方式を設定から受け取る」）に対象の `Repo` を引いて決め、Card の中の値からは決めない。
- `<Repo> #<Number>  <Title>`（`Card.Issue` の値）
- `Card.Result.Summary`（空なら出さない。s05 が定めた「いま人が何をすべきか」。mvp.md「カードは『いま人が何をすべきか』を 1 行目に出す」。s08 のプレビューはこれを出さず、この change のヘッダに委ねている）
- `段階: ` に続けて `model.IssueStages(mode, Issue.Labels)` の段階ラベルを空白区切りで並べる。1 件も無ければ `段階なし`。2 件以上あればすべて並べる（異常の状態を隠さない）。続けてバッジを出す。`sdd`（ゼロ値を含む）なら `[blocked]` / `[wip]` / `[question]` の順、`label` なら `[blocked]` / `[question]` の順で、`Labels` にあるものだけ出す（`label` の方式に `wip` ラベルは無く、作業中は段階ラベル `In Progress` が示す）
- `Issue.Body` の中に `depends on #<n>`（大文字小文字を区別しない。`<n>` は 10 進整数）が 1 つ以上あれば `depends on: #<n> #<m> …` を出現順に出す。無ければこの行を出さない（書式は design.md 未決事項の既定値）

1 行目 `<Repo> #<Number>  <Title>` は**タイトル行**である。タイトル行は `<Repo> #<Number>  ` を接頭辞として `Title` を続け、表示幅が端末の幅を超えれば端末の幅で MUST 折り返し、`Title` を全文出す。折り返して生まれた継続行の行頭には接頭辞と同じ表示幅の空白を置き、`Title` の開始位置に縦を揃える。幅の判定と折り返し位置は表示幅（`ansi.StringWidth` が返す値）で決め、全角文字と絵文字を含むタイトルでも桁がずれないようにする。端末の幅から接頭辞の表示幅を引いた残りが 2 列未満になるとき（全角 1 文字が入らない幅）は、空白を置かずに端末の幅で折り返す。折り返し位置にあった空白 1 個は改行に置き換わって消える（`ansi.Wrap` の仕様。design.md D1）ので、行を連結して `Title` と比べるときは空白を除いて比べる。タイトル行とその継続行は、いずれも表示幅が端末の幅を超えない。
この規則は PR 詳細画面のタイトル行（Requirement「PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す」の 1）にも同じく適用する。接頭辞が `<Repo> PR#<Number>  ` に変わるだけである。

ヘッダ領域のうちタイトル行と継続行を除く各行は、表示幅が端末の幅を超えれば端末の幅に切り詰めて末尾を `…` にする（design.md 未決事項の既定値）。対象はカード詳細の `Card.Result.Summary` / `段階` / `depends on` の行、Requirement「紐づく PR 一覧は段階順に 1 行ずつ出し、選択中の PR に印を付ける」の PR 一覧の行、PR 詳細の labels 行である。
「この段階に入ってからの経過時間」と段階の変遷タイムラインはデータ源が REST の timeline（D-001「ラベル変遷」。`GHClient.LabelTimeline`）であり、s17 が担当する。この change はヘッダに枠を確保せず、s17 が ADDED で行を足す。

#### Scenario: issue 108 のヘッダ
- **WHEN** 幅 120・高さ 40 のサイズメッセージを与えた後、`example` の issue 108 の Card（`Labels` が `stage:propose` と `question`、`Card.Result.Summary` が `PR #131 の質問に答える`）の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `org/app #108`、issue 108 の `Title`、`PR #131 の質問に答える`、`段階: stage:propose`、`[question]` がこの順で含まれ、`[blocked]` と `[wip]` と `depends on:` は含まれない

#### Scenario: 長いヘッダ行は幅で切り詰める
- **WHEN** 幅 40・高さ 40 のサイズメッセージを与えた後、`Card.Result.Summary` が表示幅 60 である issue の Card の詳細を開き、`View` から ANSI エスケープを除いて読む
- **THEN** `Summary` の行の表示幅は 40 以下で、末尾が `…` である

#### Scenario: 長い Issue タイトルは折り返して全文出す
- **WHEN** 幅 40・高さ 40 のサイズメッセージを与えた後、`Repo` が `org/app`、`Number` が 108、`Title` が表示幅 60 の issue の Card の詳細を開き、`View` から ANSI エスケープを除いて読む。タイトル行は 1 行目と、それに続く行頭が `org/app #108  ` と同じ表示幅の空白である行とする
- **THEN** タイトル行は 2 行以上あり、各行から接頭辞と行頭の空白を取り除いて連結した文字列が `Title` と一致し、どのタイトル行の表示幅も 40 以下で、`…` で終わるタイトル行は無い

#### Scenario: 段階ラベルが無い issue と depends on
- **WHEN** `Labels` が空、`Body` が `集計が遅い。\n\ndepends on #12\nDepends on #34` の issue の Card の詳細を開き、`View` を読む
- **THEN** `段階なし` と `depends on: #12 #34` が含まれる

#### Scenario: 段階ラベルが 2 つある issue は両方出す
- **WHEN** `Labels` が `stage:propose` と `stage:apply` と `blocked` と `wip` の issue の Card の詳細を開き、`View` を読む
- **THEN** `段階: stage:propose stage:apply` と `[blocked]` と `[wip]` がこの順で含まれる

#### Scenario: label 方式の issue の段階とバッジ
- **WHEN** `Options.Modes` が `org/board` を `label` にした `Model` で、`org/board` の `Labels` が `In Progress` と `question` の issue の Card の詳細を開き、`View` を読む
- **THEN** `段階: In Progress` と `[question]` が含まれ、`[wip]` は含まれない

#### Scenario: label 方式では sdd の段階ラベルを段階行に出さない
- **WHEN** 同じ `Model` で、`org/board` の `Labels` が `stage:propose` の issue の Card の詳細を開き、`View` を読む
- **THEN** `段階なし` が含まれる

### Requirement: PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す
PR 詳細画面の `View` は、対象の `model.PR` について MUST 次を上から順に出す。ヘッダ領域は 1〜2、本文領域は 3〜8（Requirement「詳細の本文領域はスクロールし、ヘッダ領域は固定する」）。
1. タイトル行。接頭辞 `<Repo> PR#<Number>  ` に `Title` を続け、Requirement「ヘッダはリポジトリ・番号・いま人が何をすべきか・現在の段階・バッジ・depends on を出す」が定めたタイトル行の規則（端末の幅で折り返して全文出し、継続行を接頭辞の表示幅ぶん字下げする）に MUST 従う
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
カード詳細画面と PR 詳細画面は、端末の幅と高さ（s08 の `Model` が保持する値）の全体を使う 1 ペインで MUST 描く（キュー画面の表とプレビューの 2 ペインは詳細に適用しない）。上からヘッダ領域（カード詳細: ヘッダ行と PR 一覧。PR 詳細: タイトル行と labels 行）、区切り線、本文領域、フッタの順で、ヘッダ領域と区切り線とフッタは常に表示され、本文領域だけが残りの高さに収まらない分をスクロールする。
本文領域は Bubbles の viewport で描き、`j` / `↓` で 1 行下へ、`k` / `↑` で 1 行上へ、`PgDn` / `PgUp` で本文領域の高さぶん動く（スクロールキーは mvp.md に無く、design.md 未決事項の既定値）。先頭と末尾で止まる。キュー画面から開いたとき、および PR 詳細とカード詳細を行き来したとき、スクロール位置は先頭に戻る。
本文領域の高さは端末の高さからヘッダ領域の行数と区切り線 1 行とフッタ 1 行を引いた値で、下限は 1 行とする。1 行を下回るときはカード詳細の PR 一覧を末尾から落として（`なし` の行を含む）本文領域 1 行を確保する（design.md 未決事項の既定値）。
折り返したタイトル行が高さを押し出す場合は、PR 一覧を落とした後にタイトル行を末尾から落とし、残った最後のタイトル行の末尾を `…` にする。タイトル行の 1 行目は落とさない（どの Issue / PR を見ているのかが分からなくなるため）。この規則により、**タイトルの折り返しを原因として**画面の行数が端末の高さを超えることは MUST 無い（タイトルが 1 行に収まるときの行数が既に端末の高さを超えている場合は、この change の範囲外であり従来どおりとする）。
フッタの左はその画面で動く操作キーのヒント（カード詳細: `Esc 戻る  Tab PR 選択  Enter PR を開く  x 展開  g PR へ  ? ヘルプ  u URL  a 回答  t todo  m merge  n 新規  o ブラウザ  q 終了`（表示幅 125 列）。`PRs` が空なら `Tab PR 選択  Enter PR を開く  g PR へ` と `m merge` を省く（`m` は選択中の PR が無ければ動かない。`n` は PR の有無によらず動くので省かない）。PR 詳細: `Esc 戻る  x 展開  g issue へ  ? ヘルプ  u URL  a 回答  m merge  n 新規  o ブラウザ  q 終了`（表示幅 90 列）。`? ヘルプ` を `a 回答` の前に置くのは、カード詳細で末尾に置くと幅 80 で切れてヘルプの入口が見えないため（`? ヘルプ` は 65 列目で終わる。design.md 未決事項の既定値）。`t` の振る舞いは s11 `todo-toggle` が定め、PR 詳細では `t` が何もしないのでヒントを出さない。`m` の振る舞いは s14 `merge-pr` が定め、`a 回答` の次に置く。`n` の振る舞いは s15 `new-issue` が定め、mvp.md キーバインド表の順で `m merge` の次（`o ブラウザ` の前）に置く。これで PR 詳細のヒントは 90 列になり、既定幅 80 の端末では末尾の `q 終了` と `o ブラウザ` が切れる（切れても動く。キュー画面のフッタと同じ判断で、キーを隠すよりキーを出すことを採った。s14 design.md）。`o` / `?` は s12 `browse-open` / `help-screen` が定める。`u` は s22 `url-picker` が定め、`? ヘルプ` の直後に置くのは幅 80 の端末で URL 一覧の入口を見せるためである（`u URL` は 72 列目で終わる。s22 design.md）。s11 までにあった `j/k スクロール` は、キュー画面のフッタ（s12 の `queue-screen` MODIFIED）と同じく移動系のキーとして `?` のヘルプに委ね、ヒントから外す）、右は s08 と同じステータス（スピナー / エラー）とする。ヒントが端末幅に収まらないときの切り詰めは s08 のフッタの規則のままである（カード詳細のヒントは幅 124 以下の端末で末尾から切れる。design.md）。

#### Scenario: 本文が高さを超えると j でスクロールする
- **WHEN** `Body` が `行01` から `行60` までの 60 段落（各段落を空行で区切る）である issue の Card の詳細を、幅 100・高さ 20 の `Model` で開き、`View` を読んでから `j` を 5 回与えて `View` を読む
- **THEN** 1 回目はヘッダ行 `org/app #<n>` と `行01` を含み `行30` を含まない。2 回目はヘッダ行を含んだまま `行01` を含まず `行06` を含む

#### Scenario: PR 詳細に移るとスクロール位置が先頭に戻る
- **WHEN** 上の `Model`（`j` を 5 回与えた後。Card の `PRs` に open PR が 1 件ある）に `Enter` を与えて PR 詳細に移り、`Esc` で戻って `View` を読む
- **THEN** `行01` を含む

#### Scenario: 詳細画面のフッタ
- **WHEN** 幅 130・高さ 40 のサイズメッセージを与えてカード詳細を開いた `Model` の `View` から ANSI エスケープを除いて読み、`Enter` で PR 詳細に移って再び読む
- **THEN** 1 回目の最終行に `Esc 戻る`、`Tab PR 選択`、`x 展開`、`? ヘルプ`、`u URL`、`a 回答`、`t todo`、`m merge`、`n 新規`、`o ブラウザ`、`q 終了` が含まれ `1-4/Tab タブ` と `R 更新` と `j/k スクロール` は含まれない。2 回目の最終行に `Esc 戻る` と `g issue へ` と `? ヘルプ` と `u URL` と `a 回答` と `m merge` と `n 新規` と `o ブラウザ` と `q 終了` が含まれ `Tab PR 選択` と `t todo` と `R 更新` は含まれない

#### Scenario: PR の無いカードのフッタには PR のキーを出さない
- **WHEN** `example` の issue 140 の Card（`PRs` 空）の詳細を幅 130・高さ 40 で開き、`View` から ANSI エスケープを除いて読む
- **THEN** 最終行に `Esc 戻る` と `x 展開` と `t todo` と `n 新規` と `o ブラウザ` が含まれ、`Tab PR 選択`、`Enter PR を開く`、`g PR へ`、`m merge` は含まれない

#### Scenario: 低い端末では PR 一覧を削って本文 1 行を残す
- **WHEN** `Body` が `行01` の 1 行で、`PRs` が `propose` / `apply` / `archive` の open PR 3 件の issue の Card の詳細を、幅 130・高さ 7 の `Model` で開き、`View` から ANSI エスケープを除いて読む
- **THEN** ヘッダ行 `org/app #<n>` と本文の `行01` とフッタの `q 終了` が含まれ、`[archive] PR#` の行は含まれない

#### Scenario: 折り返した PR タイトルが高さを埋めても画面は端末の高さに収まる
- **WHEN** `Repo` が `org/app`、`Number` が 131、`Title` が表示幅 300、`Body` が `行01` の 1 行、`Labels` が `apply` 1 件、`Comments` と `ReviewThreads` が長さ 0、`MergeState` が `{Mergeable: UNKNOWN, StatusCheckRollup: []}` の PR の詳細を、幅 40・高さ 8 の `Model` で開き、`View` から ANSI エスケープを除いて読む
- **THEN** 行数はちょうど 8 で、1 行目は `org/app PR#131` で始まり、最後のタイトル行の末尾は `…` であり、labels 行と区切り線とフッタが含まれる（フッタのヒントは 90 列なので幅 40 では末尾が切れる。この Requirement が定めたとおりで、先頭の `Esc 戻る` で在ることを確かめる）

#### Scenario: 狭い端末でも詳細は 1 ペインで全部出す
- **WHEN** 幅 60・高さ 40 の `Model` でカード詳細を開き、`View` を読む
- **THEN** ヘッダ行 `org/app #108` と PR 一覧の `PR#131` と本文の `認証まわりの仕様を決めたい` がすべて含まれ、区切り線はちょうど 1 本で、`org/app #108` と `PR#131` はその上、`認証まわりの仕様を決めたい` はその下にある（ヘッダ領域 / 区切り線 / 本文領域 の 1 ペイン）
