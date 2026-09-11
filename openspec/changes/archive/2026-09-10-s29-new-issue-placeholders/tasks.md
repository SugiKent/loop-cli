# tasks: s29-new-issue-placeholders

Refs #17

## 0. 前提

- [x] 0.1 先行 change `s15-new-issue` が archive 済みであることを、`origin/main` に `openspec/specs/new-issue/spec.md` と `openspec/specs/new-issue-action/spec.md` があることで確認する。まだ無ければ着手せず、`blocked-by: change s15-new-issue` を issue へ書き戻す（proposal の「先行 change への依存」）
- [x] 0.2 archive の直前に、`origin/main` の `openspec/specs/new-issue/spec.md` を見て、この change の 3 つの MODIFIED（Requirement「n は画面の対象のリポジトリを作成先にしてエディタを開く」「編集結果を検査して作成の確認画面に移る」「作成の確認画面は作成先と下書きを出し、y で作成して Esc で中止する」）をその時点の最新の版から写し直し、先に archive された change の追記を消していないことを `git diff` で確かめる（MODIFIED は Requirement ブロック全体を置き換えるので、写し忘れると先行 change の変更が消える）。apply の時点（`origin/main` = `82a8857`）で突き合わせ済みで、3 つの MODIFIED は main の版と意図した差分だけが違う。残る `openspec/changes/` の change（`s26-close-issue-pr` / `s27-wrap-titles` / `s28-label-picker`）はいずれも `new-issue` / `new-issue-action` を触らないので、archive までに main の版が動く経路は無い

## 1. 下書きと分割（internal/action）

- [x] 1.1 `internal/action/newissue.go` に、2 本のプレースホルダー行の文言（`タイトル（この下の行に入力してください）` / `概要（この下に入力してください）`）と、`n` で開くエディタの初期の下書き（`<タイトルの行>\n\n<概要の行>\n\n`）を足す。初期の下書きは `internal/ui` から参照するので公開し、2 本の文言はこのパッケージの中だけで使う
- [x] 1.2 `SplitNewIssue` を `(title, body string, ok bool)` を返す形に書き換える。spec の 7 段の手順どおり、行に分けて各行の末尾の `\r` を落とし、前後の空白を落とした結果で 2 本の区切りを先頭から探し、区切り以外に文言と一致する行が残っていないかを見て、いずれかで外れたら `ok` を偽にして空文字列を 2 つ返す。タイトルは 2 本の区切りに挟まれた行から空行を捨てて半角空白 1 つで連結し、本文は概要の区切りより後を改行で連結して前後の空白を落とす
- [x] 1.3 `internal/action/newissue_test.go` の `TestSplitNewIssueTakesFirstLineAsTitle` を、名前とコメントごと新しい契約（プレースホルダー行で分ける）に書き換え、Requirement「プレースホルダー行 2 本を区切りにタイトルと本文を分ける」の 11 の Scenario を検証する（挟まれた行がタイトル / 空行なしで打ち込んだ形 / 本文は下すべて / 複数行の連結 / 消した下書き / 概要だけ無い / 区切りが逆順 / 書き足した行 / 行末の空白と CRLF で本文に `\r` が残らない / 案内の行を複製 / 案内の文言を本文に引用）。Requirement「エディタに渡す下書きはプレースホルダー行 2 本を持つ」の 2 つの Scenario（初期の下書きの中身 / それをそのまま分けるとタイトルも本文も空）も足す

## 2. n の初期テキストと編集結果の検査（internal/ui）

- [x] 2.1 `internal/ui/new.go` の `newKey` が `openEditor(routeNewIssue, "")` に渡している初期テキストを、`internal/action` の初期の下書きに変える。`internal/ui` に文言を書かない。「書式の案内を初期テキストに入れると、それがそのままタイトルになる」というコメントは、区切りに使うので入れてよいという内容に書き直す
- [x] 2.2 `internal/ui/new.go` の `updateNewEdited` に、`SplitNewIssue` の `ok` が偽のときフッタへ `作成を中止しました（プレースホルダー行が見つかりません）` を出して確認画面へ進まない枝を、エラーの枝の次・タイトルが空の枝の前に足す（`new-issue`「編集結果を検査して作成の確認画面に移る」の 5 段の順）
- [x] 2.3 `internal/ui/new_test.go` の `TestNewIssueRepoIsWhatTheScreenShows` で、スタブ `Editor` が受け取る `initial` が空文字列であることを見ている検証を、初期の下書きと一致することを見る形に書き換える（キュー画面・カード詳細・PR 詳細の 3 経路）
- [x] 2.4 `internal/ui/new_test.go:18` の `const newDraft`（コメントの「1 行目がタイトル、空行を挟んで本文」も）をプレースホルダー行入りの下書きに書き換える。この定数を通っているケース（確認画面まで進む十数本）はこれで追従する
- [x] 2.5 `internal/ui/new_test.go` の個別のリテラルを直す。`TestNewIssueChecksEditedDraft` の 3 ケース（タイトルが空 / 本文が空 / マーカーを含む）をプレースホルダー行入りに書き換えて `プレースホルダー行が見つかりません` で中止するケースを 1 本足し、確認画面に進むケースでは `タイトル: ` の行と本文の行にプレースホルダー行の文言が出ないことも見る。`TestNewConfirmKeys` の `e` の検証は、受け取る `initial` が「プレースホルダー行を含む編集後の下書きそのもの」であることを見る形に直す（`y` で作られる issue の `Title` / `Body` は分割後の値のまま）
- [x] 2.6 `internal/ui/answer_test.go:338`（`TestEditRouteSeparatesAnswerFromNewIssue` の「n で始めた編集は回答として扱わない」）と `internal/ui/todo_test.go:297`（`issue 作成中の t と a と m`）が渡している下書き `タイトル\n\n本文` を、プレースホルダー行入りに書き換える。どちらも `newIssueConfirm`（`new_test.go:29`）で作成の確認画面に進むことを前提にしており、今の下書きでは `ok` が偽になって `t.Fatalf` で落ちる

## 3. docs と仕上げ

- [x] 3.1 `docs/mvp/mvp.md` キーバインド表の `n` の行の「1 行目がタイトル、以降が本文」を、タイトルと概要のプレースホルダー行で分ける形に書き換え、変更履歴に 1 行足す
- [x] 3.2 `gofmt -l .` が空で、`go build ./... && go vet ./... && go test ./...` と `golangci-lint run ./...` が通ることを確認する
- [x] 3.3 `openspec validate s29-new-issue-placeholders --strict` が通ることを確認する
