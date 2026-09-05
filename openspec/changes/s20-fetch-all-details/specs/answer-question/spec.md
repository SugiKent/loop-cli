## MODIFIED Requirements

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
