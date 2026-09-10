package ui

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestExternalEditorWithoutCommand は editor 未設定でも起動せずエラーを返すことを検証する。
func TestExternalEditorWithoutCommand(t *testing.T) {
	msg, ok := ExternalEditor("  ")("Q1: A")().(editedMsg)
	if !ok {
		t.Fatalf("editedMsg が返らなかった")
	}
	if msg.err == nil || !strings.Contains(msg.err.Error(), "editor") {
		t.Errorf("err = %v, want editor を含むエラー", msg.err)
	}
}

// TestEditorCmdSplitsCommand は `code --wait` 形式の引数と編集対象の並びを検証する。
func TestEditorCmdSplitsCommand(t *testing.T) {
	got := editorCmd("code --wait", "/tmp/x.md").Args
	want := []string{"code", "--wait", "/tmp/x.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Args = %q, want %q", got, want)
	}
}

// TestWriteTemp は下書きを入れた .md の一時ファイルができることを検証する。
func TestWriteTemp(t *testing.T) {
	path, err := writeTemp("Q1: A")
	if err != nil {
		t.Fatalf("writeTemp: %v", err)
	}
	defer func() { _ = os.Remove(path) }()

	if !strings.HasSuffix(path, ".md") {
		t.Errorf("一時ファイル名が .md で終わらない: %q", path)
	}
	// 名前は `a` と `n` のどちらの下書きでも読める draft にする（エディタが人に見せる）。
	if !strings.HasPrefix(filepath.Base(path), "loop-cli-draft-") {
		t.Errorf("一時ファイル名が loop-cli-draft- で始まらない: %q", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(b) != "Q1: A" {
		t.Errorf("内容 = %q, want %q", b, "Q1: A")
	}
}

// TestReadEdited は編集結果の読み取りと、成否によらない後片付けを検証する。
func TestReadEdited(t *testing.T) {
	runErr := errors.New("exit status 1")
	for name, tc := range map[string]struct {
		runErr   error
		wantText string
		wantErr  error
	}{
		"編集された本文を読む":     {runErr: nil, wantText: "Q1: B"},
		"非 0 終了なら本文を捨てる": {runErr: runErr, wantErr: runErr},
	} {
		t.Run(name, func(t *testing.T) {
			path, err := writeTemp("Q1: B")
			if err != nil {
				t.Fatalf("writeTemp: %v", err)
			}

			msg := readEdited(path, tc.runErr)
			if msg.text != tc.wantText {
				t.Errorf("text = %q, want %q", msg.text, tc.wantText)
			}
			if !errors.Is(msg.err, tc.wantErr) {
				t.Errorf("err = %v, want %v", msg.err, tc.wantErr)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("一時ファイルが残っている: %q", path)
			}
		})
	}
}
