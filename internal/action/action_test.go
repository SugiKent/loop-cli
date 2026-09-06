package action

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/SugiKent/loop-cli/internal/gh"
)

const fixtureDir = "../gh/testdata/fixtures/example"

// newExampleFake は example fixture を読む Fake。テストごとに Calls を空から始める。
func newExampleFake() *gh.Fake { return gh.NewFake(fixtureDir) }

// TestCommentWritesToTheKindOfTarget は不変条件 4「書き先 3 種類を混同しない」のうち
// PR / issue の 2 種類を検証する。
func TestCommentWritesToTheKindOfTarget(t *testing.T) {
	for name, tc := range map[string]struct {
		target Target
		body   string
		want   gh.Call
	}{
		"PR は CommentPR": {
			target: Target{Repo: "org/app", Number: 131, IsPR: true},
			body:   "Q1: A\nQ2: B",
			want:   gh.Call{Method: "CommentPR", Repo: "org/app", Number: 131, Body: "Q1: A\nQ2: B"},
		},
		"issue は CommentIssue": {
			target: Target{Repo: "org/app", Number: 108, IsPR: false},
			body:   "Q1: A",
			want:   gh.Call{Method: "CommentIssue", Repo: "org/app", Number: 108, Body: "Q1: A"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			client := newExampleFake()
			if err := Comment(context.Background(), client, tc.target, tc.body); err != nil {
				t.Fatalf("Comment: %v", err)
			}
			if len(client.Calls) != 1 {
				t.Fatalf("呼び出し = %d 件, want 1: %+v", len(client.Calls), client.Calls)
			}
			if client.Calls[0] != tc.want {
				t.Errorf("呼び出し = %+v, want %+v", client.Calls[0], tc.want)
			}
		})
	}
}

// TestCommentNeverTouchesLabels は不変条件 2。TUI の回答はラベルを書き換えない。
func TestCommentNeverTouchesLabels(t *testing.T) {
	client := newExampleFake()
	ctx := context.Background()
	if err := Comment(ctx, client, Target{Repo: "org/app", Number: 131, IsPR: true}, "Q1: A"); err != nil {
		t.Fatalf("Comment(PR): %v", err)
	}
	if err := Comment(ctx, client, Target{Repo: "org/app", Number: 108}, "Q1: A"); err != nil {
		t.Fatalf("Comment(issue): %v", err)
	}

	if len(client.Calls) != 2 {
		t.Fatalf("呼び出し = %d 件, want 2: %+v", len(client.Calls), client.Calls)
	}
	for _, c := range client.Calls {
		if c.Method == "AddLabel" || c.Method == "RemoveLabel" {
			t.Errorf("ラベルを書き換えている: %+v", c)
		}
	}
}

// TestCommentRejects は投稿してはならない本文を、client を呼ぶ前に止めることを検証する。
func TestCommentRejects(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		want error
	}{
		"空白だけ":         {body: " \n\t\n", want: ErrEmptyBody},
		"マーカーで始まる":     {body: "<!-- routine -->\nQ1: A", want: ErrRoutineMarker},
		"マーカーが文中にある":   {body: "routine のコメントは <!-- routine --> で始まるはずでは？", want: ErrRoutineMarker},
		"エスケープされたマーカー": {body: "&lt;!-- routine --&gt;\nQ1: A", want: ErrRoutineMarker},
	} {
		t.Run(name, func(t *testing.T) {
			client := newExampleFake()
			err := Comment(context.Background(), client, Target{Repo: "org/app", Number: 131, IsPR: true}, tc.body)
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
			if len(client.Calls) != 0 {
				t.Errorf("拒否したのに client を呼んでいる: %+v", client.Calls)
			}
		})
	}
}

// TestCommentAcceptsBlockedByLines は不変条件 8 が「警告」であって拒否ではないことを検証する。
func TestCommentAcceptsBlockedByLines(t *testing.T) {
	client := newExampleFake()
	body := "blocked-by: human\nQ1: A"
	if err := Comment(context.Background(), client, Target{Repo: "org/app", Number: 108}, body); err != nil {
		t.Fatalf("Comment: %v", err)
	}
	want := gh.Call{Method: "CommentIssue", Repo: "org/app", Number: 108, Body: body}
	if len(client.Calls) != 1 || client.Calls[0] != want {
		t.Errorf("呼び出し = %+v, want 1 件の %+v", client.Calls, want)
	}
}

func TestBlockedByLines(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		want []string
	}{
		"行頭と字下げの両方を拾う": {
			body: "Q1: A\n  blocked-by: human\nQ2: B\nblocked-by: #12",
			want: []string{"blocked-by: human", "blocked-by: #12"},
		},
		"引用の > は吸収しない": {
			body: "Q1: A\n> blocked-by: human を引用します\nQ2: B",
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := BlockedByLines(tc.body); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("BlockedByLines = %q, want %q", got, tc.want)
			}
		})
	}
}
