package action

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

const todoFixtureDir = "testdata/todo"

// TestTodoFixturesDecode は切り替えの fixture が gh issue view の形として読めることを確かめる。
func TestTodoFixturesDecode(t *testing.T) {
	for number, want := range map[int][]string{
		150: {"stage:todo", "bug"},
		151: {"stage:propose", "question"},
		152: {"blocked"},
		153: nil,
		154: {"bug", "enhancement"},
		155: {"stage:todo", "stage:propose"},
	} {
		detail, err := gh.NewFake(todoFixtureDir).ViewIssue(context.Background(), "org/app", number)
		if err != nil {
			t.Fatalf("ViewIssue(%d): %v", number, err)
		}
		if detail.Number != number {
			t.Errorf("issue-%d.json の number = %d", number, detail.Number)
		}
		var got []string
		for _, l := range detail.Labels {
			got = append(got, l.Name)
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("issue-%d.json の labels = %q, want %q", number, got, want)
		}
	}
}

// TestToggleTodo は判定と、不変条件 1（1 操作 1 ラベル）の呼び出し列を検証する。
func TestToggleTodo(t *testing.T) {
	for name, tc := range map[string]struct {
		number    int
		wantAdded bool
		wantErr   error
		wantIn    []string // err.Error() に含まれるべき文字列
		wantCalls []gh.Call
	}{
		"ラベル無しには付ける": {
			number: 153, wantAdded: true,
			wantCalls: []gh.Call{
				{Method: "ViewIssue", Repo: "org/app", Number: 153},
				{Method: "AddLabel", Repo: "org/app", Number: 153, Label: "stage:todo"},
			},
		},
		"stage:todo は外す": {
			number: 150,
			wantCalls: []gh.Call{
				{Method: "ViewIssue", Repo: "org/app", Number: 150},
				{Method: "RemoveLabel", Repo: "org/app", Number: 150, Label: "stage:todo"},
			},
		},
		"段階以外のラベルだけなら付ける": {
			number: 154, wantAdded: true,
			wantCalls: []gh.Call{
				{Method: "ViewIssue", Repo: "org/app", Number: 154},
				{Method: "AddLabel", Repo: "org/app", Number: 154, Label: "stage:todo"},
			},
		},
		"別の段階ラベルは拒否": {
			number: 151, wantErr: ErrOtherStage, wantIn: []string{"stage:propose"},
			wantCalls: []gh.Call{{Method: "ViewIssue", Repo: "org/app", Number: 151}},
		},
		"段階ラベル 2 つは stage:todo を含んでも拒否": {
			number: 155, wantErr: ErrMultipleStages, wantIn: []string{"stage:todo", "stage:propose"},
			wantCalls: []gh.Call{{Method: "ViewIssue", Repo: "org/app", Number: 155}},
		},
		"blocked は拒否": {
			number: 152, wantErr: ErrBlocked, wantIn: []string{"blocked"},
			wantCalls: []gh.Call{{Method: "ViewIssue", Repo: "org/app", Number: 152}},
		},
		"ViewIssue の失敗では書き込まない": {
			number: 999, wantErr: nil, wantIn: []string{"issue-999.json"},
			wantCalls: []gh.Call{{Method: "ViewIssue", Repo: "org/app", Number: 999}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			client := gh.NewFake(todoFixtureDir)
			added, err := ToggleTodo(context.Background(), client, "org/app", tc.number, model.ModeSDD)

			switch {
			case len(tc.wantIn) == 0 && err != nil:
				t.Fatalf("err = %v, want nil", err)
			case tc.wantErr != nil && !errors.Is(err, tc.wantErr):
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			case len(tc.wantIn) > 0 && err == nil:
				t.Fatal("err = nil, want エラー")
			}
			for _, sub := range tc.wantIn {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("err = %q に %q が無い", err, sub)
				}
			}
			if added != tc.wantAdded {
				t.Errorf("added = %v, want %v", added, tc.wantAdded)
			}
			if len(client.Calls) != len(tc.wantCalls) {
				t.Fatalf("呼び出し = %+v, want %+v", client.Calls, tc.wantCalls)
			}
			for i, want := range tc.wantCalls {
				if !reflect.DeepEqual(client.Calls[i], want) {
					t.Errorf("%d 件目 = %+v, want %+v", i+1, client.Calls[i], want)
				}
			}
		})
	}
}

// TestToggleTodoLabelMode は label 方式で書くラベルが To Do になり、判定が ILD の語彙で行われることを見る。
func TestToggleTodoLabelMode(t *testing.T) {
	for name, tc := range map[string]struct {
		number    int
		wantAdded bool
		wantErr   error
		wantIn    []string
		wantCalls []gh.Call
	}{
		"ラベル無しには To Do を付ける": {
			number: 160, wantAdded: true,
			wantCalls: []gh.Call{
				{Method: "ViewIssue", Repo: "org/board", Number: 160},
				{Method: "AddLabel", Repo: "org/board", Number: 160, Label: "To Do"},
			},
		},
		"To Do は外す": {
			number: 161,
			wantCalls: []gh.Call{
				{Method: "ViewIssue", Repo: "org/board", Number: 161},
				{Method: "RemoveLabel", Repo: "org/board", Number: 161, Label: "To Do"},
			},
		},
		"In Progress は拒否": {
			number: 162, wantErr: ErrOtherStage, wantIn: []string{"In Progress"},
			wantCalls: []gh.Call{{Method: "ViewIssue", Repo: "org/board", Number: 162}},
		},
		"sdd の段階ラベルは段階として数えない": {
			number: 163, wantAdded: true,
			wantCalls: []gh.Call{
				{Method: "ViewIssue", Repo: "org/board", Number: 163},
				{Method: "AddLabel", Repo: "org/board", Number: 163, Label: "To Do"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			client := gh.NewFake(todoFixtureDir)
			added, err := ToggleTodo(context.Background(), client, "org/board", tc.number, model.ModeLabel)

			if tc.wantErr == nil && err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			for _, sub := range tc.wantIn {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("err = %q に %q が無い", err, sub)
				}
			}
			if added != tc.wantAdded {
				t.Errorf("added = %v, want %v", added, tc.wantAdded)
			}
			if len(client.Calls) != len(tc.wantCalls) {
				t.Fatalf("呼び出し = %+v, want %+v", client.Calls, tc.wantCalls)
			}
			for i, want := range tc.wantCalls {
				if !reflect.DeepEqual(client.Calls[i], want) {
					t.Errorf("%d 件目 = %+v, want %+v", i+1, client.Calls[i], want)
				}
			}
		})
	}
}

// TestToggleTodoWritesOnlyTodoLabel は不変条件 2。全 Scenario を通してラベルは方式ごとの承認ラベルだけで、
// ラベル以外の書き込みメソッドを呼ばない。
func TestToggleTodoWritesOnlyTodoLabel(t *testing.T) {
	client := gh.NewFake(todoFixtureDir)
	for _, n := range []int{150, 151, 152, 153, 154, 155, 999} {
		_, _ = ToggleTodo(context.Background(), client, "org/app", n, model.ModeSDD)
	}
	sddCalls := len(client.Calls)
	for _, n := range []int{160, 161, 162, 163} {
		_, _ = ToggleTodo(context.Background(), client, "org/board", n, model.ModeLabel)
	}
	for i, c := range client.Calls {
		want := "stage:todo"
		if i >= sddCalls {
			want = "To Do"
		}
		switch c.Method {
		case "ViewIssue":
		case "AddLabel", "RemoveLabel":
			if c.Label != want {
				t.Errorf("%q 以外のラベルを書いている: %+v", want, c)
			}
		default:
			t.Errorf("ラベル以外の書き込みを呼んでいる: %+v", c)
		}
	}
}
