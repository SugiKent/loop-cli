## Context

s01-bootstrap で Go モジュール（`github.com/SugiKent/sugi-loop`）と `cmd/sugi-loop` の hello world ができている。`internal/*` はまだ無い。
この change は D-003 の内部構成にある `internal/config/`（config.yml 読み込み）を作る。正本は mvp.md「設定ファイル」節の YAML 例で、項目は
`repos` / `refresh_interval_sec` / `merge_method`（リポジトリ別上書き可）/ `editor` / `notify` の 5 つ。認証トークンは持たない。

使い捨てモジュールで確認した事実（Go 1.26.6 / darwin）:
- `os.UserConfigDir()` は darwin で `~/Library/Application Support` を返す。mvp.md の `~/.config/sugi-loop/config.yml` と一致しないので使わない
- 既定値を入れた struct に対して YAML を Unmarshal すると、YAML に無いフィールドは既定値のまま残る（`go.yaml.in/yaml/v3` v3.0.5）。
  `notify: false` の明示も false として読める。既定値関数や `*bool` は要らない
- 未知のフィールドを拒否するデコーダ設定があり、`token: abc` は `field token not found` のエラーになる
- 未知フィールド拒否の設定は、カスタムデコード内でノード単位にデコードする際には引き継がれない。`{name: org/web, merge_methd: rebase}` は
  エラーにならず `merge_method` が空のまま通る。`repos` 要素のマッピングは自前でキーを検査する必要がある
- `HOME` が空の状態で `os.UserHomeDir()` を呼ぶと `$HOME is not defined` のエラーが返る
- 空ファイルをデコードすると `io.EOF` が返る。これはエラーではなく「何も書かれていない」として扱い、後段の `repos` 空チェックに流す必要がある

## Goals / Non-Goals

**Goals:**
- `internal/config` に `DefaultPath` と `Load` を置き、検証済みの `Config` を返す
- 既定値（120 / squash / true / `$EDITOR`）と検証（ファイル無し / 構文 / repos / owner-name 形式 / merge_method / refresh 下限 / 未知キー）
- リポジトリ別 `merge_method` を `Load` 時に解決し、消費側が上書きを意識しなくてよい形で返す

**Non-Goals:**
- `cmd/sugi-loop` から `Load` を呼ぶ配線。設定値を最初に消費する s07 が担当し、エラーは s01 の「起動失敗は標準エラーに出て終了コード 1 になる」経由で表面化する。ここでは定義しない
- 設定ファイルの新規作成・雛形出力・`init` サブコマンド（docs に無い）
- ラベル名・routine マーカーの設定化（mvp.md が明示的に禁止）
- 設定の再読み込み（ホットリロード）

## Decisions

### ファイル構成と API

```
internal/config/config.go       # Config / Repo / MergeMethod 型、DefaultPath、Load
internal/config/config_test.go  # t.TempDir() に YAML を書いて Load を検証
```

- `type MergeMethod string`。定数 `MergeSquash = "squash"` / `MergeMerge = "merge"` / `MergeRebase = "rebase"`。値は `gh pr merge --squash` / `--merge` / `--rebase` のフラグ名と同じにし、s14 がそのまま `--<method>` に使えるようにする
- `type Repo struct { Name string; MergeMethod MergeMethod }`。`MergeMethod` は `Load` が必ず埋める（上書きが無ければグローバル値）
- `type Config struct { Repos []Repo; RefreshIntervalSec int; MergeMethod MergeMethod; Editor string; Notify bool }`。この 5 フィールド以外は持たない
- `func DefaultPath() (string, error)`: `os.UserHomeDir()` に `.config/sugi-loop/config.yml` を結合して返す。`UserHomeDir` のエラーはそのまま返す
- `func Load(path string) (*Config, error)`: ファイル読み込み → デコード → 既定値適用 → 検証 → リポジトリ別 merge_method 解決 の順。エラーはすべて `fmt.Errorf("config %s: ...", path)` の形でパスを含める。`Load` 内部で `DefaultPath` を呼ばない（テストが一時ファイルを渡せるようにする）

`Load()` 引数無し版・環境変数でのパス上書き・複数パスの探索は作らない（呼び出し側は `DefaultPath()` の結果を渡す 1 行で済む）。

### 既定値の適用方法

`Config{RefreshIntervalSec: 120, MergeMethod: MergeSquash, Editor: "$EDITOR", Notify: true}` を先に作り、その変数へデコードする。
Context の確認どおり YAML に無い項目は既定値のまま残り、`notify: false` は false になる。既定値を別の関数やテーブルに切り出さない。

### editor の環境変数展開

デコード後に `Editor` へ `os.ExpandEnv` 相当の展開を 1 回かける。mvp.md の例 `editor: $EDITOR` がそのまま動き、省略時の既定値 `$EDITOR` も同じ経路で
環境変数 `EDITOR` の値になる。展開結果が空でも `Load` は失敗させない。read-only の操作（キュー閲覧・merge）はエディタ無しで成り立ち、
`a` を押したときに初めて必要になるためで、空のときの扱いは s10 が決める。

### リポジトリ別 merge_method の YAML 表現

`repos` の各要素を「文字列」または「`name` と `merge_method` を持つマッピング」のどちらでも書けるようにする（詳細は未決事項）。

```yaml
repos:
  - org/app                       # グローバルの merge_method を使う
  - name: org/web
    merge_method: rebase          # このリポジトリだけ rebase
merge_method: squash
```

`Repo` に YAML のカスタムデコード（スカラーならそれを `Name` に、マッピングなら `name` / `merge_method` を読む）を 1 つ実装する。
マッピングの場合は、Context の確認どおり未知フィールド拒否が引き継がれないので、マッピングノードのキーを走査して `name` / `merge_method`
以外があれば `repos の要素に未知のキー "<キー>" があります` のエラーを返す。
デコード直後の `Repo.MergeMethod` は「指定があればその値、無ければ空」なので、検証後に空のものへグローバル値を埋める。

### 検証の順序とエラー文言

デコード後、次の順で検証し最初のエラーを返す。エラー文言には項目名と不正だった値を含める（spec の Scenario が文字列包含で検証する）。

1. `repos` が空（`len == 0`）→ `repos が空です`
2. 各 `repos[i].Name` の形式 → `repos[i] "<値>": owner/name 形式ではありません`。判定は `strings.Count(name, "/") == 1`、`/` の両側が非空、`strings.ContainsAny(name, " \t")` が false の 3 点だけ。GitHub の名前文字種の正規表現は書かない
3. グローバル `merge_method` → `merge_method "<値>": squash / merge / rebase のいずれかを指定してください`
4. 各 `repos[i].MergeMethod`（空以外）→ `repos[i] "<name>" の merge_method "<値>": ...`（同上）
5. `refresh_interval_sec < 1` → `refresh_interval_sec は 1 以上を指定してください`

トップレベルの未知キーと YAML 構文エラーはデコーダが返すエラーをそのままパス付きで包む。`repos` 要素内の未知キーは上記のカスタムデコードが返す。ファイル無しは `os.ReadFile` のエラーをパス付きで包む。

### YAML ライブラリ

`go.yaml.in/yaml/v3` を使う（未決事項）。`gopkg.in/yaml.v3` と同じ API で、未知フィールド拒否とノード単位のカスタムデコードがある。
`go get` で go.mod に追加する。s01 の design で述べたとおり `go mod tidy` を打つと未 import の Bubbles / Glamour / huh が go.mod から落ちるので、
追加後は `go list -m all` で 5 依存が残っていることを確認する。

## Risks / Trade-offs

- [`go mod tidy` で s01 が固定した 3 依存が落ちる] → tasks で `go list -m all` を確認し、落ちていたら `go get charm.land/<name>/v2@latest` で戻す
- [`Repo` のカスタムデコードが YAML ライブラリの API に依存する] → spec は振る舞い（文字列またはマッピング）で書いてあり、ライブラリを替えても spec は変わらない
- [`Editor` の環境変数展開で `$HOME/bin/editor` のような値も展開される] → 意図どおり。展開したくない `$` を書く用途は docs に無い
- [`os.ExpandEnv` は未定義変数を空にする] → 未設定の `EDITOR` は空文字列になり `Load` は成功する。spec の Scenario で明示している

## 未決事項

docs に記述が無く、この change の実装者が選ぶ点。既定値を 1 つずつ示す。

| 項目 | 既定値 | 根拠 |
| --- | --- | --- |
| YAML ライブラリ | `go.yaml.in/yaml/v3`（v3.0.5 時点） | `gopkg.in/yaml.v3` の後継で API 互換。未知フィールド拒否とカスタムデコードがあり、Context の確認で必要な挙動がすべて揃った |
| 設定ファイルパスの解決 | `$HOME/.config/sugi-loop/config.yml` 固定。`XDG_CONFIG_HOME` は見ない | mvp.md の記述をそのまま採る。`os.UserConfigDir()` は darwin で別の場所を返すので使えない。XDG 対応が必要になったら `DefaultPath` だけ直せばよい |
| リポジトリ別 `merge_method` の YAML 表現 | `repos` 要素を文字列または `{name, merge_method}` マッピングのどちらでも書ける | mvp.md の例（文字列の列挙）をそのまま有効にしつつ、上書きを同じ要素に書ける。代替案の別キー（`merge_method_overrides: {org/web: rebase}`）は `repos` に無いリポジトリを書けてしまい検証が増える |
| `editor: $EDITOR` の解釈 | 値に環境変数展開をかける。省略時の既定値も `$EDITOR` | mvp.md の例が YAML 上に `$EDITOR` と書いているため、展開しないと動かない |
| `editor` が空のときの扱い | `Load` は成功させ、空を `Editor` に入れる。判断は s10 | 閲覧・merge はエディタ無しで成立する。起動を止める理由が無い |
| 未知のキー（トップレベル・`repos` 要素内とも） | エラーにする | `refresh_interval:` のようなキーの打ち間違いを黙って既定値で動かすと、利用者が気づけない。`token` を書けないことの保証にもなる |
| `refresh_interval_sec` の下限 | 1 以上。0 以下はエラー | 0 や負の間隔で自動更新（s13）を回せない。上限は設けない |
| `repos` の重複 | 検証しない | docs に無い。重複しても取得結果が重複するだけで壊れない。必要になったら足す |
