issue: #38

## Why

`a` で開くエディタに入るのは `Q1: A\nQ2: B` の記号だけである（`internal/action/template.go:10-27`）。
質問文と選択肢は画面（カード詳細の blocked-by 要約、PR 詳細のコメント）にしか無く、エディタが開いている間は
その画面を読み直せない。`ExternalEditor` は Bubble Tea の外部プロセス実行で端末を明け渡し、`answer-question`
「エディタが動いている間、`Model` はキー入力を受けない」と定めてあるからで、**`A` が何の案だったかを確かめるには
エディタを閉じるしかない**。記号だけのテンプレートは、推奨をそのまま採るとき以外は役に立たない。

もう 1 つ、`## Q1.` 見出しでないコメントではテンプレートが空になり、エディタが白紙で開く
（`internal/action/template.go:22-26`。`example` の PR 131 の `<!-- routine -->\nQ1: マイグレーションを分けますか。` や、
issue の `blocked-by: human` コメントがこれに当たる。見出し形式が固定でないことは `answer-action` の
Requirement に書かれている）。局面 B の「人に何を決めてほしいか」はまさにこの形で届くので、
いま最も助けが要る場面でエディタは何も教えてくれない。ただしこの change が白紙をやめるのは
`blocked-by:` 行を持つコメント（局面 B）に限り、素の `Q1: …` 形式は引用の対象にしない（未確定の判断 Q1）。

## What Changes

- `action.AnswerTemplate` が、回答行の前に**最新の routine コメントの引用**を置く。1 行を `> <行>`、空行を `>` の
  1 文字にした引用ブロックのあと、空行 1 行を挟んで今までどおりの `Q<n>: <記号>` の行を並べる
- 引用から routine マーカー（`<!-- routine -->` とエスケープ済みの `&lt;!-- routine --&gt;`）を含む行を落とす。
  マーカーを含む本文は投稿できない（`answer-action`「routine マーカーを含む本文は投稿しない」・
  `internal/action/action.go:36-40`）ので、これは引用の前提条件である
- 質問の見出し（`## Q<n>.`）も `blocked-by:` 行も無い routine コメントは引用しない。`started:` / `session:` の作業印や
  `## PR リスク評価` が最新の routine コメントである対象で、それが引用されるのを防ぐ
- `blocked-by:` 行を持つコメント（issue の局面 B。見出し形式が固定でない）では、引用だけが入って回答行は 0 行になる。
  今は白紙で開くところに「人に何を決めてほしいか」が入る
- 引用は**そのまま投稿される**。投稿前に落とす処理は持たない（未確定の判断 Q2）
- 触るのは `internal/action/template.go` の 1 関数だけで、`internal/ui` は 1 行も変えない
  （`internal/ui/answer.go:56` が `AnswerTemplate` の戻り値をそのまま `initial` に渡している）

## Capabilities

### New Capabilities

（無し）

### Modified Capabilities

- `answer-action`: Requirement「回答テンプレートは最新の routine コメントの質問から組み立てる」の組み立て方を、
  引用ブロック + 回答行に改める。引用の書式・マーカー行の除外・引用しない条件を定める
- `answer-question`: Requirement「`a` は画面の対象を決めて回答テンプレートを入れたエディタを開く」の Scenario が
  期待する `initial` の値が変わる（記号だけ → 引用付き。`example` の対象では空文字列 → 引用だけ）

## Impact

- `internal/action/template.go`: `AnswerTemplate` の戻り値の組み立て。引用を作る非公開関数を同じファイルに 1 つ足す
- `internal/action/template_test.go`: 既存の期待値（`Q1: A\nQ2: B` など）を引用付きに直し、マーカー行の除外・
  空行の引用・引用しない条件のケースを足す
- `internal/ui/answer_test.go`（`answer-question` の Scenario を読むテスト）: スタブ `Editor` が受け取る `initial` の期待値
- `internal/ui`・`internal/model`・`internal/gh`・`internal/classify`: 変更しない
- 依存の追加は無い

## 確定した判断

- **引用は `internal/action` の 1 か所で済む。** `internal/ui/answer.go:56` が `action.AnswerTemplate(comments)` の
  戻り値をそのまま `openEditor` に渡し、`ExternalEditor` はその文字列を一時ファイルに書くだけである
  （`answer-question`「`ExternalEditor` は設定のエディタを一時ファイルで開く」）。テンプレートの中身を変える
  change は `internal/ui` に届かない
- **引用にマーカーを含めてはならない。** `action.Comment` は本文のどこかにマーカーがあれば `ErrRoutineMarker` を返し
  （`internal/action/action.go:36-40, 62`）、`internal/ui` はその前に確認画面（投稿の選択肢なし）へ移る
  （`answer-question`「編集結果を検査してから投稿する」手順 3）。マーカー行を落とさずに引用すると、
  **すべての回答が投稿できなくなる**。既存 Requirement も「テンプレートに `<!-- routine -->` を含めない」と定めている
- **引用した `blocked-by:` 行は警告を出さない。** `action.BlockedByLines` は行頭の `>` を吸収せず字面どおりに判定し、
  `answer-action` が「`> blocked-by: human` は検出しない。dispatcher も検出しない」と明記している。
  issue の `blocked-by: human` コメントを丸ごと引用しても、確認画面には入らず、dispatcher の正本も動かない
- **引用した本文は AI のコメントに見えない。** `model.IsAI` は本文の先頭がマーカーかと、行頭が `## PR リスク評価` かだけを
  見る（`internal/model/parse.go:35-46`）。行頭が `> ` になった引用はどちらにも当たらないので、投稿した回答は
  人のコメントとして分類され、dispatcher は `question` を外せる
- **引用した質問は二重にパースされない。** `model.ParseQuestions` は行頭の空白とタブだけを落として `## Q<n>.` を探す
  （`internal/model/parse.go:147-172`）。`> ## Q1.` は質問として読まれないので、引用付きの回答が
  次の `a` のテンプレート元になっても（人のコメントなので `AI` は false で選ばれないが）質問が増えることはない
- **引用は回答行の前に置く。** GitHub 上で「質問 → 回答」の読み順になり、確認画面が下書きをそのまま見せる形とも合う。
  エディタのカーソルは 1 行目（引用の先頭）に立つので、回答行までは人が移動する
- **`n`（新しい issue）の下書きは変えない。** `NewIssueDraft` は案内 2 行を区切りに使う別の経路で
  （`internal/action/newissue.go:14-30`）、質問の引用と関係が無い
- **確認画面の `e` で開き直すときは下書きをそのまま渡す。** `internal/ui/answer.go:153` が `m.answer.draft` を
  `initial` にしており、引用を作り直さない。人が引用を削ってから `e` で戻っても、削った状態が保たれる

## 未確定の判断

### Q1. エディタに入れる引用の範囲
- 選択肢 A（推奨）: 最新の routine コメントの**本文全体**（マーカー行だけ除く）を引用する。ただし質問の見出し
  （`## Q<n>.`）も `blocked-by:` 行も無いコメントは引用しない。引用されるのは grill の質問コメント
  （`routine-propose` が `## Q<n>.` 形式で書く）と issue の `blocked-by: human` コメントの 2 つで、
  「以下 2 点、回答をお願いします」の前置きや `- 依存: Q2 の回答が要る` の行も読めるようになる。
  引用しない条件は既存の 2 関数（`model.ParseQuestions` と `action.BlockedByLines`）で判定するので新しいパースは増えない。
  引かれないのは、`## PR リスク評価`・`started:` / `session:` の作業印・`example` の PR 131 のような素の
  `Q1: マイグレーションを分けますか。`。この 3 つでは**今と同じく白紙のエディタ**が開く
- 選択肢 B: パースできた質問だけを引用する（`> ## Q1. <題>` と `> - 選択肢 A（推奨）: <内容>` の行だけを組み立てる）。
  前置きと `依存:` の行は入らず、**issue の `blocked-by: human` では今と同じく白紙のまま**になる
  （見出し形式が固定でないため。issue #38 が挙げた「選択肢を理解しながら回答できる」が局面 B で満たされない）
- 選択肢 C: 最新の routine コメントの本文全体を無条件に引用する。判定が 1 つ減り、素の `Q1: …` 形式の質問も
  引用される。代わりに、`## PR リスク評価`（assess が書く長いコメント）が最新の routine コメントである PR で `a` を
  押すと、評価の全文がエディタに引用される。`started:` / `session:` の作業印も同じように引用される
- 依存: なし

### Q2. 引用をそのまま投稿するか
- 選択肢 A（推奨）: **そのまま投稿する**。GitHub 上では引用として描かれ、`Q1: B` の回答が「どの問いのどの選択肢か」を
  単独で読める形になる。落とす処理を持たないので、人が自分で書いた引用が消える事故も起きない。
  代わりに、投稿するコメントは質問文のぶん長くなる（routine の質問コメントを 1 度だけ写す量）
- 選択肢 B: 投稿前に引用行（行頭が `>`）を落とす。`n` の案内行を issue に入れないのと同じ扱いにする
  （`internal/action/newissue.go:26-28`「案内の文字列が issue になって GitHub へ出ていく方が悪い」）。
  投稿されるのは `Q1: A` だけになるが、`action.Comment` に落とす処理が増え、**人が自分の判断を引用で補強した行も
  一緒に消える**（`> 選択肢 B は次の issue に回したい` と書いたつもりの行が届かない）
- 依存: なし
