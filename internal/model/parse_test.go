package model

import "testing"

func TestIsAI(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"マーカーで始まる", "<!-- routine -->\n以下 2 点、回答をお願いします", true},
		{"エスケープ済みマーカーで始まる", "&lt;!-- routine --&gt;\nblocked-by: human", true},
		{"3 行目に PR リスク評価の見出し", "PR #131 の評価です。\n\n## PR リスク評価\n\n- 影響範囲: …", true},
		{"マーカーが無い", "Q1: A\nQ2: B", false},
		{"マーカーを途中で引用", "routine のコメントは <!-- routine --> で始まるはずでは？", false},
		{"先頭に空白がある", " <!-- routine -->\nQ1: A", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAI(tt.body); got != tt.want {
				t.Errorf("IsAI(%q) = %v, want %v", tt.body, got, tt.want)
			}
		})
	}
}

func TestParseUndecided(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		wantN  int
		wantOK bool
	}{
		{"0 件", "未確定の判断: 0 件\n\n## 概要\n…", 0, true},
		{"2 件と補足", "未確定の判断: 2 件 — merge しないでください\n…", 2, true},
		{"1 行目に無い", "issue #108 の提案。\n\n未確定の判断: 0 件", 0, false},
		{"先頭が空行でも 1 行目として読む", "\n\n未確定の判断: 3 件", 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, ok := ParseUndecided(tt.body)
			if n != tt.wantN || ok != tt.wantOK {
				t.Errorf("ParseUndecided = %d, %v; want %d, %v", n, ok, tt.wantN, tt.wantOK)
			}
		})
	}
}

func TestLatestBlockedByReturnsNewestMatchingComment(t *testing.T) {
	comments := []Comment{
		{Body: "<!-- routine -->\nblocked-by: #12"},
		{Body: "了解です"},
		{Body: "<!-- routine -->\nblocked-by: human\n## Q1. 方針を決めてください"},
	}

	c, value, ok := LatestBlockedBy(comments)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if c != &comments[2] {
		t.Errorf("3 件目のコメントが返るべき（返ったのは %q）", c.Body)
	}
	if value != "human" {
		t.Errorf("value = %q, want human", value)
	}
}

func TestLatestBlockedByPicksTheLastOfIdenticalComments(t *testing.T) {
	comments := []Comment{
		{Body: "<!-- routine -->\nblocked-by: human"},
		{Body: "<!-- routine -->\nblocked-by: human"},
		{Body: "<!-- routine -->\nblocked-by: human"},
	}

	c, _, ok := LatestBlockedBy(comments)
	if !ok || c != &comments[2] {
		t.Errorf("末尾のコメントが返るべき（ok=%v）", ok)
	}
}

func TestLatestBlockedByWithoutMatch(t *testing.T) {
	c, value, ok := LatestBlockedBy([]Comment{{Body: "Q1: A"}, {Body: "ありがとうございます"}})
	if c != nil || value != "" || ok {
		t.Errorf("LatestBlockedBy = %v, %q, %v; want nil, \"\", false", c, value, ok)
	}
}

func TestParseQuestions(t *testing.T) {
	body := "以下 2 点、回答をお願いします\n" +
		"## Q1. 名前での絞り込みを今回のスコープに含めるか\n" +
		"- 選択肢 A（推奨）: 含めない。次の issue に回す\n" +
		"- 選択肢 B: 含める\n" +
		"## Q2. カードの情報量の見直しをどこまで行うか\n" +
		"- 選択肢 A: 今回は触らない\n" +
		"- 選択肢 B（推奨）: 幅だけ直す"

	qs := ParseQuestions(body)
	if len(qs) != 2 {
		t.Fatalf("質問の件数 = %d, want 2", len(qs))
	}
	if qs[0].Number != 1 || qs[0].Title != "名前での絞り込みを今回のスコープに含めるか" {
		t.Errorf("Q1 = %+v", qs[0])
	}
	want := []Option{
		{Letter: "A", Text: "含めない。次の issue に回す", Recommended: true},
		{Letter: "B", Text: "含める"},
	}
	if len(qs[0].Options) != 2 || qs[0].Options[0] != want[0] || qs[0].Options[1] != want[1] {
		t.Errorf("Q1 の選択肢 = %+v, want %+v", qs[0].Options, want)
	}
	if qs[1].Number != 2 || len(qs[1].Options) != 2 || !qs[1].Options[1].Recommended {
		t.Errorf("Q2 = %+v", qs[1])
	}
}

func TestParseQuestionsAcceptsFullWidthColon(t *testing.T) {
	qs := ParseQuestions("## Q1. どうするか\n- 選択肢 A（推奨）：含めない")
	if len(qs) != 1 || len(qs[0].Options) != 1 {
		t.Fatalf("qs = %+v", qs)
	}
	if got := qs[0].Options[0]; got.Text != "含めない" || !got.Recommended {
		t.Errorf("選択肢 = %+v", got)
	}
}

func TestParseQuestionsWithoutHeadingIsEmpty(t *testing.T) {
	qs := ParseQuestions("<!-- routine -->\nblocked-by: human\n次の方針をコメントで教えてください")
	if len(qs) != 0 {
		t.Errorf("ParseQuestions = %+v, want 空", qs)
	}
}

// TestParseLinks は本文から取り出す URL とリンクテキストを検証する。
func TestParseLinks(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []Link
	}{
		{
			name: "裸の URL と Markdown リンクを出現順に取る",
			body: "設計は [設計メモ](https://example.com/design) を見る。\n参考: https://example.com/ref\n",
			want: []Link{
				{Text: "設計メモ", URL: "https://example.com/design"},
				{URL: "https://example.com/ref"},
			},
		},
		{
			name: "末尾の句読点と閉じ括弧を含めない",
			body: "詳細は https://example.com/a。\n（https://example.com/b）\n<https://example.com/c>",
			want: []Link{
				{URL: "https://example.com/a"},
				{URL: "https://example.com/b"},
				{URL: "https://example.com/c"},
			},
		},
		{
			name: "参照記法と相対リンクは取らない",
			body: "Closes #108\nissue org/app#140 も参照。\n[手順](./docs/mvp/mvp.md) と @user-1",
		},
		{
			name: "同じ URL が 2 回あれば 2 件返る",
			body: "https://example.com/a と https://example.com/a",
			want: []Link{{URL: "https://example.com/a"}, {URL: "https://example.com/a"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLinks(tt.body)
			if len(got) != len(tt.want) {
				t.Fatalf("件数 = %d (%+v), want %d", len(got), got, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("%d 件目 = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
