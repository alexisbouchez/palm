package stream

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Emitter struct {
	writer  io.Writer
	flusher http.Flusher
}

func NewEmitter(w io.Writer) *Emitter {
	emitter := &Emitter{writer: w}
	if f, ok := w.(http.Flusher); ok {
		emitter.flusher = f
	}
	return emitter
}

func (e *Emitter) flush() {
	if e.flusher != nil {
		e.flusher.Flush()
	}
}

const (
	EventStart               = "start"
	EventTextStart           = "text-start"
	EventTextDelta           = "text-delta"
	EventTextEnd             = "text-end"
	EventToolInputStart      = "tool-input-start"
	EventToolInputDelta      = "tool-input-delta"
	EventToolInputAvailable  = "tool-input-available"
	EventToolOutputAvailable = "tool-output-available"
	EventFinish              = "finish"
	EventError               = "error"
	EventDone                = "[DONE]"
)

func (e *Emitter) emit(data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}

	_, err = fmt.Fprintf(e.writer, "data: %s\n\n", jsonData)
	if err != nil {
		return err
	}

	e.flush()
	return nil
}

func (e *Emitter) Start(messageID string) error {
	data := map[string]any{
		"type": EventStart,
	}
	if messageID != "" {
		data["messageId"] = messageID
	}
	return e.emit(data)
}

func (e *Emitter) TextStart(id string) error {
	return e.emit(map[string]any{
		"type": EventTextStart,
		"id":   id,
	})
}

func (e *Emitter) TextDelta(id string, delta string) error {
	return e.emit(map[string]any{
		"type":  EventTextDelta,
		"id":    id,
		"delta": delta,
	})
}

func (e *Emitter) TextEnd(id string) error {
	return e.emit(map[string]any{
		"type": EventTextEnd,
		"id":   id,
	})
}

func (e *Emitter) ToolInputStart(toolCallID, toolName string) error {
	return e.emit(map[string]any{
		"type":       EventToolInputStart,
		"toolCallId": toolCallID,
		"toolName":   toolName,
	})
}

func (e *Emitter) ToolInputDelta(toolCallID, inputTextDelta string) error {
	return e.emit(map[string]any{
		"type":           EventToolInputDelta,
		"toolCallId":     toolCallID,
		"inputTextDelta": inputTextDelta,
	})
}

func (e *Emitter) ToolInputAvailable(toolCallID, toolName string, input any) error {
	return e.emit(map[string]any{
		"type":       EventToolInputAvailable,
		"toolCallId": toolCallID,
		"toolName":   toolName,
		"input":      input,
	})
}

func (e *Emitter) ToolOutputAvailable(toolCallID string, output any) error {
	return e.emit(map[string]any{
		"type":       EventToolOutputAvailable,
		"toolCallId": toolCallID,
		"output":     output,
	})
}

func (e *Emitter) Finish() error {
	return e.emit(map[string]any{
		"type": EventFinish,
	})
}

func (e *Emitter) Error(errorText string) error {
	return e.emit(map[string]any{
		"type":      EventError,
		"errorText": errorText,
	})
}

func (e *Emitter) Done() error {
	_, err := fmt.Fprintf(e.writer, "data: %s\n\n", EventDone)
	return err
}
