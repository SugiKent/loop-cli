// Package gh は GitHub のデータ層を gh CLI のサブプロセスとして提供する。
package gh

import (
	"context"
	"fmt"
	"strings"
)

// GHClient は loop-cli が使う GitHub 操作の集合。
// Client（gh サブプロセス）と Fake（JSON fixture）が実装する。
type GHClient interface {
	SearchIssues(ctx context.Context, repos []string) ([]SearchIssue, error)
	SearchPRs(ctx context.Context, repos []string) ([]SearchPR, error)
	ViewIssue(ctx context.Context, repo string, number int) (*IssueDetail, error)
	ViewPR(ctx context.Context, repo string, number int) (*PRDetail, error)
	ViewPRMergeState(ctx context.Context, repo string, number int) (*PRMergeState, error)
	ReviewThreads(ctx context.Context, repo string, number int) ([]ReviewThread, error)
	CrossReferencedPRs(ctx context.Context, repo string, number int) ([]CrossReferencedPR, error)
	LabelTimeline(ctx context.Context, repo string, number int) ([]LabelEvent, error)
	ListLabels(ctx context.Context, repo string) ([]RepoLabel, error)

	CommentIssue(ctx context.Context, repo string, number int, body string) error
	CommentPR(ctx context.Context, repo string, number int, body string) error
	AddLabel(ctx context.Context, repo string, number int, label string) error
	RemoveLabel(ctx context.Context, repo string, number int, label string) error
	EditIssueLabels(ctx context.Context, repo string, number int, add, remove []string) error
	EditPRLabels(ctx context.Context, repo string, number int, add, remove []string) error
	MergePR(ctx context.Context, repo string, number int, method string) error
	CloseIssue(ctx context.Context, repo string, number int) error
	ClosePR(ctx context.Context, repo string, number int) error
	CreateIssue(ctx context.Context, repo string, title string, body string) (string, error)
	ReplyReviewThread(ctx context.Context, repo string, number int, commentID int64, body string) error
	Browse(ctx context.Context, repo string, number int) error
	OpenURL(ctx context.Context, url string) error
}

// Error は gh が非 0 で終了したことを表す。呼び出し側は errors.As でデコード失敗と区別する。
type Error struct {
	Args     []string
	ExitCode int
	Stderr   string
}

func (e *Error) Error() string {
	return fmt.Sprintf("gh %s: exit %d: %s", strings.Join(e.Args, " "), e.ExitCode, strings.TrimSpace(e.Stderr))
}
