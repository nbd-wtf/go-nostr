package nip02

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

// GetContacts returns a slice containting the contact list of an npub
func GetContacts(npub string, relay string) ([]string, error) {

	ctx := context.Background()

	_, npubHex, _ := nip19.Decode(npub)

	// connect to relay
	r, err := nostr.RelayConnect(ctx, relay)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	// NIP-02 states that "[e]very new contact list that gets published overwrites the past ones, so it should contain all entries."
	// So if we send the relay the Contact List Kind (3) and get the latest answer, that should contain all an npub's contacts.
	now := nostr.Now()
	filter := nostr.Filter{
		Authors: []string{npubHex.(string)},
		Kinds:   []int{nostr.KindContactList},
		Until:   &now,
		Limit:   1,
	}

	e, err := r.QuerySync(ctx, filter)
	if err != nil {
		return nil, err
	}

	// parse out the contact list
	tags := e[0].Tags.GetAll([]string{"p"})
	var contacts []string
	for _, v := range tags {
		contacts = append(contacts, v[1])
	}

	return contacts, nil
}
