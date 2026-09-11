## 1. 前提の確認

- [ ] 1.1 `s33-colorful-labels` が `origin/main` の `openspec/specs/queue-screen/spec.md` に入っていることを確認する（Requirement「ラベル名は GitHub のラベル色を背景に、輝度で選んだ黒か白を文字にして描く」と「Model はリポジトリごとのラベル色を持つ」が main の spec にあること）。無ければこの change は着手できない（design.md D4。色の規則を二重に定義しないため）

## 2. 段階ラベルの表記と並び

- [ ] 2.1 `internal/ui/rows.go`: `row` に主体の段階ラベル（接頭辞を落とす前の名前）と、列に出す語を持たせる。`model.IssueStages` / `model.PRStages` に主体の種別と方式を渡して段階順の先頭を採り、`sdd` 方式の issue では `stage:` を落とした語を、段階が無ければ `-` を作る。方式は `Model.modes` から引くので、`buildRows` が方式の表を受け取る形にする
- [ ] 2.2 `internal/ui/rows.go`: `buildRows` の並びをタブで分ける。進行中タブは `Repo` 昇順 → 段階順（`todo` → `propose` → `apply` → `archive`、`label` 方式は `To Do` → `In Progress` → `Done`、段階なしを最後）→ 主体の `UpdatedAt` 降順 → 番号昇順。他の 3 タブは今の 4 キーをそのまま残す
- [ ] 2.3 `internal/ui/rows_test.go`: `queue-screen` MODIFIED Requirement「Model は Card をタブ別に並べ、選択行を 1 つ持つ」の Scenario「進行中タブはリポジトリ順、同じリポジトリでは段階順」を通す。あわせて Scenario「タブ内は優先度順、同じ優先度なら新しい順」が今やるタブで今までどおり通ることを確認する
- [ ] 2.4 `internal/ui/rows_test.go`: 段階の語の表（ADDED Requirement「進行中タブの種別の列は段階ラベル名を出す」の 4 行）を、`sdd` の issue（`stage:archive` → `archive`）/ `sdd` の PR（`propose` → `propose`）/ `label` の issue（`In Progress` → `In Progress`）/ `label` の PR（`-`）/ 段階なし（`-`）/ `docs` だけの PR（`-`）で検証する

## 3. 種別の列と色

- [ ] 3.1 `internal/ui/view.go` の `tableRow`: 進行中タブでは種別の列に `Kind()` の代わりに段階の語を出す。列幅は 10 のまま変えず、収まらない語は既存の `pad` に任せる（`In Progress` は `In Progr…` になる）
- [ ] 3.2 `internal/ui/view.go` の `tableRow`: 段階の語に、`s33-colorful-labels` が置いたラベル名を描く関数で色を付ける。色は接頭辞を落とす前のラベル名で主体のリポジトリのラベル色の表から引く。`-` には色を付けない。行全体の色を持つタブ（質問 / 方針 / merge / todo 候補 / 異常）では今までどおり列に色を置かない
- [ ] 3.3 `internal/ui/view_test.go`: ADDED Requirement「進行中タブの種別の列は段階ラベル名を出す」の Scenario 4 件（段階ラベルが種別の列に出る / 段階ラベルの無い主体は `-` になる / `label` 方式の段階ラベルは列幅で切られる / 他のタブの種別の列は変わらない）を通す
- [ ] 3.4 `internal/ui/view_test.go`: ADDED Requirement「進行中タブの段階ラベル名に色を付ける」の Scenario 3 件（段階ラベル名に背景色が付く / 色が引けない段階ラベル名は色を付けずに描く / 段階ラベルが無い行の `-` には色が付かない）と、MODIFIED Requirement「表の行は優先記号・種別・リポジトリ・番号・タイトル・経過を種別の色で出す」の Scenario「進行中の行は行全体の色を持たない」を通す。色の検証は「色を指定するエスケープ + 語 + リセット」が連続して現れることで書く（エスケープが行のどこかにある条件だけを見ると、幅で切り詰められて語が 1 文字も描かれていない場合に偽の合格が出る）

## 4. リポジトリの見出し行

- [ ] 4.1 `internal/ui/view.go`: 見出し行を組み立てる関数を足す（`── <owner/name> ── <n> 件 ` の後を端末の幅まで `─` で埋め、埋める本数は `max(幅 − 見出しの表示幅, 0)` で求め、収まらない幅では `ansi.Truncate` で切って `…` を付けない。幅 0 以下では空文字列）。関数の戻り値を直接読むテストを、幅 80 / 幅 20（名前が入らない幅）/ 幅 0 の 3 通りで書く
- [ ] 4.2 `internal/ui/view.go` の `tableLines`: 進行中タブで、行があるリポジトリが 2 つ以上あるときだけ、`repo` が変わる位置に見出し行を挟む。件数は Card の枚数を数える。`m.rows[m.tab]` の要素は増やさず、描画のときにだけ挟む（design.md D3。カーソルと選択の意味を変えないため）
- [ ] 4.3 `internal/ui/view_test.go`: ADDED Requirement「進行中タブの表はリポジトリごとに見出し行で区切る」の Scenario 5 件（2 つのリポジトリは見出し行で区切られる / リポジトリが 1 つだけなら見出し行を出さない / 見出し行は選択の対象にならない / 他のタブには見出し行を出さない / 狭い端末では見出し行を幅で切る）を通す。見出し行の幅は末尾の空白を落としてから測る

## 5. 表示状態を手元で再現する seed

- [ ] 5.1 `internal/ui/testdata/in-progress/` に fixture ディレクトリを 1 つ置く（`internal/fetch/testdata/multirepo` と同じ形。`search-issues.json` / `search-prs.json` / `labels.json` / 各 issue と PR の JSON）。`org/app` と `org/web` の 2 リポジトリぶんの進行中の Card を、段階が `propose` / `apply` / `archive` / 段階なしに散るように作る（`stage:X` + `wip` の issue で規則 1、最新コメントが人の PR で規則 2 / 3）。`internal/gh/testdata/fixtures/` には置かない（`internal/classify/fixture_test.go` の期待値表と `internal/gh/fixtures_test.go` の全件走査が対象にするため）
- [ ] 5.2 `internal/ui/view_test.go`: 5.1 の fixture を `gh.NewFake` と `fetch.Fetch` に通して得た `Result` で進行中タブの `View` を描き、見出し行 2 本・段階の語・行の並びが揃った状態を 1 本のテストで固定する（画面全体の見た目を 1 か所で確かめられるようにする）

## 6. 利用者向けの説明

- [ ] 6.1 `README.md`「画面とキー操作」: 行の列の説明（「行の列は 優先 / 種別 / リポジトリ / 番号 / タイトル / 経過」の段落）に、進行中タブでは種別の列が段階ラベル名になり GitHub のラベル色が付くこと、表がリポジトリごとに見出し行で区切られること（リポジトリが 1 つだけなら区切らないこと）を足す

## 7. 通し確認

- [ ] 7.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [ ] 7.2 `golangci-lint run` が通ることを確認する
- [ ] 7.3 `openspec validate s35-queue-grouping --strict` が通ることを確認する
