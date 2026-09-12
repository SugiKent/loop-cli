package ui

import (
	"sort"
	"strings"
	"time"

	"github.com/SugiKent/loop-cli/internal/model"
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
	// stage は主体の段階ラベル（接頭辞を落とす前の名前）。ラベル色を引く鍵になる。段階が無ければ空。
	stage string
	// stageWord は進行中タブの種別の列に出す語。段階が無ければ `-`。
	stageWord string
}

// stageRank は進行中タブの並びの第 2 キー（段階順）。sdd の issue / PR と label の issue が
// 同じ段階で同じ値になるように並べ、段階を持たない行は stageRankNone で最後に置く。
var stageRank = map[string]int{
	model.LabelStageTodo:    0,
	model.LabelToDo:         0,
	model.LabelStagePropose: 1,
	model.LabelPropose:      1,
	model.LabelInProgress:   1,
	model.LabelStageApply:   2,
	model.LabelApply:        2,
	model.LabelStageArchive: 3,
	model.LabelArchive:      3,
	model.LabelDone:         3,
}

const stageRankNone = 4

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

func newRow(c model.Card, modes map[string]model.Mode) row {
	repo, number, isPR, title, updatedAt := Subject(c)
	r := row{card: c, repo: repo, number: number, isPR: isPR, title: title, updatedAt: updatedAt}
	issue, pr := subjectOf(c)
	switch {
	case issue != nil:
		r.body, r.labels, r.comments = issue.Body, issue.Labels, issue.Comments
	case pr != nil:
		r.body, r.labels, r.comments = pr.Body, pr.Labels, pr.Comments
	}
	// 段階が 2 つ以上ある主体（`stage:propose` + `stage:apply` 等）も進行中タブに入るので、
	// 段階順で先頭の 1 つを採る（design.md D4）。方式が分からないリポジトリはゼロ値の sdd。
	stages := model.IssueStages(modes[repo], r.labels)
	if isPR {
		stages = model.PRStages(modes[repo], r.labels)
	}
	r.stageWord = "-"
	if len(stages) > 0 {
		// 列に出すときだけ `stage:` を落とす。色は接頭辞を落とす前の名前で引く（design.md D5）。
		r.stage = stages[0]
		r.stageWord = strings.TrimPrefix(stages[0], "stage:")
	}
	return r
}

// buildRows は Card を Card.Result.Tab で 4 タブに振り分け、タブごとに並べる。
// 今やる / バックログ / 異常は Priority 昇順 → 主体の UpdatedAt 降順 → Repo 昇順 → 番号昇順。
// 進行中は Repo 昇順 → 段階順 → UpdatedAt 降順 → 番号昇順にする（design.md D2。
// このタブは Priority が全行同じで、第 1 キーが並びを決めないため）。
// modes はリポジトリ名 -> 運用方式で、段階ラベルの語彙を決める。nil ならすべてゼロ値の sdd。
func buildRows(cards []model.Card, modes map[string]model.Mode) map[model.Tab][]row {
	rows := map[model.Tab][]row{
		model.TabNow: {}, model.TabBacklog: {}, model.TabInProgress: {}, model.TabAbnormal: {},
	}
	for _, c := range cards {
		tab := c.Result.Tab
		if _, ok := rows[tab]; !ok {
			continue
		}
		rows[tab] = append(rows[tab], newRow(c, modes))
	}
	for tab := range rows {
		rs := rows[tab]
		inProgress := tab == model.TabInProgress
		sort.SliceStable(rs, func(i, j int) bool {
			a, b := rs[i], rs[j]
			if inProgress {
				if a.repo != b.repo {
					return a.repo < b.repo
				}
				if ra, rb := a.stageOrder(), b.stageOrder(); ra != rb {
					return ra < rb
				}
			} else if a.card.Result.Priority != b.card.Result.Priority {
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

// stageOrder は段階順の位置。段階を持たない行は同じ段階の行より後に置く。
func (r row) stageOrder() int {
	if n, ok := stageRank[r.stage]; ok {
		return n
	}
	return stageRankNone
}
