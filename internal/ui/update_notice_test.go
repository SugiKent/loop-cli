package ui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/fetch"
)

// checker は呼ばれた回数を数える UpdateChecker。
type checker struct {
	available bool
	calls     int
}

func (c *checker) check(context.Context) bool {
	c.calls++
	return c.available
}

// drainCmd はコマンド（tea.Batch を含む）を実行し、返るメッセージを平らにして返す。
func drainCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			msgs = append(msgs, drainCmd(c)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}

// TestHeaderShowsUpdateMark は新しい版があるときヘッダに印が出ることを検証する。
func TestHeaderShowsUpdateMark(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at}, updateCheckedMsg{available: true})

	header := plain(m)[0]
	order(t, header, "↑ update", "↻ 12:04")
}

// TestHeaderHidesUpdateMarkWithoutUpdate は更新が無いとき・確認しなかったときに印が出ないことを検証する。
func TestHeaderHidesUpdateMarkWithoutUpdate(t *testing.T) {
	for _, tc := range []struct {
		name string
		msgs []tea.Msg
	}{
		{"新しい版が無い", []tea.Msg{updateCheckedMsg{available: false}}},
		{"確認していない", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msgs := append([]tea.Msg{fetchedMsg{res: exampleResult(t), at: at}}, tc.msgs...)
			m, _ := send(newModel(nil), msgs...)

			text := plainText(m)
			if strings.Contains(text, "↑ update") {
				t.Errorf("印が出ている: %q", plain(m)[0])
			}
			if last := plain(m)[len(plain(m))-1]; strings.Contains(last, "update") {
				t.Errorf("フッタに update が出ている: %q", last)
			}
		})
	}
}

// TestHeaderDropsTabNameOfFourthAtDefaultWidth は既定幅 80 で [4] のタブ名だけが落ちることを検証する。
func TestHeaderDropsTabNameOfFourthAtDefaultWidth(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at}, updateCheckedMsg{available: true})

	header := plain(m)[0]
	for _, want := range []string{"[1]今やる 1", "[2]バックログ 1", "[3]進行中 0", "[4] 0", "↑ update", "↻ 12:04"} {
		if !strings.Contains(header, want) {
			t.Errorf("ヘッダに %q が無い: %q", want, header)
		}
	}
	if strings.Contains(header, "異常") {
		t.Errorf("[4] のタブ名が残っている: %q", header)
	}
}

// TestHeaderDropsUpdateMarkBeforeClock は幅が足りないとき時刻より先に印が落ちることを検証する。
func TestHeaderDropsUpdateMarkBeforeClock(t *testing.T) {
	m, _ := send(newModel(nil), fetchedMsg{res: exampleResult(t), at: at}, updateCheckedMsg{available: true},
		tea.WindowSizeMsg{Width: 60, Height: 40})

	header := plain(m)[0]
	if !strings.Contains(header, "↻ 12:04") {
		t.Errorf("時刻が落ちている: %q", header)
	}
	if strings.Contains(header, "update") {
		t.Errorf("印が残っている: %q", header)
	}
}

// TestInitChecksUpdateOnce は確認が起動時に 1 度だけ行われ、R では繰り返さないことを検証する。
func TestInitChecksUpdateOnce(t *testing.T) {
	c := &checker{available: true}
	fetcher := func(context.Context) (*fetch.Result, error) { return &fetch.Result{}, nil }
	m := newModelOpts(fetcher, Options{CheckUpdate: c.check})

	msgs := drainCmd(m.Init())
	if c.calls != 1 {
		t.Fatalf("確認が %d 回, want 1", c.calls)
	}
	var checked bool
	for _, msg := range msgs {
		if u, ok := msg.(updateCheckedMsg); ok && u.available {
			checked = true
		}
	}
	if !checked {
		t.Error("updateCheckedMsg が返らない")
	}

	_, cmd := send(m, fetchedMsg{res: &fetch.Result{}, at: at}, rKey)
	drainCmd(cmd)
	if c.calls != 1 {
		t.Errorf("R の後に確認が %d 回, want 1", c.calls)
	}
}

// TestInitWithoutCheckerDoesNotCheck は確認の関数が nil なら確認しないことを検証する。
func TestInitWithoutCheckerDoesNotCheck(t *testing.T) {
	fetcher := func(context.Context) (*fetch.Result, error) { return &fetch.Result{}, nil }
	m := newModelOpts(fetcher, Options{})

	for _, msg := range drainCmd(m.Init()) {
		if _, ok := msg.(updateCheckedMsg); ok {
			t.Error("確認していないのに updateCheckedMsg が返った")
		}
	}
}
