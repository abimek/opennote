package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nekomeowww/go-pinecone"
	"github.com/sashabaranov/go-openai"
	"io"
	"net/http"
)

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

// query is the primary function used by ChatGPT to query data from this plugin
func openAIQueryEndpointHandler(c *gin.Context) {
	// validate token
	token := c.GetHeader("Authorization")
	//TODO: REMOVE TEH TOKEN SET ONCE PUT INTO NON-LOCALHOST ENVIRONMENT
	token = "thedata"
	if token == "" {
		c.Status(http.StatusUnauthorized)
		return
	}
	docs, err := firestoreClient.Collection("users").Where("token", "==", token).Limit(1).Documents(context.Background()).GetAll()
	if err != nil || len(docs) == 0 {
		c.JSON(http.StatusUnauthorized, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Unable to find user with this token",
		})
		return
	}

	jsonData, _ := json.Marshal(docs[0].Data())
	fmt.Println(string(jsonData))
	var user User
	if err = json.Unmarshal(jsonData, &user); err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Data Format Error",
		})
		return
	}

	var request QueryRequest
	d, _ := io.ReadAll(c.Request.Body)

	if err = json.Unmarshal(d, &request); err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Data Format Error",
		})
		return
	}
	fmt.Println(string(d))
	fmt.Println(request.Queries)

	queries := request.Queries

	gptClient := openai.NewClient(user.OpenAIApiKey)

	_, err = gptClient.ListModels(context.Background())
	if err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: PineconeError,
			content:   "Invalid OpenaAI API Key",
		})
		return
	}

	embeddings := ada002Embeddings(gptClient, queries)

	pineconeIndex, _ := pinecone.NewIndexClient(
		pinecone.WithIndexName(user.PineconeIndex),
		pinecone.WithAPIKey(user.PineconeApiKey),
		pinecone.WithEnvironment(user.PineconeEnvironment),
		pinecone.WithProjectName(user.PrinconeProjectName),
	)

	// validate credentials
	_, err = pineconeIndex.DescribeIndexStats(context.Background(), pinecone.DescribeIndexStatsParams{})
	if err != nil {
		c.JSON(http.StatusUnauthorized, RequestErrorResult{
			errorCode: PineconeError,
			content:   "Unable to login to pinecone",
		})
		return
	}

	// handle the response
	resp := QueryResponse{}
	for i, embedding := range embeddings {
		content := queryPinecone(pineconeIndex, user.TopK, embedding)
		resp.Results = append(resp.Results, QueryResult{
			Query:  queries[i],
			Result: content,
		})
	}
	c.JSON(http.StatusOK, resp)
	return
}
