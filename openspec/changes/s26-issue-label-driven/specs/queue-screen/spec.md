## MODIFIED Requirements

### Requirement: 取得が成功して Card が 0 件のときは表の領域にヒントを出す
キュー画面は、直近の取得が成功して `Cards` が 0 件（全タブが空）のとき、表の領域の代わりに次の 2 行を MUST 出す（mvp.md「初回起動（onboarding）」の文言。Markdown のバッククォートは付けない）。2 方式のどちらの利用者も設定を直せるように、両方の入口を示す。
```
stage:* / To Do ラベルの無いリポジトリは何も出ません。
issue-driven-sdd の routines-setup を回すか、repos に mode: label を設定してください
```
2 行は表の領域（表の高さ）の縦中央に置き、各行を端末幅の横中央に置く（左に空白を置き、右には足さない）。端末幅より長い行は幅で切る。プレビューの領域は変えない。
次のときは出さない。
- 取得中（`fetching` が true。初回のスピナーを含む）: `Cards` が空でも表は 0 行のまま（「取得は非同期に行い、取得中はスピナー、失敗時は前回結果を維持する」の初回取得前の表示を変えない）
- 取得失敗（フッタに error の文字列を出しているとき）: 前回結果の維持と誤解させないため。初回取得の失敗で `Cards` が空でも出さない
- 部分失敗（`Result.Errors` が 1 件以上）で `Cards` が 0 件: search は成功しているので出す（フッタの `詳細取得の失敗 …` はそのまま）
ヒントはキュー画面だけに出す。他の画面（s09 の詳細）は変えない。

#### Scenario: 取得成功で 0 件ならヒントが出る
- **WHEN** `Cards` が空で `Errors` も空の `Result` を取得完了として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `stage:* / To Do ラベルの無いリポジトリは何も出ません。` と `issue-driven-sdd の routines-setup を回すか、repos に mode: label を設定してください` の 2 行が含まれ、ヘッダの `[1]今やる 0` とフッタは従来どおり出る

#### Scenario: 取得中はヒントを出さない
- **WHEN** `New` 直後（初回取得前）の `Model` の `View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含まず、フッタに `取得中` が含まれる

#### Scenario: 取得失敗ではヒントを出さない
- **WHEN** `New` 直後に error `search issues: gh search issues: exit 1: rate limited` を取得失敗として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含まず、フッタに `rate limited` が含まれる

#### Scenario: Card があればヒントを出さない
- **WHEN** `example` の `Result` を取得完了として渡し、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含まず、表に `PR131` の行がある

#### Scenario: ヒントは横中央に置かれる
- **WHEN** `Cards` が空の `Result` を渡した `Model` に幅 120・高さ 40 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含む行の左側の空白の数は、（120 − その行の表示幅）÷ 2 の切り捨てに等しい

#### Scenario: 幅 60 でもヒントは表の領域に出てプレビューも描かれる
- **WHEN** `Cards` が空の `Result` を渡した `Model` に幅 60・高さ 40 のサイズメッセージを与え、`View` から ANSI エスケープを除いて読む
- **THEN** `routines-setup` を含む行と、プレビュー領域の `（このタブにはカードがありません）` の両方が含まれる
