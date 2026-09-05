## Context

s03-gh-client が `internal/gh`（`GHClient` / `Client` / `Fake`、fixture の命名規則、手書きの `example` fixture）を定義している。稼働リポジトリ由来の fixture はまだ無く、s05-classify はそれを `Fake` で読んで局面 A〜G をテストする（V-1）。
`cmd/sugi-loop-cli` はまだ無い。implementation-tasks.md §2 の「動作確認用 CLI」は 85-create-cli を参照しているが、そのスキルは TypeScript / pnpm 前提なので、構造（1 エントリ + サブコマンド、フレームワーク無し、dev 専用）だけを借りて Go の標準ライブラリで書く。

前提にする s03 の決定:
- fixture の内容は `Client` の標準出力そのまま。`pr-<n>.json` だけは `ViewPR`（7 フィールド）と `ViewPRMergeState`（4 フィールド）を合わせた 11 フィールドの 1 回の `gh pr view` 出力で、`Fake` は両メソッドでこの 1 ファイルを読む
- `Client` の実行関数 `run` は非公開で、テストは同じパッケージ内で差し替える。`RetryWait` は `ViewPRMergeState` の UNKNOWN 再取得の待ち
- GraphQL は `-f owner= -f name= -F number= -f query=`。`LabelTimeline` は `--paginate --jq` でオブジェクトの連続
- `Fake` は `ViewIssue` と書き込みを `Calls` に記録する

手元の `gh` 2.93.0 で公開リポジトリ `cli/cli` に対して確認した事実: `gh issue view --json comments` / `gh pr view --json comments` の `author` は `{"login":"…"}` のみで、表示名やメールは入らない。GraphQL の reviewThreads も `author{login}` だけを要求する。search の `--json` フィールドに author は無い。よって fixture に含まれる個人識別子は login（`author.login` と本文中の `@mention`）、本文中のメールアドレス、`owner/name` の 3 種類に限られる。

採取対象のリポジトリは human-turn-signals.md「稼働中プロジェクトの実データで確認した事象」で読んだものと同じ種類のリポジトリで、routine コメントは利用者本人のアカウントで投稿されている。つまり owner（個人アカウントの場合）が `author.login` にも `@mention` にも現れる。

## Goals / Non-Goals

**Goals:**
- `fixture capture --repo owner/name --alias <alias>` 1 コマンドで、V-1 に使える fixture が s03 の命名で `internal/gh/testdata/fixtures/<alias>/` にできる
- 伏せ字の規則を機械的に決め、分類に使う文字列を壊さない
- 採取した fixture が個人情報を含まないことを、コミット前にテストで確認できる
- `cmd/sugi-loop-cli` の骨組み（振り分け・`help`・終了コード）を s06 がそのまま拡張できる形で置く

**Non-Goals:**
- `classify` / `notify test` サブコマンド（s06）
- 複数リポジトリの一括採取、closed issue / merged PR の採取（V-1 は open のみで足りる。search が open だけを返す）
- 書き込み系の検証手段（validation-plan.md「未定」。発明しない）
- 伏せ字の完全性の保証。テキスト置換で機械的に取れるもの（owner / repo / login / mention / メール）だけを対象にし、本文の自由記述に含まれる固有名詞は利用者が目視で確認する
- fixture の差分更新・マージ（再採取はディレクトリごと作り直す）

## Decisions

### ファイル構成

```
cmd/sugi-loop-cli/main.go          # usage 文字列、run(args, stdout, stderr) int、main()
cmd/sugi-loop-cli/main_test.go     # help / 引数なし / 未知 / fixture 単独 / fixture foo の終了コードと出力
cmd/sugi-loop-cli/fixture.go       # fixtureCapture(args, stdout, stderr) error: flag を解析し、Check・Capture・redact・書き込み・自己検査・要約をこの順に行う
cmd/sugi-loop-cli/redact.go        # redact(files map[string][]byte, owner, name, alias string) (map[string][]byte, int, error)
cmd/sugi-loop-cli/redact_test.go   # spec「伏せ字は…」の Scenario をそのまま
internal/gh/capture.go             # (*Client).Capture
internal/gh/capture_test.go        # run を差し替えて順序・引数・UNKNOWN 再取得・途中失敗
internal/gh/fixtures_test.go       # testdata/fixtures/*/ の全ファイルを検査（login 形式 / メール / SUGI_LOOP_FIXTURE_ORIGIN）
internal/gh/client.go              # 変更: 引数組み立てと pr view の UNKNOWN 再取得を非公開関数に切り出す
internal/gh/fake.go                # 変更: ファイル名の組み立てを非公開関数に切り出す
.gitignore                         # 追加: /sugi-loop-cli /cmd/sugi-loop-cli/sugi-loop-cli
```

### Capture は internal/gh に置き、cmd は伏せ字と保存だけを持つ

`Client` の `run` と引数組み立ては非公開なので、生の標準出力を取るには `internal/gh` の中に居る必要がある。代替案は (a) `Client` に `Raw(ctx, args...)` を公開して cmd 側で引数を組み立てる、(b) 読み取り 8 メソッドそれぞれに生バイト列を返す公開版を足す。(a) は spec で確定した引数が 2 か所に書かれ、(b) は公開 API が倍になる。`Capture` 1 メソッドを足し、内部で読み取りメソッドと同じ非公開の引数組み立て関数を使う案が、公開面が最小で引数の単一ソースを保てる。
その代わり s03 の `client.go` を少し直す。各読み取りメソッドが `args := argsXxx(...)` → `c.run` → `decodeXxx` の形になるよう引数組み立てを非公開関数に切り出し、`ViewPRMergeState` の「実行して `mergeable` が `UNKNOWN` なら `RetryWait` 後に 1 回再実行」を `prViewRaw(ctx, repo, n, fields string) ([]byte, error)` のような非公開関数に切り出し、`Capture`（11 フィールド）と `ViewPRMergeState`（4 フィールド）の両方がこの関数を呼ぶ。`prViewRaw` は 1 回目の出力を `decodePRMergeState` に通して `Mergeable == "UNKNOWN"` を判定し、再取得しても UNKNOWN なら 2 回目の出力をそのまま返す（エラーにしない。s03 の `TestViewPRMergeStateStopsAfterSecondUnknown` がこれを検証している）。s03（4f9ad8b）の `client.go` には `argsXxx` も `prViewRaw` も無く、`viewPRMergeStateOnce`（実行 + 4 フィールドのデコード）だけがあるので、これを `prViewRaw` + `decodePRMergeState` に置き換えて消す。`ViewPR` はこの関数を通さない（再取得しない、という s03 の決定を変えない）。
`Capture` は伏せ字を知らない。伏せ字は fixture の都合であり `gh` の都合ではない。`internal/fixture` のような新パッケージも作らない（使うのは CLI 1 か所）。

`Capture` の返り値を `map[string][]byte` にするのは、伏せ字の login 表が全ファイルにまたがる（同一人物を同一番号にする）ためで、1 ファイルずつ書き出す streaming 形にできない。`progress func(name string)` は各 `gh` 実行の直前に呼び、CLI が進捗を標準エラーに出す。1 リポジトリで 100 回前後の `gh` 実行になり、無言だと止まって見えるため。nil 判定は入れない（CLI は常に渡し、テストは空関数を渡す）。

### 採取範囲は全 open issue / 全 open PR の詳細

D-001 の遅延取得は「`question` の issue だけ comments を取る」のように分類結果に応じて絞るが、fixture は分類器そのものをテストするためのものなので、絞る条件を採取側が知っていてはいけない（分類器のロジックを採取側に写すことになる）。全件取れば s05 はどの issue / PR に対しても `Fake` の 8 メソッドを呼べる。呼び出し回数は search 2 回 + issue 3 回 × 件数 + PR 2 回 × 件数で、issue 30 件・PR 10 件なら 52 回。search API の 30 req/分には search 2 回しか当たらず、REST / GraphQL の 5,000 req/時にも遠い。

### 伏せ字はテキスト置換で行う

JSON にデコードして値だけを書き換えて再エンコードする案は、キー順・空白・`<` `>` `&` のエスケープが `gh` の出力と変わる（Go の既定エンコーダは `<` `>` を `\u003c` `\u003e` にエスケープするので、`<!-- routine -->` が `\u003c!-- routine --\u003e` になる）。timeline はオブジェクトの連続で、配列として扱えない。テキスト置換なら `gh` の出力を置換箇所以外そのまま保て、`Fake` と `Client` が同じバイト列を読む前提が崩れない。
テキスト置換の副作用は「JSON キーやラベル名と同じ文字列を持つ login / リポジトリ名が来たとき、キーやラベルまで書き換える」ことで、これは spec の衝突ガードで採取をエラーにして止める。ガードは (a) 対象文字列が `"<x>":` の形でどこかに現れる（= JSON キー）か、`login` / `name` 以外のキーの文字列値 `"<キー>":"<x>"` として現れる（= `SUCCESS` / `COMPLETED` / `UNKNOWN` / `labeled` / `OPEN` / `MERGED` / `CheckRun` のような列挙値。`login` は置換対象そのもの、`name` は `repository.name` にリポジトリ名が必ず入るので除く）、(b) ラベル・マーカー語の固定リスト（`todo` / `propose` / `apply` / `archive` / `question` / `blocked` / `wip` / `docs` / `routine` / `human` / `stage` / `pr`）の 2 つ。キーと列挙値の一覧をコードに持たず (a) で導くのは、`gh` の出力キーや値が版で増えても追従が要らないため。比較はすべて大文字小文字を区別しない。
`@mention` だけから得た login（`author.login` に現れない）は `@x` → `@user-N` の形でしか置換せず、単語としての全文置換とガードの対象にしない。本文には `@types/node` や `@v2` のような login でない `@` 付きの語が普通に現れ、これを全文置換とガードにかけると `@context` のような語で採取が止まる（偽陽性）。`author.login` に現れる login は本文中に裸で書かれることがある（「x さんに確認」）ので全文置換する。

置換規則と順序は spec の Requirement「伏せ字は個人・組織情報だけを置き換え、分類に使う文字列を変えない」に書いたとおり。実装のメモ:
- 単語境界は Go の `regexp` に後読みが無いので `(^|[^A-Za-z0-9-])(?i:<x>)([^A-Za-z0-9-]|$)` の形で前後を捕捉し、`${1}user-N${2}` で戻す。`\b` は `-` を境界にするので使わない（`al` が `al-team` に当たる）。login 1 つにつき 1 回 `ReplaceAllString` を回す（隣接する 2 つの login を 1 回の走査で処理すると 2 つ目の境界文字が消費されて漏れる）
- `<x>` は `regexp.QuoteMeta` を通す（login と repo 名は `[A-Za-z0-9-._]` だが、`.` を含み得る）
- login 表は `map[string]int`（キーは小文字化した login）と、番号順の一覧。owner は先に 1 を割り当てる。`owner` / `name` / login の比較と置換はすべて大文字小文字を区別しない（`(?i:…)`）。`--repo` を search 結果の `nameWithOwner` で正規化する案は、`gh` を呼ぶ前に検証を終えられなくなるので採らない
- `"login"` 値の抽出は `"login"\s*:\s*"([^"]*)"`。`gh` の `--json` は 1 行の compact JSON だが、空白は許しておく
- メールの置換値 `user@example.com` は RFC 2606 の予約ドメイン。login `user` と衝突しても（`user` は `user-N` 形式でない）`@example` は直前が `r` なので mention に当たらない

### 自己検査は Fake で読み直す

伏せ字後のファイルを書いたあと `gh.NewFake(dir)` で 8 メソッドを全番号について呼び、エラーが出たらディレクトリを消してエラーにする。テキスト置換が JSON を壊す（理論上は起きないが、置換値に `"` を含めるような改修ミス）ことと、「s05 が `Fake` で読める」ことの両方を、採取の時点で確かめる。数十回のファイル読み込みで済み、`gh` は呼ばない。

### 個人情報の検査は internal/gh のテストに置く

fixture は `internal/gh/testdata/fixtures/` にあり、`go test ./internal/gh/` の一部として走るのが自然。検査は 3 つ。
1. `"login"` 値が `^user-[0-9]+$` に一致（伏せ字の漏れを機械的に検出）
2. `user@example.com` 以外のメールアドレスが無い（伏せ字の置換値がメールの正規表現に一致するため）
3. `SUGI_LOOP_FIXTURE_ORIGIN=owner/name` があれば、`example` 以外のディレクトリに `owner` と `name` が（大文字小文字を区別せず）どこにも無い

3 を環境変数にするのは、元の `owner/name` をリポジトリにコミットできないため。未設定なら `t.Skip` で「飛ばした」ことを出力に残す。`github.com/<x>/` の `<x>` が `org` であることの検査は、owner 単独を `user-1` にする規則と矛盾する（`github.com/owner/other-repo` → `github.com/user-1/other-repo`）ので入れない。3 が `example` を見ないのは、`example` が `org/app` 固定の手書きで、元リポジトリの `name` が `app` のとき偽陽性になるため。
1 と 2 は `example` にも掛かる。s03 の `example` の login は `alice` / `bob` / `routine-bot` で `user-N` 形式ではないので、値を `user-N` に直し、それを見ている `fake_test.go` の期待値 2 か所も合わせて直す（tasks 5.1）。

### CLI の骨組み

- `run(args []string, stdout, stderr io.Writer) int` が `args[0]` で振り分ける。`help` / `-h` / `--help` / 引数なし → usage を stdout、0。`fixture` → `args[1] == "capture"` なら `fixtureCapture(args[2:], stdout, stderr)`、それ以外は unknown。unknown → `unknown command: <args[0]>`（`fixture` 単独なら `fixture`）と usage を stderr、1
- サブコマンドは `error` を返し、`run` が `fmt.Fprintln(stderr, err)` して 1 を返す。`os.Exit` は `main` の 1 か所
- フラグは `flag.NewFlagSet("fixture capture", flag.ContinueOnError)` で `SetOutput(stderr)`。`--repo` / `--alias` の 2 つだけ。出力先を変えるフラグは作らない（保存先は s03 が固定している）
- usage は文字列定数 1 つ。s06 は行を足すだけ

### 実行環境の前提

- `context.Background()` を渡す。タイムアウトは付けない（採取は対話的に実行し、遅ければ Ctrl+C で止める。s03 が「タイムアウトは呼び出し側」と決めているが、CLI では不要）
- カレントディレクトリがリポジトリのルートであることを `internal/gh/testdata/fixtures` の存在で確かめる。`go run ./cmd/sugi-loop-cli …` をルートで実行する前提

## Risks / Trade-offs

- [本文中の固有名詞（顧客名・製品名・他リポジトリ名）は機械的に伏せられない] → 採取後に利用者が `git diff` でファイルを読んでからコミットする手順を tasks に入れる。`SUGI_LOOP_FIXTURE_ORIGIN` の検査は owner / repo 名の漏れだけを見る
- [リポジトリ名が英単語（`app` 等）だと本文の同じ単語も `<alias>` になる] → 分類器はラベル・マーカー・`## Q1.`・`未確定の判断` を見るので影響しない。衝突ガードのリストにある語はエラーで止める
- [login が短い（2 文字）と本文の無関係な語に当たる] → 単語境界で防ぐ。境界内で一致するものは同じ綴りの語であり、置き換わっても分類に影響しない
- [`gh` の出力が pretty print になっていると `"login"` 抽出の正規表現が空白を挟む] → `\s*` を許す。`gh --json` は端末でも 1 行で出す
- [再採取で `<alias>` ディレクトリを消す] → `example` は alias に使えないので消えない。消すのは fixture ディレクトリ 1 つで、git にあるので戻せる
- [採取中に issue が閉じる・PR が増える] → search 結果の番号だけを対象にする。search 後に閉じた issue の `issue view` は成功する（closed でも見える）ので失敗しない
- [`Capture` が `client.go` の内部構造に依存する] → 同じパッケージ内の非公開関数の共有であり、公開 API は増えない。s03 の `client_test.go` が引数を検証しているので切り出しで引数が変われば検知される
- [衝突ガードが厳しすぎて採取できないリポジトリがある] → エラー文言に衝突した文字列を出す。回避策（別リポジトリを選ぶ、または例外を足す）は利用者が決める。docs に無いので自動回避は入れない

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| 採取する詳細の範囲 | search で得た全 open issue の view / cross-refs / timeline、全 open PR の view（11 フィールド）/ review-threads | 上記 Decisions。分類条件を採取側に写さない |
| `pr-<n>.json` の UNKNOWN 再取得 | `ViewPRMergeState` と同じ処理（`prViewRaw`。`decodePRMergeState` で UNKNOWN を判定）を通す（1 回だけ） | fixture が `UNKNOWN` だらけだと局面 C を fixture でテストできない。s03 も `ViewPRMergeState` では再取得している |
| alias の形式 | `^[a-z0-9-]+$`、`example` は不可、`owner` / `name` を（大文字小文字を区別せず）含むのも不可 | ディレクトリ名として安全で、s03 の `example` を壊さない。alias が `name` を含むと置換後に元の名前が残り、`SUGI_LOOP_FIXTURE_ORIGIN` の検査で落ちる |
| owner 単独の置換先 | `user-1`（login 表の先頭に固定） | owner が個人アカウントなら `author.login` にも現れ、同一人物同一番号の規則に乗る。組織でも「user-1 = このリポジトリの所有者」と読める |
| `owner/name` の置換先 | `org/<alias>` | mvp.md の画面例が `org/app` 形式。`nameWithOwner` の `/` 区切りを保つ |
| リポジトリ名単独の置換 | 単語として `<alias>` に置き換える | `repository.name` フィールドと本文中の言及を伏せる。衝突はガードで止める |
| `@mention` だけの login の扱い | `@x` → `@user-N` のみ。全文置換とガードの対象外 | `@types` `@v2` 等の偽陽性で採取が止まるのを避ける |
| メールの置換値 | `user@example.com` | 予約ドメイン。1 種類に潰す（メール同士の同一性は分類に不要） |
| 既存の `<alias>` ディレクトリ | 消してから書く | 再採取で閉じた issue のファイルが残るのを防ぐ |
| 伏せ字の実装場所 | `cmd/sugi-loop-cli/redact.go`（package main） | 使うのは CLI だけ。新パッケージを作らない |
| `progress` の形 | `func(name string)`、nil 不可 | 最小。CLI は標準エラーに 1 行ずつ出す |
| タイムアウト | 付けない | 対話実行。Ctrl+C で止める |
| 元 owner 検査の環境変数名 | `SUGI_LOOP_FIXTURE_ORIGIN`（値は `owner/name`） | 他の環境変数と衝突しない接頭辞 |
| 元 owner 検査の一致方法 | 部分一致（大文字小文字を区別しない）。`example` は対象外 | 伏せ字より厳しく見て漏れを拾う側に倒す。owner が 2〜3 文字だと本文の無関係な語に当たり偽陽性になるが、その場合は失敗メッセージのファイルと文字列を利用者が読んで判断する（境界付き一致に緩めない） |
| 要約の文言 | `internal/gh/testdata/fixtures/<alias> に <N> ファイルを書きました（issue <i> / PR <p> / 伏せた login <l>）` | 日本語 1 行。テストは数値と alias の包含だけを見る |
| 衝突ガードのラベル・マーカー語 | `todo` / `propose` / `apply` / `archive` / `question` / `blocked` / `wip` / `docs` / `routine` / `human` / `stage` / `pr` | human-turn-signals.md のラベル名（`stage:` とその接頭辞）、マーカー、`blocked-by: human`、`## PR リスク評価` |
