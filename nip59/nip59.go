package nip59

import (
	"fmt"
	"math/rand"

	"github.com/mailru/easyjson"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip44"
)

// Seal takes a rumor, encrypts it and returns an unsigned 'seal' event, the 'seal' must be signed
// afterwards.
func Seal(rumor nostr.Event, encrypt func(string) (string, error)) (nostr.Event, error) {
	rumor.Sig = ""
	ciphertext, err := encrypt(rumor.String())

	return nostr.Event{
		Kind:      13,
		Content:   ciphertext,
		CreatedAt: nostr.Now() - nostr.Timestamp(60*rand.Int63n(600) /* up to 6 hours in the past */),
		Tags:      make(nostr.Tags, 0),
	}, err
}

// Takes a signed 'seal' and gift-wraps it using a random key, returns it signed.
//
// modify is a function that takes the gift-wrap before signing, can be used to apply
// NIP-13 PoW or other things, otherwise can be nil.
func GiftWrap(seal nostr.Event, recipientPublicKey string, modify func(*nostr.Event)) (nostr.Event, error) {
	nonceKey := nostr.GeneratePrivateKey()
	temporaryConversationKey, err := nip44.GenerateConversationKey(recipientPublicKey, nonceKey)
	if err != nil {
		return nostr.Event{}, err
	}

	ciphertext, err := nip44.Encrypt(seal.String(), temporaryConversationKey, nil)
	if err != nil {
		return nostr.Event{}, err
	}

	gw := nostr.Event{
		Kind:      1059,
		Content:   ciphertext,
		CreatedAt: nostr.Now() - nostr.Timestamp(60*rand.Int63n(600) /* up to 6 hours in the past */),
		Tags: nostr.Tags{
			nostr.Tag{"p", recipientPublicKey},
		},
	}

	// apply POW if necessary
	if modify != nil {
		modify(&gw)
	}

	err = gw.Sign(nonceKey)
	return gw, nil
}

func GiftUnwrap(gw nostr.Event, decrypt func(string) (string, error)) (seal nostr.Event, err error) {
	jevt, err := decrypt(gw.Content)
	if err != nil {
		return seal, err
	}

	err = easyjson.Unmarshal([]byte(jevt), &seal)
	if err != nil {
		return seal, err
	}

	if ok, _ := seal.CheckSignature(); !ok {
		return seal, fmt.Errorf("seal signature is invalid")
	}

	return seal, nil
}

func Unseal(seal nostr.Event, decrypt func(string) (string, error)) (rumor nostr.Event, err error) {
	jevt, err := decrypt(seal.Content)
	if err != nil {
		return rumor, err
	}

	err = easyjson.Unmarshal([]byte(jevt), &rumor)
	if err != nil {
		return rumor, err
	}

	rumor.PubKey = seal.PubKey
	return rumor, nil
}
