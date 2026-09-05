## Context

s03 の `GHClient` に `ViewIssue` / `AddLabel` / `RemoveLabel` があり、`Client` は `gh issue view <n> -R <repo> --json number,title,body,url,labels,comments` / `gh issue edit <n> -R <repo> --add-label <label>` / `--remove-label <label>` をラベル 1 つで発行する。`Fake` は書き込みと `ViewIssue` を `Calls` に記録する（不変条件 3 の順序検証のために `ViewIssue` だけ読み取りでも記録する設計）。s05 の `internal/model` に `LabelStageTodo` / `LabelBlocked` / `IssueStages` / `HasLabel` がある。s10 が `internal/action`（書き込みの直前だけを持つ薄い層）と `ui.New(fetcher, client, editor)` の `client`、フッタ右のステータス、投稿中フラグを導入している。
human-turn-signals.md の局面 E（open issue、`stage:*` 無し、`blocked` 無し → `stage:todo` を付与。優先度 4、バックログ）と不変条件 1（1 操作 1 ラベル）・2（TUI が書くラベルは `stage:todo` と `s` の `stage:propose` に限る）、mvp.md のキー表 `t`（`stage:todo` を付ける / 外す。付け直しは dispatcher への「もう一度評価しろ」の合図。内部処理 `gh issue edit --add-label` / `--remove-label`）が正本である。s05 の分類では `stage:todo` だけの issue は進行中タブ（`#n は進行中`）に入る。D-002「書き込み後は対象 1 件だけ再取得する」は s18 の担当。validation-plan.md は書き込みの検証手段を未定としている。

## Goals / Non-Goals

**Goals:**
- バックログの issue に `t` 1 回で `stage:todo` を付け、もう一度 `t` で外せる（主要フロー E）
- 不変条件 1 / 2 を `internal/action` の Requirement として持ち、`Fake.Calls` で「`ViewIssue` → `AddLabel` / `RemoveLabel` 1 回・`stage:todo` だけ」を検証できる
- 付けるか外すかの判定を GitHub の現在のラベルで行い、画面の古いラベルに依存しない

**Non-Goals:**
- `s`（`stage:todo` を外して `stage:propose` を付ける 2 呼び出し）は s15 が担当する。`ToggleTodo` はそれを含まない
- この change は書き込み後の再取得を行わず（s18 が担当する）、画面の Card のラベルも書き換えない
- この change は局面 F（段階ラベル 2 つ以上）を修復しない。human-turn-signals.md「TUI から自動修復はしない」
- この change は live でのラベル付与を検証しない（validation-plan.md が検証手段を未定としている）

## Decisions

### ファイル構成

```
internal/action/todo.go                 # ToggleTodo / ErrOtherStage / ErrMultipleStages / ErrBlocked
internal/action/todo_test.go            # testdata/todo の fixture と Fake.Calls で判定と呼び出しを検証
internal/action/testdata/todo/issue-<n>.json  # ラベルの組み合わせ別の gh issue view 出力（手書き）
internal/ui/todo.go                     # t のキー処理、対象の決定、toggleCmd と toggledMsg、ステータス文言
internal/ui/todo_test.go                # t → コマンド → 結果 の各 Scenario
internal/ui/model.go                    # t の振り分け、書き込み中フラグの共有（変更）
internal/ui/view.go                     # キュー画面とカード詳細のフッタに t todo（変更）
```

### 判定の前にラベルを読み直す

`ToggleTodo` は `ViewIssue` で現在のラベルを読んでから `AddLabel` か `RemoveLabel` を 1 回呼ぶ。画面の `Issue.Labels` を判定に使う案は、`t` を押した後に再取得が無い（s18 まで）ので、取り消しのつもりの 2 回目の `t` が同じ古いラベルを見て「もう一度付ける」になり、mvp.md の「付ける / 外す」が成り立たない。画面の Card のラベルを成功時に書き換える案（楽観的更新）は、`Model` が読み取り専用として扱っている `Cards` と詳細のコピーの 2 か所を書き換える必要があり、`Card.Result` は再分類されないので見た目とタブがずれる。読み直しは `gh` 呼び出しが 1 回増えるだけで、判定が常に GitHub の状態に一致し、`Model` は何も書き換えない。`Fake` が `ViewIssue` を記録する設計（s03）もこの順序の検証に合う。
`ViewIssue` はコメントも取るため `labels` だけの軽い呼び出しではないが、`GHClient` に labels 専用メソッドを足す（interface / `Client` / `Fake` の 3 か所の拡張）ほどの価値は無い。

### 拒否の条件と順序

docs は `t` を「付ける / 外す」としか書いていない。`stage:todo` 以外の段階ラベルが付いた issue に付けると段階ラベルが 2 つになり、局面 F（壊れた状態）を TUI が作ることになる。mvp.md `s` の「段階ラベルは同時に 1 つ」に従い拒否する。段階ラベルが既に 2 つ以上ある issue は、`stage:todo` を含んでいても外さない（人が `t` で 1 つ外せば直るが、局面 F は「表示して警告、ブラウザで開く。TUI から自動修復はしない」であり、外す操作は修復に当たる）。`blocked` は局面 E の条件（`blocked` 無し）から外れ、付け外しは dispatcher が行うものなので、`stage:todo` を付けて dispatcher を起動させる操作を拒否する。
順序は「段階 2 つ以上 → `stage:todo` あり → 別の段階 → `blocked` → 付ける」。`stage:todo` と `blocked` が同時に付いている issue は 2 番目で外す側に当たる（付いているものを外すのは常に許す。取り消しをできなくしない）。

### 対象は「画面の Card の Issue」の 1 規則

キュー画面は選択行の `Card.Issue`、カード詳細は詳細の対象の `Card.Issue`。主体が PR の Card（局面 A / C 等）でも対象は Issue で、`ToggleTodo` が段階ラベルを見て拒否するので誤って PR の issue を動かすことは無い。PR 単独の Card（`Issue` nil）は何もしない。PR 詳細でも何もしない（PR から Issue を引く経路を 1 画面のためだけに作らない）。`Subject` は使わない（`t` の対象は常に Issue で、主体の種類に依存しない）。

### 確認なし、結果はフッタ

mvp.md は `m` にだけ「確認ダイアログ必須」と書き、`t` には無い。`t` は取り消しが `t` 1 回で済む可逆な操作であり、局面 E の issue を次々に承認する体験（先頭から捌く）に 1 手足さない。切り替えはコマンドで実行し、`toggledMsg{label string; added bool; err error}` を返す。ステータスは s10 の回答のステータスと同じ場所・同じ寿命（次の取得開始か次の `t` / `a` で消える）。s10 のフィールド名（`answerStatus` / `posting`）は実装者が汎用の名前に変えてよい。spec は文言と振る舞いだけを定める。
書き込み中フラグは s10 の投稿中フラグと 1 つで共有する。ラベル切り替え中の `a` とコメント投稿中の `t` を両方止めることになり、s10 の「投稿中は `a` を無視」を狭めない。この広がりは s10 `answer-question` の 2 Requirement（`a` の投稿中無視、ステータスの消滅条件）の本文を変えるので、この change が MODIFIED で写す。archive の順は s08 → s09 → s10 → s11（s11 は s10 の ADDED を前提にする）。

### 投稿後に再取得しない

s10 と同じ。全件再取得は search 2 回を使う。`ToggleTodo` が毎回読み直すので、画面が古くても取り消しは正しく動く。反映は `R`（s12）/ 自動更新（s13）/ 1 件再取得（s18）。

## Risks / Trade-offs

- [`t` のたびに `gh issue view` が 1 回増える] → 1 操作で `gh` を 2 回起動する。この呼び出し量は REST の通常枠 5,000 req/時を圧迫しない。search の枠は使わない
- [画面と GitHub の状態がずれた瞬間の `t`] → 読み直しで判定するので、dispatcher が既に `stage:propose` に進めていれば拒否になり、F を作らない。拒否の文言に付いているラベルを出す
- [`stage:todo` を付けた後も再取得までバックログに残る] → s10 の回答と同じ扱い。ステータスに結果が出る。`R` で進行中に移る
- [`blocked` + `stage:todo` の issue で `t` を押すと外れる] → 外す操作は常に許す方針。dispatcher が `blocked` を付けた issue の `stage:todo` を人が外す意味は「取り下げ」に近く、mvp.md の未決事項（取り下げ操作）と重なるが、mvp.md の `t` は「外す」を含んでいる
- [live 検証が無い] → `Fake.Calls` と s03 の `Client` の引数検証（`issue edit 108 -R org/app --add-label stage:todo`）の組み合わせで発行する引数は確認できる。実リポジトリへの付与は利用者が判断する（未決事項）
- [切り替え中に `q` で終了] → `gh` サブプロセスは 30 秒で終わる。s10 と同じ

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| 判定に使うラベル | `ViewIssue` で読み直した現在のラベル。画面の `Issue.Labels` は使わない | 再取得が無い間の 2 回目の `t` を取り消しとして成立させる。楽観的更新は `Model` の 2 か所を書き換え、`Result` は再分類されない |
| 画面の Card のラベルの楽観的更新 | しない。`R` 等で反映 | 上と同じ。`ToggleTodo` が読み直すので画面が古くても動く |
| 別の段階ラベルが付いた issue で `t` | 拒否して理由（ラベル名）を赤で出す | mvp.md `s`「段階ラベルは同時に 1 つ」。付けると局面 F を TUI が作る |
| 段階ラベルが 2 つ以上の issue で `t` | `stage:todo` を含んでいても拒否 | 局面 F「TUI から自動修復はしない」 |
| `blocked` が付いた issue で `t`（`stage:todo` 無し） | 拒否 | 局面 E の条件は `blocked` 無し。`blocked` は dispatcher が管理する |
| `blocked` と `stage:todo` が両方付いた issue で `t` | 外す | 付いている `stage:todo` を外す操作は常に許す |
| `question` だけが付いた issue（`blocked` 無し・段階ラベル無し）で `t` | 拒否しない（付ける） | 局面 E の条件に当たる。docs は沈黙 |
| 確認ダイアログ | 出さない。取り消しはもう一度 `t` | mvp.md は `m` にだけ確認必須。可逆な 1 ラベル操作 |
| PR 単独の Card（`Issue` nil）で `t` | 何もしない | 対象が無い。代替案はステータスに `issue に紐づかない PR には stage:todo を付けられません` を出すこと |
| PR 詳細画面で `t` | 何もしない。ヒントも出さない | PR から Issue を引く経路を 1 画面のためだけに作らない。カード詳細に戻れば `t` が効く |
| 主体が PR の Card で `t` | Card の `Issue` を対象にする | 対象は常に Issue。段階ラベル付きなら `ToggleTodo` が拒否する |
| `ctx` | 30 秒のタイムアウト | s10 の投稿・s07 の `CallTimeout` と同じ |
| 書き込み中フラグ | s10 の投稿中フラグと共有し、`t` と `a` を両方止める | 二重書き込みを防ぐ。対象ごとの排他は持たない |
| ステータスの場所と寿命 | s10 の回答のステータスと共有。次の取得開始か次の `t` / `a` で消える | 行を増やさない |
| 文言 | `<Repo> #<n> の stage:todo を切り替え中` / `に stage:todo を付けました` / `から stage:todo を外しました` / `の stage:todo を切り替えられません: <err>`（赤） | テストで文字列を見る。拒否と `gh` の失敗は同じ形で `<err>` が理由を持つ |
| `ViewIssue` の失敗 | 書き込まず、切り替えられません + エラー文字列 | 現在の状態が分からないまま書かない |
| 拒否のエラー値 | `ErrOtherStage` / `ErrMultipleStages` / `ErrBlocked` の 3 つを `errors.Is` で判別でき、ラベル名を `fmt.Errorf("%w ...")` で包む | UI は `err.Error()` を出すだけで足りる |
| fixture | `internal/action/testdata/todo/issue-<n>.json` を手書き（150: `stage:todo`+`bug`、151: `stage:propose`+`question`、152: `blocked`、153: 無し、154: `bug`+`enhancement`、155: `stage:todo`+`stage:propose`）。`IssueDetail` にデコードできる最小の JSON | `example` には 140（無し）と 108（`stage:propose`）しか無い。s07 の `internal/fetch/testdata/` と同じ置き方 |
| live での検証 | tasks に入れない。手動確認は「`t` を押すとステータスが出る」まで。実リポジトリへの付与は利用者が判断する | validation-plan.md「書き込み検証は未定」 |
