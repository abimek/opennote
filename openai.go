package main

import (
	"context"
	"github.com/sashabaranov/go-openai"
)

// method that returns a list of embedding information in the right order that we sent, the Embedding field of each of these is the vector represneation
func openaiEmbedding(client *openai.Client, model openai.EmbeddingModel, user string, texts []string) ([][]float32, error) {
	request := openai.EmbeddingRequest{
		Input: texts,
		Model: model,
		User:  user,
	}

	resp, err := client.CreateEmbeddings(context.Background(), request)
	if err != nil {
		return nil, err
	}

	embeddingObjects := resp.Data
	embeddingVectors := make([][]float32, len(embeddingObjects), len(embeddingObjects))

	for i, embedding := range embeddingObjects {
		embeddingVectors[i] = embedding.Embedding
	}
	return embeddingVectors, nil
}

func ada002Embeddings(client *openai.Client, user string, texts []string) ([][]float32, error) {
	return openaiEmbedding(client, openai.AdaEmbeddingV2, user, texts)
}
