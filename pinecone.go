package main

import (
	"context"
	"github.com/nekomeowww/go-pinecone"
	"github.com/rs/zerolog/log"
)

// Querys pinecone and returns the content
func queryPinecone(indexClient *pinecone.IndexClient, topK int64, embedding []float32) []string {
	params := pinecone.QueryParams{
		IncludeMetadata: true,
		Vector:          embedding,
		TopK:            topK,
		Namespace:       "",
	}

	resp, err := indexClient.Query(context.Background(), params)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Invalid Pinecone Index On Request")
		return []string{}
	}

	var results []string

	for _, match := range resp.Matches {
		content, ok := match.Vector.Metadata["content"].(string)
		if !ok {
			continue
		}
		results = append(results, content)
	}
	return results
}
