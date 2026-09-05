package gh

import "context"

// capturePRFields は ViewPR の 7 フィールドと ViewPRMergeState の 4 フィールドを合わせたもの。
// Fake は pr-<n>.json 1 ファイルを両メソッドで読むため、fixture はこの 11 フィールドで採る。
const capturePRFields = "number,title,body,url,labels,isDraft,comments," + prMergeStateFields

// Capture は fixture 用に読み取りコマンドの標準出力を採取する。
// キーは Fake が探すファイル名、値はデコードせずそのままの標準出力。
// progress は各ファイルの取得直前にファイル名を引数として 1 回呼ばれる。
func (c *Client) Capture(ctx context.Context, repo string, progress func(name string)) (map[string][]byte, error) {
	files := map[string][]byte{}

	capture := func(name string, args []string) ([]byte, error) {
		progress(name)
		out, err := c.run(ctx, "", args...)
		if err != nil {
			return nil, err
		}
		files[name] = out
		return out, nil
	}

	repos := []string{repo}

	out, err := capture(fixtureSearchIssues, argsSearchIssues(repos))
	if err != nil {
		return nil, err
	}
	issues, err := decodeSearchIssues(out)
	if err != nil {
		return nil, decodeErr(argsSearchIssues(repos), err)
	}

	for _, issue := range issues {
		n := issue.Number
		if _, err := capture(fixtureIssue(n), argsViewIssue(repo, n)); err != nil {
			return nil, err
		}
		if _, err := capture(fixtureIssueCrossRefs(n), argsCrossReferencedPRs(repo, n)); err != nil {
			return nil, err
		}
		if _, err := capture(fixtureIssueTimeline(n), argsLabelTimeline(repo, n)); err != nil {
			return nil, err
		}
	}

	out, err = capture(fixtureSearchPRs, argsSearchPRs(repos))
	if err != nil {
		return nil, err
	}
	prs, err := decodeSearchPRs(out)
	if err != nil {
		return nil, decodeErr(argsSearchPRs(repos), err)
	}

	for _, pr := range prs {
		n := pr.Number
		// pr view は UNKNOWN 再取得を通すので run を直接呼ばず prViewRaw を使う。
		progress(fixturePR(n))
		raw, err := c.prViewRaw(ctx, repo, n, capturePRFields)
		if err != nil {
			return nil, err
		}
		files[fixturePR(n)] = raw

		if _, err := capture(fixturePRReviewThreads(n), argsReviewThreads(repo, n)); err != nil {
			return nil, err
		}
	}

	return files, nil
}
