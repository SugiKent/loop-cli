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

// TestSplitNewIssueSplitsAtPlaceholders は 2 本のプレースホルダー行を区切りに
// タイトルと本文が分かれること、区切りが揃わない下書きを分割しないことを検証する。
func TestSplitNewIssueSplitsAtPlaceholders(t *testing.T) {
	for name, tc := range map[string]struct {
		text      string
		wantTitle string
		wantBody  string
		wantOK    bool
	}{
		"区切りに挟まれた行がタイトルになり下が本文になる": {
			text:      "タイトル（この下の行に入力してください）\nキュー画面の色を見直す\n\n概要（この下に入力してください）\n種別の色が背景色と競合している。\n",
			wantTitle: "キュー画面の色を見直す",
			wantBody:  "種別の色が背景色と競合している。",
			wantOK:    true,
		},
		"初期の下書きの空行に書き足した形でも分割できる": {
			text:      "タイトル（この下の行に入力してください）\nキュー画面の色を見直す\n概要（この下に入力してください）\n種別の色が背景色と競合している。\n",
			wantTitle: "キュー画面の色を見直す",
			wantBody:  "種別の色が背景色と競合している。",
			wantOK:    true,
		},
		"本文は概要の区切りより下すべてになる": {
			text:      "タイトル（この下の行に入力してください）\n色を見直す\n\n概要（この下に入力してください）\n\n背景色と競合している。\n\n直したい行は 2 つ。\n",
			wantTitle: "色を見直す",
			wantBody:  "背景色と競合している。\n\n直したい行は 2 つ。",
			wantOK:    true,
		},
		"タイトルに複数行書かれたら半角空白 1 つで連結する": {
			text:      "タイトル（この下の行に入力してください）\nキュー画面の色を見直す\n\n（配色）\n概要（この下に入力してください）\n本文",
			wantTitle: "キュー画面の色を見直す （配色）",
			wantBody:  "本文",
			wantOK:    true,
		},
		"行末の空白と CRLF は区切りの判定に影響せず本文に \\r が残らない": {
			text:      "タイトル（この下の行に入力してください）  \r\n色を見直す\r\n\r\n概要（この下に入力してください）\t\r\n背景色と競合している。\r\n直したい行は 2 つ。\r\n",
			wantTitle: "色を見直す",
			wantBody:  "背景色と競合している。\n直したい行は 2 つ。",
			wantOK:    true,
		},
		"本文の行末の空白は残る": {
			// Markdown は行末の空白 2 つを改行として読む。落とすと人が書いた改行が消える。
			text:      "タイトル（この下の行に入力してください）\n色を見直す\n\n概要（この下に入力してください）\n1 行目です  \n2 行目です\n",
			wantTitle: "色を見直す",
			wantBody:  "1 行目です  \n2 行目です",
			wantOK:    true,
		},
		"プレースホルダー行を消した下書きは分割できない": {
			text: "キュー画面の色を見直す\n\n種別の色が背景色と競合している。",
		},
		"概要のプレースホルダー行だけが無い下書きも分割できない": {
			text: "タイトル（この下の行に入力してください）\nキュー画面の色を見直す\n\n種別の色が背景色と競合している。",
		},
		"概要の区切りがタイトルの区切りより前にある下書きは分割できない": {
			text: "概要（この下に入力してください）\n本文\n\nタイトル（この下の行に入力してください）\nタイトル",
		},
		"プレースホルダー行に書き足した下書きは分割できない": {
			text: "タイトル（この下の行に入力してください）キュー画面の色を見直す\n\n概要（この下に入力してください）\n本文",
		},
		"案内の行を複製した下書きは分割できない": {
			text: "タイトル（この下の行に入力してください）\nキュー画面の色を見直す\nタイトル（この下の行に入力してください）\n\n概要（この下に入力してください）\n本文",
		},
		"案内の文言を本文に引用した下書きは分割できない": {
			text: "タイトル（この下の行に入力してください）\n案内の文言を直したい\n\n概要（この下に入力してください）\n概要（この下に入力してください）\nこの行の文言を変えたい",
		},
	} {
		t.Run(name, func(t *testing.T) {
			title, body, ok := SplitNewIssue(tc.text)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if title != tc.wantTitle {
				t.Errorf("title = %q, want %q", title, tc.wantTitle)
			}
			if body != tc.wantBody {
				t.Errorf("body = %q, want %q", body, tc.wantBody)
			}
		})
	}
}

// TestNewIssueDraftCarriesBothPlaceholders は初期の下書きが区切り 2 本を持ち、
// 書かずに保存するとタイトルも本文も空になることを検証する。
func TestNewIssueDraftCarriesBothPlaceholders(t *testing.T) {
	if NewIssueDraft != "タイトル（この下の行に入力してください）\n\n概要（この下に入力してください）\n\n" {
		t.Fatalf("NewIssueDraft = %q", NewIssueDraft)
	}
	title, body, ok := SplitNewIssue(NewIssueDraft)
	if !ok {
		t.Fatalf("ok = false, want true（初期の下書きは自分で分割できなければならない）")
	}
	if title != "" || body != "" {
		t.Errorf("title = %q, body = %q, want どちらも空", title, body)
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
