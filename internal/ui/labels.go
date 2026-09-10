package ui

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/SugiKent/loop-cli/internal/action"
	"github.com/SugiKent/loop-cli/internal/gh"
)

// labelPickerState は `L` を押してからラベル一覧画面を抜けるまでの状態。
// target / origin / from は `L` を押した時点の写しで、一覧を出している間に自動更新で Cards が
// 入れ替わっても変えない（画面の状態を見ずに走る自動更新に印と書き先を動かされないため）。
type labelPickerState struct {
	labels []gh.RepoLabel  // 表から写したリポジトリのラベル。並びは gh の名前昇順のまま
	cursor int             // 選択位置。スクロール位置は持たない
	origin map[string]bool // 元の集合。行の印・変更件数・「変更が無い」の判定の基準
	want   map[string]bool // 送信後にこうなっていてほしい集合
	target action.Target
	label  string // ステータスに出す `<repo> #<番号>`
	from   screen // `L` を押した画面。Esc と送信で戻る先
}

// labelsFetchedMsg は 1 回の ListLabels の完了。
type labelsFetchedMsg struct {
	repo   string
	labels []gh.RepoLabel
	err    error
}

// labelsEditedMsg は 1 回のラベル一括編集の完了。add / remove は実際に書いた名前。
type labelsEditedMsg struct {
	label  string
	add    []string
	remove []string
	err    error
}

// labelKey は `L`（ラベル一覧を開く）を扱う。対象は画面が見せているものに決まる（`a` / `u` と同じ規則）。
func (m Model) labelKey() (tea.Model, tea.Cmd) {
	if m.writing {
		return m, nil
	}
	target, labels, ok := m.labelTarget()
	if !ok {
		return m, nil
	}
	m.labelPicker = labelPickerState{
		origin: labelSet(labels),
		want:   labelSet(labels),
		target: target,
		label:  fmt.Sprintf("%s #%d", target.Repo, target.Number),
		from:   m.screen,
	}
	if cached, ok := m.repoLabels[target.Repo]; ok {
		return m.openLabelPicker(cached), nil
	}
	// 取得はまだ書き込みではないが、手続きを重ねて始めさせないので書き込み中と同じ扱いにする。
	m.writing = true
	m.writeStatus, m.writeStatusErr = target.Repo+" のラベル一覧を取得中", false
	return m, listLabelsCmd(m.client, target.Repo)
}

// labelTarget は今の画面の書き先と、そこに今付いているラベル名を返す。
func (m Model) labelTarget() (action.Target, []string, bool) {
	switch m.screen {
	case screenCard:
		issue := m.detail.card.Issue
		return targetOfIssue(issue), issue.Labels, true
	case screenPR:
		pr := m.currentPR()
		return targetOfPR(pr), pr.Labels, true
	}
	rows := m.rows[m.tab]
	if m.cursor >= len(rows) {
		return action.Target{}, nil, false
	}
	r := rows[m.cursor]
	return action.Target{Repo: r.repo, Number: r.number, IsPR: r.isPR}, r.labels, true
}

// openLabelPicker は「一覧を開く判定」。キャッシュに有った経路と取得した経路が合流する 1 か所で、
// どちらを通っても結果は同じになる。
func (m Model) openLabelPicker(labels []gh.RepoLabel) Model {
	if len(labels) == 0 {
		m.writeStatus, m.writeStatusErr = "ラベルがありません", false
		return m
	}
	// 取得中も o / ? / u / Esc は動くので、結果が別の画面に届くことがある。開いた画面を踏み潰さない。
	if m.screen != m.labelPicker.from {
		m.writeStatus, m.writeStatusErr =
			m.labelPicker.target.Repo+" のラベル一覧の表示を中止しました（画面が変わりました）", false
		return m
	}
	m.labelPicker.labels = labels
	m.labelPicker.cursor = 0
	m.screen = screenLabels
	return m
}

// listLabelsCmd は ListLabels を別ゴルーチンで実行し、結果を labelsFetchedMsg で返す。
func listLabelsCmd(client gh.GHClient, repo string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
		defer cancel()
		labels, err := client.ListLabels(ctx, repo)
		return labelsFetchedMsg{repo: repo, labels: labels, err: err}
	}
}

// updateLabelsFetched は取得の完了を扱う。表は R でも自動更新でも捨てず、プロセスが終わるまで持つ。
func (m Model) updateLabelsFetched(msg labelsFetchedMsg) Model {
	m.writing = false
	if msg.err != nil {
		m.writeStatus, m.writeStatusErr = msg.repo+" のラベル一覧を取得できません: "+msg.err.Error(), true
		return m
	}
	if m.repoLabels == nil {
		m.repoLabels = map[string][]gh.RepoLabel{}
	}
	m.repoLabels[msg.repo] = msg.labels
	m.writeStatus, m.writeStatusErr = "", false
	return m.openLabelPicker(msg.labels)
}

// updateLabelKey はラベル一覧画面のキーを扱う（q / Ctrl+C は Update が先に処理する）。
// 表に無いキー（L / l / a / t / m / o / ? / u / R を含む）は何もしない。L を含めるので、
// 一覧を出したまま L を押しても戻り先は上書きされない。
func (m Model) updateLabelKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.labelPicker.cursor < len(m.labelPicker.labels)-1 {
			m.labelPicker.cursor++
		}
	case "k", "up":
		if m.labelPicker.cursor > 0 {
			m.labelPicker.cursor--
		}
	case "space":
		// 印の切り替えは GitHub に触らない。集合は写して書き換える（Model の値を共有しない）。
		name := m.labelPicker.labels[m.labelPicker.cursor].Name
		want := maps.Clone(m.labelPicker.want)
		if want[name] {
			delete(want, name)
		} else {
			want[name] = true
		}
		m.labelPicker.want = want
	case "enter":
		return m.submitLabels()
	case "esc":
		m.screen = m.labelPicker.from
		// 一覧中の WindowSizeMsg は詳細の寸法を作り直さないので、戻るときに作り直す。
		if m.screen == screenCard || m.screen == screenPR {
			m.refreshDetail()
		}
	}
	return m, nil
}

// submitLabels は `Enter` を扱う。書き込みを始める前に一覧を閉じる。一覧に留まると送信中の
// Space / Enter で二重に送れてしまい、あとから届いた結果が別の画面を引き剥がす。
func (m Model) submitLabels() (tea.Model, tea.Cmd) {
	s := m.labelPicker
	m.screen = s.from
	if m.screen == screenCard || m.screen == screenPR {
		m.refreshDetail()
	}
	if maps.Equal(s.origin, s.want) {
		m.writeStatus, m.writeStatusErr = s.label+" のラベルに変更はありません", false
		return m, nil
	}
	m.writing = true
	m.writeStatus, m.writeStatusErr = s.label+" のラベルを更新中", false
	return m, setLabelsCmd(m.client, s.target, s.label, slices.Sorted(maps.Keys(s.want)))
}

// setLabelsCmd は action.SetLabels を別ゴルーチンで実行し、結果を labelsEditedMsg で返す。
func setLabelsCmd(client gh.GHClient, target action.Target, label string, want []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
		defer cancel()
		add, remove, err := action.SetLabels(ctx, client, target, want)
		return labelsEditedMsg{label: label, add: add, remove: remove, err: err}
	}
}

// updateLabelsEdited は送信の完了を扱う。結果はフッタに出すだけで、画面も Cards も変えない
// （反映は R / 自動更新に任せる。`t` と同じ扱い）。
func (m Model) updateLabelsEdited(msg labelsEditedMsg) Model {
	m.writing = false
	switch {
	case msg.err != nil:
		// gh は付ける・外すを別の mutation で送るので、これは「何も変わっていない」を意味しない。
		m.writeStatus, m.writeStatusErr = msg.label+" のラベルを更新できません: "+msg.err.Error(), true
	case len(msg.add) == 0 && len(msg.remove) == 0:
		m.writeStatus, m.writeStatusErr = msg.label+" のラベルに変更はありません", false
	default:
		m.writeStatus, m.writeStatusErr =
			msg.label+" のラベルを更新しました（"+labelChanges(msg.add, msg.remove)+"）", false
	}
	return m
}

// labelChanges は付けた名前を +<名前>、外した名前を -<名前> にして、付ける分・外す分の順に並べる。
func labelChanges(add, remove []string) string {
	parts := make([]string, 0, len(add)+len(remove))
	for _, name := range add {
		parts = append(parts, "+"+name)
	}
	for _, name := range remove {
		parts = append(parts, "-"+name)
	}
	return strings.Join(parts, " ")
}

// labelSet は集合の写しを作る。Model の値の間で共有しないよう、毎回新しい map にする。
func labelSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

// labelChangeCount は元の集合と今の集合の差分の件数。
func labelChangeCount(origin, want map[string]bool) int {
	n := 0
	for name := range want {
		if !origin[name] {
			n++
		}
	}
	for name := range origin {
		if !want[name] {
			n++
		}
	}
	return n
}

// renderLabels はラベル一覧画面を 見出し / 空行 / ラベルの行 / 空行 / フッタ の順に描く。
func (m Model) renderLabels() string {
	s := m.labelPicker
	// 表示できる行数は 高さ − 見出し − 空行 − フッタ。スクロール位置は持たず選択位置から決める。
	h := max(m.height-3, 0)
	start := max(s.cursor-h+1, 0)
	lines := []string{fmt.Sprintf("ラベルを付け外し（変更 %d 件）", labelChangeCount(s.origin, s.want)), ""}
	for i := start; i < len(s.labels) && i < start+h; i++ {
		lines = append(lines, m.labelLine(s.labels[i], i == s.cursor))
	}
	for len(lines) < m.height-1 {
		lines = append(lines, "")
	}
	lines = cut(lines, max(m.height-1, 0))
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "…")
	}
	hint := "j/k 選択  Space 切替  Enter 送信  Esc 戻る  q 終了"
	return strings.Join(append(lines, m.footer(hint)), "\n")
}

// labelLine は一覧の 1 行 `<選択の印><状態の印> <名前>  <説明>`。色（gh label list の color）は使わない。
func (m Model) labelLine(l gh.RepoLabel, selected bool) string {
	mark := "  "
	if selected {
		mark = "▶ "
	}
	line := mark + labelMark(m.labelPicker.origin[l.Name], m.labelPicker.want[l.Name]) + " " + l.Name
	if l.Description != "" {
		line += "  " + l.Description
	}
	return line
}

// labelMark は元の集合と今の集合の組み合わせで決まる 4 種の状態の印。
func labelMark(had, wants bool) string {
	switch {
	case had && wants:
		return "[x]"
	case had:
		return "[-]"
	case wants:
		return "[+]"
	}
	return "[ ]"
}
