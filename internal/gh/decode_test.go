package gh

import (
	"testing"
	"time"
)

func TestDecodeSearchIssues(t *testing.T) {
	const in = `[{"body":"b1","commentsCount":2,"labels":[{"id":"LA_1","name":"stage:propose","color":"0e8a16","description":"d"}],"number":108,"repository":{"name":"app","nameWithOwner":"org/app"},"title":"t1","updatedAt":"2026-09-04T09:12:33Z","url":"https://github.com/org/app/issues/108"}]`

	got, err := decodeSearchIssues([]byte(in))
	if err != nil {
		t.Fatalf("decodeSearchIssues: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("件数 = %d, want 1", len(got))
	}
	is := got[0]
	if is.Repository.NameWithOwner != "org/app" || is.Repository.Name != "app" {
		t.Errorf("Repository = %+v", is.Repository)
	}
	if is.Number != 108 || is.Title != "t1" || is.Body != "b1" || is.CommentsCount != 2 {
		t.Errorf("SearchIssue = %+v", is)
	}
	if is.URL != "https://github.com/org/app/issues/108" {
		t.Errorf("URL = %q", is.URL)
	}
	if len(is.Labels) != 1 || is.Labels[0].Name != "stage:propose" {
		t.Errorf("Labels = %+v", is.Labels)
	}
	if want := time.Date(2026, 9, 4, 9, 12, 33, 0, time.UTC); !is.UpdatedAt.Equal(want) {
		t.Errorf("UpdatedAt = %v, want %v", is.UpdatedAt, want)
	}
}

func TestDecodeSearchPRs(t *testing.T) {
	const in = `[{"body":"b","isDraft":true,"labels":[{"name":"propose"}],"number":131,"repository":{"name":"app","nameWithOwner":"org/app"},"title":"t","updatedAt":"2026-09-04T10:30:00Z","url":"u"}]`

	got, err := decodeSearchPRs([]byte(in))
	if err != nil {
		t.Fatalf("decodeSearchPRs: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("件数 = %d, want 1", len(got))
	}
	if !got[0].IsDraft {
		t.Errorf("IsDraft = false, want true")
	}
	if got[0].Number != 131 || got[0].Repository.NameWithOwner != "org/app" {
		t.Errorf("SearchPR = %+v", got[0])
	}
}

func TestDecodeIssueDetail(t *testing.T) {
	const in = `{"number":108,"title":"t","body":"b","url":"u","labels":[{"name":"question"}],"comments":[{"id":"IC_1","author":{"login":"alice"},"authorAssociation":"OWNER","body":"c1","createdAt":"2026-09-04T09:00:00Z","reactionGroups":[],"url":"cu","viewerDidAuthor":true}]}`

	got, err := decodeIssueDetail([]byte(in))
	if err != nil {
		t.Fatalf("decodeIssueDetail: %v", err)
	}
	if got.Number != 108 || len(got.Labels) != 1 || got.Labels[0].Name != "question" {
		t.Errorf("IssueDetail = %+v", got)
	}
	if len(got.Comments) != 1 {
		t.Fatalf("コメント件数 = %d, want 1", len(got.Comments))
	}
	c := got.Comments[0]
	if c.ID != "IC_1" || c.Author.Login != "alice" || c.Body != "c1" || c.URL != "cu" {
		t.Errorf("Comment = %+v", c)
	}
}

func TestDecodePRDetail(t *testing.T) {
	const in = `{"number":131,"title":"t","body":"b","url":"u","labels":[{"name":"propose"}],"isDraft":false,"comments":[{"id":"IC_2","author":{"login":"bot"},"body":"<!-- routine -->\nQ1","createdAt":"2026-09-04T10:31:00Z","url":"cu"}],"mergeable":"UNKNOWN"}`

	got, err := decodePRDetail([]byte(in))
	if err != nil {
		t.Fatalf("decodePRDetail: %v", err)
	}
	if got.IsDraft {
		t.Errorf("IsDraft = true, want false")
	}
	if len(got.Comments) != 1 || got.Comments[0].Author.Login != "bot" {
		t.Errorf("Comments = %+v", got.Comments)
	}
}

func TestDecodePRMergeStateStatusCheckRollup(t *testing.T) {
	const in = `{"mergeable":"UNKNOWN","mergeStateStatus":"BLOCKED","reviewDecision":"REVIEW_REQUIRED","statusCheckRollup":[{"__typename":"CheckRun","name":"test","status":"COMPLETED","conclusion":"SUCCESS","workflowName":"CI","detailsUrl":"du","startedAt":"2026-09-04T10:35:00Z","completedAt":"2026-09-04T10:40:00Z"},{"__typename":"StatusContext","context":"ci/legacy","state":"PENDING","targetUrl":"tu"}]}`

	got, err := decodePRMergeState([]byte(in))
	if err != nil {
		t.Fatalf("decodePRMergeState: %v", err)
	}
	if got.Mergeable != "UNKNOWN" || got.MergeStateStatus != "BLOCKED" || got.ReviewDecision != "REVIEW_REQUIRED" {
		t.Errorf("PRMergeState = %+v", got)
	}
	if len(got.StatusCheckRollup) != 2 {
		t.Fatalf("statusCheckRollup 件数 = %d, want 2", len(got.StatusCheckRollup))
	}
	run := got.StatusCheckRollup[0]
	if run.Typename != "CheckRun" || run.Name != "test" || run.Status != "COMPLETED" || run.Conclusion != "SUCCESS" {
		t.Errorf("CheckRun = %+v", run)
	}
	if run.WorkflowName != "CI" || run.DetailsURL != "du" {
		t.Errorf("CheckRun = %+v", run)
	}
	ctxCheck := got.StatusCheckRollup[1]
	if ctxCheck.Typename != "StatusContext" || ctxCheck.Context != "ci/legacy" || ctxCheck.State != "PENDING" {
		t.Errorf("StatusContext = %+v", ctxCheck)
	}
}

func TestDecodeReviewThreadsFlattens(t *testing.T) {
	const in = `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[{"id":"PRRT_x","isResolved":false,"comments":{"nodes":[{"databaseId":1,"author":{"login":"a"},"body":"b","createdAt":"2026-09-05T00:00:00Z"}]}}]}}}}}`

	got, err := decodeReviewThreads([]byte(in))
	if err != nil {
		t.Fatalf("decodeReviewThreads: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("thread 件数 = %d, want 1", len(got))
	}
	th := got[0]
	if th.ID != "PRRT_x" || th.IsResolved {
		t.Errorf("ReviewThread = %+v", th)
	}
	if len(th.Comments) != 1 {
		t.Fatalf("コメント件数 = %d, want 1", len(th.Comments))
	}
	c := th.Comments[0]
	if c.DatabaseID != 1 || c.Author.Login != "a" || c.Body != "b" {
		t.Errorf("ReviewComment = %+v", c)
	}
	if want := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC); !c.CreatedAt.Equal(want) {
		t.Errorf("CreatedAt = %v, want %v", c.CreatedAt, want)
	}
}

func TestDecodeCrossReferencedPRsSkipsEmptySource(t *testing.T) {
	const in = `{"data":{"repository":{"issue":{"timelineItems":{"nodes":[{"source":{"number":131,"title":"t","body":"b","state":"OPEN","labels":{"nodes":[{"name":"propose"},{"name":"question"}]}}},{"source":{}}]}}}}}`

	got, err := decodeCrossReferencedPRs([]byte(in))
	if err != nil {
		t.Fatalf("decodeCrossReferencedPRs: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("件数 = %d, want 1（source が空のノードは無視する）", len(got))
	}
	pr := got[0]
	if pr.Number != 131 || pr.Title != "t" || pr.Body != "b" || pr.State != "OPEN" {
		t.Errorf("CrossReferencedPR = %+v", pr)
	}
	if len(pr.Labels) != 2 || pr.Labels[0] != "propose" || pr.Labels[1] != "question" {
		t.Errorf("Labels = %v, want [propose question]", pr.Labels)
	}
}

func TestDecodeLabelEventsReadsObjectStream(t *testing.T) {
	const in = `{"created_at":"2026-09-01T00:00:00Z","event":"labeled","label":"stage:todo"}{"created_at":"2026-09-02T00:00:00Z","event":"unlabeled","label":"stage:todo"}`

	got, err := decodeLabelEvents([]byte(in))
	if err != nil {
		t.Fatalf("decodeLabelEvents: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("件数 = %d, want 2", len(got))
	}
	if got[0].Event != "labeled" || got[0].Label != "stage:todo" {
		t.Errorf("1 件目 = %+v", got[0])
	}
	if got[1].Event != "unlabeled" {
		t.Errorf("2 件目 = %+v", got[1])
	}
	if want := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC); !got[0].CreatedAt.Equal(want) {
		t.Errorf("CreatedAt = %v, want %v", got[0].CreatedAt, want)
	}
}

func TestDecodeLabelEventsEmptyOutput(t *testing.T) {
	got, err := decodeLabelEvents(nil)
	if err != nil {
		t.Fatalf("decodeLabelEvents: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("件数 = %d, want 0", len(got))
	}
}

func TestDecodeBrokenJSON(t *testing.T) {
	const broken = "not json"
	if _, err := decodeSearchIssues([]byte(broken)); err == nil {
		t.Error("decodeSearchIssues がエラーを返さない")
	}
	if _, err := decodeSearchPRs([]byte(broken)); err == nil {
		t.Error("decodeSearchPRs がエラーを返さない")
	}
	if _, err := decodeIssueDetail([]byte(broken)); err == nil {
		t.Error("decodeIssueDetail がエラーを返さない")
	}
	if _, err := decodePRDetail([]byte(broken)); err == nil {
		t.Error("decodePRDetail がエラーを返さない")
	}
	if _, err := decodePRMergeState([]byte(broken)); err == nil {
		t.Error("decodePRMergeState がエラーを返さない")
	}
	if _, err := decodeReviewThreads([]byte(broken)); err == nil {
		t.Error("decodeReviewThreads がエラーを返さない")
	}
	if _, err := decodeCrossReferencedPRs([]byte(broken)); err == nil {
		t.Error("decodeCrossReferencedPRs がエラーを返さない")
	}
	if _, err := decodeLabelEvents([]byte(broken)); err == nil {
		t.Error("decodeLabelEvents がエラーを返さない")
	}
}
