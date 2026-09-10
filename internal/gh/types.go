package gh

import "time"

// Label は gh の labels[] の要素。id / color / description は読み捨てる。
type Label struct {
	Name string `json:"name"`
}

// RepoLabel は gh label list の 1 件。リポジトリで使えるラベルの定義を表す。
// issue / PR に付いているラベル（Label）とは取得元も用途も違うので型を分ける。
type RepoLabel struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// Repository は gh search の repository フィールド。
type Repository struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
}

// Author はコメントの投稿者。
type Author struct {
	Login string `json:"login"`
}

// SearchIssue は gh search issues の 1 件。
type SearchIssue struct {
	Repository    Repository `json:"repository"`
	Number        int        `json:"number"`
	Title         string     `json:"title"`
	Labels        []Label    `json:"labels"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	URL           string     `json:"url"`
	Body          string     `json:"body"`
	CommentsCount int        `json:"commentsCount"`
}

// SearchPR は gh search prs の 1 件。
type SearchPR struct {
	Repository Repository `json:"repository"`
	Number     int        `json:"number"`
	Title      string     `json:"title"`
	Labels     []Label    `json:"labels"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	URL        string     `json:"url"`
	Body       string     `json:"body"`
	IsDraft    bool       `json:"isDraft"`
}

// Comment は issue / PR のコメント。
type Comment struct {
	ID        string    `json:"id"`
	Author    Author    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	URL       string    `json:"url"`
}

// IssueDetail は gh issue view の出力。
type IssueDetail struct {
	Number   int       `json:"number"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	URL      string    `json:"url"`
	Labels   []Label   `json:"labels"`
	Comments []Comment `json:"comments"`
}

// PRDetail は gh pr view のコメント・ラベル。merge 関連は PRMergeState が持つ。
type PRDetail struct {
	Number   int       `json:"number"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	URL      string    `json:"url"`
	Labels   []Label   `json:"labels"`
	IsDraft  bool      `json:"isDraft"`
	Comments []Comment `json:"comments"`
}

// PRMergeState は merge ガードに使う PR の状態。
type PRMergeState struct {
	Mergeable         string        `json:"mergeable"`
	MergeStateStatus  string        `json:"mergeStateStatus"`
	ReviewDecision    string        `json:"reviewDecision"`
	StatusCheckRollup []StatusCheck `json:"statusCheckRollup"`
}

// StatusCheck は statusCheckRollup の要素。CheckRun と StatusContext を Typename で判別する。
type StatusCheck struct {
	Typename     string `json:"__typename"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	Context      string `json:"context"`
	State        string `json:"state"`
	WorkflowName string `json:"workflowName"`
	DetailsURL   string `json:"detailsUrl"`
}

// ReviewThread は PR の review thread（GraphQL 応答を平坦化したもの）。
type ReviewThread struct {
	ID         string
	IsResolved bool
	Comments   []ReviewComment
}

// ReviewComment は review thread 内のコメント。DatabaseID は REST の comment id。
type ReviewComment struct {
	DatabaseID int64     `json:"databaseId"`
	Author     Author    `json:"author"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"createdAt"`
}

// CrossReferencedPR は issue を参照した PR（GraphQL 応答を平坦化したもの）。
type CrossReferencedPR struct {
	Number int
	Title  string
	Body   string
	State  string
	Labels []string
}

// LabelEvent は issue timeline のラベル付け外しイベント。
type LabelEvent struct {
	CreatedAt time.Time `json:"created_at"`
	Event     string    `json:"event"`
	Label     string    `json:"label"`
}
