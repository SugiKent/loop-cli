package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/loop-cli/internal/fetch"
	"github.com/SugiKent/loop-cli/internal/gh"
	"github.com/SugiKent/loop-cli/internal/model"
	"github.com/SugiKent/loop-cli/internal/snapshot"
)

// boardModes は org/board だけを label 方式として扱う対応表。
var boardModes = map[string]model.Mode{"org/board": model.ModeLabel}

// boardCard は org/board の issue 1 件の Card。
func boardCard(number int, labels ...string) model.Card {
	result := model.Result{Situation: model.SituationE, Priority: 4, Tab: model.TabBacklog, Summary: "待つ"}
	issue := &model.Issue{
		Repo: "org/board", Number: number, Title: fmt.Sprintf("fixture %d", number),
		Labels: labels, UpdatedAt: at, Result: result,
	}
	return model.Card{Issue: issue, Result: result}
}

// boardModel は org/board の Card を持ち、書き込み先が action の todo fixture である Model。
func boardModel(cards []model.Card, opts Options) (Model, *gh.Fake) {
	fake := gh.NewFake("../action/testdata/todo")
	opts.Modes = boardModes
	// Card は Result.Tab がバックログなので、そのタブへ移ってから使う。
	m, _ := send(New(nil, fake, (&stubEditor{}).Editor, opts),
		tea.WindowSizeMsg{Width: 120, Height: 40},
		fetchedMsg{res: &fetch.Result{Cards: cards}, at: at},
		runeKey('2'))
	return m, fake
}

// TestTodoLabelModeWritesToDo は label 方式のリポジトリで t が To Do を書くことを検証する。
func TestTodoLabelModeWritesToDo(t *testing.T) {
	m, fake := boardModel([]model.Card{boardCard(160)}, Options{})

	m, cmd := send(m, tKey)
	if cmd == nil {
		t.Fatal("t でコマンドが返っていない")
	}
	if text := plainText(m); !strings.Contains(text, "org/board #160 の To Do を切り替え中") {
		t.Errorf("切り替え中の文言が To Do でない: %q", footerOf(text))
	}
	m, _ = send(m, cmd())

	want := []gh.Call{
		{Method: "ViewIssue", Repo: "org/board", Number: 160},
		{Method: "AddLabel", Repo: "org/board", Number: 160, Label: "To Do"},
	}
	if len(fake.Calls) != len(want) {
		t.Fatalf("呼び出し = %+v, want %+v", fake.Calls, want)
	}
	for i, w := range want {
		if fake.Calls[i] != w {
			t.Errorf("%d 件目 = %+v, want %+v", i+1, fake.Calls[i], w)
		}
	}
	if text := plainText(m); !strings.Contains(text, "org/board #160 に To Do を付けました") {
		t.Errorf("結果の文言が To Do でない: %q", footerOf(text))
	}
}

// TestTodoLabelModeUsesConfigOnSnapshot は、初回取得の前（スナップショット表示中）でも
// 設定の方式でラベルを書くことを検証する。
func TestTodoLabelModeUsesConfigOnSnapshot(t *testing.T) {
	fake := gh.NewFake("../action/testdata/todo")
	snap := &snapshot.Snapshot{Cards: []model.Card{boardCard(160)}, At: at}
	m := New(nil, fake, (&stubEditor{}).Editor, Options{Modes: boardModes, Snapshot: snap})
	m, _ = send(m, tea.WindowSizeMsg{Width: 120, Height: 40}, runeKey('2'))

	_, cmd := send(m, tKey)
	if cmd == nil {
		t.Fatal("t でコマンドが返っていない")
	}
	send(m, cmd())

	if len(fake.Calls) != 2 || fake.Calls[1].Label != "To Do" {
		t.Errorf("呼び出し = %+v, want 2 件目が To Do", fake.Calls)
	}
}

// TestDetailLabelModeStageLine は label 方式の段階行とバッジを検証する。
func TestDetailLabelModeStageLine(t *testing.T) {
	t.Run("In Progress と question", func(t *testing.T) {
		m, _ := boardModel([]model.Card{boardCard(160, model.LabelInProgress, model.LabelQuestion)}, Options{})
		m, _ = send(m, enterKey)

		text := plainText(m)
		if !strings.Contains(text, "段階: In Progress") || !strings.Contains(text, "[question]") {
			t.Errorf("段階行が label の語彙でない:\n%s", text)
		}
		if strings.Contains(text, "[wip]") {
			t.Errorf("label 方式で [wip] を出している:\n%s", text)
		}
	})

	t.Run("sdd の段階ラベルは段階行に出さない", func(t *testing.T) {
		m, _ := boardModel([]model.Card{boardCard(160, model.LabelStagePropose)}, Options{})
		m, _ = send(m, enterKey)

		if text := plainText(m); !strings.Contains(text, "段階なし") {
			t.Errorf("段階なし が出ていない:\n%s", text)
		}
	})

	t.Run("表に無いリポジトリは sdd の語彙", func(t *testing.T) {
		card := boardCard(160, model.LabelStagePropose)
		card.Issue.Repo = "org/app"
		m, _ := boardModel([]model.Card{card}, Options{})
		m, _ = send(m, enterKey)

		if text := plainText(m); !strings.Contains(text, "段階: stage:propose") {
			t.Errorf("sdd の語彙で出ていない:\n%s", text)
		}
	})
}

// TestDetailLabelModePRList は label 方式の PR 一覧が段階の見出しを持たないことを検証する。
func TestDetailLabelModePRList(t *testing.T) {
	t.Run("段階の見出しを持たない", func(t *testing.T) {
		card := boardCard(160)
		card.PRs = []model.PR{
			{Repo: "org/board", Number: 61, State: "OPEN", Body: "Closes #160"},
			{Repo: "org/board", Number: 62, State: "OPEN", Labels: []string{model.LabelQuestion}},
		}
		m, _ := boardModel([]model.Card{card}, Options{})
		m, _ = send(m, enterKey)

		text := plainText(m)
		line61, ok61 := lineWith(plain(m), "[-] PR#61 open")
		_, ok62 := lineWith(plain(m), "[-] PR#62 open")
		if !ok61 || !ok62 {
			t.Fatalf("PR 行が出ていない:\n%s", text)
		}
		if strings.Index(text, "PR#61") >= strings.Index(text, "PR#62") {
			t.Errorf("PR 行が並び順に出ていない:\n%s", text)
		}
		if !strings.HasPrefix(line61, "▶") {
			t.Errorf("選択中の PR に ▶ が無い: %q", line61)
		}
		for _, ng := range []string{"[propose] なし", "[apply] なし", "[archive] なし"} {
			if strings.Contains(text, ng) {
				t.Errorf("label 方式で %q を出している", ng)
			}
		}
	})

	t.Run("PR が無ければ PR の行を出さない", func(t *testing.T) {
		m, _ := boardModel([]model.Card{boardCard(160)}, Options{})
		m, _ = send(m, enterKey)

		text := plainText(m)
		if strings.Contains(text, "[-] PR#") || strings.Contains(text, "[propose] なし") {
			t.Errorf("PR の行が出ている:\n%s", text)
		}
	})
}

// TestMissingLabelModeNotice は方式の書き忘れをフッタで知らせることを検証する。
func TestMissingLabelModeNotice(t *testing.T) {
	const notice = "org/board に To Do / In Progress の issue があります。mode: label の設定漏れかもしれません"
	cards := []model.Card{boardCard(160, model.LabelToDo)}

	t.Run("sdd 扱いなら知らせる", func(t *testing.T) {
		m, _ := send(newModelOpts(nil, Options{}),
			tea.WindowSizeMsg{Width: 120, Height: 40},
			fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})
		if text := plainText(m); !strings.Contains(text, notice) {
			t.Errorf("設定漏れの知らせが無い: %q", footerOf(text))
		}
	})

	t.Run("label 扱いなら知らせない", func(t *testing.T) {
		m, _ := send(newModelOpts(nil, Options{Modes: boardModes}),
			tea.WindowSizeMsg{Width: 120, Height: 40},
			fetchedMsg{res: &fetch.Result{Cards: cards}, at: at})
		if text := plainText(m); strings.Contains(text, "設定漏れ") {
			t.Errorf("label 扱いなのに知らせている: %q", footerOf(text))
		}
	})

	t.Run("部分失敗が優先される", func(t *testing.T) {
		res := &fetch.Result{Cards: cards, Errors: []error{errors.New("ViewPR org/board#61: open pr-61.json: no such file")}}
		m, _ := send(newModelOpts(nil, Options{}),
			tea.WindowSizeMsg{Width: 120, Height: 40},
			fetchedMsg{res: res, at: at})
		text := plainText(m)
		if !strings.Contains(text, "詳細取得の失敗 1 件") {
			t.Errorf("部分失敗が出ていない: %q", footerOf(text))
		}
		if strings.Contains(text, "設定漏れ") {
			t.Errorf("部分失敗より設定漏れを優先している: %q", footerOf(text))
		}
	})
}

// TestHelpMentionsBothTodoLabels はヘルプの t の説明が 2 方式ぶんであることを検証する。
func TestHelpMentionsBothTodoLabels(t *testing.T) {
	m, _ := send(newModel(nil), tea.WindowSizeMsg{Width: 120, Height: 40}, runeKey('?'))
	if text := plainText(m); !strings.Contains(text, "stage:todo / To Do を付ける / 外す") {
		t.Errorf("ヘルプの t の説明が 2 方式ぶんでない:\n%s", text)
	}
}
