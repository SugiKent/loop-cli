package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// wrapToWidth は s を表示幅 w 以下の行に割る。w が 2 未満なら s の 1 行を返す。
// ansi.Wrap は単語境界を優先しつつ必要なら語の途中でも割るので、空白の無い日本語の
// タイトルでも幅どおりに折れる（design.md D1）。ただし全角 1 文字（幅 2）が入らない
// 幅では幅を守れず、幅 1 では先頭に空行を作って各行の表示幅が 2 になる（実測）。
// その幅では折らない。
//
// 折り返し位置にあった空白 1 個は改行に置き換わって出力から消える（ansi.Wrap の仕様）。
// 行を連結して元の文字列と比べるときは、空白を除いて比べる。
func wrapToWidth(s string, w int) []string {
	if w < 2 {
		return []string{s}
	}
	return strings.Split(ansi.Wrap(s, w, ""), "\n")
}

// wrapTitle は `<prefix><title>` を表示幅 width に収めた行に割り、2 行目以降を
// prefix の表示幅ぶん字下げしてタイトルの開始位置に縦を揃える（design.md D2）。
// 字下げの余地が無い幅（残りが 2 列未満。width が 2 未満の場合を含む）では
// 字下げを諦め、接頭辞ごと width で折る。
func wrapTitle(prefix, title string, width int) []string {
	indent := ansi.StringWidth(prefix)
	if width-indent < 2 {
		return wrapToWidth(prefix+title, width)
	}
	parts := wrapToWidth(title, width-indent)
	lines := make([]string, len(parts))
	for i, p := range parts {
		if i == 0 {
			lines[i] = prefix + p
			continue
		}
		lines[i] = strings.Repeat(" ", indent) + p
	}
	return lines
}
