## Purpose

issue と PR を close する操作の直前を担う。close してよいかの判定を 1 か所に置き、
issue と PR で書き先を混同せずに 1 回だけ close する。

## ADDED Requirements

### Requirement: CheckClose は open でない PR を拒否として返す

`internal/action` は、close の判断材料を拒否の理由として返す関数を MUST 提供する。判定は渡された値だけで行い、`gh` を呼ばない。

- 対象が PR で、その `State` が空文字列でも `OPEN` でもなければ、拒否に `<State> の PR です` を足す。空文字列は
  「状態が分からない」として拒否しない（`merge-action`「`CheckMerge` は draft と open でない PR を拒否とし、他の判断材料は警告として返す」と同じ規則）
- 対象が issue のときは拒否しない。`internal/model` の issue は状態を持たず、画面に出る issue は取得時点では必ず open である
  （`gh-client`「`Client` は gh サブプロセスを正確な引数で実行する」の `SearchIssues` が `--state open` で引く）
- draft・`question` ラベル・未確定の判断の件数・checks の状態は、close の判断には使わない。これらは merge の判断材料であり、
  close を止める理由にならない

拒否が空でないとき、`internal/ui` は close の選択肢を出さない（`close-issue-pr`「close の確認画面は対象と種別を出し、y で close して Esc で中止する」）。

#### Scenario: merged 済みの PR は拒否になる
- **WHEN** `State` が `MERGED` の PR を対象として判定する
- **THEN** 拒否は `MERGED の PR です` の 1 件である

#### Scenario: closed の PR は拒否になる
- **WHEN** `State` が `CLOSED` の PR を対象として判定する
- **THEN** 拒否は `CLOSED の PR です` の 1 件である

#### Scenario: open の PR は拒否にならない
- **WHEN** `State` が `OPEN`、`IsDraft` が true、`Labels` に `question` を持ち、本文 1 行目が `未確定の判断: 2 件` で、
  `MergeState` が nil の PR を対象として判定する
- **THEN** 拒否は空である（draft も question も未確定も checks も close を止めない）

#### Scenario: 状態が分からない PR は拒否にならない
- **WHEN** `State` が空文字列の PR を対象として判定する
- **THEN** 拒否は空である

#### Scenario: issue は拒否にならない
- **WHEN** issue を対象として判定する
- **THEN** 拒否は空である

### Requirement: Close は対象の種類に応じた書き先へ 1 回だけ close を送る

`internal/action` は、対象（リポジトリ・番号・issue か PR か）を受けて 1 回だけ close する関数を MUST 提供する。
対象が PR なら `gh-client` の `ClosePR`、issue なら `CloseIssue` を呼ぶ（`answer-question` の回答が
`action.Target` の種別で書き先を分けるのと同じ形。human-turn-signals.md 不変条件 4「書き先を混同しない」）。
拒否の判定（Requirement「`CheckClose` は open でない PR を拒否として返す」）はこの関数では行わない。
拒否と確認の提示は `internal/ui` の確認画面の責務で、`y` を押した人の判断を最後の関門にする（`merge-action` の `Merge` と同じ）。
ラベルもコメントも書かない（human-turn-signals.md 不変条件 2）。

#### Scenario: PR は ClosePR に行く
- **WHEN** `org/app` の PR 131 を対象として close する
- **THEN** `Fake.Calls` は 1 件で、`Method` が `ClosePR`、`Repo` が `org/app`、`Number` が 131 である

#### Scenario: issue は CloseIssue に行く
- **WHEN** `org/app` の issue 108 を対象として close する
- **THEN** `Fake.Calls` は 1 件で、`Method` が `CloseIssue`、`Repo` が `org/app`、`Number` が 108 である

#### Scenario: ラベルもコメントも書かない
- **WHEN** issue 108 と PR 131 をそれぞれ close した後に `Fake.Calls` を見る
- **THEN** `Method` が `AddLabel` / `RemoveLabel` / `CommentIssue` / `CommentPR` の要素は 1 件も無い

#### Scenario: close の失敗はそのまま返る
- **WHEN** `CloseIssue` がエラーを返す `client` で issue 108 を close する
- **THEN** そのエラーがそのまま返る
