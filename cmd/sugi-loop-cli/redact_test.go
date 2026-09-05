package main

import (
	"strings"
	"testing"
)

func redactOne(t *testing.T, in, owner, name, alias string) string {
	t.Helper()
	out, _, err := redact(map[string][]byte{"issue-1.json": []byte(in)}, owner, name, alias)
	if err != nil {
		t.Fatalf("redact: %v", err)
	}
	return string(out["issue-1.json"])
}

func TestRedactRepoAndURL(t *testing.T) {
	in := `{"repository":{"name":"widgets","nameWithOwner":"acme/widgets"},` +
		`"url":"https://github.com/acme/widgets/issues/5"}`
	want := `{"repository":{"name":"app","nameWithOwner":"org/app"},` +
		`"url":"https://github.com/org/app/issues/5"}`
	if got := redactOne(t, in, "acme", "widgets", "app"); got != want {
		t.Errorf("結果 =\n  %s\nwant\n  %s", got, want)
	}
}

func TestRedactSamePersonSameNumber(t *testing.T) {
	files := map[string][]byte{
		"issue-1.json": []byte(`{"author":{"login":"Alice"},"body":"@bob と @alice で確認"}`),
		"pr-2.json":    []byte(`{"author":{"login":"bob"}}`),
	}
	out, logins, err := redact(files, "acme", "widgets", "app")
	if err != nil {
		t.Fatalf("redact: %v", err)
	}
	wantIssue := `{"author":{"login":"user-2"},"body":"@user-3 と @user-2 で確認"}`
	if got := string(out["issue-1.json"]); got != wantIssue {
		t.Errorf("issue-1.json =\n  %s\nwant\n  %s", got, wantIssue)
	}
	wantPR := `{"author":{"login":"user-3"}}`
	if got := string(out["pr-2.json"]); got != wantPR {
		t.Errorf("pr-2.json =\n  %s\nwant\n  %s", got, wantPR)
	}
	if logins != 3 {
		t.Errorf("login 表の件数 = %d, want 3", logins)
	}
}

func TestRedactOwnerIsAuthor(t *testing.T) {
	in := `{"author":{"login":"acme"},"nameWithOwner":"acme/widgets","body":"@acme さん"}`
	want := `{"author":{"login":"user-1"},"nameWithOwner":"org/app","body":"@user-1 さん"}`
	if got := redactOne(t, in, "acme", "widgets", "app"); got != want {
		t.Errorf("結果 =\n  %s\nwant\n  %s", got, want)
	}
}

func TestRedactEmail(t *testing.T) {
	in := `Co-Authored-By: Alice <alice@acme.example>`
	want := `Co-Authored-By: Alice <user@example.com>`
	out, logins, err := redact(map[string][]byte{"issue-1.json": []byte(in)}, "acme", "widgets", "app")
	if err != nil {
		t.Fatalf("redact: %v", err)
	}
	if got := string(out["issue-1.json"]); got != want {
		t.Errorf("結果 = %q, want %q", got, want)
	}
	// login 表は owner だけ。メールの @example は mention として拾われない。
	if logins != 1 {
		t.Errorf("login 表の件数 = %d, want 1（owner のみ）", logins)
	}
}

func TestRedactKeepsClassificationStrings(t *testing.T) {
	in := `{"labels":[{"name":"stage:propose"},{"name":"question"}],` +
		`"body":"未確定の判断: 2 件\n\n<!-- routine -->\n## Q1. 名前での絞り込みを含めるか\n` +
		`- 選択肢 A（推奨）\nblocked-by: human\nRefs #108\n&lt;!-- routine --&gt;\n## PR リスク評価"}`
	if got := redactOne(t, in, "acme", "widgets", "app"); got != in {
		t.Errorf("分類に使う文字列が変わった:\n  %s\nwant\n  %s", got, in)
	}
}

func TestRedactWordBoundary(t *testing.T) {
	in := `also @al-team al`
	want := `also @user-2 user-1`
	if got := redactOne(t, in, "al", "widgets", "app"); got != want {
		t.Errorf("結果 = %q, want %q", got, want)
	}
}

func TestRedactCollisionErrors(t *testing.T) {
	cases := []struct {
		label       string
		in          string
		owner, name string
		want        string
	}{
		{
			label: "login が JSON キーと衝突",
			in:    `{"author":{"login":"status"},"status":"COMPLETED"}`,
			owner: "acme", name: "widgets", want: "status",
		},
		{
			label: "login が列挙値と衝突",
			in:    `{"author":{"login":"completed"},"status":"COMPLETED"}`,
			owner: "acme", name: "widgets", want: "completed",
		},
		{
			label: "リポジトリ名がラベル名と衝突",
			in:    `{"labels":[{"name":"docs"}]}`,
			owner: "acme", name: "docs", want: "docs",
		},
	}
	for _, tc := range cases {
		files := map[string][]byte{"issue-1.json": []byte(tc.in)}
		out, _, err := redact(files, tc.owner, tc.name, "app")
		if err == nil {
			t.Errorf("%s: エラーが返らなかった", tc.label)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: エラー = %v, want %q を含む", tc.label, err, tc.want)
		}
		if out != nil {
			t.Errorf("%s: 衝突時は結果を返さない", tc.label)
		}
	}
}
