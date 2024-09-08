package binary

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

// Deprecated -- the encoding used here is not very elegant, we'll have a better binary format later.
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
			item := string(data[itemStart : itemStart+itemSize])
			tag[i] = item
			curr = itemStart + itemSize
		}
		evt.Tags[t] = tag
	}

	return err
}

// Deprecated -- the encoding used here is not very elegant, we'll have a better binary format later.
func Marshal(evt *nostr.Event) ([]byte, error) {
	content := []byte(evt.Content)
	buf := make([]byte, 32+32+64+4+2+2+len(content)+65536+len(evt.Tags)*40 /* blergh */)

	hex.Decode(buf[0:32], []byte(evt.ID))
	hex.Decode(buf[32:64], []byte(evt.PubKey))
	hex.Decode(buf[64:128], []byte(evt.Sig))

	if evt.CreatedAt > MaxCreatedAt {
		return nil, fmt.Errorf("created_at is too big: %d, max is %d", evt.CreatedAt, MaxCreatedAt)
	}
	binary.BigEndian.PutUint32(buf[128:132], uint32(evt.CreatedAt))

	if evt.Kind > MaxKind {
		return nil, fmt.Errorf("kind is too big: %d, max is %d", evt.Kind, MaxKind)
	}
	binary.BigEndian.PutUint16(buf[132:134], uint16(evt.Kind))

	if contentLength := len(content); contentLength > MaxContentSize {
		return nil, fmt.Errorf("content is too large: %d, max is %d", contentLength, MaxContentSize)
	} else {
		binary.BigEndian.PutUint16(buf[134:136], uint16(contentLength))
	}
	copy(buf[136:], content)

	if tagCount := len(evt.Tags); tagCount > MaxTagCount {
		return nil, fmt.Errorf("can't encode too many tags: %d, max is %d", tagCount, MaxTagCount)
	} else {
		binary.BigEndian.PutUint16(buf[136+len(content):136+len(content)+2], uint16(tagCount))
	}

	buf = buf[0 : 136+len(content)+2]

	for _, tag := range evt.Tags {
		if itemCount := len(tag); itemCount > MaxTagItemCount {
			return nil, fmt.Errorf("can't encode a tag with so many items: %d, max is %d", itemCount, MaxTagItemCount)
		} else {
			buf = append(buf, uint8(itemCount))
		}
		for _, item := range tag {
			itemb := []byte(item)
			itemSize := len(itemb)
			if itemSize > MaxTagItemSize {
				return nil, fmt.Errorf("tag item is too large: %d, max is %d", itemSize, MaxTagItemSize)
			}
			buf = binary.BigEndian.AppendUint16(buf, uint16(itemSize))
			buf = append(buf, itemb...)
			buf = append(buf, 0)
		}
	}
	return buf, nil
}
