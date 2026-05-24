package notify

import "context"

// NoopNotifier discards all messages. Used when notifications are disabled.
type NoopNotifier struct{}

func (n *NoopNotifier) Send(_ context.Context, _ Message) error { return nil }
func (n *NoopNotifier) Name() string                            { return "noop" }
