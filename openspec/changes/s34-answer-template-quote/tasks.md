## 1. 質問のパースを実物の書式に合わせる

- [ ] 1.1 `internal/model/parse.go` の `questionRe` を「行頭の `#` が 1 つ以上」に、`optionRe` を「`選択肢` の前・
  `<Letter>` の直後・`（推奨）` の直後の `**` を読み飛ばす」形に広げる。`Text` の中の `*` は落とさない
- [ ] 1.2 `internal/model/parse_test.go` に `card-model` の MODIFIED Requirement の Scenario 4 件を足す。
  4 件は、mvp.md 形式の 2 問・上流の質問コメントの書式・見出しの記号が無い質問・見出しの無い本文である。
  上流の書式のケースは、PR #37 のコメント（`### Q1.` の見出し、`- **選択肢 A（推奨）**: …`、`- 依存: なし` の行、
  `---` の区切り、太字の段落）を写した本文を入力にして、`Question` 2 件と各 `Options` の `Letter` /
  `Recommended` / `Text` を検証する
- [ ] 1.3 `internal/ui` の詳細画面のテストが通ることを確認する。パースが広がると `questionLines`
  （`internal/ui/detail.go:311-322`）が blocked-by 要約に質問を出すようになるので、`### Q<n>.` 形式のコメントを
  持つ Card で「質問と選択肢の行が出る」ことを 1 件足す

## 2. テンプレートの引用

- [ ] 2.1 `internal/action/template.go` に引用ブロックを作る非公開関数を足す。マーカーだけの行・`blocked-by:` 行・
  `unblock-when:` 行を落とし、残った行のマーカーを `[routine マーカー]` に置き換え、各行の行末の空白を落として
  `> ` を前に付け（空行は `>` の 1 文字）、末尾に連なる `>` だけの行を落とす。マーカーの文字列は
  `internal/action/action.go` の定数を使い、この関数で定義し直さない
- [ ] 2.2 `AnswerTemplate` を「引用ブロック + 空行 1 行 + 回答行」に組み替え、引用する条件（末尾のコメントが `AI`・
  質問がパースできるか `blocked-by:` 行を持つ・写して残る行が 1 行以上）を入れる。回答行の作り方と、
  最新の `AI` コメントを選ぶ規則は変えない
- [ ] 2.3 `internal/action/template_test.go` に `answer-action`「回答テンプレートは…」の Scenario 10 件を足す。
  上流の書式・mvp.md の書式・blocked-by だけ・行の途中のマーカーの 4 件は戻り値の完全一致で、
  残りは性質（`HasRoutineMarker` が false / 引用ブロックと回答行の間の空行が 1 行 / 引用が付かない）で検証する
  （完全一致だけで書くと、区切りの 1 行を間違えたときに全件が同時に落ちて原因が読めない）

## 3. 引用だけの本文を投稿しない

- [ ] 3.1 `internal/action/action.go` に `IsBlankAnswer(body string) bool` を足し（前後の空白を除いて空でも `>` 始まりでも
  ない行が 1 つも無ければ真）、`Comment` の空判定をこれに差し替える。`internal/action/action_test.go` に
  `answer-action`「空の本文は投稿しない」の Scenario 3 件（空白だけ / 引用行だけ / 引用に書き足した本文）を足す
- [ ] 3.2 `internal/ui/answer.go` の編集完了の検査（手順 2）を `action.IsBlankAnswer` に差し替え、前後の空白を除いて
  空なら `回答を中止しました（本文が空）`、引用行が残っているなら `回答を中止しました（引用だけです）` を出す。
  `internal/ui` のテストに `answer-question`「編集結果を検査してから投稿する」の Scenario 2 件
  （引用だけの本文は投稿されない / 引用に書き足した本文は投稿される）を足す

## 4. エディタに渡る初期テキスト

- [ ] 4.1 `internal/ui` の `a` のテストで、`answer-question`「`a` は画面の対象を決めて…」の Scenario 5 件を通す。
  キュー画面のスタブ `Editor` が受け取る `initial` を上流の書式の引用付き（`> ### Q1. 分けるか` から `Q1: A` まで）で
  検証し、対象の検証は別の Scenario に分ける。`example` の issue 108・PR 131・issue 140 では空文字列のままで
  あることも検証する

## 5. 利用者向けの説明

- [ ] 5.1 `README.md` の「回答（`a`）」の節（`README.md:158-160`）を書き直す。質問文と選択肢が `>` 付きの引用として
  テンプレートの先頭に入ること、引用が入るのは最新のコメントが routine の問い（`Q<n>.` の見出し、または
  `blocked-by:` 行を持つ）のときだけであること、引用を消さずに投稿すると引用もコメントの一部として GitHub に
  出ること、引用だけのまま閉じると投稿されないことを書く

## 6. 確認

- [ ] 6.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [ ] 6.2 `golangci-lint run` が通ることを確認する
- [ ] 6.3 `openspec validate s34-answer-template-quote --strict` が通ることを確認する
