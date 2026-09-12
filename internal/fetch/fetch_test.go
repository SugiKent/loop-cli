package fetch

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

const (
	exampleDir  = "../gh/testdata/fixtures/example"
	fixturesDir = "../gh/testdata/fixtures"
)

var repos = []string{"org/app"}

// fetchNow は Fetch に渡す分類の基準時刻。testdata の PR（updatedAt 2026-09-04T10:00:00Z 以降）が規則 2 / 3 / 7 の時間切れに当たらない時刻にする。
var fetchNow = time.Date(2026, 9, 4, 10, 30, 0, 0, time.UTC)

func fetchDir(t *testing.T, dir string) *Result {
	t.Helper()
	res, err := Fetch(t.Context(), gh.NewFake(dir), repos, fetchNow, 0)
	if err != nil {
		t.Fatalf("Fetch(%s): %v", dir, err)
	}
	return res
}

// issueCard は Issue の repo / number が一致する Card を返す。
func issueCard(t *testing.T, res *Result, repo string, number int) model.Card {
	t.Helper()
	for _, c := range res.Cards {
		if c.Issue != nil && c.Issue.Repo == repo && c.Issue.Number == number {
			return c
		}
	}
	t.Fatalf("issue %s#%d の Card が無い", repo, number)
	return model.Card{}
}

// lonePR は Issue が nil で PR 1 件だけの Card からその PR を返す。
func lonePR(t *testing.T, res *Result, number int) model.PR {
	t.Helper()
	for _, c := range res.Cards {
		if c.Issue == nil && len(c.PRs) == 1 && c.PRs[0].Number == number {
			return c.PRs[0]
		}
	}
	t.Fatalf("PR #%d の単独 Card が無い", number)
	return model.PR{}
}

func prNumbers(prs []model.PR) []int {
	out := make([]int, len(prs))
	for i, pr := range prs {
		out[i] = pr.Number
	}
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestFetchExample(t *testing.T) {
	fake := gh.NewFake(exampleDir)
	res, err := Fetch(t.Context(), fake, repos, fetchNow, 0)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %v, want 空", res.Errors)
	}
	if len(res.Cards) != 2 {
		t.Fatalf("Cards 件数 = %d, want 2", len(res.Cards))
	}

	first := res.Cards[0]
	if first.Issue == nil || first.Issue.Number != 108 {
		t.Fatalf("1 枚目の Issue = %+v", first.Issue)
	}
	if !equalInts(prNumbers(first.PRs), []int{131}) {
		t.Errorf("1 枚目の PRs = %v, want [131]", prNumbers(first.PRs))
	}
	if first.Result.Situation != model.SituationA {
		t.Errorf("1 枚目の Situation = %q, want A", first.Result.Situation)
	}
	if first.PRs[0].Result.Situation != model.SituationA {
		t.Errorf("PR 131 の Situation = %q, want A", first.PRs[0].Result.Situation)
	}
	if first.Issue.Result.Situation != model.SituationInProgress {
		t.Errorf("issue 108 の Situation = %q, want in-progress", first.Issue.Result.Situation)
	}

	second := res.Cards[1]
	if second.Issue == nil || second.Issue.Number != 140 || len(second.PRs) != 0 {
		t.Fatalf("2 枚目の Card = %+v", second)
	}
	if second.Result.Situation != model.SituationE {
		t.Errorf("2 枚目の Situation = %q, want E", second.Result.Situation)
	}

	// 全 issue に ViewIssue が呼ばれる（question の有無を問わない）。
	var viewed []int
	for _, c := range fake.Calls {
		if c.Method == "ViewIssue" {
			viewed = append(viewed, c.Number)
		}
	}
	sort.Ints(viewed)
	if !equalInts(viewed, []int{108, 140}) {
		t.Errorf("ViewIssue の呼び出し = %v, want [108 140]", viewed)
	}
	if len(first.Issue.Comments) != 2 {
		t.Errorf("issue 108 の Comments = %d 件, want 2", len(first.Issue.Comments))
	}
	// issue-140.json のコメントは 0 件。取得できたので nil ではなく長さ 0 になる。
	if c := second.Issue.Comments; c == nil || len(c) != 0 {
		t.Errorf("issue 140 の Comments = %+v, want 長さ 0 の非 nil", c)
	}

	// question 付き PR でも merge 状態と review thread を取る。
	pr := first.PRs[0]
	if len(pr.Comments) != 1 {
		t.Errorf("PR 131 の Comments = %d 件, want 1", len(pr.Comments))
	}
	if pr.MergeState == nil || pr.MergeState.Mergeable != "UNKNOWN" {
		t.Errorf("PR 131 の MergeState = %+v, want Mergeable UNKNOWN", pr.MergeState)
	}
	if len(pr.ReviewThreads) != 1 {
		t.Errorf("PR 131 の ReviewThreads = %d 件, want 1", len(pr.ReviewThreads))
	}
}

const boardDir = "../gh/testdata/fixtures/board"

var (
	sddLabels   = []gh.RepoLabel{{Name: "bug"}, {Name: model.LabelStageTodo}}
	labelLabels = []gh.RepoLabel{{Name: model.LabelToDo}, {Name: model.LabelInProgress}}
	noModeLabel = []gh.RepoLabel{{Name: "bug"}, {Name: "enhancement"}}
)

// labelStub は board の fixture を読みつつ、ラベル一覧だけをリポジトリごとに差し替える。
// Fake は repo 引数でファイルを探さないので、リポジトリごとに違う方式を返す経路はここで作る。
type labelStub struct {
	*gh.Fake
	labels map[string][]gh.RepoLabel
	err    map[string]error
	mu     sync.Mutex
	calls  []string
}

func (s *labelStub) ListLabels(_ context.Context, repo string) ([]gh.RepoLabel, error) {
	s.mu.Lock()
	s.calls = append(s.calls, repo)
	s.mu.Unlock()
	if err := s.err[repo]; err != nil {
		return nil, err
	}
	return s.labels[repo], nil
}

func newLabelStub(dir string, labels map[string][]gh.RepoLabel) *labelStub {
	return &labelStub{Fake: gh.NewFake(dir), labels: labels}
}

// board は To Do を持ち stage:todo を持たないので label と判定され、In Progress が段階になる。
func TestFetchClassifiesWithDetectedMode(t *testing.T) {
	res, err := Fetch(t.Context(), gh.NewFake(boardDir), []string{"org/board"}, fetchNow, 0)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got := res.Modes["org/board"]; got != model.ModeLabel {
		t.Fatalf("Modes[org/board] = %q, want label", got)
	}
	got := issueCard(t, res, "org/board", 61).Issue.Result
	if got.Situation != model.SituationInProgress || got.Summary != "#61 は AI が作業中" {
		t.Errorf("issue 61 の Result = %+v, want in-progress / #61 は AI が作業中", got)
	}
	// label 方式の PR は段階ラベルを持たないので番号順に並ぶ。
	if prs := issueCard(t, res, "org/board", 61).PRs; len(prs) != 1 || prs[0].Number != 71 {
		t.Errorf("issue 61 の PRs = %+v, want [71]", prs)
	}
}

// 判定できないリポジトリは表に入らず、カードはゼロ値（sdd）の語彙で分類される。
func TestFetchUndetectedRepoClassifiesAsSDD(t *testing.T) {
	stub := newLabelStub(boardDir, map[string][]gh.RepoLabel{"org/board": noModeLabel})
	res, err := Fetch(t.Context(), stub, []string{"org/board"}, fetchNow, 0)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(res.Modes) != 0 {
		t.Errorf("Modes = %v, want 空", res.Modes)
	}
	if len(res.Errors) != 0 {
		t.Errorf("Errors = %v, want 空（判定できないことは失敗ではない）", res.Errors)
	}
	// In Progress は sdd の段階ラベルではないので、段階ラベルの無い issue として E になる。
	if got := issueCard(t, res, "org/board", 61).Issue.Result.Situation; got != model.SituationE {
		t.Errorf("issue 61 の Situation = %q, want E", got)
	}
}

func TestFetchListsLabelsOncePerRepo(t *testing.T) {
	stub := newLabelStub("testdata/multirepo", map[string][]gh.RepoLabel{
		"org/app": sddLabels, "org/web": sddLabels,
	})
	if _, err := Fetch(t.Context(), stub, []string{"org/app", "org/web"}, fetchNow, 0); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	got := append([]string(nil), stub.calls...)
	sort.Strings(got)
	if len(got) != 2 || got[0] != "org/app" || got[1] != "org/web" {
		t.Errorf("ListLabels の呼び出し = %v, want [org/app org/web] を 1 回ずつ", stub.calls)
	}
}

// リポジトリごとに違う一覧を返すと、方式もリポジトリごとに分かれる。
func TestFetchDetectsModePerRepo(t *testing.T) {
	stub := newLabelStub(boardDir, map[string][]gh.RepoLabel{
		"org/app": sddLabels, "org/board": labelLabels,
	})
	res, err := Fetch(t.Context(), stub, []string{"org/app", "org/board"}, fetchNow, 0)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	want := map[string]model.Mode{"org/app": model.ModeSDD, "org/board": model.ModeLabel}
	if !maps.Equal(res.Modes, want) {
		t.Fatalf("Modes = %v, want %v", res.Modes, want)
	}
	if got := issueCard(t, res, "org/board", 61).Issue.Result.Situation; got != model.SituationInProgress {
		t.Errorf("org/board の issue 61 の Situation = %q, want in-progress", got)
	}
}

// ListLabels が失敗したリポジトリは表に入らないが、Card も他のリポジトリの方式も落ちない。
func TestFetchListLabelsFailureKeepsCards(t *testing.T) {
	stub := newLabelStub(boardDir, map[string][]gh.RepoLabel{"org/board": labelLabels})
	stub.err = map[string]error{"org/app": errors.New("gh label list: exit 1")}
	res, err := Fetch(t.Context(), stub, []string{"org/app", "org/board"}, fetchNow, 0)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(res.Cards) == 0 {
		t.Fatal("Cards = 空, want 落ちていない")
	}
	if _, ok := res.Modes["org/app"]; ok {
		t.Errorf("Modes に org/app がある: %v", res.Modes)
	}
	if got := res.Modes["org/board"]; got != model.ModeLabel {
		t.Errorf("Modes[org/board] = %q, want label", got)
	}
	if len(res.Errors) != 1 {
		t.Fatalf("Errors = %v, want 1 件", res.Errors)
	}
	for _, want := range []string{"ListLabels", "org/app"} {
		if !strings.Contains(res.Errors[0].Error(), want) {
			t.Errorf("Errors[0] = %q, %q を含まない", res.Errors[0], want)
		}
	}
}

// リポジトリのエラーは番号を持つエラーより前に並ぶ。
func TestFetchRepoErrorSortsBeforeNumberedErrors(t *testing.T) {
	// partial は issue-108.json を持つが、ここではさらに ViewIssue も失敗させる。
	stub := &labelStub{
		Fake:   gh.NewFake("testdata/partial"),
		labels: map[string][]gh.RepoLabel{},
		err:    map[string]error{"org/app": errors.New("gh label list: exit 1")},
	}
	res, err := Fetch(t.Context(), &viewIssueFailure{labelStub: stub}, repos, fetchNow, 0)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(res.Errors) < 2 {
		t.Fatalf("Errors = %v, want 2 件以上", res.Errors)
	}
	if !strings.HasPrefix(res.Errors[0].Error(), "ListLabels ") {
		t.Errorf("Errors[0] = %q, want ListLabels で始まる", res.Errors[0])
	}
	if !strings.HasPrefix(res.Errors[1].Error(), "ViewIssue ") {
		t.Errorf("Errors[1] = %q, want ViewIssue で始まる", res.Errors[1])
	}
}

// viewIssueFailure は ViewIssue だけを必ず失敗させる包み。
type viewIssueFailure struct{ *labelStub }

func (v *viewIssueFailure) ViewIssue(context.Context, string, int) (*gh.IssueDetail, error) {
	return nil, errors.New("gh issue view: exit 1")
}

func TestFetchLink(t *testing.T) {
	res := fetchDir(t, "testdata/link")
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %v, want 空", res.Errors)
	}

	card := issueCard(t, res, "org/app", 108)
	if !equalInts(prNumbers(card.PRs), []int{131, 140, 151}) {
		t.Fatalf("issue 108 の PRs = %v, want [131 140 151]", prNumbers(card.PRs))
	}

	// merge 候補 PR（archive・未確定 0 件・question 無し）。
	pr151 := card.PRs[2]
	if pr151.MergeState == nil || pr151.MergeState.Mergeable != "MERGEABLE" {
		t.Errorf("PR 151 の MergeState = %+v, want MERGEABLE", pr151.MergeState)
	}
	if len(pr151.Comments) != 0 {
		t.Errorf("PR 151 の Comments = %d 件, want 0", len(pr151.Comments))
	}
	if pr151.ReviewThreads == nil || len(pr151.ReviewThreads) != 0 {
		t.Errorf("PR 151 の ReviewThreads = %+v, want 長さ 0 の非 nil", pr151.ReviewThreads)
	}
	if pr151.Result.Situation != model.SituationC {
		t.Errorf("PR 151 の Situation = %q, want C", pr151.Result.Situation)
	}

	// 未確定 1 件の apply PR。merge 状態も取るが、未確定が 0 件でないので C にならない。
	pr140 := card.PRs[1]
	if pr140.MergeState == nil {
		t.Error("PR 140 の MergeState = nil, want non-nil")
	}
	if len(pr140.ReviewThreads) != 1 {
		t.Errorf("PR 140 の ReviewThreads = %d 件, want 1", len(pr140.ReviewThreads))
	}
	if pr140.Result.Situation != model.SituationOther {
		t.Errorf("PR 140 の Situation = %q, want other", pr140.Result.Situation)
	}

	// 紐づかない PR は単独 Card になる。
	for _, tt := range []struct {
		number int
		want   model.Situation
	}{
		{60, model.SituationG},
		{61, model.SituationOther},
		{62, model.SituationOther},
		{132, model.SituationOther},
		{90, model.SituationInProgress},
	} {
		pr := lonePR(t, res, tt.number)
		if pr.Result.Situation != tt.want {
			t.Errorf("PR %d の Situation = %q, want %q", tt.number, pr.Result.Situation, tt.want)
		}
	}

	// 詳細が埋まっても分類結果は変わらない: 未確定 2 件の propose PR は other のまま。
	pr132 := lonePR(t, res, 132)
	if len(pr132.Comments) != 1 {
		t.Errorf("PR 132 の Comments = %d 件, want 1", len(pr132.Comments))
	}
	if pr132.MergeState == nil || pr132.MergeState.Mergeable != "MERGEABLE" {
		t.Errorf("PR 132 の MergeState = %+v, want MERGEABLE", pr132.MergeState)
	}
	if pr132.ReviewThreads == nil || len(pr132.ReviewThreads) != 0 {
		t.Errorf("PR 132 の ReviewThreads = %+v, want 長さ 0 の非 nil", pr132.ReviewThreads)
	}
	if pr132.Result.Situation != model.SituationOther {
		t.Errorf("PR 132 の Situation = %q, want other", pr132.Result.Situation)
	}

	// question 無し PR でもコメントを取り、最新コメントが人なら進行中。
	pr90 := lonePR(t, res, 90)
	if len(pr90.Comments) != 2 {
		t.Errorf("PR 90 の Comments = %d 件, want 2", len(pr90.Comments))
	}
	if len(pr90.ReviewThreads) != 1 {
		t.Errorf("PR 90 の ReviewThreads = %d 件, want 1", len(pr90.ReviewThreads))
	}
	if pr90.MergeState == nil {
		t.Error("PR 90 の MergeState = nil, want non-nil")
	}

	// question 無しの issue のコメントは分類に影響しない（規則 4 の分岐に入らない）。
	issue140 := issueCard(t, res, "org/app", 140)
	if len(issue140.Issue.Comments) != 1 {
		t.Errorf("issue 140 の Comments = %d 件, want 1", len(issue140.Issue.Comments))
	}
	if issue140.Issue.Result.Situation != model.SituationE {
		t.Errorf("issue 140 の Situation = %q, want E", issue140.Issue.Result.Situation)
	}
}

func TestFetchMultiRepo(t *testing.T) {
	res, err := Fetch(t.Context(), gh.NewFake("testdata/multirepo"), []string{"org/app", "org/web"}, fetchNow, 0)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("Errors = %v, want 空", res.Errors)
	}
	if len(res.Cards) != 2 {
		t.Fatalf("Cards 件数 = %d, want 2", len(res.Cards))
	}
	if got := prNumbers(issueCard(t, res, "org/web", 12).PRs); !equalInts(got, []int{30}) {
		t.Errorf("org/web の issue 12 の PRs = %v, want [30]", got)
	}
	if got := issueCard(t, res, "org/app", 12).PRs; len(got) != 0 {
		t.Errorf("org/app の issue 12 の PRs = %v, want 空", prNumbers(got))
	}
}

func TestFetchSameStage(t *testing.T) {
	res := fetchDir(t, "testdata/samestage")
	card := issueCard(t, res, "org/app", 108)
	if !equalInts(prNumbers(card.PRs), []int{131, 140}) {
		t.Fatalf("issue 108 の PRs = %v, want [131 140]", prNumbers(card.PRs))
	}
	for _, pr := range card.PRs {
		if pr.Canonical {
			t.Errorf("PR %d の Canonical = true, want false（open PR に正本は立たない）", pr.Number)
		}
	}
	if card.Result.Situation != model.SituationA {
		t.Errorf("Card の Situation = %q, want A", card.Result.Situation)
	}

	// apply + question の PR は question があっても review threads を取る。
	pr88 := lonePR(t, res, 88)
	if len(pr88.Comments) == 0 || len(pr88.ReviewThreads) == 0 {
		t.Errorf("PR 88 の Comments = %d 件、ReviewThreads = %d 件, want どちらも 1 件以上",
			len(pr88.Comments), len(pr88.ReviewThreads))
	}
	if pr88.MergeState == nil {
		t.Error("PR 88 の MergeState = nil, want non-nil")
	}
}

func TestFetchPartialFailure(t *testing.T) {
	res := fetchDir(t, "testdata/partial")
	card := issueCard(t, res, "org/app", 108)
	if !equalInts(prNumbers(card.PRs), []int{131}) {
		t.Fatalf("issue 108 の PRs = %v, want [131]", prNumbers(card.PRs))
	}
	pr131 := card.PRs[0]
	if pr131.Comments != nil || pr131.MergeState != nil || pr131.ReviewThreads != nil {
		t.Errorf("PR 131 の詳細 = %+v / %+v / %+v, want すべて nil",
			pr131.Comments, pr131.MergeState, pr131.ReviewThreads)
	}
	if pr131.Result.Situation != model.SituationOther {
		t.Errorf("PR 131 の Situation = %q, want other", pr131.Result.Situation)
	}
	if len(card.Issue.Comments) != 2 {
		t.Errorf("issue 108 の Comments = %d 件, want 2", len(card.Issue.Comments))
	}

	// 1 つの PR の 3 つの詳細が失敗し、並びはメソッド順で決まる（完了順に依存しない）。
	if len(res.Errors) != 3 {
		t.Fatalf("Errors = %v, want 3 件", res.Errors)
	}
	for i, method := range []string{"ViewPR", "ViewPRMergeState", "ReviewThreads"} {
		msg := res.Errors[i].Error()
		if !strings.HasPrefix(msg, method+" org/app#131: ") {
			t.Errorf("Errors[%d] = %q, want %q で始まる", i, msg, method+" org/app#131: ")
		}
	}
}

func TestFetchSearchIssuesFailure(t *testing.T) {
	res, err := Fetch(t.Context(), gh.NewFake("testdata/nosearch"), repos, fetchNow, 0)
	if err == nil {
		t.Fatalf("エラーを期待したが nil（Result = %+v）", res)
	}
	if res != nil {
		t.Errorf("Result = %+v, want nil", res)
	}
	for _, want := range []string{"search issues", "search-issues.json"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %q, %q を含まない", err, want)
		}
	}
}

// stub は必要なメソッドだけ上書きする GHClient。上書きしていないメソッドを呼ぶと nil パニックになる。
type stub struct {
	gh.GHClient
	searchIssues func(ctx context.Context, repos []string) ([]gh.SearchIssue, error)
	searchPRs    func(ctx context.Context, repos []string) ([]gh.SearchPR, error)
	viewIssue    func(ctx context.Context, repo string, number int) (*gh.IssueDetail, error)
}

func (s *stub) SearchIssues(ctx context.Context, repos []string) ([]gh.SearchIssue, error) {
	return s.searchIssues(ctx, repos)
}

func (s *stub) SearchPRs(ctx context.Context, repos []string) ([]gh.SearchPR, error) {
	return s.searchPRs(ctx, repos)
}

func (s *stub) ViewIssue(ctx context.Context, repo string, number int) (*gh.IssueDetail, error) {
	return s.viewIssue(ctx, repo, number)
}

// 方式を見ないテストのために、ラベル一覧は既定で空（どちらのラベルも無い＝判定できない）を返す。
func (s *stub) ListLabels(context.Context, string) ([]gh.RepoLabel, error) { return nil, nil }

func TestFetchSearchCalledOnce(t *testing.T) {
	var issues, prs int
	c := &stub{
		searchIssues: func(context.Context, []string) ([]gh.SearchIssue, error) { issues++; return nil, nil },
		searchPRs:    func(context.Context, []string) ([]gh.SearchPR, error) { prs++; return nil, nil },
	}
	if _, err := Fetch(t.Context(), c, repos, fetchNow, 0); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if issues != 1 || prs != 1 {
		t.Errorf("SearchIssues = %d 回、SearchPRs = %d 回, want どちらも 1 回", issues, prs)
	}
}

func TestFetchParentDeadline(t *testing.T) {
	c := &stub{
		searchIssues: func(ctx context.Context, _ []string) ([]gh.SearchIssue, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	res, err := Fetch(ctx, c, repos, fetchNow, 0)
	if err == nil {
		t.Fatalf("エラーを期待したが nil（Result = %+v）", res)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("所要 %v。親 ctx の期限が伝わっていない", elapsed)
	}
}

func TestFetchCanceledReturnsNoPartialResult(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	c := &stub{
		searchIssues: func(context.Context, []string) ([]gh.SearchIssue, error) {
			return []gh.SearchIssue{{
				Repository: gh.Repository{Name: "app", NameWithOwner: "org/app"},
				Number:     108,
				Labels:     []gh.Label{{Name: model.LabelQuestion}},
			}}, nil
		},
		// 詳細取得に入る直前に確実にキャンセルされた状態を作る。
		searchPRs: func(context.Context, []string) ([]gh.SearchPR, error) {
			cancel()
			return []gh.SearchPR{}, nil
		},
		viewIssue: func(ctx context.Context, _ string, _ int) (*gh.IssueDetail, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	res, err := Fetch(ctx, c, repos, fetchNow, 0)
	if err == nil {
		t.Fatalf("エラーを期待したが nil（Result = %+v）", res)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if res != nil {
		t.Errorf("Result = %+v, want nil", res)
	}
}

// TestFetchCoversAllFixtureAliases は internal/gh の全 fixture で
// 「issue / PR がちょうど 1 枚の Card に現れる」不変条件を検証する（Situation は見ない）。
func TestFetchCoversAllFixtureAliases(t *testing.T) {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatalf("fixtures を読めません: %v", err)
	}
	found := false
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		found = true
		t.Run(e.Name(), func(t *testing.T) {
			dir := filepath.Join(fixturesDir, e.Name())
			fake := gh.NewFake(dir)
			wantIssues, err := fake.SearchIssues(t.Context(), repos)
			if err != nil {
				t.Fatalf("SearchIssues: %v", err)
			}
			wantPRs, err := fake.SearchPRs(t.Context(), repos)
			if err != nil {
				t.Fatalf("SearchPRs: %v", err)
			}

			res := fetchDir(t, dir)
			if len(res.Errors) != 0 {
				t.Fatalf("Errors = %v, want 空", res.Errors)
			}

			gotIssues := map[string]int{}
			gotPRs := map[string]int{}
			for _, c := range res.Cards {
				if c.Issue != nil {
					gotIssues[key(c.Issue.Repo, c.Issue.Number)]++
				}
				for _, pr := range c.PRs {
					gotPRs[key(pr.Repo, pr.Number)]++
				}
			}
			for _, si := range wantIssues {
				if n := gotIssues[key(si.Repository.NameWithOwner, si.Number)]; n != 1 {
					t.Errorf("issue %s#%d は %d 枚の Card に現れた, want 1", si.Repository.NameWithOwner, si.Number, n)
				}
			}
			for _, sp := range wantPRs {
				if n := gotPRs[key(sp.Repository.NameWithOwner, sp.Number)]; n != 1 {
					t.Errorf("PR %s#%d は %d 枚の Card に現れた, want 1", sp.Repository.NameWithOwner, sp.Number, n)
				}
			}
			if len(gotIssues) != len(wantIssues) || len(gotPRs) != len(wantPRs) {
				t.Errorf("Card 内の issue %d 件 / PR %d 件, want %d / %d",
					len(gotIssues), len(gotPRs), len(wantIssues), len(wantPRs))
			}
		})
	}
	if !found {
		t.Fatal("fixtures ディレクトリが空です")
	}
}

func key(repo string, number int) string {
	return fmt.Sprintf("%s#%d", repo, number)
}

// TestFetchPassesGraceToClassify は取得時刻と猶予が classify.Card に届くことを、
// testdata/link のラベル無し PR 61（updatedAt 2026-09-04T10:00:00Z）で確かめる。
func TestFetchPassesGraceToClassify(t *testing.T) {
	updatedAt := time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC)

	t.Run("猶予内のその他は進行中になる", func(t *testing.T) {
		res, err := Fetch(t.Context(), gh.NewFake("testdata/link"), repos, updatedAt.Add(10*time.Minute), 30*time.Minute)
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		pr := lonePR(t, res, 61)
		if pr.Result.Situation != model.SituationInProgress {
			t.Errorf("PR 61 の Situation = %q, want in-progress", pr.Result.Situation)
		}
		if want := "PR #61 はどの局面にも当たらない（更新から 30m は様子見）"; pr.Result.Summary != want {
			t.Errorf("PR 61 の Summary = %q, want %q", pr.Result.Summary, want)
		}
	})

	t.Run("猶予以上経った now ではその他に戻る", func(t *testing.T) {
		res, err := Fetch(t.Context(), gh.NewFake("testdata/link"), repos, updatedAt.Add(31*time.Minute), 30*time.Minute)
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if got := lonePR(t, res, 61).Result.Situation; got != model.SituationOther {
			t.Errorf("PR 61 の Situation = %q, want other", got)
		}
	})
}
