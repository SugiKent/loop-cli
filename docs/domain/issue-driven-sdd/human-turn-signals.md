# issue-driven-sdd における人の出番（判定ルール）

最終更新: 2026-09-07-1500

loop-cli の分類器（`internal/classify`）と action 層はこの文書を仕様の正本とする。
関連文書として [MVP 定義](../../mvp/mvp.md) と [技術的意思決定](../../mvp/decisions.md) を参照する。

## 人の出番と検知シグナル

プラグインの規約（`routine-common` / `references/worker.md` / `routine-dispatch` / `routine-sweep`。同期点は上流 `2b1b791`）から導いた、
人が介入すべき局面とその **GitHub 上で観測できるシグナル**。この表がキューの分類器の仕様そのもの。

上流が定める人の役割は 3 つだけ。**`stage:todo` を付ける / `question` が付いた PR・issue にコメントで答える / PR を merge する。**
`blocked` / `question` / 段階ラベルの付け外しと worker の再起動は routine（dispatcher / sweep）が行い、人はラベルを触らない。
`ai-assess:requested` を付け直すと PR の AI 評価をもう一度走らせられるが、この TUI からは行わない（人の出番を並べるのが役目）。

| # | 人の役割 | 検知シグナル（すべて GitHub の状態だけで決まる） | TUI のアクション | 優先度 |
| --- | --- | --- | --- | --- |
| A | **質問に答える**（grill） | open PR、`question` ラベル、最新コメントが routine のもの | PR 会話コメントで `Q1: A` 形式の返信。作った worker が同じセッションで受け取る | 1 |
| B | **方針を決める** | open issue、`question` ラベル（`blocked` に重ねて付く。`blocked-by: human` の印）、最新コメントが routine のもの | issue に会話コメントで答える。**ラベルは触らない**。次の sweep が `question` / `blocked` を外して worker を起動し直す | 2 |
| C | **merge する** | open PR、`propose` / `apply` / `archive` ラベル、本文 1 行目が `未確定の判断: 0 件`、`question` なし、`ai-assess:requested` なし、checks 緑、mergeable | `gh pr merge`（確認ダイアログ必須） | 3 |
| D | **レビュー質問に答える** | `apply` PR、未 resolve の review thread、thread 最終コメントが routine のもの | thread への返信（REST replies エンドポイント） | 1 |
| E | **着手を承認する** | open issue、`stage:*` ラベルなし、`blocked` なし | `stage:todo` を付与 | 4（バックログ） |
| F | **壊れた状態を直す** | 段階ラベルが 2 つ以上 | 表示して警告、ブラウザで開く。TUI から自動修復はしない | 0（最上位・件数は少ない） |
| G | **段階に関係しない PR を merge する** | open PR、`docs` ラベル、`question` なし | `gh pr merge`（merge しても段階は進まない） | 5 |

分類はこの順で評価し、最初に当たった行で止める。どの行にも当たらない open PR（ラベルなし・コメントなしの PR、旧構成の `retro` PR など）は
「その他」バケットに入れて必ず画面に出す。「進行中」に混ぜると人の出番かどうかを判別できなくなるため。消えて見えなくなる項目を作らない。

F を「段階ラベル 2 つ以上」だけに絞った理由: `restart: 3/3` に達した issue と、`question` 付きのまま merge された propose PR は、
どちらも dispatcher が issue に `blocked-by: human` + `question` を書くので、B として自然に浮上する。人が merge せずに close した
propose / apply PR も同じく dispatcher が `blocked-by: human` にするので B に出る。`question` 単独（`blocked` 無し）の issue は
dispatcher が回収する残骸なので、異常扱いせず進行中に置く。

**キューに入れないもの（進行中タブに出す）**:

- `stage:propose` / `stage:apply` / `stage:archive` が付き `wip` が付いている issue。AI が動いている最中。
- `question` の付いていない open PR で最新コメントが人のもの。auto-fix が受け取り中。
- `question` の issue で最新コメントが人のもの。回答済みで、sweep が `question` を外して worker を起動し直すのを待っている。
  PR の回答は同一セッションが即座に拾うが、issue の回答は sweep 間隔でしか拾われない（Routines に issue コメントのトリガーが無い）。
- 段階ラベルも `blocked` も無く、最新の routine コメントが `release:` / `restart:` / `advance:` の issue。上流はブロック解除と
  死んだ worker の再起動を「`[]` を書いてから `[stage:X]` を書く」2 回書きで行い、1 回目のあとで死ぬと段階ラベルの無い issue が残る。
  sweep がこのコメントを目印に続きを書くので、E（着手を承認する）に出して人に `stage:todo` を付けさせると段階が巻き戻る。
- `ai-assess:requested` が付いた open PR。未確定 0 件になった PR の AI リスク評価が走っている最中で、評価を終えた assess がラベルを外す。
  評価前に merge させないため、外れるまでは C に出さない。`question` も付いている壊れた状態では、人の質問（A）を先に出す。

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

1. **ラベルは 1 つずつ付け外しする。** `--add-label` / `--remove-label` を 1 ラベルずつ別プロセスで呼び、ラベル集合の置換は行わない。
   Routine のフィルターは書き込み後のラベル集合で判定され、増えたラベルの数だけイベントが出る（上流 `2b1b791` の実測）。
   1 回に増やすラベルを 1 つに保てば、意図しない Routine を起こさずに済む。
2. **TUI が書くラベルは `stage:todo` と、`s` の強制操作で付ける `stage:propose` の 2 つに限る。** `blocked` / `question` / `wip` /
   `stage:apply` / `stage:archive` は書かない。人がラベルを触らない前提で dispatcher が状態機械を回しているため。
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
| 2026-09-07-1500 | 上流 `2b1b791` に同期。同期点を更新し、C の条件に `ai-assess:requested` なしを足し、キューに入れないものへ「2 回書きの途中の issue」と「AI 評価待ちの PR」を足し、不変条件 1 の理由をラベル集合による起動判定に書き直し、不変条件 5 に `ai-assess:requested` を足した | 上流が Routine の起動条件をラベル集合の判定に合わせて作り直し、ブロック解除と再起動を 2 回書きにし、PR の AI 評価を `ai-assess:requested` ラベルで起動する経路に変えたため |
| 2026-09-05-1805 | 上流 `d8db3842` に同期。B を「`question` 付き issue にコメントするだけ」に変更、F を段階ラベル重複のみに縮小、G から `retro` を除外、進行中に「回答済み issue（sweep 待ち）」を追加、不変条件 2〜4 を人がラベルを触らない規約に合わせて書き直し | 上流が人の操作を `stage:todo` とコメントに限定し、issue の人待ちを `question` で可視化するようになったため |
| 2026-09-05-1407 | `docs/mvp/design.md` の第 1 節と「書き込み」節の不変条件をドメイン文書として分離 | 判定ルールはプラグイン規約に属するドメイン知識であり、MVP 定義とは寿命が異なるため |
