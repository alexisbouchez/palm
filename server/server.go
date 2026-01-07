package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/alexisbouchez/palm/agent"
	"github.com/alexisbouchez/palm/provider"
	"github.com/alexisbouchez/palm/tool"
)

type Server interface{
	Start(addr string) error
}

type server struct {
	provider provider.Provider
	tools    []tool.Callable
}

type ChatRequest struct {
	Message string `json:"message"`
}

func New(provider provider.Provider, tools []tool.Callable) Server {
	return &server{
		provider: provider,
		tools:    tools,
	}
}

func (s *server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to parse chat request", "error", err)
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	slog.Info("handling chat request", "message", req.Message)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("x-vercel-ai-ui-message-stream", "v1")

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	agt := agent.New().WithProvider(s.provider)
	for _, t := range s.tools {
		agt = agt.WithTool(t)
	}

	if err := agt.Chat(req.Message, w); err != nil {
		slog.Error("agent chat failed", "error", err)
		fmt.Fprintf(w, "data: {\"type\":\"error\",\"error\":\"%s\"}\n\n", err.Error())
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

func (s *server) Start(addr string) error {
	http.HandleFunc("POST /chat", s.handleChat)
	http.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	slog.Info("http server listening for requests", "addr", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		slog.Error("server stopped", "error", err)
		return err
	}
	return nil
}
