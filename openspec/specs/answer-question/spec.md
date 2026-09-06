# answer-question Specification

## Purpose
TBD - created by archiving change s10-answer-question. Update Purpose after archive.
## Requirements
### Requirement: a は画面の対象を決めて回答テンプレートを入れたエディタを開く
`internal/ui` はエディタ起動の型 `Editor`（`func(initial string) tea.Cmd`。返すコマンドは、編集後の本文またはエラーを運ぶ `internal/ui` 内のメッセージを返す）を MUST 公開し、`Model` は `New` で受け取った `Editor` と `gh.GHClient` を保持する（`New` の引数は s08 `queue-screen`「Model は Card をタブ別に並べ、選択行を 1 つ持つ」の MODIFIED を見る）。
`Update` は `a` を画面の状態ごとに MUST 次のとおり扱う。回答の対象 `action.Target` と表示名（`<Repo> PR#<n>` または `<Repo> #<n>`。s09 の詳細ヘッダと同じ表記）を決め、対象の `Comments` から `action.AnswerTemplate` でテンプレートを作り、`Editor(テンプレート)` が返すコマンドをそのまま（他のコマンドとまとめずに）返す。
- キュー画面: 選択行の主体（s08 `Subject`。`isPR` が対象の `IsPR`）。選択行が無ければ何もしない
- カード詳細画面: 詳細の対象の `Card.Issue`（`IsPR` は false）
- PR 詳細画面: 詳細の対象の PR（`IsPR` は true）
対象に `question` ラベルが付いているかどうかは問わない（mvp.md キーバインド表の `a` は「回答・コメント」であり、`question` 無しの対象へのコメントを禁じる記述が docs に無い。design.md の未決事項）。対象の `Comments` が nil（s20 以降は取得に失敗したときだけ起きる）ならテンプレートは空である。書き込み（投稿・ラベル切り替え）の結果を待っている間（Requirement「投稿の結果をステータスに出し、再取得しない」、s11 `todo-toggle`「書き込み中は t と a を受け付けない」）は `a` / `t` を無視する。
エディタが動いている間、`Model` はキー入力を受けない（端末はエディタが使う。Bubble Tea の外部プロセス実行の仕組みに従う）。

#### Scenario: キュー画面で a を押すと主体のテンプレートでエディタが開く
- **WHEN** 主体が PR 131（`Repo` `org/app`、`Comments` が `AI` true で `Body` `<!-- routine -->\n## Q1. 分けるか\n- 選択肢 A（推奨）: 分ける\n- 選択肢 B: 分けない` の 1 件）の Card を今やるタブに持ち、渡された `initial` を記録するスタブ `Editor` で `New` した `Model` に `a` を与える
- **THEN** コマンドが返り、それを実行するとスタブは `initial` として `Q1: A` を受け取り、`Model` が保持する回答の対象は `Target{Repo: "org/app", Number: 131, IsPR: true}` である

#### Scenario: 0 行のタブで a は何もしない
- **WHEN** `example` の `Result` を渡して異常タブ（0 行）に切り替えた `Model` に `a` を与える
- **THEN** コマンドは返らず、画面はキューのままである

#### Scenario: カード詳細では Issue、PR 詳細ではその PR が対象になる
- **WHEN** `example` の issue 108 の Card（主体は PR 131）のカード詳細を開いた `Model` に `a` を与え、別に同じカードから `Enter` で PR 詳細を開いた `Model` に `a` を与える
- **THEN** 1 つ目の対象は `Target{Repo: "org/app", Number: 108, IsPR: false}`、2 つ目の対象は `Target{Repo: "org/app", Number: 131, IsPR: true}` であり、どちらもスタブに渡る `initial` は空文字列である（`example` のコメントは `## Q1.` の見出し形式ではない）

#### Scenario: question の無い issue にも a でコメントできる
- **WHEN** `example` の `Result` を渡してバックログタブ（issue 140。`Labels` 空。s20 で全 issue のコメントを取るので `Comments` は長さ 0 の非 nil）を選んだ `Model` に `a` を与える
- **THEN** コマンドが返り、対象は `Target{Repo: "org/app", Number: 140, IsPR: false}`、スタブに渡る `initial` は空文字列である

### Requirement: ExternalEditor は設定のエディタを一時ファイルで開く
`internal/ui` は `ExternalEditor(command string) Editor` を MUST 公開する。`command` は config の `editor`（s02 が `$EDITOR` を展開済み）で、`cmd/loop-cli` がこれを `New` に渡す。返る `Editor` は次の振る舞いをする。
- `command` の前後の空白を除いた結果が空なら、外部プロセスを起動せず、`editor が設定されていません（config の editor か環境変数 EDITOR）` を含むエラーを運ぶメッセージを返すコマンドを返す（s02 `config-loading`「空のときの扱いは s10 が担当」。既定値は design.md の未決事項）
- それ以外は、一時ディレクトリに `loop-cli-answer-*.md` の一時ファイルを作って `initial` を書き込み、`command` を空白で分割した先頭を実行ファイル、残りを引数とし、その末尾に一時ファイルのパスを足したプロセスを、Bubble Tea の外部プロセス実行の仕組み（描画を止め、端末の標準入出力をプロセスに渡す）で起動する。シェルを通さない
- プロセスが終了コード 0 で終わったら一時ファイルを読み、その内容を編集後の本文としてメッセージで返し、一時ファイルを削除する。終了コードが非 0 か起動に失敗したら、そのエラーを運ぶメッセージを返し、一時ファイルは削除する

#### Scenario: editor が空なら起動せずエラーになる
- **WHEN** `ExternalEditor("")` が返す `Editor` を `initial` `Q1: A` で呼び、返ったコマンドを実行する
- **THEN** 返るメッセージはエラーを持ち、エラー文字列に `editor` を含む

#### Scenario: コマンドの分割と一時ファイル
- **WHEN** `ExternalEditor("code --wait")` が組み立てるプロセスの引数と一時ファイルを、`initial` `Q1: A\nQ2: B` で確認する
- **THEN** 引数は `code` `--wait` `<一時ファイルのパス>` の順で、パスの末尾は `.md`、そのファイルの内容は `Q1: A\nQ2: B` である

#### Scenario: 終了後に一時ファイルを読んで削除する
- **WHEN** 内容が `Q1: B` の一時ファイルに対して、終了コード 0 の完了として結果を読む処理を呼ぶ
- **THEN** メッセージの本文は `Q1: B` で、一時ファイルは存在しない

#### Scenario: 非 0 で終了したら投稿せずエラーになる
- **WHEN** 一時ファイルに対して、終了コード 1 のエラーを完了として結果を読む処理を呼ぶ
- **THEN** メッセージはそのエラーを持ち、一時ファイルは存在しない

### Requirement: 編集結果を検査してから投稿する
`Update` は編集完了のメッセージを MUST 次の順で扱う。検査の規則は `answer-action` と同じ関数（`action.BlockedByLines`、`action.HasRoutineMarker`）を使い、`internal/ui` に別の判定を持たない。
1. エラーを持つ: フッタの右側に `エディタ: <エラー文字列>` を赤で出し、投稿しない。画面は `a` を押した画面のまま
2. 本文の前後の空白を除いた結果が空: フッタの右側に `回答を中止しました（本文が空）` を出し、投稿しない
3. `action.HasRoutineMarker(本文)` が真（`<!-- routine -->` または `&lt;!-- routine --&gt;` を含む）: 本文を下書きとして保持し、確認画面（Requirement「確認画面では投稿・編集に戻る・中止を選ぶ」）に移る。理由は「マーカー」で、投稿の選択肢は無い
4. `action.BlockedByLines(本文)` が 1 行以上: 本文を下書きとして保持し、確認画面に移る。理由は「blocked-by」で、投稿の選択肢がある（不変条件 8「回答エディタは投稿前にこの行を検出して警告する」）
5. それ以外: 投稿のコマンド（Requirement「投稿の結果をステータスに出し、再取得しない」）を返す

#### Scenario: 問題の無い本文はそのまま投稿される
- **WHEN** `gh.NewFake` で新しく作った `Fake`（`Result` を作るのに使った `Fake` とは別のもの。s07 の `Fetch` は `ViewIssue` を `Calls` に記録するので、`client` には `Calls` が空の `Fake` を使う。以下の Scenario も同じ）を `client`、固定文字列 `Q1: A` を返すスタブを `Editor` にして `New` し、主体が PR 131 の Card を選んだ `Model` に `a` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡し、さらに返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` はちょうど 1 件で `Method` が `CommentPR`、`Repo` が `org/app`、`Number` が 131、`Body` が `Q1: A` である。画面はキューのままである

#### Scenario: 空の本文は投稿されない
- **WHEN** スタブが ` \n` を返す状態で同じ手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** 編集完了のメッセージに対してコマンドは返らず、`Fake.Calls` は空で、フッタに `回答を中止しました（本文が空）` が含まれる

#### Scenario: マーカーを含む本文は投稿の選択肢の無い確認画面になる
- **WHEN** スタブが `<!-- routine -->\nQ1: A` を返す状態で同じ手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** 画面は確認画面で、`<!-- routine -->` を含む本文は投稿できません の行があり、フッタに `e 編集に戻る` と `Esc 中止` があって `y 投稿` は無く、`Fake.Calls` は空である

#### Scenario: blocked-by 行を含む本文は警告付きの確認画面になる
- **WHEN** スタブが `Q1: A\n  blocked-by: human` を返す状態で同じ手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** 画面は確認画面で、`回答の確認: org/app PR#131` の行、`blocked-by: で始まる行があります` の行、`blocked-by: human` の行、下書きの `Q1: A` の行があり、フッタに `y 投稿`、`e 編集に戻る`、`Esc 中止` があって、`Fake.Calls` は空である

#### Scenario: エディタの失敗は赤で出て投稿されない
- **WHEN** スタブがエラー `exit status 1` を返す状態で同じ手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** コマンドは返らず、`Fake.Calls` は空で、フッタに `エディタ: exit status 1` が含まれ、画面はキューのままである

### Requirement: 確認画面では投稿・編集に戻る・中止を選ぶ
`Model` は画面の状態として、s09 の 3 つ（キュー / カード詳細 / PR 詳細）に加えて 確認 を MUST 持ち、確認画面に移るときは戻り先（`a` を押した画面）と下書きと理由を保持する。確認画面の `View` は、端末の幅と高さの全体を使い、上から `回答の確認: <表示名>` の行、理由の行（blocked-by: `blocked-by: で始まる行があります（dispatcher はこの行を含む最新コメントを正本にします）` に続けて該当行を 1 行ずつ。マーカー: `<!-- routine --> を含む本文は投稿できません（TUI からの投稿は人の発言でなければなりません）`）、区切り線、下書き、フッタの順で描く。理由の行（blocked-by 該当行）も含めて残りの高さで切り（スクロールしない）、フッタは常に出す。フッタの左は理由が blocked-by なら `y 投稿  e 編集に戻る  Esc 中止  q 終了`、マーカーなら `e 編集に戻る  Esc 中止  q 終了`。右は s08 と同じステータス。
確認画面のキーは MUST 次のとおり。
- `y`: 理由が blocked-by のときだけ、下書きをそのまま本文として投稿のコマンドを返し、戻り先の画面に戻る。理由がマーカーなら何もしない
- `e`: 下書きを `initial` にして `Editor` を呼び、そのコマンドを返す（編集結果は Requirement「編集結果を検査してから投稿する」の手順にもう一度かかる）。画面は戻り先に戻す（エディタの終了後に再び検査される）
- `Esc`: 投稿せず戻り先の画面に戻り、フッタの右側に `回答を中止しました` を出す
- `q` / `Ctrl+C`: 終了コマンドを返す（s09「どの画面でも q で終了する」と同じ）
- 他のキー: 何もしない
確認画面で取得完了のメッセージ（s08 `fetchedMsg`）が届いたら、`Cards` と最終更新時刻は s08 の規則どおり更新し、下書きと対象は変えない。

#### Scenario: y で blocked-by 行を含む本文が投稿される
- **WHEN** blocked-by の確認画面（下書き `Q1: A\n  blocked-by: human`、対象 PR 131）の `Model` に `y` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` は 1 件で `Method` が `CommentPR`、`Body` が `Q1: A\n  blocked-by: human` であり、画面はキューである

#### Scenario: e で下書きを入れたエディタが再び開く
- **WHEN** 同じ確認画面の `Model` に `e` を与え、返ったコマンドを実行する
- **THEN** スタブは `initial` として `Q1: A\n  blocked-by: human` を受け取り、`Fake.Calls` は空である

#### Scenario: Esc で中止する
- **WHEN** 同じ確認画面の `Model` に `Esc` を与え、`View` から ANSI エスケープを除いて読む
- **THEN** 画面はキューで、`Fake.Calls` は空で、フッタに `回答を中止しました` が含まれる

#### Scenario: マーカーの確認画面で y は何もしない
- **WHEN** マーカーの確認画面（下書き `<!-- routine -->\nQ1: A`）の `Model` に `y` を与える
- **THEN** コマンドは返らず、画面は確認画面のままで、`Fake.Calls` は空である

#### Scenario: 詳細画面から入った確認画面は詳細画面に戻る
- **WHEN** `example` の issue 108 のカード詳細で `a` を押し、スタブが `Q1: A\nblocked-by: human` を返して確認画面に移った `Model` に `Esc` を与える
- **THEN** 画面はカード詳細で、詳細の対象は issue 108 の Card のままである

### Requirement: 投稿の結果をステータスに出し、再取得しない
投稿のコマンドは `action.Comment` を対象と本文で呼び（`ctx` は 30 秒のタイムアウト付き。design.md の未決事項）、その結果（表示名とエラー）を運ぶ `internal/ui` 内のメッセージを返す。`Model` は `gh` を直接呼ばず、投稿は必ずコマンド（別ゴルーチン）で MUST 行う。
- 投稿中: フッタの右側に `<表示名> にコメントを投稿中` を出す。この間の `a` / `t` は何もしない
- 成功: フッタの右側に `<表示名> にコメントしました` を出す
- 失敗: フッタの右側に `<表示名> へのコメントに失敗: <エラー文字列>` を赤で出す
投稿の成否にかかわらず、`Cards` と最終更新時刻を変えず、取得のコマンドを返さない（書き込み後の対象 1 件再取得は D-002 のとおりだが s18 が担当する。それまでは s12 の `R` か s13 の自動更新で反映する）。ラベルは触らない（`Fake.Calls` に `AddLabel` / `RemoveLabel` が現れない）。このステータスは次の取得が始まったとき、または次に `t` か `a` を押したときに消える（s11 `todo-toggle` のステータスと同じ場所・同じ寿命）。

#### Scenario: 投稿の成功がフッタに出る
- **WHEN** Requirement「編集結果を検査してから投稿する」の 1 つ目の Scenario の手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** フッタに `org/app PR#131 にコメントしました` が含まれ、`Fake.Calls` は 1 件だけで、`Cards` は変わらず、今やるタブに PR 131 の行が残る

#### Scenario: 投稿中は投稿中の表示で a は効かない
- **WHEN** 同じ手順で編集完了のメッセージを `Update` に渡した直後（投稿のコマンドを実行する前）に `View` を読み、続けて `a` を与える
- **THEN** フッタに `org/app PR#131 にコメントを投稿中` が含まれ、`a` に対してコマンドは返らない

#### Scenario: 投稿の失敗は赤で出る
- **WHEN** `*gh.Fake` を埋め込んで `CommentPR` だけがエラー `gh pr comment 131 -R org/app --body-file -: exit 1: HTTP 403` を返す型を `client` にして同じ手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** フッタに `org/app PR#131 へのコメントに失敗:` と `HTTP 403` が含まれ、`Cards` は変わらない

