package nostr

import (
	"context"
	"errors"
	"slices"
)

type RelayStore interface {
	Publish(ctx context.Context, event Event) error
	QuerySync(ctx context.Context, filter Filter, opts ...SubscriptionOption) ([]*Event, error)
}

var (
	_ RelayStore = (*Relay)(nil)
	_ RelayStore = (*MultiStore)(nil)
)

type MultiStore []RelayStore

func (multi MultiStore) Publish(ctx context.Context, event Event) error {
	errs := make([]error, len(multi))
	for i, s := range multi {
		errs[i] = s.Publish(ctx, event)
	}
	return errors.Join(errs...)
}

func (multi MultiStore) QuerySync(ctx context.Context, filter Filter, opts ...SubscriptionOption) ([]*Event, error) {
	errs := make([]error, len(multi))
	events := make([]*Event, 0, max(filter.Limit, 10))
	for i, s := range multi {
		res, err := s.QuerySync(ctx, filter, opts...)
		errs[i] = err
		events = append(events, res...)
	}
	slices.SortFunc(events, func(a, b *Event) int {
		if b.CreatedAt > a.CreatedAt {
			return 1
		} else if b.CreatedAt < a.CreatedAt {
			return -1
		}
		return 0
	})
	return events, errors.Join(errs...)
}
