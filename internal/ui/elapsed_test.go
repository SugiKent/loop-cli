package ui

import (
	"testing"
	"time"
)

func TestElapsed(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 4, 0, 0, time.UTC)
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"12 分前", now.Add(-12 * time.Minute), "12m"},
		{"3 時間前", now.Add(-3 * time.Hour), "3h"},
		{"2 日前", now.Add(-48 * time.Hour), "2d"},
		{"未来", now.Add(time.Minute), "0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Elapsed(now, tt.t); got != tt.want {
				t.Errorf("Elapsed = %q, want %q", got, tt.want)
			}
		})
	}
}
