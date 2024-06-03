package binary

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/nbd-wtf/go-nostr"
)

// Deprecated -- the encoding used here is not very elegant, we'll have a better binary format later.
func UnmarshalBinary(data []byte, evt *Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to decode leaner: %v", r)
		}
	}()

	copy(evt.ID[:], data[0:32])
	copy(evt.PubKey[:], data[32:64])
	copy(evt.Sig[:], data[64:128])
	evt.CreatedAt = nostr.Timestamp(binary.BigEndian.Uint32(data[128:132]))
	evt.Kind = binary.BigEndian.Uint16(data[132:134])
	contentLength := int(binary.BigEndian.Uint16(data[134:136]))
	evt.Content = unsafe.String(&data[136], contentLength)

	curr := 136 + contentLength

	nTags := binary.BigEndian.Uint16(data[curr : curr+2])
	curr++
	evt.Tags = make(nostr.Tags, nTags)

	for t := range evt.Tags {
		curr = curr + 1
		nItems := int(data[curr])
		tag := make(nostr.Tag, nItems)
		for i := range tag {
			curr = curr + 1
			itemSize := int(binary.BigEndian.Uint16(data[curr : curr+2]))
			itemStart := curr + 2
			item := unsafe.String(&data[itemStart], itemSize)
			tag[i] = item
			curr = itemStart + itemSize
		}
		evt.Tags[t] = tag
	}

	return err
}

// Deprecated -- the encoding used here is not very elegant, we'll have a better binary format later.
func MarshalBinary(evt *Event) []byte {
	content := []byte(evt.Content)
	buf := make([]byte, 32+32+64+4+2+2+len(content)+65536 /* blergh */)
	copy(buf[0:32], evt.ID[:])
	copy(buf[32:64], evt.PubKey[:])
	copy(buf[64:128], evt.Sig[:])
	binary.BigEndian.PutUint32(buf[128:132], uint32(evt.CreatedAt))
	binary.BigEndian.PutUint16(buf[132:134], evt.Kind)
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
	return buf
}
