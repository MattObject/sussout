package llm

import (
	"context"
	"encoding/json"
	"regexp"
)

const extractorSystemPrompt = `You are an expert at identifying and to extract underlying assumptions in human communication.
Analyze the conversation and extract any hidden or implicit assumptions the user is making.
Return your findings as a JSON object with a single key "assumptions" containing an array of strings.
Example: { "assumptions": ["The user assumes the problem is hardware-related", "The user assumes they need a new license"] }
If no assumptions are found, return an empty array: { "assumptions": [] }
Only return the JSON object, nothing else.`

var jsonRe = regexp.MustCompile(`\{[\s\S]*\}`)

type AssumptionExtractor struct {
	client *LMStudioClient
}

func NewAssumptionExtractor(client *LMStudioClient) *AssumptionExtractor {
	return &AssumptionExtractor{client: client}
}

func (e *AssumptionExtractor) Extract(ctx context.Context, messages []Message) ([]string, error) {
	msgs := make([]Message, 0, len(messages)+1)
	msgs = append(msgs, Message{Role: "system", Content: extractorSystemPrompt})
	msgs = append(msgs, messages...)

	response, err := e.client.SendMessage(ctx, msgs)
	if err != nil {
		return nil, err
	}

	match := jsonRe.FindString(response)
	if match == "" {
		return nil, nil
	}

	var result struct {
		Assumptions []string `json:"assumptions"`
	}
	if err := json.Unmarshal([]byte(match), &result); err != nil {
		return nil, nil
	}

	return result.Assumptions, nil
}
