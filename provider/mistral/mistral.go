package mistral

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alexisbouchez/palm/provider"
	"github.com/alexisbouchez/palm/stream"
)

const baseURL = "https://api.mistral.ai/v1"

type mistral struct {
	apiKey  string
	model   string
	baseURL string
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func New() provider.Provider {
	return &mistral{
		model:   "mistral-small-latest",
		baseURL: baseURL,
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

func (m *mistral) WithBaseURL(url string) provider.Provider {
	m.baseURL = url
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

	httpReq, err := http.NewRequest(http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(body))
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

type streamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []streamChoice `json:"choices"`
}

type streamChoice struct {
	Index        int           `json:"index"`
	Delta        streamDelta   `json:"delta"`
	FinishReason *string       `json:"finish_reason"`
}

type streamDelta struct {
	Role      string                `json:"role,omitempty"`
	Content   string                `json:"content,omitempty"`
	ToolCalls []streamToolCall      `json:"tool_calls,omitempty"`
}

type streamToolCall struct {
	Index    int                `json:"index"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function *streamFunction    `json:"function,omitempty"`
}

type streamFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

func (m *mistral) StreamChat(messages []provider.Message, tools []provider.Tool, writer io.Writer) (*provider.StreamResult, error) {
	req := chatRequest{
		Model:    m.model,
		Messages: messages,
		Tools:    tools,
	}

	body, err := json.Marshal(map[string]any{
		"model":    req.Model,
		"messages": req.Messages,
		"tools":    req.Tools,
		"stream":   true,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)

	slog.Info("sending request to mistral", "model", m.model)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		slog.Error("mistral request failed", "error", err)
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("received response from mistral", "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("mistral api error", "status", resp.StatusCode, "body", string(respBody))
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
	}

	emitter := stream.NewEmitter(writer)

	messageID := generateID()
	textID := generateID()

	emitter.Start(messageID)

	scanner := bufio.NewScanner(resp.Body)
	toolCalls := make(map[int]*accumulatedToolCall)
	textStarted := false
	fullContent := ""
	chunkCount := 0

	for scanner.Scan() {
		chunkCount++
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]

		if choice.Delta.Content != "" {
			if !textStarted {
				emitter.TextStart(textID)
				textStarted = true
			}
			emitter.TextDelta(textID, choice.Delta.Content)
			fullContent += choice.Delta.Content
		}

		for _, tc := range choice.Delta.ToolCalls {
			acc, exists := toolCalls[tc.Index]
			if !exists {
				acc = &accumulatedToolCall{}
				toolCalls[tc.Index] = acc
			}

			if tc.ID != "" {
				acc.ID = tc.ID
			}
			if tc.Type != "" {
				acc.Type = tc.Type
			}
			if tc.Function != nil {
				if tc.Function.Name != "" {
					acc.Name = tc.Function.Name
					emitter.ToolInputStart(acc.ID, acc.Name)
				}
				if tc.Function.Arguments != "" {
					acc.Arguments += tc.Function.Arguments
					emitter.ToolInputDelta(acc.ID, tc.Function.Arguments)
				}
			}
		}

		if choice.FinishReason != nil {
			if textStarted {
				emitter.TextEnd(textID)
			}

			for _, acc := range toolCalls {
				var inputJSON any
				if err := json.Unmarshal([]byte(acc.Arguments), &inputJSON); err != nil {
					slog.Warn("failed to parse tool input as JSON", "error", err, "args", acc.Arguments)
					inputJSON = acc.Arguments
				}
				emitter.ToolInputAvailable(acc.ID, acc.Name, inputJSON)
			}

			emitter.Finish()
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("error scanning stream", "error", err)
		return nil, fmt.Errorf("scan stream: %w", err)
	}

	slog.Info("stream completed", "chunks", chunkCount, "content_length", len(fullContent), "tool_calls", len(toolCalls))

	emitter.Done()

	result := &provider.StreamResult{
		Message: provider.Message{
			Role:    "assistant",
			Content: fullContent,
		},
	}

	if len(toolCalls) > 0 {
		result.Message.ToolCalls = make([]provider.ToolCall, 0, len(toolCalls))
		for _, acc := range toolCalls {
			result.Message.ToolCalls = append(result.Message.ToolCalls, provider.ToolCall{
				ID:   acc.ID,
				Type: acc.Type,
				Function: provider.FunctionCall{
					Name:      acc.Name,
					Arguments: acc.Arguments,
				},
			})
		}
	}

	return result, nil
}

type accumulatedToolCall struct {
	ID        string
	Type      string
	Name      string
	Arguments string
}
