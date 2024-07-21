package negentropy

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"os"

	"github.com/nbd-wtf/go-nostr"
)

const (
	protocolVersion byte = 0x61 // version 1
	maxTimestamp         = nostr.Timestamp(math.MaxInt64)
)

type Negentropy struct {
	storage          Storage
	frameSizeLimit   uint64
	idSize           int // in bytes
	IsInitiator      bool
	lastTimestampIn  nostr.Timestamp
	lastTimestampOut nostr.Timestamp
}

func NewNegentropy(storage Storage, frameSizeLimit uint64, IDSize int) (*Negentropy, error) {
	if frameSizeLimit != 0 && frameSizeLimit < 4096 {
		return nil, fmt.Errorf("frameSizeLimit too small")
	}
	if IDSize > 32 {
		return nil, fmt.Errorf("id size cannot be more than 32, got %d", IDSize)
	}
	return &Negentropy{
		storage:        storage,
		frameSizeLimit: frameSizeLimit,
		idSize:         IDSize,
	}, nil
}

func (n *Negentropy) Insert(evt *nostr.Event) {
	err := n.storage.Insert(evt.CreatedAt, evt.ID[0:n.idSize*2])
	if err != nil {
		panic(err)
	}
}

func (n *Negentropy) Initiate() ([]byte, error) {
	if n.IsInitiator {
		return []byte{}, fmt.Errorf("already initiated")
	}
	n.IsInitiator = true

	output := make([]byte, 1, 1+n.storage.Size()*n.idSize)
	output[0] = protocolVersion
	n.SplitRange(0, n.storage.Size(), Bound{Item: Item{Timestamp: maxTimestamp}}, &output)

	return output, nil
}

func (n *Negentropy) Reconcile(query []byte) (output []byte, haveIds []string, needIds []string, err error) {
	if n.IsInitiator {
		return nil, nil, nil, fmt.Errorf("initiator not asking for have/need IDs")
	}

	reader := bytes.NewReader(query)
	haveIds = make([]string, 0, 100)
	needIds = make([]string, 0, 100)

	output, err = n.ReconcileAux(reader, &haveIds, &needIds)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(output) == 1 && n.IsInitiator {
		return nil, haveIds, needIds, nil
	}

	return output, haveIds, needIds, nil
}

func (n *Negentropy) ReconcileAux(reader *bytes.Reader, haveIds, needIds *[]string) ([]byte, error) {
	n.lastTimestampIn, n.lastTimestampOut = 0, 0 // Reset for each message

	var fullOutput []byte
	fullOutput = append(fullOutput, protocolVersion)

	pv, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	if pv < 0x60 || pv > 0x6F {
		return nil, fmt.Errorf("invalid negentropy protocol version byte")
	}
	if pv != protocolVersion {
		if n.IsInitiator {
			return nil, fmt.Errorf("unsupported negentropy protocol version requested")
		}
		return fullOutput, nil
	}

	var prevBound Bound
	prevIndex := 0
	skip := false

	for reader.Len() > 0 {
		var o []byte

		doSkip := func() {
			if skip {
				skip = false
				encodedBound, err := n.encodeBound(prevBound) // Handle error appropriately
				if err != nil {
					panic(err)
				}
				o = append(o, encodedBound...)
				o = append(o, encodeVarInt(SkipMode)...)
			}
		}

		currBound, err := n.DecodeBound(reader)
		if err != nil {
			return nil, err
		}
		modeVal, err := decodeVarInt(reader)
		if err != nil {
			return nil, err
		}
		mode := Mode(modeVal)

		lower := prevIndex
		upper, err := n.storage.FindLowerBound(prevIndex, n.storage.Size(), currBound)
		if err != nil {
			return nil, err
		}

		switch mode {
		case SkipMode:
			skip = true

		case FingerprintMode:
			theirFingerprint := make([]byte, FingerprintSize)
			_, err := reader.Read(theirFingerprint)
			if err != nil {
				return nil, err
			}
			ourFingerprint, err := n.storage.Fingerprint(lower, upper)
			if err != nil {
				return nil, err // Handle the error appropriately
			}

			if !bytes.Equal(theirFingerprint, ourFingerprint.Buf[:]) {
				doSkip()
				n.SplitRange(lower, upper, currBound, &o)
			} else {
				skip = true
			}

		case IdListMode:
			numIds64, err := decodeVarInt(reader)
			if err != nil {
				return nil, err
			}
			numIds := int(numIds64)

			theirElems := make(map[string]struct{})
			idb := make([]byte, n.idSize)
			for i := 0; i < numIds; i++ {
				_, err := reader.Read(idb)
				if err != nil {
					return nil, err
				}
				theirElems[hex.EncodeToString(idb)] = struct{}{}
			}

			n.storage.Iterate(lower, upper, func(item Item, _ int) bool {
				k := item.ID
				if _, exists := theirElems[k]; !exists {
					if n.IsInitiator {
						*haveIds = append(*haveIds, k)
					}
				} else {
					delete(theirElems, k)
				}
				return true
			})

			if n.IsInitiator {
				skip = true

				for k := range theirElems {
					*needIds = append(*needIds, k)
				}
			} else {
				doSkip()

				responseIds := make([]byte, 0, n.idSize*n.storage.Size())
				responseIdsPtr := &responseIds
				numResponseIds := 0
				endBound := currBound

				n.storage.Iterate(lower, upper, func(item Item, index int) bool {
					if n.ExceededFrameSizeLimit(len(fullOutput) + len(*responseIdsPtr)) {
						endBound = Bound{item}
						upper = index
						return false
					}

					id, _ := hex.DecodeString(item.ID)
					*responseIdsPtr = append(*responseIdsPtr, id...)
					numResponseIds++
					return true
				})

				encodedBound, err := n.encodeBound(endBound)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					panic(err)
				}

				o = append(o, encodedBound...)
				o = append(o, encodeVarInt(IdListMode)...)
				o = append(o, encodeVarInt(numResponseIds)...)
				o = append(o, responseIds...)

				fullOutput = append(fullOutput, o...)
				o = []byte{}
			}

		default:
			return nil, fmt.Errorf("unexpected mode %d", mode)
		}

		// Check if the frame size limit is exceeded
		if n.ExceededFrameSizeLimit(len(fullOutput) + len(o)) {
			// Frame size limit exceeded, handle by encoding a boundary and fingerprint for the remaining range
			remainingFingerprint, err := n.storage.Fingerprint(upper, n.storage.Size())
			if err != nil {
				panic(err)
			}

			encodedBound, err := n.encodeBound(Bound{Item: Item{Timestamp: maxTimestamp}})
			if err != nil {
				panic(err)
			}
			fullOutput = append(fullOutput, encodedBound...)
			fullOutput = append(fullOutput, encodeVarInt(FingerprintMode)...)
			fullOutput = append(fullOutput, remainingFingerprint.SV()...)

			break // Stop processing further
		} else {
			// Append the constructed output for this iteration
			fullOutput = append(fullOutput, o...)
		}

		prevIndex = upper
		prevBound = currBound
	}

	return fullOutput, nil
}

func (n *Negentropy) SplitRange(lower, upper int, upperBound Bound, output *[]byte) {
	numElems := upper - lower
	const buckets = 16

	if numElems < buckets*2 {
		boundEncoded, err := n.encodeBound(upperBound)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			panic(err)
		}
		*output = append(*output, boundEncoded...)
		*output = append(*output, encodeVarInt(IdListMode)...)
		*output = append(*output, encodeVarInt(numElems)...)

		n.storage.Iterate(lower, upper, func(item Item, _ int) bool {
			id, _ := hex.DecodeString(item.ID)
			*output = append(*output, id...)
			return true
		})
	} else {
		itemsPerBucket := numElems / buckets
		bucketsWithExtra := numElems % buckets
		curr := lower

		for i := 0; i < buckets; i++ {
			bucketSize := itemsPerBucket
			if i < bucketsWithExtra {
				bucketSize++
			}
			ourFingerprint, err := n.storage.Fingerprint(curr, curr+bucketSize)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				panic(err)
			}

			curr += bucketSize

			var nextBound Bound
			if curr == upper {
				nextBound = upperBound
			} else {
				var prevItem, currItem Item

				n.storage.Iterate(curr-1, curr+1, func(item Item, index int) bool {
					if index == curr-1 {
						prevItem = item
					} else {
						currItem = item
					}
					return true
				})

				minBound := n.getMinimalBound(prevItem, currItem)
				nextBound = minBound
			}

			boundEncoded, err := n.encodeBound(nextBound)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				panic(err)
			}
			*output = append(*output, boundEncoded...)
			*output = append(*output, encodeVarInt(FingerprintMode)...)
			*output = append(*output, ourFingerprint.SV()...)
		}
	}
}

func (n *Negentropy) ExceededFrameSizeLimit(size int) bool {
	return n.frameSizeLimit != 0 && size > int(n.frameSizeLimit)-200
}

// Decoding

func (n *Negentropy) DecodeTimestampIn(reader *bytes.Reader) (nostr.Timestamp, error) {
	t, err := decodeVarInt(reader)
	if err != nil {
		return 0, err
	}

	timestamp := nostr.Timestamp(t)
	if timestamp == 0 {
		timestamp = maxTimestamp
	} else {
		timestamp--
	}

	timestamp += n.lastTimestampIn
	if timestamp < n.lastTimestampIn { // Check for overflow
		timestamp = maxTimestamp
	}
	n.lastTimestampIn = timestamp
	return timestamp, nil
}

func (n *Negentropy) DecodeBound(reader *bytes.Reader) (Bound, error) {
	timestamp, err := n.DecodeTimestampIn(reader)
	if err != nil {
		return Bound{}, err
	}

	length, err := decodeVarInt(reader)
	if err != nil {
		return Bound{}, err
	}

	id := make([]byte, length)
	if _, err = reader.Read(id); err != nil {
		return Bound{}, err
	}

	return Bound{Item{timestamp, hex.EncodeToString(id)}}, nil
}

// Encoding

// encodeTimestampOut encodes the given timestamp.
func (n *Negentropy) encodeTimestampOut(timestamp nostr.Timestamp) []byte {
	if timestamp == maxTimestamp {
		n.lastTimestampOut = maxTimestamp
		return encodeVarInt(0)
	}
	temp := timestamp
	timestamp -= n.lastTimestampOut
	n.lastTimestampOut = temp
	return encodeVarInt(int(timestamp + 1))
}

func (n *Negentropy) encodeBound(bound Bound) ([]byte, error) {
	var output []byte

	t := n.encodeTimestampOut(bound.Item.Timestamp)
	idlen := encodeVarInt(n.idSize)
	output = append(output, t...)
	output = append(output, idlen...)
	id := bound.Item.ID

	output = append(output, id...)
	return output, nil
}

func (n *Negentropy) getMinimalBound(prev, curr Item) Bound {
	if curr.Timestamp != prev.Timestamp {
		return Bound{Item{curr.Timestamp, ""}}
	}

	sharedPrefixBytes := 0

	for i := 0; i < n.idSize; i++ {
		if curr.ID[i:i+2] != prev.ID[i:i+2] {
			break
		}
		sharedPrefixBytes++
	}

	// sharedPrefixBytes + 1 to include the first differing byte, or the entire ID if identical.
	return Bound{Item{curr.Timestamp, curr.ID[:sharedPrefixBytes*2+1]}}
}
