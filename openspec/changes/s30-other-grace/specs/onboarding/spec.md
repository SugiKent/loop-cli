## MODIFIED Requirements

### Requirement: 回答を mvp.md の形式の YAML で書き出す
`internal/onboarding` は型 `Answers { Repos []string; MergeMethod config.MergeMethod; Notify bool; Editor string }` と、関数 `Marshal(a Answers) ([]byte, error)`、`Write(path string, a Answers) error` を MUST 提供する。
`Marshal` は mvp.md「設定ファイル」の例と同じ書き方で、キーを `repos` / `refresh_interval_sec` / `merge_method` / `editor` / `notify` の順に、`repos` の各要素を `  - owner/name`（2 スペース + `- `）で、`refresh_interval_sec` を常に `120` で書く。コメントは書かない。`editor` の値は入力どおりに書く（`$EDITOR` は `$EDITOR` のまま。`config.Load` が `os.ExpandEnv` で展開する）。mvp.md の例にある `other_grace_min` と `mode` は書かない（フォームで聞かない項目は `config.Load` の既定に任せる。s30 で `other_grace_min` が例に入っても `Marshal` の出力は変わらない）。
`Write` は `path` の親ディレクトリが無ければ作り（`os.MkdirAll`、0o700）、`Marshal` の結果を 0o600 で書く。既にファイルがあっても上書きしない責務は持たない（呼び出し側の存在確認が「無い」ときだけ呼ぶ）。
書き出した内容は必ず s02 の `config.Load` で読めなければならない。`Marshal` の出力を `config.Load` に通した `Config` は、`Repos[].Name` が `Answers.Repos` と同じ順、`Repos[].MergeMethod` と `MergeMethod` が `Answers.MergeMethod`、`RefreshIntervalSec` が 120、`Notify` が `Answers.Notify`、`Editor` が `Answers.Editor` を展開した値、`OtherGraceMin` が既定の 30 になる。

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
- **THEN** `Write` はエラー無しで返り、ファイルが存在し、`Load` の `Config` は `Repos` が `[{org/app rebase}]`、`MergeMethod` が `rebase`、`RefreshIntervalSec` が 120、`Notify` が false、`Editor` が `nvim`、`OtherGraceMin` が 30 である

#### Scenario: editor を空にしても読める
- **WHEN** `Answers{Repos: []string{"org/app"}, MergeMethod: "squash", Notify: true, Editor: ""}` を `Marshal` して `config.Load` で読む
- **THEN** `Load` はエラー無しで返り、`Editor` は空文字列である
