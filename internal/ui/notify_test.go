package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/model"
	"github.com/SugiKent/loop-cli/internal/snapshot"
)

// nowPRCard は主体が PR の今やるカードを 1 枚作る。
func nowPRCard(repo string, number int, priority int, updatedAt time.Time) model.Card {
	result := model.Result{Situation: model.SituationA, Priority: priority, Tab: model.TabNow, Summary: "回答する"}
	pr := model.PR{Repo: repo, Number: number, Title: "手書き PR", State: "OPEN", UpdatedAt: updatedAt, Result: result}
	return model.Card{PRs: []model.PR{pr}, Result: result}
}

// backlogCard はバックログタブに入る手書きの Card。
func backlogCard(repo string, number int) model.Card {
	result := model.Result{Situation: model.SituationE, Priority: 4, Tab: model.TabBacklog, Summary: "着手を決める"}
	return model.Card{Issue: &model.Issue{Repo: repo, Number: number, Title: "手書き", Result: result}, Result: result}
}

func TestAddedNowReturnsOnlyAddedCards(t *testing.T) {
	prev := []model.Card{
		nowPRCard("org/app", 131, 1, at),
		nowCard("org/app", 91, 1, at),
	}
	next := []model.Card{
		nowCard("org/app", 91, 1, at.Add(time.Hour)),
		nowPRCard("org/web", 88, 1, at),
		backlogCard("org/app", 140),
	}

	added := addedNow(prev, next)

	if len(added) != 1 {
		t.Fatalf("増えたカード = %d 枚, want 1: %+v", len(added), added)
	}
	if repo, number, isPR, _, _ := Subject(added[0]); repo != "org/web" || number != 88 || !isPR {
		t.Errorf("増えたカード = %s %d (PR=%v), want org/web 88 (PR=true)", repo, number, isPR)
	}
}

func TestAddedNowIsEmptyForSameCards(t *testing.T) {
	cards := exampleResult(t).Cards

	if added := addedNow(cards, cards); len(added) != 0 {
		t.Errorf("同じ集合で %d 枚返った: %+v", len(added), added)
	}
}

func TestAddedNowWithNilPrevReturnsAllNowCards(t *testing.T) {
	added := addedNow(nil, exampleResult(t).Cards)

	if len(added) != 1 {
		t.Fatalf("増えたカード = %d 枚, want 1: %+v", len(added), added)
	}
	if n := added[0].Issue.Number; n != 108 {
		t.Errorf("増えたカードの issue = %d, want 108", n)
	}
}

func TestAddedNowCountsSubjectChange(t *testing.T) {
	card := cardOf(t, exampleResult(t), 108)
	moved := card
	moved.Result = model.Result{Situation: model.SituationB, Priority: 2, Tab: model.TabNow, Summary: "方針を決める"}
	issue := *card.Issue
	issue.Result = moved.Result
	moved.Issue = &issue

	added := addedNow([]model.Card{card}, []model.Card{moved})

	if len(added) != 1 {
		t.Fatalf("増えたカード = %d 枚, want 1: %+v", len(added), added)
	}
	if _, number, isPR, _, _ := Subject(added[0]); number != 108 || isPR {
		t.Errorf("増えたカードの主体 = %d (PR=%v), want 108 (PR=false)", number, isPR)
	}
}

// notifyRecord は 1 件の通知。
type notifyRecord struct{ title, body string }

// recordingNotifier は呼び出しを順に記録する Notifier を返す。errs の i 番目が
// i+1 回目の呼び出しの戻り値になり、足りなければ nil を返す。
func recordingNotifier(errs ...error) (Notifier, *[]notifyRecord) {
	var got []notifyRecord
	return func(title, body string) error {
		got = append(got, notifyRecord{title: title, body: body})
		if i := len(got) - 1; i < len(errs) {
			return errs[i]
		}
		return nil
	}, &got
}

// bodyLines は記録した本文を行に割る。
func bodyLines(t *testing.T, r notifyRecord) []string {
	t.Helper()
	lines := strings.Split(r.body, "\n")
	if len(lines) != 2 {
		t.Fatalf("本文が 2 行でない: %q", r.body)
	}
	return lines
}

// runNotify は返ったコマンドを実行する。nil なら失敗させる。
func runNotify(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("通知のコマンドが返っていない")
	}
	if msg := cmd(); msg != nil {
		t.Errorf("通知のコマンドが %T を返した, want nil", msg)
	}
}

// at12 は 12:00 の取得完了時刻（at の 4 分前）。
var at12 = at.Add(-4 * time.Minute)

func TestNotifiesEachAddedCard(t *testing.T) {
	notify, got := recordingNotifier()
	res := exampleResult(t)
	m, _ := send(newModelOpts(nil, Options{Notify: notify}),
		fetchedMsg{res: backlogOnlyResult(t), at: at12})
	if len(*got) != 0 {
		t.Fatalf("初回取得で %d 件通知された: %+v", len(*got), *got)
	}

	_, cmd := send(m, fetchedMsg{res: res, at: at})
	runNotify(t, cmd)

	if len(*got) != 1 {
		t.Fatalf("通知 = %d 件, want 1: %+v", len(*got), *got)
	}
	if (*got)[0].title != "sugi-loop" {
		t.Errorf("title = %q, want sugi-loop", (*got)[0].title)
	}
	lines := bodyLines(t, (*got)[0])
	if want := cardOf(t, res, 108).Result.Summary; lines[0] != want {
		t.Errorf("本文 1 行目 = %q, want %q", lines[0], want)
	}
	if lines[1] != "org/app PR#131" {
		t.Errorf("本文 2 行目 = %q, want org/app PR#131", lines[1])
	}
}

func TestNotifiesTwoAddedCardsInRowOrder(t *testing.T) {
	notify, got := recordingNotifier()
	m, _ := send(newModelOpts(nil, Options{Notify: notify}),
		fetchedMsg{res: &fetch.Result{}, at: at12})

	// 今やるタブの並びは Priority が同じなら UpdatedAt の新しい順。
	res := &fetch.Result{Cards: []model.Card{
		nowPRCard("org/app", 131, 1, at),
		nowPRCard("org/web", 88, 1, at.Add(-time.Hour)),
		backlogCard("org/app", 140),
	}}
	_, cmd := send(m, fetchedMsg{res: res, at: at})
	runNotify(t, cmd)

	if len(*got) != 2 {
		t.Fatalf("通知 = %d 件, want 2: %+v", len(*got), *got)
	}
	want := []string{"org/app PR#131", "org/web PR#88"}
	for i, r := range *got {
		if line := bodyLines(t, r)[1]; line != want[i] {
			t.Errorf("%d 件目の本文 2 行目 = %q, want %q", i+1, line, want[i])
		}
	}
}

func TestNoNotifyOnFirstFetch(t *testing.T) {
	notify, got := recordingNotifier()

	_, cmd := send(newModelOpts(nil, Options{Notify: notify}), fetchedMsg{res: exampleResult(t), at: at})

	if cmd != nil {
		if msg := cmd(); msg != nil {
			t.Errorf("初回取得のコマンドが %T を返した", msg)
		}
	}
	if len(*got) != 0 {
		t.Errorf("初回取得で %d 件通知された: %+v", len(*got), *got)
	}
}

func TestNotifiesComparedWithSnapshot(t *testing.T) {
	notify, got := recordingNotifier()
	snap := &snapshot.Snapshot{Cards: backlogOnlyResult(t).Cards, At: staleAt}

	_, cmd := send(newModelOpts(nil, Options{Snapshot: snap, Notify: notify}),
		fetchedMsg{res: exampleResult(t), at: at})
	runNotify(t, cmd)

	if len(*got) != 1 {
		t.Fatalf("通知 = %d 件, want 1: %+v", len(*got), *got)
	}
	if line := bodyLines(t, (*got)[0])[1]; line != "org/app PR#131" {
		t.Errorf("本文 2 行目 = %q, want org/app PR#131", line)
	}
}

func TestNoNotifyWhenNothingAdded(t *testing.T) {
	notify, got := recordingNotifier()
	res := exampleResult(t)
	m, _ := send(newModelOpts(nil, Options{Notify: notify}), fetchedMsg{res: res, at: at12})

	m, cmd := send(m, fetchedMsg{res: res, at: at})
	if cmd != nil {
		t.Error("同じ集合で通知のコマンドが返った")
	}
	if _, cmd = send(m, fetchedMsg{res: &fetch.Result{}, at: at.Add(time.Minute)}); cmd != nil {
		t.Error("減っただけで通知のコマンドが返った")
	}
	if len(*got) != 0 {
		t.Errorf("通知 = %d 件, want 0: %+v", len(*got), *got)
	}
}

func TestNoNotifierDoesNothing(t *testing.T) {
	res := exampleResult(t)
	m, cmd := send(newModelOpts(nil, Options{}), fetchedMsg{res: backlogOnlyResult(t), at: at12})
	if cmd != nil {
		t.Error("Notify 無しでコマンドが返った")
	}

	m, cmd = send(m, fetchedMsg{res: res, at: at})
	if cmd != nil {
		t.Error("Notify 無しでコマンドが返った")
	}
	if len(m.cards) != len(res.Cards) {
		t.Errorf("Cards = %d 件, want %d 件", len(m.cards), len(res.Cards))
	}
}

func TestNotifyErrorIsIgnored(t *testing.T) {
	notify, got := recordingNotifier(errors.New("notification failed"))
	m, _ := send(newModelOpts(nil, Options{Notify: notify}), fetchedMsg{res: &fetch.Result{}, at: at12})

	res := &fetch.Result{Cards: []model.Card{
		nowPRCard("org/app", 131, 1, at),
		nowPRCard("org/web", 88, 1, at.Add(-time.Hour)),
		backlogCard("org/app", 140),
	}}
	m, cmd := send(m, fetchedMsg{res: res, at: at})
	runNotify(t, cmd)

	if len(*got) != 2 {
		t.Fatalf("通知 = %d 件, want 2: %+v", len(*got), *got)
	}
	if strings.Contains(plainText(m), "notification failed") {
		t.Errorf("通知の失敗が画面に出ている:\n%s", plainText(m))
	}
}

func TestNoNotifyOnFetchError(t *testing.T) {
	notify, got := recordingNotifier()
	res := exampleResult(t)
	m, _ := send(newModelOpts(nil, Options{Notify: notify}), fetchedMsg{res: res, at: at})

	m, cmd := send(m, fetchedMsg{
		err: errors.New("search issues: gh search issues: exit 1: rate limited"), at: at.Add(time.Minute)})
	if cmd != nil {
		t.Error("取得失敗で通知のコマンドが返った")
	}
	if len(*got) != 0 {
		t.Errorf("通知 = %d 件, want 0: %+v", len(*got), *got)
	}
	if len(m.cards) != len(res.Cards) {
		t.Errorf("Cards = %d 件, want %d 件", len(m.cards), len(res.Cards))
	}
}
