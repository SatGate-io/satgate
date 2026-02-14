package mcpserver

import (
	"encoding/json"
	"time"
)

// EventType identifies the kind of MCP proxy event.
type EventType string

const (
	EventToolCall       EventType = "tool_call"
	EventBudgetSpend    EventType = "budget_spend"
	EventBudgetExhaust  EventType = "budget_exhausted"
	EventBudgetError    EventType = "budget_error"
	EventDelegation     EventType = "delegation"
	EventAuthFailure    EventType = "auth_failure"
	EventSessionConnect EventType = "session_connect"
	EventSessionClose   EventType = "session_close"
)

// Event is a structured event emitted by the MCP proxy.
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	SessionID string                 `json:"sessionId,omitempty"`
	TokenID   string                 `json:"tokenId,omitempty"`
	BudgetID  string                 `json:"budgetId,omitempty"`
	TenantID  string                 `json:"tenantId,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// EventPublisher is the interface for publishing MCP proxy events.
// OSS: no-op. Enterprise: Redis pub/sub.
type EventPublisher interface {
	Publish(event Event)
}

// NoOpPublisher discards all events (OSS default).
type NoOpPublisher struct{}

func (n *NoOpPublisher) Publish(_ Event) {}

// ChannelPublisher sends events to a Go channel (for testing or in-process consumers).
type ChannelPublisher struct {
	ch chan Event
}

// NewChannelPublisher creates a publisher that sends events to a buffered channel.
func NewChannelPublisher(bufSize int) *ChannelPublisher {
	return &ChannelPublisher{ch: make(chan Event, bufSize)}
}

func (c *ChannelPublisher) Publish(event Event) {
	select {
	case c.ch <- event:
	default:
		// Drop if buffer full — non-blocking
	}
}

// Events returns the channel for consuming events.
func (c *ChannelPublisher) Events() <-chan Event {
	return c.ch
}

// MarshalEvent serializes an event to JSON.
func MarshalEvent(e Event) ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalEvent deserializes a JSON event.
func UnmarshalEvent(data []byte) (Event, error) {
	var e Event
	err := json.Unmarshal(data, &e)
	return e, err
}
