// Package onboarding は設定ファイルが無い初回起動で、フォームの回答を config.yml に書き出す。
package onboarding

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/SugiKent/sugi-loop/internal/config"
)

// ErrAborted は利用者がフォームを中止したこと。設定ファイルは書かれていない。
var ErrAborted = errors.New("設定の作成を中止しました")

// Answers はフォームの回答。refresh_interval_sec は聞かず常に 120 を書く。
type Answers struct {
	Repos       []string
	MergeMethod config.MergeMethod
	Notify      bool
	Editor      string
}

// file は書き出す YAML のキーと順。config.Config は repos がマッピングになるので使わない。
type file struct {
	Repos              []string           `yaml:"repos"`
	RefreshIntervalSec int                `yaml:"refresh_interval_sec"`
	MergeMethod        config.MergeMethod `yaml:"merge_method"`
	Editor             string             `yaml:"editor"`
	Notify             bool               `yaml:"notify"`
}

// ParseRepos は複数行入力を owner/name のリストにする。空行は捨てる。
func ParseRepos(text string) ([]string, error) {
	var repos []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !config.IsRepoName(line) {
			return nil, fmt.Errorf("%s: owner/name 形式ではありません", line)
		}
		repos = append(repos, line)
	}
	if len(repos) == 0 {
		return nil, errors.New("repos を 1 件以上入力してください")
	}
	return repos, nil
}

// Marshal は回答を mvp.md「設定ファイル」の例と同じ形式の YAML にする。
func Marshal(a Answers) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(file{
		Repos:              a.Repos,
		RefreshIntervalSec: 120,
		MergeMethod:        a.MergeMethod,
		Editor:             a.Editor,
		Notify:             a.Notify,
	}); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write は Marshal の結果を path に書く。親ディレクトリが無ければ作る。
func Write(path string, a Answers) error {
	data, err := Marshal(a)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
