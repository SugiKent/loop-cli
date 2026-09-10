# tasks: 2026-09-10-s27-wrap-titles

Refs #2

## 0. 前提

- [x] 0.1 先行 change `s26-issue-label-driven` が archive 済みであることを、`origin/main` の `openspec/specs/card-detail/spec.md` に `Options.Modes` と `model.IssueStages(mode, Issue.Labels)` の記述があることで確認する。まだなら着手せず、`blocked-by: change s26-issue-label-driven` を issue へ書き戻す（proposal の「先行 change への依存」）

## 1. 折り返しヘルパー

- [x] 1.1 `internal/ui` に `wrapToWidth(s string, w int) []string` を足す。design.md D1 のとおり `ansi.Wrap` で表示幅 `w` 以下の行に割り、`w` が 1 未満なら `s` の 1 行を返す
- [x] 1.2 `internal/ui` に `wrapTitle(prefix, title string, width int) []string` を足す。design.md D2 のとおり `wrapToWidth(title, width - StringWidth(prefix))` の結果へ、1 行目に接頭辞を、2 行目以降に同じ表示幅の空白を前置する
- [x] 1.3 この 2 つのテストを書き、次を検証する。幅に収まるタイトルが 1 行のまま返る／全角 40 字のタイトルを幅 40 で折るとどの行も表示幅 40 以下になる／各行から接頭辞と字下げを剥がして連結すると元のタイトルと一致する／`width - StringWidth(prefix) < 1` の幅では字下げが付かない／`width` が 0 のとき 1 行で返る

## 2. キュー画面の表

- [x] 2.1 `tableRow` を `[]string` を返す形に変え、design.md D3 のとおり 1 行目に固定列とタイトルの先頭断片と経過を、継続行に 43 列の空白とタイトルの続きと経過列ぶんの空白を置く。どの行も表示幅を端末幅ちょうどにし、継続行にも 1 行目と同じ色を付ける。`titleW <= 0` なら今までどおり 1 行だけ返す
- [x] 2.2 `tableLines` が各 Card の行を平らに並べてから `cut` に渡すよう直す
- [x] 2.3 Scenario「長いタイトルは折り返して全文出す」と「折り返した行はタイトルの開始位置に揃い、経過は 1 行目だけに出る」のテストを書き、各行の 43 列目以降を連結すると `Title` と一致すること、経過と `▶` が 1 行目にだけ出ることを検証する
- [x] 2.4 既存の `TestLongTitleIsTruncated` を Scenario「長いタイトルは切り詰める」（端末幅 48 以下でタイトルを出さない場合）に書き換え、Scenario「A の Card の行」「E の Card の行」「選択行に印が付く」のテストを新しい戻り値に合わせる

## 3. 詳細画面のヘッダ

- [x] 3.1 `cardHeaderLines` と `prHeaderLines` を `(lines []string, titleLines int)` を返す形に変え、1 行目を `wrapTitle` の戻り値に差し替える。`detailHeader` の呼び出し側も新しい戻り値に合わせる
- [x] 3.2 Scenario「長い Issue タイトルは折り返して全文出す」「長い PR タイトルは折り返して全文出す」「全角文字の PR タイトルは表示幅で折り返す」のテストを書き、タイトル行を接頭辞と字下げを剥がして連結した文字列が `Title` と一致することを検証する
- [x] 3.3 既存のヘッダのテストを Scenario「長いヘッダ行は幅で切り詰める」（`Summary` が対象）と「labels 行は今までどおり幅で切り詰める」に書き換える

## 4. 高さの上限

- [x] 4.1 `detailHeader` に design.md D5 の規則を足す。PR 一覧を落としても本文 1 行を確保できないとき、`titleLines` の範囲の行を末尾から落とし、残った最後のタイトル行を `ansi.Truncate(l, m.width-1, "") + "…"` で `…` 付きにする（1 行目は落とさない）
- [x] 4.2 Scenario「折り返した PR タイトルが高さを埋めても画面は端末の高さに収まる」のテストを書き、幅 40・高さ 8 で `View` の行数がちょうど 8 になり、1 行目・labels 行・区切り線・フッタが残り、最後のタイトル行が `…` で終わることを検証する

## 5. ドキュメント

- [x] 5.1 `docs/mvp/mvp.md` のキュー画面の画面図（51-73 行目あたり）を、長いタイトルが 2 行に折り返された形へ描き直す（design.md D6）

## 6. 仕上げ

- [x] 6.1 `openspec validate 2026-09-10-s27-wrap-titles --strict` が通ることを確認する
- [x] 6.2 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
