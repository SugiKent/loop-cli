package action

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// failingMergePR は MergePR だけが失敗する client。
type failingMergePR struct{ *gh.Fake }

var errNotMergeable = errors.New("gh pr merge 151 -R org/app --squash: exit 1: Pull request is not mergeable")

func (f failingMergePR) MergePR(_ context.Context, _ string, _ int, _ string) error {
	return errNotMergeable
}

// greenChecks は CheckRun の test が SUCCESS 1 件だけの merge 状態。
func greenChecks() *gh.PRMergeState {
	return &gh.PRMergeState{
		Mergeable:        "MERGEABLE",
		MergeStateStatus: "CLEAN",
		StatusCheckRollup: []gh.StatusCheck{
			{Typename: "CheckRun", Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
		},
	}
}

// pendingChecks は StatusContext の ci/legacy が PENDING の merge 状態。
func pendingChecks() *gh.PRMergeState {
	return &gh.PRMergeState{
		Mergeable:        "UNKNOWN",
		MergeStateStatus: "BLOCKED",
		StatusCheckRollup: []gh.StatusCheck{
			{Typename: "StatusContext", Context: "ci/legacy", State: "PENDING"},
		},
	}
}

// TestCheckMerge は draft と open でない PR が拒否になり、他の判断材料が警告として並ぶことを検証する。
func TestCheckMerge(t *testing.T) {
	tests := []struct {
		name         string
		pr           model.PR
		wantBlocked  []string
		wantWarnings []string
	}{
		{
			name: "draft は拒否になる",
			pr: model.PR{
				IsDraft:    true,
				Body:       "未確定の判断: 0 件",
				MergeState: &gh.PRMergeState{},
			},
			wantBlocked: []string{"draft の PR です"},
		},
		{
			name: "merged 済みの PR は拒否になる",
			pr: model.PR{
				State:      "MERGED",
				Body:       "未確定の判断: 0 件",
				MergeState: &gh.PRMergeState{},
			},
			wantBlocked: []string{"MERGED の PR です"},
		},
		{
			name: "AI 評価が未完了なら警告になる",
			pr: model.PR{
				Labels:     []string{model.LabelPropose, model.LabelAIAssess},
				Body:       "未確定の判断: 0 件",
				MergeState: greenChecks(),
			},
			wantWarnings: []string{"ai-assess:requested が付いています（AI 評価が未完了）"},
		},
		{
			name: "question と未確定と checks は警告になる",
			pr: model.PR{
				Labels:     []string{model.LabelPropose, model.LabelQuestion},
				Body:       "未確定の判断: 2 件 — merge しないでください",
				MergeState: pendingChecks(),
			},
			wantWarnings: []string{
				"question ラベルが付いています",
				"本文 1 行目が「未確定の判断: 2 件」です",
				"checks が緑ではありません",
			},
		},
		{
			name: "判断材料が揃っていれば空の列が返る",
			pr: model.PR{
				Labels:     []string{model.LabelApply},
				Body:       "未確定の判断: 0 件",
				MergeState: greenChecks(),
			},
		},
		{
			name: "1 行目が未確定の形でなければ未確定の警告は出ない",
			pr: model.PR{
				Body:       "issue #108 の提案。\n\nCloses #108",
				MergeState: &gh.PRMergeState{},
			},
		},
		{
			name: "MergeState が nil なら checks の警告が出る",
			pr:   model.PR{Body: "未確定の判断: 0 件"},
			wantWarnings: []string{
				"checks が緑ではありません",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked, warnings := CheckMerge(tt.pr)
			if strings.Join(blocked, "|") != strings.Join(tt.wantBlocked, "|") {
				t.Errorf("blocked = %q, want %q", blocked, tt.wantBlocked)
			}
			if strings.Join(warnings, "|") != strings.Join(tt.wantWarnings, "|") {
				t.Errorf("warnings = %q, want %q", warnings, tt.wantWarnings)
			}
		})
	}
}

// TestMergeCallsMergePROnce は設定の方式で MergePR がちょうど 1 回呼ばれることを検証する。
func TestMergeCallsMergePROnce(t *testing.T) {
	for _, method := range []string{"squash", "merge", "rebase"} {
		t.Run(method, func(t *testing.T) {
			fake := gh.NewFake("")
			if err := Merge(context.Background(), fake, "org/app", 151, method); err != nil {
				t.Fatalf("Merge(%s): %v", method, err)
			}
			want := gh.Call{Method: "MergePR", Repo: "org/app", Number: 151, MergeMethod: method}
			if len(fake.Calls) != 1 || fake.Calls[0] != want {
				t.Fatalf("呼び出し = %+v, want 1 件の %+v", fake.Calls, want)
			}
			assertNoWrites(t, fake.Calls)
		})
	}
}

// TestMergeRejectsBadMethod は方式が不正なら client を呼ばないことを検証する。
func TestMergeRejectsBadMethod(t *testing.T) {
	for _, method := range []string{"", "SQUASH"} {
		t.Run("method="+method, func(t *testing.T) {
			fake := gh.NewFake("")
			err := Merge(context.Background(), fake, "org/app", 151, method)
			if !errors.Is(err, ErrBadMergeMethod) {
				t.Fatalf("err = %v, want ErrBadMergeMethod", err)
			}
			if !strings.Contains(err.Error(), method) {
				t.Errorf("err = %q, want %q を含む", err, method)
			}
			if len(fake.Calls) != 0 {
				t.Errorf("呼び出し = %+v, want 空", fake.Calls)
			}
		})
	}
}

// TestMergeReturnsGHFailure は gh の失敗がそのまま返ることと、失敗しても書き込みが起きないことを検証する。
func TestMergeReturnsGHFailure(t *testing.T) {
	fake := gh.NewFake("")
	err := Merge(context.Background(), failingMergePR{fake}, "org/app", 151, "squash")
	if !errors.Is(err, errNotMergeable) {
		t.Fatalf("err = %v, want %v", err, errNotMergeable)
	}
	assertNoWrites(t, fake.Calls)
}

// assertNoWrites は Merge がラベルもコメントも書かないことを確かめる（不変条件 2）。
func assertNoWrites(t *testing.T, calls []gh.Call) {
	t.Helper()
	for _, c := range calls {
		switch c.Method {
		case "AddLabel", "RemoveLabel", "CommentIssue", "CommentPR", "CreateIssue", "ReplyReviewThread", "Browse", "OpenURL":
			t.Errorf("Merge が %s を呼んでいる: %+v", c.Method, c)
		}
	}
}
