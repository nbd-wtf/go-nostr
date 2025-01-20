package sdk

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/nbd-wtf/go-nostr"
)

var kvStoreLastFetchPrefix = byte('f')

func makeLastFetchKey(kind int, pubkey string) []byte {
	buf := make([]byte, 1+5+32)
	buf[0] = kvStoreLastFetchPrefix
	binary.LittleEndian.PutUint32(buf[1:], uint32(kind))
	hex.Decode(buf[5:], []byte(pubkey))
	return buf
}

// encodeTimestamp encodes a unix timestamp as 4 bytes
func encodeTimestamp(t nostr.Timestamp) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(t))
	return b
}

// decodeTimestamp decodes a 4-byte timestamp into unix seconds
func decodeTimestamp(b []byte) nostr.Timestamp {
	return nostr.Timestamp(binary.BigEndian.Uint32(b))
}
