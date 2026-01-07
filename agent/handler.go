package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	toolStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type StreamHandler interface {
	io.Writer
}

type SSEHandler struct {
	writer io.Writer
}

func NewSSEHandler(w io.Writer) *SSEHandler {
	return &SSEHandler{writer: w}
}

func (h *SSEHandler) Write(p []byte) (n int, err error) {
	return h.writer.Write(p)
}

type spinnerModel struct {
	spinner  spinner.Model
	message  string
	quitting bool
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m spinnerModel) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), dimStyle.Render(m.message))
}

type ConsoleHandler struct {
	writer      io.Writer
	buffer      strings.Builder
	program     *tea.Program
	isStreaming bool
}

func NewConsoleHandler(w io.Writer) *ConsoleHandler {
	return &ConsoleHandler{writer: w}
}

func (h *ConsoleHandler) startSpinner(message string) {
	if h.program != nil {
		return
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))

	m := spinnerModel{
		spinner: s,
		message: message,
	}

	h.program = tea.NewProgram(m)
	go h.program.Run()
}

func (h *ConsoleHandler) stopSpinner() {
	if h.program != nil {
		h.program.Quit()
		time.Sleep(10 * time.Millisecond)
		h.program = nil
	}
}

func (h *ConsoleHandler) Write(p []byte) (n int, err error) {
	h.buffer.Write(p)

	data := h.buffer.String()
	for {
		idx := strings.Index(data, "\n\n")
		if idx == -1 {
			break
		}

		event := data[:idx]
		data = data[idx+2:]

		if strings.HasPrefix(event, "data: ") {
			eventData := strings.TrimPrefix(event, "data: ")

			if eventData == "[DONE]" {
				h.stopSpinner()
				continue
			}

			var eventObj map[string]any
			if err := json.Unmarshal([]byte(eventData), &eventObj); err == nil {
				h.handleEvent(eventObj)
			}
		}
	}

	h.buffer.Reset()
	h.buffer.WriteString(data)

	return len(p), nil
}

func (h *ConsoleHandler) handleEvent(event map[string]any) {
	eventType, _ := event["type"].(string)

	switch eventType {
	case "start":
		h.startSpinner("Thinking...")

	case "text-start":
		h.stopSpinner()
		h.isStreaming = true

	case "text-delta":
		if delta, ok := event["delta"].(string); ok {
			fmt.Fprint(h.writer, delta)
		}

	case "text-end":
		h.isStreaming = false
		fmt.Fprintln(h.writer)

	case "tool-input-start":
		h.stopSpinner()
		if toolName, ok := event["toolName"].(string); ok {
			icon := toolStyle.Render("⚙")
			label := dimStyle.Render(" Calling tool:")
			name := toolStyle.Render(" " + toolName)
			fmt.Fprintf(h.writer, "\n%s%s%s\n", icon, label, name)
		}
		h.startSpinner("Executing...")

	case "tool-output-available":
		h.stopSpinner()
		if output, ok := event["output"].(map[string]any); ok {
			if result, ok := output["result"].(string); ok {
				icon := successStyle.Render("✓")
				label := dimStyle.Render(" Result:")
				fmt.Fprintf(h.writer, "%s%s %s\n\n", icon, label, result)
			} else if errMsg, ok := output["error"].(string); ok {
				icon := errorStyle.Render("✗")
				label := dimStyle.Render(" Error:")
				fmt.Fprintf(h.writer, "%s%s %s\n\n", icon, label, errorStyle.Render(errMsg))
			}
		}
		h.startSpinner("Thinking...")

	case "finish":
		h.stopSpinner()
		if !h.isStreaming {
			fmt.Fprintln(h.writer)
		}
	}
}

func (h *ConsoleHandler) Flush() {
	if h.buffer.Len() > 0 {
		scanner := bufio.NewScanner(strings.NewReader(h.buffer.String()))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				eventData := strings.TrimPrefix(line, "data: ")
				if eventData != "[DONE]" {
					var eventObj map[string]any
					if err := json.Unmarshal([]byte(eventData), &eventObj); err == nil {
						h.handleEvent(eventObj)
					}
				}
			}
		}
		h.buffer.Reset()
	}
}
