package gh

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// recorder は Client の run を差し替え、受け取った引数と stdin を記録する。
type recorder struct {
	args  [][]string
	stdin []string
	// outs は呼び出し順に返す標準出力。足りなければ最後の要素を繰り返す。
	outs []string
	err  error
}

func (r *recorder) run(_ context.Context, stdin string, args ...string) ([]byte, error) {
	r.args = append(r.args, args)
	r.stdin = append(r.stdin, stdin)
	if r.err != nil {
		return nil, r.err
	}
	if len(r.outs) == 0 {
		return nil, nil
	}
	i := len(r.args) - 1
	if i >= len(r.outs) {
		i = len(r.outs) - 1
	}
	return []byte(r.outs[i]), nil
}

func newTestClient(rec *recorder) *Client {
	return &Client{run: rec.run, RetryWait: 0}
}

func (r *recorder) calls() int { return len(r.args) }

func wantArgs(t *testing.T, rec *recorder, want string) {
	t.Helper()
	if rec.calls() != 1 {
		t.Fatalf("実行回数 = %d, want 1", rec.calls())
	}
	if got := strings.Join(rec.args[0], " "); got != want {
		t.Errorf("引数 =\n  %s\nwant\n  %s", got, want)
	}
}

func TestSearchIssuesArgs(t *testing.T) {
	rec := &recorder{outs: []string{"[]"}}
	if _, err := newTestClient(rec).SearchIssues(t.Context(), []string{"org/app", "org/web"}); err != nil {
		t.Fatalf("SearchIssues: %v", err)
	}
	wantArgs(t, rec, "search issues --repo org/app --repo org/web --state open --limit 200 "+
		"--json repository,number,title,labels,updatedAt,url,body,commentsCount")
}

func TestSearchPRsArgs(t *testing.T) {
	rec := &recorder{outs: []string{"[]"}}
	if _, err := newTestClient(rec).SearchPRs(t.Context(), []string{"org/app"}); err != nil {
		t.Fatalf("SearchPRs: %v", err)
	}
	wantArgs(t, rec, "search prs --repo org/app --state open --limit 200 "+
		"--json repository,number,title,labels,updatedAt,url,body,isDraft")
}

func TestViewIssueArgs(t *testing.T) {
	rec := &recorder{outs: []string{`{"number":108}`}}
	got, err := newTestClient(rec).ViewIssue(t.Context(), "org/app", 108)
	if err != nil {
		t.Fatalf("ViewIssue: %v", err)
	}
	wantArgs(t, rec, "issue view 108 -R org/app --json number,title,body,url,labels,comments")
	if got.Number != 108 {
		t.Errorf("Number = %d, want 108", got.Number)
	}
}

func TestViewPRArgsAndNoRetry(t *testing.T) {
	rec := &recorder{outs: []string{`{"number":131,"isDraft":false,"mergeable":"UNKNOWN"}`}}
	got, err := newTestClient(rec).ViewPR(t.Context(), "org/app", 131)
	if err != nil {
		t.Fatalf("ViewPR: %v", err)
	}
	wantArgs(t, rec, "pr view 131 -R org/app --json number,title,body,url,labels,isDraft,comments")
	if got.Number != 131 {
		t.Errorf("Number = %d, want 131", got.Number)
	}
}

func TestViewPRMergeStateArgs(t *testing.T) {
	rec := &recorder{outs: []string{`{"mergeable":"MERGEABLE"}`}}
	if _, err := newTestClient(rec).ViewPRMergeState(t.Context(), "org/app", 131); err != nil {
		t.Fatalf("ViewPRMergeState: %v", err)
	}
	wantArgs(t, rec, "pr view 131 -R org/app --json mergeable,mergeStateStatus,statusCheckRollup,reviewDecision")
}

func TestViewPRMergeStateRetriesOnceOnUnknown(t *testing.T) {
	rec := &recorder{outs: []string{`{"mergeable":"UNKNOWN"}`, `{"mergeable":"MERGEABLE"}`}}
	got, err := newTestClient(rec).ViewPRMergeState(t.Context(), "org/app", 131)
	if err != nil {
		t.Fatalf("ViewPRMergeState: %v", err)
	}
	if rec.calls() != 2 {
		t.Errorf("実行回数 = %d, want 2", rec.calls())
	}
	if got.Mergeable != "MERGEABLE" {
		t.Errorf("Mergeable = %q, want MERGEABLE", got.Mergeable)
	}
}

func TestViewPRMergeStateStopsAfterSecondUnknown(t *testing.T) {
	rec := &recorder{outs: []string{`{"mergeable":"UNKNOWN"}`}}
	got, err := newTestClient(rec).ViewPRMergeState(t.Context(), "org/app", 131)
	if err != nil {
		t.Fatalf("ViewPRMergeState: %v", err)
	}
	if rec.calls() != 2 {
		t.Errorf("実行回数 = %d, want 2（3 回目は実行しない）", rec.calls())
	}
	if got.Mergeable != "UNKNOWN" {
		t.Errorf("Mergeable = %q, want UNKNOWN", got.Mergeable)
	}
}

func TestViewPRMergeStateNoRetryWhenMergeable(t *testing.T) {
	rec := &recorder{outs: []string{`{"mergeable":"MERGEABLE"}`, `{"mergeable":"CONFLICTING"}`}}
	got, err := newTestClient(rec).ViewPRMergeState(t.Context(), "org/app", 131)
	if err != nil {
		t.Fatalf("ViewPRMergeState: %v", err)
	}
	if rec.calls() != 1 {
		t.Errorf("実行回数 = %d, want 1", rec.calls())
	}
	if got.Mergeable != "MERGEABLE" {
		t.Errorf("Mergeable = %q", got.Mergeable)
	}
}

func TestReviewThreadsArgs(t *testing.T) {
	rec := &recorder{outs: []string{`{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[]}}}}}`}}
	if _, err := newTestClient(rec).ReviewThreads(t.Context(), "org/app", 131); err != nil {
		t.Fatalf("ReviewThreads: %v", err)
	}
	want := []string{"api", "graphql", "-f", "owner=org", "-f", "name=app", "-F", "number=131",
		"-f", "query=" + `query($owner:String!,$name:String!,$number:Int!){ repository(owner:$owner,name:$name){ pullRequest(number:$number){ reviewThreads(first:50){ nodes{ id isResolved comments(first:100){ nodes{ databaseId author{login} body createdAt } } } } } } }`}
	if rec.calls() != 1 {
		t.Fatalf("実行回数 = %d, want 1", rec.calls())
	}
	if !reflect.DeepEqual(rec.args[0], want) {
		t.Errorf("引数 =\n  %q\nwant\n  %q", rec.args[0], want)
	}
}

func TestCrossReferencedPRsArgs(t *testing.T) {
	rec := &recorder{outs: []string{`{"data":{"repository":{"issue":{"timelineItems":{"nodes":[]}}}}}`}}
	if _, err := newTestClient(rec).CrossReferencedPRs(t.Context(), "org/app", 108); err != nil {
		t.Fatalf("CrossReferencedPRs: %v", err)
	}
	want := []string{"api", "graphql", "-f", "owner=org", "-f", "name=app", "-F", "number=108",
		"-f", "query=" + `query($owner:String!,$name:String!,$number:Int!){ repository(owner:$owner,name:$name){ issue(number:$number){ timelineItems(first:50, itemTypes:[CROSS_REFERENCED_EVENT]){ nodes{ ... on CrossReferencedEvent { source { ... on PullRequest { number title body state labels(first:20){nodes{name}} } } } } } } } }`}
	if rec.calls() != 1 {
		t.Fatalf("実行回数 = %d, want 1", rec.calls())
	}
	if !reflect.DeepEqual(rec.args[0], want) {
		t.Errorf("引数 =\n  %q\nwant\n  %q", rec.args[0], want)
	}
}

// GraphQL クエリは gh に渡す前に構文として成立している必要がある（波括弧の対応）。
func TestGraphQLQueriesHaveBalancedBraces(t *testing.T) {
	for name, q := range map[string]string{
		"reviewThreadsQuery":      reviewThreadsQuery,
		"crossReferencedPRsQuery": crossReferencedPRsQuery,
	} {
		if open, close := strings.Count(q, "{"), strings.Count(q, "}"); open != close {
			t.Errorf("%s: { が %d 個、} が %d 個で対応していない", name, open, close)
		}
	}
}

func TestLabelTimelineArgsAndDecode(t *testing.T) {
	rec := &recorder{outs: []string{
		`{"created_at":"2026-09-01T00:00:00Z","event":"labeled","label":"stage:todo"}` +
			`{"created_at":"2026-09-02T00:00:00Z","event":"unlabeled","label":"stage:todo"}`,
	}}
	got, err := newTestClient(rec).LabelTimeline(t.Context(), "org/app", 108)
	if err != nil {
		t.Fatalf("LabelTimeline: %v", err)
	}
	wantArgs(t, rec, `api repos/org/app/issues/108/timeline --paginate --jq `+labelTimelineJQ)
	if len(got) != 2 {
		t.Fatalf("件数 = %d, want 2", len(got))
	}
	if got[0].Event != "labeled" || got[0].Label != "stage:todo" {
		t.Errorf("1 件目 = %+v", got[0])
	}
	if got[1].Event != "unlabeled" {
		t.Errorf("2 件目 = %+v", got[1])
	}
}

func TestCommentIssueArgsAndStdin(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).CommentIssue(t.Context(), "org/app", 108, "了解です"); err != nil {
		t.Fatalf("CommentIssue: %v", err)
	}
	wantArgs(t, rec, "issue comment 108 -R org/app --body-file -")
	if rec.stdin[0] != "了解です" {
		t.Errorf("stdin = %q", rec.stdin[0])
	}
}

func TestCommentPRArgsAndStdin(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).CommentPR(t.Context(), "org/app", 131, "Q1: A\nQ2: B"); err != nil {
		t.Fatalf("CommentPR: %v", err)
	}
	wantArgs(t, rec, "pr comment 131 -R org/app --body-file -")
	if rec.stdin[0] != "Q1: A\nQ2: B" {
		t.Errorf("stdin = %q", rec.stdin[0])
	}
}

func TestAddLabelPassesSingleLabel(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).AddLabel(t.Context(), "org/app", 108, "stage:todo"); err != nil {
		t.Fatalf("AddLabel: %v", err)
	}
	wantArgs(t, rec, "issue edit 108 -R org/app --add-label stage:todo")
}

func TestRemoveLabelPassesSingleLabel(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).RemoveLabel(t.Context(), "org/app", 108, "stage:todo"); err != nil {
		t.Fatalf("RemoveLabel: %v", err)
	}
	wantArgs(t, rec, "issue edit 108 -R org/app --remove-label stage:todo")
}

func TestMergePRUsesMethodFlag(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).MergePR(t.Context(), "org/app", 151, "squash"); err != nil {
		t.Fatalf("MergePR: %v", err)
	}
	wantArgs(t, rec, "pr merge 151 -R org/app --squash")
}

func TestCloseIssueArgs(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).CloseIssue(t.Context(), "org/app", 108); err != nil {
		t.Fatalf("CloseIssue: %v", err)
	}
	wantArgs(t, rec, "issue close 108 -R org/app")
	if rec.stdin[0] != "" {
		t.Errorf("stdin = %q, want 空", rec.stdin[0])
	}
}

func TestClosePRArgs(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).ClosePR(t.Context(), "org/app", 131); err != nil {
		t.Fatalf("ClosePR: %v", err)
	}
	wantArgs(t, rec, "pr close 131 -R org/app")
}

func TestCloseIssueNonZeroExitReturnsError(t *testing.T) {
	const stderr = "could not close issue"
	args := []string{"issue", "close", "108", "-R", "org/app"}
	rec := &recorder{err: &Error{Args: args, ExitCode: 1, Stderr: stderr}}

	err := newTestClient(rec).CloseIssue(t.Context(), "org/app", 108)
	if err == nil {
		t.Fatal("エラーが返らない")
	}
	for _, want := range []string{"issue close 108 -R org/app", "exit 1", stderr} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Error() = %q, want %q を含む", err.Error(), want)
		}
	}
}

func TestCreateIssueReturnsURL(t *testing.T) {
	rec := &recorder{outs: []string{"https://github.com/org/app/issues/200\n"}}
	got, err := newTestClient(rec).CreateIssue(t.Context(), "org/app", "タイトル", "本文")
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	wantArgs(t, rec, "issue create -R org/app --title タイトル --body-file -")
	if rec.stdin[0] != "本文" {
		t.Errorf("stdin = %q", rec.stdin[0])
	}
	if got != "https://github.com/org/app/issues/200" {
		t.Errorf("URL = %q", got)
	}
}

func TestCreateIssueLastLineNotURL(t *testing.T) {
	rec := &recorder{outs: []string{"Creating issue in org/app\n"}}
	_, err := newTestClient(rec).CreateIssue(t.Context(), "org/app", "タイトル", "本文")
	if err == nil {
		t.Fatal("エラーが返らない")
	}
	var ghErr *Error
	if errors.As(err, &ghErr) {
		t.Errorf("*Error になっている（gh は正常終了しているのでデコードエラーにする）: %v", err)
	}
}

func TestReplyReviewThreadArgsAndJSONStdin(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).ReplyReviewThread(t.Context(), "org/app", 88, 3935153121, "修正しました"); err != nil {
		t.Fatalf("ReplyReviewThread: %v", err)
	}
	wantArgs(t, rec, "api -X POST repos/org/app/pulls/88/comments/3935153121/replies --input -")
	if rec.stdin[0] != `{"body":"修正しました"}` {
		t.Errorf("stdin = %q", rec.stdin[0])
	}
}

func TestBrowseArgs(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).Browse(t.Context(), "org/app", 108); err != nil {
		t.Fatalf("Browse: %v", err)
	}
	wantArgs(t, rec, "browse 108 -R org/app")
}

func TestNonZeroExitReturnsError(t *testing.T) {
	const stderr = "GraphQL: Could not resolve to an issue or pull request with the number of 99999999. (repository.issue)"
	args := []string{"issue", "view", "99999999", "-R", "org/app", "--json", "number,title,body,url,labels,comments"}
	rec := &recorder{err: &Error{Args: args, ExitCode: 1, Stderr: stderr}}

	_, err := newTestClient(rec).ViewIssue(t.Context(), "org/app", 99999999)
	var ghErr *Error
	if !errors.As(err, &ghErr) {
		t.Fatalf("*Error ではない: %v", err)
	}
	if ghErr.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", ghErr.ExitCode)
	}
	if !strings.Contains(ghErr.Stderr, stderr) {
		t.Errorf("Stderr = %q", ghErr.Stderr)
	}
	msg := ghErr.Error()
	if !strings.Contains(msg, "issue view 99999999 -R org/app") || !strings.Contains(msg, stderr) {
		t.Errorf("Error() = %q", msg)
	}
}

func TestBrokenJSONIsNotGhError(t *testing.T) {
	rec := &recorder{outs: []string{"not json"}}
	_, err := newTestClient(rec).SearchIssues(t.Context(), []string{"org/app"})
	if err == nil {
		t.Fatal("エラーが返らない")
	}
	var ghErr *Error
	if errors.As(err, &ghErr) {
		t.Errorf("*Error になっている: %v", err)
	}
	if !strings.Contains(err.Error(), "search issues") {
		t.Errorf("エラー文字列に引数が無い: %v", err)
	}
}

func TestContextDeadlineExceeded(t *testing.T) {
	c := &Client{run: func(ctx context.Context, _ string, args ...string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	_, err := c.SearchIssues(ctx, []string{"org/app"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

// stubGHInPath は PATH に実行可能な空の gh を置く。LookPath は実行ビットを要求する。
func stubGHInPath(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("gh スタブの作成: %v", err)
	}
	t.Setenv("PATH", dir)
}

func TestCheckGhNotInPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	rec := &recorder{}
	err := newTestClient(rec).Check(t.Context())
	if err == nil {
		t.Fatal("エラーが返らない")
	}
	if !strings.Contains(err.Error(), "gh") {
		t.Errorf("エラー文字列に gh が無い: %v", err)
	}
	if rec.calls() != 0 {
		t.Errorf("gh が無いのに実行している: %v", rec.args)
	}
}

func TestCheckNotAuthenticated(t *testing.T) {
	stubGHInPath(t)
	const stderr = "X Failed to log in to github.com using token (GH_TOKEN)"
	rec := &recorder{err: &Error{Args: []string{"auth", "status"}, ExitCode: 1, Stderr: stderr}}

	err := newTestClient(rec).Check(t.Context())
	var ghErr *Error
	if !errors.As(err, &ghErr) {
		t.Fatalf("*Error ではない: %v", err)
	}
	if !strings.Contains(ghErr.Stderr, stderr) {
		t.Errorf("Stderr = %q", ghErr.Stderr)
	}
}

func TestCheckOK(t *testing.T) {
	stubGHInPath(t)
	rec := &recorder{}
	if err := newTestClient(rec).Check(t.Context()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	wantArgs(t, rec, "auth status")
}

// TestOpenURLRunsBrowserCommand は OpenURL がブラウザ起動コマンドに URL をそのまま渡し、
// gh を呼ばないことを検証する。
func TestOpenURLRunsBrowserCommand(t *testing.T) {
	var urls []string
	ghCalls := 0
	c := &Client{
		run: func(context.Context, string, ...string) ([]byte, error) {
			ghCalls++
			return nil, nil
		},
		openBrowser: func(_ context.Context, _, url string) (int, string, error) {
			urls = append(urls, url)
			return 0, "", nil
		},
	}

	if err := c.OpenURL(context.Background(), "https://example.com/a b"); err != nil {
		t.Fatalf("OpenURL: %v", err)
	}
	if len(urls) != 1 || urls[0] != "https://example.com/a b" {
		t.Errorf("渡した URL = %q, want [https://example.com/a b]", urls)
	}
	if ghCalls != 0 {
		t.Errorf("gh を %d 回呼んでいる, want 0", ghCalls)
	}
}

// TestBrowserCommandPerOS は OS ごとのコマンド名を検証する。
func TestBrowserCommandPerOS(t *testing.T) {
	if got := browserCommand("darwin"); got != "open" {
		t.Errorf("darwin = %q, want open", got)
	}
	if got := browserCommand("linux"); got != "xdg-open" {
		t.Errorf("linux = %q, want xdg-open", got)
	}
}

// TestOpenURLExitCodeError は終了コードが 0 でないときのエラー文字列を検証する。
func TestOpenURLExitCodeError(t *testing.T) {
	c := &Client{
		openBrowser: func(context.Context, string, string) (int, string, error) {
			return 1, "no browser\n", nil
		},
	}

	err := c.OpenURL(context.Background(), "https://example.com/a")
	if err == nil {
		t.Fatal("エラーが返っていない")
	}
	for _, want := range []string{"https://example.com/a", "exit 1", "no browser"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("エラー %q に %q が含まれない", err.Error(), want)
		}
	}
}

// TestRunBrowserMissingCommand は起動できないコマンドを error として返すことを検証する
// （OpenURL はこれをコマンド名と URL を含むエラーに包む）。
func TestRunBrowserMissingCommand(t *testing.T) {
	_, _, err := runBrowser(t.Context(), "loop-cli-no-such-command", "https://example.com/a")
	if err == nil {
		t.Fatal("起動できないコマンドでエラーが返っていない")
	}

	c := &Client{openBrowser: func(context.Context, string, string) (int, string, error) {
		return 0, "", err
	}}
	msg := c.OpenURL(t.Context(), "https://example.com/a").Error()
	for _, want := range []string{browserCommand(runtime.GOOS), "https://example.com/a"} {
		if !strings.Contains(msg, want) {
			t.Errorf("エラー %q に %q が含まれない", msg, want)
		}
	}
}

func TestListLabelsArgsAndDecode(t *testing.T) {
	rec := &recorder{outs: []string{
		`[{"name":"docs","description":".claude/ と docs/ だけの PR","color":"0075ca"},` +
			`{"name":"wip","description":"","color":"ededed"}]`,
	}}
	got, err := newTestClient(rec).ListLabels(t.Context(), "org/app")
	if err != nil {
		t.Fatalf("ListLabels: %v", err)
	}
	wantArgs(t, rec, "label list -R org/app --json name,description,color --sort name --order asc --limit 1000")

	want := []RepoLabel{
		{Name: "docs", Description: ".claude/ と docs/ だけの PR", Color: "0075ca"},
		{Name: "wip", Description: "", Color: "ededed"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ラベル = %+v, want %+v", got, want)
	}
}

func TestListLabelsDecodeErrorNamesCommand(t *testing.T) {
	rec := &recorder{outs: []string{"{"}}
	_, err := newTestClient(rec).ListLabels(t.Context(), "org/app")
	if err == nil {
		t.Fatal("エラーが返らない")
	}
	for _, want := range []string{"label list", "decode"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("エラー文字列に %q が無い: %v", want, err)
		}
	}
}

func TestEditIssueLabelsRepeatsFlagsInOneRun(t *testing.T) {
	rec := &recorder{}
	err := newTestClient(rec).EditIssueLabels(t.Context(), "org/app", 108,
		[]string{"docs", "wip"}, []string{"blocked"})
	if err != nil {
		t.Fatalf("EditIssueLabels: %v", err)
	}
	wantArgs(t, rec, "issue edit 108 -R org/app --add-label docs --add-label wip --remove-label blocked")
}

func TestEditPRLabelsUsesPREdit(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).EditPRLabels(t.Context(), "org/app", 131, []string{"docs"}, nil); err != nil {
		t.Fatalf("EditPRLabels: %v", err)
	}
	wantArgs(t, rec, "pr edit 131 -R org/app --add-label docs")
}

func TestEditIssueLabelsRemoveOnly(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).EditIssueLabels(t.Context(), "org/app", 108, nil, []string{"question"}); err != nil {
		t.Fatalf("EditIssueLabels: %v", err)
	}
	wantArgs(t, rec, "issue edit 108 -R org/app --remove-label question")
}

func TestEditIssueLabelsWithoutChangesDoesNotRunGh(t *testing.T) {
	rec := &recorder{}
	if err := newTestClient(rec).EditIssueLabels(t.Context(), "org/app", 108, nil, nil); err != nil {
		t.Fatalf("EditIssueLabels: %v", err)
	}
	if rec.calls() != 0 {
		t.Errorf("実行回数 = %d, want 0: %v", rec.calls(), rec.args)
	}
}

func TestEditPRLabelsReturnsGhFailure(t *testing.T) {
	rec := &recorder{err: &Error{Args: []string{"pr", "edit", "131"}, ExitCode: 1, Stderr: "HTTP 403"}}
	err := newTestClient(rec).EditPRLabels(t.Context(), "org/app", 131, []string{"docs"}, nil)
	if err == nil {
		t.Fatal("エラーが返らない")
	}
	for _, want := range []string{"pr edit", "exit 1", "HTTP 403"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("エラー文字列に %q が無い: %v", want, err)
		}
	}
}
