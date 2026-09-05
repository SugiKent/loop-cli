package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/sugi-loop/internal/fetch"
	"github.com/SugiKent/sugi-loop/internal/gh"
	"github.com/SugiKent/sugi-loop/internal/model"
)

// at は画面のテストで使う取得完了時刻（mvp.md の画面例の 12:04）。
var at = time.Date(2026, 9, 5, 12, 4, 0, 0, time.FixedZone("JST", 9*3600))

// fixtureDir は example fixture の場所。
const fixtureDir = "../gh/testdata/fixtures/example"

// newModel は回答を使わない画面のテスト用の Model。
// エディタは押されない前提だが、nil で落ちないようにエラーを返すスタブを入れる。
func newModel(fetcher Fetcher) Model {
	return newModelOpts(fetcher, Options{})
}

// newModelOpts は起動時の選択肢を変えた newModel。
func newModelOpts(fetcher Fetcher, opts Options) Model {
	return New(fetcher, gh.NewFake(fixtureDir), (&stubEditor{msg: editedMsg{err: errors.New("使わない")}}).Editor, opts)
}

// stubEditor はエディタを起動せず固定の結果を返す Editor。渡された下書きを記録する。
type stubEditor struct {
	initial string
	calls   int
	msg     editedMsg
}

func (s *stubEditor) Editor(initial string) tea.Cmd {
	s.initial = initial
	s.calls++
	msg := s.msg
	return func() tea.Msg { return msg }
}

// exampleResult は example fixture から s07 の Fetch で作った Result。
func exampleResult(t *testing.T) *fetch.Result {
	t.Helper()
	res, err := fetch.Fetch(context.Background(), gh.NewFake("../gh/testdata/fixtures/example"), []string{"org/app"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	return res
}

// cardOf は example の Card から issue 番号で 1 枚選ぶ。
func cardOf(t *testing.T, res *fetch.Result, number int) model.Card {
	t.Helper()
	for _, c := range res.Cards {
		if c.Issue != nil && c.Issue.Number == number {
			return c
		}
	}
	t.Fatalf("issue %d の Card が無い", number)
	return model.Card{}
}

// nowCard は今やるタブに入る手書きの Card を 1 枚作る。
func nowCard(repo string, number int, priority int, updatedAt time.Time) model.Card {
	result := model.Result{Situation: model.SituationA, Priority: priority, Tab: model.TabNow, Summary: "回答する"}
	issue := &model.Issue{Repo: repo, Number: number, Title: "手書き", UpdatedAt: updatedAt, Result: result}
	return model.Card{Issue: issue, Result: result}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

// confirmModel は s10 の手順（blocked-by 行のある下書き）で確認画面に移った Model と投稿先の Fake を返す。
func confirmModel(t *testing.T) (Model, *gh.Fake) {
	t.Helper()
	m, fake := answerModel([]model.Card{prCard(nil)}, &stubEditor{msg: editedMsg{text: "Q1: A\n  blocked-by: human"}})
	m, _ = answer(t, m)
	if m.screen != screenConfirm {
		t.Fatalf("確認画面に移っていない: screen = %d", m.screen)
	}
	return m, fake
}
