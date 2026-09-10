# design: 2026-09-10-s26-wrap-titles

Refs #2

## Context

`internal/ui/detail.go` の `prHeaderLines` はヘッダ行を `[]string` で返し、`cardHeaderLines` も同じ形で返す。
`internal/ui/view.go` の `renderDetail` はその各行を `ansi.Truncate(l, m.width, "…")` に通してから区切り線・本文・フッタと連結する。
本文領域の高さは `detailHeader` が `max(m.height-2-len(header), 1)` で決め、1 行を割るときは PR 一覧を末尾から落とす。

つまりヘッダは既に行のスライスであり、1 つの論理行が 1 つの表示行に対応するという前提を持たない。
折り返しは 1 行目をスライスの複数要素に展開するだけで済む。

制約は 3 つある。

- 表示幅は `github.com/charmbracelet/x/ansi` の `StringWidth` で測る。プロジェクトの既存コードはすべてこれを使っており、`len` や `utf8.RuneCountInString` は使わない
- `detailHeader` は `[]string` を不透明に受け取るので、今のままではどの要素がタイトル行かを区別できない
- 折り返しでヘッダが伸びると本文領域が痩せる。`detailHeader` は本文領域の高さをヘッダの行数から引いて出すので、折り返しで増えた行もそのまま引かれる。ただし固定ヘッダ自体が端末高を超える経路が、折り返しによって新たに現実的になる

## Goals / Non-Goals

**Goals:**

- PR 詳細画面のヘッダ 1 行目でタイトルを全文読めるようにする
- 折り返した行をタイトルの開始位置に縦揃えし、どこからがタイトルの続きかを目で追えるようにする
- 全角文字・絵文字を含むタイトルで桁がずれないようにする
- 折り返しを原因として画面の行数が端末の高さを超えないようにする

**Non-Goals:**

- キュー画面の表とプレビューの変更（proposal の Q1 の回答を待つ）
- カード詳細ヘッダの Issue タイトルの折り返し（proposal の Q2 の回答を待つ）
- 折り返しと無関係に既に存在する高さ超過の経路を直すこと（proposal の「確定した判断」9）
- 本文（Glamour が折り返す）とフッタ・PR 一覧・labels 行の折り返し
- 詳細ヘッダのスクロール（ヘッダ領域は固定のまま）

## Decisions

### D1. 折り返しは `ansi.Wrap` を使う

`ansi` v0.11.8 の `Wrap`（`wrap.go:267-278`）は「必要なら単語境界を割る」と契約している。実測では全角 28 字を幅 20 で折ると各行の表示幅が 20 / 20 / 20 / 20 / 4 になり、英語の文では単語境界を守って 12 / 14 / 15 / 17 になる。日本語タイトルでは幅どおりに折れ、英語混じりのタイトルでは単語が途中で切れない。
幅の計算は `GraphemeWidth`（`width.go:66-68`）で、`StringWidth` と同じ規則なので、Scenario が検証する表示幅と一致する。

- 代替案: `ansi.Hardwrap`。実測で日本語の結果は `Wrap` と同一だが、英語では単語の途中で割る（20 / 20 / 20 / 1）。読みやすさで劣るので採らない
- 代替案: `ansi.Wordwrap`。長い語を割らないので、空白の無い日本語タイトルが 1 行のまま（表示幅 84）返る。幅の保証が崩れるので採らない
- 代替案: `lipgloss` の `Style.Width()`。行数を数える前にレンダリングが要るうえ、Lip Gloss は右側を空白で埋めるので `renderDetail` の連結と噛み合わない（`emptyHintLines` が同じ理由で Lip Gloss の配置関数を避けている）

### D2. 折り返しは接頭辞を除いた幅で行い、1 行目にだけ接頭辞を前置する

`ansi.Wrap(prefix+title, width, "")` を素直に呼ぶと、2 行目以降が幅 `width` いっぱいで返る。そこへ接頭辞ぶんの字下げを足すと `width` を超える。
そこでタイトルだけを `width - StringWidth(prefix)` で折り、1 行目に接頭辞を、2 行目以降に同じ表示幅の空白を前置する。これで全行が `width` 以下に収まる。

`internal/ui` に関数を 1 つ足す。

```go
// wrapTitle は `<prefix><title>` を幅 width に収めた行に割り、2 行目以降を prefix の表示幅ぶん字下げする。
func wrapTitle(prefix, title string, width int) []string
```

境界の扱いを 3 つ決める。

- `width < 1`（初回の `WindowSizeMsg` が届く前の `m.width == 0` を含む）: `prefix + title` の 1 行をそのまま返す。`renderDetail` の `ansi.Truncate` が従来どおり切る
- `width - StringWidth(prefix) < 1`: 字下げすると 1 行に 1 列も入らず行が無限に増える。字下げを諦め、`prefix + title` 全体を幅 `width` で折る
- それ以外: 上の規則どおり

### D3. 折り返しは `prHeaderLines` の中で行い、`renderDetail` は変えない

ヘッダの行数は `detailHeader` が本文領域の高さを決めるのに使うので、折り返しは高さ計算より前に済んでいなければならない。
折り返し済みの行はすべて幅以下なので、`renderDetail` の `ansi.Truncate(l, m.width, "…")` はそれらに対して何もしない（`truncate.go:66-69` が `StringWidth(s) <= length` で入力をそのまま返す）。labels 行や PR 一覧の行を切る役目は残るので、`view.go` は変更しない。

### D4. ヘッダが端末高を超えるときはタイトルの末尾行から落とす

`detailHeader` は今、PR 一覧を末尾から落として本文 1 行を確保している。PR 一覧を落としても足りないときに、折り返したタイトル行を末尾から落とす。タイトルの 1 行目は必ず残す（残さないとどの PR を見ているのか分からなくなる）。

これを実装するには `detailHeader` がタイトル行の範囲を知る必要がある。`prHeaderLines` / `cardHeaderLines` が行数を添えて返す形にする。

```go
// prHeaderLines は PR 詳細のヘッダ行と、その先頭何行がタイトル行かを返す。
func (m Model) prHeaderLines() (lines []string, titleLines int)
```

`detailHeader` は `fixed` の先頭 `titleLines` 行だけを削減の対象にする。

落とした後に残る最後のタイトル行へ `…` を付けるとき、`ansi.Truncate(l, m.width, "…")` は使えない。折り返し済みの行は幅以下なので `Truncate` が何もせず、`…` が付かないためである（実測で確認）。`ansi.Truncate(l, m.width-1, "") + "…"` のように、幅を 1 列空けてから明示的に付ける。

`#2` が直したいのは端末**幅**による切り詰めなので、端末の**高さ**が足りない場合に末尾を諦めることは依頼と衝突しない。高さが足りない端末では、そもそも全文を同時に出す手立てが無い。

- 代替案: ヘッダをスクロールさせる。ヘッダ領域が固定であることは `card-detail` の Requirement で、この change の範囲を超える
- 代替案: `Summary` や `段階` の行を先に落とす。`段階` は `docs/mvp/mvp.md:85` がヘッダの内容として列挙しており、落とすと mvp.md の改訂が要る
- 代替案: 何もしない（行数が端末高を超えるまま）。alt screen（`view.go:41-45`）のフレームが崩れるので採らない

## Risks / Trade-offs

- **既存テストが切り詰めを期待している** → `detail_test.go` の PR 詳細ヘッダに関するテストは、タイトル行の折り返しと labels 行の切り詰めに分ける。spec delta の Scenario がそのまま対応する
- **タイトルが複数行に割れると `strings.Contains(view, title)` で検証できない** → Scenario が定めたとおり、タイトル行を接頭辞と字下げを剥がして連結してから `Title` と比較する。画面全体から空白を除いて `Contains` する書き方は、折り返しが壊れていても通ってしまうので使わない
- **本文領域が痩せる** → 長いタイトルの PR では本文が数行減る。本文はスクロールできるので情報は失われない。高さの計算式は変えないので、既存の「低い端末では PR 一覧を削って本文 1 行を残す」Scenario は通る
- **`prHeaderLines` の戻り値が 2 つに増える** → 呼び出し元は `detailHeader` だけなので影響は閉じている。`cardHeaderLines` は Q2 の回答が「Issue タイトルも折り返す」になったときに同じ形へ揃える
- **`ansi` の挙動がバージョンで変わる** → `go.mod` の `github.com/charmbracelet/x/ansi` を v0.11.8 に固定したまま使い、Scenario で表示幅を直接検証する（実装をハードコードしたテストにしない）

## 未決事項

docs/mvp と docs/domain が沈黙している点。実装者が選ぶ既定値を 1 つずつ示す。

- **折り返しアルゴリズム**: 既定値は `ansi.Wrap`（D1）。mvp.md は折り返しに触れていない
- **継続行の字下げ幅**: 既定値は接頭辞 `<Repo> PR#<Number>  ` の表示幅（#2 が「タイトルの開始位置に合わせてインデントする」と書いている）
- **字下げの余地が無い端末での扱い**: 既定値は字下げを付けず端末幅で折る（D2）
- **`m.width == 0` の扱い**: 既定値は折り返さず 1 行で返す（D2）
- **高さが足りないときに落とす順**: 既定値は PR 一覧 → タイトルの末尾行（D4）。`Summary` と `段階` は落とさない
- **落としたことの見せ方**: 既定値は残った最後のタイトル行の末尾を `…` にする（D4）

## Open Questions

なし。proposal の Q1 / Q2 は spec delta と tasks を書き換える意思決定なので、後回しにできる未知としてこの節に置かず、`proposal.md` の `## 未確定の判断` に置いて PR コメントで人に問う。
