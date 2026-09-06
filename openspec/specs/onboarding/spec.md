# onboarding Specification

## Purpose
TBD - created by archiving change s08a-onboarding. Update Purpose after archive.
## Requirements
### Requirement: 設定ファイルが無いときだけ onboarding に入る
`cmd/loop-cli` は `config.Load` を呼ぶ前に、`config.DefaultPath()` のパスにファイルが存在するかを MUST 確認する。判定順は 存在確認 → （無ければ）onboarding フォーム → 書き出し → `config.Load` → `gh.Client.Check` → TUI である（`Check` 以降は `tui-entrypoint`「loop-cli バイナリが起動して 1 フレーム描画する」の手順 2 以降と同じ）。
- ファイルが存在する: onboarding に入らず `config.Load` へ進む。壊れた設定（YAML エラー・検証エラー）は `config.Load` のエラーで終了し、ファイルを上書きしない（mvp.md「設定ファイルが壊れているときは onboarding に入らず、従来どおり原因を出して終了する」）
- ファイルが存在しない（`os.Stat` が「存在しない」を返す）: 標準入力が端末なら onboarding フォームを出し、書き出しが成功したら同じパスを `config.Load` で読む。`Config` を自前で組み立てない
- 存在確認がそれ以外のエラーを返す（権限不足など）: そのエラーをパス付きで標準エラーに出して終了コード 1
- 標準入力が端末でない: フォームを出さず、`設定ファイルがありません: <パス>` を標準エラーに出して終了コード 1（`tui-entrypoint`「起動失敗は標準エラーに出て終了コード 1 になる」）
この判定は `cmd/loop-cli` の非公開関数（設定ファイルのパス・端末かどうかの真偽値・フォームを起動する関数を引数に取る）に置き、テストではフォームを起動する関数を呼ばれた回数を数えるスタブに差し替える。

#### Scenario: 設定ファイルがあればフォームに入らない
- **WHEN** 一時ディレクトリに `repos:\n  - org/app\n` の設定ファイルを置き、そのパス・端末 true・呼ばれた回数を数えるスタブを判定関数に渡す
- **THEN** エラー無しで返り、スタブは 0 回呼ばれる

#### Scenario: 壊れた設定はフォームに入らず上書きされない
- **WHEN** 一時ディレクトリに `repos: [org/app\n` の設定ファイルを置いて判定関数を通した後、そのパスを `config.Load` で読む
- **THEN** 判定関数はエラー無しで返りスタブは 0 回呼ばれ、`config.Load` はパスを含むエラーを返し、ファイルの内容は `repos: [org/app\n` のままである

#### Scenario: 端末で設定ファイルが無ければフォームに入る
- **WHEN** 一時ディレクトリの存在しないパス・端末 true・スタブを判定関数に渡す
- **THEN** スタブがそのパスで 1 回呼ばれ、判定関数はスタブの返り値をそのまま返す

#### Scenario: 端末でなければフォームに入らず exit 1 相当のエラー
- **WHEN** 一時ディレクトリの存在しないパス・端末 false・スタブを判定関数に渡す
- **THEN** スタブは 0 回呼ばれ、`設定ファイルがありません` とそのパスを含むエラーが返る

### Requirement: フォームは repos / merge_method / notify / editor を聞き、repos を owner/name 規則で検証する
`internal/onboarding` は huh のフォームを標準入出力で実行する関数 `Run(path string) error` を MUST 提供する。フォームの項目と順は次のとおりで、`refresh_interval_sec` は聞かない（既定 120。mvp.md「初回起動（onboarding）」）。
1. `repos`: 複数行入力。1 行に 1 リポジトリを `owner/name` で書く。入力の検証は公開関数 `ParseRepos(text string) ([]string, error)` で行い、各行を `strings.TrimSpace` して空行を捨て、残りが 0 行ならエラー `repos を 1 件以上入力してください`、`config.IsRepoName` が false の行があればエラー `<その行>: owner/name 形式ではありません`（s02 `config-loading`「不正な設定はパスと原因を含むエラーになる」と同じ規則）。検証を通るまでフォームは次に進まない
2. `merge_method`: `squash` / `merge` / `rebase` の 3 択の select。既定 `squash`
3. `notify`: confirm。既定 true
4. `editor`: 1 行の input。既定は文字列 `$EDITOR`（展開しない。`config.Load` が読むときに展開する）。空にしたら空のまま書く
`Run` はフォームの完了後、回答を `Answers` にして Requirement「回答を mvp.md の形式の YAML で書き出す」の関数で `path` に書く。フォームが huh の既定の中止操作（Ctrl+C）で終わったら、ファイルを書かずにエラー値 `ErrAborted` を返す。huh の API 名は断定しない。README で確認して上記の振る舞いに合わせる。

#### Scenario: 改行区切りの repos を解析する
- **WHEN** `ParseRepos("org/app\n\n  org/web  \n")` を呼ぶ
- **THEN** `[]string{"org/app", "org/web"}` とエラー nil が返る

#### Scenario: 空の repos は拒否する
- **WHEN** `ParseRepos("\n  \n")` を呼ぶ
- **THEN** エラーが返り、文言に `repos` を含む

#### Scenario: owner/name でない行は拒否する
- **WHEN** `ParseRepos("org/app\napp\n")`、`ParseRepos("org/app/extra")`、`ParseRepos("org/ app")` をそれぞれ呼ぶ
- **THEN** いずれもエラーが返り、文言にそれぞれ `app`、`org/app/extra`、`org/ app` と `owner/name` を含む

### Requirement: 回答を mvp.md の形式の YAML で書き出す
`internal/onboarding` は型 `Answers { Repos []string; MergeMethod config.MergeMethod; Notify bool; Editor string }` と、関数 `Marshal(a Answers) ([]byte, error)`、`Write(path string, a Answers) error` を MUST 提供する。
`Marshal` は mvp.md「設定ファイル」の例と同じ形式で、キーを `repos` / `refresh_interval_sec` / `merge_method` / `editor` / `notify` の順に、`repos` の各要素を `  - owner/name`（2 スペース + `- `）で、`refresh_interval_sec` を常に `120` で書く。コメントは書かない。`editor` の値は入力どおりに書く（`$EDITOR` は `$EDITOR` のまま。`config.Load` が `os.ExpandEnv` で展開する）。
`Write` は `path` の親ディレクトリが無ければ作り（`os.MkdirAll`、0o700）、`Marshal` の結果を 0o600 で書く。既にファイルがあっても上書きしない責務は持たない（呼び出し側の存在確認が「無い」ときだけ呼ぶ）。
書き出した内容は必ず s02 の `config.Load` で読めなければならない。`Marshal` の出力を `config.Load` に通した `Config` は、`Repos[].Name` が `Answers.Repos` と同じ順、`Repos[].MergeMethod` と `MergeMethod` が `Answers.MergeMethod`、`RefreshIntervalSec` が 120、`Notify` が `Answers.Notify`、`Editor` が `Answers.Editor` を展開した値になる。

#### Scenario: mvp.md の設定例と同じ YAML になる
- **WHEN** `Answers{Repos: []string{"org/app", "org/web"}, MergeMethod: "squash", Notify: true, Editor: "$EDITOR"}` を `Marshal` する
- **THEN** 出力は次の文字列（末尾に改行 1 つ。行頭の空白は `repos` の要素の 2 スペースだけ）とバイト単位で等しい
  ```
  repos:
    - org/app
    - org/web
  refresh_interval_sec: 120
  merge_method: squash
  editor: $EDITOR
  notify: true
  ```

#### Scenario: 書き出した YAML を config.Load が読める
- **WHEN** 環境変数 `EDITOR` を `nvim` にし、`Answers{Repos: []string{"org/app"}, MergeMethod: "rebase", Notify: false, Editor: "$EDITOR"}` を一時ディレクトリの `loop-cli/config.yml`（`loop-cli` ディレクトリは事前に無い）へ `Write` し、同じパスを `config.Load` で読む
- **THEN** `Write` はエラー無しで返り、ファイルが存在し、`Load` の `Config` は `Repos` が `[{org/app rebase}]`、`MergeMethod` が `rebase`、`RefreshIntervalSec` が 120、`Notify` が false、`Editor` が `nvim` である

#### Scenario: editor を空にしても読める
- **WHEN** `Answers{Repos: []string{"org/app"}, MergeMethod: "squash", Notify: true, Editor: ""}` を `Marshal` して `config.Load` で読む
- **THEN** `Load` はエラー無しで返り、`Editor` は空文字列である

### Requirement: 中止したときは設定ファイルを書かない
`Run` が huh の既定の中止操作（Ctrl+C）で `ErrAborted` を返したとき、`path` にファイルは MUST 存在しない。`cmd/loop-cli` は `ErrAborted` を受けたら `設定の作成を中止しました（<パス> は書いていません）` を標準エラーに出して終了コード 1 で終わる（docs に記述が無い点。design.md の未決事項の既定値）。ファイルが書かれないことは、フォームを起動しない単体テストでは常に真になるので、tasks 6.1 の手動確認（フォームで Ctrl+C）で見る。

#### Scenario: 中止のエラーは設定ファイルを残さない
- **WHEN** フォームを起動する関数が `ErrAborted` を返すスタブの状態で、存在しないパスに対して `cmd/loop-cli` の判定関数を呼び、返ったエラーを起動失敗の文言にする
- **THEN** 文言に `設定の作成を中止しました` とそのパスを含む

