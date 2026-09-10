## MODIFIED Requirements

### Requirement: fixture capture は指定リポジトリの fixture を採取して保存する
`loop-cli-dev fixture capture --repo <owner/name> --alias <alias>` は、稼働中リポジトリ 1 件の open issue / open PR と、その全件の詳細と、そのリポジトリで使えるラベルの一覧を採取し、伏せ字にして `internal/gh/testdata/fixtures/<alias>/` に s03 `gh-fake` の命名規則で MUST 保存する。
- `--repo` は `owner/name` 形式（`/` で 2 つに分かれ、どちらも空でない）でなければならない。`--alias` は `^[a-z0-9-]+$` に一致し、`example`（s03 の手書き fixture）以外で、かつ `owner` / `name` のどちらも（大文字小文字を区別せず）部分文字列として含んではならない（alias は `name` の置換先なので、含むと元の名前が fixture に残り、次の Requirement の検査 3 が落ちる）
- 保存先はカレントディレクトリからの相対パス `internal/gh/testdata/fixtures/<alias>`。`internal/gh/testdata/fixtures` が存在しなければ、リポジトリのルートで実行するよう促すエラーを返す。`<alias>` ディレクトリが既にあれば中身を消してから書く（再採取で古い issue のファイルが残らないようにする）
- サブコマンドは、s03 `Client.Check`、`Client.Capture`（`progress` で受け取ったファイル名を 1 行ずつ標準エラーに出す）、伏せ字（次の Requirement）、書き込み、自己検査、要約をこの順に実行する。`Check` と `Capture` と伏せ字が失敗したときは、そのエラーをそのまま返し、ファイルを書かない
- `Capture` は `labels.json`（`ListLabels` の標準出力）も採る（採取の順序は s03 `gh-client`「Client は fixture 用に読み取りコマンドの標準出力を採取する」が定める）。`labels.json` のラベル名は `"name": "…"` の形なので、次の Requirement の衝突ガードの (a)（`login` / `name` 以外のキーの値）には当たらない。固定 12 語（`todo` / `propose` / … ）に一致するラベル名だけがガードに掛かり、それ以外のプロジェクト固有ラベルがリポジトリ名や login と同じ文字列だと、伏せ字がラベル名まで書き換える
- 自己検査: 書き込んだ全ファイルを、s03 の `Fake`（`NewFake(<保存先>)`）で読み直し、`SearchIssues` / `SearchPRs` / `ListLabels` と、各 issue の `ViewIssue` / `CrossReferencedPRs` / `LabelTimeline`、各 PR の `ViewPR` / `ViewPRMergeState` / `ReviewThreads` がすべてエラー無く返ることを確認する。1 つでも失敗したら保存先ディレクトリを削除してエラーを返す
- 成功したら標準出力に、保存先・書いたファイル数・issue 数・PR 数・伏せた login 数を 1 行で出す

#### Scenario: 採取して保存する
- **WHEN** `gh` が使える状態で、open issue 2 件（108, 140）と open PR 1 件（131）のリポジトリ `owner/name` に対し `fixture capture --repo owner/name --alias app` を実行する
- **THEN** `internal/gh/testdata/fixtures/app/` に `search-issues.json` / `search-prs.json` / `labels.json` / `issue-108.json` / `issue-108-cross-refs.json` / `issue-108-timeline.json` / `issue-140.json` / `issue-140-cross-refs.json` / `issue-140-timeline.json` / `pr-131.json` / `pr-131-review-threads.json` の 11 ファイルができ、標準出力に `11` と `app` を含む要約が出て、終了コードは 0 である

#### Scenario: alias に example は使えない
- **WHEN** `fixture capture --repo owner/name --alias example` を実行する
- **THEN** `gh` を実行せずにエラーが返り、エラー文字列に `example` を含み、終了コードは 1 である

#### Scenario: repo の形式が不正
- **WHEN** `fixture capture --repo name --alias app` を実行する
- **THEN** `gh` を実行せずにエラーが返り、エラー文字列に `owner/name` を含み、終了コードは 1 である

#### Scenario: alias に owner や name を含められない
- **WHEN** `fixture capture --repo acme/widgets --alias my-widgets` を実行する
- **THEN** `gh` を実行せずにエラーが返り、エラー文字列に `widgets` を含み、終了コードは 1 である（`--alias acme-app` も同様に `acme` を含むエラー。大文字小文字は区別しない）

#### Scenario: リポジトリのルート以外で実行する
- **WHEN** `internal/gh/testdata/fixtures` が無いディレクトリで実行する
- **THEN** `gh` を実行せずにエラーが返り、エラー文字列に `internal/gh/testdata/fixtures` を含む

#### Scenario: 自己検査に失敗したらディレクトリを残さない
- **WHEN** 書き込んだファイルのいずれかを `Fake` が読めない
- **THEN** `internal/gh/testdata/fixtures/<alias>` は削除され、エラーが返る
