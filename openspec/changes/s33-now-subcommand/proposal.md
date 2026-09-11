issue: #36

## Why

`loop-cli` のサブコマンドは `version` と `update` の 2 つだけで（`cmd/loop-cli/main.go:57-68`）、
「今やる」に何が並んでいるかは TUI を人が開かないと分からない。AI agent（Claude Code など）は
TUI を操作できないので、同じ判定結果を使えず、人の出番を自分で拾えない。

分類そのものは既にライブラリとして揃っている（`internal/fetch` + `internal/classify`）。
足りないのは、それを 1 回の実行で読める形に出す入口だけである。

## What Changes

- `loop-cli now` サブコマンドを足す。設定ファイルの `repos` を全件取得し、`[1]今やる` に入るカードだけを
  TUI と同じ優先度順で標準出力に出して終わる（TUI は起動しない）
- 分類は `fetch.Fetch`（`internal/fetch/fetch.go:50`）と `classify.Card` をそのまま呼ぶ。判定は 1 行も書かない
- サブコマンドが設定ファイルと `gh` を使うのは `now` が初めてなので、`tui-entrypoint` の
  Requirement「引数はサブコマンドに振り分ける」と「起動失敗は標準エラーに出て終了コード 1 になる」を書き換える
  （`version` / `update` は今までどおり設定ファイルを読まず `gh` も呼ばない）
- 使い方（`usage`）と README にも `now` の 1 行を足す

## 確定した判断

1. **分類は既存のものをそのまま使う。** `fetch.Fetch`（`internal/fetch/fetch.go:50`）が open issue / PR を
   検索して詳細を並行取得し、`classify.Card` が 4 タブへ振り分ける。「今やる」は `model.TabNow`
   （`internal/model/model.go:62`）。`now` はこの結果を `Card.Result.Tab == TabNow` で絞るだけで、
   新しい判定ルールを持たない。`docs/domain/issue-driven-sdd/human-turn-signals.md` が正本のまま変わらない。
2. **並び順は TUI と同じ。** `internal/ui/rows.go:84-96` の `buildRows` は、カードを優先度の昇順で並べ、
   同じ優先度なら主体の `UpdatedAt` が新しいものを先に、それも同じならリポジトリ名の昇順、
   最後に番号の昇順で並べる。`now` はこの順をそのまま使い、並べ替えのキーを新しく決めない。
3. **「いま人が何をすべきか」の 1 行は既にある。** `model.Result.Summary`（`internal/model/model.go:130-136`）に
   `PR #131 の質問に答える`（`internal/classify/classify.go:116`）のような文字列が入っている。これをそのまま出す。
4. **スナップショットは読まないし書かない。** `~/.cache/loop-cli/snapshot.json` は TUI の `savingFetcher`
   （`cmd/loop-cli/main.go:189-198`）だけが書くもので、TUI を起動していない環境では存在しないか古い。
   agent が読む値としては当てにできないので、`now` は毎回 `gh` で取得し、結果で上書きもしない。
5. **部分失敗は結果を捨てない。** `fetch.Fetch` は詳細取得 1 件の失敗を `Result.Errors`
   （`internal/fetch/fetch.go:23-27`）に積んで一覧は返す。TUI がフッタに `詳細取得の失敗 N 件` を出すのと同じ扱いで、
   `now` も一覧を出したうえで失敗を伝え、終了コードは 0 にする。検索そのものの失敗（`Fetch` が error を返す）と
   `gh` の未認証・不在は終了コード 1 で終わる。`Errors` にはリポジトリのラベル一覧の失敗も入り
   （`internal/fetch/fetch.go:117-135`）、その場合は運用方式の判定が既定に倒れて分類がずれるので、
   `errors` が空でなければ一覧が不完全であり得ることを spec と README に書く。
6. **`gh` の確認と失敗の文言は TUI と同じ。** 実行の先頭で `gh.Client.Check` を呼び、`checkError`
   （`cmd/loop-cli/main.go:224-233`）と同じ 2 行を標準エラーに出して終了コード 1 で終わる。文言の正本は
   `tui-entrypoint` の Requirement「起動失敗は標準エラーに出て終了コード 1 になる」のままにし、
   `now-command` 側では定義し直さずに参照する。
7. **onboarding のフォームは出さない。** `now` は端末とは限らない場所（agent のサブプロセス）で走る。
   設定ファイルが無ければフォームに入らず、その場で終わる。
8. **出す範囲は「今やる」だけ、名前は `now`。** 他のタブ（バックログ / 進行中 / 異常）を選べる
   `--tab` は、要るようになってから別 issue で足す。issue #36 も「まずは『今やる』」と書いている
   （CLAUDE.md「投機的な機能・将来の拡張に備えたコードは書かない」）。
9. **リポジトリは設定ファイルからだけ引く。** `--repo owner/name` のような指定は足さない。
   `repos` が空の設定は `config.Load` が弾く（`internal/config/config.go:139-141`）ので、
   対象を書き忘れたまま GitHub 全体を検索する経路も生まれない。
10. **`subject` は指し先の種別と番号だけにする。** タイトル・URL・ラベル・更新時刻は `issue` と `prs` に載せ、
    `subject` は `{type, number}` でそこを指す。同じ値を二重に出さないため。主体の判定は
    `internal/ui` の `Subject`（`internal/ui/rows.go:45-54`）を使い、`internal/ui` には手を入れない。
11. **本文とコメントは出さない。** 出力が数百 KB になり、`gh` から取り直せる内容を二重に持つことになる。
    agent は `url` を見れば本文を取れる。
12. **`prs[].state` は出さない。** `Fetch` は open の検索結果からしか PR を作らないので
    （`internal/model/model.go:277-289` が `State: "OPEN"` を固定で入れる）、この欄は常に `OPEN` になる。
13. **余分な引数は黙って捨てず、エラーにする。** いまの `run` は `args[1:]` を見ないので
    `loop-cli now --repo x` が素通りする。`loop-cli-dev` は余分な引数も未知フラグも拒否している
    （`cmd/loop-cli-dev/classify_test.go:131-133`）ので、`now` もそちらに合わせる。
14. **JSON は 2 スペースで整形し、末尾に改行を付ける。** `jq` はどちらでも読めるので、人が直接打ったときに
    読める方を採る。
15. **出力は JSON だけにする。** PR #39 のコメント（2026-09-11）で「出力の方式は JSON そのままで良い」と
    決まった。人が読む表や `--json` フラグは持たず、`loop-cli now` は常に 1 つの JSON オブジェクト
    （取得時刻・件数・カードの配列・部分失敗の配列）を標準出力に出す。

16. **分類結果と状態は広く出す。** PR #39 のコメント（2026-09-11）で「状態なども、CLI の出力に含める
    ようにしてください」と言われたので、局面の記号（`situation`）と要約（`summary`）に加えて、
    画面の種別（`kind`）・優先度の整数（`priority`）・PR 詳細画面が出している状態
    （`未確定の判断` の件数・`mergeable`・`mergeStateStatus`・`reviewDecision`・checks・未解決の
    review thread 数）も出す。値はすべて取得済みのもので、`gh` の呼び出しは増えない。
17. **`kind` と `priority` を出す判断は明示的に選ばれた。** PR #39 で「画面の種別と内部の優先度も出すか」を
    問い、`Q2: B`（出す）の回答を得た（2026-09-11）。上の 16 と合わせて、出力の範囲は確定した。

## 明示的に延期した判断と残るリスク

- **`kind` と `priority` は画面のために作った値のまま機械契約になる。** 種別の日本語
  （`internal/model/model.go:109-128`）を変えたり、局面を 1 つ足して優先度の整数（同 `:69-90`）が
  ずれたりすると、それに依存した agent の分岐が黙って壊れる。仕組みとしての保護は置かず、
  「分岐に使うなら `situation` の記号の方が安定する」と README に書くだけにする。
  `internal/model` を触る後続の change は、`now` の出力が動くことを踏まえて判断する。
- **issue 側の状態を畳んだ欄は作らない。** `blocked` / `wip` / `question` は `issue.labels` に入り、
  局面は `situation` と `summary` で読める。TUI にも無い値を `now` だけが持つことになるので、
  要るようになってから別 issue で足す。
- **`errors` が空でないとき、`items` が不完全でも終了コードは 0 のまま。** リポジトリのラベル一覧の
  取得が失敗すると運用方式を判定できず既定の sdd に倒れるので、分類が静かにずれる。agent 側で
  `errors` を見ないと気付けない。終了コードを分ける案は採らない（一部の失敗で「今やる」が読めなくなる方が困る）。
- **呼び出しの間隔は CLI で制御しない。** 1 回の `now` は TUI の 1 回の取得と同じだけ `gh` を呼ぶ。
  短い間隔で叩けばレート制限に当たるが、キャッシュも制限も持たず agent 側の判断に任せる。
- **他のタブ（バックログ / 進行中 / 異常）は出さない。** `--tab` は別 issue。
- **手元で `loop-cli now` を実行する確認は tasks に入れない。** `gh` の認証と設定ファイルが要り、
  セッション内で完了できないため。代わりに fixture（`board` / `example`）と手で組み立てた
  `fetch.Result` で、出力のキー・並び順・状態の欄・失敗経路を単体テストで固定する。

## Capabilities

### New Capabilities

- `now-command`: `loop-cli now` が設定のリポジトリを取得し、「今やる」のカードだけを機械が読める形で
  標準出力に出して終わる。出力のキーと引数の扱いを定め、取得の失敗と部分失敗の扱い、終了コード、
  スナップショットを触らないことまでを含む

### Modified Capabilities

- `tui-entrypoint`: Requirement「引数はサブコマンドに振り分ける」に `now` を足し、
  「サブコマンドは設定ファイルを読まず `gh` も呼ばない」を `version` / `update` だけの規則に直す。
  使い方（`unknown command` のときに出す文面）にも `now` の 1 行を足す。
  Requirement「起動失敗は標準エラーに出て終了コード 1 になる」は、主語を TUI の起動に限定し、
  設定ファイルと `Check` の文言をサブコマンドから参照できる正本として位置づけ直す

## Impact

- `cmd/loop-cli/main.go`: `run` の `switch` に `now` を足し、依存を束ねて `runNow` に渡す。`usage` に 1 行足し、
  `run` の doc コメント（`cmd/loop-cli/main.go:48`）を `version` / `update` に限定した文に直す
- `cmd/loop-cli/now.go` と `cmd/loop-cli/now_test.go`: 出力の組み立て、`runNow`、そのテスト
- `internal/fetch` / `internal/classify` / `internal/model` / `internal/ui`: 変更しない（読むだけ）
- `README.md`: 「更新」の節の近くに `now` の説明を足す
- `docs/mvp` と `docs/domain/issue-driven-sdd/human-turn-signals.md`: 変更しない（判定は変わらない）
