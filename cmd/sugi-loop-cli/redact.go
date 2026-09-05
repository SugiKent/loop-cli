package main

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	emailRe = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	// loginRe は "login":"<x>" の値。gh の --json は 1 行だが空白は許しておく。
	loginRe = regexp.MustCompile(`"login"\s*:\s*"([^"]*)"`)
	// mentionRe は本文中の @mention。直前が [A-Za-z0-9_.-] のものはメールなどなので拾わない。
	mentionRe = regexp.MustCompile(`(^|[^A-Za-z0-9_.-])@([A-Za-z0-9][A-Za-z0-9-]*)`)
	// jsonKeyRe / jsonPairRe は衝突ガード用。JSON キーと、キーと文字列値の組を拾う。
	jsonKeyRe   = regexp.MustCompile(`"([^"]+)"\s*:`)
	jsonPairRe  = regexp.MustCompile(`"([^"]+)"\s*:\s*"([^"]*)"`)
	redactedRe  = regexp.MustCompile(`^user-[0-9]+$`)
	emailRedact = "user@example.com"
)

// guardWords は分類に使うラベル名・マーカー語。login やリポジトリ名がこれらと一致したら
// 全文置換がラベルを壊すので採取を止める。
var guardWords = []string{
	"todo", "propose", "apply", "archive", "question", "blocked",
	"wip", "docs", "routine", "human", "stage", "pr",
}

// redact は採取した全ファイルの個人・組織情報を伏せ字にする。
// 第 2 返り値は login 表の件数（owner を含む）。
func redact(files map[string][]byte, owner, name, alias string) (map[string][]byte, int, error) {
	names := slices.Sorted(maps.Keys(files))

	// 1. メールアドレス
	out := make(map[string]string, len(files))
	for _, n := range names {
		out[n] = emailRe.ReplaceAllString(string(files[n]), emailRedact)
	}

	// 2. login 表。owner が 1、次に "login" 値、最後に @mention。
	table := map[string]int{strings.ToLower(owner): 1}
	var wordLogins []string // owner と "login" 値由来の login（単語として全文置換する対象）
	wordLogins = append(wordLogins, owner)
	add := func(login string) {
		key := strings.ToLower(login)
		if _, ok := table[key]; !ok {
			table[key] = len(table) + 1
		}
	}
	for _, n := range names {
		for _, m := range loginRe.FindAllStringSubmatch(out[n], -1) {
			if m[1] == "" {
				continue
			}
			if _, ok := table[strings.ToLower(m[1])]; !ok {
				wordLogins = append(wordLogins, m[1])
			}
			add(m[1])
		}
	}
	for _, n := range names {
		for _, m := range mentionRe.FindAllStringSubmatch(out[n], -1) {
			if redactedRe.MatchString(m[2]) {
				continue
			}
			add(m[2])
		}
	}

	// 3. 衝突ガード
	if err := guardCollision(out, names, wordLogins, name); err != nil {
		return nil, 0, err
	}

	// 4. 置換
	for _, n := range names {
		s := out[n]
		s = strings.ReplaceAll(s, owner+"/"+name, "org/"+alias)
		for _, login := range wordLogins {
			s = replaceWord(s, login, "user-"+strconv.Itoa(table[strings.ToLower(login)]))
		}
		s = replaceWord(s, name, alias)
		s = mentionRe.ReplaceAllStringFunc(s, func(m string) string {
			sub := mentionRe.FindStringSubmatch(m)
			if redactedRe.MatchString(sub[2]) {
				return m
			}
			num, ok := table[strings.ToLower(sub[2])]
			if !ok {
				return m
			}
			return sub[1] + "@user-" + strconv.Itoa(num)
		})
		out[n] = s
	}

	result := make(map[string][]byte, len(out))
	for n, s := range out {
		result[n] = []byte(s)
	}
	return result, len(table), nil
}

// guardCollision は owner / name / "login" 値由来の login が JSON キー・列挙値・
// ラベル名と衝突していないかを大文字小文字を区別せずに調べる。
func guardCollision(out map[string]string, names, wordLogins []string, name string) error {
	keys := map[string]bool{}
	values := map[string]bool{} // login / name 以外のキーの文字列値
	for _, n := range names {
		for _, m := range jsonKeyRe.FindAllStringSubmatch(out[n], -1) {
			keys[strings.ToLower(m[1])] = true
		}
		for _, m := range jsonPairRe.FindAllStringSubmatch(out[n], -1) {
			k := strings.ToLower(m[1])
			if k == "login" || k == "name" {
				continue
			}
			values[strings.ToLower(m[2])] = true
		}
	}

	targets := append([]string{name}, wordLogins...)
	for _, t := range targets {
		lower := strings.ToLower(t)
		switch {
		case keys[lower]:
			return fmt.Errorf("伏せ字を中止しました: %q は JSON のキーと衝突します（置換すると出力が壊れます）", t)
		case values[lower]:
			return fmt.Errorf("伏せ字を中止しました: %q は JSON の値と衝突します（置換すると出力が壊れます）", t)
		case slices.Contains(guardWords, lower):
			return fmt.Errorf("伏せ字を中止しました: %q は分類に使うラベル・マーカー語と衝突します", t)
		}
	}
	return nil
}

// replaceWord は word を単語として（前後が [A-Za-z0-9-] でない出現だけ）置き換える。
// \b は - を境界にするため使わない（owner "al" が "al-team" に当たる）。
func replaceWord(s, word, repl string) string {
	re := regexp.MustCompile(`(^|[^A-Za-z0-9-])(?i:` + regexp.QuoteMeta(word) + `)([^A-Za-z0-9-]|$)`)
	return re.ReplaceAllString(s, "${1}"+repl+"${2}")
}
