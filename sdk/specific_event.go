package sdk

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/sdk/hints"
)

// FetchSpecificEventParameters contains options for fetching specific events.
type FetchSpecificEventParameters struct {
	// WithRelays indicates whether to include relay information in the response
	// (this causes the request to take longer as it will wait for all relays to respond).
	WithRelays bool

	// SkipLocalStore indicates whether to skip checking the local store for the event
	// and storing the result in the local store.
	SkipLocalStore bool
}

// FetchSpecificEventFromInput tries to get a specific event from a NIP-19 code or event ID.
// It supports nevent, naddr, and note NIP-19 codes, as well as raw event IDs.
func (sys *System) FetchSpecificEventFromInput(
	ctx context.Context,
	input string,
	params FetchSpecificEventParameters,
) (event *nostr.Event, successRelays []string, err error) {
	var pointer nostr.Pointer

	prefix, data, err := nip19.Decode(input)
	if err == nil {
		switch prefix {
		case "nevent":
			pointer = data.(nostr.EventPointer)
		case "naddr":
			pointer = data.(nostr.EntityPointer)
		case "note":
			pointer = nostr.EventPointer{ID: data.(string)}
		default:
			return nil, nil, fmt.Errorf("invalid code '%s'", input)
		}
	} else {
		if nostr.IsValid32ByteHex(input) {
			pointer = nostr.EventPointer{ID: input}
		} else {
			return nil, nil, fmt.Errorf("failed to decode '%s': %w", input, err)
		}
	}

	return sys.FetchSpecificEvent(ctx, pointer, params)
}

// FetchSpecificEvent tries to get a specific event using a Pointer (EventPointer or EntityPointer).
// It first checks the local store, then queries relays associated with the event or author,
// and finally falls back to general-purpose relays.
func (sys *System) FetchSpecificEvent(
	ctx context.Context,
	pointer nostr.Pointer,
	params FetchSpecificEventParameters,
) (event *nostr.Event, successRelays []string, err error) {
	// this is for deciding what relays will go on nevent and nprofile later
	priorityRelays := make([]string, 0, 8)

	var filter nostr.Filter
	author := ""
	relays := make([]string, 0, 10)
	fallback := make([]string, 0, 10)
	successRelays = make([]string, 0, 10)

	switch v := pointer.(type) {
	case nostr.EventPointer:
		author = v.Author
		filter.IDs = []string{v.ID}
		relays = append(relays, v.Relays...)
		relays = appendUnique(relays, sys.FallbackRelays.Next())
		fallback = append(fallback, sys.JustIDRelays.URLs...)
		fallback = appendUnique(fallback, sys.FallbackRelays.Next())
		priorityRelays = append(priorityRelays, v.Relays...)
	case nostr.EntityPointer:
		author = v.PublicKey
		filter.Authors = []string{v.PublicKey}
		filter.Tags = nostr.TagMap{"d": []string{v.Identifier}}
		filter.Kinds = []int{v.Kind}
		relays = append(relays, v.Relays...)
		relays = appendUnique(relays, sys.FallbackRelays.Next())
		fallback = append(fallback, sys.FallbackRelays.Next(), sys.FallbackRelays.Next())
		priorityRelays = append(priorityRelays, v.Relays...)
	}

	// try to fetch in our internal eventstore first
	if !params.SkipLocalStore {
		if res, _ := sys.StoreRelay.QuerySync(ctx, filter); len(res) != 0 {
			evt := res[0]
			return evt, nil, nil
		}
	}

	if author != "" {
		// fetch relays for author
		authorRelays := sys.FetchOutboxRelays(ctx, author, 3)

		// after that we register these hints as associated with author
		// (we do this after fetching author outbox relays because we are already going to prioritize these hints)
		now := nostr.Now()
		for _, relay := range priorityRelays {
			sys.Hints.Save(author, nostr.NormalizeURL(relay), hints.LastInHint, now)
		}

		// arrange these
		relays = appendUnique(relays, authorRelays...)
		priorityRelays = appendUnique(priorityRelays, authorRelays...)
	}

	var result *nostr.Event
	fetchProfileOnce := sync.Once{}

attempts:
	for _, attempt := range []struct {
		label          string
		relays         []string
		slowWithRelays bool
	}{
		{
			label:  "fetchspecific",
			relays: relays,
			// set this to true if the caller wants relays, so we won't return immediately
			//   but will instead wait a little while to see if more relays respond
			slowWithRelays: params.WithRelays,
		},
		{
			label:          "fetchspecific",
			relays:         fallback,
			slowWithRelays: false,
		},
	} {
		// actually fetch the event here
		countdown := 6.0
		subManyCtx := ctx

		for ie := range sys.Pool.FetchMany(
			subManyCtx,
			attempt.relays,
			filter,
			nostr.WithLabel(attempt.label),
		) {
			fetchProfileOnce.Do(func() {
				go sys.FetchProfileMetadata(ctx, ie.PubKey)
			})

			successRelays = append(successRelays, ie.Relay.URL)
			if result == nil || ie.CreatedAt > result.CreatedAt {
				result = ie.Event
			}

			if !attempt.slowWithRelays {
				break attempts
			}

			countdown = min(countdown-0.5, 1)
		}
	}

	if result == nil {
		return nil, nil, fmt.Errorf("couldn't find this %v", pointer)
	}

	// save stuff in cache and in internal store
	if !params.SkipLocalStore {
		sys.StoreRelay.Publish(ctx, *result)
	}

	// put priority relays first so they get used in nevent and nprofile
	slices.SortFunc(successRelays, func(a, b string) int {
		vpa := slices.Contains(priorityRelays, a)
		vpb := slices.Contains(priorityRelays, b)
		if vpa == vpb {
			return 1
		}
		if vpa && !vpb {
			return 1
		}
		return -1
	})

	return result, successRelays, nil
}
