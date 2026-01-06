package mistral

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/alexisbouchez/palm/provider"
)

const baseURL = "https://api.mistral.ai/v1"

type mistral struct {
	apiKey string
	model  string
}

func New() provider.Provider {
	return &mistral{
		model: "mistral-small-latest",
	}
}

func (m *mistral) WithAPIKey(key string) provider.Provider {
	m.apiKey = key
	return m
}

func (m *mistral) WithModel(model string) provider.Provider {
	m.model = model
	return m
}

type chatRequest struct {
	Model    string             `json:"model"`
	Messages []provider.Message `json:"messages"`
	Tools    []provider.Tool    `json:"tools,omitempty"`
}

func (m *mistral) Chat(messages []provider.Message, tools []provider.Tool) (*provider.ChatResponse, error) {
	req := chatRequest{
		Model:    m.model,
		Messages: messages,
		Tools:    tools,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp provider.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &chatResp, nil
}
