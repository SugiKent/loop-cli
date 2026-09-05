// Command sugi-loop は今やるキュー画面を出す TUI。
package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/SugiKent/sugi-loop/internal/config"
	"github.com/SugiKent/sugi-loop/internal/fetch"
	"github.com/SugiKent/sugi-loop/internal/gh"
	"github.com/SugiKent/sugi-loop/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	client := gh.NewClient()
	if err := client.Check(context.Background()); err != nil {
		return err
	}

	repos := make([]string, len(cfg.Repos))
	for i, r := range cfg.Repos {
		repos[i] = r.Name
	}

	// editor が空でも起動は失敗させない（a を押したときにフッタにエラーが出る）。
	m := ui.New(func(ctx context.Context) (*fetch.Result, error) {
		return fetch.Fetch(ctx, client, repos)
	}, client, ui.ExternalEditor(cfg.Editor))
	_, err = tea.NewProgram(m).Run()
	return err
}
