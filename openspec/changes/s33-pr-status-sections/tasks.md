## 1. セクションの見出し行

- [ ] 1.1 `internal/ui` に見出し行を組み立てる関数を足す（`── <名前> ` の後を幅まで `─` で埋め、継ぎ足す本数は `max(幅 − 見出しの表示幅, 0)` で求め、収まらない幅では `ansi.Truncate` で切って `…` を付けない）。関数の戻り値を直接読むテストを書き、幅 100 / 幅 99 / 名前が入らない狭い幅 / 幅 0 の 4 通りを通す（`View` 越しでは viewport が各行を幅まで空白で埋めるので、幅の検証にならない）
- [ ] 1.2 `prBodyLines`（`internal/ui/detail.go`）に `本文` / `コメント` / `review thread` の見出し行を入れる。`card-detail` の ADDED Requirement「本文領域のセクションは見出し行で区切る」の幅の Scenario 2 件（幅 100 の 1 ペイン / 幅 140 の 2 ペインで左 99。どちらも末尾の空白を落としてから幅と末尾の文字を見る）を通す
- [ ] 1.3 中身が 0 行のセクションは見出しごと出さない条件を入れる。MODIFIED Requirement「PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す」の Scenario「詳細の取得に失敗した PR」（`Body` が空なので `── 本文 ` が出ない）を通す
- [ ] 1.4 `cardBodyLines`（同）に `blocked-by` / `コメント` の見出し行を入れる。`blocked-by` の見出しは `model.LatestBlockedBy` が見つかったときだけ出す。MODIFIED Requirement「本文領域は Issue 本文・最新 blocked-by の要約・コメント時系列を出す」の Scenario 2 件（blocked-by とコメントの手前に見出し行が出る / Issue 本文が空なら blocked-by の見出しを出さない）を通す
- [ ] 1.5 「そのセクションより上に本文領域の行が 1 行以上あるときにだけ出す」条件を入れる。ADDED Requirement の Scenario「本文領域の 1 行目には見出し行を出さない」を通す

## 2. checks の見出しと色

- [ ] 2.1 `checkLines`（`internal/ui/detail.go`）に checks の見出し行を足す。`StatusCheckRollup` が 1 件以上なら `classify.ChecksGreen` で `checks: 緑` / `checks: 緑以外`、0 件なら `checks: なし`、`MergeState` が nil なら `checks: 取得失敗` を、いずれも字下げ無しで出す。MODIFIED Requirement の Scenario 5 件（PR 131 の詳細 / 未 resolve の thread が先頭に出る / checks が無い merge 状態 / 全部の checks が成功なら見出しは緑になる / 詳細の取得に失敗した PR）を通す
- [ ] 2.2 各 check の行に、状態ごとの色を行全体へ付ける。緑にする集合は `Typename` ごとに `classify.ChecksGreen` と一致させる（`CheckRun` は `SUCCESS` / `SKIPPED` / `NEUTRAL`、`StatusContext` は `SUCCESS` だけ）。ADDED Requirement「checks の行は成功を緑・失敗を赤で出す」の Scenario 3 件（失敗した check の行だけが赤になる / StatusContext の SKIPPED は緑にしない / 色を落としても状態の語は読める）を通す。色の値は `internal/ui/view.go` の既存の定義と同じ場所にまとめる
- [ ] 2.3 2.2 の色の Scenario が使う PR fixture を `internal/ui/testdata/pr-status/pr-131.json` に置く（`internal/ui/testdata/merge` と同じ形）。`statusCheckRollup` に `SUCCESS` / `FAILURE` / `Conclusion` が空で `Status` が `IN_PROGRESS` / `SKIPPED` / `StatusContext` の `PENDING` を混ぜ、`body` の 1 行目を `未確定の判断: 0 件 — レビューをお願いします` にして issue #35 が挙げた見え方を再現する。`internal/gh/testdata/fixtures/example` は 10 以上のテストが共有しているので触らない

## 3. 既存テストの追随

- [ ] 3.1 `internal/ui/detail_test.go` の `TestPRDetailOfPR131` の期待順に見出し行と `checks: 緑以外` を足す
- [ ] 3.2 カード詳細と PR 詳細の他の既存テストが、見出し行を足した後も通ることを確認する

## 4. ドキュメント

- [ ] 4.1 README の「PR 詳細」の節（`README.md:142-144` 付近）に、本文領域がセクションの見出し行で区切られること、checks の行が成功=緑 / 失敗=赤で出ることを 1〜2 文で足す

## 5. 通し確認

- [ ] 5.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [ ] 5.2 `openspec validate --strict` が緑であることを確認する
