package negentropy

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

const (
	ProtocolVersion   byte   = 0x61 // Version 1
	MaxU64            uint64 = ^uint64(0)
	FrameSizeMinLimit uint64 = 4096
)

type Negentropy struct {
	Storage
	frameSizeLimit   uint64
	IsInitiator      bool
	lastTimestampIn  uint64
	lastTimestampOut uint64
}

func NewNegentropy(storage Storage, frameSizeLimit uint64) (*Negentropy, error) {
	if frameSizeLimit != 0 && frameSizeLimit < 4096 {
		return nil, errors.New("frameSizeLimit too small")
	}
	return &Negentropy{
		Storage:        storage,
		frameSizeLimit: frameSizeLimit,
	}, nil
}

func (n *Negentropy) Initiate() ([]byte, error) {
	if n.IsInitiator {
		return []byte{}, errors.New("already initiated")
	}
	n.IsInitiator = true

	output := make([]byte, 1, 1+n.Storage.Size()*IDSize)
	output[0] = ProtocolVersion
	n.SplitRange(0, n.Storage.Size(), Bound{Item: Item{Timestamp: MaxU64}}, &output)

	return output, nil
}

func (n *Negentropy) Reconcile(query []byte) ([]byte, error) {
	if n.IsInitiator {
		return []byte{}, errors.New("initiator not asking for have/need IDs")
	}
	var haveIds, needIds []string

	output, err := n.ReconcileAux(query, &haveIds, &needIds)
	if err != nil {
		return nil, err
	}

	if len(output) == 1 && n.IsInitiator {
		return nil, nil
	}

	return output, nil
}

// ReconcileWithIDs when IDs are expected to be returned.
func (n *Negentropy) ReconcileWithIDs(query []byte, haveIds, needIds *[]string) ([]byte, error) {
	if !n.IsInitiator {
		return nil, errors.New("non-initiator asking for have/need IDs")
	}

	output, err := n.ReconcileAux(query, haveIds, needIds)
	if err != nil {
		return nil, err
	}
	if len(output) == 1 {
		// Assuming an empty string is a special case indicating a condition similar to std::nullopt
		return nil, nil
	}

	return output, nil
}

func (n *Negentropy) ReconcileAux(query []byte, haveIds, needIds *[]string) ([]byte, error) {
	n.lastTimestampIn, n.lastTimestampOut = 0, 0 // Reset for each message

	var fullOutput []byte
	fullOutput = append(fullOutput, ProtocolVersion)

	protocolVersion, err := getByte(&query)
	if err != nil {
		return nil, err
	}
	if protocolVersion < 0x60 || protocolVersion > 0x6F {
		return nil, errors.New("invalid negentropy protocol version byte")
	}
	if protocolVersion != ProtocolVersion {
		if n.IsInitiator {
			return nil, errors.New("unsupported negentropy protocol version requested")
		}
		return fullOutput, nil
	}

	storageSize := n.Storage.Size()
	var prevBound Bound
	prevIndex := 0
	skip := false

	// Convert the loop to process the query until it's consumed
	for len(query) > 0 {
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

		currBound, err := n.DecodeBound(&query)
		if err != nil {
			return nil, err
		}
		modeVal, err := decodeVarInt(&query)
		if err != nil {
			return nil, err
		}
		mode := Mode(modeVal)

		lower := prevIndex
		upper, err := n.Storage.FindLowerBound(prevIndex, storageSize, currBound)
		if err != nil {
			return nil, err
		}

		switch mode {
		case SkipMode:
			skip = true

		case FingerprintMode:
			theirFingerprint, err := getBytes(&query, FingerprintSize)
			if err != nil {
				return nil, err
			}
			ourFingerprint, err := n.Storage.Fingerprint(lower, upper)
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
			numIds, err := decodeVarInt(&query)
			if err != nil {
				return nil, err
			}

			theirElems := make(map[string]struct{})
			for i := 0; i < numIds; i++ {
				e, err := getBytes(&query, IDSize)
				if err != nil {
					return nil, err
				}
				theirElems[hex.EncodeToString(e)] = struct{}{}
			}

			n.Storage.Iterate(lower, upper, func(item Item, _ int) bool {
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

				responseIds := make([]byte, 0, IDSize*n.Storage.Size())
				responseIdsPtr := &responseIds
				numResponseIds := 0
				endBound := currBound

				n.Storage.Iterate(lower, upper, func(item Item, index int) bool {
					if n.ExceededFrameSizeLimit(len(fullOutput) + len(*responseIdsPtr)) {
						endBound = *NewBoundWithItem(item)
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
			return nil, errors.New("unexpected mode")
		}

		// Check if the frame size limit is exceeded
		if n.ExceededFrameSizeLimit(len(fullOutput) + len(o)) {
			// Frame size limit exceeded, handle by encoding a boundary and fingerprint for the remaining range
			remainingFingerprint, err := n.Storage.Fingerprint(upper, storageSize)
			if err != nil {
				panic(err)
			}

			encodedBound, err := n.encodeBound(Bound{Item: Item{Timestamp: MaxU64}})
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
	const Buckets = 16

	if numElems < Buckets*2 {
		boundEncoded, err := n.encodeBound(upperBound)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			panic(err)
		}
		*output = append(*output, boundEncoded...)
		*output = append(*output, encodeVarInt(IdListMode)...)
		*output = append(*output, encodeVarInt(numElems)...)

		n.Storage.Iterate(lower, upper, func(item Item, _ int) bool {
			id, _ := hex.DecodeString(item.ID)
			*output = append(*output, id...)
			return true
		})
	} else {
		itemsPerBucket := numElems / Buckets
		bucketsWithExtra := numElems % Buckets
		curr := lower

		for i := 0; i < Buckets; i++ {
			bucketSize := itemsPerBucket
			if i < bucketsWithExtra {
				bucketSize++
			}
			ourFingerprint, err := n.Storage.Fingerprint(curr, curr+bucketSize)
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

				n.Storage.Iterate(curr-1, curr+1, func(item Item, index int) bool {
					if index == curr-1 {
						prevItem = item
					} else {
						currItem = item
					}
					return true
				})

				minBound, err := getMinimalBound(prevItem, currItem)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					panic(err)
				}
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

func (n *Negentropy) DecodeTimestampIn(encoded *[]byte) (uint64, error) {
	t, err := decodeVarInt(encoded)
	if err != nil {
		return 0, err
	}
	timestamp := uint64(t)
	if timestamp == 0 {
		timestamp = MaxU64
	} else {
		timestamp--
	}

	timestamp += n.lastTimestampIn
	if timestamp < n.lastTimestampIn { // Check for overflow
		timestamp = MaxU64
	}
	n.lastTimestampIn = timestamp
	return timestamp, nil
}

func (n *Negentropy) DecodeBound(encoded *[]byte) (Bound, error) {
	timestamp, err := n.DecodeTimestampIn(encoded)
	if err != nil {
		return Bound{}, err
	}

	length, err := decodeVarInt(encoded)
	if err != nil {
		return Bound{}, err
	}

	id, err := getBytes(encoded, length)
	if err != nil {
		return Bound{}, err
	}

	bound, err := NewBound(timestamp, hex.EncodeToString(id))
	if err != nil {
		return Bound{}, err
	}

	return *bound, nil
}

// Encoding

// encodeTimestampOut encodes the given timestamp.
func (n *Negentropy) encodeTimestampOut(timestamp uint64) []byte {
	if timestamp == MaxU64 {
		n.lastTimestampOut = MaxU64
		return encodeVarInt(0)
	}
	temp := timestamp
	timestamp -= n.lastTimestampOut
	n.lastTimestampOut = temp
	return encodeVarInt(int(timestamp + 1))
}

// encodeBound encodes the given Bound into a byte slice.
func (n *Negentropy) encodeBound(bound Bound) ([]byte, error) {
	var output []byte

	t := n.encodeTimestampOut(bound.Item.Timestamp)
	idlen := encodeVarInt(bound.IDLen)
	output = append(output, t...)
	output = append(output, idlen...)
	id := bound.Item.ID

	if len(id) < bound.IDLen {
		return nil, errors.New("ID length exceeds bound")
	}
	output = append(output, id...)
	return output, nil
}

func getMinimalBound(prev, curr Item) (Bound, error) {
	if curr.Timestamp != prev.Timestamp {
		bound, err := NewBound(curr.Timestamp, "")
		return *bound, err
	}

	sharedPrefixBytes := 0

	for i := 0; i < IDSize; i++ {
		if curr.ID[i:i+2] != prev.ID[i:i+2] {
			break
		}
		sharedPrefixBytes++
	}

	// sharedPrefixBytes + 1 to include the first differing byte, or the entire ID if identical.
	// Ensure not to exceed the slice's length.
	bound, err := NewBound(curr.Timestamp, curr.ID[:sharedPrefixBytes*2+1])
	return *bound, err
}
