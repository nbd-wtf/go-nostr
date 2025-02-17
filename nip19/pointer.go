package nip19

import "github.com/nbd-wtf/go-nostr"

func EncodePointer(pointer nostr.Pointer) string {
	switch v := pointer.(type) {
	case nostr.ProfilePointer:
		res, _ := EncodeProfile(v.PublicKey, v.Relays)
		return res
	case nostr.EventPointer:
		res, _ := EncodeEvent(v.ID, v.Relays, v.Author)
		return res
	case nostr.EntityPointer:
		res, _ := EncodeEntity(v.PublicKey, v.Kind, v.Identifier, v.Relays)
		return res
	}
	return ""
}
