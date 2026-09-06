package snapshot

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
)

// savedAt は往復のテストで使う保存時刻。JSON から戻した time.Time の場所情報が
// 一致するよう UTC で与える。
var savedAt = time.Date(2026, 9, 5, 3, 4, 0, 0, time.UTC)

// exampleCards は example fixture から s07 の Fetch で作った Card 群。
func exampleCards(t *testing.T) []model.Card {
	t.Helper()
	res, err := fetch.Fetch(context.Background(), gh.NewFake("../gh/testdata/fixtures/example"), []string{"org/app"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	return res.Cards
}

// cardOf は Card 群から issue 番号で 1 枚選ぶ。
func cardOf(t *testing.T, cards []model.Card, number int) model.Card {
	t.Helper()
	for _, c := range cards {
		if c.Issue != nil && c.Issue.Number == number {
			return c
		}
	}
	t.Fatalf("issue %d の Card が無い", number)
	return model.Card{}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("HOME", "/tmp/h")

	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if want := "/tmp/h/.cache/loop-cli/snapshot.json"; got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	cards := exampleCards(t)
	before := Snapshot{Cards: cards, At: savedAt}
	path := filepath.Join(t.TempDir(), "loop-cli", "snapshot.json")

	if err := Save(path, before); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, before) {
		t.Errorf("往復で変わった:\n got %+v\nwant %+v", got, before)
	}

	// s20 で全 issue のコメントを取るので、issue 140 は長さ 0 の非 nil で往復する。
	if c := cardOf(t, got.Cards, 140).Issue.Comments; c == nil || len(c) != 0 {
		t.Errorf("issue 140 の Comments = %v, want 長さ 0 の非 nil", c)
	}
	wantThreads := cardOf(t, cards, 108).PRs[0].ReviewThreads
	gotThreads := cardOf(t, got.Cards, 108).PRs[0].ReviewThreads
	if (wantThreads == nil) != (gotThreads == nil) {
		t.Errorf("issue 108 の ReviewThreads の nil が変わった: got %v, want %v", gotThreads, wantThreads)
	}
}

func TestSaveLoadKeepsNilAndEmptySlices(t *testing.T) {
	nilCard := model.Card{Issue: &model.Issue{Repo: "org/app", Number: 1}}
	emptyCard := model.Card{Issue: &model.Issue{Repo: "org/app", Number: 2, Comments: []model.Comment{}}}
	path := filepath.Join(t.TempDir(), "snapshot.json")

	if err := Save(path, Snapshot{Cards: []model.Card{nilCard, emptyCard}, At: savedAt}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.Cards[0].Issue.Comments != nil {
		t.Errorf("1 枚目の Comments = %v, want nil", got.Cards[0].Issue.Comments)
	}
	if c := got.Cards[1].Issue.Comments; c == nil || len(c) != 0 {
		t.Errorf("2 枚目の Comments = %v, want 長さ 0 の非 nil", c)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "no-such.json")); err == nil {
		t.Fatal("存在しないパスの Load がエラーにならない")
	}
}

func TestLoadBrokenJSONKeepsFile(t *testing.T) {
	const body = `{"cards": [`
	path := filepath.Join(t.TempDir(), "snapshot.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("壊れた JSON の Load がエラーにならない")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != body {
		t.Errorf("ファイルが変わった: %q", got)
	}
}
