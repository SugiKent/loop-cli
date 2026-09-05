package gh

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

var (
	_ GHClient = (*Client)(nil)
	_ GHClient = (*Fake)(nil)
)

// Call は Fake が記録した呼び出し。使わないフィールドはゼロ値のまま。
type Call struct {
	Method      string
	Repo        string
	Number      int
	Body        string
	Label       string
	MergeMethod string
	Title       string
	CommentID   int64
}

// Fake は fixture ディレクトリから GHClient と同じ型を返す。
// ディレクトリ 1 つがリポジトリ 1 件に対応するので repo 引数はファイル探索に使わない。
type Fake struct {
	Dir   string
	Calls []Call
}

// NewFake は dir 配下の fixture を読む Fake を返す。
func NewFake(dir string) *Fake {
	return &Fake{Dir: dir}
}

func (f *Fake) read(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(f.Dir, name))
}

func (f *Fake) SearchIssues(_ context.Context, _ []string) ([]SearchIssue, error) {
	b, err := f.read("search-issues.json")
	if err != nil {
		return nil, err
	}
	return decodeSearchIssues(b)
}

func (f *Fake) SearchPRs(_ context.Context, _ []string) ([]SearchPR, error) {
	b, err := f.read("search-prs.json")
	if err != nil {
		return nil, err
	}
	return decodeSearchPRs(b)
}

// ViewIssue は読み取りだが Calls に記録する。
// 不変条件 3（ラベルを外す → 読み直す → 付ける）の順序をテストが検証するため。
func (f *Fake) ViewIssue(_ context.Context, repo string, number int) (*IssueDetail, error) {
	f.Calls = append(f.Calls, Call{Method: "ViewIssue", Repo: repo, Number: number})
	b, err := f.read(fmt.Sprintf("issue-%d.json", number))
	if err != nil {
		return nil, err
	}
	return decodeIssueDetail(b)
}

func (f *Fake) ViewPR(_ context.Context, _ string, number int) (*PRDetail, error) {
	b, err := f.read(fmt.Sprintf("pr-%d.json", number))
	if err != nil {
		return nil, err
	}
	return decodePRDetail(b)
}

// ViewPRMergeState は ViewPR と同じ pr-<n>.json を読む。再取得も待ちもしない。
func (f *Fake) ViewPRMergeState(_ context.Context, _ string, number int) (*PRMergeState, error) {
	b, err := f.read(fmt.Sprintf("pr-%d.json", number))
	if err != nil {
		return nil, err
	}
	return decodePRMergeState(b)
}

func (f *Fake) ReviewThreads(_ context.Context, _ string, number int) ([]ReviewThread, error) {
	b, err := f.read(fmt.Sprintf("pr-%d-review-threads.json", number))
	if err != nil {
		return nil, err
	}
	return decodeReviewThreads(b)
}

func (f *Fake) CrossReferencedPRs(_ context.Context, _ string, number int) ([]CrossReferencedPR, error) {
	b, err := f.read(fmt.Sprintf("issue-%d-cross-refs.json", number))
	if err != nil {
		return nil, err
	}
	return decodeCrossReferencedPRs(b)
}

func (f *Fake) LabelTimeline(_ context.Context, _ string, number int) ([]LabelEvent, error) {
	b, err := f.read(fmt.Sprintf("issue-%d-timeline.json", number))
	if err != nil {
		return nil, err
	}
	return decodeLabelEvents(b)
}

func (f *Fake) CommentIssue(_ context.Context, repo string, number int, body string) error {
	f.Calls = append(f.Calls, Call{Method: "CommentIssue", Repo: repo, Number: number, Body: body})
	return nil
}

func (f *Fake) CommentPR(_ context.Context, repo string, number int, body string) error {
	f.Calls = append(f.Calls, Call{Method: "CommentPR", Repo: repo, Number: number, Body: body})
	return nil
}

func (f *Fake) AddLabel(_ context.Context, repo string, number int, label string) error {
	f.Calls = append(f.Calls, Call{Method: "AddLabel", Repo: repo, Number: number, Label: label})
	return nil
}

func (f *Fake) RemoveLabel(_ context.Context, repo string, number int, label string) error {
	f.Calls = append(f.Calls, Call{Method: "RemoveLabel", Repo: repo, Number: number, Label: label})
	return nil
}

func (f *Fake) MergePR(_ context.Context, repo string, number int, method string) error {
	f.Calls = append(f.Calls, Call{Method: "MergePR", Repo: repo, Number: number, MergeMethod: method})
	return nil
}

func (f *Fake) CreateIssue(_ context.Context, repo string, title string, body string) (string, error) {
	f.Calls = append(f.Calls, Call{Method: "CreateIssue", Repo: repo, Title: title, Body: body})
	return fmt.Sprintf("https://github.com/%s/issues/0", repo), nil
}

func (f *Fake) ReplyReviewThread(_ context.Context, repo string, number int, commentID int64, body string) error {
	f.Calls = append(f.Calls, Call{Method: "ReplyReviewThread", Repo: repo, Number: number, CommentID: commentID, Body: body})
	return nil
}

func (f *Fake) Browse(_ context.Context, repo string, number int) error {
	f.Calls = append(f.Calls, Call{Method: "Browse", Repo: repo, Number: number})
	return nil
}
