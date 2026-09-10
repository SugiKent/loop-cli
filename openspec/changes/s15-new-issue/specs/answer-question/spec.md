## MODIFIED Requirements

### Requirement: ExternalEditor は設定のエディタを一時ファイルで開く
`internal/ui` は `ExternalEditor(command string) Editor` を MUST 公開する。`command` は config の `editor`（s02 が `$EDITOR` を展開済み）で、`cmd/loop-cli` がこれを `New` に渡す。返る `Editor` は次の振る舞いをする。
- `command` の前後の空白を除いた結果が空なら、外部プロセスを起動せず、`editor が設定されていません（config の editor か環境変数 EDITOR）` を含むエラーを運ぶメッセージを返すコマンドを返す（s02 `config-loading`「空のときの扱いは s10 が担当」。既定値は design.md の未決事項）
- それ以外は、一時ディレクトリに `loop-cli-draft-*.md` の一時ファイルを作って `initial` を書き込み、`command` を空白で分割した先頭を実行ファイル、残りを引数とし、その末尾に一時ファイルのパスを足したプロセスを、Bubble Tea の外部プロセス実行の仕組み（描画を止め、端末の標準入出力をプロセスに渡す）で起動する。シェルを通さない
- プロセスが終了コード 0 で終わったら一時ファイルを読み、その内容を編集後の本文としてメッセージで返し、一時ファイルを削除する。終了コードが非 0 か起動に失敗したら、そのエラーを運ぶメッセージを返し、一時ファイルは削除する

一時ファイルの名前を `loop-cli-draft-*.md` にするのは、この `Editor` を `a`（回答の下書き）と s15 `new-issue` の `n`（新しい issue の下書き）が共有するためである。エディタは開いているファイル名を人に見せるので、どちらの下書きを書いているときも読める名前にする。

#### Scenario: editor が空なら起動せずエラーになる
- **WHEN** `ExternalEditor("")` が返す `Editor` を `initial` `Q1: A` で呼び、返ったコマンドを実行する
- **THEN** 返るメッセージはエラーを持ち、エラー文字列に `editor` を含む

#### Scenario: コマンドの分割と一時ファイル
- **WHEN** `ExternalEditor("code --wait")` が組み立てるプロセスの引数と一時ファイルを、`initial` `Q1: A\nQ2: B` で確認する
- **THEN** 引数は `code` `--wait` `<一時ファイルのパス>` の順で、パスの末尾は `.md`、ファイル名は `loop-cli-draft-` で始まり、そのファイルの内容は `Q1: A\nQ2: B` である

#### Scenario: 終了後に一時ファイルを読んで削除する
- **WHEN** 内容が `Q1: B` の一時ファイルに対して、終了コード 0 の完了として結果を読む処理を呼ぶ
- **THEN** メッセージの本文は `Q1: B` で、一時ファイルは存在しない

#### Scenario: 非 0 で終了したら投稿せずエラーになる
- **WHEN** 一時ファイルに対して、終了コード 1 のエラーを完了として結果を読む処理を呼ぶ
- **THEN** メッセージはそのエラーを持ち、一時ファイルは存在しない

### Requirement: 編集結果を検査してから投稿する
`Update` は `a` で始めた編集（`a` の押下と、回答の確認画面の `e` で開き直したもの）の完了のメッセージを MUST 次の順で扱う。検査の規則は `answer-action` と同じ関数（`action.BlockedByLines`、`action.HasRoutineMarker`）を使い、`internal/ui` に別の判定を持たない。`n` で始めた編集の完了は s15 `new-issue`「編集結果を検査して作成の確認画面に移る」が扱う。編集完了のメッセージ自体は経路を運ばないので、`Model` がエディタを開くたびに経路を記録し、次にエディタを開くまで保持する（s15 `new-issue`「編集結果を検査して作成の確認画面に移る」）。
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

#### Scenario: n で始めた編集は回答として扱わない
- **WHEN** 固定文字列 `タイトル\n\n本文` を返すスタブを `Editor` にして `New` し、主体が PR 131 の Card を選んだ `Model` に `n` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** 画面は作成の確認画面で、`Fake.Calls` は空であり、`回答の確認:` の行は無い

#### Scenario: 回答の確認画面の e から戻った編集完了は回答として投稿される
- **WHEN** blocked-by の確認画面（下書き `Q1: A\n  blocked-by: human`、対象 PR 131）の `Model` に `e` を与え、`Q1: A` を返すスタブの編集完了のメッセージを `Update` に渡し、さらに返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` は 1 件で `Method` が `CommentPR`、`Body` が `Q1: A` であり、`Method` が `CreateIssue` の要素は無い
