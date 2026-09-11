package claude

import (
	"errors"
	"strings"
	"testing"
)

func TestDecodeToolResult200(t *testing.T) {
	body, err := decodeToolResult(fixture(t, "get_run_log_200.jsonl"))
	if err != nil {
		t.Fatalf("decodeToolResult: %v", err)
	}
	if strings.Contains(body, "HTTP 200") {
		t.Errorf("本文に HTTP 200 の行が残っている:\n%s", body)
	}
	if !strings.HasPrefix(body, `{"session_id"`) {
		t.Errorf("本文がヘッダ行から始まっていない:\n%s", body)
	}
}

func TestDecodeToolResultIgnoresAssistantText(t *testing.T) {
	body, err := decodeToolResult(fixture(t, "get_run_log_200_with_assistant.jsonl"))
	if err != nil {
		t.Fatalf("decodeToolResult: %v", err)
	}
	if strings.Contains(body, "無視すべき地の文") {
		t.Errorf("assistant の文章が本文に混ざった:\n%s", body)
	}
	if !strings.Contains(body, "CI が緑になりました") {
		t.Errorf("tool_result の本文が返っていない:\n%s", body)
	}
}

func TestDecodeToolResult404(t *testing.T) {
	_, err := decodeToolResult(fixture(t, "get_run_log_404.jsonl"))
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("err = %v, want *HTTPError", err)
	}
	if httpErr.Status != 404 {
		t.Errorf("Status = %d, want 404", httpErr.Status)
	}
	if !strings.Contains(err.Error(), "not_found_error") {
		t.Errorf("エラー文字列 = %q", err.Error())
	}
}

func TestDecodeToolResultWeeklyLimit(t *testing.T) {
	_, err := decodeToolResult(fixture(t, "weekly_limit.txt"))
	var limitErr *LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("err = %v, want *LimitError", err)
	}
	if want := "Sep 14 at 2am (Asia/Tokyo)"; limitErr.Resets != want {
		t.Errorf("Resets = %q, want %q", limitErr.Resets, want)
	}
}

func TestDecodeToolResultNotLoggedIn(t *testing.T) {
	_, err := decodeToolResult(fixture(t, "not_logged_in.txt"))
	if !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("err = %v, want ErrNotLoggedIn", err)
	}
	var limitErr *LimitError
	if errors.As(err, &limitErr) {
		t.Error("未ログインが利用上限として判定された")
	}
}

func TestDecodeToolResultLaunchFailure(t *testing.T) {
	_, err := decodeToolResult([]byte("error: unknown flag --tools\n"))
	if err == nil {
		t.Fatal("エラーを期待した")
	}
	if !strings.Contains(err.Error(), "unknown flag --tools") {
		t.Errorf("標準出力の先頭がエラーに添えられていない: %v", err)
	}
}
