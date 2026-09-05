## ADDED Requirements

### Requirement: リポジトリ名の検証規則を公開する
`internal/config` は関数 `IsRepoName(name string) bool` を MUST 提供する。`owner/name` の形式（`/` がちょうど 1 つ、`owner` と `name` が空でない、空白とタブを含まない）なら true。`Load` の検証（「不正な設定はパスと原因を含むエラーになる」）はこの関数を使い、onboarding のフォームも同じ関数で各行を検証する。規則を 2 か所に書かない。

#### Scenario: owner/name 形式を受け入れる
- **WHEN** `IsRepoName("org/app")` を呼ぶ
- **THEN** true が返る

#### Scenario: owner/name でない文字列を拒否する
- **WHEN** `IsRepoName("app")`、`IsRepoName("org/app/extra")`、`IsRepoName("/app")`、`IsRepoName("org/")`、`IsRepoName("org/ app")` をそれぞれ呼ぶ
- **THEN** いずれも false が返る
