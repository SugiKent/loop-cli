## MODIFIED Requirements

### Requirement: a は画面の対象を決めて回答テンプレートを入れたエディタを開く
`internal/ui` はエディタ起動の型 `Editor`（`func(initial string) tea.Cmd`。返すコマンドは、編集後の本文またはエラーを運ぶ `internal/ui` 内のメッセージを返す）を MUST 公開し、`Model` は `New` で受け取った `Editor` と `gh.GHClient` を保持する（`New` の引数は s08 `queue-screen`「Model は Card をタブ別に並べ、選択行を 1 つ持つ」の MODIFIED を見る）。
`Update` は `a` を画面の状態ごとに MUST 次のとおり扱う。回答の対象 `action.Target` と表示名（`<Repo> PR#<n>` または `<Repo> #<n>`。s09 の詳細ヘッダと同じ表記）を決め、対象の `Comments` から `action.AnswerTemplate` でテンプレートを作り、`Editor(テンプレート)` が返すコマンドをそのまま（他のコマンドとまとめずに）返す。
- キュー画面: 選択行の主体（s08 `Subject`。`isPR` が対象の `IsPR`）。選択行が無ければ何もしない
- カード詳細画面: 詳細の対象の `Card.Issue`（`IsPR` は false）
- PR 詳細画面: 詳細の対象の PR（`IsPR` は true）
対象に `question` ラベルが付いているかどうかは問わない（mvp.md キーバインド表の `a` は「回答・コメント」であり、`question` 無しの対象へのコメントを禁じる記述が docs に無い。design.md の未決事項）。テンプレートの中身は `answer-action`「回答テンプレートは最新の routine コメントの質問から組み立てる」が定める。`internal/ui` はその文字列をそのまま `Editor` へ渡し、質問の引用を自分で作ることも消すこともしない。対象の `Comments` が nil（s20 以降は取得に失敗したときだけ起きる）ならテンプレートは空である。書き込み（投稿・ラベル切り替え）の結果を待っている間（Requirement「投稿の結果をステータスに出し、再取得しない」、s11 `todo-toggle`「書き込み中は t と a を受け付けない」）は `a` / `t` を無視する。
エディタが動いている間、`Model` はキー入力を受けない（端末はエディタが使う。Bubble Tea の外部プロセス実行の仕組みに従う）。

#### Scenario: キュー画面で a を押すと主体のテンプレートでエディタが開く
- **WHEN** 主体が PR 131（`Repo` `org/app`、`Comments` が `AI` true で `Body` `<!-- routine -->\n### Q1. 分けるか\n- **選択肢 A（推奨）**: 分ける\n- **選択肢 B**: 分けない` の 1 件）の Card を今やるタブに持ち、渡された `initial` を記録するスタブ `Editor` で `New` した `Model` に `a` を与える
- **THEN** コマンドが返り、それを実行するとスタブは `initial` として `> ### Q1. 分けるか\n> - **選択肢 A（推奨）**: 分ける\n> - **選択肢 B**: 分けない\n\nQ1: A` を受け取る

#### Scenario: 回答の対象は画面が見せているものに決まる
- **WHEN** 上と同じ `Model` に `a` を与える
- **THEN** `Model` が保持する回答の対象は `Target{Repo: "org/app", Number: 131, IsPR: true}` である

#### Scenario: 0 行のタブで a は何もしない
- **WHEN** `example` の `Result` を渡して異常タブ（0 行）に切り替えた `Model` に `a` を与える
- **THEN** コマンドは返らず、画面はキューのままである

#### Scenario: カード詳細では Issue、PR 詳細ではその PR が対象になる
- **WHEN** `example` の issue 108 の Card（主体は PR 131）のカード詳細を開いた `Model` に `a` を与え、別に同じカードから `Enter` で PR 詳細を開いた `Model` に `a` を与える
- **THEN** 1 つ目の対象は `Target{Repo: "org/app", Number: 108, IsPR: false}`、2 つ目の対象は `Target{Repo: "org/app", Number: 131, IsPR: true}` である

#### Scenario: question の無い issue にも a でコメントできる
- **WHEN** `example` の `Result` を渡してバックログタブ（issue 140。`Labels` 空。s20 で全 issue のコメントを取るので `Comments` は長さ 0 の非 nil）を選んだ `Model` に `a` を与える
- **THEN** コマンドが返り、対象は `Target{Repo: "org/app", Number: 140, IsPR: false}`、スタブに渡る `initial` は空文字列である

### Requirement: 編集結果を検査してから投稿する
`Update` は `a` で始めた編集（`a` の押下と、回答の確認画面の `e` で開き直したもの）の完了のメッセージを MUST 次の順で扱う。検査の規則は `answer-action` と同じ関数（`action.IsBlankAnswer`、`action.HasRoutineMarker`、`action.BlockedByLines`）を使い、`internal/ui` に別の判定を持たない。`n` で始めた編集の完了は s15 `new-issue`「編集結果を検査して作成の確認画面に移る」が扱う。編集完了のメッセージ自体は経路を運ばないので、`Model` がエディタを開くたびに経路を記録し、次にエディタを開くまで保持する（s15 `new-issue`「編集結果を検査して作成の確認画面に移る」）。
1. エラーを持つ: フッタの右側に `エディタ: <エラー文字列>` を赤で出し、投稿しない。画面は `a` を押した画面のまま
2. `action.IsBlankAnswer(本文)` が真（空白だけ、または引用行と空行だけ）: 投稿しない。本文の前後の空白を除いた結果が空ならフッタの右側に `回答を中止しました（本文が空）`、そうでなければ（引用行が残っている場合）`回答を中止しました（引用だけです）` を出す。引用だけのテンプレートを開いてそのまま閉じた下書きを投稿すると、dispatcher が「人が答えた」とみなして worker が推奨案で進むため（`answer-action`「空の本文は投稿しない」）
3. `action.HasRoutineMarker(本文)` が真（`<!-- routine -->` または `&lt;!-- routine --&gt;` を含む）: 本文を下書きとして保持し、確認画面（Requirement「確認画面では投稿・編集に戻る・中止を選ぶ」）に移る。理由は「マーカー」で、投稿の選択肢は無い
4. `action.BlockedByLines(本文)` が 1 行以上: 本文を下書きとして保持し、確認画面に移る。理由は「blocked-by」で、投稿の選択肢がある（不変条件 8「回答エディタは投稿前にこの行を検出して警告する」）
5. それ以外: 投稿のコマンド（Requirement「投稿の結果をステータスに出し、再取得しない」）を返す

#### Scenario: 問題の無い本文はそのまま投稿される
- **WHEN** `gh.NewFake` で新しく作った `Fake`（`Result` を作るのに使った `Fake` とは別のもの。s07 の `Fetch` は `ViewIssue` を `Calls` に記録するので、`client` には `Calls` が空の `Fake` を使う。以下の Scenario も同じ）を `client`、固定文字列 `Q1: A` を返すスタブを `Editor` にして `New` し、主体が PR 131 の Card を選んだ `Model` に `a` を与え、返ったコマンドを実行して得たメッセージを `Update` に渡し、さらに返ったコマンドを実行して得たメッセージを `Update` に渡す
- **THEN** `Fake.Calls` はちょうど 1 件で `Method` が `CommentPR`、`Repo` が `org/app`、`Number` が 131、`Body` が `Q1: A` である。画面はキューのままである

#### Scenario: 空の本文は投稿されない
- **WHEN** スタブが ` \n` を返す状態で同じ手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** 編集完了のメッセージに対してコマンドは返らず、`Fake.Calls` は空で、フッタに `回答を中止しました（本文が空）` が含まれる

#### Scenario: 引用だけの本文は投稿されない
- **WHEN** スタブが `> 認可の方針をどこに書きますか。\n>\n> - 選択肢 A（推奨）: docs/policy.md` を返す状態で同じ手順を行い、`View` から ANSI エスケープを除いて読む
- **THEN** 編集完了のメッセージに対してコマンドは返らず、`Fake.Calls` は空で、フッタに `回答を中止しました（引用だけです）` が含まれる

#### Scenario: 引用に書き足した本文は投稿される
- **WHEN** スタブが `> 認可の方針をどこに書きますか。\n\nA でお願いします` を返す状態で同じ手順を行う
- **THEN** `Fake.Calls` は 1 件で `Method` が `CommentPR`、`Body` はスタブが返した文字列そのままである

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
