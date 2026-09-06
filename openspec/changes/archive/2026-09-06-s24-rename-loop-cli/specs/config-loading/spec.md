## MODIFIED Requirements

### Requirement: 設定ファイルのパスが決まる
`internal/config` は設定ファイルの既定パスを返す関数 `DefaultPath` を MUST 提供する。返すパスは design.md の未決事項で定めた既定値
（ユーザーのホームディレクトリ直下の `.config/loop-cli/config.yml`）である。ホームディレクトリが決定できない場合はエラーを返す。

#### Scenario: 既定パスがホーム直下の .config になる
- **WHEN** `HOME` が `/Users/alice` の状態で `DefaultPath` を呼ぶ
- **THEN** `/Users/alice/.config/loop-cli/config.yml` が返る

#### Scenario: ホームディレクトリが決まらない
- **WHEN** ホームディレクトリを決定できない環境で `DefaultPath` を呼ぶ
- **THEN** エラーが返り、パスは返らない
