package gh

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func decodeSearchIssues(b []byte) ([]SearchIssue, error) {
	var v []SearchIssue
	return v, json.Unmarshal(b, &v)
}

func decodeSearchPRs(b []byte) ([]SearchPR, error) {
	var v []SearchPR
	return v, json.Unmarshal(b, &v)
}

func decodeRepoLabels(b []byte) ([]RepoLabel, error) {
	var v []RepoLabel
	return v, json.Unmarshal(b, &v)
}

func decodeIssueDetail(b []byte) (*IssueDetail, error) {
	var v IssueDetail
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func decodePRDetail(b []byte) (*PRDetail, error) {
	var v PRDetail
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func decodePRMergeState(b []byte) (*PRMergeState, error) {
	var v PRMergeState
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// reviewThreadsResponse は GraphQL の reviewThreads 応答の入れ子。平坦化のためだけに使う。
type reviewThreadsResponse struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				ReviewThreads struct {
					Nodes []struct {
						ID         string `json:"id"`
						IsResolved bool   `json:"isResolved"`
						Comments   struct {
							Nodes []ReviewComment `json:"nodes"`
						} `json:"comments"`
					} `json:"nodes"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
}

func decodeReviewThreads(b []byte) ([]ReviewThread, error) {
	var resp reviewThreadsResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	nodes := resp.Data.Repository.PullRequest.ReviewThreads.Nodes
	threads := make([]ReviewThread, 0, len(nodes))
	for _, n := range nodes {
		threads = append(threads, ReviewThread{ID: n.ID, IsResolved: n.IsResolved, Comments: n.Comments.Nodes})
	}
	return threads, nil
}

// crossRefsResponse は GraphQL の timelineItems 応答の入れ子。
// PR 以外からの参照は source が空オブジェクトになり、Number が 0 のままになる。
type crossRefsResponse struct {
	Data struct {
		Repository struct {
			Issue struct {
				TimelineItems struct {
					Nodes []struct {
						Source struct {
							Number int    `json:"number"`
							Title  string `json:"title"`
							Body   string `json:"body"`
							State  string `json:"state"`
							Labels struct {
								Nodes []Label `json:"nodes"`
							} `json:"labels"`
						} `json:"source"`
					} `json:"nodes"`
				} `json:"timelineItems"`
			} `json:"issue"`
		} `json:"repository"`
	} `json:"data"`
}

func decodeCrossReferencedPRs(b []byte) ([]CrossReferencedPR, error) {
	var resp crossRefsResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	nodes := resp.Data.Repository.Issue.TimelineItems.Nodes
	prs := make([]CrossReferencedPR, 0, len(nodes))
	for _, n := range nodes {
		if n.Source.Number == 0 {
			continue
		}
		labels := make([]string, 0, len(n.Source.Labels.Nodes))
		for _, l := range n.Source.Labels.Nodes {
			labels = append(labels, l.Name)
		}
		prs = append(prs, CrossReferencedPR{
			Number: n.Source.Number,
			Title:  n.Source.Title,
			Body:   n.Source.Body,
			State:  n.Source.State,
			Labels: labels,
		})
	}
	return prs, nil
}

// decodeLabelEvents は --jq が出す JSON オブジェクトの連続を EOF まで読む。
func decodeLabelEvents(b []byte) ([]LabelEvent, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	events := []LabelEvent{}
	for {
		var e LabelEvent
		err := dec.Decode(&e)
		if errors.Is(err, io.EOF) {
			return events, nil
		}
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
}
