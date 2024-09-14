package negentropy

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"unsafe"

	"github.com/nbd-wtf/go-nostr"
)

const (
	protocolVersion byte = 0x61 // version 1
	maxTimestamp         = nostr.Timestamp(math.MaxInt64)
)

var infiniteBound = Bound{Item: Item{Timestamp: maxTimestamp}}

type Negentropy struct {
	storage          Storage
	sealed           bool
	frameSizeLimit   int
	isClient         bool
	lastTimestampIn  nostr.Timestamp
	lastTimestampOut nostr.Timestamp

	Haves    chan string
	HaveNots chan string
}

func NewNegentropy(storage Storage, frameSizeLimit int) *Negentropy {
	return &Negentropy{
		storage:        storage,
		frameSizeLimit: frameSizeLimit,
	}
}

func (n *Negentropy) String() string {
	label := "unsealed"
	if n.sealed {
		label = "server"
		if n.isClient {
			label = "client"
		}
	}
	return fmt.Sprintf("<Negentropy %s with %d items>", label, n.storage.Size())
}

func (n *Negentropy) Insert(evt *nostr.Event) {
	err := n.storage.Insert(evt.CreatedAt, evt.ID)
	if err != nil {
		panic(err)
	}
}

func (n *Negentropy) seal() {
	if !n.sealed {
		n.storage.Seal()
	}
	n.sealed = true
}

func (n *Negentropy) Initiate() string {
	n.seal()
	n.isClient = true

	n.Haves = make(chan string, n.storage.Size()/2)
	n.HaveNots = make(chan string, n.storage.Size()/2)

	output := NewStringHexWriter(make([]byte, 0, 1+n.storage.Size()*64))
	output.WriteByte(protocolVersion)
	n.SplitRange(0, n.storage.Size(), infiniteBound, output)

	return output.Hex()
}

func (n *Negentropy) Reconcile(msg string) (output string, err error) {
	n.seal()
	reader := NewStringHexReader(msg)

	output, err = n.reconcileAux(reader)
	if err != nil {
		return "", err
	}

	if len(output) == 2 && n.isClient {
		close(n.Haves)
		close(n.HaveNots)
		return "", nil
	}

	return output, nil
}

func (n *Negentropy) reconcileAux(reader *StringHexReader) (string, error) {
	n.lastTimestampIn, n.lastTimestampOut = 0, 0 // reset for each message

	fullOutput := NewStringHexWriter(make([]byte, 0, 5000))
	fullOutput.WriteByte(protocolVersion)

	pv, err := reader.ReadHexByte()
	if err != nil {
		return "", fmt.Errorf("failed to read pv: %w", err)
	}
	if pv != protocolVersion {
		return "", fmt.Errorf("unsupported negentropy protocol version %v", pv)
	}

	var prevBound Bound
	prevIndex := 0
	skipping := false // this means we are currently coalescing ranges into skip

	partialOutput := NewStringHexWriter(make([]byte, 0, 100))
	for reader.Len() > 0 {
		partialOutput.Reset()

		finishSkip := func() {
			// end skip range, if necessary, so we can start a new bound that isn't a skip
			if skipping {
				skipping = false
				n.encodeBound(partialOutput, prevBound)
				partialOutput.WriteByte(byte(SkipMode))
			}
		}

		currBound, err := n.DecodeBound(reader)
		if err != nil {
			return "", fmt.Errorf("failed to decode bound: %w", err)
		}
		modeVal, err := decodeVarInt(reader)
		if err != nil {
			return "", fmt.Errorf("failed to decode mode: %w", err)
		}
		mode := Mode(modeVal)

		lower := prevIndex
		upper := n.storage.FindLowerBound(prevIndex, n.storage.Size(), currBound)

		switch mode {
		case SkipMode:
			skipping = true

		case FingerprintMode:
			var theirFingerprint [FingerprintSize]byte
			if err := reader.ReadHexBytes(theirFingerprint[:]); err != nil {
				return "", fmt.Errorf("failed to read fingerprint: %w", err)
			}
			ourFingerprint := n.storage.Fingerprint(lower, upper)

			if theirFingerprint == ourFingerprint {
				skipping = true
			} else {
				finishSkip()
				n.SplitRange(lower, upper, currBound, partialOutput)
			}

		case IdListMode:
			numIds, err := decodeVarInt(reader)
			if err != nil {
				return "", fmt.Errorf("failed to decode number of ids: %w", err)
			}

			// what they have
			theirItems := make([]string, 0, numIds)
			for i := 0; i < numIds; i++ {
				if id, err := reader.ReadString(64); err != nil {
					return "", fmt.Errorf("failed to read id (#%d/%d) in list: %w", i, numIds, err)
				} else {
					theirItems = append(theirItems, id)
				}
			}

			// what we have
			for _, item := range n.storage.Range(lower, upper) {
				id := item.ID

				if idx, theyHave := slices.BinarySearch(theirItems, id); theyHave {
					// if we have and they have, ignore
					theirItems[idx] = ""
				} else {
					// if we have and they don't, notify client
					if n.isClient {
						n.Haves <- id
					}
				}
			}

			if n.isClient {
				// notify client of what they have and we don't
				for _, id := range theirItems {
					if id != "" {
						n.HaveNots <- id
					}
				}

				// client got list of ids, it's done, skip
				skipping = true
			} else {
				// server got list of ids, reply with their own ids for the same range
				finishSkip()

				responseIds := strings.Builder{}
				responseIds.Grow(64 * 100)
				responses := 0

				endBound := currBound

				for index, item := range n.storage.Range(lower, upper) {
					if n.frameSizeLimit-200 < fullOutput.Len()+1+8+responseIds.Len() {
						endBound = Bound{item}
						upper = index
						break
					}
					responseIds.WriteString(item.ID)
					responses++
				}

				n.encodeBound(partialOutput, endBound)
				partialOutput.WriteByte(byte(IdListMode))
				encodeVarIntToHex(partialOutput, responses)
				partialOutput.WriteHex(responseIds.String())

				fullOutput.WriteHex(partialOutput.Hex())
				partialOutput.Reset()
			}

		default:
			return "", fmt.Errorf("unexpected mode %d", mode)
		}

		if n.frameSizeLimit-200 < fullOutput.Len()+partialOutput.Len() {
			// frame size limit exceeded, handle by encoding a boundary and fingerprint for the remaining range
			remainingFingerprint := n.storage.Fingerprint(upper, n.storage.Size())
			n.encodeBound(fullOutput, infiniteBound)
			fullOutput.WriteByte(byte(FingerprintMode))
			fullOutput.WriteBytes(remainingFingerprint[:])

			break // stop processing further
		} else {
			// append the constructed output for this iteration
			fullOutput.WriteHex(partialOutput.Hex())
		}

		prevIndex = upper
		prevBound = currBound
	}

	return fullOutput.Hex(), nil
}

func (n *Negentropy) SplitRange(lower, upper int, upperBound Bound, output *StringHexWriter) {
	numElems := upper - lower
	const buckets = 16

	if numElems < buckets*2 {
		// we just send the full ids here
		n.encodeBound(output, upperBound)
		output.WriteByte(byte(IdListMode))
		encodeVarIntToHex(output, numElems)

		for _, item := range n.storage.Range(lower, upper) {
			output.WriteHex(item.ID)
		}
	} else {
		itemsPerBucket := numElems / buckets
		bucketsWithExtra := numElems % buckets
		curr := lower

		for i := 0; i < buckets; i++ {
			bucketSize := itemsPerBucket
			if i < bucketsWithExtra {
				bucketSize++
			}
			ourFingerprint := n.storage.Fingerprint(curr, curr+bucketSize)
			curr += bucketSize

			var nextBound Bound
			if curr == upper {
				nextBound = upperBound
			} else {
				var prevItem, currItem Item

				for index, item := range n.storage.Range(curr-1, curr+1) {
					if index == curr-1 {
						prevItem = item
					} else {
						currItem = item
					}
				}

				minBound := getMinimalBound(prevItem, currItem)
				nextBound = minBound
			}

			n.encodeBound(output, nextBound)
			output.WriteByte(byte(FingerprintMode))
			output.WriteBytes(ourFingerprint[:])
		}
	}
}

func (n *Negentropy) Name() string {
	p := unsafe.Pointer(n)
	return fmt.Sprintf("%d", uintptr(p)&127)
}
