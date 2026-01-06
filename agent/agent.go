package agent

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alexisbouchez/palm/provider"
	"github.com/alexisbouchez/palm/tool"
)

type Agent interface {
	WithProvider(provider provider.Provider) Agent
	WithTool(tool tool.Callable) Agent
	Run() error
}

type agent struct {
	provider provider.Provider
	tools    []tool.Callable
}

func New() Agent {
	return &agent{}
}

func (a *agent) WithProvider(provider provider.Provider) Agent {
	a.provider = provider
	return a
}

func (a *agent) WithTool(tool tool.Callable) Agent {
	a.tools = append(a.tools, tool)
	return a
}

func (a *agent) Run() error {
	if a.provider == nil {
		return errors.New("provider undefined")
	}

	reader := bufio.NewReader(os.Stdin)
	messages := []provider.Message{}
	providerTools := a.buildTools()

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			break
		}

		messages = append(messages, provider.Message{
			Role:    "user",
			Content: input,
		})

		for {
			resp, err := a.provider.Chat(messages, providerTools)
			if err != nil {
				return fmt.Errorf("chat: %w", err)
			}

			if len(resp.Choices) == 0 {
				return errors.New("no choices in response")
			}

			assistantMsg := resp.Choices[0].Message
			for i := range assistantMsg.ToolCalls {
				if assistantMsg.ToolCalls[i].Type == "" {
					assistantMsg.ToolCalls[i].Type = "function"
				}
			}
			messages = append(messages, assistantMsg)

			if len(assistantMsg.ToolCalls) == 0 {
				fmt.Println(assistantMsg.Content)
				break
			}

			for _, tc := range assistantMsg.ToolCalls {
				result, err := a.executeTool(tc)
				if err != nil {
					result = fmt.Sprintf("error: %v", err)
				}

				messages = append(messages, provider.Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})
			}
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
