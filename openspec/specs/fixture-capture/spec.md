# fixture-capture Specification

## Purpose
TBD - created by archiving change s04-fixture-capture. Update Purpose after archive.
## Requirements
### Requirement: fixture capture は指定リポジトリの fixture を採取して保存する
`sugi-loop-cli fixture capture --repo <owner/name> --alias <alias>` は、稼働中リポジトリ 1 件の open issue / open PR と分類に必要な詳細を採取し、伏せ字にして `internal/gh/testdata/fixtures/<alias>/` に s03 `gh-fake` の命名規則で MUST 保存する。
- `--repo` は `owner/name` 形式（`/` で 2 つに分かれ、どちらも空でない）でなければならない。`--alias` は `^[a-z0-9-]+$` に一致し、`example`（s03 の手書き fixture）以外で、かつ `owner` / `name` のどちらも（大文字小文字を区別せず）部分文字列として含んではならない（alias は `name` の置換先なので、含むと元の名前が fixture に残り、次の Requirement の検査 3 が落ちる）。違反は `gh` を実行する前のエラー
- 保存先はカレントディレクトリからの相対パス `internal/gh/testdata/fixtures/<alias>`。`internal/gh/testdata/fixtures` が存在しなければ、リポジトリのルートで実行するよう促すエラーを返す。`<alias>` ディレクトリが既にあれば中身を消してから書く（再採取で古い issue のファイルが残らないようにする）
- サブコマンドは、s03 `Client.Check`、`Client.Capture`（`progress` で受け取ったファイル名を 1 行ずつ標準エラーに出す）、伏せ字（次の Requirement）、書き込み、自己検査、要約をこの順に実行する。`Check` と `Capture` と伏せ字が失敗したときは、そのエラーをそのまま返し、ファイルを書かない
- 自己検査: 書き込んだ全ファイルを、s03 の `Fake`（`NewFake(<保存先>)`）で読み直し、`SearchIssues` / `SearchPRs` と、各 issue の `ViewIssue` / `CrossReferencedPRs` / `LabelTimeline`、各 PR の `ViewPR` / `ViewPRMergeState` / `ReviewThreads` がすべてエラー無く返ることを確認する。1 つでも失敗したら保存先ディレクトリを削除してエラーを返す
- 成功したら標準出力に、保存先・書いたファイル数・issue 数・PR 数・伏せた login 数を 1 行で出す

#### Scenario: 採取して保存する
- **WHEN** `gh` が使える状態で、open issue 2 件（108, 140）と open PR 1 件（131）のリポジトリ `owner/name` に対し `fixture capture --repo owner/name --alias app` を実行する
- **THEN** `internal/gh/testdata/fixtures/app/` に `search-issues.json` / `search-prs.json` / `issue-108.json` / `issue-108-cross-refs.json` / `issue-108-timeline.json` / `issue-140.json` / `issue-140-cross-refs.json` / `issue-140-timeline.json` / `pr-131.json` / `pr-131-review-threads.json` の 10 ファイルができ、標準出力に `10` と `app` を含む要約が出て、終了コードは 0 である

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

### Requirement: 伏せ字は個人・組織情報だけを置き換え、分類に使う文字列を変えない
伏せ字は採取した全ファイル（`map[string][]byte`）に対して 1 回の処理として行い、テキスト置換で MUST 実装する（JSON にデコードして再エンコードしない。`gh` の出力形式を置換箇所以外そのまま保つため）。処理は次の順で行う。`owner` / `name` / login / 衝突ガードの比較と置換は、すべて大文字小文字を区別しない（`--repo` を search 結果の `nameWithOwner` で正規化することはしない）。
1. **メールアドレス**（`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`）を全ファイルで `user@example.com` に置き換える
2. **login 表**を作る。`owner` を `user-1` にする。次にファイル名の昇順で全ファイルを走査し、`"login":"<x>"`（`"login"` の後の空白は許す）の値を初出順に `user-2`, `user-3`, … と番号付けする。続いて同じ順で `@<x>`（`x` は `[A-Za-z0-9][A-Za-z0-9-]*`、直前の文字が `[A-Za-z0-9_.-]` でないもの、`x` が `user-<数字>` 形式でないもの）を走査し、表に無い `x` を初出順に続きの番号にする。同じ login は常に同じ番号になる
3. **衝突ガード**。リポジトリ名 `name`、`owner`、手順 2 で `"login"` 値から得た login のいずれかが、(a) いずれかのファイルに JSON キーとして現れる（`"<x>"` の直後に空白を挟んで `:` が続く）か、`login` / `name` 以外のキーの JSON 文字列値として現れる（`"<キー>"\s*:\s*"<x>"` の形。`success` / `failure` / `completed` / `unknown` / `labeled` / `unlabeled` / `open` / `merged` / `checkrun` のような列挙値を拾う。`login` 値は置換対象そのもので、`name` 値はリポジトリ名 `repository.name` として必ず現れるので除く）、または (b) `todo` / `propose` / `apply` / `archive` / `question` / `blocked` / `wip` / `docs` / `routine` / `human` / `stage` / `pr` のいずれかに一致する場合、伏せ字は衝突した文字列を含むエラーを返し、ファイルを書かない（全文置換が JSON キー・列挙値・ラベル名を壊すため）
4. 各ファイルに次の置換を順に行う。「単語として」は、対象の直前と直後が `[A-Za-z0-9-]` でない（または文字列の先頭・末尾である）出現を指し、login 1 つにつき 1 回の走査で置き換える
   - `owner/name` → `org/<alias>`
   - `owner` と、手順 2 で `"login"` 値から得た各 login を、単語として `user-N` に
   - `name` を単語として `<alias>` に
   - `@<x>`（手順 2 と同じ形、`x` が `user-<数字>` でないもの）を表の番号で `@user-N` に

次の文字列は上の規則で変化しない（分類器がこれらに依存する）: ラベル名（`stage:todo` 等）、`<!-- routine -->` とそのエスケープ形 `&lt;!-- routine --&gt;`、`## Q1.` 形式の見出し、`未確定の判断: N 件`、`blocked-by:` で始まる行、`Refs #n` / `Closes #n`、`## PR リスク評価`。login 表の作成と置換は `@mention` に登場したものも含めるので、`@` 付きの言及と `author.login` は同じ番号になる。

#### Scenario: owner/name と URL が alias になる
- **WHEN** `owner` が `acme`、`name` が `widgets`、alias が `app` で、`{"repository":{"name":"widgets","nameWithOwner":"acme/widgets"},"url":"https://github.com/acme/widgets/issues/5"}` を伏せ字にする
- **THEN** 結果は `{"repository":{"name":"app","nameWithOwner":"org/app"},"url":"https://github.com/org/app/issues/5"}` である

#### Scenario: 同一人物は同一番号になる
- **WHEN** `owner` が `acme`、`name` が `widgets`、alias が `app` で、`issue-1.json` に `"author":{"login":"Alice"}` と本文 `@bob と @alice で確認`、`pr-2.json` に `"author":{"login":"bob"}` がある 2 ファイルを伏せ字にする
- **THEN** `issue-1.json` は `"author":{"login":"user-2"}` と `@user-3 と @user-2 で確認`、`pr-2.json` は `"author":{"login":"user-3"}` になる（`acme` は `user-1`、`Alice` は `user-2`、`bob` は `user-3`）

#### Scenario: owner が投稿者でもある
- **WHEN** `owner` が `acme`、`name` が `widgets`、alias が `app` で、`"author":{"login":"acme"}` と `"nameWithOwner":"acme/widgets"` と本文 `@acme さん` を含むファイルを伏せ字にする
- **THEN** `"author":{"login":"user-1"}`、`"nameWithOwner":"org/app"`、`@user-1 さん` になる

#### Scenario: メールアドレスは置き換わり、その @ は mention にならない
- **WHEN** `owner` が `acme`、`name` が `widgets`、alias が `app` で、本文 `Co-Authored-By: Alice <alice@acme.example>` を伏せ字にする
- **THEN** `Co-Authored-By: Alice <user@example.com>` になり、login 表に `example` は入らない

#### Scenario: 分類に使う文字列は変わらない
- **WHEN** `{"labels":[{"name":"stage:propose"},{"name":"question"}],"body":"未確定の判断: 2 件\n\n<!-- routine -->\n## Q1. 名前での絞り込みを含めるか\n- 選択肢 A（推奨）\nblocked-by: human\nRefs #108\n&lt;!-- routine --&gt;\n## PR リスク評価"}` を、`owner` が `acme`、`name` が `widgets` の状態で伏せ字にする
- **THEN** 出力は入力と 1 バイトも変わらない

#### Scenario: login の一部を含む単語は変わらない
- **WHEN** `owner` が `al`、`name` が `widgets`、alias が `app` で、本文 `also @al-team al` を伏せ字にする
- **THEN** `also @user-2 user-1` になる（`also` と `al-team` は `al` に単語として一致しない。`@al-team` は表に無い mention として `@user-2`、末尾の `al` は owner として `user-1`）

#### Scenario: login が JSON キーと衝突する
- **WHEN** `"author":{"login":"status"}` と `"status":"COMPLETED"` を含むファイルを伏せ字にする
- **THEN** `status` を含むエラーが返り、ファイルは書かれない

#### Scenario: login が JSON の列挙値と衝突する
- **WHEN** `"author":{"login":"completed"}` と `"status":"COMPLETED"` を含むファイルを伏せ字にする
- **THEN** `completed` を含むエラーが返り、ファイルは書かれない

#### Scenario: リポジトリ名がラベル名と衝突する
- **WHEN** `name` が `docs` の状態で伏せ字にする
- **THEN** `docs` を含むエラーが返り、ファイルは書かれない（`stage` / `pr` も同様）

### Requirement: 採取した fixture は個人情報を含まないことをテストで検査する
`internal/gh` のテストは `internal/gh/testdata/fixtures/` 直下の全 `<alias>` ディレクトリ（`example` を含む。3 だけは `example` を除く）の全ファイルに対して次を MUST 検査する。
1. `"login":"<x>"`（手順 2 と同じ形。コロン後の空白を許容）の値がすべて `^user-[0-9]+$` に一致する
2. `user@example.com` 以外のメールアドレス（上記の正規表現に一致するもの）が無い（伏せ字の置換値 `user@example.com` 自体がこの正規表現に一致するため）
3. 環境変数 `SUGI_LOOP_FIXTURE_ORIGIN` に `owner/name` が設定されているとき、`example` 以外のディレクトリのどのファイルにも `owner` と `name` のどちらも（大文字小文字を区別せず）含まれない（`example` は手書きで `org/app` 固定なので対象外）。未設定のときはこの項目を `t.Skip` で明示的に飛ばす

このテストは採取直後に利用者が `SUGI_LOOP_FIXTURE_ORIGIN=<owner/name> go test ./internal/gh/` で実行し、通ってからコミットする。CI では 1 と 2 が常に走る。

#### Scenario: 伏せ字済みの fixture は通る
- **WHEN** `fixtures/app/` の全ファイルの login が `user-N` 形式で `user@example.com` 以外のメールアドレスを含まず、`SUGI_LOOP_FIXTURE_ORIGIN=acme/widgets` で `acme` も `widgets` も含まれない
- **THEN** テストは通る

#### Scenario: 元の login が残っている
- **WHEN** `fixtures/app/issue-5.json` に `"login":"alice"` が残っている
- **THEN** テストは失敗し、失敗メッセージにファイルパスと `alice` を含む

#### Scenario: 元の owner が残っている
- **WHEN** `SUGI_LOOP_FIXTURE_ORIGIN=acme/widgets` で、`fixtures/app/` のいずれかのファイルの本文に `Acme` が残っている
- **THEN** テストは失敗し、失敗メッセージにファイルパスと `acme` を含む

#### Scenario: example は元 owner の検査の対象外
- **WHEN** `SUGI_LOOP_FIXTURE_ORIGIN=someone/app` で、`fixtures/example/` に `org/app` が含まれる
- **THEN** 3 は `example` を見ないのでテストは通る

#### Scenario: 環境変数が無ければ元 owner の検査は飛ばす
- **WHEN** `SUGI_LOOP_FIXTURE_ORIGIN` を設定せずにテストを実行する
- **THEN** 1 と 2 は検査され、3 は skip として報告される

