## MODIFIED Requirements

### Requirement: a は画面の対象を決めて回答テンプレートを入れたエディタを開く
`internal/ui` はエディタ起動の型 `Editor`（`func(initial string) tea.Cmd`。返すコマンドは、編集後の本文またはエラーを運ぶ `internal/ui` 内のメッセージを返す）を MUST 公開し、`Model` は `New` で受け取った `Editor` と `gh.GHClient` を保持する（`New` の引数は s08 `queue-screen`「Model は Card をタブ別に並べ、選択行を 1 つ持つ」の MODIFIED を見る）。
`Update` は `a` を画面の状態ごとに MUST 次のとおり扱う。回答の対象 `action.Target` と表示名（`<Repo> PR#<n>` または `<Repo> #<n>`。s09 の詳細ヘッダと同じ表記）を決め、対象の `Comments` から `action.AnswerTemplate` でテンプレートを作り、`Editor(テンプレート)` が返すコマンドをそのまま（他のコマンドとまとめずに）返す。
- キュー画面: 選択行の主体（s08 `Subject`。`isPR` が対象の `IsPR`）。選択行が無ければ何もしない
- カード詳細画面: 詳細の対象の `Card.Issue`（`IsPR` は false）
- PR 詳細画面: 詳細の対象の PR（`IsPR` は true）
対象に `question` ラベルが付いているかどうかは問わない（mvp.md キーバインド表の `a` は「回答・コメント」であり、`question` 無しの対象へのコメントを禁じる記述が docs に無い。design.md の未決事項）。対象の `Comments` が nil（未取得）ならテンプレートは空である。書き込み（投稿・ラベル切り替え）の結果を待っている間（Requirement「投稿の結果をステータスに出し、再取得しない」、s11 `todo-toggle`「書き込み中は t と a を受け付けない」）は `a` / `t` を無視する。
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
- **WHEN** `example` の `Result` を渡してバックログタブ（issue 140。`Labels` 空、`Comments` nil）を選んだ `Model` に `a` を与える
- **THEN** コマンドが返り、対象は `Target{Repo: "org/app", Number: 140, IsPR: false}`、スタブに渡る `initial` は空文字列である

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
