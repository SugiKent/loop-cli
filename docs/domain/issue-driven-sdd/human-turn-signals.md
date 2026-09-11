# issue-driven-sdd における人の出番（判定ルール）

最終更新: 2026-09-11-1030

loop-cli の分類器（`internal/classify`）と action 層はこの文書を仕様の正本とする。
この文書は 2 つの運用方式を扱う。既定は issue-driven-sdd（`stage:*`）で、issue-label-driven（`To Do` / `In Progress` / `Done`）の
差分は「issue-label-driven のシグナル対応表」の節にまとめる。
関連文書として [MVP 定義](../../mvp/mvp.md) と [技術的意思決定](../../mvp/decisions.md) を参照する。

## 人の出番と検知シグナル

プラグインの規約（`routine-common` / `references/worker.md` / `routine-dispatch` / `routine-sweep`。同期点は上流 `2b1b791`）から導いた、
人が介入すべき局面とその **GitHub 上で観測できるシグナル**。この表がキューの分類器の仕様そのもの。

上流が定める人の役割は 3 つだけ。**`stage:todo` を付ける / `question` が付いた PR・issue にコメントで答える / PR を merge する。**
`blocked` / `question` / 段階ラベルの付け外しと worker の再起動は routine（dispatcher / sweep）が行い、人はラベルを触らない。
`ai-assess:requested` を付け直すと PR の AI 評価をもう一度走らせられる。これは `L`（ラベル一覧）から付け直せる
（`L` はリポジトリのラベルを制限なく扱う。#3 の回答で決まった）。

| # | 人の役割 | 検知シグナル（すべて GitHub の状態だけで決まる） | TUI のアクション | 優先度 |
| --- | --- | --- | --- | --- |
| A | **質問に答える**（grill） | open PR、`question` ラベル、最新コメントが routine のもの | PR 会話コメントで `Q1: A` 形式の返信。作った worker が同じセッションで受け取る | 1 |
| B | **方針を決める** | open issue、`question` ラベル（`blocked` に重ねて付く。`blocked-by: human` の印）、最新コメントが routine のもの | issue に会話コメントで答える。**ラベルは触らない**。次の sweep が `question` / `blocked` を外して worker を起動し直す | 2 |
| C | **merge する** | open PR、`propose` / `apply` / `archive` ラベル、本文 1 行目が `未確定の判断: 0 件`、`question` なし、`ai-assess:requested` なし（付いたまま 3 時間動いていない PR は付いていても当たる）、checks 緑、mergeable | `gh pr merge`（確認ダイアログ必須） | 3 |
| D | **レビュー質問に答える** | `apply` PR、未 resolve の review thread、thread 最終コメントが routine のもの | thread への返信（REST replies エンドポイント） | 1 |
| E | **着手を承認する** | open issue、`stage:*` ラベルなし、`blocked` なし | `stage:todo` を付与 | 4（バックログ） |
| F | **壊れた状態を直す** | 段階ラベルが 2 つ以上 | 表示して警告、ブラウザで開く。TUI から自動修復はしない | 0（最上位・件数は少ない） |
| G | **段階に関係しない PR を merge する** | open PR、`docs` ラベル、`question` なし | `gh pr merge`（merge しても段階は進まない） | 5 |

分類はこの順で評価し、最初に当たった行で止める。どの行にも当たらない open PR（ラベルなし・コメントなしの PR、旧構成の `retro` PR など）は
「その他」バケットに入れて必ず画面に出す。「進行中」に混ぜると人の出番かどうかを判別できなくなるため。消えて見えなくなる項目を作らない。

ただし「その他」になった open PR のうち、最終更新（`updatedAt`）から `other_grace_min` 分（既定 30）未満のものは
`[3]進行中` に出し、猶予を超えたら今までどおり `[1]今やる` に出す。PR のその他の大半は routine の状態機械の途中
（PR を作った直後でラベルが無い、checks が pending、`未確定の判断: N 件` が N > 0 のまま grill 中、`ai-assess:requested` が付く前）で、
人が判断できないためである。誰も触らないまま猶予を超えたものだけが人の出番として浮上する。`other_grace_min: 0` で猶予なし。
issue の「その他」（`question` と `blocked` があるのにコメントが取れなかったもの）は猶予の対象にしない。これは B（人待ち）の
取りこぼしなので、人にすぐ出す。時間の判断を持つのは `Card()` だけで、判定表の行とフォールバックの意味（どの行にも当たらない）は変えない。

F を「段階ラベル 2 つ以上」だけに絞った理由: `restart: 3/3` に達した issue と、`question` 付きのまま merge された propose PR は、
どちらも dispatcher が issue に `blocked-by: human` + `question` を書くので、B として自然に浮上する。人が merge せずに close した
propose / apply PR も同じく dispatcher が `blocked-by: human` にするので B に出る。`question` 単独（`blocked` 無し）の issue は
dispatcher が回収する残骸なので、異常扱いせず進行中に置く。

**キューに入れないもの（進行中タブに出す）**:

- `stage:propose` / `stage:apply` / `stage:archive` が付き `wip` が付いている issue。AI が動いている最中。
- `question` も `ai-assess:requested` も付いていない open PR で最新コメントが人のもの。auto-fix が受け取り中。
  `ai-assess:requested` は worker が人の回答を反映し終えてから付けるので、付いていれば下の「AI 評価待ち」で扱う。
- `question` の issue で最新コメントが人のもの。回答済みで、sweep が `question` を外して worker を起動し直すのを待っている。
  PR の回答は同一セッションが即座に拾うが、issue の回答は sweep 間隔でしか拾われない（Routines に issue コメントのトリガーが無い）。
- 段階ラベルも `blocked` も無く、最新の routine コメントが `release:` / `restart:` / `advance:` の issue。上流はブロック解除と
  死んだ worker の再起動を「`[]` を書いてから `[stage:X]` を書く」2 回書きで行い、1 回目のあとで死ぬと段階ラベルの無い issue が残る。
  sweep がこのコメントを目印に続きを書くので、E（着手を承認する）に出して人に `stage:todo` を付けさせると段階が巻き戻る。
- `ai-assess:requested` が付いた open PR。未確定 0 件になった PR の AI リスク評価が走っている最中で、評価を終えた assess がラベルを外す。
  評価前に merge させないため、外れるまでは C に出さない。`question` も付いている壊れた状態では、人の質問（A）を先に出す。
- 上の PR の規則（最新コメントが人・AI 評価待ち）は、PR が最後に動いてから（`updatedAt`）3 時間以内に限る。どちらも AI が次に動く前提で
  人の目から外す規則なので、Routine が止まるとその PR は永久に見えなくなる。3 時間は sweep が「最新コメントが人のまま routine の返信が無い PR」を
  止まったとみなす時間に揃えた。時間切れのうち、最新コメントが人の PR は「その他」に「人のコメントに AI が応答していない」と出す
  （判定表に流すと、反映されていない依頼が C に見えるため）。AI 評価待ちの PR は判定表で分類し、C に出す。

### issue-label-driven のシグナル対応表

OpenSpec を挟まない **issue-label-driven**（以下 ILD）のリポジトリは、`To Do` / `In Progress` / `Done` の 3 ラベルだけで進む。
局面 A–G の意味は同じで、違うのはラベルの語彙と、下の 7 点の規則だけである。方式はリポジトリのラベル一覧が正本で、
取得のたびに判定する（`stage:todo` があれば sdd、無くて `To Do` があれば ILD。D-005）。
ILD のラベル遷移は全部 1 回書きで、`In Progress` が `To Do` を置き換える。
PR にラベルは付かず、issue との紐づけは `Closes #n` だけである（issue 1 件 = PR 1 本）。ただし分類はその紐づけで絞り込まない。

| # | 検知シグナル（ILD） | sdd との違い |
| --- | --- | --- |
| A | open PR、`question` ラベル | 最新コメントが人でも A のまま。ILD は PR の「最新コメントが人 → 進行中」を適用しない |
| B | open issue、`question`（`blocked` に重ねて付く）、最新コメントが routine のもの | 同じ |
| C | open PR、`question` なし、checks 緑、mergeable | 段階ラベル・本文 1 行目 `未確定の判断: 0 件`・`Closes #n` を見ない。`question` だけが merge 禁止の印 |
| D | open PR、未 resolve の review thread、thread 最終コメントが routine のもの | 対象の絞り込み（sdd は `apply` ラベル）が無くなり、**行 C より先に評価する** |
| E | open issue、`To Do` / `In Progress` / `Done` なし、`blocked` なし | 段階ラベルの語彙だけが違う |
| F | `To Do` / `In Progress` / `Done` が 2 つ以上 | PR には段階ラベルが無いので、F になるのは issue だけ |
| G | （無し） | `docs` ラベルが無い方式なので評価しない |

行 D を行 C より先に評価するのは、ILD の C から本文 1 行目のゲートが外れるためである。C を先に見ると、未 resolve の
レビュー質問がある緑の PR が常に「merge する」になり、D が checks の赤い PR にしか出なくなる。優先度も D が 1、C が 3 でこの順に一致する。

行 C と行 D から `Closes #n` の絞り込みを外したのは、ILD の open PR を全件キューに出すためである。`Closes #n` は
「routine が作った PR の印」として使っていたが、それで絞ると外部から来た PR や `Closes` を書き忘れた PR が人から見えなくなる。
結果として ILD の open PR は「`question` あり → A」「未 resolve の AI thread あり → D」「`question` 無しで mergeable かつ
checks 緑 → C」「それ以外 → その他」の 4 つに分かれ、どれも `[1]今やる` に並ぶ（`[3]進行中` には 1 件も入らない）。

**ILD でキューに入れないもの（進行中タブに出す）**:

- `In Progress` の 1 件だけが付き、`question` が付いていない issue。AI が動いている最中。
  ILD には `wip` が無く `In Progress` が段階と作業中の印を兼ねるので、`question` を除外しないと `In Progress` + `blocked` + `question` の
  issue が常に進行中へ落ち、行 B（方針を決める）が恒久的に空になる。
- 段階ラベルも `blocked` も無く、最新の routine コメントが `restart:` の issue。ILD の 2 回書きの残骸は `restart:` だけで、
  `release:` / `advance:` を書く経路が無い。
- `ai-assess:requested` の規則は適用しない（ILD にこのラベルは無い）。
- **PR の「最新コメントが人」の 2 規則は適用しない。** sdd では auto-fix や worker が同じ PR を受け取って続きを進めるので進行中に
  落とす意味があるが、ILD は 1 issue = 1 PR を人が捌く方式で、キューから外れると人の出番が見えなくなる。
- 進行中の残り 2 規則（issue の「最新コメントが人」、`question` 単独の issue）は sdd と同じである。

**方式を判定できないリポジトリ**: `stage:todo` も `To Do` も持たないリポジトリ（と、ラベル一覧の取得に失敗したリポジトリ、
初回取得が終わる前のスナップショット表示）は方式が分からない。分類と表示はゼロ値の `sdd` の語彙で行い、`t` は書き込まずに
フッタへ理由を赤で出す。推測したラベルを書くと、書いたラベルで worker が起動しないか、方式の違う worker を起動するためである。

### 稼働中プロジェクトの実データで確認した事象（設計判断の根拠）

issue-driven-sdd を運用しているリポジトリを 1 つ読んで確認した。

- `question` 付き（未確定 N > 0）のまま人が propose PR を merge した例があった。dispatcher は段階を進めず、issue に `blocked-by: human` を書いた。
  → 局面 C の merge ガードは実際に人のミスを防ぐ。TUI は `question` 付き、または 1 行目の N > 0 の PR の merge を拒否する。
- grill の問いは PR 会話コメント 1 本に束ねられ、人は `Q1: A` `Q2: A` と会話コメントで返している。
  → 局面 A の返信先は会話コメント（`gh pr comment`）でよい。review thread ではない。
- routine コメントのマーカーが `&lt;!-- routine --&gt;` と HTML エスケープされて投稿された例が複数ある。また旧構成の `assess-pr-risk` の
  「PR リスク評価」コメントはマーカーなしで投稿されている（`assess-pr-risk` は上流 `e83def7e` でプラグインから外れ、
  プロジェクト側で作る場合は `<!-- routine -->` 始まりが必須になった。既存コメントは残るので判定は維持する）。
  → 「誰の番か」の判定は本文マーカーで行うが、**エスケープ済みマーカーと `## PR リスク評価` 見出しも AI 発として扱う**。
  routine は利用者本人のアカウントで投稿するので、author による判定は不可能。これがこのツールの中核不変条件。
- 同一イベントから複数セッションが起動し、同趣旨のコメントが 3 連続で付くことがある。
  → 詳細画面ではコメントを畳み、`blocked-by:` 行を含む**最新**コメントだけを正本として上部に要約表示する。
- archive PR が `未確定の判断: 0 件` で人の merge 待ちになっていた（`mergeable` は `UNKNOWN`）。
  → merge 可否は表示時に `gh pr view --json mergeable,mergeStateStatus,statusCheckRollup` で取り直す。search 結果には載らない。
- 上流の規約変更（`d8db3842`）で、人の回答後に proposal を直す propose PR がもう 1 本作られるようになった。同じ issue に
  merge 済みの `propose` PR が複数並び、古い方には `question` が残る。dispatcher は最新の merge だけを見る。
  → カードでは同段階の PR を全部並べつつ、最新 merge を正本として印を付ける。古い PR の `question` を異常扱いしない。

---

## 人が書き込むときの不変条件

すべて `gh` 経由。プラグインの規約に従う制約を action 層の不変条件にする。

1. **ラベル集合の置換は行わない。** `--add-label` / `--remove-label` だけを使い、渡さなかったラベルには触らない
   （他の書き手が付けたラベルを消さない）。`t` は 1 ラベルずつ別プロセスで呼び、`L` の送信は 1 回の `gh` 実行で複数ラベルを変える。
   Routine のフィルターは書き込み後のラベル集合で判定され、増えたラベルの数だけイベントが出る（上流 `2b1b791` の実測）。
   まとめて送るほうが意図しない Routine は起きにくいが、`gh` は「付ける」と「外す」を別の mutation として送るので**原子的ではなく**、
   片方だけ通ることがある。また付けるほうが先に走るため、`stage:apply` を足して `wip` を外す送信は書き込み後の集合に `wip` が残った
   状態で評価され、worker の Routine の `NOT_IN [wip]` に当たって**意図した起動も起きない**ことがある。
2. **`t` の `stage:todo`、`s` の `stage:propose` に加えて、`L` で人が明示的に選んだラベルは書く。** `L` はリポジトリのラベルを
   制限なく扱うので、`blocked` / `question` / `wip` / `stage:apply` / `stage:archive` も人の操作で書ける（#3 の回答）。
   誤って段階ラベルを 2 つ付けた issue では、以後 `t` が `ErrMultipleStages` で拒否され続けるので、戻すのも `L` から行う。
3. **`s` の「外して付け直す」は 2 呼び出しで行う。** `stage:todo` を外し、`gh issue view --json labels` で読み直してから `stage:propose` を付ける。
4. **回答の書き先は 3 種類を混同しない。** grill の問い（PR 会話コメント）→ `gh pr comment`。issue の `question` → `gh issue comment`。
   review thread → REST replies。
5. **merge ガード。** `question` あり、`ai-assess:requested` あり（AI 評価が未完了）、1 行目 `未確定の判断: N 件` で N > 0、checks 失敗、`isDraft` の
   いずれかなら merge を拒否し理由を表示する。
6. **Issue 作成は承認ではない。** `n` で作った issue に段階ラベルは付けない。着手させるなら続けて `t`。
7. **TUI は `<!-- routine -->` を書かない。** TUI から投稿する文章はすべて人の発言。dispatcher は「`<!-- routine -->` で始まらない
   コメント」を人の回答とみなして `question` を外すので、これを書くと回答が拾われない。
8. **人のコメントに `blocked-by:` で始まる行を含めない。** dispatcher は著者に関係なく「`blocked-by:` 行を含む最新コメント」を
   正本にするため、人が引用しただけで判定が変わる。回答エディタは投稿前にこの行を検出して警告する。

---

## 変更履歴

| 日時 | 変更内容 | 理由 |
| --- | --- | --- |
| 2026-09-12-0100 | 「その他」の open PR に最終更新からの時間猶予（`other_grace_min`、既定 30 分）を足し、猶予内は進行中タブに出すことにした。issue の「その他」は対象外 | PR のその他の大半が routine の処理中で人が判断できず、ゾンビ化した PR だけを人に出すため（s30-other-grace・#43） |
| 2026-09-12-0030 | PR の進行中の規則（最新コメントが人・AI 評価待ち）を `updatedAt` から 3 時間以内に限り、時間切れの扱いと、`ai-assess:requested` 付き PR を「最新コメントが人」の規則から外すことを足した。行 C の `ai-assess:requested` なしの条件に時間切れの例外を足した | ca-ai-role-play で assess Routine が止まり `ai-assess:requested` が外れなくなり、open PR が全件進行中に隠れたため（s32-stale-in-progress-pr） |
| 2026-09-10-0900 | 冒頭の `ai-assess:requested` の一文と不変条件 1・2 を `L`（ラベル一覧）に合わせて改訂。不変条件 1 を「ラベル集合の置換は行わない」に狭め、まとめ送信が原子的でないことと Routine の起動順の帰結を追記 | `L` が任意のラベルを 1 回の書き込みでまとめて変えるようになり、「1 ラベルずつ」「書くラベルは 2 つに限る」が事実と合わなくなったため（#3・s28-label-picker） |
| 2026-09-11-1030 | ILD の方式の正本をラベル一覧に変え、行 C / 行 D から `Closes #n` を外し、PR の「最新コメントが人」の 2 規則を ILD で適用しないことにした | ILD の open PR を全件「今やる」に出し、方式の書き忘れという状態を無くすため（#24） |
| 2026-09-10-0700 | issue-label-driven（`mode: label`）のシグナル対応表と、その方式でキューに入れないものを追加した | `To Do` / `In Progress` / `Done` の 3 ラベルで進むリポジトリを同じキューに載せるため（#5） |
| 2026-09-07-1500 | 上流 `2b1b791` に同期。同期点を更新し、C の条件に `ai-assess:requested` なしを足し、キューに入れないものへ「2 回書きの途中の issue」と「AI 評価待ちの PR」を足し、不変条件 1 の理由をラベル集合による起動判定に書き直し、不変条件 5 に `ai-assess:requested` を足した | 上流が Routine の起動条件をラベル集合の判定に合わせて作り直し、ブロック解除と再起動を 2 回書きにし、PR の AI 評価を `ai-assess:requested` ラベルで起動する経路に変えたため |
| 2026-09-05-1805 | 上流 `d8db3842` に同期。B を「`question` 付き issue にコメントするだけ」に変更、F を段階ラベル重複のみに縮小、G から `retro` を除外、進行中に「回答済み issue（sweep 待ち）」を追加、不変条件 2〜4 を人がラベルを触らない規約に合わせて書き直し | 上流が人の操作を `stage:todo` とコメントに限定し、issue の人待ちを `question` で可視化するようになったため |
| 2026-09-05-1407 | `docs/mvp/design.md` の第 1 節と「書き込み」節の不変条件をドメイン文書として分離 | 判定ルールはプラグイン規約に属するドメイン知識であり、MVP 定義とは寿命が異なるため |
