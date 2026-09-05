package ui

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/sugi-loop/internal/fetch"
	"github.com/SugiKent/sugi-loop/internal/gh"
	"github.com/SugiKent/sugi-loop/internal/model"
)

// Fetcher は Card 群の取得。cmd/sugi-loop が fetch.Fetch を client と repos で閉じて渡す。
type Fetcher func(ctx context.Context) (*fetch.Result, error)

// fetchedMsg は 1 回の取得の完了。at は完了時刻で、経過とヘッダの時刻の基準になる。
type fetchedMsg struct {
	res *fetch.Result
	err error
	at  time.Time
}

// tabOrder は 1–4 と Tab の並び。
var tabOrder = []model.Tab{model.TabNow, model.TabBacklog, model.TabInProgress, model.TabAbnormal}

// Model は今やるキュー画面。壁時計は読まず、経過は最後の取得完了時刻を基準にする。
type Model struct {
	fetcher Fetcher
	client  gh.GHClient
	editor  Editor
	cards   []model.Card
	rows    map[model.Tab][]row

	tab    model.Tab
	cursor int

	at       time.Time // ゼロ値は未取得
	fetching bool
	errText  string
	partial  string

	width       int
	height      int
	showPreview bool

	screen screen
	detail detailState

	answer          answerState
	posting         bool
	answerStatus    string
	answerStatusErr bool

	spinner spinner.Model
}

// New は取得前の Model を返す。初期状態は常に「これから取得する」。
// client は回答の投稿に、editor は下書きの編集に使う。
func New(fetcher Fetcher, client gh.GHClient, editor Editor) Model {
	return Model{
		fetcher:  fetcher,
		client:   client,
		editor:   editor,
		rows:     buildRows(nil),
		tab:      model.TabNow,
		fetching: true,
		width:    80,
		height:   24,
		spinner:  spinner.New(spinner.WithSpinner(spinner.MiniDot)),
	}
}

// fetchCmd は Fetcher を別ゴルーチンで実行し、完了を fetchedMsg で返す。
func fetchCmd(fetcher Fetcher) tea.Cmd {
	return func() tea.Msg {
		res, err := fetcher(context.Background())
		return fetchedMsg{res: res, err: err, at: time.Now()}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchCmd(m.fetcher))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.screen == screenCard || m.screen == screenPR {
			m.refreshDetail()
		}
		return m, nil

	case spinner.TickMsg:
		if !m.fetching {
			return m, nil
		}
		s, cmd := m.spinner.Update(msg)
		m.spinner = s
		return m, cmd

	case fetchedMsg:
		m.fetching = false
		if msg.err != nil {
			// D-002: 失敗時は前回の Cards と最終更新時刻を維持する。
			m.errText = msg.err.Error()
			m.partial = ""
		} else {
			m.cards = msg.res.Cards
			m.rows = buildRows(m.cards)
			m.at = msg.at
			m.errText = ""
			m.partial = ""
			if n := len(msg.res.Errors); n > 0 {
				m.partial = fmt.Sprintf("詳細取得の失敗 %d 件: %v", n, msg.res.Errors[0])
			}
		}
		m.clampCursor()
		return m, nil

	case editedMsg:
		return m.updateEdited(msg)

	case postedMsg:
		return m.updatePosted(msg), nil

	case tea.KeyPressMsg:
		key := msg.String()
		if key == "q" || key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.screen == screenConfirm {
			return m.updateConfirmKey(key)
		}
		// a は 3 画面すべてで効き、対象は画面が見せているものに決まる。
		if key == "a" {
			return m.answerKey()
		}
		if m.screen != screenQueue {
			return m.updateDetailKey(key), nil
		}
		return m.updateKey(key)
	}
	return m, nil
}

// updateKey はキュー画面のキーを扱う。表に無いキーと後続 change のキーは何もしない。
func (m Model) updateKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.cursor < len(m.rows[m.tab])-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "1", "2", "3", "4":
		m.tab = tabOrder[key[0]-'1']
		m.clampCursor()
	case "tab":
		m.tab = tabOrder[(m.tabIndex()+1)%len(tabOrder)]
		m.clampCursor()
	case "p":
		// 2 ペインでは表もプレビューも出ているので何もしない。
		if !m.twoPane() {
			m.showPreview = !m.showPreview
		}
	case "enter":
		return m.openDetail(), nil
	}
	return m, nil
}

func (m Model) tabIndex() int {
	for i, t := range tabOrder {
		if t == m.tab {
			return i
		}
	}
	return 0
}

// clampCursor は選択行を 0 ≤ 添字 < 行数 に収める。行数 0 なら 0。
func (m *Model) clampCursor() {
	n := len(m.rows[m.tab])
	if m.cursor >= n {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// twoPane は表とプレビューを同時に出せる端末かを返す。
func (m Model) twoPane() bool { return m.width >= 80 && m.height >= 20 }
