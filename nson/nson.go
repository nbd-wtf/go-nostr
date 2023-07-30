package nson

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"unsafe"

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

const (
	ID_START         = 7
	ID_END           = 7 + 64
	PUBKEY_START     = 83
	PUBKEY_END       = 83 + 64
	SIG_START        = 156
	SIG_END          = 156 + 128
	CREATED_AT_START = 299
	CREATED_AT_END   = 299 + 10

	NSON_STRING_START = 318     // the actual json string for the "nson" field
	NSON_VALUES_START = 318 + 2 // skipping the first byte which delimits the nson size

	NSON_MARKER_START = 309 // this is used just to determine if an event is nson or not
	NSON_MARKER_END   = 317 // it's just the `,"nson":` (including ,": garbage to reduce false positives) part
)

var NotNSON = fmt.Errorf("not nson")

func UnmarshalBytes(data []byte, evt *nostr.Event) (err error) {
	return Unmarshal(unsafe.String(unsafe.SliceData(data), len(data)), evt)
}

// Unmarshal turns a NSON string into a nostr.Event struct
func Unmarshal(data string, evt *nostr.Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to decode nson: %v", r)
		}
	}()

	// check if it's nson
	if data[NSON_MARKER_START:NSON_MARKER_END] != ",\"nson\":" {
		return NotNSON
	}

	// nson values
	nsonSize, nsonDescriptors := parseDescriptors(data)

	// static fields
	evt.ID = data[ID_START:ID_END]
	evt.PubKey = data[PUBKEY_START:PUBKEY_END]
	evt.Sig = data[SIG_START:SIG_END]
	ts, _ := strconv.ParseUint(data[CREATED_AT_START:CREATED_AT_END], 10, 64)
	evt.CreatedAt = nostr.Timestamp(ts)

	// dynamic fields
	// kind
	kindChars := int(nsonDescriptors[0])
	kindStart := NSON_VALUES_START + nsonSize + 9 // len(`","kind":`)
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

	return err
}

func MarshalBytes(evt *nostr.Event) ([]byte, error) {
	v, err := Marshal(evt)
	return unsafe.Slice(unsafe.StringData(v), len(v)), err
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
	base.Grow(NSON_VALUES_START + // everything up to "nson":
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

func parseDescriptors(data string) (int, []byte) {
	nsonSizeBytes, _ := hex.DecodeString(data[NSON_STRING_START:NSON_VALUES_START])
	size := int(nsonSizeBytes[0]) * 2 // number of bytes is given, we x2 because the string is in hex
	values, _ := hex.DecodeString(data[NSON_VALUES_START : NSON_VALUES_START+size])
	return size, values
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
		ne.descriptorsSize, ne.descriptors = parseDescriptors(ne.data)
	}
}

func (ne Event) GetID() string     { return ne.data[ID_START:ID_END] }
func (ne Event) GetPubkey() string { return ne.data[PUBKEY_START:PUBKEY_END] }
func (ne Event) GetSig() string    { return ne.data[SIG_START:SIG_END] }
func (ne Event) GetCreatedAt() nostr.Timestamp {
	ts, _ := strconv.ParseUint(ne.data[CREATED_AT_START:CREATED_AT_END], 10, 64)
	return nostr.Timestamp(ts)
}

func (ne *Event) GetKind() int {
	ne.parseDescriptors()

	kindChars := int(ne.descriptors[0])
	kindStart := NSON_VALUES_START + ne.descriptorsSize + 9 // len(`","kind":`)
	kind, _ := strconv.Atoi(ne.data[kindStart : kindStart+kindChars])

	return kind
}

func (ne *Event) GetContent() string {
	ne.parseDescriptors()

	kindChars := int(ne.descriptors[0])
	kindStart := NSON_VALUES_START + ne.descriptorsSize + 9 // len(`","kind":`)

	contentChars := int(binary.BigEndian.Uint16(ne.descriptors[1:3]))
	contentStart := kindStart + kindChars + 12 // len(`,"content":"`)
	content, _ := strconv.Unquote(`"` + ne.data[contentStart:contentStart+contentChars] + `"`)

	return content
}
