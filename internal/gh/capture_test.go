package gh

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// captureRunner は引数列で応答を切り替える run。recorder と違い、
// 「pr view だけ 2 回目の出力を変える」「特定のコマンドだけ失敗させる」ができる。
type captureRunner struct {
	args [][]string
	// outs は引数列（空白連結）の前方一致 → 呼び出し順に返す標準出力。
	outs map[string][]string
	// fail は前方一致したコマンドを *Error で失敗させる。
	fail string
}

func (r *captureRunner) run(_ context.Context, _ string, args ...string) ([]byte, error) {
	joined := strings.Join(args, " ")
	r.args = append(r.args, args)
	if r.fail != "" && strings.HasPrefix(joined, r.fail) {
		return nil, &Error{Args: args, ExitCode: 1, Stderr: "boom"}
	}
	for prefix, outs := range r.outs {
		if !strings.HasPrefix(joined, prefix) {
			continue
		}
		n := 0
		for _, prev := range r.args {
			if strings.HasPrefix(strings.Join(prev, " "), prefix) {
				n++
			}
		}
		if n > len(outs) {
			n = len(outs)
		}
		return []byte(outs[n-1]), nil
	}
	return []byte("{}"), nil
}

func (r *captureRunner) joined() []string {
	out := make([]string, 0, len(r.args))
	for _, a := range r.args {
		out = append(out, strings.Join(a, " "))
	}
	return out
}

// oneIssueOnePR は issue 108 と PR 131 が 1 件ずつ見つかる状態を作る。
func oneIssueOnePR(mergeables ...string) *captureRunner {
	prOuts := make([]string, 0, len(mergeables))
	for _, m := range mergeables {
		prOuts = append(prOuts, `{"number":131,"mergeable":"`+m+`"}`)
	}
	return &captureRunner{outs: map[string][]string{
		"search issues": {`[{"number":108}]`},
		"search prs":    {`[{"number":131}]`},
		"pr view 131":   prOuts,
	}}
}

func TestCaptureFilesAndProgressOrder(t *testing.T) {
	rec := oneIssueOnePR("MERGEABLE")
	c := &Client{run: rec.run, RetryWait: 0}

	var seen []string
	files, err := c.Capture(t.Context(), "org/app", func(name string) { seen = append(seen, name) })
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}

	want := []string{
		"search-issues.json",
		"issue-108.json", "issue-108-cross-refs.json", "issue-108-timeline.json",
		"search-prs.json",
		"pr-131.json", "pr-131-review-threads.json",
	}
	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Errorf("progress の順序 = %v, want %v", seen, want)
	}
	if len(files) != len(want) {
		t.Fatalf("ファイル数 = %d, want %d: %v", len(files), len(want), files)
	}
	for _, name := range want {
		if _, ok := files[name]; !ok {
			t.Errorf("キー %s が無い", name)
		}
	}
	if got := string(files["search-issues.json"]); got != `[{"number":108}]` {
		t.Errorf("search-issues.json = %q", got)
	}
	if got := string(files["pr-131.json"]); got != `{"number":131,"mergeable":"MERGEABLE"}` {
		t.Errorf("pr-131.json = %q", got)
	}
}

func TestCaptureArgsMatchReadMethods(t *testing.T) {
	rec := oneIssueOnePR("MERGEABLE")
	c := &Client{run: rec.run, RetryWait: 0}
	if _, err := c.Capture(t.Context(), "org/app", func(string) {}); err != nil {
		t.Fatalf("Capture: %v", err)
	}

	want := []string{
		"search issues --repo org/app --state open --limit 200 " +
			"--json repository,number,title,labels,updatedAt,url,body,commentsCount",
		"issue view 108 -R org/app --json number,title,body,url,labels,comments",
		strings.Join(argsCrossReferencedPRs("org/app", 108), " "),
		strings.Join(argsLabelTimeline("org/app", 108), " "),
		"search prs --repo org/app --state open --limit 200 " +
			"--json repository,number,title,labels,updatedAt,url,body,isDraft",
		"pr view 131 -R org/app --json number,title,body,url,labels,isDraft,comments," +
			"mergeable,mergeStateStatus,statusCheckRollup,reviewDecision",
		strings.Join(argsReviewThreads("org/app", 131), " "),
	}
	got := rec.joined()
	if len(got) != len(want) {
		t.Fatalf("実行回数 = %d, want %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%d 番目の引数 =\n  %s\nwant\n  %s", i, got[i], want[i])
		}
	}
	// GraphQL の引数が期待した形であることを直接確かめる。
	if !strings.Contains(got[6], "-f owner=org -f name=app -F number=131") {
		t.Errorf("review-threads の引数 = %s", got[6])
	}
}

func TestCaptureRetriesPRViewOnUnknown(t *testing.T) {
	rec := oneIssueOnePR("UNKNOWN", "MERGEABLE")
	c := &Client{run: rec.run, RetryWait: 0}

	var seen []string
	files, err := c.Capture(t.Context(), "org/app", func(name string) { seen = append(seen, name) })
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if got := string(files["pr-131.json"]); got != `{"number":131,"mergeable":"MERGEABLE"}` {
		t.Errorf("pr-131.json = %q, want 2 回目の出力", got)
	}
	prViews := 0
	for _, a := range rec.joined() {
		if strings.HasPrefix(a, "pr view 131") {
			prViews++
		}
	}
	if prViews != 2 {
		t.Errorf("pr view の実行回数 = %d, want 2", prViews)
	}
	if strings.Count(strings.Join(seen, ","), "pr-131.json") != 1 {
		t.Errorf("progress は pr-131.json を 1 回だけ呼ぶ: %v", seen)
	}
}

func TestCaptureNoRetryWhenMergeable(t *testing.T) {
	rec := oneIssueOnePR("MERGEABLE", "UNKNOWN")
	c := &Client{run: rec.run, RetryWait: 0}
	if _, err := c.Capture(t.Context(), "org/app", func(string) {}); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	prViews := 0
	for _, a := range rec.joined() {
		if strings.HasPrefix(a, "pr view 131") {
			prViews++
		}
	}
	if prViews != 1 {
		t.Errorf("pr view の実行回数 = %d, want 1", prViews)
	}
}

func TestCaptureEmptyRepo(t *testing.T) {
	rec := &captureRunner{outs: map[string][]string{
		"search issues": {"[]"},
		"search prs":    {"[]"},
	}}
	c := &Client{run: rec.run, RetryWait: 0}

	files, err := c.Capture(t.Context(), "org/app", func(string) {})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if len(files) != 2 || files["search-issues.json"] == nil || files["search-prs.json"] == nil {
		t.Errorf("files = %v, want search 2 件だけ", files)
	}
	if len(rec.args) != 2 {
		t.Errorf("実行回数 = %d, want 2", len(rec.args))
	}
}

func TestCaptureStopsOnFailure(t *testing.T) {
	rec := oneIssueOnePR("MERGEABLE")
	rec.fail = "issue view 108"
	c := &Client{run: rec.run, RetryWait: 0}

	_, err := c.Capture(t.Context(), "org/app", func(string) {})
	var ghErr *Error
	if !errors.As(err, &ghErr) {
		t.Fatalf("エラー = %v, want *gh.Error", err)
	}
	got := rec.joined()
	if len(got) != 2 {
		t.Fatalf("実行回数 = %d, want 2（search issues と失敗した issue view）: %v", len(got), got)
	}
}
