# close-action Specification

## Purpose
issue と PR を close する操作の直前を担う。issue と PR で書き先を混同せずに 1 回だけ close し、
close が上流の routine に却下として伝わる PR を人に知らせる。

## Requirements

### Requirement: Close は対象の種類に応じた書き先へ 1 回だけ close を送る

`internal/action` は、対象（リポジトリ・番号・issue か PR か）を受けて 1 回だけ close する関数を MUST 提供する。
対象が PR なら `gh-client` の `ClosePR`、issue なら `CloseIssue` を呼ぶ（`answer-question` の回答が `action.Target` の種別で
書き先を分けるのと同じ形。human-turn-signals.md 不変条件 4「書き先を混同しない」）。種別で書き先が変わる操作は
`action.Target` を受け取り、種別で変わらない操作は `(repo, number)` を受け取る（`merge-action` の `Merge` と
`todo-action` の `ToggleTodo` は後者）。

close の可否は判定しない。issue も PR も画面に出ているものは取得時点で open であり（`gh-client` の `SearchIssues` /
`SearchPRs` が `--state open` で引く）、`internal/fetch` は検索の結果からしか Card を作らないので、
「open でないものを止める」判定は発火しない（design.md「拒否を持たない」）。この関数はラベルもコメントも書かず、
human-turn-signals.md 不変条件 2 を守る。close の理由（`gh issue close --reason`）は渡さない。

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
- **THEN** そのエラーがそのまま返り、`Fake.Calls` に `ClosePR` は無い

### Requirement: CheckClose は却下として扱われる PR の注意を返す

`internal/action` は、close の注意を返す関数を MUST 提供する。判定は渡された値だけで行い、`gh` を呼ばない。

- 対象が PR で、`Labels` に `propose` または `apply` が含まれるとき、注意に
  `merge せずに close した PR は却下として扱われ、issue に blocked-by: human が書き戻されます` を足す。
  上流の dispatcher と sweep は、merge されずに close された `propose` / `apply` PR を「人が却下した」とみなして
  issue に `blocked-by: human` を書き戻す（human-turn-signals.md「人が merge せずに close した propose / apply PR も
  同じく dispatcher が `blocked-by: human` にするので B に出る」）。close が別の人の出番を作ることを、押す前に人へ知らせる
- 対象が issue のとき、および `archive` / `docs` / ラベル無しの PR のときは注意を返さない。`archive` PR の close は
  段階を戻さず、`docs` PR は issue に紐づかない

注意は close を止めない。止めるかどうかは `internal/ui` の確認画面で人が決める（`merge-action` の警告と同じ扱い）。

#### Scenario: propose ラベルの PR には注意が出る
- **WHEN** `Labels` が `propose` と `question` の PR を対象として判定する
- **THEN** 注意は `merge せずに close した PR は却下として扱われ、issue に blocked-by: human が書き戻されます` の 1 件である

#### Scenario: apply ラベルの PR にも注意が出る
- **WHEN** `Labels` が `apply` の PR を対象として判定する
- **THEN** 注意は 1 件である

#### Scenario: archive と docs とラベル無しの PR には注意が出ない
- **WHEN** `Labels` が `archive` の PR、`docs` の PR、空の PR をそれぞれ対象として判定する
- **THEN** どれも注意は空である

#### Scenario: issue には注意が出ない
- **WHEN** `Labels` に `propose` を持つ issue を対象として判定する
- **THEN** 注意は空である
