package ui

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
	"github.com/SugiKent/loop-cli/internal/snapshot"
)

// Options は New の起動時の選択肢。ゼロ値は
// 「スナップショット無し・自動更新無し・通知無し」で、従来どおりの Model になる。
type Options struct {
	Snapshot        *snapshot.Snapshot // 起動時の stale 表示に使う前回の Card 群と保存時刻
	RefreshInterval time.Duration      // 自動更新の間隔。0 なら自動更新しない
	Notify          Notifier           // デスクトップ通知。nil なら通知しない
}

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

	width  int
	height int

	screen   screen
	detail   detailState
	helpFrom screen // ヘルプ画面を開いた画面。? / Esc で戻る先
	urls     urlListState

	answer         answerState
	writing        bool
	writeStatus    string
	writeStatusErr bool

	refreshInterval time.Duration
	notify          Notifier

	spinner spinner.Model
}

// New は取得前の Model を返す。初期状態は常に「これから取得する」。
// client は回答の投稿に、editor は下書きの編集に使う。
func New(fetcher Fetcher, client gh.GHClient, editor Editor, opts Options) Model {
	m := Model{
		fetcher:         fetcher,
		client:          client,
		editor:          editor,
		rows:            buildRows(nil),
		tab:             model.TabNow,
		fetching:        true,
		width:           80,
		height:          24,
		refreshInterval: opts.RefreshInterval,
		notify:          opts.Notify,
		spinner:         spinner.New(spinner.WithSpinner(spinner.MiniDot)),
	}
	// スナップショットがあれば前回の表と保存時刻から始める（D-002「起動直後は stale 表示」）。
	if opts.Snapshot != nil {
		m.cards = opts.Snapshot.Cards
		m.rows = buildRows(m.cards)
		m.at = opts.Snapshot.At
	}
	return m
}

// fetchCmd は Fetcher を別ゴルーチンで実行し、完了を fetchedMsg で返す。
func fetchCmd(fetcher Fetcher) tea.Cmd {
	return func() tea.Msg {
		res, err := fetcher(context.Background())
		return fetchedMsg{res: res, err: err, at: time.Now()}
	}
}

func (m Model) Init() tea.Cmd {
	if m.refreshInterval > 0 {
		return tea.Batch(m.spinner.Tick, fetchCmd(m.fetcher), tickCmd(m.refreshInterval))
	}
	return tea.Batch(m.spinner.Tick, fetchCmd(m.fetcher))
}

// startFetch は 2 回目以降の取得を始める。R（s12）と自動更新（s13）が使う共通の経路で、
// 取得中かどうかは見ない（呼び手が fetching を見てから呼ぶ）。
func (m *Model) startFetch() tea.Cmd {
	m.fetching = true
	m.errText = ""
	m.partial = ""
	m.writeStatus, m.writeStatusErr = "", false
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
			// 通知の比較は差し替える前の Cards と、最終更新時刻の有無（前回があるか）で決まる。
			prev, hadPrev := m.cards, !m.at.IsZero()
			m.cards = msg.res.Cards
			m.rows = buildRows(m.cards)
			m.at = msg.at
			m.errText = ""
			m.partial = ""
			if n := len(msg.res.Errors); n > 0 {
				m.partial = fmt.Sprintf("詳細取得の失敗 %d 件: %v", n, msg.res.Errors[0])
			}
			if m.notify != nil && hadPrev {
				if added := addedNow(prev, m.cards); len(added) > 0 {
					m.clampCursor()
					return m, notifyCmd(m.notify, added)
				}
			}
		}
		m.clampCursor()
		return m, nil

	case editedMsg:
		return m.updateEdited(msg)

	case postedMsg:
		return m.updatePosted(msg), nil

	case toggledMsg:
		return m.updateToggled(msg), nil

	case browsedMsg:
		return m.updateBrowsed(msg), nil

	case openedURLMsg:
		return m.updateOpenedURL(msg), nil

	case refreshTickMsg:
		return m.updateTick()

	case tea.KeyPressMsg:
		key := msg.String()
		if key == "q" || key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.screen == screenConfirm {
			return m.updateConfirmKey(key)
		}
		// ヘルプ画面は a / t / o より先に振り分ける（後ろだと閉じずに書き込みが起きる）。
		if m.screen == screenHelp {
			return m.updateHelpKey(key), nil
		}
		// URL 一覧画面と u も a / t / o / ? より先に振り分ける
		// （後ろだと一覧の裏にある対象へ書き込みやブラウザ起動が起きる）。
		// 一覧画面の分岐が先なので、一覧を出したまま u を押しても戻り先は上書きされない。
		if m.screen == screenURL {
			return m.updateURLKey(key)
		}
		if key == "u" {
			return m.urlKey()
		}
		// a は 3 画面すべてで効き、対象は画面が見せているものに決まる。
		if key == "a" {
			return m.answerKey()
		}
		// t の対象は常に Issue で、キュー画面とカード詳細の 2 画面で効く。
		if key == "t" {
			return m.todoKey()
		}
		// o と ? も 3 画面すべてで効く。updateDetailKey は Cmd を返せないのでここに置く。
		if key == "o" {
			return m.browseKey()
		}
		if key == "?" {
			m.helpFrom = m.screen
			m.screen = screenHelp
			return m, nil
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
	case "enter":
		return m.openDetail(), nil
	case "R":
		if m.fetching {
			return m, nil
		}
		return m, m.startFetch()
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
