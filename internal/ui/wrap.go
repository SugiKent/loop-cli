package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// wrapToWidth は s を表示幅 w 以下の行に割る。w が 1 未満なら s の 1 行を返す。
// ansi.Wrap は単語境界を優先しつつ必要なら語の途中でも割るので、空白の無い日本語の
// タイトルでも幅どおりに折れる（design.md D1）。
func wrapToWidth(s string, w int) []string {
	if w < 1 {
		return []string{s}
	}
	return strings.Split(ansi.Wrap(s, w, ""), "\n")
}

// wrapTitle は `<prefix><title>` を表示幅 width に収めた行に割り、2 行目以降を
// prefix の表示幅ぶん字下げしてタイトルの開始位置に縦を揃える（design.md D2）。
func wrapTitle(prefix, title string, width int) []string {
	indent := ansi.StringWidth(prefix)
	// 字下げの余地が無い幅では字下げを諦め、接頭辞ごと width で折る。
	if width < 1 || width-indent < 1 {
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
