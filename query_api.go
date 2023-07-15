package main

// QueryRequest is what is sent to the server by GPT for a response, it is a list of queries
type QueryRequest struct {
	Queries []string `json:"queries" binding:"required"`
}

type QueryResponse struct {
	Results []QueryResult `json:"results"`
}

type QueryResult struct {
	Query  string   `json:"query"`
	Result []string `json:"result"`
}
