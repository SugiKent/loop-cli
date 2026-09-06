package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/model"
)

// Notifier はデスクトップ通知を 1 件出す。cmd/loop-cli が beeep を包んで渡し、
// internal/ui は beeep を import しない。nil なら通知しない。
type Notifier func(title, body string) error

// notifyKey は今やるカードの同一性。主体（リポジトリ・番号・PR か）が変われば
// 人がすべきことが変わっているので、別のカードとして数える。
type notifyKey struct {
	repo   string
	number int
	isPR   bool
}

// addedNow は今やるタブで prev に無く next にある Card を next の順で返す。
// 減った Card と、要約や最終更新時刻だけが変わった Card は返さない。
func addedNow(prev, next []model.Card) []model.Card {
	seen := make(map[notifyKey]bool)
	for _, c := range prev {
		if c.Result.Tab == model.TabNow {
			seen[keyOf(c)] = true
		}
	}
	var added []model.Card
	for _, c := range next {
		if c.Result.Tab == model.TabNow && !seen[keyOf(c)] {
			added = append(added, c)
		}
	}
	return added
}

func keyOf(c model.Card) notifyKey {
	repo, number, isPR, _, _ := Subject(c)
	return notifyKey{repo: repo, number: number, isPR: isPR}
}

// notifyBody は通知の本文。1 行目が「いま人が何をすべきか」、2 行目が主体の表示名。
func notifyBody(c model.Card) string {
	repo, number, isPR, _, _ := Subject(c)
	name := fmt.Sprintf("%s #%d", repo, number)
	if isPR {
		name = fmt.Sprintf("%s PR#%d", repo, number)
	}
	return c.Result.Summary + "\n" + name
}

// notifyCmd は増えた Card を 1 件 1 通知で順に知らせる。beeep は OS の通知が終わるまで
// 待つことがあるので Update の中では呼ばない。失敗は無視し、Model には何も返さない。
func notifyCmd(notify Notifier, cards []model.Card) tea.Cmd {
	return func() tea.Msg {
		for _, c := range cards {
			_ = notify("loop-cli", notifyBody(c))
		}
		return nil
	}
}
