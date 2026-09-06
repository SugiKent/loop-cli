package onboarding

import (
	"errors"

	"charm.land/huh/v2"

	"github.com/SugiKent/loop-cli/internal/config"
)

// Run は初回起動のフォームを標準入出力で実行し、回答を path に書き出す。
// 利用者が中止したときは ErrAborted を返し、ファイルは書かない。
func Run(path string) error {
	var (
		reposText   string
		mergeMethod = config.MergeSquash
		notify      = true
		editor      = "$EDITOR"
	)

	form := huh.NewForm(huh.NewGroup(
		huh.NewText().
			Title("repos").
			Description("owner/name を 1 行に 1 つ").
			Placeholder("org/app\norg/web").
			Validate(func(s string) error {
				_, err := ParseRepos(s)
				return err
			}).
			Value(&reposText),
		huh.NewSelect[config.MergeMethod]().
			Title("merge_method").
			Options(
				huh.NewOption("squash", config.MergeSquash),
				huh.NewOption("merge", config.MergeMerge),
				huh.NewOption("rebase", config.MergeRebase),
			).
			Value(&mergeMethod),
		huh.NewConfirm().
			Title("notify").
			Value(&notify),
		huh.NewInput().
			Title("editor").
			Value(&editor),
	))

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return ErrAborted
		}
		return err
	}

	repos, err := ParseRepos(reposText)
	if err != nil {
		return err
	}
	return Write(path, Answers{Repos: repos, MergeMethod: mergeMethod, Notify: notify, Editor: editor})
}
