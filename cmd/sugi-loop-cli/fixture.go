package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/SugiKent/sugi-loop/internal/gh"
)

// fixturesDir は fixture の保存先の親。カレントがリポジトリのルートである確認にも使う。
const fixturesDir = "internal/gh/testdata/fixtures"

var aliasRe = regexp.MustCompile(`^[a-z0-9-]+$`)

func fixtureCapture(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("fixture capture", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", "", "採取するリポジトリ（owner/name）")
	alias := fs.String("alias", "", "保存先ディレクトリ名（^[a-z0-9-]+$）")
	if err := fs.Parse(args); err != nil {
		return err
	}

	owner, name, err := validateRepo(*repo)
	if err != nil {
		return err
	}
	if err := validateAlias(*alias, owner, name); err != nil {
		return err
	}
	if _, err := os.Stat(fixturesDir); err != nil {
		return fmt.Errorf("%s がありません。リポジトリのルートで実行してください: %w", fixturesDir, err)
	}

	ctx := context.Background()
	client := gh.NewClient()
	if err := client.Check(ctx); err != nil {
		return err
	}
	files, err := client.Capture(ctx, *repo, func(n string) { _, _ = fmt.Fprintln(stderr, n) })
	if err != nil {
		return err
	}
	redacted, logins, err := redact(files, owner, name, *alias)
	if err != nil {
		return err
	}

	dir := filepath.Join(fixturesDir, *alias)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for n, b := range redacted {
		if err := os.WriteFile(filepath.Join(dir, n), b, 0o644); err != nil {
			return err
		}
	}

	issues, prs, err := verifyFixtures(ctx, dir)
	if err != nil {
		return errors.Join(err, os.RemoveAll(dir))
	}

	_, _ = fmt.Fprintf(stdout, "%s に %d ファイルを書きました（issue %d / PR %d / 伏せた login %d）\n",
		dir, len(redacted), issues, prs, logins)
	return nil
}

func validateRepo(repo string) (owner, name string, err error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || name == "" {
		return "", "", fmt.Errorf("--repo は owner/name 形式で指定してください: %q", repo)
	}
	return owner, name, nil
}

func validateAlias(alias, owner, name string) error {
	if !aliasRe.MatchString(alias) {
		return fmt.Errorf("--alias は ^[a-z0-9-]+$ に一致する必要があります: %q", alias)
	}
	if alias == "example" {
		return errors.New("--alias に example は使えません（s03 の手書き fixture を上書きするため）")
	}
	lower := strings.ToLower(alias)
	for _, part := range []string{owner, name} {
		if strings.Contains(lower, strings.ToLower(part)) {
			return fmt.Errorf("--alias に %q を含められません（伏せ字後も元の名前が fixture に残るため）", part)
		}
	}
	return nil
}

// verifyFixtures は書き込んだ fixture を Fake で読み直し、issue 数と PR 数を返す。
func verifyFixtures(ctx context.Context, dir string) (issues, prs int, err error) {
	f := gh.NewFake(dir)
	is, err := f.SearchIssues(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("自己検査に失敗しました: %w", err)
	}
	ps, err := f.SearchPRs(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("自己検査に失敗しました: %w", err)
	}
	for _, i := range is {
		if _, err := f.ViewIssue(ctx, "", i.Number); err != nil {
			return 0, 0, fmt.Errorf("自己検査に失敗しました: %w", err)
		}
		if _, err := f.CrossReferencedPRs(ctx, "", i.Number); err != nil {
			return 0, 0, fmt.Errorf("自己検査に失敗しました: %w", err)
		}
		if _, err := f.LabelTimeline(ctx, "", i.Number); err != nil {
			return 0, 0, fmt.Errorf("自己検査に失敗しました: %w", err)
		}
	}
	for _, p := range ps {
		if _, err := f.ViewPR(ctx, "", p.Number); err != nil {
			return 0, 0, fmt.Errorf("自己検査に失敗しました: %w", err)
		}
		if _, err := f.ViewPRMergeState(ctx, "", p.Number); err != nil {
			return 0, 0, fmt.Errorf("自己検査に失敗しました: %w", err)
		}
		if _, err := f.ReviewThreads(ctx, "", p.Number); err != nil {
			return 0, 0, fmt.Errorf("自己検査に失敗しました: %w", err)
		}
	}
	return len(is), len(ps), nil
}
