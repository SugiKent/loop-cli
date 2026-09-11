// Package config は loop-cli の設定ファイル（~/.config/loop-cli/config.yml）を読み込む。
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// MergeMethod は gh pr merge に渡す merge 方式。値は gh のフラグ名と一致する。
type MergeMethod string

// サポートする merge 方式。
const (
	MergeSquash MergeMethod = "squash"
	MergeMerge  MergeMethod = "merge"
	MergeRebase MergeMethod = "rebase"
)

// Repo は監視対象リポジトリ。MergeMethod は Load が必ず埋める。
// ClaudeConfigDir はそのリポジトリの Routine を回している Claude のプロファイルのパス
// （CLAUDE_CONFIG_DIR に渡す値）。省略できて、既定値は持たない。
type Repo struct {
	Name            string
	MergeMethod     MergeMethod
	ClaudeConfigDir string
}

// Config は設定ファイルの内容。
type Config struct {
	Repos              []Repo      `yaml:"repos"`
	RefreshIntervalSec int         `yaml:"refresh_interval_sec"`
	MergeMethod        MergeMethod `yaml:"merge_method"`
	Editor             string      `yaml:"editor"`
	Notify             bool        `yaml:"notify"`
}

// DefaultPath は設定ファイルの既定パス $HOME/.config/loop-cli/config.yml を返す。
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "loop-cli", "config.yml"), nil
}

// UnmarshalYAML は repos の要素を「文字列」または
// 「{name, merge_method, claude_config_dir} のマッピング」として読む。
func (r *Repo) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		r.Name = node.Value
		return nil
	case yaml.MappingNode:
		// 未知フィールド拒否はノード単位のデコードに引き継がれないので自前でキーを検査する。
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			value := node.Content[i+1].Value
			switch key {
			case "name":
				r.Name = value
			case "merge_method":
				r.MergeMethod = MergeMethod(value)
			case "claude_config_dir":
				r.ClaudeConfigDir = value
			case "mode":
				// s30 で廃止。値が正しくても止める（汎用の未知キーの文言では理由が読み取れない）。
				return errors.New("repos の mode は廃止しました。運用方式はリポジトリのラベル一覧から判定するので、この行を削除してください")
			default:
				return fmt.Errorf("repos の要素に未知のキー %q があります", key)
			}
		}
		if r.Name == "" {
			return errors.New("repos の要素に name がありません")
		}
		return nil
	default:
		return errors.New("repos の要素は文字列か {name, merge_method, claude_config_dir} のマッピングで書いてください")
	}
}

// Load は path の設定ファイルを読み込み、既定値の適用と検証を済ませた Config を返す。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}

	cfg := Config{
		RefreshIntervalSec: 120,
		MergeMethod:        MergeSquash,
		Editor:             "$EDITOR",
		Notify:             true,
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	// 空ファイルは io.EOF になる。何も書かれていないだけなので既定値のまま検証へ進める。
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}

	cfg.Editor = os.ExpandEnv(cfg.Editor)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}

	for i := range cfg.Repos {
		if cfg.Repos[i].MergeMethod == "" {
			cfg.Repos[i].MergeMethod = cfg.MergeMethod
		}
		cfg.Repos[i].ClaudeConfigDir = expandPath(cfg.Repos[i].ClaudeConfigDir)
	}
	return &cfg, nil
}

// expandPath は環境変数と先頭の ~ を展開する。パスが存在するかは確かめない
// （設定ファイルを別のマシンから持ってきたときに起動できなくなるため）。
// ~alice のような他人のホームの書き方は展開しない。
func expandPath(path string) string {
	path = os.ExpandEnv(path)
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}

func validate(cfg *Config) error {
	if len(cfg.Repos) == 0 {
		return errors.New("repos が空です")
	}
	for i, repo := range cfg.Repos {
		if !IsRepoName(repo.Name) {
			return fmt.Errorf("repos[%d] %q: owner/name 形式ではありません", i, repo.Name)
		}
	}
	if !isMergeMethod(cfg.MergeMethod) {
		return fmt.Errorf("merge_method %q: squash / merge / rebase のいずれかを指定してください", cfg.MergeMethod)
	}
	for i, repo := range cfg.Repos {
		if repo.MergeMethod != "" && !isMergeMethod(repo.MergeMethod) {
			return fmt.Errorf("repos[%d] %q の merge_method %q: squash / merge / rebase のいずれかを指定してください", i, repo.Name, repo.MergeMethod)
		}
	}
	if cfg.RefreshIntervalSec < 1 {
		return errors.New("refresh_interval_sec は 1 以上を指定してください")
	}
	return nil
}

// IsRepoName は owner/name 形式かを返す。Load の検証と onboarding のフォームが同じ規則を使う。
func IsRepoName(name string) bool {
	owner, repo, found := strings.Cut(name, "/")
	return found && owner != "" && repo != "" && !strings.Contains(repo, "/") && !strings.ContainsAny(name, " \t")
}

func isMergeMethod(m MergeMethod) bool {
	return m == MergeSquash || m == MergeMerge || m == MergeRebase
}
