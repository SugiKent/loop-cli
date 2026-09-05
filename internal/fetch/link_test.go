package fetch

import "testing"

func TestLinkedIssue(t *testing.T) {
	tests := []struct {
		name  string
		title string
		body  string
		want  int
		wanto bool
	}{
		{"title の段階と番号で紐づく", "[propose] #108 選択 UI をモーダル化する", "", 108, true},
		{"本文の Closes で紐づく", "propose: ログインのセッション仕様", "issue #108 の提案。\n\nCloses #108", 108, true},
		{"本文の Refs で紐づく", "docs: README を直す", "Refs #48", 48, true},
		{"title が本文より優先される", "[apply] #48 監視ツールを導入する", "Refs #12\nCloses #48", 48, true},
		{"番号だけの言及は紐づけない", "fix typo", "#108 と同じ問題", 0, false},
		{"小文字のキーワードも採る", "fix", "closes #7", 7, true},
		{"単語でないキーワードは採らない", "fix", "xRefs #7", 0, false},
		{"段階でない角括弧は採らない", "[WIP] #7", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, ok := LinkedIssue(tt.title, tt.body)
			if n != tt.want || ok != tt.wanto {
				t.Errorf("LinkedIssue(%q, %q) = %d, %v, want %d, %v", tt.title, tt.body, n, ok, tt.want, tt.wanto)
			}
		})
	}
}
