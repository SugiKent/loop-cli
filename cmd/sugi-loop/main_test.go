package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func keyPress(k tea.Key) tea.KeyPressMsg {
	return tea.KeyPressMsg(k)
}

func TestViewShowsAppNameAndQuitHint(t *testing.T) {
	content := model{}.View().Content

	if !strings.Contains(content, "sugi-loop") {
		t.Errorf("初期フレームにアプリ名 sugi-loop が無い: %q", content)
	}
	if !strings.Contains(content, "q") {
		t.Errorf("初期フレームに q での終了案内が無い: %q", content)
	}
}

func TestQuitKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tea.Key
	}{
		{"q", tea.Key{Code: 'q', Text: "q"}},
		{"ctrl+c", tea.Key{Code: 'c', Mod: tea.ModCtrl}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key.String(); got != tt.name {
				t.Fatalf("キーの文字列表現が想定と違う: got %q, want %q", got, tt.name)
			}

			_, cmd := model{}.Update(keyPress(tt.key))
			if cmd == nil {
				t.Fatalf("%s で終了コマンドが返らなかった", tt.name)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Errorf("%s で返ったコマンドが終了メッセージを生まない: %T", tt.name, cmd())
			}
		})
	}
}

func TestOtherKeyDoesNotQuit(t *testing.T) {
	m, cmd := model{}.Update(keyPress(tea.Key{Code: 'j', Text: "j"}))

	if cmd != nil {
		t.Errorf("j で nil 以外のコマンドが返った: %T", cmd())
	}
	if !strings.Contains(m.View().Content, "sugi-loop") {
		t.Error("j の後にモデルが描画可能でなくなった")
	}
}
