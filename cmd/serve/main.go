package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alexisbouchez/palm/env"
	"github.com/alexisbouchez/palm/provider/mistral"
	"github.com/alexisbouchez/palm/server"
	"github.com/alexisbouchez/palm/tool"
)

type WeatherInput struct {
	Location string `json:"location" description:"The city name" required:"true"`
}

func main() {
	addr := env.GetVar("HTTP_ADDR", ":4096")

	provider := mistral.New().
		WithAPIKey(os.Getenv("MISTRAL_API_KEY"))

	weatherTool := tool.New[WeatherInput]().
		WithName("get_weather").
		WithDescription("Get the weather in a location").
		WithExecute(func(input WeatherInput) (string, error) {
			slog.Debug("executing weather tool", "location", input.Location)
			return "The weather in " + input.Location + " is sunny, 22°C", nil
		})

	srv := server.New(provider, []tool.Callable{weatherTool})

	if err := srv.Start(addr); err != nil {
		if strings.Contains(err.Error(), "address already in use") {
			fmt.Fprintf(os.Stderr, "\n❌ Port %s is already in use!\n\n", addr)
			fmt.Fprintf(os.Stderr, "To fix this:\n")
			fmt.Fprintf(os.Stderr, "1. Find the process using the port:\n")
			fmt.Fprintf(os.Stderr, "   lsof -i %s\n\n", addr)
			fmt.Fprintf(os.Stderr, "2. Kill the process:\n")
			fmt.Fprintf(os.Stderr, "   kill <PID>\n")
			fmt.Fprintf(os.Stderr, "   or: killall serve\n\n")
			fmt.Fprintf(os.Stderr, "3. Or use a different port:\n")
			fmt.Fprintf(os.Stderr, "   HTTP_ADDR=:8080 go run ./cmd/serve\n\n")
		} else {
			slog.Error("server failed to start", "error", err)
		}
		os.Exit(1)
	}
}
