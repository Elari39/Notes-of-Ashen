package aiclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ParseAssistantJSON(content string) (*Response, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("ai response is empty")
	}
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("ai response is not json")
	}
	var resp Response
	if err := json.Unmarshal([]byte(content[start:end+1]), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func validateProbeJSON(content string) error {
	content = strings.TrimSpace(content)
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return fmt.Errorf("ai probe response is not json")
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content[start:end+1]), &value); err != nil {
		return fmt.Errorf("decode ai probe response: %w", err)
	}
	rawOK, exists := value["ok"]
	if !exists {
		return fmt.Errorf("ai probe response is not ok")
	}
	var ok bool
	if err := json.Unmarshal(rawOK, &ok); err != nil {
		return fmt.Errorf("decode ai probe response: %w", err)
	}
	if !ok {
		return fmt.Errorf("ai probe response is not ok")
	}
	return nil
}
