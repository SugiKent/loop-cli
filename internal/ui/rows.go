package ui

import (
	"sort"
	"time"

	"github.com/SugiKent/sugi-loop/internal/model"
)

// row は表の 1 行。主体（Card.Result を出した Issue または PR）から引いた値を持ち、
// View が Subject を呼び直さないようにする。
type row struct {
	card      model.Card
	repo      string
	number    int
	isPR      bool
	title     string
	updatedAt time.Time
	body      string
	labels    []string
	comments  []model.Comment
}

// subjectOf は Card.Result を出した Issue または PR を返す。返るのは片方だけ。
func subjectOf(c model.Card) (*model.Issue, *model.PR) {
	if c.Issue != nil && c.Issue.Result == c.Result {
		return c.Issue, nil
	}
	for i := range c.PRs {
		if c.PRs[i].Result == c.Result {
			return nil, &c.PRs[i]
		}
	}
	// 分類の結果と一致する要素が無い Card は s07 の Fetch では作られないが、落ちないようにする。
	if c.Issue != nil {
		return c.Issue, nil
	}
	if len(c.PRs) > 0 {
		return nil, &c.PRs[0]
	}
	return nil, nil
}

// Subject は行の主体のリポジトリ / 番号 / Issue か PR か / タイトル / 最終更新時刻を返す。
func Subject(c model.Card) (repo string, number int, isPR bool, title string, updatedAt time.Time) {
	issue, pr := subjectOf(c)
	switch {
	case issue != nil:
		return issue.Repo, issue.Number, false, issue.Title, issue.UpdatedAt
	case pr != nil:
		return pr.Repo, pr.Number, true, pr.Title, pr.UpdatedAt
	}
	return "", 0, false, "", time.Time{}
}

func newRow(c model.Card) row {
	repo, number, isPR, title, updatedAt := Subject(c)
	r := row{card: c, repo: repo, number: number, isPR: isPR, title: title, updatedAt: updatedAt}
	issue, pr := subjectOf(c)
	switch {
	case issue != nil:
		r.body, r.labels, r.comments = issue.Body, issue.Labels, issue.Comments
	case pr != nil:
		r.body, r.labels, r.comments = pr.Body, pr.Labels, pr.Comments
	}
	return r
}

// buildRows は Card を Card.Result.Tab で 4 タブに振り分け、タブごとに
// Priority 昇順 → 主体の UpdatedAt 降順 → Repo 昇順 → 番号昇順に並べる。
func buildRows(cards []model.Card) map[model.Tab][]row {
	rows := map[model.Tab][]row{
		model.TabNow: {}, model.TabBacklog: {}, model.TabInProgress: {}, model.TabAbnormal: {},
	}
	for _, c := range cards {
		tab := c.Result.Tab
		if _, ok := rows[tab]; !ok {
			continue
		}
		rows[tab] = append(rows[tab], newRow(c))
	}
	for tab := range rows {
		rs := rows[tab]
		sort.SliceStable(rs, func(i, j int) bool {
			a, b := rs[i], rs[j]
			if a.card.Result.Priority != b.card.Result.Priority {
				return a.card.Result.Priority < b.card.Result.Priority
			}
			if !a.updatedAt.Equal(b.updatedAt) {
				return a.updatedAt.After(b.updatedAt)
			}
			if a.repo != b.repo {
				return a.repo < b.repo
			}
			return a.number < b.number
		})
	}
	return rows
}
