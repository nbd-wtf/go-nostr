package sdk

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

const eventRelayPrefix = byte('r')

func makeEventRelayKey(eventID []byte) []byte {
	// format: 'r' + first 8 bytes of event ID
	key := make([]byte, 9)
	key[0] = eventRelayPrefix
	copy(key[1:], eventID[:8])
	return key
}

func encodeRelayList(relays []string) []byte {
	totalSize := 0
	for _, relay := range relays {
		totalSize += 2 + len(relay) // 2 bytes for length prefix
	}

	buf := make([]byte, totalSize)
	offset := 0

	for _, relay := range relays {
		binary.LittleEndian.PutUint16(buf[offset:], uint16(len(relay)))
		offset += 2
		copy(buf[offset:], relay)
		offset += len(relay)
	}

	return buf
}

func decodeRelayList(data []byte) []string {
	relays := make([]string, 0)
	offset := 0

	for offset < len(data) {
		if offset+2 > len(data) {
			return nil // malformed
		}

		length := int(binary.LittleEndian.Uint16(data[offset:]))
		offset += 2

		if offset+length > len(data) {
			return nil // malformed
		}

		relay := string(data[offset : offset+length])
		relays = append(relays, relay)
		offset += length
	}

	return relays
}

func (sys *System) trackEventRelayCommon(eventID string, relay string) {
	// decode the event ID hex into bytes
	idBytes, err := hex.DecodeString(eventID)
	if err != nil || len(idBytes) < 8 {
		return
	}

	// get the key for this event
	key := makeEventRelayKey(idBytes)

	// update the relay list atomically
	sys.KVStore.Update(key, func(data []byte) ([]byte, error) {
		var relays []string
		if data != nil {
			relays = decodeRelayList(data)
		} else {
			relays = make([]string, 0, 1)
		}

		// check if relay is already in list
		for _, r := range relays {
			if r == relay {
				return data, nil // no change needed
			}
		}

		// append new relay
		relays = append(relays, relay)
		return encodeRelayList(relays), nil
	})
}

// GetEventRelays returns all known relay URLs that have been seen to carry the given event.
func (sys *System) GetEventRelays(eventID string) ([]string, error) {
	// decode the event ID hex into bytes
	idBytes, err := hex.DecodeString(eventID)
	if err != nil || len(idBytes) < 8 {
		return nil, fmt.Errorf("invalid event id")
	}

	// get the key for this event
	key := makeEventRelayKey(idBytes)

	// get stored relay list
	data, err := sys.KVStore.Get(key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	return decodeRelayList(data), nil
}
