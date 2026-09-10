# tasks: s29-new-issue-placeholders

Refs #17

## 0. 前提

- [ ] 0.1 先行 change `s15-new-issue` が archive 済みであることを、`origin/main` に `openspec/specs/new-issue/spec.md` と `openspec/specs/new-issue-action/spec.md` があることで確認する。まだ無ければ着手せず、`blocked-by: change s15-new-issue` を issue へ書き戻す（proposal の「先行 change への依存」）
- [ ] 0.2 archive の直前に、`origin/main` の `openspec/specs/new-issue/spec.md` を見て、この change の 3 つの MODIFIED（Requirement「n は画面の対象のリポジトリを作成先にしてエディタを開く」「編集結果を検査して作成の確認画面に移る」「作成の確認画面は作成先と下書きを出し、y で作成して Esc で中止する」）をその時点の最新の版から写し直し、先に archive された change の追記を消していないことを `git diff` で確かめる（MODIFIED は Requirement ブロック全体を置き換えるので、写し忘れると先行 change の変更が消える）

## 1. 下書きと分割（internal/action）

- [ ] 1.1 `internal/action/newissue.go` に、2 本のプレースホルダー行の文言（`タイトル（この下の行に入力してください）` / `概要（この下に入力してください）`）と、`n` で開くエディタの初期の下書き（`<タイトルの行>\n\n<概要の行>\n\n`）を足す。初期の下書きは `internal/ui` から参照するので公開し、2 本の文言はこのパッケージの中だけで使う
- [ ] 1.2 `SplitNewIssue` を `(title, body string, ok bool)` を返す形に書き換える。行に分けて前後の空白を落とした結果で 2 本の区切りを先頭から探し、見つからなければ `ok` を偽にして空文字列を 2 つ返す。タイトルは 2 本の区切りに挟まれた行から空行を捨てて半角空白 1 つで連結し、本文は概要の区切りより後を改行で連結して前後の空白を落とす（spec の 5 段の手順どおり）
- [ ] 1.3 `internal/action/newissue_test.go` の `TestSplitNewIssueTakesFirstLineAsTitle` を、`new-issue-action` の Requirement「プレースホルダー行 2 本を区切りにタイトルと本文を分ける」の 8 つの Scenario を検証する形に書き換える（挟まれた行がタイトル / 本文は下すべて / 複数行の連結 / 消した下書き / 概要だけ無い / 区切りが逆順 / 書き足した行 / 行末の空白と CRLF）。Requirement「エディタに渡す下書きはプレースホルダー行 2 本を持つ」の 2 つの Scenario（初期の下書きの中身 / それをそのまま分けるとタイトルも本文も空）も足す
- [ ] 1.4 `internal/action/newissue_test.go` の `CreateIssue` のテスト 3 本（`TestCreateIssueReturnsURLWithoutLabels` / `TestCreateIssueReturnsGHFailure` / `TestCreateIssueRejectsBeforeCallingGH`）が `SplitNewIssue` を通らずタイトルと本文を直に渡していることを確認し、通っている箇所があれば分割後の値を渡す形に直す（`action.CreateIssue` の検査はこの change で変えない）

## 2. n の初期テキストと編集結果の検査（internal/ui）

- [ ] 2.1 `internal/ui/new.go` の `newKey` が `openEditor(routeNewIssue, "")` に渡している初期テキストを、`internal/action` の初期の下書きに変える。`internal/ui` に文言を書かない。「書式の案内を初期テキストに入れると、それがそのままタイトルになる」というコメントは、区切りに使うので入れてよいという内容に書き直す
- [ ] 2.2 `internal/ui/new.go` の `updateNewEdited` に、`SplitNewIssue` の `ok` が偽のときフッタへ `作成を中止しました（プレースホルダー行が見つかりません）` を出して確認画面へ進まない枝を、エラーの枝の次・タイトルが空の枝の前に足す（`new-issue`「編集結果を検査して作成の確認画面に移る」の 5 段の順）
- [ ] 2.3 `internal/ui/new_test.go` の `TestNewIssueRepoIsWhatTheScreenShows` で、スタブ `Editor` が受け取る `initial` が空文字列であることを見ている検証を、初期の下書きと一致することを見る形に書き換える（キュー画面・カード詳細・PR 詳細の 3 経路）
- [ ] 2.4 `internal/ui/new_test.go` の `TestNewIssueChecksEditedDraft` のスタブが返す固定文字列を、プレースホルダー行 2 本を含む下書きに書き換え（タイトルと本文がそろう / タイトルが空 / 本文が空 / マーカーを含む）、`プレースホルダー行が見つかりません` で中止するケースを 1 本足す。確認画面に進むケースでは、`タイトル: ` の行と本文の行にプレースホルダー行の文言が出ないことも見る
- [ ] 2.5 `internal/ui/new_test.go` の `TestNewConfirmKeys` と `TestNewIssueResultShowsURL` のスタブが返す固定文字列をプレースホルダー行入りに書き換え、`e` が受け取る `initial` が「プレースホルダー行を含む編集後の下書きそのもの」であることを見る形に直す（`y` で作られる issue の `Title` / `Body` は分割後の値のまま）
- [ ] 2.6 `internal/ui/answer_test.go` の `TestEditRouteSeparatesAnswerFromNewIssue` の「n で始めた編集は回答として扱わない」が渡している下書き `タイトル\n\n本文` をプレースホルダー行入りに書き換える（今の下書きでは作成の確認画面へ進めず、経路の検証が成立しない）

## 3. docs と仕上げ

- [ ] 3.1 `docs/mvp/mvp.md` キーバインド表の `n` の行の「1 行目がタイトル、以降が本文」を、タイトルと概要のプレースホルダー行で分ける形に書き換え、変更履歴に 1 行足す
- [ ] 3.2 `gofmt -l .` が空で、`go build ./... && go vet ./... && go test ./...` と `golangci-lint run ./...` が通ることを確認する
- [ ] 3.3 `openspec validate s29-new-issue-placeholders --strict` が通ることを確認する
