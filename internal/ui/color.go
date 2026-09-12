package ui

import (
	"math"
	"strconv"

	"charm.land/lipgloss/v2"
)

// labelTextThreshold は文字色を黒と白で切り替える相対輝度（design.md D1）。
// この点で黒と白のコントラスト比が等しくなるので、どの 16 進色でも 4.58:1 を下回らない。
const labelTextThreshold = 0.179

// stateColor は状態語の 1 色。端末の背景が暗いか明るいかで使う値を変える（design.md D3）。
// 背景を敷かない文字は端末の地の色に乗るので、1 組ではどちらかの端末で 4.5:1 を割る。
type stateColor struct{ dark, light string }

var (
	stateGreen  = stateColor{dark: "#3FB950", light: "#1A7F37"} // 良い
	stateRed    = stateColor{dark: "#F85149", light: "#CF222E"} // 悪い
	stateYellow = stateColor{dark: "#D29922", light: "#9A6700"} // 保留
	statePurple = stateColor{dark: "#A371F7", light: "#8250DF"} // merged
)

// stateColors は状態を表す語 -> 良し悪しの色。表に無い語（SKIPPED / NEUTRAL / CANCELLED を含む）
// には色を付けない。
var stateColors = map[string]stateColor{
	"MERGEABLE": stateGreen,
	"CLEAN":     stateGreen,
	"SUCCESS":   stateGreen,
	"緑":         stateGreen,
	"open":      stateGreen,

	"CONFLICTING":     stateRed,
	"DIRTY":           stateRed,
	"BLOCKED":         stateRed,
	"FAILURE":         stateRed,
	"ERROR":           stateRed,
	"TIMED_OUT":       stateRed,
	"STARTUP_FAILURE": stateRed,
	"緑以外":             stateRed,
	"closed":          stateRed,
	"取得失敗":            stateRed,

	"UNKNOWN":         stateYellow,
	"BEHIND":          stateYellow,
	"UNSTABLE":        stateYellow,
	"DRAFT":           stateYellow,
	"HAS_HOOKS":       stateYellow,
	"PENDING":         stateYellow,
	"QUEUED":          stateYellow,
	"IN_PROGRESS":     stateYellow,
	"WAITING":         stateYellow,
	"ACTION_REQUIRED": stateYellow,
	"EXPECTED":        stateYellow,

	"merged": statePurple,
}

// renderLabelName はラベル名を GitHub のラベル色を背景に、輝度で選んだ黒か白を文字にして描く。
// color は `#` の無い 16 進 6 桁。空・6 桁として読めない値では色を付けずに名前を返す。
// 名前の左右に空白や記号は足さないので、ANSI を除いた結果は name と一致する（design.md D2）。
func renderLabelName(name, color string) string {
	l, ok := luminance(color)
	if !ok {
		return name
	}
	text := "#FFFFFF"
	if l > labelTextThreshold {
		text = "#000000"
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#" + color)).
		Foreground(lipgloss.Color(text)).
		Render(name)
}

// renderStateWord は状態を表す語に良し悪しの文字色を付ける（背景色は付けない）。
// 表に無い語はそのまま返す。dark は端末の背景が暗いかどうか。
func renderStateWord(word string, dark bool) string {
	c, ok := stateColors[word]
	if !ok {
		return word
	}
	hex := c.light
	if dark {
		hex = c.dark
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(word)
}

// luminance は 16 進 6 桁の色の相対輝度（WCAG）。読めない値は ok=false。
func luminance(color string) (float64, bool) {
	if len(color) != 6 {
		return 0, false
	}
	n, err := strconv.ParseUint(color, 16, 32)
	if err != nil {
		return 0, false
	}
	lin := func(v uint64) float64 {
		c := float64(v) / 255
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(n>>16&0xff) + 0.7152*lin(n>>8&0xff) + 0.0722*lin(n&0xff), true
}

// labelName はそのリポジトリのラベル色の表から色を引いてラベル名を描く。
// 表に無いラベル（ListLabels が失敗したリポジトリ、取得後に増えたラベル）は色が付かない。
func (m Model) labelName(repo, name string) string {
	return renderLabelName(name, m.labelColors[repo][name])
}

// labelNames は複数のラベル名をそれぞれ色付きにして返す。
func (m Model) labelNames(repo string, names []string) []string {
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = m.labelName(repo, name)
	}
	return out
}

// stateWord は状態語を端末の背景に合わせた色で描く。
func (m Model) stateWord(word string) string {
	return renderStateWord(word, !m.lightBG)
}
