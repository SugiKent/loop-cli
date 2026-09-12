# issue-driven-sdd における人の出番

最終更新: 2026-09-12-2130

loop-cli の分類器（`internal/classify`）と action 層は、この文書を仕様の正本とする。
既定は issue-driven-sdd（SDD、`stage:*`）、もう一方は issue-label-driven（ILD、`To Do` / `In Progress` / `Done`）。
方式はリポジトリのラベル一覧から毎回判定する（`stage:todo` があれば SDD、無くて `To Do` があれば ILD）。
関連文書: [MVP 定義](../../mvp/mvp.md)、[技術的意思決定](../../mvp/decisions.md)。

## 人の役割

上流 `2b1b791` の `routine-common` / `references/worker.md` / `routine-dispatch` / `routine-sweep` に基づく。
人が行うのは次の 3 つだけ。

- issue に着手ラベルを付ける
- `question` が付いた issue / PR、未解決の review thread に答える
- PR を merge する

`blocked` / `question` / 段階ラベルの遷移と worker の再起動は routine が行う。
SDD では `ai-assess:requested` を付け直すと AI 評価を再実行できる。任意ラベルの変更は `L` を使う。

## 分類

分類は SDD では表の順に評価し、最初に一致した局面を採用する。ILD だけは D を C より先に評価し、G を使わない。

| 局面 | 人の役割 | SDD の条件 | ILD の条件 | TUI のアクション |
| --- | --- | --- | --- | --- |
| A | 質問に答える | open PR、`question`、最新コメントが routine | open PR、`question`。最新コメントの著者は問わない | PR 会話へ `Q1: A` 形式で返信 |
| B | 方針を決める | open issue、`blocked` と `question`、最新コメントが routine | 同左 | issue にコメントする。ラベルは触らない |
| C | merge する | open PR、`propose` / `apply` / `archive`、本文先頭が `未確定の判断: 0 件`、`question` なし、`ai-assess:requested` なし（付いたまま 3 時間動かない場合を除く）、checks 成功、mergeable | open PR、`question` なし、checks 成功、mergeable | 確認後に `gh pr merge` |
| D | レビュー質問に答える | `apply` PR、未解決 thread、thread の最新コメントが routine | open PR、未解決 thread、thread の最新コメントが routine | REST replies endpoint で thread に返信 |
| E | 着手を承認する | open issue、`stage:*` なし、`blocked` なし | open issue、段階ラベルなし、`blocked` なし | SDD は `stage:todo`、ILD は `To Do` を付ける |
| F | 壊れた状態を直す | `stage:*` が 2 つ以上 | issue の `To Do` / `In Progress` / `Done` が 2 つ以上 | 警告してブラウザで開く。自動修復しない |
| G | 段階外の PR を merge する | open PR、`docs`、`question` なし | 該当なし | 確認後に `gh pr merge` |

優先度は F=0、A/D=1、B=2、C=3、E=4、G=5。ILD は本文先頭や `Closes #n` で PR を絞らないため、D を C より先に評価する。

どの局面にも一致しない open PR は「その他」として `[1]今やる` に出す。ただし SDD のみ、`updatedAt` から
`other_grace_min` 分（既定 30）未満の PR は routine の処理途中とみなし `[3]進行中` に出す。`other_grace_min: 0` で猶予なし。
issue の「その他」と ILD の PR には猶予を適用しない。猶予は `Card()` の表示分類だけに作用し、上の局面判定を変えない。

## `[3]進行中` に出す状態

### SDD

- `stage:propose` / `stage:apply` / `stage:archive` と `wip` が付いた issue
- open PR で、`question` も `ai-assess:requested` もなく、最新コメントが人、かつ更新から 3 時間以内
- `ai-assess:requested` が付いた open PR。ただし `question` 付きは A、更新から 3 時間を超えたものは C の判定へ進める
- `question` の issue で、最新コメントが人（sweep 待ち）
- 段階ラベルも `blocked` もなく、最新の routine コメントが `release:` / `restart:` / `advance:` の issue（2 回書きの途中）
- 「その他」の open PR で `other_grace_min` の猶予内

3 時間を超えて最新コメントが人のままの PR は、「人のコメントに AI が応答していない」として「その他」に出す。
`ai-assess:requested` の時間切れは C の判定へ進める。

### ILD

ILD の段階遷移は 1 回書きで、`In Progress` が `To Do` を置き換える。PR には段階ラベルを付けない。

- `In Progress` だけが付き、`question` がない issue
- `question` の issue で、最新コメントが人（sweep 待ち）
- 段階ラベルも `blocked` もなく、最新の routine コメントが `restart:` の issue

ILD では `ai-assess:requested`、PR の「最新コメントが人」、`other_grace_min` を使わない。open PR はすべて A / D / C / その他のいずれかで `[1]今やる` に出す。

### 方式を判定できない場合

`stage:todo` も `To Do` もない、ラベル取得に失敗した、または初回取得前なら、読み取り表示は SDD の語彙を使う。
書き込みは行わず、`t` のフッタに理由を赤で出す。

## 判定に必要な例外

| 例外 | 扱い |
| --- | --- |
| `question` 単独の issue | dispatcher が回収する残骸なので異常にせず進行中 |
| `restart: 3/3`、`question` のまま merge、または人が PR を close | dispatcher が issue に `blocked-by: human` + `question` を書くため B として扱う |
| routine marker が `&lt;!-- routine --&gt;` | `<!-- routine -->` と同じく AI 発として扱う |
| marker のない旧 `## PR リスク評価` コメント | AI 発として扱う |
| 同趣旨の routine コメントが連続 | `blocked-by:` 行を含む最新コメントを正本にし、詳細画面では畳む |
| `mergeable` が `UNKNOWN` になり得る | 表示時に `gh pr view --json mergeable,mergeStateStatus,statusCheckRollup` で取り直す |
| 同じ issue に同段階の merge 済み PR が複数 | すべて表示し、最新 merge を正本として印を付ける。古い PR の `question` は異常にしない |

routine は利用者本人のアカウントで投稿するため、AI / 人の判定を author に依存させない。

## 書き込みの不変条件

書き込みはすべて `gh` 経由で行う。

1. **ラベル集合を置換しない。** `--add-label` / `--remove-label` だけを使う。`t` は 1 ラベルずつ、`L` は選択した複数ラベルを 1 回の `gh` 実行で変更する。
   `L` も原子的ではなく追加が先なので、`stage:apply` を追加しながら `wip` を外すと、Routine の `NOT_IN [wip]` が発火しない場合がある。
2. **書けるラベルを暗黙に制限しない。** `t` / `s` の段階ラベルに加え、`L` では人が選んだ任意ラベルを書く。段階ラベル重複の修正も `L` で行う。
3. **`s` の段階変更は 2 回に分ける。** `stage:todo` を外し、`gh issue view --json labels` で読み直してから `stage:propose` を付ける。
4. **返信先を分ける。** grill は PR 会話、issue の `question` は issue 会話、review thread は REST replies。grill の複数質問は 1 コメントにまとめる。
5. **merge をガードする。** `question`、有効な `ai-assess:requested`、`未確定の判断: N 件`（N > 0）、checks 失敗、`isDraft` のどれかがあれば拒否して理由を表示する。
6. **issue 作成を承認とみなさない。** `n` で作った issue に段階ラベルを付けない。着手は `t` で行う。
7. **TUI は `<!-- routine -->` を書かない。** TUI の投稿は人の発言であり、この marker を含む本文は投稿を拒否する。
8. **人のコメントに `blocked-by:` 行を書かない。** dispatcher が著者を問わず最新の該当行を正本にするため、投稿前に警告する。

## 変更履歴

| 日時 | 変更内容 | 理由 |
| --- | --- | --- |
| 2026-09-12-2130 | SDD / ILD の重複説明を表へ統合し、背景説明を例外と不変条件へ圧縮 | Opus 5 向けに、一般的な説明を減らしコードから推測できない固有コンテキストへ集中するため |
| 2026-09-12-0100 | 「その他」の open PR に `other_grace_min`（既定 30 分）を追加 | 処理中の PR を隠し、ゾンビ化した PR だけを人に出すため（s30-other-grace・#43） |
| 2026-09-12-0030 | PR の進行中判定を 3 時間に制限し、時間切れ処理を追加 | assess Routine 停止時に PR が進行中へ隠れ続けるのを防ぐため（s32-stale-in-progress-pr） |
| 2026-09-10-0900 | `L` に合わせてラベル変更の不変条件を改訂 | 複数ラベル変更が非原子的で Routine の起動に影響するため（#3・s28-label-picker） |
| 2026-09-11-1030 | 方式の正本をラベル一覧にし、ILD の全 open PR を表示 | 方式の書き忘れをなくすため（#24） |
| 2026-09-10-0700 | ILD のシグナルを追加 | 3 ラベル方式を同じキューに載せるため（#5） |
| 2026-09-07-1500 | 上流 `2b1b791` に同期 | 起動条件、2 回書き、AI 評価経路の変更へ追随するため |
| 2026-09-05-1805 | 上流 `d8db3842` に同期 | 人の操作を `stage:todo` と回答へ限定した変更に追随するため |
| 2026-09-05-1407 | MVP 文書から判定ルールを分離 | プラグイン規約に属するドメイン知識として保守するため |
