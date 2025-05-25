package nip19

import (
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

func EncodePointer(pointer nostr.Pointer) string {
	switch v := pointer.(type) {
	case nostr.ProfilePointer:
		if v.Relays == nil {
			res, _ := EncodePublicKey(v.PublicKey)
			return res
		} else {
			res, _ := EncodeProfile(v.PublicKey, v.Relays)
			return res
		}
	case nostr.EventPointer:
		res, _ := EncodeEvent(v.ID, v.Relays, v.Author)
		return res
	case nostr.EntityPointer:
		res, _ := EncodeEntity(v.PublicKey, v.Kind, v.Identifier, v.Relays)
		return res
	}
	return ""
}

func ToPointer(code string) (nostr.Pointer, error) {
	prefix, data, err := Decode(code)
	if err != nil {
		return nil, err
	}

	switch prefix {
	case "npub":
		return nostr.ProfilePointer{PublicKey: data.(string)}, nil
	case "nprofile":
		return data.(nostr.ProfilePointer), nil
	case "nevent":
		return data.(nostr.EventPointer), nil
	case "note":
		return nostr.EventPointer{ID: data.(string)}, nil
	case "naddr":
		return data.(nostr.EntityPointer), nil
	default:
		return nil, fmt.Errorf("unexpected prefix '%s' to '%s'", prefix, code)
	}
}
