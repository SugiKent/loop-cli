# design: 2026-09-10-s26-wrap-titles

Refs #2

## Context

`internal/ui/detail.go` の `cardHeaderLines` と `prHeaderLines` は、ヘッダ行を `[]string` で返す。
`internal/ui/view.go` の `renderDetail` はその各行を `ansi.Truncate(l, m.width, "…")` に通してから区切り線・本文・フッタと連結する。
本文領域の高さは `detailHeader` が `max(m.height-2-len(header), 1)` で決め、1 行を割るときは PR 一覧を末尾から落とす。

つまりヘッダは既に行のスライスであり、1 つの論理行が 1 つの表示行に対応するという前提を持たない。
折り返しは 1 行目をスライスの複数要素に展開するだけで済む。

制約は 2 つある。

- 表示幅は `github.com/charmbracelet/x/ansi` の `StringWidth` で測る。プロジェクトの既存コードはすべてこれを使っており、`len` や `utf8.RuneCountInString` は使わない
- 折り返しでヘッダが伸びると本文領域が痩せる。`detailHeader` は本文領域の高さをヘッダの行数から引いて出すので、折り返しで増えた行もそのまま引かれる。ただし固定ヘッダ自体が端末高を超える経路が、折り返しによって新たに現実的になる

## Goals / Non-Goals

**Goals:**

- 詳細画面のヘッダ 1 行目でタイトルを全文読めるようにする
- 折り返した行をタイトルの開始位置に縦揃えし、どこからがタイトルの続きかを目で追えるようにする
- 全角文字・絵文字を含むタイトルで桁がずれないようにする
- 折り返しても画面全体の行数が端末の高さを超えないようにする

**Non-Goals:**

- キュー画面の表とプレビューの変更（proposal の Q1 の回答を待つ）
- 本文（Glamour が折り返す）とフッタ・PR 一覧・`Summary` の折り返し
- 詳細ヘッダのスクロール（ヘッダ領域は固定のまま）

## Decisions

### D1. 折り返しは `ansi.Wrap` ではなく `ansi.Hardwrap` を使う

タイトルは 1 つの文字列だが、日本語には単語境界の空白が無い。`ansi.Wrap`（単語境界で折る）は、空白の無い日本語タイトルでは 1 語とみなして幅を超えるか、意図しない位置で切る。
`ansi.Hardwrap(s, width, true)` は表示幅ちょうどで折り、`preserveSpace` を false にすれば行頭の空白を落とす。
英語混じりのタイトルで単語が途中で切れるが、切り詰めて読めないことに比べれば軽い。

- 代替案: `lipgloss` の `Style.Width()` に折り返させる。行数を数える前にレンダリングが要るうえ、Lip Gloss は右側を空白で埋めるので `renderDetail` の連結と噛み合わない（`emptyHintLines` が同じ理由で Lip Gloss の配置関数を避けている）

### D2. 折り返しは `cardHeaderLines` / `prHeaderLines` の中で行い、`renderDetail` は折り返し済みの行を切らない

ヘッダの行数は `detailHeader` が本文領域の高さを決めるのに使うので、折り返しは高さ計算より前に済んでいなければならない。
`renderDetail` が `ansi.Truncate` を全行に掛けたままだと、折り返し済みの行は幅以内なので実害は無いが、意図が読めなくなる。
ヘッダ行を組み立てる側が「この行はもう幅に収まっている」ことを保証し、`renderDetail` は切り詰めを続ける（1 行目以外の行は依然として切る必要がある）。

`internal/ui` に 1 つ関数を足す。

```go
// wrapTitle は `<prefix><title>` を幅 width で折り返し、2 行目以降を prefix の表示幅ぶん字下げする。
func wrapTitle(prefix, title string, width int) []string
```

`cardHeaderLines` は `wrapTitle(fmt.Sprintf("%s #%d  ", issue.Repo, issue.Number), issue.Title, m.width)`、
`prHeaderLines` は `wrapTitle(fmt.Sprintf("%s PR#%d  ", pr.Repo, pr.Number), pr.Title, m.width)` を呼ぶ。
呼ぶ側が 2 か所あるのでヘルパーを 1 つ置く（CLAUDE.md「一度しか使わない処理のために、ヘルパーや抽象化を増やさない」に触れない）。

### D3. インデントの余地が無い端末ではインデントを付けない

`width - StringWidth(prefix)` が 1 未満になる端末（狭い端末や長いリポジトリ名）では、字下げすると 1 行に 1 文字も入らず無限に行が増える。
このときは字下げを諦め、2 行目以降を幅 `width` でそのまま折る。タイトルが読めることを字揃えより優先する。

### D4. ヘッダが端末高を超えるときはタイトルの末尾行から落とす

`detailHeader` は今、PR 一覧を末尾から落として本文 1 行を確保している。PR 一覧を全部落としても足りないときに、折り返したタイトルの行を末尾から落とす。
最後に残ったタイトル行の末尾は `ansi.Truncate(l, m.width, "…")` に通して `…` にし、「まだ続きがある」ことを見せる。タイトルの 1 行目は必ず残す（残さないとどのカードを見ているのか分からなくなる）。

- 代替案: ヘッダをスクロールさせる。ヘッダ領域が固定であることは `card-detail` の Requirement で、この change の範囲を超える
- 代替案: 何もしない（現状のまま行数が端末高を超える）。alt screen のフレームが崩れるので採らない

### D5. Issue と PR のタイトルを同じ規則で扱う

proposal の「確定した判断」7 のとおり。カード詳細ヘッダ 1 行目は Issue のタイトルで、PR 詳細ヘッダ 1 行目は PR のタイトル。同じ位置・同じ見た目の行なので、片方だけ折り返すと理由を説明できない。

## Risks / Trade-offs

- **既存テストが切り詰めを期待している** → `detail_test.go` の「長いヘッダ行は幅で切り詰める」相当のテストは、タイトルの折り返しと `Summary` の切り詰めに分ける。spec delta の Scenario がそのまま対応する
- **本文領域が痩せる** → 長いタイトルのカードでは本文が数行減る。本文はスクロールできるので情報は失われない。高さの計算式は変えないので、既存の「低い端末では PR 一覧を削って本文 1 行を残す」Scenario は通る
- **英単語が途中で切れる** → D1 のトレードオフ。日本語タイトルが主なので、単語境界より幅の正しさを採る
- **`ansi.Hardwrap` の挙動がバージョンで変わる** → `go.mod` の `github.com/charmbracelet/x/ansi` を固定したまま使い、Scenario で表示幅を直接検証する（実装をハードコードしたテストにしない）

## Open Questions

なし。proposal の Q1 は spec delta と tasks を書き換える意思決定なので、後回しにできる未知としてこの節に置かず、`proposal.md` の `## 未確定の判断` に置いて PR コメントで人に問う。
