package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/alexisbouchez/palm/agent"
	"github.com/alexisbouchez/palm/provider/mistral"
	"github.com/alexisbouchez/palm/tool"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

type WeatherInput struct {
	Location string `json:"location" description:"The city name" required:"true"`
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	provider := mistral.New().
		WithAPIKey(os.Getenv("MISTRAL_API_KEY"))

	weatherTool := tool.New[WeatherInput]().
		WithName("get_weather").
		WithDescription("Get the weather in a location").
		WithExecute(func(input WeatherInput) (string, error) {
			slog.Debug("executing weather tool", "location", input.Location)
			return fmt.Sprintf("The weather in %s is sunny, 22°C", input.Location), nil
		})

	consoleHandler := agent.NewConsoleHandler(os.Stdout)

	agt := agent.New().
		WithProvider(provider).
		WithTool(weatherTool).
		WithStreamHandler(consoleHandler)

	if term.IsTerminal(int(os.Stdin.Fd())) {
		reader := bufio.NewReader(os.Stdin)
		promptStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

		for {
			fmt.Print(promptStyle.Render("❯") + " ")
			input, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
				os.Exit(1)
			}

			input = strings.TrimSpace(input)
			if input == "" {
				continue
			}
			if input == "exit" || input == "quit" {
				break
			}

			if err := agt.Chat(input, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}

		if err := agt.Chat(strings.TrimSpace(input), os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
