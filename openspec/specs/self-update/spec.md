# self-update Specification

## Purpose
TBD - created by archiving change s23-self-update. Update Purpose after archive.
## Requirements
### Requirement: 現在の版とモジュールパスは build info から取る

`internal/version` は、実行中のバイナリのモジュールパス・版・手元 build かどうかを返す関数を MUST 公開する。値は `runtime/debug.ReadBuildInfo()` から取り、ネットワークもファイルも読まない。

- 版は `Main.Version`。空文字列なら `(devel)` とする
- 手元 build かどうかは、`Settings` に `vcs.revision` があるか、版が `(devel)` かで決める。Go はリポジトリの作業ツリーで `go build` したバイナリにも VCS 由来の擬似バージョン（末尾 `+dirty`）を刻むため、版の文字列だけでは `go install <module>@<version>` と区別できない。`vcs.*` の設定は module cache から入れたバイナリには付かない
- `ReadBuildInfo` が読めない場合、モジュールパスは空文字列、版は `(devel)`、手元 build とする
- 呼び出し側はモジュールパスをこの関数からのみ得る。Go のコードにモジュールパスの定数を書かない（`go.mod` と import パスを除く）

#### Scenario: go install で入れたバイナリ

- **WHEN** `go install <module>/cmd/loop-cli@latest` で入れたバイナリで現在の版を取る
- **THEN** モジュールパスは `go.mod` の `module` の値と等しく、版は `v` で始まり（タグまたは擬似バージョン）、手元 build ではない

#### Scenario: リポジトリで go build したバイナリは手元 build になる

- **WHEN** 作業ツリーで `go build ./cmd/loop-cli` して作ったバイナリで現在の版を取る
- **THEN** 手元 build であり、版は `(devel)` または VCS 由来の擬似バージョンである

### Requirement: 最新の版は go list -m -json で取る

`internal/version` は、モジュールパスを受け取って最新の版の文字列を返す関数を MUST 公開する。`go list -m -json <module>@latest` を実行し、標準出力の JSON の `Version` を返す。

- `go` の呼び出しは差し替え可能な関数を通して行い、テストは本物の `go` とネットワークを使わない
- `go` は module の外（`os.TempDir()`）で走らせる。vendor ディレクトリを持つプロジェクトの中では `-mod=vendor` が自動で効き、`@latest` を問い合わせられないため
- コマンドが失敗したら、コマンドと標準エラーを含むエラーを返す。`go` が PATH に無い場合は `exec.ErrNotFound` を包んだエラーになる
- 標準出力が JSON として読めない、または `Version` が空なら、その出力を含むエラーを返す
- 呼び出し側が渡した `context.Context` で中断できる

#### Scenario: vendor ディレクトリを持つプロジェクトの中でも問い合わせられる

- **WHEN** `vendor/` を持つ Go プロジェクトのディレクトリで `update` を実行する
- **THEN** `cannot query module due to -mod=vendor` にはならず、最新の版の問い合わせが行われる

#### Scenario: 最新の版が返る

- **WHEN** 差し替えた実行関数が `{"Path":"github.com/SugiKent/loop-cli","Version":"v0.0.0-20260905143706-3ce8a526b4ec"}` を返す状態で最新の版を取る
- **THEN** 実行されたコマンドは `go list -m -json github.com/SugiKent/loop-cli@latest` で、戻り値は `v0.0.0-20260905143706-3ce8a526b4ec` である

#### Scenario: go の失敗はエラーになる

- **WHEN** 差し替えた実行関数が標準エラー `module lookup disabled` とともに失敗する状態で最新の版を取る
- **THEN** エラーに `go list` と `module lookup disabled` が含まれる

#### Scenario: JSON として読めない出力

- **WHEN** 差し替えた実行関数が `not json` を返す
- **THEN** エラーに `not json` が含まれる

### Requirement: version サブコマンドは現在の版を 1 行出す

`loop-cli version` は、標準出力に `loop-cli <現在の版>` の 1 行を MUST 書き、終了コード 0 で終わる。ネットワークを使わず、最新の版は調べない。

#### Scenario: 版が出る

- **WHEN** 引数 `version` で実行する
- **THEN** 標準出力は `loop-cli <現在の版>` の 1 行で、終了コードは 0 である（手元 build なら擬似バージョンか `(devel)` がそのまま出る）

### Requirement: update は新しい版があるときだけ go install で入れ直す

`loop-cli update` は次の順で MUST 動く。

1. 現在の版とモジュールパスを取る
2. 最新の版を取る。失敗したらそのエラーを標準エラーに書き、終了コード 1 で終わる（`go` が PATH に無い場合は `go が見つかりません` と `https://go.dev/dl/` の 2 行を書く）
3. 手元 build ではなく、現在の版が最新の版と文字列として等しければ、標準出力に `最新版です: <版>` と書き、`go install` を実行せず終了コード 0 で終わる
4. それ以外は `go install <モジュールパス>/cmd/loop-cli@latest` を実行する。`go` の標準出力と標準エラーはそのまま流す。失敗したらそのエラーを標準エラーに書き、終了コード 1 で終わる
5. 成功したら標準出力に `更新しました: <現在の版> → <最新の版>` と書き、終了コード 0 で終わる

版の比較は文字列の一致だけで行い、semver の大小比較は行わない（design.md）。

#### Scenario: 新しい版があれば install する

- **WHEN** 現在の版が `v0.0.0-20260901000000-aaaaaaaaaaaa`、最新の版が `v0.0.0-20260905143706-3ce8a526b4ec` の状態で `update` を実行する
- **THEN** `go install github.com/SugiKent/loop-cli/cmd/loop-cli@latest` が実行され、標準出力に `更新しました: v0.0.0-20260901000000-aaaaaaaaaaaa → v0.0.0-20260905143706-3ce8a526b4ec` が出て、終了コードは 0 である

#### Scenario: 同じ版なら install しない

- **WHEN** 現在の版と最新の版が等しい状態で `update` を実行する
- **THEN** `go install` は実行されず、標準出力に `最新版です: <その版>` が出て、終了コードは 0 である

#### Scenario: 手元 build は比較せず install する

- **WHEN** 現在の版が手元 build のものである状態で `update` を実行する
- **THEN** 最新の版と等しいかによらず `go install` が実行され、終了コードは 0 である

#### Scenario: go install の失敗

- **WHEN** `go install` が標準エラー `permission denied` とともに失敗する状態で `update` を実行する
- **THEN** 標準エラーに `permission denied` が出て、終了コードは 1 である

#### Scenario: go が無い

- **WHEN** `go` が PATH に無い状態で `update` を実行する
- **THEN** 標準エラーに `go が見つかりません` と `https://go.dev/dl/` が出て、終了コードは 1 である

### Requirement: TUI は起動時に 1 度だけ更新を調べ、あればヘッダに出す

`ui.Options` は更新の確認を行う関数の欄を MUST 持ち、`nil` なら確認を行わない。関数は `context.Context` を受け取り、新しい版があるかどうかだけを返す（ヘッダに版を出さないので版は使わない）。`Model` は `Init` が返すコマンドで、取得のコマンドと並べてこの確認を 1 度だけ MUST 開始する。自動更新の tick（s13）や `R`（s12）では再度行わない。

結果のメッセージを受け取ったら、新しい版があるときだけ `Model` に印を持ち、キュー画面のヘッダに `↑ update` を出す（Requirement「ヘッダはタブ名と件数と最終更新時刻、フッタはキーヒントとステータスを出す」）。確認が失敗した場合と、新しい版が無い場合は、画面に何も出さず、フッタのステータス（`errText` 等）も変えない。確認は画面の他の動きを止めない。

`cmd/loop-cli` が渡す関数は、手元 build なら何も調べずに「新しい版は無い」を返し、そうでなければ 10 秒の時間制限を付けて最新の版を取り、現在の版と違えば「新しい版がある」を返す（design.md 未決事項の既定値）。

#### Scenario: 新しい版があるとヘッダに出る

- **WHEN** 「新しい版がある」を返す確認関数を `Options` に渡した `Model` に、その結果のメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** 1 行目に `↑ update` が含まれる

#### Scenario: 新しい版が無ければ何も出ない

- **WHEN** 「新しい版は無い」を返す確認関数の結果のメッセージを与えた `Model` の `View` から ANSI エスケープを除いて読む
- **THEN** どの行にも `↑ update` は含まれない

#### Scenario: 確認の失敗は画面を変えない

- **WHEN** 確認関数が失敗を「新しい版は無い」に畳んだ結果のメッセージを与えた `Model` の `View` から ANSI エスケープを除いて読む
- **THEN** 1 行目に `↑ update` は含まれず、最終行にエラーのステータスも出ない

#### Scenario: 確認しない設定では確認関数を呼ばない

- **WHEN** `Options` の確認の関数が `nil` の `Model` の `Init` を呼ぶ
- **THEN** 確認は行われず、取得のコマンドだけが返る

#### Scenario: R では再確認しない

- **WHEN** 確認関数を渡した `Model` で `Init` の後に `R` を押す
- **THEN** 確認関数が呼ばれた回数は 1 のままである

