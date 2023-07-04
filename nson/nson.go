package nson

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

/*
           nson size
             kind chars
               content chars
                   number of tags (let's say it's two)
                     number of items on the first tag (let's say it's three)
                       number of chars on the first item
                           number of chars on the second item
                               number of chars on the third item
                                   number of items on the second tag (let's say it's two)
                                     number of chars on the first item
                                         number of chars on the second item
   "nson":"xxkkccccttnn111122223333nn11112222"
*/

// Unmarshal turns a NSON string into a nostr.Event struct
func Unmarshal(data string) (evt *nostr.Event, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to decode nson: %v", r)
		}
	}()

	// check if it's nson
	if data[311:315] != "nson" {
		return nil, fmt.Errorf("not nson")
	}

	// nson values
	nsonSizeBytes, _ := hex.DecodeString(data[318 : 318+2])
	nsonSize := int(nsonSizeBytes[0]) * 2 // number of bytes is given, we x2 because the string is in hex
	nsonDescriptors, _ := hex.DecodeString(data[320 : 320+nsonSize])

	evt = &nostr.Event{}

	// static fields
	evt.ID = data[7 : 7+64]
	evt.PubKey = data[83 : 83+64]
	evt.Sig = data[156 : 156+128]
	ts, _ := strconv.ParseInt(data[299:299+10], 10, 64)
	evt.CreatedAt = nostr.Timestamp(ts)

	// dynamic fields
	// kind
	kindChars := int(nsonDescriptors[0])
	kindStart := 320 + nsonSize + 9 // len(`","kind":`)
	evt.Kind, _ = strconv.Atoi(data[kindStart : kindStart+kindChars])

	// content
	contentChars := int(binary.BigEndian.Uint16(nsonDescriptors[1:3]))
	contentStart := kindStart + kindChars + 12 // len(`,"content":"`)
	evt.Content, _ = strconv.Unquote(data[contentStart-1 : contentStart+contentChars+1])

	// tags
	nTags := int(nsonDescriptors[3])
	evt.Tags = make(nostr.Tags, nTags)
	tagsStart := contentStart + contentChars + 9 // len(`","tags":`)

	nsonIndex := 3
	tagsIndex := tagsStart
	for t := 0; t < nTags; t++ {
		nsonIndex++
		tagsIndex += 1 // len(`[`) or len(`,`)
		nItems := int(nsonDescriptors[nsonIndex])
		tag := make(nostr.Tag, nItems)
		for n := 0; n < nItems; n++ {
			nsonIndex++
			itemStart := tagsIndex + 2 // len(`["`) or len(`,"`)
			itemChars := int(binary.BigEndian.Uint16(nsonDescriptors[nsonIndex:]))
			nsonIndex++
			tag[n], _ = strconv.Unquote(data[itemStart-1 : itemStart+itemChars+1])
			tagsIndex = itemStart + itemChars + 1 // len(`"`)
		}
		tagsIndex += 1 // len(`]`)
		evt.Tags[t] = tag
	}

	return evt, err
}

func Marshal(evt *nostr.Event) (string, error) {
	// start building the nson descriptors (without the first byte that represents the nson size)
	nsonBuf := make([]byte, 256)

	// build the tags
	nTags := len(evt.Tags)
	nsonBuf[3] = uint8(nTags)
	nsonIndex := 3 // start here

	tagBuilder := strings.Builder{}
	tagBuilder.Grow(1000) // a guess
	tagBuilder.WriteString(`[`)
	for t, tag := range evt.Tags {
		nItems := len(tag)
		nsonIndex++
		nsonBuf[nsonIndex] = uint8(nItems)

		tagBuilder.WriteString(`[`)
		for i, item := range tag {
			v := strconv.Quote(item)
			nsonIndex++
			binary.BigEndian.PutUint16(nsonBuf[nsonIndex:], uint16(len(v)-2))
			nsonIndex++
			tagBuilder.WriteString(v)
			if nItems > i+1 {
				tagBuilder.WriteString(`,`)
			}
		}
		tagBuilder.WriteString(`]`)
		if nTags > t+1 {
			tagBuilder.WriteString(`,`)
		}
	}
	tagBuilder.WriteString(`]}`)
	nsonBuf = nsonBuf[0 : nsonIndex+1]

	kind := strconv.Itoa(evt.Kind)
	kindChars := len(kind)
	nsonBuf[0] = uint8(kindChars)

	content := strconv.Quote(evt.Content)
	contentChars := len(content) - 2
	binary.BigEndian.PutUint16(nsonBuf[1:3], uint16(contentChars))

	// actually build the json
	base := strings.Builder{}
	base.Grow(320 + // everything up to "nson":
		2 + len(nsonBuf)*2 + // nson
		9 + kindChars + // kind and its label
		12 + contentChars + // content and its label
		9 + tagBuilder.Len() + // tags and its label
		2, // the end
	)
	base.WriteString(`{"id":"` + evt.ID + `","pubkey":"` + evt.PubKey + `","sig":"` + evt.Sig + `","created_at":` + strconv.FormatInt(int64(evt.CreatedAt), 10) + `,"nson":"`)

	nsonSizeBytes := len(nsonBuf)
	if nsonSizeBytes > 255 {
		return "", fmt.Errorf("can't encode to nson, there are too many tags or tag items")
	}
	base.WriteString(hex.EncodeToString([]byte{uint8(nsonSizeBytes)})) // nson size (bytes)

	base.WriteString(hex.EncodeToString(nsonBuf)) // nson descriptors
	base.WriteString(`","kind":` + kind + `,"content":` + content + `,"tags":`)
	base.WriteString(tagBuilder.String() /* includes the end */)

	return base.String(), nil
}

// A nson.Event is basically a wrapper over the string that makes it easy to get each event property (except tags).
type Event struct {
	data string

	descriptorsSize int
	descriptors     []byte
}

func New(nsonText string) Event {
	return Event{data: nsonText}
}

func (ne *Event) parseDescriptors() {
	if ne.descriptors == nil {
		nsonSizeBytes, _ := hex.DecodeString(ne.data[318 : 318+2])
		ne.descriptorsSize = int(nsonSizeBytes[0])
		ne.descriptors, _ = hex.DecodeString(ne.data[320 : 320+ne.descriptorsSize])
	}
}

func (ne *Event) parseKind() {
	if ne.descriptors == nil {
		nsonSizeBytes, _ := hex.DecodeString(ne.data[318 : 318+2])
		ne.descriptorsSize = int(nsonSizeBytes[0])
		ne.descriptors, _ = hex.DecodeString(ne.data[320 : 320+ne.descriptorsSize])
	}
}

func (ne Event) GetID() string     { return ne.data[7 : 7+64] }
func (ne Event) GetPubkey() string { return ne.data[83 : 83+64] }
func (ne Event) GetSig() string    { return ne.data[156 : 156+128] }
func (ne Event) GetCreatedAt() nostr.Timestamp {
	ts, _ := strconv.ParseInt(ne.data[299:299+10], 10, 64)
	return nostr.Timestamp(ts)
}

func (ne *Event) GetKind() int {
	ne.parseDescriptors()

	kindChars := int(ne.descriptors[0])
	kindStart := 320 + ne.descriptorsSize + 9 // len(`","kind":`)
	kind, _ := strconv.Atoi(ne.data[kindStart : kindStart+kindChars])

	return kind
}

func (ne *Event) GetContent() string {
	ne.parseDescriptors()

	kindChars := int(ne.descriptors[0])
	kindStart := 320 + ne.descriptorsSize + 9 // len(`","kind":`)

	contentChars := int(binary.BigEndian.Uint16(ne.descriptors[1:3]))
	contentStart := kindStart + kindChars + 12 // len(`,"content":"`)
	content, _ := strconv.Unquote(`"` + ne.data[contentStart:contentStart+contentChars] + `"`)

	return content
}
