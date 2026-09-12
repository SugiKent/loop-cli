# design: s30-other-grace

Refs #43

## Context

分類は `internal/classify` の純粋関数 3 つで行う。`Issue()` / `PR()` が human-turn-signals.md の判定表を要素 1 件に当て、`Card()` が Issue と open PR 群の結果から最上位の局面を `Card.Result` に置く。`Card()` は `in-progress` を候補から外すので、要素が `in-progress` になればカードはそれに引っ張られない。

分類は `fetch.Fetch` が 1 回の取得の最後に呼ぶ。s32 以降、`Fetch(ctx, client, repos, now)` と `Card(c, mode, now)` と `PR(pr, mode, now)` は取得時刻 `now` を引数で受け取り、`PR()` が進行中の規則 2 / 3 / 7 の時間切れ（3 時間）に使う。`Fetch` も `Card()` も壁時計は読まず、`cmd/loop-cli` の `Fetcher` の閉包が呼ぶたびに `time.Now()` を渡す。運用方式は s30-mode-from-labels 以降 `Fetch` がラベル一覧から判定し、`Result.Modes` に入れる。

「その他」（`other`）は判定表のフォールバックで、open PR がどの行にも当たらないとき、および issue に `question` と `blocked` があるのに B に当たらないときに出る。優先度 6・今やるタブ・種別「その他」で、色は付かない。

`config.yml` は `internal/config` が読む。`repos` 以外は省略でき、既定値は `Load` が埋める。未知のキーはエラー。onboarding の `Marshal` は 5 キーを固定順で書き、`refresh_interval_sec` は聞かずに `120` を書く。`docs/mvp` は凍結されており（CLAUDE.md）、設定ファイルの例は更新しない。利用者向けの説明は README が持つ。

`label` 方式では、s30-mode-from-labels が「open PR は全件今やるに出し、進行中には 1 件も入れない」と決めている（human-turn-classify「方式が label の入力は…open PR を全件今やるに出す」）。

## Goals / Non-Goals

**Goals:**

- 一過性の `other`（routine が数分で次の状態へ進める open PR）を今やるタブから外す
- 一定時間誰も触っていない `other` は今までどおり今やるタブに出し、現れた瞬間にデスクトップ通知が飛ぶ
- `Issue()` / `PR()` と fixture の期待値表を変えない
- 猶予の長さを設定で変えられ、`0` で今までの挙動に戻せる

**Non-Goals:**

- 「その他になった時刻」をローカルに記録すること（proposal「経過時間の起点」）
- `label` 方式の PR に猶予を当てること（全件今やるに出す決定を覆さない）
- `docs/mvp` の更新（凍結）
- `other` に落ちる理由（checks 待ち・未確定 N 件・ラベル無し）を要約に書き分けること。判断材料として有用だが、この change の目的（一過性のその他を人の目から外す）とは別の改善で、`Issue()` / `PR()` の変更を伴う
- 取得と取得の間に、時間経過だけで行をタブ間で動かすこと（分類は取得のたびに行う。下記 Risks）
- onboarding のフォームで `other_grace_min` を聞くこと

## Decisions

### D1. 猶予は `Card()` が要素の結果に対して適用し、`Issue()` / `PR()` は変えない

`Card(c model.Card, mode model.Mode, now time.Time, grace time.Duration) model.Card` にする（`now` は s32 で既にある。`grace` を足す）。`Card()` は今までどおり `Issue()` / `PR()` で各要素の `Result` を埋めたあと、`mode` が `sdd` のとき、`State` が `OPEN` の PR ごとに次を当てる。

- `Result.Situation` が `other`、かつ `grace > 0`、かつ PR の `UpdatedAt` が取得できていて（ゼロ値以外）、かつ `now.Sub(UpdatedAt) < grace` なら、その PR の `Result` を `in-progress` に置き換える。要約は `PR #<n> はどの局面にも当たらない（更新から <M>m は様子見）`。`<M>` は `grace` を分に切り捨てた整数
- それ以外の PR と、`Issue` は触らない。`mode` が `label` なら何も当てない

issue の `other` を対象にしない理由: issue が `other` になるのは `question` と `blocked` があるのに `Comments` が nil または空のときだけで、`Comments` が nil になるのは詳細取得に失敗したときだけ（human-turn-classify「入力の前提」）。これは取得の欠損であり、本来は B（人待ち・優先度 2）の取りこぼしである。routine の途中という本 change の動機に当たらない。既存の spec も「進行中に落とすと人待ちの issue が見えなくなる」と MUST の根拠に書いている。隠す根拠が無い。

置き換えは `Card.Result` を決める前に行うので、候補の選び方（`in-progress` を候補から外す。候補が無ければ先頭の open PR の要約）はそのまま使える。要素の `Result` も置き換えるので、`Card.Result` と要素の `Result` の一致（s08 `subjectOf` が主体を引く手がかり）も保たれる。

判定表の関数を変えない理由は 2 つある。`Issue()` / `PR()` は「GitHub の状態だけで決まる」判定表そのもので、時間を入れると fixture の期待値表（V-1）が取得時刻に依存する。もう 1 つは、人が `other` の意味（どの行にも当たらない）を読むとき、時間の条件が混ざっていない方が追いやすい。

- 代替案: `PR()` に `grace` も渡し、フォールバックの手前で猶予を見る。`PR()` は s32 で `now` を持つので技術的には可能だが、`PR()` の `other` は「どの行にも当たらない」という判定表の結論であり、そこに表示の都合を混ぜると fixture の期待値表（`PR()` を直接呼ぶ）が猶予にも依存する。`Card()` に置けば `PR()` の結論は変わらない
- 代替案: `internal/ui` の `buildRows` でタブを振り分け直す。`Card.Result.Tab` と実際のタブが食い違い、`addedNow`（`Card.Result.Tab` を見る）と表示がずれる。分類の結果はすべて `Result` に閉じるという s05 / s07 の形を崩す
- 代替案: 新しい `Situation`（例 `settling`）を足す。`Priority()` / `Tab()` / `Kind()` と s08 の色表・優先記号の表に 1 行ずつ足すことになるが、表示は `in-progress` と同じ（進行中タブ・色なし・記号なし）なので、区別する値を持つ理由が無い。要約で区別できる

カードの要約への影響: `fallbackSummary` は候補が無いとき先頭の open PR の要約を採るので、「`wip` の issue + 猶予中の PR」のカードは進行中タブで `PR #<n> はどの局面にも当たらない（更新から 30m は様子見）` が見出しになる。この change の前は同じカードが `other` として今やるタブに出ていたので、見出しの変化は今やるから外す判断の帰結である。要約が PR 番号と理由を含むので、カード詳細で issue の状態は読める。この帰結は Scenario で固定する。

### D2. 猶予は `Fetch` の引数で受け取り、`now` と一緒に `classify.Card` へ渡す

`Fetch(ctx context.Context, client gh.GHClient, repos []string, now time.Time, grace time.Duration) (*Result, error)` にし、`classify.Card` の呼び出しへそのまま渡す。`now` は s32 が既に引数にしているので、`grace` を隣に足すだけである。`Fetch` は壁時計を読まない（s32 と同じ）。

`grace` は `time.Duration` で受け取る。`internal/fetch` / `internal/classify` は `internal/config` を import しない（既存の方針）ので、分から `Duration` への変換は `cmd/loop-cli` が行う（`RefreshInterval` と同じ）。

- 代替案: `Fetch` の引数を struct にまとめる。呼び出し側と spec の書き換えが増える。引数 1 つの追加で足りる
- 代替案: `now` と `grace` を 1 つの `cutoff time.Time`（`UpdatedAt.After(cutoff)` なら猶予内）にまとめる。判定は 1 式になるが、猶予なし（`grace` 0）を表す `cutoff` が無い（ゼロ値は「すべて猶予内」、`now` は「未来の `UpdatedAt` だけ猶予内」になる）ので、結局 `grace` 0 の分岐が要る。要約の `<M>` も出せなくなる

### D3. 猶予の起点は要素の `UpdatedAt`

`Issue.UpdatedAt` / `PR.UpdatedAt` は `gh search` の `updatedAt` で、コメント・ラベル変更・本文編集・push のいずれでも更新される。「誰も触っていない時間」を取るのに追加の取得は要らない。

`UpdatedAt` がゼロ値（取得できなかった）なら猶予を当てず `other` のまま出す。いつ触られたか分からないものは隠さない。`now` が `UpdatedAt` より前（ローカル時計が GitHub より遅れている）なら差は負で `< grace` を満たし、猶予内として扱う。この場合 `UpdatedAt` は取得できていて「いま触られた」と読むのが自然であり、ずれの数秒のために作成直後の PR を今やるタブに出す方が害が大きい。

既定 30 分の根拠: routine の状態遷移（PR 作成 → ラベル付与 → checks → `ai-assess:requested` → 評価完了）が通常この時間に収まることを前提にしている。checks が 30 分を超えるリポジトリでは、CI が走っているだけの PR が猶予明けに今やるタブに出て通知が飛ぶ。その場合は `other_grace_min` を CI の所要時間より長くする（未決事項）。

### D4. `other_grace_min` は分の整数、既定 30、`0` で無効、負数はエラー

`refresh_interval_sec` と同じ書き方（単位を名前に含める整数）にする。`0` を「猶予なし」に使うのは、今までの挙動へ戻す手段を用意するためで、`refresh_interval_sec` が `1` 以上を要求するのと違い `0` を受け入れる。負数は意味が無いのでエラーにし、エラー文字列に `other_grace_min` を含める。上限は設けない（`time.Duration` が溢れるのは 1.5e11 分を超えたときで、起こり得ない入力への防御は書かない）。

設定にする理由: 猶予の適切な長さはリポジトリの CI の所要時間で変わる（D3）。定数にすると CI が長いリポジトリで調整できない。

- 代替案: `internal/classify` の定数にする。`Config` / 検証 / README / mvp.md の変更が消えて change は小さくなるが、上記の調整手段が無くなる。利用者がこの change の入口で「設定で変えられること」を選んでいる

onboarding の `Marshal` は書かない。`Load` が既定を埋めるので、書き出した YAML を `Load` に通す往復の性質は変わらない。onboarding spec の「mvp.md の例と同じ形式」は「5 キーの並びと書き方」を指すと明示し、`other_grace_min` を書かないことを spec に書く。

### D5. 通知は既存の差分比較に任せる

`addedNow` は `Card.Result.Tab` が今やるの Card を主体のキーで比べる。猶予内は進行中タブにいるので比較に入らず、猶予を超えた取得で今やるタブに現れると「増えた」になり、要約 `PR #<n> はどの局面にも当たらない` で通知が飛ぶ。これがゾンビの知らせになる。この帰結を `desktop-notify` の Scenario で固定する。

### D6. docs の更新

- `human-turn-signals.md`「その他」バケットの段落に、猶予の規則（sdd の PR で `UpdatedAt` から `other_grace_min` 分未満は進行中に出す。以上なら今やるに出す。`0` で無効。label には当てない）を足し、変更履歴に 1 行足す。この文書は分類器の正本なので、コードより先に直す
- `README.md` の設定表と例に `other_grace_min` を足す
- `docs/mvp` は凍結されているので触らない（CLAUDE.md）。起点を `updatedAt` にする判断の記録は proposal と human-turn-signals.md が持つ

## Risks / Trade-offs

- **今やるタブへ移るのは取得のたび** → 猶予を超えても次の取得（既定 120 秒周期、または `R`）までは進行中タブに残る。ずれは最大で `refresh_interval_sec` 秒。30 分の猶予に対して無視できる
- **`UpdatedAt` は routine や bot の書き込みでも延びる** → 自動 rebase される Dependabot PR、bot がコメントを付け続ける PR は、base が動くたびに猶予が延び、今やるタブに出ないことがある。進行中タブには出ているので消えはしないが、「誰も触っていない」の判定に人と bot の区別は無い。区別するには `CreatedAt` の上限や著者の判定が要り、この change では入れない。実運用で bot の PR が埋もれるなら別 change で上限を足す
- **checks が猶予より長いリポジトリでは CI 中の PR が今やるタブに出る** → `other_grace_min` を CI の所要時間より長くする（D3）
- **猶予内に取得が失敗し続けると進行中に残る** → 取得失敗時は前回の `Cards` を維持する（D-002）ので、分類も更新されない。既存の挙動で、この change が新たに作る問題ではない
- **snapshot からの起動は前回の分類のまま** → 起動直後は前回取得時の `now` で分類した結果が出る。初回取得で直る。stale 表示は D-002 が認めている

## 未決事項

docs/mvp と docs/domain が沈黙している点。実装者が選ぶ既定値を 1 つずつ示す。

- **猶予中の要約**: 既定値は `PR #<n> はどの局面にも当たらない（更新から <M>m は様子見）`。`<M>` は `int(grace.Minutes())`
- **既定 30 分**: 既定値は 30。前提は「routine の状態遷移と checks が 30 分に収まる」で、収まらないリポジトリでは利用者が長くする（D3）
- **`UpdatedAt` がゼロ値のとき**: 既定値は猶予を当てず `other` のまま（D3）
- **`now` が `UpdatedAt` より前のとき**: 既定値は猶予内（D3）
- **`other_grace_min` の型と境界**: 既定値は整数の分、`0` 以上、既定 30（D4）
- **`Card()` / `Fetch` の引数の形**: 既定値は既存の `now time.Time` の隣に位置引数 `grace time.Duration` を足す（D1 / D2）

## Open Questions

なし。
