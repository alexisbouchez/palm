package main

import (
	"fmt"
	"os"

	"github.com/alexisbouchez/palm/agent"
)

func main() {
	a := agent.New()
	if err := a.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
}
