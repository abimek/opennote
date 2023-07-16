package main

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

const (
	QueryNotesDescription = "This function returns notes from the users personal NOTES. If you ask about lets say Zustand the React state-management library it'll return relevant information from the users own notes. "
	QueryNotesName        = "query_notes"
)

func function_call_defintions() []openai.FunctionDefinition {
	return []openai.FunctionDefinition{{
		Name:        QueryNotesName,
		Description: QueryNotesDescription,
		Parameters: &jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"queries": {
					Type:        jsonschema.Array,
					Description: "List of queries for the users notes, like 'Zustand Usage' or 'B-Tree Implementation'",
					Items: &jsonschema.Definition{
						Type: "string",
					},
				},
			},
		},
	}}
}
