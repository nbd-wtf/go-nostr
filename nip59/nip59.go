package nip59

import (
	"fmt"
	"math/rand"

	"github.com/mailru/easyjson"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip44"
)

// GiftWrap takes a 'rumor', encrypts it with our own key, making a 'seal', then encrypts that with a nonce key and
// signs that (after potentially applying a modify function, which can be nil otherwise), yielding a 'gift-wrap'.
func GiftWrap(
	rumor nostr.Event,
	recipient string,
	encrypt func(plaintext string) (string, error),
	sign func(*nostr.Event) error,
	modify func(*nostr.Event),
) (nostr.Event, error) {
	rumor.Sig = ""

	rumorCiphertext, err := encrypt(rumor.String())
	if err != nil {
		return nostr.Event{}, err
	}

	seal := nostr.Event{
		Kind:      nostr.KindSeal,
		Content:   rumorCiphertext,
		CreatedAt: nostr.Now() - nostr.Timestamp(60*rand.Int63n(600) /* up to 6 hours in the past */),
		Tags:      make(nostr.Tags, 0),
	}
	if err := sign(&seal); err != nil {
		return nostr.Event{}, err
	}

	nonceKey := nostr.GeneratePrivateKey()
	temporaryConversationKey, err := nip44.GenerateConversationKey(recipient, nonceKey)
	if err != nil {
		return nostr.Event{}, err
	}

	sealCiphertext, err := nip44.Encrypt(seal.String(), temporaryConversationKey)
	if err != nil {
		return nostr.Event{}, err
	}

	gw := nostr.Event{
		Kind:      nostr.KindGiftWrap,
		Content:   sealCiphertext,
		CreatedAt: nostr.Now() - nostr.Timestamp(60*rand.Int63n(600) /* up to 6 hours in the past */),
		Tags: nostr.Tags{
			nostr.Tag{"p", recipient},
		},
	}
	if modify != nil {
		modify(&gw)
	}
	if err := gw.Sign(nonceKey); err != nil {
		return nostr.Event{}, err
	}

	return gw, nil
}

func GiftUnwrap(
	gw nostr.Event,
	decrypt func(otherpubkey, ciphertext string) (string, error),
) (rumor nostr.Event, err error) {
	jseal, err := decrypt(gw.PubKey, gw.Content)
	if err != nil {
		return rumor, fmt.Errorf("failed to decrypt seal: %w", err)
	}

	var seal nostr.Event
	err = easyjson.Unmarshal([]byte(jseal), &seal)
	if err != nil {
		return rumor, fmt.Errorf("seal is invalid json: %w", err)
	}

	if ok, _ := seal.CheckSignature(); !ok {
		return rumor, fmt.Errorf("seal signature is invalid")
	}

	jrumor, err := decrypt(seal.PubKey, seal.Content)
	if err != nil {
		return rumor, fmt.Errorf("failed to decrypt rumor: %w", err)
	}

	err = easyjson.Unmarshal([]byte(jrumor), &rumor)
	if err != nil {
		return rumor, fmt.Errorf("rumor is invalid json: %w", err)
	}

	rumor.PubKey = seal.PubKey
	rumor.ID = rumor.GetID()

	return rumor, nil
}
