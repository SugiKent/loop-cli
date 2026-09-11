package claude

import (
	"testing"
	"time"
)

func TestParseLogSplitsKindAndTool(t *testing.T) {
	body := `{"session_id": "session_01ABC", "events_fetched": 3, "events_shown": 3, "next_cursor": null}
[2026-09-10T12:03:00Z] tool_use Bash: git push -u origin HEAD
[2026-09-10T12:02:00Z] assistant: 変更を push します
[2026-09-10T12:01:00Z] env[info]: 環境を起動しました`

	got := ParseLog(body)
	want := []Entry{
		{At: time.Date(2026, 9, 10, 12, 3, 0, 0, time.UTC), Kind: "tool_use", Tool: "Bash", Text: "git push -u origin HEAD"},
		{At: time.Date(2026, 9, 10, 12, 2, 0, 0, time.UTC), Kind: "assistant", Text: "変更を push します"},
		{At: time.Date(2026, 9, 10, 12, 1, 0, 0, time.UTC), Kind: "env[info]", Text: "環境を起動しました"},
	}
	if len(got) != len(want) {
		t.Fatalf("件数 = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if !got[i].At.Equal(want[i].At) || got[i].Kind != want[i].Kind || got[i].Tool != want[i].Tool || got[i].Text != want[i].Text {
			t.Errorf("[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseLogDropsHeaderAndMalformedLines(t *testing.T) {
	body := `{"session_id": "session_01ABC", "events_fetched": 1, "events_shown": 1, "next_cursor": null}
[thinking]

[2026-09-10T12:03:00Z] init: 開始`

	got := ParseLog(body)
	if len(got) != 1 {
		t.Fatalf("件数 = %d, want 1: %+v", len(got), got)
	}
	if got[0].Kind != "init" {
		t.Errorf("Kind = %q, want init", got[0].Kind)
	}
}

func TestFinalAnswer(t *testing.T) {
	entries := ParseLog("[2026-09-10T12:05:00Z] result: success is_error=false turns=110 duration=1168s — CI が緑になりました")
	if len(entries) != 1 || entries[0].Kind != "result" {
		t.Fatalf("entries = %+v", entries)
	}
	got, ok := FinalAnswer(entries)
	if !ok || got != "CI が緑になりました" {
		t.Errorf("FinalAnswer = %q, %v; want CI が緑になりました, true", got, ok)
	}
}

func TestFinalAnswerAbsent(t *testing.T) {
	entries := ParseLog("[2026-09-10T12:05:00Z] tool_use Bash: go test ./...")
	if got, ok := FinalAnswer(entries); ok || got != "" {
		t.Errorf("FinalAnswer = %q, %v; want \"\", false", got, ok)
	}
}
