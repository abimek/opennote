package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nekomeowww/go-pinecone"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	"io"
	"sync"
	"time"
)

// SessionTimeLimit is the max duration of a session, it is in minutes
const SessionTimeLimit = 5

var sessions map[string]*session

// Make this read write mutex if performance is an issue
var sessionsMutex sync.Mutex

type session struct {
	user   User
	userMu sync.RWMutex

	index      *pinecone.IndexClient
	chatClient *openai.Client
	req        openai.ChatCompletionRequest
	deleteTime time.Time
	charLength int
}

// sessionTimer will timeout sessions that should be expired, the default is 5 min per session for now
func sessionTimer() {
	for range time.Tick(time.Minute) {
		sessionsMutex.Lock()
		for k, v := range sessions {
			if time.Now().After(v.deleteTime) {
				delete(sessions, k)
			}
		}
		sessionsMutex.Unlock()
	}
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

// updateTimer will update the session delete time to be SessionTimeLimit minutes in the future
func (s *session) updateTimer() {
	s.deleteTime = time.Now().Add(SessionTimeLimit * time.Minute)
}
func GetSessionWithoutPermanance(user User) (*session, error) {
	s := &session{
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

	if err := s.ValidateCredentials(); err != nil {
		return nil, err
	}

	s.req = openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo0613,
		Messages:  []openai.ChatCompletionMessage{},
		Stream:    true,
		Functions: function_call_defintions(),
	}
	return s, nil

}

// GetSession will see if a session exists, if so return it, otherwise it will validate the credentials in the passed
// in user object (credentials for pinecone and openai) and then create a session and return it.
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
		Model:     openai.GPT3Dot5Turbo0613,
		Messages:  []openai.ChatCompletionMessage{},
		Functions: function_call_defintions(),
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

func RemoveIndex(s []openai.ChatCompletionMessage, index int) []openai.ChatCompletionMessage {
	return append(s[:index], s[index+1:]...)
}

// Message will send a message to the chatbot with the context
func (s *session) Message(message string) (string, error) {
	fmt.Println(message)
	fmt.Println("MESSAGE^^^")
	s.updateTimer()
	if s.charLength+len(message) > 4097 {
		s.charLength -= len(s.req.Messages[0].Content)
		s.req.Messages = RemoveIndex(s.req.Messages, 0)
	}
	s.req.Messages = append(s.req.Messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: message,
	})
	resp, err := s.chatClient.CreateChatCompletion(context.Background(), s.req)
	if err != nil {
		fmt.Println("Why Here")
		return "", err
	}
	call := resp.Choices[0].Message.FunctionCall
	if call != nil {
		switch call.Name {
		case "query_notes":
			// query our notes for information
			response := s.queryNotes(call.Arguments)
			fmt.Println("resp")
			fmt.Println(response)
			s.req.Messages = append(s.req.Messages, resp.Choices[0].Message)
			s.req.Messages = append(s.req.Messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleFunction,
				Name:    QueryNotesName,
				Content: response,
			})
			resp, err = s.chatClient.CreateChatCompletion(context.Background(), s.req)
			if err != nil {
				fmt.Println("here is a joke")
				return "", err
			}
		}
	}
	s.req.Messages = append(s.req.Messages, resp.Choices[0].Message)
	return resp.Choices[0].Message.Content, nil
}

// queryNotes will query embed the query and use the embedding to query pinecone and get the content and return it
func (s *session) queryNotes(query string) string {
	fmt.Println("queyr Notes")
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

type ClientChan chan string

// Message will send a message to the chatbot with the context
func (s *session) Message2(message string, c *gin.Context) (string, error) {
	fmt.Println("WE DOING IT")
	s.updateTimer()
	if s.charLength+len(message) > 4097 {
		s.charLength -= len(s.req.Messages[0].Content)
		s.req.Messages = RemoveIndex(s.req.Messages, 0)
	}
	s.req.Messages = append(s.req.Messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: message,
	})
	stream, err := s.chatClient.CreateChatCompletionStream(context.Background(), s.req)
	if err != nil {
		fmt.Println("Why Here")
		return "", err
	}
	defer stream.Close()
	charComp := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: "",
	}
	c.Stream(func(w io.Writer) bool {

		callArgs := ""
		callName := ""
		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				fmt.Printf("Stream error: %v\n", err)
				return false
			}
			call := resp.Choices[0].Delta.FunctionCall
			if resp.Choices[0].Delta.Content != "" {
				c.SSEvent("message", resp.Choices[0].Delta.Content)
				c.Writer.Flush()
				charComp.Content += resp.Choices[0].Delta.Content
				fmt.Println(resp.Choices[0].Delta.Content)
			}
			if call != nil {
				callArgs += call.Arguments
				if callName == "" {
					callName = call.Name
				}
			}
			if resp.Choices[0].FinishReason == openai.FinishReasonFunctionCall {
				switch callName {
				case "query_notes":
					fmt.Println("here reached")
					// query our notes for information
					response := s.queryNotes(callArgs)
					s.req.Messages = append(s.req.Messages, openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleFunction,
						Name:    QueryNotesName,
						Content: response,
					})
					stream, err = s.chatClient.CreateChatCompletionStream(context.Background(), s.req)
					if err != nil {
						fmt.Println("here is a joke")
						return false
					}
				}
			}
		}
		return false
	})
	s.req.Messages = append(s.req.Messages, charComp)
	return charComp.Content, nil
}
