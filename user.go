package main

type User struct {
	Uid                 string `json:"Uid"`
	OpenAIApiKey        string `json:"OpenAIApiKey"`
	PineconeApiKey      string `json:"PineconeApiKey"`
	PineconeIndex       string `json:"PineconeIndex"`
	PineconeEnvironment string `json:"PineconeEnvironment"`
	PineconeProjectName string `json:"PineconeProjectName"`
	TopK                int64  `json:"TopK"`
}
