package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/nekomeowww/go-pinecone"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	"sync"
)

var sessions map[string]*session

// Make this read write mutex if performance is an issue
var sessionsMutex sync.Mutex

type session struct {
	user   User
	userMu sync.RWMutex

	index      *pinecone.IndexClient
	chatClient *openai.Client
	req        openai.ChatCompletionRequest
}

// GetSessionIfExists will return the session if it is in he sessions map, it is primarily used for the updated api when
// the session needs to be updated mid way.
func GetSessionIfExists(uid string) *session {
	sessionsMutex.Lock()
	s, ok := sessions[uid]
	sessionsMutex.Unlock()
	if ok {
		return s
	}
	return nil
}

func GetSession(user User) (*session, error) {
	sessionsMutex.Lock()
	s, ok := sessions[user.Uid]
	sessionsMutex.Unlock()
	if ok {
		return s, nil
	}
	s = &session{
		user: user,
	}
	pineconeIndex, _ := pinecone.NewIndexClient(
		pinecone.WithIndexName(user.PineconeIndex),
		pinecone.WithAPIKey(user.PineconeApiKey),
		pinecone.WithEnvironment(user.PineconeEnvironment),
		pinecone.WithProjectName(user.PineconeProjectName),
	)
	s.chatClient = openai.NewClient(user.OpenAIApiKey)
	s.index = pineconeIndex

	sessionsMutex.Lock()
	sessions[user.Uid] = s
	sessionsMutex.Unlock()

	if err := s.ValidateCredentials(); err != nil {
		return nil, err
	}

	s.req = openai.ChatCompletionRequest{
		Model:        openai.GPT3Dot5Turbo0613,
		Messages:     []openai.ChatCompletionMessage{},
		FunctionCall: function_call_defintions(),
	}
	return s, nil
}

// ValidateCredentials will check to see if the Pinecone credentials and the OpenAI credentials are invalid
func (s *session) ValidateCredentials() error {
	_, err := s.chatClient.ListModels(context.Background())
	if err != nil {
		log.Error().
			Err(err).
			Str("User", s.user.Uid).
			Msg("Invalid OpenAI Token")
		return errors.New("Invalid OpenAI API Key")
	}

	// validate credentials
	_, err = s.index.DescribeIndexStats(context.Background(), pinecone.DescribeIndexStatsParams{})
	if err != nil {
		s.userMu.RLock()
		log.Error().
			Err(err).
			Str("User", s.user.Uid).
			Msg("Invalaid Pinecone Credentials")
		s.userMu.RUnlock()
		return errors.New("Invalid Pinecone Credentials")
	}
	return nil
}

// Message will send a message to the chatbot with the context
func (s *session) Message(message string) (string, error) {
	s.req.Messages = append(s.req.Messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: message,
	})
	resp, err := s.chatClient.CreateChatCompletion(context.Background(), s.req)
	if err != nil {
		return "", err
	}
	call := resp.Choices[0].Message.FunctionCall
	if call != nil {
		switch call.Name {
		case "query_notes":
			// query our notes for information
			response := s.queryNotes(call.Arguments)
			s.req.Messages = append(s.req.Messages, resp.Choices[0].Message)
			s.req.Messages = append(s.req.Messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleFunction,
				Name:    QueryNotesName,
				Content: response,
			})
			resp, err = s.chatClient.CreateChatCompletion(context.Background(), s.req)
			if err != nil {
				return "", err
			}
		}
	}
	s.req.Messages = append(s.req.Messages, resp.Choices[0].Message)
	return resp.Choices[0].Message.Content, nil
}

// queryNotes will query embed the query and use the embedding to query pinecone and get the content and return it
func (s *session) queryNotes(query string) string {
	var request QueryRequest
	if err := json.Unmarshal([]byte(query), &request); err != nil {
		// we're going to keep this specific log because it's important for bug testing and logging in the future
		s.userMu.RLock()
		log.Error().
			Err(err).
			Str("User", s.user.Uid).
			Str("Content", query).
			Msg("Invalid json data trying to unmarshal in QueryRequest")
		s.userMu.RUnlock()
		return ""
	}

	queries := request.Queries

	s.userMu.RLock()
	embeddings, err := ada002Embeddings(s.chatClient, s.user.Uid, queries)
	s.userMu.RUnlock()

	if err != nil {
		return ""
	}

	// handle the response
	resp := QueryResponse{}
	for i, embedding := range embeddings {
		s.userMu.RLock()
		content := queryPinecone(s.index, s.user.TopK, embedding)
		s.userMu.RUnlock()
		resp.Results = append(resp.Results, QueryResult{
			Query:  queries[i],
			Result: content,
		})
	}
	data, _ := json.Marshal(resp)
	return string(data)
}
