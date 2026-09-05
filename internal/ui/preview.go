package ui

import (
	"slices"
	"strings"
	"time"

	"charm.land/glamour/v2"

	"github.com/SugiKent/sugi-loop/internal/model"
)

// routineMarkers は落とすマーカー行。エスケープ済みの形はレンダラの出力に混ざるので両方見る。
var routineMarkers = []string{"<!-- routine -->", "&lt;!-- routine --&gt;"}

// previewLines は選択行の 1 行目・本文・コメントを h 行ぶん返す。
func (m Model) previewLines(h int) []string {
	rows := m.rows[m.tab]
	if m.cursor >= len(rows) {
		return cut([]string{"（このタブにはカードがありません）"}, h)
	}
	r := rows[m.cursor]

	var lines []string
	if first := firstLine(r.body); first != "" {
		if len(r.labels) > 0 {
			first += "   labels: " + strings.Join(r.labels, " ")
		}
		lines = append(lines, first)
	}
	if r.body != "" {
		lines = append(lines, renderMarkdown(r.body, m.width)...)
	}
	lines = append(lines, m.commentLines(r.comments)...)
	return cut(lines, h)
}

// commentLines はコメントを並び順に描く。AI 発は見出しと本文の全行に ▌ を付ける。
func (m Model) commentLines(comments []model.Comment) []string {
	var lines []string
	for _, c := range comments {
		at := c.CreatedAt.In(m.location()).Format("15:04")
		body := renderMarkdown(stripMarkers(c.Body), m.width-1)
		if c.AI {
			lines = append(lines, "▌AI  "+at)
			for _, l := range body {
				lines = append(lines, "▌"+l)
			}
			continue
		}
		lines = append(lines, c.Author+"  "+at)
		lines = append(lines, body...)
	}
	return lines
}

// location は時刻を書く基準のタイムゾーン。取得完了時刻に合わせる。
func (m Model) location() *time.Location {
	if m.at.IsZero() {
		return time.Local
	}
	return m.at.Location()
}

// firstLine は先頭の空行を除いた最初の行を返す。
func firstLine(body string) string {
	for _, l := range strings.Split(body, "\n") {
		if strings.TrimSpace(l) != "" {
			return strings.TrimRight(l, "\r")
		}
	}
	return ""
}

// stripMarkers は routine マーカーだけの行を落とす。部分一致の行は触らない。
func stripMarkers(body string) string {
	lines := strings.Split(body, "\n")
	kept := make([]string, 0, len(lines))
	for _, l := range lines {
		if slices.Contains(routineMarkers, strings.TrimSpace(l)) {
			continue
		}
		kept = append(kept, l)
	}
	return strings.Join(kept, "\n")
}

// renderMarkdown は Glamour で幅 w にレンダリングする。失敗したら入力をそのまま行に割る。
// Glamour v2 は端末の明暗の自動判定を持たないので、GLAMOUR_STYLE（既定 dark）に従う。
func renderMarkdown(s string, w int) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	if w < 1 {
		w = 1
	}
	r, err := glamour.NewTermRenderer(glamour.WithEnvironmentConfig(), glamour.WithWordWrap(w))
	if err != nil {
		return strings.Split(s, "\n")
	}
	out, err := r.Render(s)
	if err != nil {
		return strings.Split(s, "\n")
	}
	return strings.Split(strings.Trim(out, "\n"), "\n")
}
