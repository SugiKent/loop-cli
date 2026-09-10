## 0. 前提の確認

- [ ] 0.1 `origin/main` の `openspec/changes/` に `s26-issue-label-driven` が残っていないこと（archive 済みであること）を確認する。残っていたら着手せず `blocked-by: change s26-issue-label-driven` で issue へ書き戻す（この change の `card-detail` / `help-screen` / `queue-screen` の delta は s26 の delta を土台に写しており、s26 より先に archive すると s26 の変更を打ち消す）

## 1. ラベルの取得と一括編集（internal/gh）

- [ ] 1.1 `internal/gh/types.go` に `Label`（`Name` / `Description` / `Color`）を足し、`internal/gh/decode.go` に `gh label list` の出力をデコードする関数を足す
- [ ] 1.2 `internal/gh/gh.go` の `GHClient` に `ListLabels` / `EditIssueLabels` / `EditPRLabels` を足し、`Call` に `AddLabels` / `RemoveLabels` の欄を足す
- [ ] 1.3 `internal/gh/client.go` に 3 メソッドを実装する（`label list -R … --json name,description,color --sort name --order asc --limit 100`、`issue edit <n> -R … --add-label … --remove-label …`、`pr edit <n> -R … --add-label … --remove-label …`。フラグは 1 名前 1 回ずつ繰り返す。付ける・外すがどちらも空なら `gh` を実行しない）
- [ ] 1.4 `internal/gh/testdata/fixtures/example/labels.json` を手書きで作る（`stage:*` / `wip` / `blocked` / `question` / `propose` / `apply` / `archive` / `docs` / `ai-assess:requested` と、プロジェクト固有のラベル 1〜2 件。名前の昇順）
- [ ] 1.5 `internal/gh/fake.go` の `ListLabels` で `labels.json` を読み、`ListLabels` / `EditIssueLabels` / `EditPRLabels` の呼び出しを `Calls` に記録する
- [ ] 1.6 `internal/gh/client_test.go` と `fake_test.go` に、実行する引数・デコード・空のときに `gh` を呼ばないこと・`Fake` の記録のテストを書く

## 2. fixture 採取の追随（internal/gh, cmd/loop-cli-dev）

- [ ] 2.1 `internal/gh/capture.go` の `Capture` で `labels.json` を採る
- [ ] 2.2 `cmd/loop-cli-dev` の fixture capture の自己検査に `ListLabels` を足し、要約のファイル数の期待値を直す
- [ ] 2.3 `internal/gh/capture_test.go` と `cmd/loop-cli-dev/fixture` のテストの期待値（ファイル数 10 → 11）を直す

## 3. 送信の action（internal/action）

- [ ] 3.1 `internal/action` に、`Target` と「送信後にこうなっていてほしいラベル名の集合」を受け取る関数を足す（`ViewPR` / `ViewIssue` で読み直し → 名前の昇順で付ける並び・外す並びを作る → どちらも空なら書かずに戻る → `EditPRLabels` / `EditIssueLabels` を 1 回呼ぶ）
- [ ] 3.2 `internal/action` にテストを書く（付けると外すが 1 回の呼び出しにまとまる・PR は `EditPRLabels`・読み直した結果と一致していれば書かない・変更 0 件なら読み直しもしない・段階ラベルを拒否しない）

## 4. ラベル一覧画面（internal/ui）

- [ ] 4.1 `internal/ui/labels.go` を新しく作り、画面の状態（ラベルの並び・選択位置・送信後の集合・写し取った `Target`・戻り先）と、`L` のハンドラ（対象の決定と写し取り、キャッシュの有無で分かれる 2 経路、0 件と失敗のステータス）を実装する
- [ ] 4.2 `internal/ui/detail.go` の `screen` にラベル一覧を足し、`internal/ui/model.go` のキー振り分けに「ラベル一覧画面の分岐 → `L` のハンドラ」の順で置く（どちらも `a` / `t` / `m` / `o` / `?` の判定より前、URL 一覧の分岐の直後）
- [ ] 4.3 一覧画面のキー（`j` / `k` / `↑` / `↓` / `Space` / `Enter` / `Esc` / `q`、他は何もしない）と、`Esc` で戻るときの詳細の作り直しを実装する
- [ ] 4.4 取得のコマンドと結果のメッセージを実装する（`Model` にリポジトリ名をキーにしたラベルの表を持ち、`R` でも自動更新でも捨てない。届いたときに画面が変わっていたら一覧を開かず中止のステータスを出す）
- [ ] 4.5 送信のコマンドと結果のメッセージを実装する（送信中は書き込み中フラグを立て、結果を受けたら戻り先の画面に戻り、`+docs -wip` の形でステータスを出す）
- [ ] 4.6 一覧画面の `View` を実装する（見出しと変更件数・4 種の状態の印・名前と説明・空行・フッタ、幅での切り詰め、選択に追従する表示位置）

## 5. ヒントとヘルプ

- [ ] 5.1 `internal/ui/view.go` のキュー画面のヒントに `L ラベル` を `t todo` の次で足す
- [ ] 5.2 `internal/ui/detail.go` のカード詳細のヒントに `L ラベル` を `t todo` の次で、PR 詳細のヒントに `a 回答` の次で足す
- [ ] 5.3 `internal/ui/help.go` の `helpKeys` に `L  ラベルを一覧から付け外し` の行を `t` の直後で足す

## 6. テスト

- [ ] 6.1 `internal/ui` に `L` の画面遷移のテストを書く（3 画面で開く・2 回目は `gh` を呼ばない・0 件はステータス・取得失敗は赤・取得中に画面が変わったら開かない・書き込み中は何もしない・0 行 / 確認 / merge 確認 / ヘルプ / URL 一覧では何もしない・小文字の `l` は何もしない）
- [ ] 6.2 `internal/ui` に一覧画面のキーのテストを書く（`j` / `k` の移動・`Space` が `gh` を呼ばない・2 回押すと戻る・`Esc` が変更予定を捨てる・他のキーは何もしない・`q` で終了）
- [ ] 6.3 `internal/ui` に描画のテストを書く（見出しの変更件数・4 種の印・選択の印・件数が高さを超えるときの追従・フッタ）
- [ ] 6.4 `internal/ui` に送信のテストを書く（`Fake.Calls` の `EditIssueLabels` / `EditPRLabels` の引数・変更 0 件・結果のステータス 3 種・送信後に戻り先へ戻る・`Cards` が変わらない）
- [ ] 6.5 既存テストのうち、キュー・カード詳細・PR 詳細のフッタのヒントとヘルプの行数を見ているものを新しい期待値に直す（`s26-issue-label-driven` の実装後の値を土台にする）

## 7. 正本ドキュメント

- [ ] 7.1 `docs/mvp/mvp.md` のキーバインド表に `L`（ラベル一覧を開いて付け外し / 内部処理は `gh label list` と `gh issue edit` / `gh pr edit`）の行を足し、変更履歴に 1 行足す（`l` のカンバン列移動の行は変えない）
- [ ] 7.2 `docs/domain/issue-driven-sdd/human-turn-signals.md` の 3 か所を直し、変更履歴に 1 行足す
  - 冒頭の「`ai-assess:requested` を付け直すと … この TUI からは行わない」を、`L` から付け直せることに書き換える
  - 不変条件 1 を「ラベル集合の置換は行わない（`--add-label` / `--remove-label` だけを使う）。`t` は 1 ラベルずつ、`L` の送信は 1 回で複数ラベルを変える」に書き換える
  - 不変条件 2 を「`t` の `stage:todo`、`s` の `stage:propose` に加えて、`L` で人が明示的に選んだラベルは書く」に書き換え、誤って段階ラベルや `question` を触ったときに何が起きるかを添える

## 8. 仕上げ

- [ ] 8.1 `openspec validate --strict` が緑になることを確認する
- [ ] 8.2 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
