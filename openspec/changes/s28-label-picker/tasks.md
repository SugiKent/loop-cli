## 0. 先行 change との突き合わせ

- [ ] 0.1 着手の前に、`s26-issue-label-driven` のコードが `origin/main` に入っていることを確認する（この change のテストは s26 のフッタとヘルプの値を土台にする）。入っていなければ着手せず `blocked-by: change s26-issue-label-driven` で issue へ書き戻す
- [ ] 0.2 archive の直前に、`origin/main` の `openspec/specs/` を見て、この change の 6 つの MODIFIED（proposal の Impact の表と `gh-client` の 1 本）を**その時点の最新の版から写し直す**。先に archive された change の追記を消していないことを `git diff` で確かめる（MODIFIED は Requirement ブロック全体を置き換えるので、写し忘れると先行 change の変更が消える）

## 1. ラベルの取得と一括編集（internal/gh）

- [ ] 1.1 `internal/gh/types.go` に `RepoLabel`（`Name` / `Description` / `Color`）を足し（既存の `Label { Name string }` は変えない）、`internal/gh/decode.go` に `gh label list` の出力をデコードする関数を足し、`internal/gh/gh.go` の `GHClient` に `ListLabels` / `EditIssueLabels` / `EditPRLabels` を足して `Call` に `AddLabels` / `RemoveLabels` の欄を足す
- [ ] 1.2 `internal/gh/client.go` に 3 メソッドを実装する（`label list -R … --json name,description,color --sort name --order asc --limit 100`、`issue edit <n> -R … --add-label … --remove-label …`、`pr edit <n> -R … --add-label … --remove-label …`。フラグは 1 名前 1 回ずつ繰り返す。付ける・外すがどちらも空なら `gh` を実行しない）
- [ ] 1.3 `internal/gh/fake.go` の `ListLabels` で `labels.json` を読み、`ListLabels` / `EditIssueLabels` / `EditPRLabels` の呼び出しを `Calls` に記録する
- [ ] 1.4 `internal/gh/client_test.go` と `fake_test.go` に、実行する引数・デコード・空のときに `gh` を呼ばないこと・`Fake` の記録のテストを書く
- [ ] 1.5 手元の `gh` で `gh label list --help` と `gh issue edit --help` を読み、`--sort` / `--order` / `--limit` の既定と `--add-label` の繰り返し指定が spec の記述どおりかを確かめる（違っていたら spec の delta を直してから先へ進む）

## 2. fixture（internal/gh, cmd/loop-cli-dev）

- [ ] 2.1 fixture を手書きで作る。`internal/gh/testdata/fixtures/example/labels.json`（`stage:*` / `wip` / `blocked` / `question` / `propose` / `apply` / `archive` / `docs` / `ai-assess:requested` と、プロジェクト固有のラベル 1〜2 件。名前の昇順）と、`L` のテスト用の `internal/action/testdata/labels/`（`Labels` が `wip` の issue 150、`docs` の付いた issue 151、`labels.json` は `docs` と `wip` の 2 件）。ラベルの件数を変える 3 つ（`labels-empty/` = 0 件、`labels-three/` = 3 件、`labels-many/` = 30 件 `label-00`〜`label-29`）も同じ場所に置く。`../action/testdata/todo` と `testdata/merge` が同じやり方をしている
- [ ] 2.2 `internal/gh/capture.go` の `Capture` で `labels.json` を採り（issue / PR の繰り返しの後に 1 回）、`cmd/loop-cli-dev` の fixture capture の自己検査に `ListLabels` を足して要約のファイル数を直す
- [ ] 2.3 `internal/gh/capture_test.go`（map のキー 7 → 8、`progress` の順序）と `cmd/loop-cli-dev` の fixture のテスト（ファイル数 10 → 11）の期待値を直す

## 3. 送信の action（internal/action）

- [ ] 3.1 `internal/action` に、`Target` と「送信後にこうなっていてほしいラベル名の集合」を受け取る関数を足す（`ViewPR` / `ViewIssue` で読み直し → 名前の昇順で付ける並び・外す並びを作る → どちらも空なら書かずに戻る → `EditPRLabels` / `EditIssueLabels` を 1 回呼ぶ）
- [ ] 3.2 `internal/action` にテストを書く（付けると外すが 1 回の呼び出しにまとまる・PR は `EditPRLabels`・読み直した結果と一致していれば書かない・段階ラベルを拒否しない）

## 4. ラベル一覧画面の状態と遷移（internal/ui）

- [ ] 4.1 `internal/ui/labels.go` を新しく作り、画面の状態（ラベルの並び・選択位置・元の集合・今の集合・写し取った `Target`・戻り先）を定義する
- [ ] 4.2 `L` のハンドラを実装する（対象の決定と 3 つの写し取り、キャッシュの有無で分かれる 2 経路、合流点での「一覧を開く判定」）
- [ ] 4.3 取得のコマンドと結果のメッセージを実装する（リポジトリ名をキーにした表は `R` でも自動更新でも捨てない。0 件・取得失敗・画面が変わったときのステータス 3 種）
- [ ] 4.4 `internal/ui/detail.go` の `screen` にラベル一覧を足し、`internal/ui/model.go` のキー振り分けに「ラベル一覧画面の分岐 → `u` のハンドラ → `L` のハンドラ」の順で置く（一覧の分岐は `key == "u"` より前、`L` のハンドラは `a` の判定より前）
- [ ] 4.5 一覧画面のキー（`j` / `k` / `↑` / `↓` / `Space` / `Enter` / `Esc` / `q`、他は何もしない）と、`Esc` で戻るときの詳細の作り直しを実装する

## 5. 送信と描画（internal/ui）

- [ ] 5.1 `Enter` の処理と結果のメッセージを実装する（先に画面を戻り先へ戻す → 変更 0 件ならコマンドを返さずステータスだけ出す → そうでなければ書き込み中フラグを立てて `更新中` を出しコマンドを返す。結果は `+docs -wip` の整形・変更なし・失敗の赤の 3 種で、画面は変えない）
- [ ] 5.2 一覧画面の `View` を実装する（見出しと変更件数・4 種の状態の印・名前と説明・空行・フッタ、幅での切り詰め、選択に追従する表示位置）

## 6. ヒントとヘルプ

- [ ] 6.1 4 つのフッタのヒントに `L ラベル` を足し（`internal/ui/view.go` のキュー画面と `internal/ui/detail.go` のカード詳細は `t todo` の次、PR 詳細は `a 回答` の次）、`internal/ui/help.go` の `helpKeys` に `L  ラベルを一覧から付け外し` の行を `t` の直後で足す

## 7. テスト（internal/ui）

- [ ] 7.1 `L` の画面遷移のテストを書く（3 画面で開く・2 回目は `gh` を呼ばない・0 件はステータス・0 件で 2 回押しても開かない・取得失敗は赤・取得中に画面が変わったら開かない・書き込み中は何もしない・0 行 / 確認 / merge 確認 / ヘルプ / URL 一覧では何もしない・小文字の `l` は何もしない）
- [ ] 7.2 一覧画面のキーのテストを書く（`j` / `k` の移動・`Space` が `gh` を呼ばない・同じ行で 2 回押すと戻る・`Esc` が変更予定を捨てる・他のキーは何もしない・`q` で終了）
- [ ] 7.3 写し取りのテストを書く（一覧を出している間に `Cards` を差し替えた取得完了を渡しても、印と件数が動かず、送信先も入れ替わらない）
- [ ] 7.4 描画のテストを書く（見出しの変更件数・4 種の印・選択の印・件数が高さを超えるときの追従・フッタ）
- [ ] 7.5 送信のテストを書く（`Enter` で先に一覧が閉じる・`Fake.Calls` の引数・変更 0 件で `gh` を呼ばない・結果が届いても画面を奪わない・ステータス 3 種・`Cards` が変わらない）
- [ ] 7.6 既存テストのうち、キュー・カード詳細・PR 詳細のフッタのヒントとヘルプの行数を見ているものを新しい期待値に直す（`view_test.go` / `detail_test.go` / `help_test.go`。`s26-issue-label-driven` の実装後の値を土台にする）

## 8. 正本ドキュメント

- [ ] 8.1 `docs/mvp/mvp.md` のキーバインド表に `L`（ラベル一覧を開いて付け外し / 内部処理は `gh label list` と `gh issue edit` / `gh pr edit`）の行を足し、変更履歴に 1 行足す（`l` のカンバン列移動の行は変えない。画面構成のフッタ例は `u URL` も入っていないので変えない）
- [ ] 8.2 `docs/domain/issue-driven-sdd/human-turn-signals.md` の 3 か所を直し、変更履歴に 1 行足す
  - 冒頭の「`ai-assess:requested` を付け直すと … この TUI からは行わない」を、`L` から付け直せることに書き換える
  - 不変条件 1 を「ラベル集合の置換は行わない（`--add-label` / `--remove-label` だけを使う）。`t` は 1 ラベルずつ、`L` の送信は 1 回で複数ラベルを変える」に書き換える。あわせて、2 つの mutation が原子的でないこと、付けるほうが先に走るために意図した Routine が起動しない場合があることを添える
  - 不変条件 2 を「`t` の `stage:todo`、`s` の `stage:propose` に加えて、`L` で人が明示的に選んだラベルは書く」に書き換え、段階ラベルが 2 つになった issue では `t` が以後拒否されることを添える
- [ ] 8.3 `openspec/config.yaml` の `context` の「TUI が書くラベルは stage:todo と s の stage:propose のみ」を Q1 の結論に直す（後続の propose がこの context を読むため）

## 9. 仕上げ

- [ ] 9.1 `openspec validate --strict` と `go build ./... && go vet ./... && go test ./...` が通ることを確認する。あわせて `L` を押して一覧が出ることを開発環境で 1 度確かめる（Bubble Tea v2 の `KeyPressMsg.String()` が `Shift` + `l` を `"L"` で返すかは `R` の前例からの推定なので、実機で 1 回見る）
