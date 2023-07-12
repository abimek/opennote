package main

import "github.com/google/uuid"

type User struct {
	Uid                 string `json:"uid"`
	OpenAIApiKey        string `json:"open_ai_api_key"`
	PineconeApiKey      string `json:"pinecone_api_key"`
	PineconeIndex       string `json:"pinecone_index"`
	PineconeEnvironment string `json:"pinecone_environment"`
	PrinconeProjectName string `json:"pinecone_project_name"`
	TopK                int64  `json:"top_k"`
	Token               string `json:"token"`
}

func UserWithToken() (User, error) {
	token, err := uuid.NewRandom()
	if err != nil {
		return User{}, err
	}
	return User{
		Token: token.String(),
		TopK:  2,
	}, nil
}

// RandomToken assigns a new random token to the user
func (u *User) RandomToken() error {
	token, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	u.Token = token.String()
	return nil
}
