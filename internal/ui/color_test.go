package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// 文字色の SGR。truecolor で書かれるので、どちらが選ばれたかはこの断片で見分けられる。
const (
	fgBlack = "38;2;0;0;0"
	fgWhite = "38;2;255;255;255"
)

func TestLabelTextColorFollowsLuminance(t *testing.T) {
	tests := []struct {
		color string
		want  string
		label string
	}{
		{"fbca04", fgBlack, "黒"}, // 明るい黄
		{"d876e3", fgBlack, "黒"}, // 明るい紫
		{"b60205", fgWhite, "白"}, // 暗い赤
		{"0075ca", fgWhite, "白"}, // 暗い青
	}
	for _, tt := range tests {
		got := renderLabelName("name", tt.color)
		if !strings.Contains(got, tt.want) {
			t.Errorf("%s の文字色が %s でない: %q", tt.color, tt.label, got)
		}
	}
}

func TestLabelNameGetsBackgroundColor(t *testing.T) {
	got := renderLabelName("propose", "0e8a16")

	if !strings.Contains(got, "48;2;14;138;22") {
		t.Errorf("背景色 0e8a16 が指定されていない: %q", got)
	}
}

// contrast は 2 つの相対輝度のコントラスト比。
func contrast(a, b float64) float64 {
	if a < b {
		a, b = b, a
	}
	return (a + 0.05) / (b + 0.05)
}

func TestStateColorsMeetAAOnTheirOwnBackground(t *testing.T) {
	for name, c := range map[string]stateColor{
		"緑": stateGreen, "赤": stateRed, "黄": stateYellow, "紫": statePurple,
	} {
		// 暗い端末の値は黒に対して、明るい端末の値は白に対して 4.5 以上であること。
		for _, tt := range []struct {
			hex  string
			bg   float64
			side string
		}{
			{c.dark, 0, "暗い端末（黒 #000000）"},
			{c.light, 1, "明るい端末（白 #FFFFFF）"},
		} {
			l, ok := luminance(strings.TrimPrefix(tt.hex, "#"))
			if !ok {
				t.Fatalf("%s の %s が 16 進 6 桁として読めない: %q", name, tt.side, tt.hex)
			}
			if ratio := contrast(l, tt.bg); ratio < 4.5 {
				t.Errorf("%s の %s のコントラスト比が %.2f で 4.5 未満: %q", name, tt.side, ratio, tt.hex)
			}
		}
	}
}

func TestStateWordFollowsTerminalBackground(t *testing.T) {
	if got := renderStateWord("MERGEABLE", true); !strings.Contains(got, "38;2;63;185;80") {
		t.Errorf("暗い端末の緑 #3FB950 でない: %q", got)
	}
	if got := renderStateWord("MERGEABLE", false); !strings.Contains(got, "38;2;26;127;55") {
		t.Errorf("明るい端末の緑 #1A7F37 でない: %q", got)
	}
}

func TestStateWordHasNoBackgroundColor(t *testing.T) {
	got := renderStateWord("MERGEABLE", true)

	if strings.Contains(got, "48;2;") {
		t.Errorf("状態語に背景色が付いている: %q", got)
	}
}

func TestUnreadableColorIsDrawnPlain(t *testing.T) {
	for _, color := range []string{"", "ffff", "ffffff0", "zzzzzz", "#0e8a16"} {
		if got := renderLabelName("propose", color); got != "propose" {
			t.Errorf("色 %q で色が付いた: %q", color, got)
		}
	}
}

func TestWordOutsideTheTableIsDrawnPlain(t *testing.T) {
	for _, word := range []string{"SKIPPED", "NEUTRAL", "CANCELLED", "mergeable", ""} {
		if got := renderStateWord(word, true); got != word {
			t.Errorf("表に無い語 %q に色が付いた: %q", word, got)
		}
	}
}

func TestStrippingANSIGivesBackTheInput(t *testing.T) {
	if got := ansi.Strip(renderLabelName("stage:propose", "0e8a16")); got != "stage:propose" {
		t.Errorf("ラベル名の表示テキストが変わった: %q", got)
	}
	if got := ansi.Strip(renderStateWord("CONFLICTING", true)); got != "CONFLICTING" {
		t.Errorf("状態語の表示テキストが変わった: %q", got)
	}
}
