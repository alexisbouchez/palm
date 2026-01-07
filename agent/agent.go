package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/alexisbouchez/palm/provider"
	"github.com/alexisbouchez/palm/stream"
	"github.com/alexisbouchez/palm/tool"
)

type Agent interface {
	WithProvider(provider provider.Provider) Agent
	WithTool(tool tool.Callable) Agent
	WithStreamHandler(handler StreamHandler) Agent
	Chat(message string, writer io.Writer) error
}

type agent struct {
	provider       provider.Provider
	tools          []tool.Callable
	messages       []provider.Message
	streamHandler  StreamHandler
}

func New() Agent {
	return &agent{
		messages: []provider.Message{},
	}
}

func (a *agent) WithProvider(provider provider.Provider) Agent {
	a.provider = provider
	return a
}

func (a *agent) WithTool(tool tool.Callable) Agent {
	a.tools = append(a.tools, tool)
	return a
}

func (a *agent) WithStreamHandler(handler StreamHandler) Agent {
	a.streamHandler = handler
	return a
}

func (a *agent) Chat(message string, writer io.Writer) error {
	if a.provider == nil {
		return errors.New("provider undefined")
	}

	a.messages = append(a.messages, provider.Message{
		Role:    "user",
		Content: message,
	})

	providerTools := a.buildTools()

	outputWriter := writer
	if a.streamHandler != nil {
		outputWriter = a.streamHandler
	}

	for {
		streamResult, err := a.provider.StreamChat(a.messages, providerTools, outputWriter)
		if err != nil {
			return fmt.Errorf("stream chat: %w", err)
		}

		assistantMsg := streamResult.Message
		for i := range assistantMsg.ToolCalls {
			if assistantMsg.ToolCalls[i].Type == "" {
				assistantMsg.ToolCalls[i].Type = "function"
			}
		}
		a.messages = append(a.messages, assistantMsg)

		if len(assistantMsg.ToolCalls) == 0 {
			break
		}

		emitter := stream.NewEmitter(outputWriter)
		for _, tc := range assistantMsg.ToolCalls {
			result, err := a.executeTool(tc)

			var outputData any
			if err != nil {
				outputData = map[string]string{"error": err.Error()}
				result = fmt.Sprintf("error: %v", err)
			} else {
				if err := json.Unmarshal([]byte(result), &outputData); err != nil {
					outputData = map[string]string{"result": result}
				}
			}

			emitter.ToolOutputAvailable(tc.ID, outputData)

			a.messages = append(a.messages, provider.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return nil
}

func (a *agent) buildTools() []provider.Tool {
	tools := make([]provider.Tool, len(a.tools))
	for i, t := range a.tools {
		tools[i] = provider.Tool{
			Type: "function",
			Function: provider.ToolFunction{
				Name:        t.GetName(),
				Description: t.GetDescription(),
				Parameters:  t.GetParameters(),
			},
		}
	}
	return tools
}

func (a *agent) executeTool(tc provider.ToolCall) (string, error) {
	for _, t := range a.tools {
		if t.GetName() == tc.Function.Name {
			return t.Call(json.RawMessage(tc.Function.Arguments))
		}
	}
	return "", fmt.Errorf("tool not found: %s", tc.Function.Name)
}
