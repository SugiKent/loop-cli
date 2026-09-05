package ui

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Editor は下書きを人に編集させる手段。実機は $EDITOR を外部プロセスで開き、
// テストは固定の結果を返すスタブを渡す。返すコマンドは editedMsg を生む。
type Editor func(initial string) tea.Cmd

// editedMsg は編集の完了。err があれば投稿しない。
type editedMsg struct {
	text string
	err  error
}

// errNoEditor は config の editor も $EDITOR も空のとき。
var errNoEditor = errors.New("editor が設定されていません（config の editor か環境変数 EDITOR）")

// ExternalEditor は command を外部プロセスとして開く Editor を返す。
// command は空白で分割し、シェルを通さない（設定値をシェルに評価させない）。
func ExternalEditor(command string) Editor {
	return func(initial string) tea.Cmd {
		if strings.TrimSpace(command) == "" {
			return failedEdit(errNoEditor)
		}
		path, err := writeTemp(initial)
		if err != nil {
			return failedEdit(err)
		}
		return tea.ExecProcess(editorCmd(command, path), func(runErr error) tea.Msg {
			return readEdited(path, runErr)
		})
	}
}

// failedEdit は編集を始められなかったことを伝えるコマンド。
func failedEdit(err error) tea.Cmd {
	return func() tea.Msg { return editedMsg{err: err} }
}

// writeTemp は initial を書いた一時ファイルを作り、そのパスを返す。
func writeTemp(initial string) (string, error) {
	f, err := os.CreateTemp("", "sugi-loop-answer-*.md")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(initial); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// editorCmd は `code --wait` のような空白区切りの設定値と編集対象から *exec.Cmd を組む。
func editorCmd(command, path string) *exec.Cmd {
	fields := strings.Fields(command)
	return exec.Command(fields[0], append(fields[1:], path)...)
}

// readEdited はエディタ終了後の一時ファイルを読み、成否によらず削除する。
func readEdited(path string, runErr error) editedMsg {
	if runErr != nil {
		_ = os.Remove(path)
		return editedMsg{err: runErr}
	}
	b, err := os.ReadFile(path)
	_ = os.Remove(path)
	if err != nil {
		return editedMsg{err: err}
	}
	return editedMsg{text: string(b)}
}
