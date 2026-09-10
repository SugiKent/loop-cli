package gh

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const exampleDir = "testdata/fixtures/example"

func TestFakeSearchIssues(t *testing.T) {
	got, err := NewFake(exampleDir).SearchIssues(t.Context(), []string{"org/app"})
	if err != nil {
		t.Fatalf("SearchIssues: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("件数 = %d, want 2", len(got))
	}
	if got[0].Number != 108 || got[1].Number != 140 {
		t.Errorf("Number = %d, %d, want 108, 140", got[0].Number, got[1].Number)
	}
	if got[0].Repository.NameWithOwner != "org/app" {
		t.Errorf("NameWithOwner = %q", got[0].Repository.NameWithOwner)
	}
	if len(got[0].Labels) != 2 || got[0].Labels[0].Name != "stage:propose" || got[0].Labels[1].Name != "question" {
		t.Errorf("108 の Labels = %+v", got[0].Labels)
	}
	if len(got[1].Labels) != 0 {
		t.Errorf("140 の Labels = %+v, want 空", got[1].Labels)
	}
}

func TestFakeSearchPRs(t *testing.T) {
	got, err := NewFake(exampleDir).SearchPRs(t.Context(), []string{"org/app"})
	if err != nil {
		t.Fatalf("SearchPRs: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("件数 = %d, want 1", len(got))
	}
	if got[0].Number != 131 || got[0].IsDraft {
		t.Errorf("SearchPR = %+v", got[0])
	}
}

func TestFakeViewIssue(t *testing.T) {
	got, err := NewFake(exampleDir).ViewIssue(t.Context(), "org/app", 108)
	if err != nil {
		t.Fatalf("ViewIssue: %v", err)
	}
	if len(got.Comments) != 2 {
		t.Fatalf("コメント件数 = %d, want 2", len(got.Comments))
	}
	if !strings.HasPrefix(got.Comments[0].Body, "<!-- routine -->") {
		t.Errorf("1 件目の body = %q", got.Comments[0].Body)
	}
	if got.Comments[1].Author.Login != "user-2" {
		t.Errorf("2 件目の author = %q", got.Comments[1].Author.Login)
	}
}

func TestFakeViewPR(t *testing.T) {
	got, err := NewFake(exampleDir).ViewPR(t.Context(), "org/app", 131)
	if err != nil {
		t.Fatalf("ViewPR: %v", err)
	}
	if got.IsDraft {
		t.Errorf("IsDraft = true, want false")
	}
	if len(got.Comments) != 1 || got.Comments[0].Author.Login != "user-1" {
		t.Errorf("Comments = %+v", got.Comments)
	}
}

func TestFakeViewPRMergeStateKeepsUnknown(t *testing.T) {
	f := NewFake(exampleDir)
	got, err := f.ViewPRMergeState(t.Context(), "org/app", 131)
	if err != nil {
		t.Fatalf("ViewPRMergeState: %v", err)
	}
	if got.Mergeable != "UNKNOWN" {
		t.Errorf("Mergeable = %q, want UNKNOWN（Fake は再取得しない）", got.Mergeable)
	}
	if len(got.StatusCheckRollup) != 2 {
		t.Fatalf("statusCheckRollup 件数 = %d, want 2", len(got.StatusCheckRollup))
	}
	if got.StatusCheckRollup[0].Typename != "CheckRun" || got.StatusCheckRollup[0].Name != "test" {
		t.Errorf("CheckRun = %+v", got.StatusCheckRollup[0])
	}
	if got.StatusCheckRollup[1].Typename != "StatusContext" || got.StatusCheckRollup[1].Context != "ci/legacy" {
		t.Errorf("StatusContext = %+v", got.StatusCheckRollup[1])
	}
}

func TestFakeGraphQLAndTimelineFixtures(t *testing.T) {
	f := NewFake(exampleDir)

	threads, err := f.ReviewThreads(t.Context(), "org/app", 131)
	if err != nil {
		t.Fatalf("ReviewThreads: %v", err)
	}
	if len(threads) != 1 || len(threads[0].Comments) != 1 {
		t.Errorf("ReviewThreads = %+v", threads)
	}

	prs, err := f.CrossReferencedPRs(t.Context(), "org/app", 108)
	if err != nil {
		t.Fatalf("CrossReferencedPRs: %v", err)
	}
	if len(prs) != 1 || prs[0].Number != 131 {
		t.Errorf("CrossReferencedPRs = %+v（source が空のノードは無視する）", prs)
	}

	events, err := f.LabelTimeline(t.Context(), "org/app", 108)
	if err != nil {
		t.Fatalf("LabelTimeline: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("LabelTimeline 件数 = %d, want 2", len(events))
	}
}

// Fake と Client が同じデコード関数を通ることを、同じバイト列で確認する。
func TestFakeAndClientShareDecoding(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(exampleDir, "pr-131-review-threads.json"))
	if err != nil {
		t.Fatalf("fixture の読み込み: %v", err)
	}
	rec := &recorder{outs: []string{string(b)}}

	fromClient, err := newTestClient(rec).ReviewThreads(t.Context(), "org/app", 131)
	if err != nil {
		t.Fatalf("Client.ReviewThreads: %v", err)
	}
	fromFake, err := NewFake(exampleDir).ReviewThreads(t.Context(), "org/app", 131)
	if err != nil {
		t.Fatalf("Fake.ReviewThreads: %v", err)
	}
	if !reflect.DeepEqual(fromClient, fromFake) {
		t.Errorf("Client = %+v\nFake   = %+v", fromClient, fromFake)
	}
}

func TestFakeMissingFixture(t *testing.T) {
	_, err := NewFake(exampleDir).ViewIssue(t.Context(), "org/app", 999)
	if err == nil {
		t.Fatal("エラーが返らない")
	}
	if !strings.Contains(err.Error(), "issue-999.json") {
		t.Errorf("エラー文字列にパスが無い: %v", err)
	}
}

func TestFakeRecordsAddLabel(t *testing.T) {
	f := NewFake(exampleDir)
	if err := f.AddLabel(t.Context(), "org/app", 108, "stage:todo"); err != nil {
		t.Fatalf("AddLabel: %v", err)
	}
	want := []Call{{Method: "AddLabel", Repo: "org/app", Number: 108, Label: "stage:todo"}}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Errorf("Calls = %+v, want %+v", f.Calls, want)
	}
}

// 不変条件 3: stage:todo を外し、読み直してから stage:propose を付ける。
func TestFakeRecordsRemoveViewAddOrder(t *testing.T) {
	f := NewFake(exampleDir)
	ctx := t.Context()
	if err := f.RemoveLabel(ctx, "org/app", 108, "stage:todo"); err != nil {
		t.Fatalf("RemoveLabel: %v", err)
	}
	if _, err := f.ViewIssue(ctx, "org/app", 108); err != nil {
		t.Fatalf("ViewIssue: %v", err)
	}
	if err := f.AddLabel(ctx, "org/app", 108, "stage:propose"); err != nil {
		t.Fatalf("AddLabel: %v", err)
	}
	want := []Call{
		{Method: "RemoveLabel", Repo: "org/app", Number: 108, Label: "stage:todo"},
		{Method: "ViewIssue", Repo: "org/app", Number: 108},
		{Method: "AddLabel", Repo: "org/app", Number: 108, Label: "stage:propose"},
	}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Errorf("Calls = %+v, want %+v", f.Calls, want)
	}
}

func TestFakeRecordsCommentPRBody(t *testing.T) {
	f := NewFake(exampleDir)
	if err := f.CommentPR(t.Context(), "org/app", 131, "Q1: A"); err != nil {
		t.Fatalf("CommentPR: %v", err)
	}
	want := []Call{{Method: "CommentPR", Repo: "org/app", Number: 131, Body: "Q1: A"}}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Errorf("Calls = %+v, want %+v", f.Calls, want)
	}
}

func TestFakeCreateIssueReturnsPlaceholderURL(t *testing.T) {
	f := NewFake(exampleDir)
	url, err := f.CreateIssue(t.Context(), "org/app", "タイトル", "本文")
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if url != "https://github.com/org/app/issues/0" {
		t.Errorf("URL = %q", url)
	}
	want := []Call{{Method: "CreateIssue", Repo: "org/app", Title: "タイトル", Body: "本文"}}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Errorf("Calls = %+v, want %+v", f.Calls, want)
	}
}

func TestFakeDoesNotRecordOtherReads(t *testing.T) {
	f := NewFake(exampleDir)
	ctx := t.Context()
	if _, err := f.SearchIssues(ctx, []string{"org/app"}); err != nil {
		t.Fatalf("SearchIssues: %v", err)
	}
	if _, err := f.ViewPR(ctx, "org/app", 131); err != nil {
		t.Fatalf("ViewPR: %v", err)
	}
	if _, err := f.ViewPRMergeState(ctx, "org/app", 131); err != nil {
		t.Fatalf("ViewPRMergeState: %v", err)
	}
	if len(f.Calls) != 0 {
		t.Errorf("Calls = %+v, want 空", f.Calls)
	}
}

func TestFakeCallsConcurrent(t *testing.T) {
	f := NewFake(exampleDir)
	ctx := t.Context()
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.ViewIssue(ctx, "org/app", 108); err != nil {
				t.Errorf("ViewIssue: %v", err)
			}
		}()
	}
	wg.Wait()

	if len(f.Calls) != 8 {
		t.Fatalf("Calls 件数 = %d, want 8", len(f.Calls))
	}
	for i, c := range f.Calls {
		if c.Method != "ViewIssue" || c.Repo != "org/app" || c.Number != 108 {
			t.Errorf("Calls[%d] = %+v", i, c)
		}
	}
}

// TestFakeRecordsOpenURL は Fake が OpenURL の URL を記録することを検証する。
func TestFakeRecordsOpenURL(t *testing.T) {
	f := NewFake(exampleDir)

	if err := f.OpenURL(t.Context(), "https://example.com/design"); err != nil {
		t.Fatalf("OpenURL: %v", err)
	}

	want := Call{Method: "OpenURL", URL: "https://example.com/design"}
	if len(f.Calls) != 1 || !reflect.DeepEqual(f.Calls[0], want) {
		t.Fatalf("呼び出し = %+v, want [%+v]", f.Calls, want)
	}
}

// Fake は labels.json の並びをそのまま返す（gh の --sort に相当する並べ替えは行わない）。
func TestFakeListLabels(t *testing.T) {
	got, err := NewFake(exampleDir).ListLabels(t.Context(), "org/app")
	if err != nil {
		t.Fatalf("ListLabels: %v", err)
	}

	var names []string
	for _, l := range got {
		names = append(names, l.Name)
	}
	want := []string{
		"ai-assess:requested", "apply", "archive", "blocked", "docs", "p1", "propose",
		"question", "stage:apply", "stage:archive", "stage:propose", "stage:todo", "tui", "wip",
	}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("名前の並び = %v, want %v", names, want)
	}
	first := RepoLabel{
		Name: "ai-assess:requested", Description: "AI によるリスク評価を要求する PR", Color: "c5def5",
	}
	if len(got) > 0 && got[0] != first {
		t.Errorf("1 件目 = %+v, want %+v", got[0], first)
	}
}

func TestFakeMissingLabelsFixture(t *testing.T) {
	_, err := NewFake(filepath.Join("testdata", "fixtures")).ListLabels(t.Context(), "org/app")
	if err == nil {
		t.Fatal("エラーが返らない")
	}
	if !strings.Contains(err.Error(), "labels.json") {
		t.Errorf("エラー文字列にパスが無い: %v", err)
	}
}

func TestFakeRecordsListLabels(t *testing.T) {
	f := NewFake(exampleDir)
	ctx := t.Context()
	for range 2 {
		if _, err := f.ListLabels(ctx, "org/app"); err != nil {
			t.Fatalf("ListLabels: %v", err)
		}
	}
	want := []Call{
		{Method: "ListLabels", Repo: "org/app"},
		{Method: "ListLabels", Repo: "org/app"},
	}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Errorf("Calls = %+v, want %+v", f.Calls, want)
	}
}

func TestFakeRecordsEditIssueLabels(t *testing.T) {
	f := NewFake(exampleDir)
	err := f.EditIssueLabels(t.Context(), "org/app", 108, []string{"docs", "wip"}, []string{"blocked"})
	if err != nil {
		t.Fatalf("EditIssueLabels: %v", err)
	}
	want := []Call{{
		Method: "EditIssueLabels", Repo: "org/app", Number: 108,
		AddLabels: []string{"docs", "wip"}, RemoveLabels: []string{"blocked"},
	}}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Errorf("Calls = %+v, want %+v", f.Calls, want)
	}
}

// PR へのラベル書き込みは issue と別の Method で記録される（不変条件 4）。
func TestFakeRecordsEditPRLabels(t *testing.T) {
	f := NewFake(exampleDir)
	if err := f.EditPRLabels(t.Context(), "org/app", 131, []string{"docs"}, nil); err != nil {
		t.Fatalf("EditPRLabels: %v", err)
	}
	want := []Call{{Method: "EditPRLabels", Repo: "org/app", Number: 131, AddLabels: []string{"docs"}}}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Errorf("Calls = %+v, want %+v", f.Calls, want)
	}
}
