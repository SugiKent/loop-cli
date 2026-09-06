package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/model"
)

// wide は 2 ペインが崩れない大きさで example を読み込んだ Model を返す。
func wide(t *testing.T) Model {
	t.Helper()
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, fetchedMsg{res: exampleResult(t), at: at})
	return m
}

func TestPreviewOfQuestionCard(t *testing.T) {
	lines := plain(wide(t))

	first, ok := lineWith(lines, "issue #108 の提案。")
	if !ok {
		t.Fatalf("プレビューの 1 行目が無い: %q", lines)
	}
	if !strings.Contains(first, "labels: propose question") {
		t.Errorf("1 行目に labels が無い: %q", first)
	}
	if _, ok := lineWith(lines, "▌AI"); !ok {
		t.Errorf("AI コメントの見出しに ▌ が無い: %q", lines)
	}
	comment, ok := lineWith(lines, "Q1: マイグレーションを分けますか。")
	if !ok {
		t.Fatalf("AI コメントの本文が無い: %q", lines)
	}
	if !strings.HasPrefix(comment, "▌") {
		t.Errorf("AI コメントの本文に ▌ が無い: %q", comment)
	}

	text := strings.Join(lines, "\n")
	for _, ng := range []string{"PR #131 の質問に答える", "<!-- routine -->", "&lt;!-- routine --&gt;"} {
		if strings.Contains(text, ng) {
			t.Errorf("プレビューに %q が出ている", ng)
		}
	}
}

func TestPreviewOfHumanCommentHasNoBar(t *testing.T) {
	card := nowCard("org/app", 1, 1, at)
	card.Issue.Body = "本文"
	card.Issue.Comments = []model.Comment{{Author: "user-2", Body: "Q1: A", CreatedAt: at, AI: false}}
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40},
		fetchedMsg{res: &fetch.Result{Cards: []model.Card{card}}, at: at})

	lines := plain(m)

	head, ok := lineWith(lines, "user-2")
	if !ok {
		t.Fatalf("人のコメントの見出しが無い: %q", lines)
	}
	body, ok := lineWith(lines, "Q1: A")
	if !ok {
		t.Fatalf("人のコメントの本文が無い: %q", lines)
	}
	if strings.HasPrefix(head, "▌") || strings.HasPrefix(body, "▌") {
		t.Errorf("人のコメントに ▌ が付いた: %q / %q", head, body)
	}
}

func TestPreviewOfCardWithoutLabelsAndComments(t *testing.T) {
	m, _ := send(wide(t), runeKey('2'))

	text := plainText(m)

	if !strings.Contains(text, "起動時に設定ファイルが無いと落ちる") {
		t.Errorf("プレビューに本文が無い: %q", text)
	}
	if strings.Contains(text, "labels:") {
		t.Errorf("Labels が空なのに labels: が出た: %q", text)
	}
	if strings.Contains(text, "▌") {
		t.Errorf("コメントが無いのに ▌ が出た: %q", text)
	}
}

func TestPreviewOfEmptyTab(t *testing.T) {
	m, _ := send(wide(t), runeKey('4'))

	if !strings.Contains(plainText(m), "（このタブにはカードがありません）") {
		t.Errorf("0 行のタブの案内が無い: %q", plainText(m))
	}
}

func TestStripMarkersRemovesOnlyMarkerLines(t *testing.T) {
	body := "<!-- routine -->\nQ1: A\n  &lt;!-- routine --&gt;  \n<!-- routine --> を含む行"

	got := stripMarkers(body)

	want := "Q1: A\n<!-- routine --> を含む行"
	if got != want {
		t.Errorf("stripMarkers = %q, want %q", got, want)
	}
}
