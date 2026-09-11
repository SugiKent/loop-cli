## 1. セクションの見出し行

- [ ] 1.1 `internal/ui` に見出し行を組み立てる関数を足す（`── <名前> ` の後を幅まで `─` で埋め、収まらない幅では `ansi.Truncate` で切って `…` を付けない）。`card-detail` の ADDED Requirement「本文領域のセクションは見出し行で区切る」の幅の Scenario 2 件（幅 100 の 1 ペイン / 幅 140 の 2 ペインで左 99）に対応するテストを `internal/ui` に書いて通す
- [ ] 1.2 `prBodyLines`（`internal/ui/detail.go`）に `本文` / `コメント` / `review thread` の見出し行を入れる。`card-detail` の MODIFIED Requirement「PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す」の Scenario「PR 131 の詳細」を、見出し行を含む並び順で通す
- [ ] 1.3 `cardBodyLines`（同）に `blocked-by` / `コメント` の見出し行を入れる。`blocked-by` の見出しは `model.LatestBlockedBy` が見つかったときだけ出す。MODIFIED Requirement「本文領域は Issue 本文・最新 blocked-by の要約・コメント時系列を出す」の Scenario「blocked-by とコメントの手前に見出し行が出る」を通す
- [ ] 1.4 「そのセクションより上に本文領域の行が 1 行以上あるときにだけ見出しを出す」条件を入れる。ADDED Requirement の Scenario「本文領域の 1 行目には見出し行を出さない」（`Body` が空・コメント 0 件・`blocked-by` 無しの issue）を通す

## 2. checks の見出しと色

- [ ] 2.1 `checkLines`（`internal/ui/detail.go`）に checks の見出し行を足す。`StatusCheckRollup` が 1 件以上なら `classify.ChecksGreen` で `checks: 緑` / `checks: 緑以外`、0 件なら `checks: なし`、`MergeState` が nil なら `checks: 取得失敗` を、いずれも字下げ無しで出す。MODIFIED Requirement の Scenario 4 件（PR 131 の詳細 / 未 resolve の thread が先頭に出る / checks が無い merge 状態 / 全部の checks が成功なら見出しは緑になる / 詳細の取得に失敗した PR）を通す
- [ ] 2.2 各 check の行に、状態ごとの色を行全体へ付ける（成功 3 語は緑 `#0E8A16`、失敗 6 語は赤 `#B60205`、それ以外は色を付けない）。ADDED Requirement「checks の行は成功を緑・失敗を赤で出す」の Scenario 2 件（失敗した check の行だけが赤になる / 色を落としても状態の語は読める）を通す。色の値は `internal/ui/view.go` の既存の定義と同じ場所にまとめる

## 3. 手元で見るための seed

- [ ] 3.1 `internal/ui/testdata/pr-status/pr-131.json` を足す（`internal/ui/testdata/merge` と同じ形の fixture）。`statusCheckRollup` に成功（`SUCCESS`）・失敗（`FAILURE`）・実行中（`Conclusion` が空で `Status` が `IN_PROGRESS`）・`SKIPPED`・`StatusContext` の `PENDING` を混ぜ、`body` の 1 行目を `未確定の判断: 0 件 — レビューをお願いします` にして issue #35 が挙げた見え方を再現する
- [ ] 3.2 3.1 の fixture から PR 詳細を描き、ANSI エスケープを付けたままの画面全体を `t.Log` に出すテストを `internal/ui` に書く。`go test ./internal/ui/ -run <その名前> -v` で、色と区切りが付いた実物の画面を手元で読めるようにする

## 4. 既存テストの追随

- [ ] 4.1 `internal/ui/detail_test.go` の既存テストのうち、`checks: なし` の字下げと本文の並びを読むもの（`TestPRDetailOfPR131` / `TestPRDetailReviewThreadsAndChecks` / `TestPRDetailWithEmptyChecksAndThreads` / `TestPRDetailFetchFailed`）を新しい並びで通す。行が増えて画面から溢れる分は、viewport のスクロールで読む（`internal/ui/detail_test.go` の既存の読み方に合わせる）
- [ ] 4.2 カード詳細の既存テスト（`internal/ui/detail_test.go` の本文・blocked-by・コメントを読むもの）を新しい並びで通す

## 5. ドキュメント

- [ ] 5.1 README の「PR 詳細」の節（`README.md:142-144` 付近）に、本文領域がセクションの見出し行で区切られること、checks の行が成功=緑 / 失敗=赤で出ることを 1〜2 文で足す
- [ ] 5.2 `docs/mvp` を触っていないことを `git diff origin/main --stat` で確認する（CLAUDE.md「`docs/mvp` はこれ以降更新しない」）

## 6. 通し確認

- [ ] 6.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [ ] 6.2 `openspec validate --strict` が緑であることを確認する
