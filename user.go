package main

type User struct {
	Uid                 string `json:"uid"`
	OpenAIApiKey        string `json:"open_ai_api_key"`
	PineconeApiKey      string `json:"pinecone_api_key"`
	PineconeIndex       string `json:"pinecone_index"`
	PineconeEnvironment string `json:"pinecone_environment"`
	PrinconeProjectName string `json:"pinecone_project_name"`
	TopK                int64  `json:"top_k"`
}
