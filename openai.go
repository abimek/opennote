package main

import (
	"context"
	"github.com/sashabaranov/go-openai"
	"log"
)

const (
	USER = "Abi" // USER is a unique id for the specific isntance requesting embeddings, it will later change to be instance
	// of the app but for now we will keep it a constant
)

// method that returns a list of embedding information in the right order that we sent, the Embedding field of each of these is the vector represneation
func openaiEmbedding(client *openai.Client, model openai.EmbeddingModel, texts []string) [][]float32 {
	request := openai.EmbeddingRequest{
		Input: texts,
		Model: model,
		User:  USER,
	}

	resp, err := client.CreateEmbeddings(context.Background(), request)
	if err != nil {
		log.Fatal(err)
	}

	embeddingObjects := resp.Data
	embeddingVectors := make([][]float32, len(embeddingObjects), len(embeddingObjects))

	for i, embedding := range embeddingObjects {
		embeddingVectors[i] = embedding.Embedding
	}
	return embeddingVectors
}

func ada002Embeddings(client *openai.Client, texts []string) [][]float32 {
	return openaiEmbedding(client, openai.AdaEmbeddingV2, texts)
}
