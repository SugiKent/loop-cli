package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestWrapToWidthKeepsShortStringOnOneLine(t *testing.T) {
	got := wrapToWidth("短いタイトル", 40)

	if len(got) != 1 || got[0] != "短いタイトル" {
		t.Errorf("wrapToWidth = %q, want 1 行の %q", got, "短いタイトル")
	}
}

func TestWrapToWidthSplitsByDisplayWidth(t *testing.T) {
	title := strings.Repeat("あ", 40) // 表示幅 80

	got := wrapToWidth(title, 40)

	if len(got) != 2 {
		t.Fatalf("行数 = %d, want 2: %q", len(got), got)
	}
	for i, l := range got {
		if w := ansi.StringWidth(l); w > 40 {
			t.Errorf("%d 行目の表示幅 = %d, want <= 40: %q", i+1, w, l)
		}
	}
	if joined := strings.Join(got, ""); joined != title {
		t.Errorf("連結 = %q, want %q", joined, title)
	}
}

// TestWrapToWidthDropsOnlyBreakSpaces は空白を含む文字列で、折り返し位置の空白 1 個以外は
// 失われないことを検証する（ansi.Wrap は改行に置き換えた空白を出力しない）。
func TestWrapToWidthDropsOnlyBreakSpaces(t *testing.T) {
	title := strings.Repeat("word ", 12) + "end" // 表示幅 63

	got := wrapToWidth(title, 20)

	if len(got) < 3 {
		t.Fatalf("行数 = %d, want 3 以上: %q", len(got), got)
	}
	for i, l := range got {
		if w := ansi.StringWidth(l); w > 20 {
			t.Errorf("%d 行目の表示幅 = %d, want <= 20: %q", i+1, w, l)
		}
	}
	strip := func(s string) string { return strings.ReplaceAll(s, " ", "") }
	if got, want := strip(strings.Join(got, "")), strip(title); got != want {
		t.Errorf("空白を除いた連結 = %q, want %q", got, want)
	}
}

// TestWrapToWidthReturnsOneLineForNarrowWidth は全角 1 文字が入らない幅では折らないことを
// 検証する。幅 1 の ansi.Wrap は先頭に空行を作り、各行の表示幅が 2 になってしまう。
func TestWrapToWidthReturnsOneLineForNarrowWidth(t *testing.T) {
	title := strings.Repeat("あ", 40)

	for _, w := range []int{0, 1} {
		got := wrapToWidth(title, w)
		if len(got) != 1 || got[0] != title {
			t.Errorf("幅 %d の wrapToWidth = %q, want 1 行の元の文字列", w, got)
		}
	}
}

func TestWrapTitleIndentsContinuationLines(t *testing.T) {
	prefix := "org/app #108  "       // 表示幅 14
	title := strings.Repeat("あ", 30) // 表示幅 60

	got := wrapTitle(prefix, title, 40)

	if len(got) < 2 {
		t.Fatalf("行数 = %d, want 2 以上: %q", len(got), got)
	}
	if !strings.HasPrefix(got[0], prefix) {
		t.Errorf("1 行目が接頭辞で始まらない: %q", got[0])
	}
	indent := strings.Repeat(" ", ansi.StringWidth(prefix))
	var joined string
	for i, l := range got {
		if w := ansi.StringWidth(l); w > 40 {
			t.Errorf("%d 行目の表示幅 = %d, want <= 40: %q", i+1, w, l)
		}
		if i == 0 {
			joined += strings.TrimPrefix(l, prefix)
			continue
		}
		if !strings.HasPrefix(l, indent) {
			t.Errorf("%d 行目が接頭辞ぶん字下げされていない: %q", i+1, l)
		}
		joined += strings.TrimPrefix(l, indent)
	}
	if joined != title {
		t.Errorf("接頭辞と字下げを剥がした連結 = %q, want %q", joined, title)
	}
}

func TestWrapTitleWithoutRoomForIndentDoesNotIndent(t *testing.T) {
	prefix := "org/app #108  " // 表示幅 14
	title := strings.Repeat("あ", 20)

	got := wrapTitle(prefix, title, 15)

	if len(got) < 2 {
		t.Fatalf("行数 = %d, want 2 以上: %q", len(got), got)
	}
	for i, l := range got {
		if w := ansi.StringWidth(l); w > 15 {
			t.Errorf("%d 行目の表示幅 = %d, want <= 15: %q", i+1, w, l)
		}
		if i > 0 && strings.HasPrefix(l, " ") {
			t.Errorf("字下げの余地が無いのに %d 行目が空白で始まる: %q", i+1, l)
		}
	}
}

func TestWrapTitleReturnsOneLineForZeroWidth(t *testing.T) {
	got := wrapTitle("org/app #108  ", strings.Repeat("あ", 30), 0)

	if len(got) != 1 || got[0] != "org/app #108  "+strings.Repeat("あ", 30) {
		t.Errorf("幅 0 の wrapTitle = %q, want 1 行の接頭辞 + タイトル", got)
	}
}
