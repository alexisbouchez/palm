package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alexisbouchez/palm/agent"
	"github.com/alexisbouchez/palm/provider/mistral"
	"github.com/alexisbouchez/palm/tool"
)

type WeatherInput struct {
	Location string `json:"location"`
}

func main() {
	provider := mistral.New().
		WithAPIKey(os.Getenv("MISTRAL_API_KEY"))

	weatherTool := tool.New[WeatherInput]().
		WithName("get_weather").
		WithDescription("Get the weather in a location").
		WithParameters(json.RawMessage(`{
			"type": "object",
			"properties": {
				"location": {
					"type": "string",
					"description": "The city name"
				}
			},
			"required": ["location"]
		}`)).
		WithExecute(func(input WeatherInput) (string, error) {
			return fmt.Sprintf("The weather in %s is sunny, 22°C", input.Location), nil
		})

	agt := agent.New().
		WithProvider(provider).
		WithTool(weatherTool)

	if err := agt.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
