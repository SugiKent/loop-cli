## 1. セクションの見出し行

- [x] 1.1 `internal/ui` に見出し行を組み立てる関数を足す（`── <名前> ` の後を幅まで `─` で埋め、継ぎ足す本数は `max(幅 − 見出しの表示幅, 0)` で求め、収まらない幅では `ansi.Truncate` で切って `…` を付けない）。関数の戻り値を直接読むテストを書き、幅 100 / 幅 99 / 名前が入らない狭い幅 / 幅 0 の 4 通りを通す（`View` 越しでは viewport が各行を幅まで空白で埋めるので、幅の検証にならない）
- [x] 1.2 `prBodyLines`（`internal/ui/detail.go`）に `本文` / `コメント` / `review thread` の見出し行を入れる。`card-detail` の ADDED Requirement「本文領域のセクションは見出し行で区切る」の幅の Scenario 2 件（幅 100 の 1 ペイン / 幅 140 の 2 ペインで左 99。どちらも末尾の空白を落としてから幅と末尾の文字を見る）を通す
- [x] 1.3 中身が 0 行のセクションは見出しごと出さない条件を入れる。MODIFIED Requirement「PR 詳細は 1 行目判定・紐づけ・本文・会話・review thread・checks を出す」の Scenario「詳細の取得に失敗した PR」（`Body` が空なので `── 本文 ` が出ない）を通す
- [x] 1.4 `cardBodyLines`（同）に `blocked-by` / `コメント` の見出し行を入れる。`blocked-by` の見出しは `model.LatestBlockedBy` が見つかったときだけ出す。MODIFIED Requirement「本文領域は Issue 本文・最新 blocked-by の要約・コメント時系列を出す」の Scenario 2 件（blocked-by とコメントの手前に見出し行が出る / Issue 本文が空なら blocked-by の見出しを出さない）を通す
- [x] 1.5 「そのセクションより上に本文領域の行が 1 行以上あるときにだけ出す」条件を入れる。ADDED Requirement の Scenario「本文領域の 1 行目には見出し行を出さない」を通す

## 2. checks の見出し

- [x] 2.1 `checkLines`（`internal/ui/detail.go`）に checks の見出し行を足す。`StatusCheckRollup` が 1 件以上なら `classify.ChecksGreen` で `checks: 緑` / `checks: 緑以外`、0 件なら `checks: なし`、`MergeState` が nil なら `checks: 取得失敗` を、いずれも字下げ無しで出す。MODIFIED Requirement の Scenario 5 件（PR 131 の詳細 / 未 resolve の thread が先頭に出る / checks が無い merge 状態 / 全部の checks が成功なら見出しは緑になる / 詳細の取得に失敗した PR）を通す
- [x] 2.2 見出しの `緑` / `緑以外` を `m.stateWord` で塗る（`checks:` の見出しと `なし` は塗らない。#34 が入れた既存の状態語の扱いに合わせる）。MODIFIED Requirement「カード詳細と PR 詳細はラベル名と状態語に色を付ける」の Scenario「checks の見出しの値に色が付く」を通す

## 3. 既存テストの追随

- [x] 3.1 `internal/ui/detail_test.go` の `TestPRDetailOfPR131` の期待順に見出し行と `checks: 緑以外` を足す
- [x] 3.2 カード詳細と PR 詳細の他の既存テストが、見出し行を足した後も通ることを確認する（`──` で始まる行を数えるテストは、ヘッダとの区切り線だけを数える形に直す）

## 4. ドキュメント

- [x] 4.1 README の「PR 詳細」の節（`README.md:142-144` 付近）に、本文領域がセクションの見出し行で区切られること、checks の一覧に `checks: 緑` / `checks: 緑以外` の見出しが付くことを 1〜2 文で足す

## 5. 通し確認

- [x] 5.1 `gofmt -l .` の出力が空で、`go build ./... && go vet ./... && go test ./...` が通ることを確認する
- [x] 5.2 `openspec validate --strict` が緑であることを確認する
