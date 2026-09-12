package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// helpKeys は実装済みのキーと動作。前半は mvp.md キーバインド表の順、後半は表に無いキー。
// 後続 change は自分のキーを実装したときにここへ 1 行足す。
var helpKeys = [][2]string{
	{"j / k / ↑ / ↓", "行移動（キュー）/ スクロール（詳細）"},
	{"1–4 / Tab", "タブ切替（今やる / バックログ / 進行中 / 異常）"},
	{"Enter", "詳細を開く / PR を開く（カード詳細）"},
	{"a", "回答・コメント（$EDITOR を開く）"},
	{"t", "stage:todo / To Do を付ける / 外す"},
	{"L", "ラベルを一覧から付け外し"},
	{"m", "PR を merge する（確認あり）"},
	{"c", "issue / PR を close する（確認あり）"},
	{"n", "選択中の repo に issue を作る（確認あり）"},
	{"o", "ブラウザで開く"},
	{"u", "URL 一覧を開く（セッション URL を含む）"},
	{"g", "PR ↔ issue を相互ジャンプ（詳細）"},
	{"R", "全件再取得（キュー）/ セッション取得（詳細）"},
	{"?", "ヘルプを開く / 閉じる"},
	{"q", "終了"},
	{"Esc", "1 つ前の画面に戻る（詳細 / ヘルプ）"},
	{"Tab", "PR 選択（カード詳細）"},
	{"x", "routine コメントの展開 / 折りたたみ（詳細）"},
	{"PgUp / PgDn", "ページ単位のスクロール（詳細）"},
	{"G / End", "本文の末尾へ飛ぶ（詳細）"},
	{"Home", "本文の先頭へ飛ぶ（詳細）"},
}

// helpLines はキーの行。キーの表記を表示幅 16 に揃える。
func helpLines() []string {
	lines := make([]string, 0, len(helpKeys))
	for _, k := range helpKeys {
		lines = append(lines, pad(k[0], 16)+k[1])
	}
	return lines
}

// updateHelpKey はヘルプ画面のキーを扱う（q / Ctrl+C は Update が先に処理する）。
func (m Model) updateHelpKey(key string) Model {
	if key != "?" && key != "esc" {
		return m
	}
	m.screen = m.helpFrom
	// ヘルプ中の WindowSizeMsg は詳細の寸法を作り直さないので、戻るときに作り直す。
	if m.screen == screenCard || m.screen == screenPR {
		m.refreshDetail()
	}
	return m
}

// renderHelp はヘルプ画面を 見出し / 空行 / キーの行 / 空行 / フッタ の順に描く。スクロールは持たない。
func (m Model) renderHelp() string {
	lines := append([]string{"キーバインド", ""}, helpLines()...)
	for len(lines) < m.height-1 {
		lines = append(lines, "")
	}
	lines = cut(lines, max(m.height-1, 0))
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "…")
	}
	return strings.Join(append(lines, m.footer("? / Esc 閉じる  q 終了")), "\n")
}
