// Package model は internal/gh の生の JSON と画面の間で使う共通の型を定義する。
// 分類（internal/classify）が使う値だけを持ち、画面の都合は持ち込まない。
package model

import (
	"time"

	"github.com/SugiKent/loop-cli/internal/gh"
)

// ラベル名はプラグイン規約に固定で、設定では変えられない（mvp.md）。
const (
	LabelStageTodo    = "stage:todo"
	LabelStagePropose = "stage:propose"
	LabelStageApply   = "stage:apply"
	LabelStageArchive = "stage:archive"
	LabelPropose      = "propose"
	LabelApply        = "apply"
	LabelArchive      = "archive"
	LabelDocs         = "docs"
	LabelQuestion     = "question"
	LabelAIAssess     = "ai-assess:requested"
	LabelBlocked      = "blocked"
	LabelWip          = "wip"
)

// Situation は human-turn-signals.md の判定表の行。ゼロ値 "" は未分類。
type Situation string

const (
	SituationA          Situation = "A"
	SituationB          Situation = "B"
	SituationC          Situation = "C"
	SituationD          Situation = "D"
	SituationE          Situation = "E"
	SituationF          Situation = "F"
	SituationG          Situation = "G"
	SituationOther      Situation = "other"
	SituationInProgress Situation = "in-progress"
)

// Tab は画面のタブ。
type Tab string

const (
	TabNow        Tab = "今やる"
	TabBacklog    Tab = "バックログ"
	TabInProgress Tab = "進行中"
	TabAbnormal   Tab = "異常"
)

// Priority は並び順（小さいほど上）。未分類は最下位の 8。
func (s Situation) Priority() int {
	switch s {
	case SituationF:
		return 0
	case SituationA, SituationD:
		return 1
	case SituationB:
		return 2
	case SituationC:
		return 3
	case SituationE:
		return 4
	case SituationG:
		return 5
	case SituationOther:
		return 6
	case SituationInProgress:
		return 7
	default:
		return 8
	}
}

// Tab はその局面を出すタブ。未分類は空文字列。
func (s Situation) Tab() Tab {
	switch s {
	case SituationA, SituationB, SituationC, SituationD, SituationG, SituationOther:
		return TabNow
	case SituationE:
		return TabBacklog
	case SituationF:
		return TabAbnormal
	case SituationInProgress:
		return TabInProgress
	default:
		return ""
	}
}

// Kind は画面に出す種別。未分類は空文字列。
func (s Situation) Kind() string {
	switch s {
	case SituationA, SituationD:
		return "質問"
	case SituationB:
		return "方針"
	case SituationC, SituationG:
		return "merge"
	case SituationE:
		return "todo 候補"
	case SituationF:
		return "異常"
	case SituationOther:
		return "その他"
	case SituationInProgress:
		return "進行中"
	default:
		return ""
	}
}

// Result は分類結果。Summary は「いま人が何をすべきか」の 1 行。
type Result struct {
	Situation Situation
	Priority  int
	Tab       Tab
	Summary   string
}

// Comment は issue / PR のコメント。AI は本文で判定した「routine の発言か」。
type Comment struct {
	Author    string
	Body      string
	CreatedAt time.Time
	URL       string
	AI        bool
}

// Issue は分類対象の issue。Comments は作成順で末尾が最新、nil は詳細の取得失敗。
type Issue struct {
	Repo      string
	Number    int
	Title     string
	URL       string
	Body      string
	Labels    []string
	UpdatedAt time.Time
	Comments  []Comment
	Result    Result
}

// PR は分類対象の PR。State は OPEN / MERGED / CLOSED。
// MergeState / ReviewThreads の nil は詳細の取得失敗。Canonical は同段階の merge 済み PR の正本の印。
type PR struct {
	Repo          string
	Number        int
	Title         string
	URL           string
	Body          string
	Labels        []string
	IsDraft       bool
	State         string
	UpdatedAt     time.Time
	Comments      []Comment
	MergeState    *gh.PRMergeState
	ReviewThreads []gh.ReviewThread
	Canonical     bool
	Result        Result
}

// Card は issue 1 件と紐づく PR 群。Issue が nil のカードは PR 単独。
type Card struct {
	Issue  *Issue
	PRs    []PR
	Result Result
}

// HasLabel は labels に name があるかを返す。
func HasLabel(labels []string, name string) bool {
	for _, l := range labels {
		if l == name {
			return true
		}
	}
	return false
}

var (
	issueStageOrder = []string{LabelStageTodo, LabelStagePropose, LabelStageApply, LabelStageArchive}
	prStageOrder    = []string{LabelPropose, LabelApply, LabelArchive}
)

func stages(labels []string, order []string) []string {
	var out []string
	for _, s := range order {
		if HasLabel(labels, s) {
			out = append(out, s)
		}
	}
	return out
}

// IssueStages は issue の段階ラベルを段階順で返す。
func IssueStages(labels []string) []string { return stages(labels, issueStageOrder) }

// PRStages は PR の段階ラベルを段階順で返す。
func PRStages(labels []string) []string { return stages(labels, prStageOrder) }

func labelNames(labels []gh.Label) []string {
	var out []string
	for _, l := range labels {
		out = append(out, l.Name)
	}
	return out
}

// IssueFromSearch は gh search issues の 1 件を Issue に写す。詳細（Comments）は入れない。
func IssueFromSearch(si gh.SearchIssue) Issue {
	return Issue{
		Repo:      si.Repository.NameWithOwner,
		Number:    si.Number,
		Title:     si.Title,
		URL:       si.URL,
		Body:      si.Body,
		Labels:    labelNames(si.Labels),
		UpdatedAt: si.UpdatedAt,
	}
}

// PRFromSearch は gh search prs の 1 件を PR に写す。search は open だけを返すので State は OPEN。
func PRFromSearch(sp gh.SearchPR) PR {
	return PR{
		Repo:      sp.Repository.NameWithOwner,
		Number:    sp.Number,
		Title:     sp.Title,
		URL:       sp.URL,
		Body:      sp.Body,
		Labels:    labelNames(sp.Labels),
		IsDraft:   sp.IsDraft,
		State:     "OPEN",
		UpdatedAt: sp.UpdatedAt,
	}
}

// CommentFrom は gh のコメントを Comment に写し、本文から AI を判定する。
func CommentFrom(c gh.Comment) Comment {
	return Comment{
		Author:    c.Author.Login,
		Body:      c.Body,
		CreatedAt: c.CreatedAt,
		URL:       c.URL,
		AI:        IsAI(c.Body),
	}
}
