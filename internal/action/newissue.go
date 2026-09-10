package action

import (
	"context"
	"errors"
	"strings"

	"github.com/SugiKent/loop-cli/internal/gh"
)

// ErrEmptyTitle はタイトルが空の下書き。タイトルの無い issue はキューの表で見分けられない。
var ErrEmptyTitle = errors.New("タイトルが空です")

// titlePlaceholder / bodyPlaceholder は下書きを分ける区切りに使う案内の行。
// 文言と分割を同じパッケージに置き、片方だけを直して分割できなくなる事故を防ぐ。
const (
	titlePlaceholder = "タイトル（この下の行に入力してください）"
	bodyPlaceholder  = "概要（この下に入力してください）"
)

// NewIssueDraft は `n` で開くエディタの初期テキスト。案内の 2 行が区切りそのものなので、
// 案内はタイトルにも本文にも入らない。行の下の空行は、人がそこに書き始めるためにある。
const NewIssueDraft = titlePlaceholder + "\n\n" + bodyPlaceholder + "\n\n"

// SplitNewIssue は $EDITOR で書いた 1 枚の下書きを、2 本のプレースホルダー行を区切りに
// タイトルと本文へ分ける。見出し記号や引用記号は解釈せず字面どおりに分ける。
// 区切りが揃わない下書きと、区切りに使った 2 本以外に案内の文言と一致する行が残る下書きは
// 分割せず ok を偽にする（案内の文字列が issue になって GitHub へ出ていく方が悪い）。
func SplitNewIssue(text string) (title, body string, ok bool) {
	lines := strings.Split(text, "\n")
	// CRLF の下書きを LF と同じに扱う。行末の空白は本文では残す（Markdown の改行が消えるため）。
	for i, l := range lines {
		lines[i] = strings.TrimSuffix(l, "\r")
	}

	titleAt := placeholderAt(lines, 0, titlePlaceholder)
	if titleAt < 0 {
		return "", "", false
	}
	bodyAt := placeholderAt(lines, titleAt+1, bodyPlaceholder)
	if bodyAt < 0 {
		return "", "", false
	}
	for i, l := range lines {
		if i == titleAt || i == bodyAt {
			continue
		}
		if t := strings.TrimSpace(l); t == titlePlaceholder || t == bodyPlaceholder {
			return "", "", false
		}
	}

	// GitHub の issue のタイトルは 1 行なので、挟まれた行は半角空白 1 つで連結する。
	var titleLines []string
	for _, l := range lines[titleAt+1 : bodyAt] {
		if t := strings.TrimSpace(l); t != "" {
			titleLines = append(titleLines, t)
		}
	}
	title = strings.Join(titleLines, " ")
	body = strings.TrimSpace(strings.Join(lines[bodyAt+1:], "\n"))
	return title, body, true
}

// placeholderAt は from 以降で、前後の空白を落とした結果が want と一致する最初の行を返す。
// 無ければ -1。行末の空白や \r があっても区切りとして働かせるために空白を落としてから比べる。
func placeholderAt(lines []string, from int, want string) int {
	for i := from; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == want {
			return i
		}
	}
	return -1
}

// CreateIssue は検査を通った下書きで issue を 1 つ作り、作成された issue の URL を返す。
// ラベルも assignee も指定しない（不変条件 6「Issue 作成は承認ではない」・不変条件 2）。
// 作った issue に着手させるかは、人が続けて `t` を押して決める。
func CreateIssue(ctx context.Context, client gh.GHClient, repo string, title string, body string) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", ErrEmptyTitle
	}
	if strings.TrimSpace(body) == "" {
		return "", ErrEmptyBody
	}
	// 不変条件 7「TUI は <!-- routine --> を書かない」は本文とタイトルの両方に及ぶ。
	if HasRoutineMarker(title) || HasRoutineMarker(body) {
		return "", ErrRoutineMarker
	}
	return client.CreateIssue(ctx, repo, title, body)
}
