package gh

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 伏せ字の規則と同じ形。redact（cmd/loop-cli-dev）が置き換えた結果を検査する。
var (
	fixtureLoginRe = regexp.MustCompile(`"login"\s*:\s*"([^"]*)"`)
	fixtureUserRe  = regexp.MustCompile(`^user-[0-9]+$`)
	fixtureEmailRe = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
)

// fixtureFiles は testdata/fixtures 直下の <alias> ディレクトリごとのファイル一覧を返す。
func fixtureFiles(t *testing.T) map[string][]string {
	t.Helper()
	root := filepath.Join("testdata", "fixtures")
	dirs, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("fixtures を読めません: %v", err)
	}
	out := map[string][]string{}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(root, d.Name()))
		if err != nil {
			t.Fatalf("%s を読めません: %v", d.Name(), err)
		}
		for _, e := range entries {
			out[d.Name()] = append(out[d.Name()], filepath.Join(root, d.Name(), e.Name()))
		}
	}
	if len(out) == 0 {
		t.Fatal("fixtures ディレクトリが空です")
	}
	return out
}

func TestFixturesContainNoPersonalData(t *testing.T) {
	byAlias := fixtureFiles(t)

	t.Run("login は user-N 形式", func(t *testing.T) {
		for _, paths := range byAlias {
			for _, p := range paths {
				b, err := os.ReadFile(p)
				if err != nil {
					t.Fatalf("%s: %v", p, err)
				}
				for _, m := range fixtureLoginRe.FindAllStringSubmatch(string(b), -1) {
					if !fixtureUserRe.MatchString(m[1]) {
						t.Errorf("%s: 伏せ字されていない login %q", p, m[1])
					}
				}
			}
		}
	})

	t.Run("メールアドレスは user@example.com だけ", func(t *testing.T) {
		for _, paths := range byAlias {
			for _, p := range paths {
				b, err := os.ReadFile(p)
				if err != nil {
					t.Fatalf("%s: %v", p, err)
				}
				for _, m := range fixtureEmailRe.FindAllString(string(b), -1) {
					if m != "user@example.com" {
						t.Errorf("%s: 伏せ字されていないメールアドレス %q", p, m)
					}
				}
			}
		}
	})

	t.Run("元の owner/name が残っていない", func(t *testing.T) {
		origin := os.Getenv("SUGI_LOOP_FIXTURE_ORIGIN")
		if origin == "" {
			t.Skip("SUGI_LOOP_FIXTURE_ORIGIN=owner/name を設定すると元 owner の検査を行います")
		}
		owner, name, ok := strings.Cut(origin, "/")
		if !ok || owner == "" || name == "" {
			t.Fatalf("SUGI_LOOP_FIXTURE_ORIGIN は owner/name 形式で指定してください: %q", origin)
		}
		for alias, paths := range byAlias {
			// example は org/app 固定の手書きなので対象外。
			if alias == "example" {
				continue
			}
			for _, p := range paths {
				b, err := os.ReadFile(p)
				if err != nil {
					t.Fatalf("%s: %v", p, err)
				}
				lower := strings.ToLower(string(b))
				for _, want := range []string{owner, name} {
					if strings.Contains(lower, strings.ToLower(want)) {
						t.Errorf("%s: 元の %q が残っています", p, strings.ToLower(want))
					}
				}
			}
		}
	})
}
