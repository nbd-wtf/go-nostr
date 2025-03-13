package nip13

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	nostr "github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestCheck(t *testing.T) {
	const eventID = "000000000e9d97a1ab09fc381030b346cdd7a142ad57e6df0b46dc9bef6c7e2d"
	tests := []struct {
		minDifficulty int
		wantErr       error
	}{
		{-1, nil},
		{0, nil},
		{1, nil},
		{35, nil},
		{36, nil},
		{37, ErrDifficultyTooLow},
		{42, ErrDifficultyTooLow},
	}
	for i, tc := range tests {
		if err := Check(eventID, tc.minDifficulty); err != tc.wantErr {
			t.Errorf("%d: Check(%q, %d) returned %v; want err: %v", i, eventID, tc.minDifficulty, err, tc.wantErr)
		}
	}
}

func TestCommittedDifficulty(t *testing.T) {
	tests := []struct {
		result int
		id     string
		tags   nostr.Tags
	}{
		{18, "000000000e9d97a1ab09fc381030b346cdd7a142ad57e6df0b46dc9bef6c7e2d", nostr.Tags{{"-"}, {"nonce", "654", "18"}}},
		{36, "000000000e9d97a1ab09fc381030b346cdd7a142ad57e6df0b46dc9bef6c7e2d", nostr.Tags{{"nonce", "12315", "36"}}},
		{0, "000000000e9d97a1ab09fc381030b346cdd7a142ad57e6df0b46dc9bef6c7e2d", nostr.Tags{{"nonce", "12315", "37"}}},
		{0, "000000000e9d97a1ab09fc381030b346cdd7a142ad57e6df0b46dc9bef6c7e2d", nostr.Tags{{"nonce", "654", "64"}, {"t", "spam"}}},
		{0, "000000000e9d97a1ab09fc381030b346cdd7a142ad57e6df0b46dc9bef6c7e2d", nostr.Tags{}},
	}
	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			work := CommittedDifficulty(&nostr.Event{ID: tc.id, Tags: tc.tags})
			require.Equal(t, tc.result, work)
		})
	}
}

func TestDoWorkShort(t *testing.T) {
	event := nostr.Event{
		Kind:    nostr.KindTextNote,
		Content: "It's just me mining my own business",
		PubKey:  "a48380f4cfcc1ad5378294fcac36439770f9c878dd880ffa94bb74ea54a6f243",
	}
	pow, err := DoWork(context.Background(), event, 2)
	if err != nil {
		t.Fatal(err)
	}
	testNonceTag(t, pow, 2)
}

func TestDoWorkLong(t *testing.T) {
	if testing.Short() {
		t.Skip("too consuming for short mode")
	}
	for _, difficulty := range []int{8, 16} {
		difficulty := difficulty
		t.Run(fmt.Sprintf("%dbits", difficulty), func(t *testing.T) {
			t.Parallel()
			event := nostr.Event{
				Kind:    nostr.KindTextNote,
				Content: "It's just me mining my own business",
				PubKey:  "a48380f4cfcc1ad5378294fcac36439770f9c878dd880ffa94bb74ea54a6f243",
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			pow, err := DoWork(ctx, event, difficulty)
			if err != nil {
				t.Fatal(err)
			}
			event.Tags = append(event.Tags, pow)
			if err := Check(event.GetID(), difficulty); err != nil {
				t.Error(err)
			}
			testNonceTag(t, pow, difficulty)
		})
	}
}

func testNonceTag(t *testing.T, tag nostr.Tag, commitment int) {
	t.Helper()
	if tag[0] != "nonce" {
		t.Errorf("tag[0] = %q; want 'nonce'", tag[0])
	}
	if n, err := strconv.ParseInt(tag[1], 10, 64); err != nil || n < 1 {
		t.Errorf("tag[1] = %q; want an int greater than 0", tag[1])
	}
	if n, err := strconv.Atoi(tag[2]); err != nil || n != commitment {
		t.Errorf("tag[2] = %q; want %d", tag[2], commitment)
	}
}

func TestDoWorkTimeout(t *testing.T) {
	event := nostr.Event{
		Kind:    nostr.KindTextNote,
		Content: "It's just me mining my own business",
		PubKey:  "a48380f4cfcc1ad5378294fcac36439770f9c878dd880ffa94bb74ea54a6f243",
	}
	done := make(chan error)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		_, err := DoWork(ctx, event, 256)
		done <- err
	}()
	select {
	case <-time.After(time.Second):
		t.Error("DoWork took too long to timeout")
	case err := <-done:
		if !errors.Is(err, ErrGenerateTimeout) {
			t.Errorf("DoWork returned %v; want ErrDoWorkTimeout", err)
		}
	}
}
