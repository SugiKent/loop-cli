## Context

分類器の語彙は `internal/model/model.go` の定数と `IssueStages` / `PRStages` に集約されているが、**規則そのもの**も sdd の状態機械に結びついている（`classify.go:33` の「段階 1 件 + `wip`」、`classify.go:110` の `isC` が段階ラベルを要求する、`classify.go:125` の `isD` が `apply` を要求する）。issue-label-driven（以下 ILD）は別のラベル名を使い、`wip` を持たず、PR に段階ラベルを付けない。この 3 点のうち後ろ 2 つは規則の差なので、語彙の差し替えだけでは足りない。

制約は 3 つある。

- 既存の sdd リポジトリの分類結果を 1 件も変えられない（#5 成功条件 6）
- 分類は純粋関数である（`human-turn-classify`「分類は純粋関数で…」）。方式を知るために I/O を足せない
- ILD の plugin（`sugiken-dev-plugins-public` の `issue-label-driven`）は未 push で、規約は #5 の本文に写された範囲しか読めない。読めない部分は proposal.md「未確定の判断」で人に問う

背景と目的は proposal.md「Why」を参照。

## Goals / Non-Goals

**Goals:**
- 2 方式のリポジトリを 1 本のキューに混ぜても、タブ・優先度・キー操作が方式によらず同じに見える
- 方式の分岐点をコードの少数の箇所に閉じ込め、どちらの方式でも同じ関数を通す
- 回帰を「既存テストを 1 行も書き換えずに通す」ことで示す

**Non-Goals:**
- ラベル名を設定で変えられるようにすること（mvp.md の固定方針を維持する）
- リポジトリのラベル集合から方式を自動判定すること
- ILD 向けに新しいキー操作や画面を足すこと。既存の 4 タブとキーをそのまま使う
- 実在の ILD リポジトリでの実測。fixture と単体テストで閉じる

## Decisions

### 方式は `model.Mode` の値として issue / PR が持ち、分類器のシグネチャを変えない

`classify.Issue(is model.Issue)` / `classify.PR(pr model.PR)` を保ち、`model.Issue` / `model.PR` に `Mode` フィールドを足す。方式を引数で足す案（`classify.Issue(is, mode)`）と比べて、`Card` 経由の呼び出し（`classify.Card` は Card の中の issue と PR を回す）で方式を配り直さずに済み、fixture テストと `loop-cli-dev classify` の呼び出しも 1 か所の代入で済む。

`Mode` のゼロ値 `""` は `sdd` として扱う。既存のテストが書く `model.Issue{Labels: …}` のリテラルは `Mode` を持たないので、ゼロ値が sdd であることが回帰の安全弁になる。

代替案として `Vocab` 構造体（ラベル名の表）を渡す設計も検討した。この案を採らない理由は、方式の差が規則にも及ぶ（`wip` の有無、PR 段階ラベルの有無）ため、表を持ってもなお分岐が残り、表と分岐を二重に管理することになるからである。

### 語彙は `model` の関数が方式で切り替え、規則の分岐は `classify` に置く

`IssueStages(mode, labels)` / `PRStages(mode, labels)` / `TodoLabel(mode)` / `IsMidRelabel(mode, comments)` が語彙を持ち、呼び出し側は方式ごとの定数を直接参照しない。規則の差（進行中の規則 1、行 C / D の対象、行 G と規則 7 の有無）は `classify.Issue` / `classify.PR` の中の分岐で書く。

方式ごとに分類器を 2 本作る案は採らない。局面 A / B / E / F と進行中の規則 2〜5 は完全に共通で、2 本にすると共通部分を二重に持つことになる（CLAUDE.md「パターン競合は平均化しない」の対象は競合するパターンであり、ここは同じ 1 本の規則に方式の分岐が数点あるだけである）。

### `LinkedIssue` を `internal/fetch` から `internal/model` へ移す

ILD の PR には段階ラベルが無く、「routine が作った PR か」を `Closes #n` の有無で見分ける（proposal.md Q1 / Q3）。この判定を分類器が使うが、`internal/fetch` は `internal/classify` を import しているので、逆向きの import はできない。紐づけの規則を 2 か所に書くのは避けたいので、規則の正本を `internal/model` に移し、`fetch` と `classify` の両方がそこを呼ぶ。`internal/fetch` は同名の関数を公開しない（薄い委譲関数を残さない）。

### リポジトリごとの方式は `Fetch` に表で渡す

`Fetch(ctx, client, repos, modes)` の `modes` は `map[string]model.Mode`。`cmd/loop-cli/main.go` は既に `cfg.Repos` から `map[string]string`（merge 方式）を組み立てて画面へ渡しており、同じ形をもう 1 つ作るだけで済む。`internal/fetch` が `internal/config` を import しない現在の境界も保てる。

### 画面の出し分けは 5 か所に閉じる

画面が方式を見るのは次の 5 か所である。`detail.go` の `cardHeaderLines` が段階行を組み立てるところ、同じ関数がバッジを並べるところ、`prListLines` が PR 一覧の見出しを出すところ、`help.go` が `t` の説明を書くところ、`view.go` がキュー 0 件のヒントを出すところ。行の色・優先記号・タブの割り当ては `Situation` が決めるので方式に依存しない。ヘルプは画面全体の一覧なので、選択中のカードの方式で出し分けず両方のラベル名を並べる。

### ILD の fixture は手書きで置く

`gh.Capture` は実在のリポジトリを要求するが、ILD で運用中のリポジトリがまだ無い。fixture は `gh` の出力形式の JSON なので手で書ける。`internal/gh/fixtures_test.go` が `login` を `user-N` 形式に限っているので、手書きの JSON もその形式に従う。`internal/classify/fixture_test.go` の期待値表は alias ごとに方式と期待 `Situation` を持つ形にする。

## Risks / Trade-offs

- **ILD の plugin が未 push で、規約の一部を推測している** → 推測が要る 3 点（行 C の対象、本文 1 行目の規約、行 D の対象）を proposal.md「未確定の判断」に出し、PR コメントで確定してから実装する。確定するまで実装を始めない
- **手書き fixture が実データとずれる** → 実リポジトリでの確認は別 issue に切り出す。この change の完了条件は「規約として書かれた振る舞いを分類器が再現すること」に閉じる
- **`To Do` にはスペースが含まれる** → `gh issue edit --add-label "To Do"` はサブプロセスの引数として渡るのでシェルのクォートは要らないが、`Fake` の記録と突き合わせるテストで実際の引数を確認する
- **ILD リポジトリの `mode` を書き忘れると、全 issue が局面 E に落ちてバックログが溢れる** → キュー 0 件のヒントと mvp.md の設定ファイル節に `mode: label` を明記する。誤設定そのものを検出する仕組みは持たない（ラベル一覧の取得が増え、自動判定を避けた理由と衝突する）
- **`IssueStages` / `PRStages` / `IsMidRelabel` の引数が 1 つ増える** → 呼び出し箇所は 8 か所で、すべてこの change の中で書き換わる。外部に公開する API ではない

## Migration Plan

設定ファイルは後方互換で、`mode` を書かない既存の `config.yml` は `sdd` として動く。データの移行も再取得も要らない。

## Open Questions

無し。判断が要る 3 点は proposal.md「未確定の判断」に置き、PR コメントで人に問う（この 3 点は spec と tasks を変えるので、design の Open Questions には置かない）。
