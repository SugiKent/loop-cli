package gh

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// GraphQL クエリは操作宣言付きの完全な文字列で -f query= に渡す。
const (
	reviewThreadsQuery = `query($owner:String!,$name:String!,$number:Int!){ repository(owner:$owner,name:$name){ pullRequest(number:$number){ reviewThreads(first:50){ nodes{ id isResolved comments(first:100){ nodes{ databaseId author{login} body createdAt } } } } } } }`

	crossReferencedPRsQuery = `query($owner:String!,$name:String!,$number:Int!){ repository(owner:$owner,name:$name){ issue(number:$number){ timelineItems(first:50, itemTypes:[CROSS_REFERENCED_EVENT]){ nodes{ ... on CrossReferencedEvent { source { ... on PullRequest { number title body state labels(first:20){nodes{name}} } } } } } } } }`

	labelTimelineJQ = `.[] | select(.event=="labeled" or .event=="unlabeled") | {created_at, event, label: .label.name}`
)

// Client は gh サブプロセスで GHClient を実装する。
type Client struct {
	run func(ctx context.Context, stdin string, args ...string) ([]byte, error)
	// openBrowser は OS のブラウザ起動コマンドの実行。gh とは別の経路なので run とは分けて持つ。
	openBrowser func(ctx context.Context, name, url string) (exitCode int, stderr string, err error)
	// RetryWait は ViewPRMergeState が mergeable UNKNOWN で再取得するまでの待ち時間。
	RetryWait time.Duration
}

// NewClient は os/exec で gh を起動する Client を返す。
func NewClient() *Client {
	return &Client{run: runGH, openBrowser: runBrowser, RetryWait: 2 * time.Second}
}

func runGH(ctx context.Context, stdin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1")
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return stdout.Bytes(), nil
	}
	// ctx が切れた場合、cmd.Run は「signal: killed」の ExitError を返すので ctx を先に見る。
	if ctx.Err() != nil {
		return nil, fmt.Errorf("gh %s: %w", strings.Join(args, " "), ctx.Err())
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil, &Error{Args: args, ExitCode: exitErr.ExitCode(), Stderr: stderr.String()}
	}
	return nil, fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
}

func decodeErr(args []string, err error) error {
	return fmt.Errorf("gh %s: decode: %w", strings.Join(args, " "), err)
}

// splitRepo は owner/name を分解する。gh に渡す前の形式は s02 の設定検証が保証している。
func splitRepo(repo string) (owner, name string) {
	owner, name, _ = strings.Cut(repo, "/")
	return owner, name
}

// Check は gh の存在と認証を確認する。起動時に 1 回だけ呼ぶ。
func (c *Client) Check(ctx context.Context) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh が見つかりません。GitHub CLI をインストールして gh auth login を実行してください: %w", err)
	}
	if _, err := c.run(ctx, "", "auth", "status"); err != nil {
		return err
	}
	return nil
}

// 引数の組み立ては Capture と共有するため関数に分けてある。
func argsSearchIssues(repos []string) []string {
	args := []string{"search", "issues"}
	for _, r := range repos {
		args = append(args, "--repo", r)
	}
	return append(args, "--state", "open", "--limit", "200",
		"--json", "repository,number,title,labels,updatedAt,url,body,commentsCount")
}

func argsSearchPRs(repos []string) []string {
	args := []string{"search", "prs"}
	for _, r := range repos {
		args = append(args, "--repo", r)
	}
	return append(args, "--state", "open", "--limit", "200",
		"--json", "repository,number,title,labels,updatedAt,url,body,isDraft")
}

func argsViewIssue(repo string, number int) []string {
	return []string{"issue", "view", strconv.Itoa(number), "-R", repo,
		"--json", "number,title,body,url,labels,comments"}
}

func argsPRView(repo string, number int, fields string) []string {
	return []string{"pr", "view", strconv.Itoa(number), "-R", repo, "--json", fields}
}

func argsReviewThreads(repo string, number int) []string {
	owner, name := splitRepo(repo)
	return []string{"api", "graphql",
		"-f", "owner=" + owner, "-f", "name=" + name, "-F", "number=" + strconv.Itoa(number),
		"-f", "query=" + reviewThreadsQuery}
}

func argsCrossReferencedPRs(repo string, number int) []string {
	owner, name := splitRepo(repo)
	return []string{"api", "graphql",
		"-f", "owner=" + owner, "-f", "name=" + name, "-F", "number=" + strconv.Itoa(number),
		"-f", "query=" + crossReferencedPRsQuery}
}

func argsLabelTimeline(repo string, number int) []string {
	owner, name := splitRepo(repo)
	path := fmt.Sprintf("repos/%s/%s/issues/%d/timeline", owner, name, number)
	return []string{"api", path, "--paginate", "--jq", labelTimelineJQ}
}

func (c *Client) SearchIssues(ctx context.Context, repos []string) ([]SearchIssue, error) {
	args := argsSearchIssues(repos)

	out, err := c.run(ctx, "", args...)
	if err != nil {
		return nil, err
	}
	issues, err := decodeSearchIssues(out)
	if err != nil {
		return nil, decodeErr(args, err)
	}
	return issues, nil
}

func (c *Client) SearchPRs(ctx context.Context, repos []string) ([]SearchPR, error) {
	args := argsSearchPRs(repos)

	out, err := c.run(ctx, "", args...)
	if err != nil {
		return nil, err
	}
	prs, err := decodeSearchPRs(out)
	if err != nil {
		return nil, decodeErr(args, err)
	}
	return prs, nil
}

func (c *Client) ViewIssue(ctx context.Context, repo string, number int) (*IssueDetail, error) {
	args := argsViewIssue(repo, number)

	out, err := c.run(ctx, "", args...)
	if err != nil {
		return nil, err
	}
	detail, err := decodeIssueDetail(out)
	if err != nil {
		return nil, decodeErr(args, err)
	}
	return detail, nil
}

func (c *Client) ViewPR(ctx context.Context, repo string, number int) (*PRDetail, error) {
	args := argsPRView(repo, number, "number,title,body,url,labels,isDraft,comments")

	out, err := c.run(ctx, "", args...)
	if err != nil {
		return nil, err
	}
	detail, err := decodePRDetail(out)
	if err != nil {
		return nil, decodeErr(args, err)
	}
	return detail, nil
}

const prMergeStateFields = "mergeable,mergeStateStatus,statusCheckRollup,reviewDecision"

// ViewPRMergeState は mergeable が UNKNOWN のとき RetryWait 待って 1 回だけ取り直す。
// GitHub が mergeable を非同期に計算するため、直後の 1 回目は UNKNOWN になりやすい。
func (c *Client) ViewPRMergeState(ctx context.Context, repo string, number int) (*PRMergeState, error) {
	out, err := c.prViewRaw(ctx, repo, number, prMergeStateFields)
	if err != nil {
		return nil, err
	}
	state, err := decodePRMergeState(out)
	if err != nil {
		return nil, decodeErr(argsPRView(repo, number, prMergeStateFields), err)
	}
	return state, nil
}

// prViewRaw は pr view の標準出力を返す。mergeable が UNKNOWN なら RetryWait 後に 1 回だけ
// 取り直し、2 回目も UNKNOWN ならその出力をそのまま返す（エラーにしない）。
// fields には mergeable を含める（Capture の 11 フィールドと ViewPRMergeState の 4 フィールド）。
func (c *Client) prViewRaw(ctx context.Context, repo string, number int, fields string) ([]byte, error) {
	args := argsPRView(repo, number, fields)

	out, err := c.run(ctx, "", args...)
	if err != nil {
		return nil, err
	}
	state, err := decodePRMergeState(out)
	if err != nil {
		return nil, decodeErr(args, err)
	}
	if state.Mergeable != "UNKNOWN" {
		return out, nil
	}
	select {
	case <-time.After(c.RetryWait):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return c.run(ctx, "", args...)
}

func (c *Client) ReviewThreads(ctx context.Context, repo string, number int) ([]ReviewThread, error) {
	args := argsReviewThreads(repo, number)

	out, err := c.run(ctx, "", args...)
	if err != nil {
		return nil, err
	}
	threads, err := decodeReviewThreads(out)
	if err != nil {
		return nil, decodeErr(args, err)
	}
	return threads, nil
}

func (c *Client) CrossReferencedPRs(ctx context.Context, repo string, number int) ([]CrossReferencedPR, error) {
	args := argsCrossReferencedPRs(repo, number)

	out, err := c.run(ctx, "", args...)
	if err != nil {
		return nil, err
	}
	prs, err := decodeCrossReferencedPRs(out)
	if err != nil {
		return nil, decodeErr(args, err)
	}
	return prs, nil
}

func (c *Client) LabelTimeline(ctx context.Context, repo string, number int) ([]LabelEvent, error) {
	args := argsLabelTimeline(repo, number)

	out, err := c.run(ctx, "", args...)
	if err != nil {
		return nil, err
	}
	events, err := decodeLabelEvents(out)
	if err != nil {
		return nil, decodeErr(args, err)
	}
	return events, nil
}

func (c *Client) CommentIssue(ctx context.Context, repo string, number int, body string) error {
	_, err := c.run(ctx, body, "issue", "comment", strconv.Itoa(number), "-R", repo, "--body-file", "-")
	return err
}

func (c *Client) CommentPR(ctx context.Context, repo string, number int, body string) error {
	_, err := c.run(ctx, body, "pr", "comment", strconv.Itoa(number), "-R", repo, "--body-file", "-")
	return err
}

func (c *Client) AddLabel(ctx context.Context, repo string, number int, label string) error {
	_, err := c.run(ctx, "", "issue", "edit", strconv.Itoa(number), "-R", repo, "--add-label", label)
	return err
}

func (c *Client) RemoveLabel(ctx context.Context, repo string, number int, label string) error {
	_, err := c.run(ctx, "", "issue", "edit", strconv.Itoa(number), "-R", repo, "--remove-label", label)
	return err
}

func (c *Client) MergePR(ctx context.Context, repo string, number int, method string) error {
	_, err := c.run(ctx, "", "pr", "merge", strconv.Itoa(number), "-R", repo, "--"+method)
	return err
}

// CreateIssue は作成した issue の URL（標準出力の末尾行）を返す。
func (c *Client) CreateIssue(ctx context.Context, repo string, title string, body string) (string, error) {
	args := []string{"issue", "create", "-R", repo, "--title", title, "--body-file", "-"}
	out, err := c.run(ctx, body, args...)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	url := lines[len(lines)-1]
	if !strings.HasPrefix(url, "https://") {
		return "", decodeErr(args, fmt.Errorf("末尾行が URL ではありません: %q", url))
	}
	return url, nil
}

func (c *Client) ReplyReviewThread(ctx context.Context, repo string, number int, commentID int64, body string) error {
	owner, name := splitRepo(repo)
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return err
	}
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/comments/%d/replies", owner, name, number, commentID)
	_, err = c.run(ctx, string(payload), "api", "-X", "POST", path, "--input", "-")
	return err
}

func (c *Client) Browse(ctx context.Context, repo string, number int) error {
	_, err := c.run(ctx, "", "browse", strconv.Itoa(number), "-R", repo)
	return err
}

// OpenURL は OS のブラウザ起動コマンドで任意の URL を開く。gh は使わない
// （gh browse はリポジトリと番号しか受け取れず、本文に書かれた URL を開けないため）。
func (c *Client) OpenURL(ctx context.Context, url string) error {
	name := browserCommand(runtime.GOOS)
	code, stderr, err := c.openBrowser(ctx, name, url)
	if err != nil {
		return fmt.Errorf("%s %s: %w", name, url, err)
	}
	if code != 0 {
		return fmt.Errorf("%s %s: exit %d: %s", name, url, code, strings.TrimSpace(stderr))
	}
	return nil
}

// browserCommand は OS ごとのブラウザ起動コマンド名。
func browserCommand(goos string) string {
	if goos == "darwin" {
		return "open"
	}
	return "xdg-open"
}

// runBrowser はコマンドを ctx 付きでシェルを経由せずに実行し、終了コードと stderr を返す。
// 標準出力は読み捨てる。起動できなかったときだけ error を返す。
func runBrowser(ctx context.Context, name, url string) (int, string, error) {
	cmd := exec.CommandContext(ctx, name, url)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return 0, stderr.String(), nil
	}
	// ctx が切れた場合、cmd.Run は「signal: killed」の ExitError を返すので ctx を先に見る。
	if ctx.Err() != nil {
		return 0, stderr.String(), ctx.Err()
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), stderr.String(), nil
	}
	return 0, stderr.String(), err
}
