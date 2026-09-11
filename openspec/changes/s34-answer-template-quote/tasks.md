## 1. テンプレートの引用

- [ ] 1.1 `internal/action/template.go`: 選んだ routine コメントの本文を引用ブロックに写す非公開関数を足す
  （マーカー（`<!-- routine -->` とエスケープ済みの形）を含む行を落とし、残った各行の行末の空白を落として、
  空になった行は `>`、それ以外は `> ` を前に付ける。末尾に連なる `>` だけの行は出さない）。
  `model.ParseQuestions` が 0 件で `BlockedByLines` も 0 行のコメントは引用しない条件を同じ関数に入れ、
  `AnswerTemplate` の戻り値を「引用ブロック + 空行 1 行 + 回答行」に組み替える（どちらかが空なら片方だけ、
  両方空なら空文字列）。`go build ./internal/action` が通ることで確認する
- [ ] 1.2 `internal/action/template_test.go` に `answer-action` の MODIFIED Requirement の Scenario 9 件を足し、
  戻り値の完全一致で検証する。9 件は、推奨を既定値にした 2 問・推奨が無ければ先頭と選択肢が無ければ空・
  最新の routine コメントを採る・blocked-by は引用だけ・末尾の空行を出さない・見出しの無いコメント・
  作業印のコメント・エスケープ済みのマーカー・コメントが無い、の各 Scenario。
  引用付きのテンプレートを `HasRoutineMarker` に渡して false になることも同じテストで見る（マーカーが残ると
  回答を 1 件も投稿できなくなるため）

## 2. エディタに渡る初期テキスト

- [ ] 2.1 `internal/ui` の `a` のテスト: `answer-question` の MODIFIED Requirement の Scenario 4 件を通す。
  キュー画面のスタブ `Editor` が受け取る `initial` の期待値を引用付き（`> ## Q1. 分けるか` から `Q1: A` まで）に
  直し、`example` の issue 108 のカード詳細・PR 131 の PR 詳細・issue 140 のバックログでは空文字列のままで
  あることを検証する
- [ ] 2.2 `internal/ui` の投稿・確認画面の既存テストが、テンプレートの変更後も通ることを確認する
  （スタブ `Editor` が固定文字列を返す Scenario はテンプレートに依存しないので、期待値の書き換えが要らないことを
  `go test ./internal/ui` で確かめる）

## 3. 利用者向けの説明

- [ ] 3.1 `README.md` の「回答（`a`）」の節（`README.md:158-160`）に、質問文と選択肢が `>` 付きの引用として
  テンプレートの先頭に入ること、引用が入るのは `## Q1.` 形式の質問コメントと `blocked-by:` 行を持つコメントだけで
  あること、引用を消さずに投稿すると引用もコメントの一部として GitHub に出ることを足す

## 4. 確認

- [ ] 4.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [ ] 4.2 `golangci-lint run` が通ることを確認する
- [ ] 4.3 `openspec validate s34-answer-template-quote --strict` が通ることを確認する
