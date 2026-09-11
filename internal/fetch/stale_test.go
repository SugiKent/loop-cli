package fetch

import (
	"testing"
	"time"

	"github.com/SugiKent/loop-cli/internal/gh"
)

// PR 90（apply、最新コメントが人、updatedAt 2026-09-04T10:00:00Z）の分類は Fetch に渡した now で決まる。
func TestFetchPassesNowToClassify(t *testing.T) {
	for _, tt := range []struct {
		name string
		now  time.Time
		want string
	}{
		{"30 分後は auto-fix が受け取り中", time.Date(2026, 9, 4, 10, 30, 0, 0, time.UTC), "PR #90 は auto-fix が受け取り中"},
		{"1 日後は AI が応答していない", time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC), "PR #90 は人のコメントに AI が応答していない"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Fetch(t.Context(), gh.NewFake("testdata/link"), repos, tt.now)
			if err != nil {
				t.Fatalf("Fetch: %v", err)
			}
			if got := lonePR(t, res, 90).Result.Summary; got != tt.want {
				t.Errorf("PR 90 の Summary = %q, want %q", got, tt.want)
			}
		})
	}
}
