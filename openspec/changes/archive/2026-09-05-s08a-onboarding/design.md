## Context

s08 までの `cmd/sugi-loop/main.go` は `DefaultPath` → `Load` → `NewClient` → `Check` → `ui.New` → プログラム実行を `run() error` で直列に行い、失敗は `main()` が標準エラーに 1 行出して exit 1 にする。設定ファイルが無いときは `Load` の `open … no such file` で終わる。`gh.Client.Check`（s03）は `gh` が無いときは `exec.LookPath` のエラーを `%w` で包んだ文言、`gh auth status` が非 0 のときは `*gh.Error{Args, ExitCode, Stderr}` を返す。`internal/ui` は `fetchedMsg` で `cards` / `errText` / `partial` / `fetching` を持ち、表は `tableLines(h)` が現在タブの行を h 行に切って返す。
mvp.md「初回起動（onboarding）」が 4 項目（フォームで作る / gh の案内 / 空キューのヒント / 壊れた設定は上書きしない）を定めた。この change はその 4 項目だけを実装する。

## Goals / Non-Goals

**Goals:**
- 設定ファイルが無い端末での起動を、huh のフォーム → 書き出し → 通常起動の 1 本の流れにする
- `gh` が無い・未認証のときに、原因と次の一手を 2 行で示す
- 取得成功で 0 件のとき、mvp.md のヒントを表の領域に出す
- 壊れた設定は onboarding に入らず従来どおり終了する（上書きしない）

**Non-Goals:**
- 既存の設定ファイルを編集・再作成するサブコマンド（`sugi-loop init` のようなもの）は作らない。docs に無い
- `refresh_interval_sec` やリポジトリ別 `merge_method` をフォームで聞くこと。mvp.md が「聞かず既定 120」と決めている。リポジトリ別上書きは手で編集する
- フォームの中の `gh` 呼び出し（リポジトリの存在確認など）。`Check` は書き出し後に 1 回だけ
- 空キューのヒント以外の空状態（タブ単位の空表示、詳細画面の空）は扱わない
- スナップショットからの stale 表示（s13）

## Decisions

### 判定順は「存在確認 → フォーム → Load」で、`Load` の中身は変えない

`config.Load` は「無い」も「壊れている」も同じ `error` で返すので、onboarding の分岐に `Load` のエラーを使うと、壊れた設定を「無い」と誤って上書きする余地ができる。`os.Stat` で「存在しない」（`errors.Is(err, fs.ErrNotExist)`）を先に見て、それ以外は全部 `Load` に任せる。`Load` は変えない。
判定は `cmd/sugi-loop/main.go` の非公開関数 `ensureConfig(path string, isTerminal bool, form func(path string) error) error` に置く。`run()` は `ensureConfig(path, term.IsTerminal(os.Stdin.Fd()), onboarding.Run)` のように呼び、テストはスタブを渡す。`run()` の順は 存在確認 → `Load` → `NewClient` → `Check` → `ui.New` → プログラム実行。`tui-entrypoint`「1 フレーム描画」を MODIFIED し、手順 1 を「存在確認 → 無ければフォーム → `config.Load`」に変え、Scenario「設定ファイルが無い端末ではフォームの後にキュー画面が出る」を 1 件足す（写し元は s10 の現行版。下記「先行 change との衝突」）。判定の詳細は `onboarding` capability の Requirement「設定ファイルが無いときだけ onboarding に入る」に書いてある。
端末かどうかは標準入力で判定する。`github.com/charmbracelet/x/term` が Bubble Tea 経由で go.sum にあり、`IsTerminal(fd)` を持つ（無ければ `golang.org/x/term`）。判定に使う関数名はどちらでも良く、spec は振る舞いだけを定める。

### onboarding は `internal/onboarding` に置く

D-003 の内部構成は `ui/` に `form` を含めているが、onboarding のフォームは Bubble Tea の `Model` の外（TUI を起動する前）で完結し、`internal/ui` の型を何も使わない。`internal/ui` に置くと `ui` のテスト（`New` の署名が s10 で変わる最中）に巻き込まれる。s07 が `internal/fetch` を足した前例に倣い、`internal/onboarding` を独立させる。`internal/config` を import する（`MergeMethod` の型と `IsRepoName`）。逆向きの依存は無い。
公開するものは `Answers` / `Marshal` / `Write` / `ParseRepos` / `Run` / `ErrAborted` の 6 つ。`Run` だけが huh を使う。

### フォームは huh、テストは huh を回さない

huh は `charm.land/huh/v2` を `go get` する（s08 design は「s15 まで入らない」としたが前倒し。s15 の issue 作成フォームは同じ依存を使う）。項目は mvp.md のとおり 4 つ。
- `repos`: 複数行入力（huh の Text 相当）。検証関数に `ParseRepos` を渡し、エラーならその文言を出して先に進ませない。プレースホルダは `org/app\norg/web`
- `merge_method`: select。選択肢は `squash` / `merge` / `rebase` の 3 つで既定 `squash`。`config.MergeSquash` などの定数を値にする
- `notify`: confirm。既定 true
- `editor`: 1 行入力。既定値は文字列 `$EDITOR`。展開しない（展開すると今日のエディタ名がファイルに固定され、`config.Load` の展開が無意味になる）
フォームの中止は huh が返す「利用者が中止した」を表すエラーで判定し、`ErrAborted` に写す。huh のエラー値の名前は README で確認する。
テストは Bubble Tea の Program を回さず、`ParseRepos` / `Marshal` / `Write` を直接検証する（ブリーフ「入力値 → YAML 文字列生成 → `config.Load` で読める」）。`Run` は手動確認だけ。

### YAML は `yaml.v3` のエンコーダで書き、mvp.md の例とバイト単位で一致させる

`Marshal` は `repos` / `refresh_interval_sec` / `merge_method` / `editor` / `notify` の順にフィールドを持つ非公開の構造体を `go.yaml.in/yaml/v3` の Encoder（インデント 2）で書く。`fmt` で組み立てると `editor` に YAML の特殊文字（`:` や `#`）が入ったときの引用を自前で扱うことになる。エンコーダなら `$EDITOR` は引用なしのまま出て、mvp.md の例（コメントを除く）とバイト単位で一致する。テストがそれを固定する。`config.Config` を直接エンコードしない（`Repo` は `Name` / `MergeMethod` の構造体なのでマッピングになり、mvp.md の `- org/app` の形にならない）。
`Write` は `os.MkdirAll(filepath.Dir(path), 0o700)` の後に `os.WriteFile(path, data, 0o600)`。認証情報は無いが、利用者の設定なので他人に読ませる理由が無い。

### `Check` の失敗は `cmd/sugi-loop` で 2 行に写す

s03 の `Check` は変えない（`gh-client` の Requirement は触らない）。`cmd/sugi-loop/main.go` に非公開関数 `checkError(err error) error` を置き、`errors.Is(err, exec.ErrNotFound)`（`exec.LookPath` の `*exec.Error` は `Unwrap` を持ち、s03 は `%w` で包んでいる）なら gh 無しの 2 行、`errors.As(err, &ghErr)` で `*gh.Error` なら認証失敗の 2 行、それ以外は `err` をそのまま返す。2 行は `\n` で結んだ 1 つの `error` にして `main()` の `fmt.Fprintln(os.Stderr, err)` で出す。`main()` は変えない。
`gh auth status` の stderr は複数行になることがある（複数ホストの状態を列挙する）。`TrimSpace` して 1 行目の後ろに付け、途中の改行はそのまま出す。文言の検証は `strings.Contains` だけにする。

### 空キューのヒントは `tableLines` の 0 件分岐

`internal/ui/view.go` の `tableLines(h)` で、`len(m.cards) == 0 && !m.fetching && m.errText == ""` のときヒントの 2 行を返す。条件を `cards` で見るのは、全タブ空 ⇔ `Cards` が 0 件だから。`partial`（部分失敗）は条件に入れない（search は成功している）。1 ペインでプレビューを出しているときは `tableLines` を呼ばないので自然に出ない。
縦は `(h - 2) / 2` 行の空行を上に置き、横は各行を `(m.width - 表示幅) / 2` の空白で左詰めする。右には空白を足さない（Lip Gloss の配置関数は右詰めの空白を足すので使わない）。テストは ANSI を除いた行の先頭の空白数と行末に空白が無いことを見る。h が 2 未満なら入る分だけ出す。
文言は mvp.md の 1 文を `。` で 2 行に分け、Markdown のバッククォートを外す（端末に Markdown を出さない）。s09 の詳細画面は `render()` の先頭で分岐しており、この変更は届かない。

### 先行 change との衝突

- `internal/ui`: s09（実装中）が `render()` に画面の分岐と `footer(hint)` を足している。この change は `tableLines` の中だけを変える。s10 が `New(fetcher, client, editor)` に変えるので、テストの `New(nil)` は着手時点の署名に合わせる（tasks 1.1）
- `tui-entrypoint`: この change は「起動失敗」と「1 フレーム描画」の 2 ブロックを MODIFIED する。「1 フレーム描画」の写し元は s10 の現行版（`openspec/changes/s10-answer-question/specs/tui-entrypoint/spec.md`。`New(fetcher, client, editor)` と Scenario「editor が空でも起動する」を含む）で、この change の差分は手順 1 を「存在確認 → フォーム → `Load`」に変えることと、Scenario「設定ファイルが無い端末ではフォームの後にキュー画面が出る」を 1 件足すことの 2 点だけ。**archive の順は s10 → s08a** に固定する。s10 の本文がレビューで変わったら、s08a の着手時に写し直す（tasks 1.1）
- `cmd/sugi-loop/main.go`: s10 が `ui.New` の引数を足す。この change は `run()` の前半（`ensureConfig` / `checkError`）を変える。同じ関数を触るので、後から入る方が手で合わせる

## Risks / Trade-offs

- [huh v2 の API 名が想定と違う] → spec は振る舞いで書いてある。README で確認する。フォームは `Run` 1 関数に閉じ、テストは huh を回さない
- [非端末の判定が CI や `go run` のパイプで意図と違う] → 標準入力だけで判定する。Bubble Tea 自体が端末を要求するので、端末でなければどのみち起動できない
- [`ensureConfig` と `Load` の間でファイルが消える / 壊れる（競合）] → 起こり得ない前提で防御しない。`Load` のエラーで終わる
- [`yaml.v3` のエンコーダの出力形式（インデント・引用）がバージョンで変わる] → テストが mvp.md の例との一致を固定する。変われば気付く
- [`exec.ErrNotFound` の判定が s03 の包み方に依存する] → s03 は `fmt.Errorf("…: %w", err)` で包んでいる（`internal/gh/client.go` の `Check`）。tasks 1.1 で確認する
- [ヒントの条件が `errText` に依存し、s13 の stale 表示で意味が変わる] → s13 が `errText` の扱いを変えるなら、そのときヒントの条件も見直す。この change では取得失敗 = `errText != ""`

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| フォームの中止（huh の既定の中止操作 Ctrl+C）の扱い | 設定ファイルを書かず `設定の作成を中止しました（<パス> は書いていません）` を出して exit 1 | 中止で空の設定を書くと次回も `Load` エラーになり、フォームに戻れない。書かなければ次回また onboarding に入る |
| 標準入力が端末でないときに設定ファイルが無い | `設定ファイルがありません: <パス>` を出して exit 1。フォームは出さない | huh のフォームは端末を要求する。パスを出せば手で書ける |
| `Check` 失敗の文言 | gh 無し: `gh が見つかりません` / `https://cli.github.com/ から GitHub CLI をインストールしてください`。認証失敗: `gh の認証に失敗しました: <stderr>` / `gh auth login を実行してください` | mvp.md は「原因と次の一手を 1 行ずつ」と `gh auth login` だけを決めている |
| ヒントを出す位置 | 表の領域（2 ペインは表の高さ、1 ペインは表を出しているときの残り高さ）の縦横中央 | mvp.md「画面中央」。ヘッダとフッタは残す（`[1]今やる 0` と `q 終了` が `tui-entrypoint` の期待） |
| 部分失敗で 0 件のとき | ヒントを出す | search は成功しており「前回結果の維持」ではない。部分失敗はフッタに出ている |
| ヒントの文言の改行 | mvp.md の 1 文を `。` で 2 行に分け、バッククォートを外す | 端末幅に収まりやすい。Markdown は端末に出さない |
| 書き出すファイルのモード | ディレクトリ 0o700、ファイル 0o600 | 利用者の設定。共有する理由が無い |
| `repos` のプレースホルダ | `org/app\norg/web` | mvp.md の設定例と同じ |
| package の場所 | `internal/onboarding` | Decisions のとおり。D-003 の `ui/ form` は Bubble Tea の画面の意味で、TUI 起動前のフォームは別 |
| `refresh_interval_sec` の値 | 常に `120` を書く（省略しない） | mvp.md の設定例に行があり、利用者が後で編集する手がかりになる |
| gh のインストール先 URL | `https://cli.github.com/` | mvp.md は「`gh` のインストール先 URL」とだけ書き、値を決めていない |
