issue: #42

## Why

進行中タブの行は、どれも同じ見た目で並ぶ。このタブに入るのは `SituationInProgress` だけなので
（`internal/model/model.go:93-107`）、種別の列は常に `進行中`、優先度は常に 7（同 `:69-91`）、
行の色は `kindStyle` に `進行中` の項目が無いので無色（`internal/ui/view.go:29-35`）。
並びは第 1 キーの優先度が全行同じで効かず、経過の新しい順になってリポジトリが交互に現れる
（`internal/ui/rows.go:71-98`）。

分類はこのタブの中を 8 通りに分けている（`#n は AI が作業中` / `#n は回答済み。sweep 待ち` /
`#n は question のみ。dispatcher の回収待ち` / `#n は段階ラベルの書き直し中。sweep 待ち` /
`#n は進行中` / `PR #n は auto-fix が受け取り中` / `PR #n は回答済み。worker が受け取り中` /
`PR #n は AI 評価待ち`。`internal/classify/classify.go`）。未 archive の `s30-other-grace` が
入ると 9 通りになる（`PR #n はどの局面にも当たらない（更新から <M>m は様子見）`）。

その 8〜9 通りの違いは **キューの表に出ていない**。違いを運ぶ `Result.Summary` は
デスクトップ通知の本文（`internal/ui/notify.go:53`）とカード詳細のヘッダ
（`internal/ui/detail.go:181-182`）には出ているのに、表を組む `tableRow` は
`Situation.Kind()` しか読まない（`internal/ui/view.go:211-262`）。`loop-cli-dev classify` の
キューの列も 優先度 / 種別 / リポジトリ / 番号 / タイトル / 経過 の 6 列で、`Summary` を含まない
（`cmd/loop-cli-dev/classify.go:160`、`openspec/specs/dev-cli/spec.md`）。

段階も同じで、表で読めるのは主体が PR の行だけである。PR のタイトルが
`[propose] #35 …` の形をしているためで（mvp.md の画面構成の例がそう設計されている。
`docs/mvp/mvp.md:55-64`）、主体が issue の行には段階がどこにも出ない。カード詳細は
`段階: stage:propose` を出しているので（`internal/ui/detail.go:186-188`）、
一覧だけが段階を見せていない。

## What Changes

- 進行中タブの表を**リポジトリごとに区切る**。グループの先頭に、リポジトリ名と件数を載せた見出し行を
  1 行置く（`── <owner/name> ── <n> 件 ────…`）。行があるリポジトリが 1 つだけのときも出す
- 進行中タブの**並びをリポジトリ優先に変える**。並び替えのキーを、リポジトリ昇順 → 段階順 →
  経過の新しい順 → 番号昇順の 4 本にする。優先度は進行中タブでは全行同じなので、
  第 1 キーから外れても並びは崩れない
- 進行中タブの**種別の列に 1 語の状態を出す**。今は全行 `進行中` で情報が無い列を使う。
  出すのは段階ラベル名（`propose` / `apply` / `archive` / `todo`、無ければ `-`）とする
- その 1 語に**色を付ける**。ラベル名の色の規則は `s33-colorful-labels`（PR #37）が
  `queue-screen` に入れるものを引き、この change では定義し直さない
- 今やる / バックログ / 異常のタブは変えない
- 表の高さの配分・スクロールの有無・キー操作・カーソルの意味・プレビュー・ヘッダ・フッタは変えない。
  見出し行のぶん表に出るカードが減ることは受け入れる

## Capabilities

### New Capabilities

（無し）

### Modified Capabilities

- `queue-screen`: 進行中タブのタブ内の並び順（Requirement「Model は Card をタブ別に並べ、選択行を 1 つ持つ」）と、
  種別の列の中身・行の色（Requirement「表の行は優先記号・種別・リポジトリ・番号・タイトル・経過を種別の色で出す」）を
  改め、リポジトリの見出し行と列の色の Requirement を足す

列に出す 1 語を段階ラベルにしたので、`human-turn-classify` と `card-model` は Modified に入らない。

## Impact

- `internal/ui/rows.go`: 進行中タブだけ並び替えのキーを変え、行に 1 語の状態を持たせる。
  `buildRows` が運用方式の表を引数で受け取る形になる
- `internal/ui/model.go`: `buildRows` の呼び出し 2 か所に方式を渡す。`fetchedMsg` の中では
  `m.rows = buildRows(...)`（`:202`）が `m.modes = msg.res.Modes`（`:204`）より**前**にあるので、
  順序を直すか `msg.res.Modes` を直接渡す。`New` のスナップショット経路（`:135`）には方式が無い
- `internal/ui/view.go`: `tableLines` がグループの見出し行を挟み、`tableRow` が進行中タブで
  種別の列を差し替える
- `internal/ui/rows_test.go` / `view_test.go`: 並び順・見出し行・列の中身と色の期待値
- `internal/ui/testdata/`: 進行中タブの表示状態を再現する fixture を 1 つ足す
- `internal/model` / `internal/gh` / `internal/fetch` / `internal/classify` / `internal/action`: 変更しない
- `README.md`: 「画面とキー操作」の行の列の説明
- 依存の追加は無い

## 確定した判断

- **進行中タブの行は 1 つも色が付いていない。** `kindStyle`（`internal/ui/view.go:29-35`）は
  質問 / 方針 / merge / todo 候補 / 異常の 5 種別にしか項目が無い。issue #42 の「色もつけて」という
  依頼は、このタブでは**色が 1 つも無いところへ色を足す作業**を指す
- **進行中タブでは優先度が第 1 キーとして働いていない。** `Priority` / `Tab` / `Kind()` はすべて
  `Situation` から導かれ（`internal/classify/classify.go:18-20`、`internal/model/model.go:69-128`）、
  `Tab` が `進行中` になるのは `in-progress` だけなので `Priority` は全行 7。だから `buildRows` の
  第 1 キーをリポジトリに差し替えても、このタブの中では並びの意味が変わらない。**他のタブでは変わる**
  ので、対象を進行中タブだけに絞った
- **段階ラベルが 2 件ある主体が進行中タブに入り得る。** `classify.Issue` の規則 4（`:43-46`）と
  規則 5（`:47-50`）は `len(stages)` を見ずに `in-progress` を返し、`len(stages) >= 2` の局面 F の
  判定（`:64`）はその**後ろ**にある。`classify.PR` も同じで、規則 2 / 3（`:99-110`）が先、
  F（`:133`）が後。つまり `stage:propose` + `stage:apply` + `question` の issue は進行中タブに
  2 段階のまま入る。列の値と並びの第 2 キーは、この場合を定義しておく必要がある
- **未 archive の `s30-other-grace` が、段階ラベルを持たない行を進行中タブの常連にする。**
  この change は `origin/main` のツリーに入っており（`openspec/changes/s30-other-grace/`）、
  「PR を作った直後でラベルが無い / checks が pending / grill 中 / `ai-assess:requested` が付く前」の
  `other` を猶予（既定 30 分）の間だけ進行中に置く。**そのような PR に段階ラベルは付いていない**ので、
  列に段階ラベルを出す以上、猶予中の行は全部 `-` と出る
- **`s30-other-grace` は進行中タブの表示を変えない前提で書かれている。** 同 change の Impact は
  「`internal/ui`: 変更なし（進行中タブの表示はそのまま）」と宣言しており、この change がその前提を
  更新する。`queue-screen` の delta は互いに触らないので spec の衝突は無い
- **段階ラベルは新しい判定を作らずに引ける。** `model.IssueStages(mode, labels)` /
  `model.PRStages(mode, labels)`（`internal/model/model.go:212-226`）が段階順に返し、
  方式は `Model.modes`（`queue-screen`「運用方式は取得のたびに判定した結果を持つ」）にある。
  一方、**「何を待っているか」は分類しか知らない。** 規則 1 / 4 / 5 / 6 はどれも `stage:propose` の
  issue に当たり得るので、段階ラベルでは「健全に作業中」と「sweep 待ち」を区別できない
- **状態の語を出すには分類の結果に値を足すことになる。** `classify` が返すのは `Situation` と
  `Summary` の 2 つだけで、規則の番号は残らない。`Summary` の文面を `internal/ui` で解析するのは
  判定の二重化になる（CLAUDE.md「パターン競合は平均化しない」）。`Result` に値を足すと
  `human-turn-classify` の「優先度・タブ・種別・要約の表」を MODIFIED することになり、
  `s30-other-grace` が同じ Requirement を MODIFIED しているので順番待ちになる
- **色は `s33-colorful-labels` の規則を引く。** `routines-setup` が作るラベル色の実値は
  `stage:todo` `fbca04` / `stage:propose` `0e8a16` / `stage:apply` `1d76db` / `stage:archive` `5319e7`
  （`internal/gh/testdata/fixtures/example/labels.json`）。この 4 色を**文字色**に使う案は、
  PR #37 の design D3 が計算で否決している（黒背景でも白背景でも AA を満たす色は相対輝度 0.175〜0.183 に
  限られ、見分けの付く 4 色が入らない）。実測でも `5319e7` は黒に対して 2.6、`fbca04` は白に対して
  1.6 になる。背景色に回して輝度から黒白の文字色を選ぶ規則は PR #37 が `queue-screen` に入れるので、
  この change は引く側に回り、実装はその archive を前提にする
- **PR #37 の delta の文面は、この change が上書きする。** 同 change は「キューの表にラベル列は無い」
  「キューの表にラベル列は足さない（列を足す提案が来たときにそこで止まる）」と書いている。
  この change がその提案なので、進行中タブについてその前提を明示的に更新する。
  「行全体に色を付ける描画の中には色を置かない」という制約は当たらない。進行中の行は
  `kindStyle` に項目が無く行全体の色を持たないので、内側のリセットで外側の背景が落ちる問題が起きない
- **見出し行はリポジトリが 1 つでも出す。** 「2 つ以上のときだけ出す」とデータで分岐させると、
  自動更新（既定 120 秒）で 2 つ目のリポジトリの行が 1 枚増減するたびに見出しが出入りし、
  表の全行が 1 行ずれる。カーソルの添字は変わらないので、利用者が何もしていないのに選択行が
  画面上を動く。単一リポジトリの利用者が 1 行失うより、表示が揺れないことを採る
- **見出し行は選択できない。** カーソル（`Model.cursor`）が指すのは「何枚目の Card か」で、
  タイトルの折り返しで 1 枚が複数行になるため描画行の数とは既に一致していない。`tableLines` が
  見出し行を描画のときに挟むだけなので、`j` / `k` の移動も、プレビューの対象も、
  `a` / `t` / `m` / `c` / `o` / `u` の対象決定も変わらない
- **表にスクロールを足す手立ては既にある。** URL 一覧（`internal/ui/urls.go:157`）とラベル一覧
  （`internal/ui/labels.go:253`）は `start := max(cursor-h+1, 0)` で、スクロール位置の状態を持たずに
  選択行へ追従している。ただしキューの表は 1 枚の Card が複数行になるので、窓の起点を
  窓の起点を描画行の添字で取り直す必要がある（Card の添字では、折り返した行を数え落とす）。
  この change では足さない（「明示的に延期した判断と残るリスク」に理由を書く）
- **`docs/mvp` の沈黙は制約ではない。** CLAUDE.md は「`docs/mvp` は背景と当初の判断を知るために読む。
  そこに無い仕様は issue と `openspec/` で決める」と書いている。mvp.md のキーバインド表に表の
  スクロールが無いことや、`進行中` の色が書かれていないことは、やらない根拠にしない

## 回答で確定した判断（PR #48 のコメント、2026-09-12「全て A で」）

- **種別の列に出す 1 語は段階ラベルにする**（`todo` / `propose` / `apply` / `archive`、
  `label` 方式は `To Do` / `In Progress`、無ければ `-`）。`model.IssueStages` / `model.PRStages` で
  引けるので `internal/classify` と `internal/model` に手を入れず、この change だけで完結する。
  分類の状態の語（`sweep 待ち` など）を出す案は、分類の結果に値を足すことになり
  `human-turn-classify` の同じ Requirement を `s30-other-grace` が MODIFIED しているため見送った。
  代償は「明示的に延期した判断と残るリスク」に書く
- **区切りと色を入れるのは進行中タブだけにする。** 今やるタブは優先度順の 1 本のキューのまま残す。
  今やるタブをリポジトリで区切ると、「先頭から捌けば優先度の高い順」という README 冒頭の約束が崩れ、
  `!!` の質問が 2 番目のリポジトリの見出しの下に沈む。進行中タブは優先度が全行同じなので、
  区切っても失うものが無い。異常タブを対象にしなかったので、黄の背景の内側に色を置く問題も起きない
- **区切りはリポジトリの 1 段にする。** ステータス（段階）は種別の列と並びの第 2 キーで示す。
  リポジトリ → ステータスの 2 段にすると、3 リポジトリ × 3 ステータスで見出しが 12 行になり、
  高さ 24 の端末（表は 11 行）でカードが 1 枚も見えなくなる。1 段なら 3 リポジトリで見出し 3 行、
  カードに 8 行残る。同じ段階の行はリポジトリの中で隣り合うので、見出しが無くてもまとまりは読める

## 明示的に延期した判断と残るリスク

- **段階ラベルでは「詰まっているか」が区別できない。** 進行中の規則 1（`wip` あり = AI が健全に
  作業中）・規則 4（人が回答済みで sweep 待ち）・規則 5（question だけの残骸）・規則 6（ラベル
  書き直し中）はどれも `stage:propose` の issue に当たり得るので、列はどれも `propose` になる。
  分類の結果に状態を表す値を足せば解けるが、`human-turn-classify` の同じ Requirement を
  `s30-other-grace` が MODIFIED しているので順番待ちになる。段階ラベルで一覧がどこまで読めるように
  なるかを見てから、別 issue で決める
- **`s30-other-grace` の猶予中の PR は列が `-` になる。** 同 change は「PR を作った直後でラベルが
  無い / checks が pending / grill 中」の PR を猶予（既定 30 分）の間だけ進行中タブに置く。
  そのような PR に段階ラベルは付いていないので、その行について列は何も語らない。
  この change の価値は `s30-other-grace` の後で目減りする
- **表にスクロールを足さない。** 見出し行のぶん、選択行が表の外に出る場面が増える。今も
  タイトルの折り返しで同じことが起きており（`queue-screen`「1 枚の Card の途中の行までしか
  表示されないことがある」）、`cut`（`internal/ui/view.go:296-304`）は先頭から切るだけで追従しない。
  選択行が見えないまま `t` を押すと確認画面を通らずにラベルを書き込む（`internal/ui/todo.go`。
  `m` / `c` は確認画面が対象を見せる）。追従を足す手立ては URL 一覧とラベル一覧にあるが、
  キューの表は 1 枚の Card が複数行になるので窓の起点を描画行で取り直すことになり、
  この change の範囲を超える。読みにくさが残るかを見てから別 issue で決める
- **`loop-cli-dev classify` の「種別」列とは意味がずれる。** dev CLI の列は `Situation.Kind()` の
  ままなので、この change の後は TUI の進行中タブと dev CLI で同名の列が別の意味になる。
  dev CLI は分類の検証用（V-1）で人が一覧を捌く道具ではないので、揃えない
- **段階ラベル名の接頭辞をカード詳細と揃えない。** 一覧では `stage:` を落として `propose` と出し、
  カード詳細は `段階: stage:propose` を出したままにする（`internal/ui/detail.go:186-188`）。
  一覧は列幅 10 に収める必要があり、詳細は幅に余裕がある
- **表の最終行が見出し行になることを許す。** `cut` が表の高さで切るので、グループの行が 1 行も
  残らない見出しが末尾に出ることがある。1 行を捨てる規則を足すほどの害が無い
