## 1. 質問のパースを実物の書式に合わせる

- [x] 1.1 `internal/model/parse.go` の `questionRe` を「行頭の `#` が 1 つ以上」に、`optionRe` を「`選択肢` の前・
  `<Letter>` の直後・`（推奨）` の直後の `**` を読み飛ばす」形に広げる。`Text` の中の `*` は落とさない
- [x] 1.2 `internal/model/parse_test.go` に `card-model` の MODIFIED Requirement の Scenario 4 件を足す。
  4 件は、mvp.md 形式の 2 問・上流の質問コメントの書式・見出しの記号が無い質問・見出しの無い本文である。
  上流の書式のケースは、PR #37 のコメント（`### Q1.` の見出し、`- **選択肢 A（推奨）**: …`、`- 依存: なし` の行、
  `---` の区切り、太字の段落）を写した本文を入力にして、`Question` 2 件と各 `Options` の `Letter` /
  `Recommended` / `Text` を検証する
- [x] 1.3 `internal/ui` の詳細画面のテストが通ることを確認する。パースが広がると `questionLines`
  （`internal/ui/detail.go:311-322`）が blocked-by 要約に質問を出すようになるので、`### Q<n>.` 形式のコメントを
  持つ Card で「質問と選択肢の行が出る」ことを 1 件足す

## 2. テンプレートの引用

- [x] 2.1 `internal/action/template.go` に、対象のコメントを選ぶ非公開関数を足す。`comments` を末尾から先頭へ見て、
  `AI` が true で、かつ質問がパースできるか `BlockedByLines` が 1 行以上を返す最初のコメントを返す（無ければ
  見つからないことを返す）。引用と回答行はどちらもこのコメントから作る
- [x] 2.2 `internal/action/template.go` に引用ブロックを作る非公開関数を足す。範囲の始まりは最初の質問見出しの行
  （見出しが無ければ、マーカー行と `blocked-by:` / `unblock-when:` の行を飛ばした後の最初の非空行）、終わりは
  末尾から空行・区切り線だけの行・`_` で挟まれた 1 行を落とした位置とする。範囲の各行から、マーカーだけの行・
  `blocked-by:` 行・`unblock-when:` 行を落とし、残った行のマーカーを `[routine マーカー]` に置き換え、行末の空白を
  落として `> ` を前に付け（空行は `>` の 1 文字）、末尾に連なる `>` だけの行を落とす。マーカーの文字列は
  `internal/action/action.go` の定数を使い、この関数で定義し直さない
- [x] 2.3 `AnswerTemplate` を「引用ブロック + 空行 1 行 + 回答行」に組み替える。回答行は 2.1 が選んだコメントから
  作り、`Q<n>: <記号>` の作り方（推奨 → 先頭 → 空）は変えない
- [x] 2.4 `internal/action/template_test.go` に `answer-action`「回答テンプレートは…」の Scenario 12 件を足す。
  mvp.md の書式・blocked-by だけ・作業印を飛び越える・行の途中のマーカー・末尾の区切り線と署名の 5 件は
  戻り値の完全一致で、上流の書式のケースは行ごとの包含と先頭行・末尾 2 行で、残りは性質
  （`HasRoutineMarker` が false / 空文字列）で検証する（完全一致だけで書くと、区切りの 1 行を間違えたときに
  全件が同時に落ちて原因が読めない）

## 3. 引用だけの本文を投稿しない

- [x] 3.1 `internal/action/action.go` に `IsBlankAnswer(body string) bool` を足し（前後の空白を除いて空でも `>` 始まりでも
  ない行が 1 つも無ければ真）、`Comment` の空判定をこれに差し替える。`internal/action/action_test.go` に
  `answer-action`「空の本文は投稿しない」の Scenario 3 件（空白だけ / 引用行だけ / 引用に書き足した本文）を足す
- [x] 3.2 `internal/ui/answer.go` の編集完了の検査（手順 2）を `action.IsBlankAnswer` に差し替え、前後の空白を除いて
  空なら `回答を中止しました（本文が空）`、引用行が残っているなら `回答を中止しました（引用だけです）` を出す。
  `internal/ui` のテストに `answer-question`「編集結果を検査してから投稿する」の Scenario 2 件
  （引用だけの本文は投稿されない / 引用に書き足した本文は投稿される）を足す

## 4. エディタに渡る初期テキスト

- [x] 4.1 `internal/ui` の `a` のテストで、`answer-question`「`a` は画面の対象を決めて…」の Scenario 5 件を通す。
  キュー画面のスタブ `Editor` が受け取る `initial` を上流の書式の引用付き（`> ### Q1. 分けるか` から `Q1: A` まで）で
  検証し、対象の検証は別の Scenario に分ける。`example` の issue 108・PR 131・issue 140 では空文字列のままで
  あることも検証する

## 5. 利用者向けの説明

- [x] 5.1 `README.md` の「回答（`a`）」の節（`README.md:158-160`）を書き直す。質問文と選択肢が `>` 付きの引用として
  テンプレートの先頭に入ること、対象はコメント列を遡って見つけた最初の問い（`Q<n>.` の見出し、または
  `blocked-by:` 行を持つ routine コメント）であること、引用を消さずに投稿すると引用もコメントの一部として
  GitHub に出ること、引用だけのまま閉じると投稿されないことを書く

## 6. 確認

- [x] 6.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [x] 6.2 `golangci-lint run` が通ることを確認する
- [x] 6.3 `openspec validate s34-answer-template-quote --strict` が通ることを確認する
