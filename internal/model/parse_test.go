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

// TestParseQuestionsUpstreamFormat は routine が実際に投稿する書式（`###` の見出しと太字の選択肢）を読む。
// この書式が読めないと、局面 A で回答テンプレートが白紙のまま開く。
func TestParseQuestionsUpstreamFormat(t *testing.T) {
	body := "<!-- routine -->\n" +
		"未確定の判断が 2 件あります。`Q1: A` の形で返してください。\n" +
		"\n---\n\n" +
		"### Q1. 状態語の色を端末の背景に追随させるか\n" +
		"\n" +
		"**何の話か**: 説明の段落。\n" +
		"\n" +
		"- **選択肢 A（推奨）**: 端末の背景色を受け取り 2 組を切り替える\n" +
		"- **選択肢 B**: 状態語にも背景色を敷く\n" +
		"- 依存: なし\n" +
		"\n---\n\n" +
		"### Q2. 色を付ける範囲\n" +
		"\n" +
		"- **選択肢 A**（推奨）: 状態語まで広げる\n" +
		"- **選択肢 B**: ラベル名だけにする"

	qs := ParseQuestions(body)
	if len(qs) != 2 {
		t.Fatalf("質問の件数 = %d, want 2", len(qs))
	}
	if qs[0].Number != 1 || qs[0].Title != "状態語の色を端末の背景に追随させるか" {
		t.Errorf("Q1 = %+v", qs[0])
	}
	want := []Option{
		{Letter: "A", Text: "端末の背景色を受け取り 2 組を切り替える", Recommended: true},
		{Letter: "B", Text: "状態語にも背景色を敷く"},
	}
	if len(qs[0].Options) != 2 || qs[0].Options[0] != want[0] || qs[0].Options[1] != want[1] {
		t.Errorf("Q1 の選択肢 = %+v, want %+v（`- 依存: なし` は選択肢にしない）", qs[0].Options, want)
	}
	if qs[1].Number != 2 || len(qs[1].Options) != 2 || !qs[1].Options[0].Recommended {
		t.Errorf("Q2 = %+v", qs[1])
	}
}

// TestParseQuestionsWithoutHeadingMarkIsEmpty は `#` の見出しを持たない行を質問にしないことを見る。
// 緩めると、人が書いた `Q1: A` の回答コメントまで質問に化ける。
func TestParseQuestionsWithoutHeadingMarkIsEmpty(t *testing.T) {
	qs := ParseQuestions("<!-- routine -->\nQ1: マイグレーションを分けますか。\nQ2: 期限はいつですか。")
	if len(qs) != 0 {
		t.Errorf("ParseQuestions = %+v, want 空", qs)
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

func TestIsMidRelabel(t *testing.T) {
	tests := []struct {
		name     string
		comments []Comment
		want     bool
	}{
		{
			name:     "最新の routine コメントが restart",
			comments: []Comment{{Body: "<!-- routine -->\nadvance: stage:apply", AI: true}, {Body: "<!-- routine -->\nrestart: 1/3", AI: true}},
			want:     true,
		},
		{
			name:     "人のコメントが後にあっても最新の routine コメントを見る",
			comments: []Comment{{Body: "<!-- routine -->\nrelease: stage:apply", AI: true}, {Body: "了解しました"}},
			want:     true,
		},
		{
			name:     "通常の routine コメント",
			comments: []Comment{{Body: "<!-- routine -->\nblocked-by: human\nunblock-when: comment", AI: true}},
		},
		{
			name:     "AI のコメントが 1 件も無い",
			comments: []Comment{{Body: "restart: 1/3"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMidRelabel(ModeSDD, tt.comments); got != tt.want {
				t.Errorf("IsMidRelabel = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMidRelabelLabelModeOnlyMarksRestart(t *testing.T) {
	release := []Comment{{Body: "<!-- routine -->\nrelease: To Do", AI: true}}
	if IsMidRelabel(ModeLabel, release) {
		t.Error("label 方式で release: を残骸の目印にしている")
	}
	restart := []Comment{{Body: "<!-- routine -->\nrestart: 1/3", AI: true}}
	if !IsMidRelabel(ModeLabel, restart) {
		t.Error("label 方式で restart: を残骸の目印にしていない")
	}
}

func TestClosesIssue(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
		ok   bool
	}{
		{name: "本文の Closes を読む", body: "ラベル一覧をモーダルで出す。\n\nCloses #12", want: 12, ok: true},
		{name: "Refs は採らない", body: "Refs #48"},
		{name: "番号だけの言及は採らない", body: "#108 と同じ問題"},
		{name: "単語の途中は採らない", body: "xCloses #12"},
		{name: "大文字小文字を区別しない", body: "closes #7", want: 7, ok: true},
		{name: "最初の 1 件を返す", body: "Closes #3\nCloses #9", want: 3, ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ClosesIssue(tt.body)
			if got != tt.want || ok != tt.ok {
				t.Errorf("ClosesIssue(%q) = %d, %v, want %d, %v", tt.body, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestUnblockWhen(t *testing.T) {
	body := "<!-- routine -->\nblocked-by: human\nunblock-when: docs\n\n方針を決めてください"
	if got, ok := UnblockWhen(body); !ok || got != "docs" {
		t.Errorf("UnblockWhen = %q, %v; want docs, true", got, ok)
	}
	if got, ok := UnblockWhen("<!-- routine -->\nblocked-by: #589"); ok || got != "" {
		t.Errorf("UnblockWhen = %q, %v; want \"\", false", got, ok)
	}
}

func TestSessionURLID(t *testing.T) {
	body := "進行中です。\nhttps://claude.ai/code/session_01N9YWTYcwFdwhLD38CdASgA で見られます。\n" +
		"前の回は https://claude.ai/code/session_01OLD でした。"
	if got, ok := SessionURLID(body); !ok || got != "session_01N9YWTYcwFdwhLD38CdASgA" {
		t.Errorf("SessionURLID = %q, %v; want session_01N9YWTYcwFdwhLD38CdASgA, true", got, ok)
	}
	if got, ok := SessionURLID("URL の無い本文"); ok || got != "" {
		t.Errorf("SessionURLID = %q, %v; want \"\", false", got, ok)
	}
}

func TestLatestSessionID(t *testing.T) {
	comments := []Comment{
		{Body: "<!-- routine -->\nstarted: 2026-09-11T06:24:00Z\nsession: session_01AAA"},
		{Body: "人のコメント"},
		{Body: "&lt;!-- routine --&gt;\nsession: session_01BBB"},
	}
	if got, ok := LatestSessionID(comments); !ok || got != "session_01BBB" {
		t.Errorf("LatestSessionID = %q, %v; want session_01BBB, true", got, ok)
	}
}

func TestLatestSessionIDIgnoresHumanComment(t *testing.T) {
	comments := []Comment{{Body: "session: session_01CCC と書いてあるだけの人のコメント"}}
	if got, ok := LatestSessionID(comments); ok || got != "" {
		t.Errorf("LatestSessionID = %q, %v; want \"\", false", got, ok)
	}
}
