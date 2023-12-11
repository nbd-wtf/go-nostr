package binary

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

func Unmarshal(data []byte, evt *nostr.Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to decode binary: %v", r)
		}
	}()

	evt.ID = hex.EncodeToString(data[0:32])
	evt.PubKey = hex.EncodeToString(data[32:64])
	evt.Sig = hex.EncodeToString(data[64:128])
	evt.CreatedAt = nostr.Timestamp(binary.BigEndian.Uint32(data[128:132]))
	evt.Kind = int(binary.BigEndian.Uint16(data[132:134]))
	contentLength := int(binary.BigEndian.Uint16(data[134:136]))
	evt.Content = string(data[136 : 136+contentLength])

	curr := 136 + contentLength

	nTags := binary.BigEndian.Uint16(data[curr : curr+2])
	curr++
	evt.Tags = make(nostr.Tags, nTags)

	for t := range evt.Tags {
		curr++
		nItems := int(data[curr])
		tag := make(nostr.Tag, nItems)
		for i := range tag {
			curr = curr + 1
			itemSize := int(binary.BigEndian.Uint16(data[curr : curr+2]))
			itemStart := curr + 2
			itemEnd := itemStart + itemSize
			item := string(data[itemStart:itemEnd])
			tag[i] = item
			curr = itemEnd
		}
		evt.Tags[t] = tag
	}

	return err
}

func Marshal(evt *nostr.Event) ([]byte, error) {
	content := []byte(evt.Content)
	buf := make([]byte, 32+32+64+4+2+2+len(content)+65536 /* blergh */)

	hex.Decode(buf[0:32], []byte(evt.ID))
	hex.Decode(buf[32:64], []byte(evt.PubKey))
	hex.Decode(buf[64:128], []byte(evt.Sig))

	binary.BigEndian.PutUint32(buf[128:132], uint32(evt.CreatedAt))
	binary.BigEndian.PutUint16(buf[132:134], uint16(evt.Kind))
	binary.BigEndian.PutUint16(buf[134:136], uint16(len(content)))
	copy(buf[136:], content)

	curr := 136 + len(content)

	binary.BigEndian.PutUint16(buf[curr:curr+2], uint16(len(evt.Tags)))
	curr++

	for _, tag := range evt.Tags {
		curr++
		buf[curr] = uint8(len(tag))
		for _, item := range tag {
			curr++
			itemb := []byte(item)
			itemSize := len(itemb)
			binary.BigEndian.PutUint16(buf[curr:curr+2], uint16(itemSize))
			itemEnd := curr + 2 + itemSize
			copy(buf[curr+2:itemEnd], itemb)
			curr = itemEnd
		}
	}
	buf = buf[0 : curr+1]
	return buf, nil
}
