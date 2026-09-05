package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SugiKent/sugi-loop/internal/fetch"
	"github.com/SugiKent/sugi-loop/internal/gh"
	"github.com/SugiKent/sugi-loop/internal/model"
)

// at は画面のテストで使う取得完了時刻（mvp.md の画面例の 12:04）。
var at = time.Date(2026, 9, 5, 12, 4, 0, 0, time.FixedZone("JST", 9*3600))

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
