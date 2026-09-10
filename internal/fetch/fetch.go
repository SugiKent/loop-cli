package fetch

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/SugiKent/loop-cli/internal/classify"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// CallTimeout は gh 呼び出し 1 回あたりの期限（s03 design.md が s07 に委ねた値）。
const CallTimeout = 30 * time.Second

// detailConcurrency は詳細取得の並行度。secondary rate limit に当たらない範囲。
const detailConcurrency = 4

// Result は Fetch の返り値。Errors は詳細取得 1 件ごとの部分失敗。
type Result struct {
	Cards  []model.Card
	Errors []error
}

// detailError は Errors を決定的に並べるために発生元を保持する。
// 1 つの PR が 3 つの詳細で失敗し得るので、メソッドも並び順のキーに使う。
type detailError struct {
	isPR   bool
	repo   string
	number int
	method string
	err    error
}

// Fetch は repos の open issue / open PR を取得し、分類済みの Card 群を返す。
// modes はリポジトリ名から運用方式を引く表で、表に無いリポジトリは model.Mode のゼロ値（sdd）。
// search の失敗と ctx の中断は (nil, error)、詳細取得 1 件の失敗は Result.Errors に積む。
func Fetch(ctx context.Context, client gh.GHClient, repos []string, modes map[string]model.Mode) (*Result, error) {
	searchIssues, err := call(ctx, func(ctx context.Context) ([]gh.SearchIssue, error) {
		return client.SearchIssues(ctx, repos)
	})
	if err != nil {
		return nil, fmt.Errorf("search issues: %w", err)
	}
	searchPRs, err := call(ctx, func(ctx context.Context) ([]gh.SearchPR, error) {
		return client.SearchPRs(ctx, repos)
	})
	if err != nil {
		return nil, fmt.Errorf("search prs: %w", err)
	}

	issues := make([]model.Issue, len(searchIssues))
	for i, si := range searchIssues {
		issues[i] = model.IssueFromSearch(si)
	}
	prs := make([]model.PR, len(searchPRs))
	for i, sp := range searchPRs {
		prs[i] = model.PRFromSearch(sp)
	}

	errs := fetchDetails(ctx, client, issues, prs)
	if ctx.Err() != nil {
		return nil, fmt.Errorf("fetch: %w", ctx.Err())
	}

	return &Result{Cards: buildCards(issues, prs, modes), Errors: sortErrors(errs)}, nil
}

// call は 1 回の gh 呼び出しに CallTimeout の子 ctx を付ける。
func call[T any](ctx context.Context, f func(context.Context) (T, error)) (T, error) {
	c, cancel := context.WithTimeout(ctx, CallTimeout)
	defer cancel()
	return f(c)
}

// fetchDetails は全 issue / 全 PR の詳細を並行取得し、部分失敗を返す。
func fetchDetails(ctx context.Context, client gh.GHClient, issues []model.Issue, prs []model.PR) []detailError {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []detailError
		sem  = make(chan struct{}, detailConcurrency)
	)
	fail := func(isPR bool, method, repo string, number int, err error) {
		mu.Lock()
		defer mu.Unlock()
		errs = append(errs, detailError{isPR: isPR, repo: repo, number: number, method: method,
			err: fmt.Errorf("%s %s#%d: %w", method, repo, number, err)})
	}
	run := func(f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			f()
		}()
	}

	for i := range issues {
		// 1. 全 open issue のコメント。
		run(func() {
			detail, err := call(ctx, func(ctx context.Context) (*gh.IssueDetail, error) {
				return client.ViewIssue(ctx, issues[i].Repo, issues[i].Number)
			})
			if err != nil {
				fail(false, "ViewIssue", issues[i].Repo, issues[i].Number, err)
				return
			}
			issues[i].Comments = comments(detail.Comments)
		})
	}

	for i := range prs {
		pr := &prs[i]
		// 2. 全 open PR のコメント。
		run(func() {
			detail, err := call(ctx, func(ctx context.Context) (*gh.PRDetail, error) {
				return client.ViewPR(ctx, pr.Repo, pr.Number)
			})
			if err != nil {
				fail(true, "ViewPR", pr.Repo, pr.Number, err)
				return
			}
			pr.Comments = comments(detail.Comments)
		})
		// 3. 全 open PR の merge 状態。
		run(func() {
			ms, err := call(ctx, func(ctx context.Context) (*gh.PRMergeState, error) {
				return client.ViewPRMergeState(ctx, pr.Repo, pr.Number)
			})
			if err != nil {
				fail(true, "ViewPRMergeState", pr.Repo, pr.Number, err)
				return
			}
			pr.MergeState = ms
		})
		// 4. 全 open PR の review thread。
		run(func() {
			th, err := call(ctx, func(ctx context.Context) ([]gh.ReviewThread, error) {
				return client.ReviewThreads(ctx, pr.Repo, pr.Number)
			})
			if err != nil {
				fail(true, "ReviewThreads", pr.Repo, pr.Number, err)
				return
			}
			pr.ReviewThreads = th
		})
	}

	wg.Wait()
	return errs
}

func comments(cs []gh.Comment) []model.Comment {
	out := make([]model.Comment, len(cs))
	for i, c := range cs {
		out[i] = model.CommentFrom(c)
	}
	return out
}

// buildCards は PR を紐づけ先の issue に集め、紐づかない PR を単独 Card にする。
func buildCards(issues []model.Issue, prs []model.PR, modes map[string]model.Mode) []model.Card {
	cards := make([]model.Card, len(issues))
	index := map[string]map[int]int{}
	for i := range issues {
		cards[i].Issue = &issues[i]
		if index[issues[i].Repo] == nil {
			index[issues[i].Repo] = map[int]int{}
		}
		index[issues[i].Repo][issues[i].Number] = i
	}

	var lone []model.Card
	for _, pr := range prs {
		if n, ok := LinkedIssue(pr.Title, pr.Body); ok {
			if i, found := index[pr.Repo][n]; found {
				cards[i].PRs = append(cards[i].PRs, pr)
				continue
			}
		}
		lone = append(lone, model.Card{PRs: []model.PR{pr}})
	}
	cards = append(cards, lone...)

	for i := range cards {
		mode := modes[cardRepo(cards[i])]
		sortPRs(cards[i].PRs, mode)
		cards[i] = classify.Card(cards[i], mode)
	}
	return cards
}

// cardRepo は Card のリポジトリを返す。1 枚の Card は 1 リポジトリ分しか持たない。
func cardRepo(c model.Card) string {
	if c.Issue != nil {
		return c.Issue.Repo
	}
	if len(c.PRs) > 0 {
		return c.PRs[0].Repo
	}
	return ""
}

// sortPRs は段階順（propose → apply → archive → 段階無し）→ 番号昇順に並べる。
func sortPRs(prs []model.PR, mode model.Mode) {
	sort.SliceStable(prs, func(i, j int) bool {
		si, sj := stageRank(prs[i], mode), stageRank(prs[j], mode)
		if si != sj {
			return si < sj
		}
		return prs[i].Number < prs[j].Number
	})
}

func stageRank(pr model.PR, mode model.Mode) int {
	stages := model.PRStages(mode, pr.Labels)
	if len(stages) == 0 {
		return 3
	}
	switch stages[0] {
	case model.LabelPropose:
		return 0
	case model.LabelApply:
		return 1
	default:
		return 2
	}
}

// methodRank は同じ PR の複数の失敗を並行実行の完了順に依らず並べるための順位。
func methodRank(method string) int {
	switch method {
	case "ViewPR":
		return 0
	case "ViewPRMergeState":
		return 1
	default: // ReviewThreads。issue 側は ViewIssue の 1 種類だけなので順位を使わない。
		return 2
	}
}

// sortErrors は issue のエラーを (repo, number) 順、
// 続けて PR のエラーを (repo, number, メソッド) 順に並べる。
func sortErrors(errs []detailError) []error {
	sort.SliceStable(errs, func(i, j int) bool {
		if errs[i].isPR != errs[j].isPR {
			return !errs[i].isPR
		}
		if errs[i].repo != errs[j].repo {
			return errs[i].repo < errs[j].repo
		}
		if errs[i].number != errs[j].number {
			return errs[i].number < errs[j].number
		}
		return methodRank(errs[i].method) < methodRank(errs[j].method)
	})
	var out []error
	for _, e := range errs {
		out = append(out, e.err)
	}
	return out
}
