package badgerh

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/nbd-wtf/go-nostr"
)

func encodeKey(pubhintkey, relay string) []byte {
	k := make([]byte, 32+len(relay))
	hex.Decode(k[0:32], []byte(pubhintkey))
	copy(k[32:], relay)
	return k
}

func parseKey(k []byte) (pubkey string, relay string) {
	pubkey = hex.EncodeToString(k[0:32])
	relay = string(k[32:])
	return
}

func encodeValue(tss timestamps) []byte {
	v := make([]byte, 16)
	binary.LittleEndian.PutUint32(v[0:], uint32(tss[0]))
	binary.LittleEndian.PutUint32(v[4:], uint32(tss[1]))
	binary.LittleEndian.PutUint32(v[8:], uint32(tss[2]))
	binary.LittleEndian.PutUint32(v[12:], uint32(tss[3]))
	return v
}

func parseValue(v []byte) timestamps {
	return timestamps{
		nostr.Timestamp(binary.LittleEndian.Uint32(v[0:])),
		nostr.Timestamp(binary.LittleEndian.Uint32(v[4:])),
		nostr.Timestamp(binary.LittleEndian.Uint32(v[8:])),
		nostr.Timestamp(binary.LittleEndian.Uint32(v[12:])),
	}
}
