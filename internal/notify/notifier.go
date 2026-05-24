package notify

import (
	"context"
	"time"
)

type Level int

const (
	LevelInfo Level = iota
	LevelError
)

// Message is the payload sent to a Notifier.
type Message struct {
	Level        Level
	DatabaseName string
	Subject      string
	Body         string
	Timestamp    time.Time
	Hostname     string
}

// Notifier sends a notification message.
type Notifier interface {
	Send(ctx context.Context, msg Message) error
	Name() string
}
