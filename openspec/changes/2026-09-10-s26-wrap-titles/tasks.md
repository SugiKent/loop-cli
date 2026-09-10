# tasks: 2026-09-10-s26-wrap-titles

Refs #2

## 1. 折り返しヘルパー

- [ ] 1.1 `internal/ui` に `wrapTitle(prefix, title string, width int) []string` を足す。design.md D1〜D2 のとおり、タイトルを `width - StringWidth(prefix)` で `ansi.Wrap` に折らせ、1 行目に接頭辞を、2 行目以降に同じ表示幅の空白を前置する
- [ ] 1.2 `wrapTitle` のテストを書き、次を検証する。幅に収まるタイトルが 1 行のまま返る／全角 40 字のタイトルを幅 40 で折るとどの行も表示幅 40 以下になる／各行から接頭辞と字下げを剥がして連結すると元のタイトルと一致する／`width - StringWidth(prefix) < 1` の幅では字下げが付かない／`width` が 0 のとき 1 行で返る

## 2. PR 詳細ヘッダへの適用

- [ ] 2.1 `prHeaderLines` を `(lines []string, titleLines int)` を返す形に変え、1 行目を `wrapTitle` の戻り値に差し替える。`detailHeader` の呼び出し側も新しい戻り値に合わせる
- [ ] 2.2 Scenario「長い PR タイトルは折り返して全文出す」と「全角文字の PR タイトルは表示幅で折り返す」のテストを書き、タイトル行を接頭辞と字下げを剥がして連結した文字列が `Title` と一致することを検証する
- [ ] 2.3 既存の PR 詳細ヘッダのテストを labels 行の切り詰めに書き換え、Scenario「labels 行は今までどおり幅で切り詰める」が通ることを確認する

## 3. 高さの上限

- [ ] 3.1 `detailHeader` に design.md D4 の規則を足す。PR 一覧を落としても本文 1 行を確保できないとき、`titleLines` の範囲の行を末尾から落とし、残った最後のタイトル行を `ansi.Truncate(l, m.width-1, "") + "…"` で `…` 付きにする（1 行目は落とさない）
- [ ] 3.2 Scenario「折り返した PR タイトルが高さを埋めても画面は端末の高さに収まる」のテストを書き、幅 40・高さ 8 で `View` の行数がちょうど 8 になり、1 行目・labels 行・区切り線・フッタが残り、最後のタイトル行が `…` で終わることを検証する

## 4. 仕上げ

- [ ] 4.1 `openspec validate 2026-09-10-s26-wrap-titles --strict` が通ることを確認する
- [ ] 4.2 `go build ./... && go vet ./... && go test ./...` が通ることを確認する
