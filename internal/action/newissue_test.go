package action

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/SugiKent/loop-cli/internal/gh"
)

// failingCreateIssue は CreateIssue だけが失敗する client。
type failingCreateIssue struct{ *gh.Fake }

func (f failingCreateIssue) CreateIssue(context.Context, string, string, string) (string, error) {
	return "", errors.New("gh issue create -R org/app: exit 1: HTTP 403")
}

// TestSplitNewIssueTakesFirstLineAsTitle は 1 行目がタイトル、残りが本文になることを検証する。
func TestSplitNewIssueTakesFirstLineAsTitle(t *testing.T) {
	for name, tc := range map[string]struct {
		text      string
		wantTitle string
		wantBody  string
	}{
		"1 行目がタイトルになり空行が落ちる": {
			text:      "n キーで issue を作る\n\n選択中の repo に作る。\n本文はここから。\n",
			wantTitle: "n キーで issue を作る",
			wantBody:  "選択中の repo に作る。\n本文はここから。",
		},
		"1 行だけの下書きは本文が空になる": {
			text:      "  タイトルだけ  ",
			wantTitle: "タイトルだけ",
			wantBody:  "",
		},
		"空行から始まる下書きはタイトルが空になる": {
			text:      "\n本文だけ書いた",
			wantTitle: "",
			wantBody:  "本文だけ書いた",
		},
	} {
		t.Run(name, func(t *testing.T) {
			title, body := SplitNewIssue(tc.text)
			if title != tc.wantTitle {
				t.Errorf("title = %q, want %q", title, tc.wantTitle)
			}
			if body != tc.wantBody {
				t.Errorf("body = %q, want %q", body, tc.wantBody)
			}
		})
	}
}

// TestCreateIssueReturnsURLWithoutLabels は作成が 1 回で URL を返し、ラベルを書かないことを検証する。
func TestCreateIssueReturnsURLWithoutLabels(t *testing.T) {
	fake := gh.NewFake("")
	url, err := CreateIssue(context.Background(), fake, "org/app", "タイトル", "本文")
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if url != "https://github.com/org/app/issues/0" {
		t.Errorf("URL = %q", url)
	}
	want := []gh.Call{{Method: "CreateIssue", Repo: "org/app", Title: "タイトル", Body: "本文"}}
	if !reflect.DeepEqual(fake.Calls, want) {
		t.Errorf("呼び出し = %+v, want %+v", fake.Calls, want)
	}
	assertNoIssueWrites(t, fake.Calls)
}

// TestCreateIssueReturnsGHFailure は gh の失敗がそのまま返ることを検証する。
func TestCreateIssueReturnsGHFailure(t *testing.T) {
	fake := gh.NewFake("")
	url, err := CreateIssue(context.Background(), failingCreateIssue{fake}, "org/app", "タイトル", "本文")
	if url != "" {
		t.Errorf("URL = %q, want 空", url)
	}
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("err = %v, want HTTP 403 を含むエラー", err)
	}
	assertNoIssueWrites(t, fake.Calls)
}

// TestCreateIssueRejectsBeforeCallingGH は空とマーカーの下書きで gh を呼ばないことを検証する。
func TestCreateIssueRejectsBeforeCallingGH(t *testing.T) {
	for name, tc := range map[string]struct {
		title, body string
		wantErr     error
	}{
		"タイトルが空":  {title: "   ", body: "本文", wantErr: ErrEmptyTitle},
		"本文が空":    {title: "タイトル", body: "\n\t\n", wantErr: ErrEmptyBody},
		"本文にマーカー": {title: "タイトル", body: "routine のコメントは <!-- routine --> で始まる", wantErr: ErrRoutineMarker},
		"タイトルにエスケープ済みのマーカー": {title: "&lt;!-- routine --&gt; を含むタイトル", body: "本文", wantErr: ErrRoutineMarker},
	} {
		t.Run(name, func(t *testing.T) {
			fake := gh.NewFake("")
			url, err := CreateIssue(context.Background(), fake, "org/app", tc.title, tc.body)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if url != "" {
				t.Errorf("URL = %q, want 空", url)
			}
			if len(fake.Calls) != 0 {
				t.Errorf("呼び出し = %+v, want 空", fake.Calls)
			}
		})
	}
}

// assertNoIssueWrites は CreateIssue がラベルもコメントも merge も書かないことを確かめる（不変条件 2 / 6）。
func assertNoIssueWrites(t *testing.T, calls []gh.Call) {
	t.Helper()
	for _, c := range calls {
		switch c.Method {
		case "AddLabel", "RemoveLabel", "CommentIssue", "CommentPR", "MergePR", "ReplyReviewThread":
			t.Errorf("CreateIssue が %s を呼んでいる: %+v", c.Method, c)
		}
	}
}
