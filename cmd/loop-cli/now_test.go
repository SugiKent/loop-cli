package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
	"github.com/SugiKent/loop-cli/internal/snapshot"
)

// nowAt は分類の基準時刻。fixture の issue / PR が規則 2 / 3 / 7 の時間切れに当たらない時刻にする。
var nowAt = time.Date(2026, 9, 4, 10, 30, 0, 0, time.UTC)

// boardAt は board fixture（updatedAt が 2026-09-05）用の基準時刻。
var boardAt = time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)

// fetchFixture は fixture から取得結果を作る。repos は fixture のリポジトリ名に合わせる
// （合わないと運用方式が判定されず、既定の sdd に倒れて分類が変わる）。
func fetchFixture(t *testing.T, dir, repo string, at time.Time) *fetch.Result {
	t.Helper()
	res, err := fetch.Fetch(t.Context(), gh.NewFake(dir), []string{repo}, at, 0)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	return res
}

// decodeNow は buildNow の出力を JSON へ通し、汎用の map として読み直す。
// 構造体を直接見ずにキー名と値を検証する（agent が読む契約は JSON の側にあるため）。
func decodeNow(t *testing.T, out nowOutput) map[string]any {
	t.Helper()
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return got
}

// TestBuildNowKeepsOnlyNowTab は board fixture の「今やる」2 枚だけが優先度順で出ることを検証する。
func TestBuildNowKeepsOnlyNowTab(t *testing.T) {
	res := fetchFixture(t, "../../internal/gh/testdata/fixtures/board", "org/board", boardAt)

	out := buildNow(res, boardAt)

	if out.Count != 2 || len(out.Items) != 2 {
		t.Fatalf("count = %d, items = %d, want 2 と 2", out.Count, len(out.Items))
	}
	// 主体は PR #72（局面 A）→ PR #71（局面 C）の順。issue 62（バックログ）と 63（異常）は入らない。
	want := []nowSubject{{Type: "pr", Number: 72}, {Type: "pr", Number: 71}}
	for i, w := range want {
		if out.Items[i].Subject != w {
			t.Errorf("items[%d].subject = %+v, want %+v", i, out.Items[i].Subject, w)
		}
	}
	for _, item := range out.Items {
		if item.Issue != nil && (item.Issue.Number == 62 || item.Issue.Number == 63) {
			t.Errorf("他のタブの issue #%d が入った", item.Issue.Number)
		}
	}
	if got := out.Items[0]; got.Situation != "A" || got.Kind != "質問" || got.Priority != 1 ||
		got.Summary != "PR #72 の質問に答える" || got.Repo != "org/board" {
		t.Errorf("items[0] = %+v, want A / 質問 / 1 / PR #72 の質問に答える / org/board", got)
	}
}

// TestBuildNowSubjectPointsIntoCard は PR が主体のカードで subject と issue / prs が噛み合うことを検証する。
func TestBuildNowSubjectPointsIntoCard(t *testing.T) {
	res := fetchFixture(t, "../../internal/gh/testdata/fixtures/example", "org/app", nowAt)

	out := buildNow(res, nowAt)

	if len(out.Items) != 1 {
		t.Fatalf("items = %d 件, want 1", len(out.Items))
	}
	item := out.Items[0]
	if item.Subject != (nowSubject{Type: "pr", Number: 131}) {
		t.Errorf("subject = %+v, want {pr 131}", item.Subject)
	}
	if item.Issue == nil || item.Issue.Number != 108 {
		t.Fatalf("issue = %+v, want #108", item.Issue)
	}
	found := false
	for _, pr := range item.PRs {
		if pr.Number == 131 {
			found = true
			if pr.URL == "" || pr.Title == "" {
				t.Errorf("prs の 131 に title / url が無い: %+v", pr)
			}
		}
	}
	if !found {
		t.Errorf("prs に 131 が無い: %+v", item.PRs)
	}
}

// card は指定の局面と主体を持つ「今やる」のカードを作る。
func card(repo string, issue *model.Issue, prs []model.PR, situation model.Situation, subjectIsPR bool) model.Card {
	res := model.Result{
		Situation: situation,
		Priority:  situation.Priority(),
		Tab:       situation.Tab(),
		Summary:   fmt.Sprintf("%s の出番", repo),
	}
	c := model.Card{Issue: issue, PRs: prs, Result: res}
	if subjectIsPR {
		c.PRs[0].Result = res
	} else {
		c.Issue.Result = res
	}
	return c
}

// issueCardAt は issue 単独（局面 B）のカードを作る。
func issueCardAt(repo string, number int, updatedAt time.Time) model.Card {
	issue := &model.Issue{Repo: repo, Number: number, Title: fmt.Sprintf("issue %d", number),
		URL: fmt.Sprintf("https://github.com/%s/issues/%d", repo, number), UpdatedAt: updatedAt}
	return card(repo, issue, nil, model.SituationB, false)
}

// TestBuildNowEmpty は「今やる」が 0 件でも count 0 と空配列が出ることを検証する。
func TestBuildNowEmpty(t *testing.T) {
	res := &fetch.Result{Cards: []model.Card{
		{Issue: &model.Issue{Repo: "org/app", Number: 1}, Result: model.Result{Situation: model.SituationE, Tab: model.TabBacklog}},
	}}

	out := buildNow(res, nowAt)

	if got := decodeNow(t, out); got["count"] != float64(0) {
		t.Errorf("count = %v, want 0", got["count"])
	}
	// nil のスライスは null になるので、生の JSON で [] を確かめる（design.md D3）。
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, want := range []string{`"items":[]`, `"errors":[]`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("JSON に %s が無い: %s", want, b)
		}
	}
}

// TestBuildNowOrdersByUpdatedAtThenRepoThenNumber は優先度が同じときの tiebreak を検証する。
func TestBuildNowOrdersByUpdatedAtThenRepoThenNumber(t *testing.T) {
	old := nowAt.Add(-2 * time.Hour)
	res := &fetch.Result{Cards: []model.Card{
		issueCardAt("org/web", 10, old),
		issueCardAt("org/app", 20, old),
		issueCardAt("org/app", 5, old),
		issueCardAt("org/zzz", 99, nowAt),
	}}

	out := buildNow(res, nowAt)

	want := []nowSubject{
		{Type: "issue", Number: 99}, // 更新が新しい
		{Type: "issue", Number: 5},  // 以降は同じ更新時刻。リポジトリ名昇順 → 番号昇順
		{Type: "issue", Number: 20},
		{Type: "issue", Number: 10},
	}
	if len(out.Items) != len(want) {
		t.Fatalf("items = %d 件, want %d", len(out.Items), len(want))
	}
	for i, w := range want {
		if out.Items[i].Subject != w {
			t.Errorf("items[%d].subject = %+v, want %+v", i, out.Items[i].Subject, w)
		}
	}
}

// TestBuildNowLonePRHasNullIssue は PR 単独のカードで issue が null になることを検証する。
func TestBuildNowLonePRHasNullIssue(t *testing.T) {
	pr := model.PR{Repo: "org/app", Number: 200, Title: "単独 PR", URL: "https://github.com/org/app/pull/200"}
	res := &fetch.Result{Cards: []model.Card{card("org/app", nil, []model.PR{pr}, model.SituationC, true)}}

	got := decodeNow(t, buildNow(res, nowAt))

	items := got["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %d 件, want 1", len(items))
	}
	item := items[0].(map[string]any)
	if item["issue"] != nil {
		t.Errorf("issue = %v, want null", item["issue"])
	}
	subject := item["subject"].(map[string]any)
	if subject["type"] != "pr" {
		t.Errorf("subject.type = %v, want pr", subject["type"])
	}
}

// statefulPR は merge 状態と review thread が取れた PR。
func statefulPR() model.PR {
	return model.PR{
		Repo: "org/app", Number: 300, Title: "状態のある PR", URL: "https://github.com/org/app/pull/300",
		Labels: []string{"apply"}, Body: "未確定の判断: 0 件 — レビューをお願いします\n\nRefs #1",
		MergeState: &gh.PRMergeState{
			Mergeable: "MERGEABLE", MergeStateStatus: "CLEAN", ReviewDecision: "",
			StatusCheckRollup: []gh.StatusCheck{{Typename: "CheckRun", Name: "CI / build", Conclusion: "SUCCESS"}},
		},
		ReviewThreads: []gh.ReviewThread{{ID: "t1", IsResolved: false}},
	}
}

// TestBuildNowPRState は PR の状態 7 欄が写ることを検証する。
func TestBuildNowPRState(t *testing.T) {
	res := &fetch.Result{Cards: []model.Card{card("org/app", nil, []model.PR{statefulPR()}, model.SituationC, true)}}

	out := buildNow(res, nowAt)

	pr := out.Items[0].PRs[0]
	if pr.Undecided == nil || *pr.Undecided != 0 {
		t.Errorf("undecided = %v, want 0", pr.Undecided)
	}
	if pr.Mergeable == nil || *pr.Mergeable != "MERGEABLE" {
		t.Errorf("mergeable = %v, want MERGEABLE", pr.Mergeable)
	}
	if pr.MergeStateStatus == nil || *pr.MergeStateStatus != "CLEAN" {
		t.Errorf("merge_state_status = %v, want CLEAN", pr.MergeStateStatus)
	}
	if pr.ReviewDecision == nil || *pr.ReviewDecision != "" {
		t.Errorf("review_decision = %v, want 空文字", pr.ReviewDecision)
	}
	if pr.ChecksGreen == nil || !*pr.ChecksGreen {
		t.Errorf("checks_green = %v, want true", pr.ChecksGreen)
	}
	if len(pr.Checks) != 1 || pr.Checks[0] != (nowCheck{Name: "CI / build", State: "SUCCESS"}) {
		t.Errorf("checks = %+v, want [{CI / build SUCCESS}]", pr.Checks)
	}
	if pr.UnresolvedThreads == nil || *pr.UnresolvedThreads != 1 {
		t.Errorf("unresolved_threads = %v, want 1", pr.UnresolvedThreads)
	}
}

// TestBuildNowPRStateNullOnFetchFailure は詳細が取れなかった PR で状態 6 欄が null になることを検証する。
// checks_green を false にしないのは「checks が赤い」と読めてしまうため（design.md D2b）。
func TestBuildNowPRStateNullOnFetchFailure(t *testing.T) {
	pr := model.PR{Repo: "org/app", Number: 301, Title: "取得に失敗した PR",
		URL: "https://github.com/org/app/pull/301", Labels: []string{"apply"}}
	res := &fetch.Result{Cards: []model.Card{card("org/app", nil, []model.PR{pr}, model.SituationC, true)}}

	got := decodeNow(t, buildNow(res, nowAt))

	items := got["items"].([]any)
	prs := items[0].(map[string]any)["prs"].([]any)
	jsonPR := prs[0].(map[string]any)
	for _, key := range []string{"mergeable", "merge_state_status", "review_decision", "checks_green", "checks", "unresolved_threads"} {
		v, ok := jsonPR[key]
		if !ok {
			t.Errorf("%s のキーが無い", key)
			continue
		}
		if v != nil {
			t.Errorf("%s = %v, want null", key, v)
		}
	}
	if jsonPR["number"] != float64(301) || jsonPR["title"] != "取得に失敗した PR" ||
		jsonPR["url"] != "https://github.com/org/app/pull/301" {
		t.Errorf("number / title / url が出ていない: %+v", jsonPR)
	}
	if labels, ok := jsonPR["labels"].([]any); !ok || len(labels) != 1 || labels[0] != "apply" {
		t.Errorf("labels = %v, want [apply]", jsonPR["labels"])
	}
}

// TestRunNowRejectsExtraArgs は now の後ろの引数を黙って捨てないことを検証する。
// 設定も gh も触る前に終わるので、この経路は本物の依存を組み立てない。
func TestRunNowRejectsExtraArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run([]string{"now", "--repo", "org/app"}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want 空", stdout.String())
	}
	if lines := strings.Split(strings.TrimRight(stderr.String(), "\n"), "\n"); len(lines) != 1 ||
		!strings.Contains(lines[0], "--repo") {
		t.Errorf("stderr = %q, want --repo を含む 1 行", stderr.String())
	}
}

// stubDeps は runNow に渡す依存のスタブ。gh もネットワークも使わない。
type stubDeps struct {
	path      string
	pathErr   error
	checkErr  error
	result    *fetch.Result
	fetchErr  error
	fetchArgs struct {
		repos []string
		now   time.Time
		grace time.Duration
	}
}

func (s *stubDeps) deps() nowDeps {
	return nowDeps{
		configPath: func() (string, error) { return s.path, s.pathErr },
		check:      func(context.Context) error { return s.checkErr },
		fetch: func(_ context.Context, repos []string, now time.Time, grace time.Duration) (*fetch.Result, error) {
			s.fetchArgs.repos, s.fetchArgs.now, s.fetchArgs.grace = repos, now, grace
			return s.result, s.fetchErr
		},
	}
}

// runNowWith は設定ファイルを書いてから runNow を呼ぶ。
func runNowWith(t *testing.T, s *stubDeps) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runNow(t.Context(), s.deps(), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// TestRunNowMissingConfig は設定ファイルが無ければフォームに入らず 1 で終わることを検証する。
func TestRunNowMissingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	s := &stubDeps{path: path}

	code, stdout, stderr := runNowWith(t, s)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want 空", stdout)
	}
	if !strings.Contains(stderr, "設定ファイルがありません") || !strings.Contains(stderr, path) {
		t.Errorf("stderr = %q, want 設定ファイルがありません + パス", stderr)
	}
}

// TestRunNowGhNotFound は gh が無いときに 2 行の案内が出ることを検証する。
func TestRunNowGhNotFound(t *testing.T) {
	s := &stubDeps{path: writeConfig(t, "repos:\n  - org/app\n"), checkErr: fmt.Errorf("gh: %w", exec.ErrNotFound)}

	code, stdout, stderr := runNowWith(t, s)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want 空", stdout)
	}
	lines := strings.Split(strings.TrimRight(stderr, "\n"), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "gh が見つかりません") ||
		!strings.Contains(lines[1], "https://cli.github.com/") {
		t.Errorf("stderr = %q, want gh が見つかりません + インストール先の 2 行", stderr)
	}
}

// TestRunNowFetchFailure は検索そのものが失敗したら標準出力を空のまま 1 で終わることを検証する。
func TestRunNowFetchFailure(t *testing.T) {
	s := &stubDeps{path: writeConfig(t, "repos:\n  - org/app\n"), fetchErr: errors.New("search issues: gh: exit 1")}

	code, stdout, stderr := runNowWith(t, s)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want 空", stdout)
	}
	if !strings.Contains(stderr, "search issues: gh: exit 1") {
		t.Errorf("stderr = %q, want 取得のエラー", stderr)
	}
}

// TestRunNowSucceeds は成功時に 1 つの JSON オブジェクトが出て 0 で終わることを検証する。
func TestRunNowSucceeds(t *testing.T) {
	s := &stubDeps{
		path:   writeConfig(t, "repos:\n  - org/app\nother_grace_min: 45\n"),
		result: &fetch.Result{Cards: []model.Card{card("org/app", nil, []model.PR{statefulPR()}, model.SituationC, true)}},
	}

	code, stdout, stderr := runNowWith(t, s)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", code, stderr)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout が 1 つの JSON オブジェクトでない: %v (%q)", err, stdout)
	}
	if got["count"] != float64(1) {
		t.Errorf("count = %v, want 1", got["count"])
	}
	if !strings.HasSuffix(stdout, "}\n") {
		t.Errorf("末尾に改行が無い: %q", stdout[max(0, len(stdout)-8):])
	}
	// 設定の repos と other_grace_min がそのまま取得に渡る（TUI と同じ分類にする）。
	if len(s.fetchArgs.repos) != 1 || s.fetchArgs.repos[0] != "org/app" {
		t.Errorf("repos = %v, want [org/app]", s.fetchArgs.repos)
	}
	if s.fetchArgs.grace != 45*time.Minute {
		t.Errorf("grace = %v, want 45m", s.fetchArgs.grace)
	}
	// 分類の基準時刻と fetched_at は同じ値（時間切れの判定と出力の時刻をそろえる）。
	fetchedAt, err := time.Parse(time.RFC3339Nano, got["fetched_at"].(string))
	if err != nil {
		t.Fatalf("fetched_at のパース: %v", err)
	}
	if !fetchedAt.Equal(s.fetchArgs.now) {
		t.Errorf("fetched_at = %v, want 取得に渡した %v", fetchedAt, s.fetchArgs.now)
	}
}

// TestRunNowEmptySucceeds は「今やる」が 0 件でも 0 で終わることを検証する。
func TestRunNowEmptySucceeds(t *testing.T) {
	s := &stubDeps{path: writeConfig(t, "repos:\n  - org/app\n"), result: &fetch.Result{}}

	code, stdout, stderr := runNowWith(t, s)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", code, stderr)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("Unmarshal: %v (%q)", err, stdout)
	}
	if got["count"] != float64(0) {
		t.Errorf("count = %v, want 0", got["count"])
	}
	if items, ok := got["items"].([]any); !ok || len(items) != 0 {
		t.Errorf("items = %v, want []", got["items"])
	}
	if errs, ok := got["errors"].([]any); !ok || len(errs) != 0 {
		t.Errorf("errors = %v, want []", got["errors"])
	}
}

// TestRunNowPartialFailureKeepsItems は部分失敗でも一覧を出して 0 で終わることを検証する。
func TestRunNowPartialFailureKeepsItems(t *testing.T) {
	s := &stubDeps{
		path: writeConfig(t, "repos:\n  - org/app\n"),
		result: &fetch.Result{
			Cards:  []model.Card{card("org/app", nil, []model.PR{statefulPR()}, model.SituationC, true)},
			Errors: []error{errors.New("ViewPR org/app#131: gh: exit 1")},
		},
	}

	code, stdout, stderr := runNowWith(t, s)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", code, stderr)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("Unmarshal: %v (%q)", err, stdout)
	}
	if items := got["items"].([]any); len(items) != 1 {
		t.Errorf("items = %d 件, want 1", len(items))
	}
	errs := got["errors"].([]any)
	if len(errs) != 1 || errs[0] != "ViewPR org/app#131: gh: exit 1" {
		t.Errorf("errors = %v, want 失敗 1 件", errs)
	}
}

// TestRunNowLeavesSnapshotUntouched は既定の場所のスナップショットを読み書きしないことを検証する。
// HOME を差し替えて snapshot.DefaultPath が指す実際のパスに置く（別の場所に置くと常に通るテストになる）。
func TestRunNowLeavesSnapshotUntouched(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	snapPath, err := snapshot.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(snapPath), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	const body = `{"cards":[],"at":"2026-09-01T00:00:00Z"}`
	if err := os.WriteFile(snapPath, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	before, err := os.Stat(snapPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	s := &stubDeps{
		path:   writeConfig(t, "repos:\n  - org/app\n"),
		result: &fetch.Result{Cards: []model.Card{card("org/app", nil, []model.PR{statefulPR()}, model.SituationC, true)}},
	}

	if code, _, stderr := runNowWith(t, s); code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", code, stderr)
	}

	after, err := os.Stat(snapPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	got, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != body {
		t.Errorf("スナップショットの中身が変わった: %q", got)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Errorf("スナップショットの更新時刻が変わった: %v → %v", before.ModTime(), after.ModTime())
	}
}
