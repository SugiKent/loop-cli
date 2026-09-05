# sugi-loop 技術的意思決定

最終更新: 2026-09-05-1805

sugi-loop の技術・データ層の判断とその理由。古い決定は上書きせず、変更日時と理由を追記する。
体験と画面の仕様は [mvp.md](./mvp.md)、分類ルールは [人の出番（判定ルール）](../domain/issue-driven-sdd/human-turn-signals.md) を見る。

---

## D-001 データ取得は「広く 2 回検索してクライアント側で分類する」

決定日時: 2026-09-05-1407

**方針: 広く 2 回検索して、分類はクライアント側で行う。** ラベル別に 6 回検索する方式は search API の
30 req/分の制限に近づき、クエリ文字列のクォート（`label:"stage:todo"`）も壊れやすい。検証では
クエリ文字列形式の `user:… label:"stage:todo"` は失敗し、フラグ形式の `--repo A --repo B` は動作した。

1 回の更新で行う呼び出し:

```bash
# open issue 全件（設定した全リポジトリ）
gh search issues --repo O/a --repo O/b … --state open --limit 200 \
  --json repository,number,title,labels,updatedAt,url,body,commentsCount
# open PR 全件
gh search prs --repo O/a --repo O/b … --state open --limit 200 \
  --json repository,number,title,labels,updatedAt,url,body,isDraft
```

分類に本文以外が必要な項目だけ、遅延で追加取得する。

| 分類 | 追加取得 | 目的 |
| --- | --- | --- |
| `question` の issue | `gh issue view --json comments` | 最新コメントが AI か人か（回答待ちか、sweep 待ちの回答済みか） |
| `question` の PR | `gh pr view --json comments` | 最新コメントが AI か人か（回答待ちか回答済みか） |
| merge 候補 PR | `gh pr view --json mergeable,mergeStateStatus,statusCheckRollup,reviewDecision` | merge ガード |
| `apply` PR | `gh api graphql`（`reviewThreads(first:50){isResolved comments{databaseId author body}}`） | 局面 D |

2026-09-05-1805 追記: `blocked` の issue に対する `blocked-by: human` 検出の行を `question` の issue に置き換えた。上流が
`blocked-by: human` を `question` ラベルで可視化するようになり、ラベルだけで局面 B を判定できる（D-004 を見る）。

2026-09-05-2130 追記（s20-fetch-all-details）: **上の遅延取得表をやめ、詳細は全件先取りにした。** search で得た
すべての open issue に `gh issue view --json comments`、すべての open PR に `gh pr view --json comments`・
`gh pr view --json mergeable,…`・`gh api graphql`（reviewThreads）を無条件に呼ぶ。理由は、取得条件から外れた対象の
詳細が `nil` のまま残り、カード詳細と PR 詳細に `未取得` が並ぶこと。一覧を見ても「調べていないから分からない」が
残るので、確認の手数が減らないという道具の目的そのものを損なっていた。全件取得にすると `nil` は「取得に失敗した」
だけを意味するようになり、表示も `取得失敗` に変わる。分類結果は変わらない（`classify` が詳細を読む分岐は
`question` / merge 候補の条件 / `apply` ラベルで守られており、上の取得条件と一致していた）。

呼び出し回数は open が issue N 件・PR M 件のとき `2 + N + 3M` になる（従来は `2 + α + M`）。search 2 回は REST の
search 枠（30 req/分）のままで変わらず、増えるのは GraphQL 枠（5,000 point/時）だけである。GraphQL は回数ではなく
point で数えるので単価は実測が要るが、**仮に 1 呼び出し 1 point とする**と `refresh_interval_sec` の既定値 120 秒
（30 回/時）では 1 更新あたり 166 回が上限の境目で、search の 2 回を除くと `N + 3M ≤ 164` になる。issue 100 件 +
PR 20 件（160）は余裕、issue 200 件 + PR 50 件（350）は超える。この枠は sugi-loop の専有ではなく、手動の `R` と
利用者自身の他の `gh` 利用が同じトークンの枠を使う。取得件数の上限やレート制限のガードは設けていない。
`detailConcurrency` は 4 のままなので、呼び出しが増えた分だけ 1 回の更新にかかる時間は伸びる。

**Issue と PR の紐づけ（カード化）** は 2 つの根拠を合わせる。どちらも `gh` で取れる（稼働リポジトリで確認済み）。

1. PR 側: title の `[<段階>] #<n>` と本文の `Refs #n` / `Closes #n` をパースする。search 結果の title / body だけで済み、追加呼び出し不要。
2. Issue 側（取りこぼし防止、詳細を開いたときだけ）: GraphQL の `timelineItems(itemTypes:[CROSS_REFERENCED_EVENT])` で
   Issue を参照した PR を引き、`propose` / `apply` / `archive` ラベルを持ち title か本文に `#n` を含むものだけを採る。
   無関係な `docs` PR が本文で番号に触れているだけのものは除外する。

**ラベル変遷** は REST の timeline から取る。DB は持たない。GitHub がイベントを保持しているので、ローカルに履歴を溜める必要がない。

```bash
gh api "repos/{o}/{r}/issues/{n}/timeline" --paginate \
  --jq '.[] | select(.event=="labeled" or .event=="unlabeled") | {created_at, event, label: .label.name}'
```

これで「`stage:propose` が何回付け直されたか（再起動回数）」「今の段階に入って何時間か」「`wip` がいつ外れたか」を計算する。
1 Issue 1 リクエスト（通常 1 ページ）なので、カード詳細を開いたときと、カンバンの段階列にあるカード（数十件）の
背景取得に限る。バックログ列は取らない。REST の通常枠は 5,000 req/時で、この量なら問題にならない。
SQLite を使うのは、GitHub 側から消える情報（削除されたコメント等）を残したくなった場合の Phase 3 以降の選択肢に留める。

**`mergeable: UNKNOWN`** は GitHub が初回参照時に遅延計算している状態で、恒久的な値ではない。merge ガードの取得で
UNKNOWN を見たら 2 秒後に 1 回だけ再取得する。しないと新しい PR が全部「merge 不可」に見える。

search のレート制限は 30 req/分。上記は 1 更新あたり search 2 回 + 詳細は選択時と分類時のみなので、自動更新 2 分間隔で余裕がある。
Phase 2（[implementation-tasks.md](./implementation-tasks.md)）では `gh api graphql` の `search()` エイリアスで 2 検索を 1 リクエストに畳み、`rateLimit { remaining resetAt }` を
同時に取ってステータスバーに出す。


---

## D-002 永続 DB を持たず、スナップショットキャッシュのみ

決定日時: 2026-09-05-1407

- 永続 DB は持たない。メモリ上のスナップショットを `~/.cache/sugi-loop/snapshot.json` に保存し、起動直後は stale 表示 → 背景で再取得。
  スナップショットは消しても GitHub から全部作り直せる派生データに限る。
- 手動 `R`、自動更新は既定 120 秒（設定可）。書き込み後は対象 1 件だけ再取得する。
- 取得中はステータスバーにスピナー、失敗時は前回結果を維持してエラーを赤で表示。


---

## D-003 言語・TUI ライブラリ・配布方法の選定

決定日時: 2026-09-05-1407

| 層 | 採用 | 理由 |
| --- | --- | --- |
| 言語 | **Go**（手元に go 1.26） | 単一バイナリ配布。gh-dash / prs など GitHub TUI の先行例が Go + Bubble Tea で、設計を参照できる |
| TUI | **Bubble Tea v2** + **Bubbles v2**（list / viewport / textarea / help / key） | 2026-02 に v2 正式版。Elm 型で `gh` 呼び出しを `tea.Cmd` として非同期化しやすい |
| スタイル | **Lip Gloss v2** | 色・枠・レイアウト。TrueColor と 256 色のダウンサンプリングをレンダラが処理 |
| Markdown | **Glamour** | Issue / PR 本文とコメントを色付きで端末表示 |
| フォーム | **huh** | Issue 作成フォーム（リポジトリ選択・タイトル・本文） |
| GitHub | **`gh` サブプロセス**（`gh search` / `gh issue` / `gh pr` / `gh api`） | 利用者の要件。認証・GHES・credential store を再利用。すべて `--json` で受ける |
| 配布 | `go install` + GoReleaser。**gh extension（`gh sugi-loop`）としての配布も候補** | `gh` 前提のツールなので `gh extension install <owner>/gh-sugi-loop` が自然 |

比較した代替: Rust + ratatui（入力・非同期ループを自作する範囲が広い）、TypeScript + Ink v7 / OpenTUI（単一バイナリ化に手間、OpenTUI は 0.x）、
Python + Textual（最速で作れるが配布サイズと起動速度で劣る）。

### 内部構成

```
cmd/sugi-loop/main.go
internal/
  config/      # config.yml 読み込み
  gh/          # GHClient interface と gh サブプロセス実装（+ JSON fixture の fake）
  model/       # Card（Issue + 紐づく PR 群 + ラベル変遷）・Comment・分類結果
  classify/    # 人の出番の判定表（domain/issue-driven-sdd/human-turn-signals.md）を実装する純粋関数。fixture でテスト
  ui/          # Bubble Tea: queue / detail / reply / form / help
  action/      # ラベル・コメント・merge の書き込み（不変条件 1〜7 を持つ）
```

`GHClient` を interface にし、稼働中リポジトリから取った JSON を個人・組織情報を伏せて fixture にし、 `classify` を live データなしで
テストする。分類器がこのツールの価値の本体なので、ここだけは先にテストを書く。

---

## D-004 上流プラグイン `d8db3842` への同期。人はラベルを触らず、TUI もそれに従う

決定日時: 2026-09-05-1805

上流 issue-driven-sdd（2026-09-05 の `d257ef45` → `e83def7e` → `d8db3842`）で人の役割が
「`stage:todo` を付ける / `question` の PR・issue にコメントで答える / PR を merge する」の 3 つに固定された。
`blocked` / `question` / 段階ラベルの付け外しと worker の再起動は dispatcher と sweep が行う。

sugi-loop 側の変更:

| 変更 | 理由 |
| --- | --- |
| `u`（ブロック解除。方針コメント → `blocked` 外し → 段階ラベル付け直し）を削除 | 人が `blocked` を外すと dispatcher の状態機械と衝突する。issue にコメントすれば sweep が外して再起動する |
| 局面 B の検知を「`blocked` + 最新 `blocked-by:` に `human`」から「`question` ラベル」に変更 | 上流が `question` を issue にも付け、`is:open label:question` で人待ちを全部出せるようにした |
| 局面 F を「段階ラベル 2 つ以上」だけに縮小 | `restart: 3/3` と `question` 付き merge は dispatcher が `blocked-by: human` に変換するので B に出る |
| 局面 G から `retro` を除外 | 上流が `retro` を旧構成としてプラグイン対象外にした。その他バケットには出る |
| `## PR リスク評価` 見出しの AI 判定は維持 | `assess-pr-risk` はプラグインから外れたが、既存コメントは残っている。外すと人の発言と誤判定して A の PR がキューから消える |

TUI が書くラベルは `stage:todo` と `s` の `stage:propose` の 2 つだけになり、書き込み面が小さくなった。
上流は複数リポジトリの横断をスコープ外と明記しており、sugi-loop の位置づけは変わらない。

---

## 参考（Perplexity 調査 2026-09-05）

- Bubble Tea v2 発表: https://charm.land/blog/v2/ / https://github.com/charmbracelet/bubbletea
- Lip Gloss / Bubbles / Glamour / huh: https://github.com/charmbracelet
- gh-dash（セクション = search クエリの設計、YAML 設定、`r/R` 更新）: https://github.com/dlvhdr/gh-dash / https://gh-dash.dev/configuration/
- prs（dhth）: https://github.com/dhth/prs / octo.nvim（`gh api graphql` を subprocess で使う先行例）: https://github.com/pwntester/octo.nvim
- `gh search issues` / `gh search prs`（`--repo` 複数指定）: https://cli.github.com/manual/gh_search_issues / https://cli.github.com/manual/gh_search_prs
- Search API レート制限（認証済み 30 req/分）: https://docs.github.com/en/rest/search/search
- GraphQL レート制限・`rateLimit`: https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api
- review thread への返信（`addPullRequestReviewThreadReply` / REST replies）: https://docs.github.com/en/graphql/reference/pulls
- ratatui: https://ratatui.rs/ / Ink: https://github.com/vadimdemedes/ink / OpenTUI: https://github.com/anomalyco/opentui / Textual: https://github.com/Textualize/textual

---

## 変更履歴

| 日時 | 変更内容 | 理由 |
| --- | --- | --- |
| 2026-09-05-2130 | D-001 の遅延取得表を全件先取りに置き換え、呼び出し回数の見積もりを追記 | 取得条件から外れた詳細が `未取得` のまま並び、確認の手数が減らなかったため（s20-fetch-all-details） |
| 2026-09-05-1805 | D-004 を追加、D-001 の遅延取得表の `blocked` 行を `question` の issue に置換 | 上流 `d8db3842` の規約変更に同期するため |
| 2026-09-05-1407 | `docs/mvp/design.md` のデータ層（取得・キャッシュ）・技術選定・参考リンクを意思決定記録として分離 | docs 管理規約が求める `docs/mvp/decisions.md` を正本にし、各決定に決定日時を付けて後から変更履歴を追えるようにするため |
