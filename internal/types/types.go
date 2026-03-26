package types

import (
	"context"
	"time"
)

// Channel defines the interface for communication channels
type Channel interface {
	Name() string
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	SendMessage(ctx context.Context, jid string, text string) error
	IsConnected() bool
	OwnsJid(jid string) bool
}

// NewMessage represents an incoming message
type NewMessage struct {
	ID         string
	ChatJID    string
	Sender     string
	SenderName string
	Content    string
	Timestamp  time.Time
	IsFromMe   bool
	IsBot      bool
}

// RegisteredGroup represents a group registered with the assistant
type RegisteredGroup struct {
	JID             string
	Name            string
	Folder          string
	Trigger         string
	AddedAt         time.Time
	RequiresTrigger bool
	IsMain          bool
	ContainerConfig string // JSON string for additional config
}

// ContainerInput is the JSON passed to the agent container
type ContainerInput struct {
	Prompt        string `json:"prompt"`
	SessionID     string `json:"sessionId,omitempty"`
	GroupFolder   string `json:"groupFolder"`
	ChatJID       string `json:"chatJid"`
	IsMain          bool   `json:"isMain"`
	AssistantName string `json:"assistantName"`
}

// ScheduledTask represents a task to be run by the scheduler
type ScheduledTask struct {
	ID            string    `json:"id"`
	GroupFolder   string    `json:"groupFolder"`
	ChatJID       string    `json:"chatJid"`
	Prompt        string    `json:"prompt"`
	ScheduleType  string    `json:"schedule_type"` // "cron", "interval", "once"
	ScheduleValue string    `json:"schedule_value"`
	NextRun       time.Time `json:"next_run"`
	LastRun       time.Time `json:"last_run"`
	LastResult    string    `json:"last_result"`
	Status        string    `json:"status"` // "active", "paused", "completed"
	CreatedAt     time.Time `json:"created_at"`
}

// TaskRunLog records the outcome of a task execution
type TaskRunLog struct {
	TaskID     string    `json:"task_id"`
	RunAt      time.Time `json:"run_at"`
	DurationMS int64     `json:"duration_ms"`
	Status     string    `json:"status"` // "success", "error"
	Result     string    `json:"result"`
	Error      string    `json:"error"`
}

// ContainerOutput is the JSON streamed back from the agent container
type ContainerOutput struct {
	Status       string      `json:"status"` // "success" or "error"
	Result       interface{} `json:"result"`
	NewSessionID string      `json:"newSessionId,omitempty"`
	Error        string      `json:"error,omitempty"`
}
