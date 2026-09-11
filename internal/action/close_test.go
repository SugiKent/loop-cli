package action

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/SugiKent/loop-cli/internal/gh"
)

// failingCloseIssue は CloseIssue だけが失敗する client。
type failingCloseIssue struct{ *gh.Fake }

var errCannotClose = errors.New("gh issue close 108 -R org/app: exit 1: could not close issue")

func (f failingCloseIssue) CloseIssue(_ context.Context, _ string, _ int) error {
	return errCannotClose
}

// TestCloseWritesToKindOfTarget は不変条件 4「書き先を混同しない」を Calls で検証する。
func TestCloseWritesToKindOfTarget(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		want   []gh.Call
	}{
		{
			name:   "PR は ClosePR に行く",
			target: Target{Repo: "org/app", Number: 131, IsPR: true},
			want:   []gh.Call{{Method: "ClosePR", Repo: "org/app", Number: 131}},
		},
		{
			name:   "issue は CloseIssue に行く",
			target: Target{Repo: "org/app", Number: 108},
			want:   []gh.Call{{Method: "CloseIssue", Repo: "org/app", Number: 108}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := gh.NewFake("")
			if err := Close(t.Context(), fake, tt.target); err != nil {
				t.Fatalf("Close: %v", err)
			}
			if !reflect.DeepEqual(fake.Calls, tt.want) {
				t.Errorf("Calls = %+v, want %+v", fake.Calls, tt.want)
			}
		})
	}
}

// TestCloseWritesNoLabelsOrComments は不変条件 2 を検証する。
func TestCloseWritesNoLabelsOrComments(t *testing.T) {
	fake := gh.NewFake("")
	ctx := t.Context()
	if err := Close(ctx, fake, Target{Repo: "org/app", Number: 108}); err != nil {
		t.Fatalf("issue の Close: %v", err)
	}
	if err := Close(ctx, fake, Target{Repo: "org/app", Number: 131, IsPR: true}); err != nil {
		t.Fatalf("PR の Close: %v", err)
	}
	for _, c := range fake.Calls {
		switch c.Method {
		case "AddLabel", "RemoveLabel", "EditIssueLabels", "EditPRLabels", "CommentIssue", "CommentPR":
			t.Errorf("ラベルかコメントを書いている: %+v", c)
		}
	}
}

func TestCloseReturnsGhError(t *testing.T) {
	fake := gh.NewFake("")
	client := failingCloseIssue{fake}

	err := Close(t.Context(), client, Target{Repo: "org/app", Number: 108})
	if !errors.Is(err, errCannotClose) {
		t.Fatalf("err = %v, want %v", err, errCannotClose)
	}
	for _, c := range fake.Calls {
		if c.Method == "ClosePR" {
			t.Errorf("issue の close で ClosePR を呼んでいる: %+v", c)
		}
	}
}

// TestCheckClose は「close が別の人の出番を作る」PR にだけ注意が出ることを検証する。
func TestCheckClose(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		labels []string
		want   []string
	}{
		{
			name:   "propose ラベルの PR には注意が出る",
			target: Target{Repo: "org/app", Number: 131, IsPR: true},
			labels: []string{"propose", "question"},
			want:   []string{rejectedPRNote},
		},
		{
			name:   "apply ラベルの PR にも注意が出る",
			target: Target{Repo: "org/app", Number: 132, IsPR: true},
			labels: []string{"apply"},
			want:   []string{rejectedPRNote},
		},
		{
			name:   "archive PR には注意が出ない",
			target: Target{Repo: "org/app", Number: 133, IsPR: true},
			labels: []string{"archive"},
		},
		{
			name:   "docs PR には注意が出ない",
			target: Target{Repo: "org/app", Number: 134, IsPR: true},
			labels: []string{"docs"},
		},
		{
			name:   "ラベル無しの PR には注意が出ない",
			target: Target{Repo: "org/app", Number: 135, IsPR: true},
		},
		{
			name:   "propose ラベルでも issue には注意が出ない",
			target: Target{Repo: "org/app", Number: 108},
			labels: []string{"propose"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckClose(tt.target, tt.labels); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckClose = %#v, want %#v", got, tt.want)
			}
		})
	}
}
