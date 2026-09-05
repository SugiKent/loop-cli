## Context

s03 の `GHClient` に `CommentPR` / `CommentIssue` があり、`Client` は `gh pr comment <n> -R <repo> --body-file -` / `gh issue comment <n> -R <repo> --body-file -` を本文を標準入力で渡して実行する。`Fake` は書き込みを `Calls` に記録する。s05 の `internal/model` に `Comment.AI`（本文マーカーで判定）、`ParseQuestions`（`## Q<n>.` の見出しと `選択肢 <Letter>（推奨）: …`）、`LatestBlockedBy`（各行を `strings.TrimSpace` して `blocked-by:` で始まる行）がある。s02 の `Config.Editor` は `$EDITOR` を展開済みで、空のときの扱いは s10 に委ねられている。
s08 の `internal/ui` は `New(fetcher)` で取得関数だけを受け取り、キュー画面のキー処理は `a` を「何もしない」にしている。s09 が画面の状態（キュー / カード詳細 / PR 詳細）と詳細の対象（Card のコピー）を足し、`a` を詳細画面でも効かせるかは s10 に委ねている。s08 / s09 はレビュー中で、この change は両者の specs / design を前提にする。
human-turn-signals.md「人が書き込むときの不変条件」のうち、コメント投稿に関わるのは 2（TUI が書くラベルは 2 つに限る）、4（書き先 3 種類を混同しない）、7（`<!-- routine -->` を書かない）、8（`blocked-by:` 行を含めない。投稿前に検出して警告）である。mvp.md「カード詳細」が回答テンプレート（`## Q1.` と選択肢をパース、`（推奨）` を既定値、issue の `blocked-by: human` はパースできた分だけ、`<!-- routine -->` を含めない）を、キーバインド表が `a`（`$EDITOR` を開く。`gh pr comment -R … --body-file` / `gh issue comment`。ラベルは触らない）を定めている。D-002 の「書き込み後は対象 1 件だけ再取得する」は s18 の担当。validation-plan.md は書き込みの検証手段を未定としている。
`internal/action` はまだ無い。D-003 の内部構成は `action/`（ラベル・コメント・merge の書き込み。不変条件 1〜7 を持つ）を定めている。

## Goals / Non-Goals

**Goals:**
- キュー画面と詳細画面の `a` で、対象（PR か issue か）に応じた書き先に、テンプレート入りのエディタで書いた回答を投稿できる（主要フロー A / B）
- 不変条件 4 / 7 / 8 と「ラベルを触らない」を `internal/action` の Requirement として持ち、`Fake.Calls` で引数と呼び出し回数を検証できる
- エディタ起動を差し替え可能にし、投稿までのフロー（検査・確認・投稿・結果表示）を `gh` もエディタも起動せずにテストできる

**Non-Goals:**
- この change は review thread への返信 `A` を実装しない。書き先の 3 種類目として s16 が担当する。`t` は s11 が、`m` は s14 が、`n` と `s` は s15 が担当する
- この change は書き込み後の対象 1 件再取得を実装しない（s18 が担当する）。投稿後に自動で全件を再取得することもしない
- この change は live での投稿検証を計画しない。validation-plan.md が検証手段を未定としているので、未決事項に記す
- エディタの代替（TUI 内のテキスト入力）。mvp.md は `$EDITOR` を開くと定めている

## Decisions

### ファイル構成

```
internal/action/action.go        # Target / Comment / BlockedByLines / ErrEmptyBody / ErrRoutineMarker
internal/action/template.go      # AnswerTemplate
internal/action/action_test.go   # Fake の Calls で書き先・拒否・検出を検証
internal/action/template_test.go # テンプレートの組み立て
internal/ui/editor.go            # Editor 型 / ExternalEditor / editedMsg / 引数の組み立てと結果の読み取り
internal/ui/editor_test.go       # 空の editor / 引数の組み立て / 一時ファイルの読み取りと削除
internal/ui/answer.go            # 回答の状態、対象の決定、a のキー処理、編集結果の検査、確認画面のキーと View、投稿コマンドと postedMsg
internal/ui/answer_test.go       # a → エディタ → 検査 → 確認 → 投稿 の各 Scenario
internal/ui/model.go             # New の引数、client / editor / 回答の状態、a と確認画面の振り分け（変更）
internal/ui/view.go              # フッタに a 回答、確認画面の View への振り分け（変更）
cmd/sugi-loop/main.go            # ui.New に client と ui.ExternalEditor(cfg.Editor) を渡す（変更）
```

### action 層は「書き込みの直前」だけを持つ

`internal/action` は `gh.GHClient` を受け取って書き込みメソッドを呼ぶ薄い層で、Bubble Tea も `internal/ui` も知らない。持つのは不変条件の判定（空 / マーカー / `blocked-by:`）と書き先の分岐だけで、状態を持たない純粋な関数にする。UI 側の検査（Requirement「編集結果を検査してから投稿する」）は `action.BlockedByLines` と `action.Comment` の判定と同じ関数・同じ文字列を使い、判定を 2 か所に持たない。
`Comment` が `blocked-by:` 行を拒否しないのは、不変条件 8 が「検出して警告する」であり、引用のつもりの人が確認のうえで投稿する余地を残すため。マーカー（不変条件 7）は「書かない」であり余地が無いので `Comment` が拒否する。
テンプレート（`AnswerTemplate`）も `internal/action` に置く。回答本文の組み立てであり、`tea` を import せず単体でテストできる。`internal/model` に置く案は、s05 が「テンプレートの組み立ては s10」と切り分けており、`model` は分類に使う値とパーサだけを持つ方針に合わない。

### `New` に `gh.GHClient` と `Editor` を足す

書き込みには `client` が、エディタ起動には差し替え可能な `Editor` が要り、どちらも `Model` の外から入れるしかない。s08 は「`Model` は取得関数だけを受け取る」とし、`internal/ui` が `internal/gh` に依存しない利点を挙げたが、s09 が `gh.PRMergeState` / `gh.ReviewThread` の型を表示に使っており、その利点は既に無い。`client` を直接渡せば s11（`t`）/ s14（`m`）/ s15（`n` / `s`）が同じ `client` で `internal/action` の関数を呼ぶだけで済む。書き込みごとに閉じた関数を渡す案は、後続 change のたびに `New` の引数が増える。
`New(fetcher Fetcher, client gh.GHClient, editor Editor)` の 3 引数にし、s08 の Requirement を MODIFIED で写して署名だけ変える。s13 が stale 表示のためにさらに引数を足す予定なので、archive の順は s08 → s09 → s10 → s13。

### `Editor` はコマンドを返す関数

実機のエディタは Bubble Tea の外部プロセス実行の仕組み（描画を止め、端末をプロセスに渡し、終了時にコールバックの結果をメッセージとして戻す）で起動する必要があり、これは `tea.Cmd` として実行される。同期的に文字列を返す関数では表現できない。よって `Editor` は `func(initial string) tea.Cmd` とし、返すコマンドが `editedMsg{text string; err error}` を返す。テストのスタブは `func(initial string) tea.Cmd { return func() tea.Msg { return editedMsg{text: "Q1: A"} } }` の形で書け、`initial` を記録すればテンプレートも検証できる。
実機用の `ExternalEditor(command string) Editor` は、(1) 一時ファイルを作って `initial` を書く、(2) `strings.Fields(command)` の先頭を実行ファイル、残りを引数、末尾に一時ファイルのパスを足した `*exec.Cmd` を組む、(3) 外部プロセス実行のコマンドを、終了時に一時ファイルを読んで削除して `editedMsg` を返すコールバック付きで返す、の 3 段に分け、(2) の引数の組み立てと (3) のコールバック（`func(path string, runErr error) editedMsg`）を非公開関数として単体でテストする。外部プロセス実行のコマンドそのものはプログラムの外で実行できないため、テストしない（tasks の手動確認で見る）。
シェルを通さないのは、`editor: code --wait` のような空白区切りの引数を素直に扱うためと、設定値をシェルに評価させないため。引用符やパイプを含む `editor` は対象にしない。
`command` が空のときは `ExternalEditor` が返す `Editor` がエラーの `editedMsg` を返すコマンドを返す。`Model` に「エディタが設定されているか」を持たせず、`a` のフローがエラー表示に一本化される。

### `a` を押した直後の状態と確認画面

`Model` に回答の状態 `answer{target action.Target; label string; draft string; reason（blocked-by / マーカー）; from（戻り先の画面）}` を足し、`a` を押した時点で `target` と `label` を埋める。編集完了のメッセージは `target` を持たないので、`Model` 側で保持する。確認画面は s09 の `screen` に 4 つ目の値として足し、s09 と同じく 1 つの `Model` の中で `switch screen` する（別の Bubble Tea Model を持たない）。
確認画面を全画面にするのは、警告と下書きの全行を読めるようにするため。フッタだけに警告を出すと `blocked-by:` の行がどれか分からない。確認画面は下書きを残りの高さで切って描くだけでスクロールを持たず、フッタ（`y 投稿 / e 編集に戻る / Esc 中止`）は常に出す（回答は短い。長い下書きは `e` でエディタに戻れば全文が読める）。
`e` で戻り先の画面に戻してからエディタを開くのは、エディタ終了後の編集完了メッセージが「`a` を押した画面」で処理される形に揃えるため。確認画面のまま開くと、編集完了の扱いを 2 通り持つことになる。

### 投稿はコマンド、結果はフッタ

投稿は `action.Comment` を実行して `postedMsg{label string; err error}` を返すコマンドで行い、`Model` は `gh` を直接呼ばない（s08 の取得と同じ形）。投稿中は `posting` を立てて `a` を無視する。同じ対象への二重投稿を防ぐためであり、対象ごとの排他は持たない。
成功 / 失敗はフッタの右側（s08 のステータスの場所）に出す。s08 の `errText` は取得の失敗用で、次の取得開始で消える。回答のステータスは別のフィールドに持ち、赤（失敗）か通常（成功・中止）で描く。表示の優先は取得中のスピナー > 回答のステータス > 取得の失敗。消えるタイミングは次の取得開始か次の `a`。
`ctx` は `context.WithTimeout(context.Background(), 30*time.Second)`。s07 の `CallTimeout` と同じ値で、`gh` が固まっても TUI が投稿中のまま止まらない。

### 投稿後に再取得しない

D-002 の「書き込み後は対象 1 件だけ再取得する」は s18 の担当。ここで全件再取得を発行すると search 2 回が走り、投稿のたびに 30 req/分の枠を使う。何もしなければ、回答済みの PR / issue は次の取得（s12 `R`、s13 自動更新）で進行中に移る。それまで今やるタブに残ることは許容する。

### 対象の決め方

キュー画面は s08 の `Subject`（`Card.Result` を出した Issue または PR）。局面 A は PR、局面 B は issue が主体になるので、`a` はそのまま正しい書き先になる。カード詳細は Issue、PR 詳細はその PR。主体が PR のカードで issue にコメントしたいときは、カード詳細を開いて `a` を押せばよい。`question` の有無を問わないのは、docs に禁じる記述が無く、局面 E の issue や「その他」の PR にコメントしたい場面（担当への一言など）を塞ぐ理由が無いため。ラベルは触らないので、`question` 無しの対象にコメントしても dispatcher の状態機械には影響しない。

## Risks / Trade-offs

- [Bubble Tea v2 の外部プロセス実行の API 名が想定と違う] → spec は振る舞いで書いてある。実装時に `charm.land/bubbletea/v2` の `exec.go` を読む（v2.0.9 に `*exec.Cmd` とコールバックを受ける関数がある）
- [外部プロセス実行のコマンドをテストで実行できない] → 引数の組み立てとコールバックを分けて単体テストし、起動そのものは tasks の手動確認（テンプレートが入ったエディタが開き、空にして閉じると投稿されない）で見る
- [live での投稿検証が無い] → validation-plan.md が未定としている。`Fake.Calls` の検証と、s03 の `Client` の引数検証（`pr comment 131 -R org/app --body-file -`）の組み合わせで、投稿される引数は確認できる。実リポジトリへの投稿は利用者が判断する（未決事項）
- [`s10` が `New` を変え、s13 がさらに変える] → archive の順を s08 → s09 → s10 → s13 にし、s13 が s10 の MODIFIED を写す
- [`a` の判定が `question` 無しでも通るので、誤って無関係の issue にコメントできる] → エディタを空にして閉じれば投稿されない。確認ダイアログを足す案は、局面 A / B の回答のたびに 1 手増える。mvp.md は `m` にだけ「確認ダイアログ必須」と書いている
- [人が `blocked-by:` 行を引用して `y` で投稿すると dispatcher の判定が変わる] → 不変条件 8 は警告までを求めている。警告文に理由（dispatcher が正本にする）を書く
- [マーカーを本文のどこでも拒否するので、マーカーについて言及する回答を書けない] → `<!-- routine -->` を文中に書く回答は運用上まず無い。必要なら人がブラウザで書く
- [投稿中に `q` で終了すると投稿ゴルーチンが途中で終わる] → `gh` サブプロセスは 30 秒のタイムアウトで終わる。s08 の取得と同じ扱い
- [確認画面にスクロールが無い] → 回答は数行。長い下書きは `e` で戻れば読める
- [エディタ起動中に自動更新（s13）や取得完了が来る] → メッセージはキューに溜まり、エディタ終了後に処理される。`fetchedMsg` は s08 の規則どおり `Cards` を更新するだけで、回答の対象は変えない

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| `question` の無い対象で `a` を押したとき | コメントを許可する。書き先は対象の種類（PR / issue）で決まる | mvp.md の `a` は「回答・コメント」。禁じる記述が無く、ラベルを触らないので状態機械に影響しない |
| 詳細画面での `a` の対象 | カード詳細は `Card.Issue`、PR 詳細はその PR | 画面が表示している対象にコメントするのが自然。主体が PR のカードで issue に書きたいときはカード詳細から |
| テンプレートの元にするコメント | 対象の `Comments` のうち最新の `AI` true のコメント。`Comments` が nil なら空 | mvp.md「質問コメント」は routine が書く。後ろに人のコメントがあっても routine の最新を採る |
| 推奨が無い質問の既定値 | 先頭の選択肢の `Letter`。選択肢が無ければ `Q<n>: ` で空 | mvp.md は推奨を既定値とするだけ。空欄より候補が入っている方が編集が少ない |
| テンプレートの末尾改行 | 付けない（`Q1: A\nQ2: B`） | mvp.md の表記どおり。エディタが保存時に付けても投稿に支障は無い |
| 投稿する本文 | エディタの内容をそのまま（前後の空白を除かない） | 空の判定だけ `TrimSpace`。本文を書き換えない |
| `editor` が空 | 起動せず `editor が設定されていません（config の editor か環境変数 EDITOR）` を赤で出す | s02 が s10 に委ねた。`vi` 等への暗黙の fallback は docs に無い |
| `editor` の分割 | 空白で分割（`strings.Fields`）。先頭が実行ファイル、残りが引数、末尾に一時ファイルのパス。シェルを通さない | `code --wait` 形式を扱う。設定値をシェルに評価させない |
| 一時ファイル | `os.CreateTemp("", "sugi-loop-answer-*.md")`。終了後（成否とも）に削除 | Markdown として編集できる拡張子。残骸を残さない |
| エディタの非 0 終了 | `エディタ: <err>` を赤で出し、投稿しない | 保存せず終了した意図と読む |
| 空の本文 | `回答を中止しました（本文が空）` を出し、投稿しない。`action.Comment` の `ErrEmptyBody` と同じ判定 | 中身の無い回答で `question` が外れるのを防ぐ |
| マーカーの判定範囲 | `<!-- routine -->` / `&lt;!-- routine --&gt;` が本文のどこかにあれば拒否。大文字小文字・空白の揺れは吸収しない | 不変条件 7「書かない」だけ |
| `## PR リスク評価` 見出しを含む本文 | 拒否しない | 不変条件 7 はマーカーだけを定める。s05 `IsAI` はこの見出しを AI 扱いするので、投稿後に TUI では AI コメントとして折りたたまれる。既知の差 |
| `blocked-by:` の検出規則 | 各行を `strings.TrimSpace` して `blocked-by:` で始まる行。大文字小文字を区別。`>` は吸収しない | s05 `LatestBlockedBy`（`internal/model/parse.go`）と同じ。dispatcher の判定とずらさない |
| 確認画面のキー | `y` 投稿 / `e` 編集に戻る / `Esc` 中止 / `q` 終了 | mvp.md に無い画面。`Esc` は s09 の「戻る」と揃える |
| 確認画面の構成 | 全画面。`回答の確認: <表示名>` / 理由と該当行 / 区切り線 / 下書き（残りの高さで切る）/ フッタ（常に出す）。スクロール無し | 警告の該当行と下書き全体を読める。回答は短い |
| 表示名 | `<Repo> PR#<n>` / `<Repo> #<n>` | s09 の詳細ヘッダと同じ表記 |
| 投稿の `ctx` | 30 秒のタイムアウト | s07 の `CallTimeout` と同じ値 |
| 投稿中の `a` | 無視する | 二重投稿を防ぐ。対象ごとの排他は持たない |
| 投稿後の再取得 | しない | 1 件再取得は s18。全件再取得は search の枠を使う |
| 回答ステータスの場所と寿命 | フッタの右側。次の取得開始か次の `a` で消える。優先はスピナー > 回答ステータス > 取得エラー | s08 のステータスの場所を共有し、行を増やさない |
| 成功 / 失敗 / 中止の文言 | `<表示名> にコメントしました` / `<表示名> へのコメントに失敗: <err>`（赤）/ `回答を中止しました` | テストで文字列を見る |
| `fetchedMsg` が確認画面中に届いたとき | `Cards` と時刻は更新し、下書きと対象は変えない | s09 の「詳細の対象を差し替えない」と同じ |
| エディタ実行中のキー入力 | 受けない（端末はエディタが使う） | Bubble Tea の外部プロセス実行の仕組みに従う |
| live での投稿検証 | tasks に入れない。手動確認は「エディタが開く → 空にして閉じる → 投稿されない」まで | validation-plan.md「書き込み検証は未定」。検証用リポジトリの用意は利用者の判断 |
