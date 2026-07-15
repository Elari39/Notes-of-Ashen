package aiclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

type openAIChatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	Temperature    float64        `json:"temperature"`
	MaxTokens      int            `json:"max_tokens"`
	ResponseFormat responseFormat `json:"response_format"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *providerError `json:"error,omitempty"`
}

func newOpenAIChatRequest(model, system, user string, temperature float64, maxTokens int) openAIChatRequest {
	return openAIChatRequest{
		Model:          strings.TrimSpace(model),
		Messages:       []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
		Temperature:    temperature,
		MaxTokens:      maxTokens,
		ResponseFormat: responseFormat{Type: "json_object"},
	}
}

func parseOpenAIContent(raw []byte) (string, error) {
	var response openAIChatResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return "", fmt.Errorf("decode ai response: %w", err)
	}
	if response.Error != nil && strings.TrimSpace(response.Error.Message) != "" {
		return "", fmt.Errorf("ai request failed: %s", response.Error.Message)
	}
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("ai response has no choices")
	}
	return response.Choices[0].Message.Content, nil
}
