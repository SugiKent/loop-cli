package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/loop-cli/internal/action"
	"github.com/SugiKent/loop-cli/internal/classify"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// defaultMergeMethod は Options の対応表に無いリポジトリで使う方式。
// gh pr merge は方式のフラグが無いと対話式になるので、空では渡さない（design.md 未決事項の既定値）。
const defaultMergeMethod = "squash"

// mergeState は `m` を押してから確認画面を抜けるまでの状態。
// pr は取り直した値で、画面の Card とは別に持つ（取得完了で Cards が入れ替わっても確認中の対象は変えない）。
type mergeState struct {
	repo     string
	number   int
	label    string
	method   string
	pr       model.PR
	blocked  []string
	warnings []string
	from     screen // `m` を押した画面。確認画面から戻る先
}

// mergeFetchedMsg は merge の判断材料の取り直しの完了。
type mergeFetchedMsg struct {
	label  string
	detail *gh.PRDetail
	state  *gh.PRMergeState
	err    error
}

// mergedMsg は 1 回の merge の完了。
type mergedMsg struct {
	label string
	err   error
}

// mergeKey は `m`（merge）を扱う。押した時点で GitHub から判断材料を取り直す
// （画面の値は取得時点のもので、merge の判断には使わない。human-turn-signals.md「merge 可否は表示時に取り直す」）。
func (m Model) mergeKey() (tea.Model, tea.Cmd) {
	if m.writing {
		return m, nil
	}
	repo, number, label, ok := m.mergeTarget()
	if !ok {
		return m, nil
	}
	m.merge = mergeState{repo: repo, number: number, label: label, method: m.mergeMethod(repo), from: m.screen}
	// 取り直しはまだ書き込みではないが、merge の手続きを重ねて始めさせないので書き込み中と同じ扱いにする。
	m.writing = true
	m.writeStatus, m.writeStatusErr = label+" の状態を取得中", false
	return m, mergeFetchCmd(m.client, repo, number, label)
}

// mergeTarget は今の画面の merge 対象のリポジトリ・番号・表示名を返す。
func (m Model) mergeTarget() (string, int, string, bool) {
	switch m.screen {
	case screenCard, screenPR:
		if len(m.detail.card.PRs) == 0 {
			return "", 0, "", false
		}
		pr := m.currentPR()
		return pr.Repo, pr.Number, prLabel(pr), true
	case screenQueue:
		rows := m.rows[m.tab]
		if m.cursor >= len(rows) {
			return "", 0, "", false
		}
		r := rows[m.cursor]
		if !r.isPR {
			return "", 0, "", false
		}
		return r.repo, r.number, fmt.Sprintf("%s PR#%d", r.repo, r.number), true
	}
	return "", 0, "", false
}

// mergeMethod は対象のリポジトリの merge 方式。対応表に無ければ squash。
func (m Model) mergeMethod(repo string) string {
	if method, ok := m.mergeMethods[repo]; ok {
		return method
	}
	return defaultMergeMethod
}

// mergeFetchCmd は ViewPR と ViewPRMergeState を並行に呼び、両方の完了を 1 つのメッセージで返す。
func mergeFetchCmd(client gh.GHClient, repo string, number int, label string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
		defer cancel()

		var (
			wg        sync.WaitGroup
			detail    *gh.PRDetail
			detailErr error
			state     *gh.PRMergeState
			stateErr  error
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			detail, detailErr = client.ViewPR(ctx, repo, number)
		}()
		go func() {
			defer wg.Done()
			state, stateErr = client.ViewPRMergeState(ctx, repo, number)
		}()
		wg.Wait()

		// 両方失敗したときは ViewPR のエラーを出す。
		err := detailErr
		if err == nil {
			err = stateErr
		}
		return mergeFetchedMsg{label: label, detail: detail, state: state, err: err}
	}
}

// updateMergeFetched は取り直しの完了を扱う。判断材料が欠けたまま merge の入口を開かない。
func (m Model) updateMergeFetched(msg mergeFetchedMsg) Model {
	m.writing = false
	// 取り直しの間も ? / u / Enter / Esc / g は動くので、結果が別の画面に届くことがある。
	// 成否を問わず捨てる（ヘルプ画面や URL 一覧画面を確認画面で踏み潰さない。
	// 戻り先が押した画面のままなので、確認画面を抜けた先が今の詳細と食い違うのも防ぐ）。
	if m.screen != m.merge.from {
		m.writeStatus, m.writeStatusErr = msg.label+" の merge を中止しました（画面が変わりました）", false
		return m
	}
	if msg.err != nil {
		m.writeStatus, m.writeStatusErr = msg.label+" の状態を取得できません: "+msg.err.Error(), true
		return m
	}
	m.writeStatus, m.writeStatusErr = "", false
	m.merge.pr = model.PR{
		Repo:       m.merge.repo,
		Number:     m.merge.number,
		Title:      msg.detail.Title,
		Body:       msg.detail.Body,
		Labels:     labelNamesOf(msg.detail.Labels),
		IsDraft:    msg.detail.IsDraft,
		MergeState: msg.state,
	}
	m.merge.blocked, m.merge.warnings = action.CheckMerge(m.merge.pr)
	m.screen = screenMergeConfirm
	return m
}

func labelNamesOf(labels []gh.Label) []string {
	out := make([]string, len(labels))
	for i, l := range labels {
		out[i] = l.Name
	}
	return out
}

// updateMergeConfirmKey は merge の確認画面のキーを扱う（q / Ctrl+C は Update が先に処理する）。
func (m Model) updateMergeConfirmKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y":
		// draft は「まだ人に見せる状態ですらない」という作者の宣言なので、merge の道を出さない。
		if len(m.merge.blocked) > 0 {
			return m, nil
		}
		m.screen = m.merge.from
		m.writing = true
		m.writeStatus, m.writeStatusErr = m.merge.label+" を merge 中", false
		return m, mergeCmd(m.client, m.merge.repo, m.merge.number, m.merge.method, m.merge.label)
	case "esc":
		m.screen = m.merge.from
		m.writeStatus, m.writeStatusErr = "merge を中止しました", false
	}
	return m, nil
}

// mergeCmd は action.Merge を別ゴルーチンで実行し、結果を mergedMsg で返す。
func mergeCmd(client gh.GHClient, repo string, number int, method, label string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
		defer cancel()
		return mergedMsg{label: label, err: action.Merge(ctx, client, repo, number, method)}
	}
}

// updateMerged は merge の完了を扱う。結果はフッタに出すだけで、再取得はしない（1 件再取得は s18）。
func (m Model) updateMerged(msg mergedMsg) Model {
	m.writing = false
	if msg.err != nil {
		m.writeStatus, m.writeStatusErr = msg.label+" の merge に失敗: "+msg.err.Error(), true
		return m
	}
	m.writeStatus, m.writeStatusErr = msg.label+" を merge しました", false
	return m
}

// renderMergeConfirm は merge 前の確認を全画面で描く。判断材料を本文より上に置き、
// `y` を押す指が止まる位置に警告を出す。スクロールは持たない（s10 の確認画面と同じ）。
func (m Model) renderMergeConfirm() string {
	s := m.merge
	labels := "なし"
	if len(s.pr.Labels) > 0 {
		labels = strings.Join(s.pr.Labels, " ")
	}
	checks := "緑以外"
	if classify.ChecksGreen(s.pr.MergeState) {
		checks = "緑"
	}
	lines := []string{
		"merge の確認: " + s.label,
		"方式: " + s.method,
		"labels: " + labels,
		strings.TrimSpace("mergeable: " + s.pr.MergeState.Mergeable + " " + s.pr.MergeState.MergeStateStatus),
		"checks: " + checks,
	}
	for _, b := range s.blocked {
		lines = append(lines, errorStyle.Render("merge できません: "+b))
	}
	for _, w := range s.warnings {
		lines = append(lines, "注意: "+w)
	}
	lines = append(lines, strings.Repeat("─", max(m.width, 0)))
	lines = append(lines, strings.Split(s.pr.Body, "\n")...)

	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "…")
	}
	lines = cut(lines, max(m.height-1, 0))
	return strings.Join(append(lines, m.footer(m.mergeConfirmHint())), "\n")
}

// mergeConfirmHint は merge の確認画面のフッタ左。拒否があるときは merge の道を出さない。
func (m Model) mergeConfirmHint() string {
	if len(m.merge.blocked) > 0 {
		return "Esc 中止  q 終了"
	}
	return "y merge  Esc 中止  q 終了"
}
