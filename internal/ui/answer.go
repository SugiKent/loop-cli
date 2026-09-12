package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/loop-cli/internal/action"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// postTimeout は 1 回の投稿の上限。s07 の CallTimeout と同じ値で、gh が固まっても投稿中のまま止まらない。
const postTimeout = 30 * time.Second

// answerReason は確認画面に入った理由。
type answerReason int

const (
	reasonNone answerReason = iota
	reasonBlockedBy
	reasonMarker
)

// answerState は `a` を押してから投稿が終わるまでの状態。
// 編集完了のメッセージは対象を持たないので、対象は Model 側で保持する。
type answerState struct {
	target action.Target
	label  string
	draft  string
	reason answerReason
	from   screen // `a` を押した画面。確認画面から戻る先
}

// postedMsg は 1 回の投稿の完了。
type postedMsg struct {
	label string
	err   error
}

// answerKey は `a`（回答）を扱う。対象は画面が見せているものに決まる。
func (m Model) answerKey() (tea.Model, tea.Cmd) {
	if m.writing {
		return m, nil
	}
	target, label, comments, ok := m.answerTarget()
	if !ok {
		return m, nil
	}
	m.answer = answerState{target: target, label: label, from: m.screen}
	m.writeStatus, m.writeStatusErr = "", false
	cmd := m.openEditor(routeAnswer, action.AnswerTemplate(comments))
	return m, cmd
}

// answerTarget は今の画面の回答対象・表示名・テンプレートの元にするコメントを返す。
func (m Model) answerTarget() (action.Target, string, []model.Comment, bool) {
	switch m.screen {
	case screenCard:
		issue := m.detail.card.Issue
		return targetOfIssue(issue), issueLabel(issue), issue.Comments, true
	case screenPR:
		pr := m.currentPR()
		return targetOfPR(pr), prLabel(pr), pr.Comments, true
	}
	rows := m.rows[m.tab]
	if m.cursor >= len(rows) {
		return action.Target{}, "", nil, false
	}
	r := rows[m.cursor]
	label := fmt.Sprintf("%s #%d", r.repo, r.number)
	if r.isPR {
		label = fmt.Sprintf("%s PR#%d", r.repo, r.number)
	}
	return action.Target{Repo: r.repo, Number: r.number, IsPR: r.isPR}, label, r.comments, true
}

func targetOfIssue(i *model.Issue) action.Target {
	return action.Target{Repo: i.Repo, Number: i.Number}
}

func targetOfPR(p model.PR) action.Target {
	return action.Target{Repo: p.Repo, Number: p.Number, IsPR: true}
}

func issueLabel(i *model.Issue) string { return fmt.Sprintf("%s #%d", i.Repo, i.Number) }

func prLabel(p model.PR) string { return fmt.Sprintf("%s PR#%d", p.Repo, p.Number) }

// updateAnswerEdited は `a` で始めた編集の完了を扱う。投稿できる本文だけが投稿コマンドに進み、
// 不変条件 7 / 8 に触れる本文は確認画面へ回す。
func (m Model) updateAnswerEdited(msg editedMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.err != nil:
		m.writeStatus, m.writeStatusErr = "エディタ: "+msg.err.Error(), true
	case action.IsBlankAnswer(msg.text):
		// 引用だけの下書きをそのまま投稿すると、dispatcher が「人が答えた」とみなす。
		if strings.TrimSpace(msg.text) == "" {
			m.writeStatus, m.writeStatusErr = "回答を中止しました（本文が空）", false
		} else {
			m.writeStatus, m.writeStatusErr = "回答を中止しました（引用だけです）", false
		}
	case action.HasRoutineMarker(msg.text):
		m.answer.draft, m.answer.reason = msg.text, reasonMarker
		m.screen = screenConfirm
	case len(action.BlockedByLines(msg.text)) > 0:
		m.answer.draft, m.answer.reason = msg.text, reasonBlockedBy
		m.screen = screenConfirm
	default:
		return m.post(msg.text)
	}
	return m, nil
}

// post は投稿を始める。投稿中は a を無視して二重投稿を防ぐ。
func (m Model) post(body string) (tea.Model, tea.Cmd) {
	m.writing = true
	m.writeStatus, m.writeStatusErr = m.answer.label+" にコメントを投稿中", false
	return m, postCmd(m.client, m.answer.target, m.answer.label, body)
}

// postCmd は action.Comment を別ゴルーチンで実行し、結果を postedMsg で返す。
func postCmd(client gh.GHClient, target action.Target, label, body string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
		defer cancel()
		return postedMsg{label: label, err: action.Comment(ctx, client, target, body)}
	}
}

// updatePosted は投稿の完了を扱う。結果はフッタに出すだけで、再取得はしない（1 件再取得は s18）。
func (m Model) updatePosted(msg postedMsg) Model {
	m.writing = false
	if msg.err != nil {
		m.writeStatus, m.writeStatusErr = msg.label+" へのコメントに失敗: "+msg.err.Error(), true
		return m
	}
	m.writeStatus, m.writeStatusErr = msg.label+" にコメントしました", false
	return m
}

// updateConfirmKey は確認画面のキーを扱う。
func (m Model) updateConfirmKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y":
		// マーカーは不変条件 7 が「書かない」なので、投稿する道を出さない。
		if m.answer.reason == reasonMarker {
			return m, nil
		}
		m.screen = m.answer.from
		return m.post(m.answer.draft)
	case "e":
		m.screen = m.answer.from
		cmd := m.openEditor(routeAnswer, m.answer.draft)
		return m, cmd
	case "esc":
		m.screen = m.answer.from
		m.writeStatus, m.writeStatusErr = "回答を中止しました", false
	}
	return m, nil
}

// renderConfirm は投稿前の確認を全画面で描く。警告の該当行と下書きを同じ画面で読めるようにする。
func (m Model) renderConfirm() string {
	lines := []string{"回答の確認: " + m.answer.label}
	if m.answer.reason == reasonMarker {
		lines = append(lines, "<!-- routine --> を含む本文は投稿できません（TUI からの投稿は人の発言でなければなりません）")
	} else {
		lines = append(lines, "blocked-by: で始まる行があります（dispatcher はこの行を含む最新コメントを正本にします）")
		lines = append(lines, action.BlockedByLines(m.answer.draft)...)
	}
	lines = append(lines, strings.Repeat("─", max(m.width, 0)))
	lines = append(lines, strings.Split(m.answer.draft, "\n")...)

	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "…")
	}
	lines = cut(lines, max(m.height-1, 0))
	return strings.Join(append(lines, m.footer(m.confirmHint())), "\n")
}

// confirmHint は確認画面のフッタ左。マーカーのときは投稿の道を出さない。
func (m Model) confirmHint() string {
	if m.answer.reason == reasonMarker {
		return "e 編集に戻る  Esc 中止  q 終了"
	}
	return "y 投稿  e 編集に戻る  Esc 中止  q 終了"
}
