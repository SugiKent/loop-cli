# loop-cli MVP

最終更新: 2026-09-10-0700

issue-driven-sdd（Claude Code Routines が GitHub Issue のラベルで propose → apply → archive を回す構成）と
issue-label-driven（`To Do` / `In Progress` / `Done` の 3 ラベルだけで進む構成）で、
**人の出番だけ**を複数リポジトリ横断で 1 本のキューに並べ、先頭から捌くための TUI。
`gh` CLI をデータ層に使い、ローカルで動く。

- 対象プラグイン: https://github.com/SugiKent/sugiken-dev-plugin-public/tree/main/plugins/issue-driven-sdd

---

人の出番の判定ルール（分類器の仕様）は [人の出番（判定ルール）](../domain/issue-driven-sdd/human-turn-signals.md) を正本とする。
技術選定とデータ層の判断は [decisions.md](./decisions.md)、実装順序は [implementation-tasks.md](./implementation-tasks.md) を見る。

---

## 体験の中心

**リポジトリ切り替えという概念を持たない。** 全リポジトリの「今やるべきこと」が 1 本のリストに優先度順で並び、
先頭を開いて捌くと次が先頭になる。リポジトリは各行の 1 列にすぎない。gh-dash がセクション（クエリ）単位で
表を分けるのと対照的に、ここでは「人の役割」で分類し、リポジトリは横断する。

### 2 つのビューと 1 つのカード

表示単位は **カード = Issue 1 件 + それに紐づく PR 群**。propose / apply / archive の PR は別の行にせず、Issue カードの中に
段階順に並べる。PR 単独の行が出るのは、Issue に紐づかない `docs` PR と「その他」バケットの PR だけ。

| ビュー | 目的 | 並び |
| --- | --- | --- |
| **今やるキュー**（既定、`v` で切替） | 人の出番だけを先頭から捌く | [人の出番（判定ルール）](../domain/issue-driven-sdd/human-turn-signals.md) の優先度順。カードは「いま人が何をすべきか」を 1 行目に出す |
| **カンバン**（`v` で切替） | ループ全体の流れを見る | 列 = バックログ（段階なし）→ `stage:todo` → `stage:propose` → `stage:apply` → `stage:archive`。全リポジトリのカードが同じ列に混ざる |

カンバンの列はラベル 1 本で決まるので、GitHub Projects を使わずラベルだけから構成できる。`blocked` / `wip` / `question` は
カード上のバッジで示し、列は動かさない。段階ラベルが 2 つある異常カードは両列に赤で出す。

```
┌ カンバン ─────────────────────────────────────────────────────────────────────────────┐
│ バックログ 12     │ todo 3           │ propose 4         │ apply 2          │ archive 2   │
│ ─────────────    │ ──────────────   │ ───────────────   │ ──────────────   │ ─────────── │
│ app #140 集計が… │ web #48 監視を…   │ app #108 選択UI…   │ app #91 旧機能…   │ app #89 分離… │
│ app #137 CI が…  │ tpl #3 ログイン…  │   🟣 PR131 merged  │   ⛔ blocked      │   PR151 ●   │
│ app #135 lint…   │ app #114 トップ… │   ⛔ blocked human │                  │   merge待ち  │
│ …                │                  │ app #107 振り返り… │                  │ app #103 作… │
│                  │                  │   ⛔ blocked       │                  │   PR144 ●   │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

### 画面構成（今やるキュー）

```
┌ loop-cli ── [1]今やる 7  [2]バックログ 12  [3]進行中 5  [4]異常 1 ────── ↻ 12:04 ─┐
│ 優先 種別    リポジトリ           #     タイトル                             経過    │
│ ▶ !!  質問    org/app             PR131 [propose] #108 選択 UI をモーダル化  12m     │
│                                         する                                         │
│   !!  質問    org/web             PR 88 [apply] #48 監視ツールを導入する     1h      │
│   !   方針    org/app             #108  選択 UI をモーダル化する             3h      │
│   !   方針    org/app             #91   旧機能を完全に削除する               2d      │
│   ●   merge   org/app             PR151 [archive] #89 decouple-feature-      5h      │
│                                         api-from-web                                 │
│   ●   merge   org/app             PR144 [archive] #103 block-creation-       9h      │
│                                         for-archived-orgs                            │
│   ●   merge   org/template        PR 12 [apply] #3 ログイン画面を作る        1d      │
├──────────────────────────────────────────────────────────────────────────────────────┤
│ 未確定の判断: 2 件 — merge しないでください        labels: propose question         │
│                                                                                      │
│ ▌AI  12:04  以下 2 点、回答をお願いします（記号だけで OK です・例: Q1: A）           │
│ ▌    ## Q1. 名前での絞り込みを今回のスコープに含めるか                                │
│ ▌    - 選択肢 A（推奨）: 含めない。…                                                 │
│ ▌    - 選択肢 B: 含める。…                                                          │
│ ▌    ## Q2. カードの情報量の見直しをどこまで行うか                                    │
│ ▌    …                                                                               │
├──────────────────────────────────────────────────────────────────────────────────────┤
│ Enter 開く  a 回答  t todo  m merge  n 新規issue  o ブラウザ  R 更新  ? ヘルプ  q 終了│
└──────────────────────────────────────────────────────────────────────────────────────┘
```

- 上段: キュー（タブで 4 分類）。行の色は種別で固定（質問=マゼンタ、方針=赤、merge=緑、todo 候補=シアン、異常=黄背景）。
  GitHub 側のラベル色（`question` D876E3、`blocked` B60205、`propose` 0E8A16 など）と揃える。
- タイトルは切り詰めず、列幅に収まらなければ折り返して全文を出す。継続行はタイトルの開始位置に揃え、`▶` と経過は 1 行目にだけ出す。表はスクロールを持たないので、折り返しで増えた行のぶん表に出るカードの枚数は減る。
- 下段: 選択行のプレビュー。Issue 本文 / PR 本文を Markdown レンダリングし、コメントは AI 発を左バーで区別する。
- 端末サイズによらず常に上下 2 ペインで出す。狭い端末では表とプレビューがそれぞれ短くなる。
- ヘッダ右の `↻ 12:04` は最後に取得が完了した時刻。新しい版が出ているときは、その左に `↑ update` が出る（`loop-cli update` で入れ直す）。幅が足りないときはタブ名 → `↑ update` → 時刻 の順に落とす。

### カード詳細（Enter）

1 枚のカードに Issue と PR 群をまとめて出す。

- ヘッダ: リポジトリ、Issue 番号、現在の段階、バッジ（blocked / wip / question）、この段階に入ってからの経過時間。
- **段階の変遷**: `stage:todo` 付与 → `stage:propose` 付与 → … をタイムラインで表示。同じ段階ラベルの付け直し回数
  （= dispatcher による再起動回数）と `wip` の付け外しも並べる。データ源は [decisions.md](./decisions.md) の D-001「ラベル変遷」。
- **紐づく PR**: `[propose] PR#131 merged` `[propose] PR#140 merged（最新・正本）` `[apply] なし` のように段階順に並べ、
  各 PR の 1 行目判定・ラベル・checks・merge 状態を出す。同じ段階の merge 済み PR が複数あれば最新に印を付ける
  （人の回答後に proposal を直す PR が増えるため。dispatcher も最新だけを見る）。PR を選んで Enter で PR の会話・review thread へ入る。
- Issue 本文、`depends on #m`、最新 `blocked-by:` の要約（`human` なら「人に何を決めてほしいか」の選択肢と推奨がここにある）、
  コメント時系列（routine コメントは折りたたみ、`x` で展開）。
- PR 側: 1 行目の判定結果、`Refs #n` / `Closes #n`、会話コメント、review thread（未 resolve を先頭）、checks 状態。
- 質問コメントは `## Q1.` 見出しと `選択肢 A/B` をパースして、回答テンプレート `Q1: A\nQ2: A` を `a` で事前入力する。
  推奨（`（推奨）`）を既定値にする。issue の `blocked-by: human` コメントは見出し形式が固定でないため、パースできた分だけ事前入力する。
  テンプレートに `<!-- routine -->` を絶対に含めない（人の発言として扱わせるため）。

### キーバインド

| キー | 動作 | 内部処理 |
| --- | --- | --- |
| `j` / `k` / `↑↓` | 行移動 | |
| `1`–`4` / `Tab` | タブ切替（今やる / バックログ / 進行中 / 異常） | |
| `v` | ビュー切替（今やるキュー ↔ カンバン） | |
| `h` / `l` / `←→` | カンバンの列移動 | |
| `Enter` | 詳細を開く | `gh issue view` / `gh pr view --json` |
| `a` | 回答・コメント（`$EDITOR` を開く）。PR の `question`（局面 A）と issue の `question`（局面 B）の両方。ラベルは触らない | `gh pr comment -R … --body-file` / `gh issue comment` |
| `A` | review thread に返信（PR 詳細で thread 選択中） | `gh api -X POST repos/{o}/{r}/pulls/{n}/comments/{id}/replies` |
| `t` | `stage:todo` を付ける / 外す（付け直しは dispatcher への「もう一度評価しろ」の合図） | `gh issue edit --add-label` / `--remove-label`（1 操作 1 ラベル） |
| `s` | 順番を飛ばして即着手 | 確認 → `stage:todo` が付いていれば外す → `stage:propose` を付ける（段階ラベルは同時に 1 つ）。TUI が段階ラベルを書く唯一の強制操作 |
| `m` | merge | ガード判定 → 確認 → `gh pr merge -R … --<method>` |
| `n` | 新規 Issue 作成。選択中の対象の repo に `$EDITOR` で作成（1 行目がタイトル、以降が本文）→ 確認 → 作成。repo は選ばせない | `gh issue create -R <選択中の repo>`。ラベルは付けない（段階ラベルも assignee も付けない） |
| `o` | ブラウザで開く | `gh browse` / `open <url>` |
| `u` | URL 一覧を開く（本文・コメント・review thread の URL を出典付きで並べ、選んで開く） | `open <url>`（macOS 以外は `xdg-open <url>`） |
| `g` | PR ↔ issue を相互ジャンプ | `Refs #n` / `Closes #n` をパース |
| `/` | 絞り込み（タイトル・リポジトリ・番号） | クライアント側 |
| `R` | 全件再取得 | |
| `?` | ヘルプ | |
| `q` | 終了 | |

---

## 設定ファイル `~/.config/loop-cli/config.yml`

```yaml
repos:
  - org/app
  - org/web
  - name: org/board
    mode: label               # sdd | label。既定 sdd。リポジトリごとの運用方式
refresh_interval_sec: 120
merge_method: squash          # squash | merge | rebase。リポジトリ別上書き可
editor: $EDITOR
notify: true                  # 人の出番が新しく増えたらデスクトップ通知
```

`mode` はそのリポジトリの運用方式を指す。`sdd` は issue-driven-sdd（`stage:*` + `wip` + PR の段階ラベル）、
`label` は issue-label-driven（`To Do` / `In Progress` / `Done` の 3 ラベル）である。省略すると `sdd` になる。
2 方式のリポジトリは 1 本のキューに混ざり、同じ 4 タブ・同じキー操作・同じ優先度で捌ける。

ラベル名と routine マーカーはプラグインの規約に固定し、設定で変えられるようにしない。分類器の正しさはこの規約に依存するため。
方式を足すときも、その方式ぶんの語彙を持たせるだけで、ラベル名そのものは設定で変えられるようにしない。

認証は `gh auth` を再利用する。トークンを設定ファイルに保存しない。

### 初回起動（onboarding）

- `~/.config/loop-cli/config.yml` が無いときは、エラーで終了せず huh のフォームで `repos`（`owner/name` を 1 件以上。改行区切りで複数可）、`merge_method`（squash / merge / rebase、既定 squash）、`notify`（既定 true）、`editor`（既定 `$EDITOR`）を聞き、設定ファイルを書き出して、そのまま TUI を起動する。`refresh_interval_sec` は聞かず既定 120。
- `gh` が無い、または `gh auth status` が失敗するときは、標準エラーに原因と次の一手（`gh` のインストール先 URL または `gh auth login`）を 1 行ずつ出して終了コード 1。
- キューが 0 件（全タブ空）のときは、画面中央に「`stage:*` / `To Do` ラベルの無いリポジトリは何も出ません。issue-driven-sdd の `routines-setup` を回すか、`repos` に `mode: label` を設定してください」のヒントを出す（「前提と未決事項」の 1 項目目を利用者に見せる形）。onboarding のフォームは `mode` を聞かない（リポジトリごとの値を聞く形になっていないため。設定ファイルの手編集に委ねる）。
- 設定ファイルが壊れているとき（YAML エラーや検証エラー）は onboarding に入らず、従来どおり原因を出して終了する（上書きしない）。

---

## 前提と未決事項

- **`routines-setup` を回していないリポジトリは空**。`stage:*` ラベルが無いリポジトリを `mode: sdd`（既定）で設定しても TUI には何も出ない。
  issue-label-driven のリポジトリは `mode: label` を書けば `To Do` / `In Progress` / `Done` で分類される。どちらの規約も持たない
  リポジトリは、issue が「着手を承認する」としてバックログに並ぶだけになる。最初の動作確認は稼働中のリポジトリ 1 件で行う。
  `mode: label` の書き忘れは、`sdd` 扱いのリポジトリに `To Do` / `In Progress` の issue があればフッタで知らせる。
- `question` ラベル（PR・issue とも）と `blocked` の付け外し、1 行目の N はプラグイン側（worker / dispatcher）が同期する規約。
  TUI は読むだけで書かない。TUI が書くラベルは `stage:todo` と `s` の `stage:propose` だけ。
- 上流の「人が持つ操作」にある「取り下げる・止める（段階ラベルを外す）」は TUI に対応キーが無い。必要になったら追加する。
- merge method はリポジトリ設定に従うが、`gh pr merge` の既定は対話式なので設定で明示する。
- ツール名 `loop-cli`（リポジトリ名）で進める。gh extension にするなら `gh-loop-cli`。
- 上流プラグインは「複数リポジトリの横断」をスコープ外（1 Routine 1 リポジトリ）と明記している。この隙間を埋めるのが loop-cli。

---

## 変更履歴

| 日時 | 変更内容 | 理由 |
| --- | --- | --- |
| 2026-09-10-1100 | キュー画面の画面図を、長いタイトルが折り返して全文出る形に描き直し、折り返しの規則を 1 項目足した | タイトルの切り詰めをやめて折り返しに変えたため（#2・s27-wrap-titles） |
| 2026-09-10-0800 | キーバインド表の `n` を「選択中の対象の repo に `$EDITOR` で作成」に変更（フォームでの repo 選択を廃止） | 気づいた瞬間に repo を選び直す往復を無くすため。作成先は画面が見せている対象から決まる（#6・s15-new-issue） |
| 2026-09-10-0700 | 設定ファイルに `mode: sdd \| label` を、0 件ヒントの文言と「前提と未決事項」に issue-label-driven の扱いを追加 | `To Do` / `In Progress` / `Done` の 3 ラベルで進むリポジトリを同じキューに載せるため（#5） |
| 2026-09-06-1200 | 画面構成にヘッダの `↑ update`（新しい版があるときの印）を追加 | `go install` で入れた後に更新に気づける手立てが無かったため（s23-self-update） |
| 2026-09-06-0024 | キーバインド表に `u`（URL 一覧を開く）を追加 | PR / issue の本文に貼られた URL へ、GitHub を経由せずキーボードだけで到達できるようにするため（s22-url-picker） |
| 2026-09-05-2130 | 「設定ファイル」節の直後に「初回起動（onboarding）」小節を追加（config.yml 無しはフォームで作成、gh 未認証の案内、空キューのヒント、壊れた設定は上書きしない） | 利用者要望。最初の起動は設定ファイルが無い状態から始まるため |
| 2026-09-05-1805 | 上流 `d8db3842` に同期。`u`（ブロック解除）を削除し、`a` を issue の `question` 回答にも使う。カード詳細に「同段階の複数 PR は最新が正本」を追加。未決事項に取り下げ操作の未対応と横断スコープを追記 | 上流が人の操作を `stage:todo` とコメントに限定したため、ラベルを操作する `u` は規約違反になる |
| 2026-09-05-1407 | `docs/mvp/design.md` を分割し、体験・画面・キーバインド・設定ファイル・前提と未決事項を MVP 定義として集約 | docs 管理規約の固定構成（`docs/mvp/mvp.md`）に合わせ、後続スキルが正本を名前で参照できるようにするため |
