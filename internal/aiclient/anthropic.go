package aiclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type anthropicRequest struct {
	Model       string        `json:"model"`
	System      string        `json:"system"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *providerError `json:"error,omitempty"`
}

func newAnthropicRequest(model, system, user string, temperature float64, maxTokens int) anthropicRequest {
	return anthropicRequest{
		Model:       strings.TrimSpace(model),
		System:      system,
		Messages:    []chatMessage{{Role: "user", Content: user}},
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
}

func anthropicModelsURL(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse anthropic models url: %w", err)
	}
	query := parsed.Query()
	query.Set("limit", "1000")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func parseAnthropicContent(raw []byte) (string, error) {
	var response anthropicResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return "", fmt.Errorf("decode ai response: %w", err)
	}
	if response.Error != nil && strings.TrimSpace(response.Error.Message) != "" {
		return "", fmt.Errorf("ai request failed: %s", response.Error.Message)
	}
	var content strings.Builder
	for _, block := range response.Content {
		if block.Type == "text" {
			content.WriteString(block.Text)
		}
	}
	if content.Len() == 0 {
		return "", fmt.Errorf("ai response has no text content")
	}
	return content.String(), nil
}
