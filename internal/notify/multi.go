package notify

import (
	"context"
	"errors"
)

// MultiNotifier fans out a message to multiple Notifiers.
// A failure in one notifier does not prevent others from being called.
type MultiNotifier struct {
	notifiers []Notifier
}

func NewMultiNotifier(nn ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: nn}
}

func (m *MultiNotifier) Send(ctx context.Context, msg Message) error {
	var errs []error
	for _, n := range m.notifiers {
		if err := n.Send(ctx, msg); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m *MultiNotifier) Name() string { return "multi" }
