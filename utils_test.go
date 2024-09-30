package nostr

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidRelayURL(t *testing.T) {
	tests := []struct {
		u    string
		want bool
	}{
		{"ws://127.0.0.1", true},
		{"ws://localhost", true},
		{"wss://localhost", true},
		{"wss://relay.nostr.com", true},
		{"http://127.0.0.1", false},
		{"127.0.0.1", false},
	}

	for _, test := range tests {
		got := IsValidRelayURL(test.u)
		assert.Equal(t, test.want, got)
	}
}

func TestEventsCompare(t *testing.T) {
	list := []Event{
		{CreatedAt: 12},
		{CreatedAt: 8},
		{CreatedAt: 26},
		{CreatedAt: 1},
	}

	slices.SortFunc(list, CompareEvent)
	require.Equal(t, []Event{
		{CreatedAt: 1},
		{CreatedAt: 8},
		{CreatedAt: 12},
		{CreatedAt: 26},
	}, list)

	slices.SortFunc(list, CompareEventReverse)
	require.Equal(t, []Event{
		{CreatedAt: 26},
		{CreatedAt: 12},
		{CreatedAt: 8},
		{CreatedAt: 1},
	}, list)
}

func TestEventsComparePtr(t *testing.T) {
	list := []*Event{
		{CreatedAt: 12},
		{CreatedAt: 8},
		{CreatedAt: 26},
		{CreatedAt: 1},
	}

	slices.SortFunc(list, CompareEventPtr)
	require.Equal(t, []*Event{
		{CreatedAt: 1},
		{CreatedAt: 8},
		{CreatedAt: 12},
		{CreatedAt: 26},
	}, list)

	slices.SortFunc(list, CompareEventPtrReverse)
	require.Equal(t, []*Event{
		{CreatedAt: 26},
		{CreatedAt: 12},
		{CreatedAt: 8},
		{CreatedAt: 1},
	}, list)
}
