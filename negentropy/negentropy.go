package negentropy

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"slices"
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
	isInitiator      bool
	lastTimestampIn  nostr.Timestamp
	lastTimestampOut nostr.Timestamp
	haveIds          []string
	needIds          []string
}

func NewNegentropy(storage Storage, frameSizeLimit int) (*Negentropy, error) {
	return &Negentropy{
		storage:        storage,
		frameSizeLimit: frameSizeLimit,
	}, nil
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

func (n *Negentropy) Initiate() []byte {
	n.seal()
	n.isInitiator = true

	n.haveIds = make([]string, 0, n.storage.Size()/2)
	n.needIds = make([]string, 0, n.storage.Size()/2)

	output := bytes.NewBuffer(make([]byte, 0, 1+n.storage.Size()*32))
	output.WriteByte(protocolVersion)
	n.SplitRange(0, 0, n.storage.Size(), infiniteBound, output)

	return output.Bytes()
}

func (n *Negentropy) Reconcile(step int, query []byte) (output []byte, haveIds []string, needIds []string, err error) {
	n.seal()
	reader := bytes.NewReader(query)

	output, err = n.reconcileAux(step, reader)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(output) == 1 && n.isInitiator {
		slices.Sort(n.haveIds)
		slices.Sort(n.needIds)
		return nil, n.haveIds, n.needIds, nil
	}

	return output, nil, nil, nil
}

func (n *Negentropy) reconcileAux(step int, reader *bytes.Reader) ([]byte, error) {
	n.lastTimestampIn, n.lastTimestampOut = 0, 0 // reset for each message

	fullOutput := bytes.NewBuffer(make([]byte, 0, 5000))
	fullOutput.WriteByte(protocolVersion)

	pv, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	if pv < 0x60 || pv > 0x6f {
		return nil, fmt.Errorf("invalid protocol version byte")
	}
	if pv != protocolVersion {
		if n.isInitiator {
			return nil, fmt.Errorf("unsupported negentropy protocol version requested")
		}
		return fullOutput.Bytes(), nil
	}

	var prevBound Bound
	prevIndex := 0
	skip := false

	partialOutput := bytes.NewBuffer(make([]byte, 0, 100))
	for reader.Len() > 0 {
		partialOutput.Reset()

		doSkip := func() {
			if skip {
				skip = false
				encodedBound := n.encodeBound(prevBound)
				fmt.Println(n.Name(), step, "~>      skip", prevBound)
				partialOutput.Write(encodedBound)
				partialOutput.WriteByte(SkipMode)
			}
		}

		currBound, err := n.DecodeBound(reader)
		if err != nil {
			return nil, err
		}
		fmt.Println(n.Name(), step, "<~ read bound", currBound)
		modeVal, err := decodeVarInt(reader)
		if err != nil {
			return nil, err
		}
		mode := Mode(modeVal)

		lower := prevIndex
		upper := n.storage.FindLowerBound(prevIndex, n.storage.Size(), currBound)
		fmt.Println(n.Name(), step, "<~ [", lower, n.storage.GetBound(lower), "---", n.storage.GetBound(upper), upper, "]")

		switch mode {
		case SkipMode:
			fmt.Println(n.Name(), step, "<~  skip")
			fmt.Println(n.Name(), step, "~>    will be skipped")
			skip = true

		case FingerprintMode:
			fmt.Println(n.Name(), step, "<~  fingerprint")

			var theirFingerprint [FingerprintSize]byte
			_, err := reader.Read(theirFingerprint[:])
			if err != nil {
				return nil, err
			}
			ourFingerprint, err := n.storage.Fingerprint(lower, upper)
			if err != nil {
				return nil, err
			}

			fmt.Println(n.Name(), step, "<~    ours", hex.EncodeToString(ourFingerprint[:]))
			fmt.Println(n.Name(), step, "<~    thrs", hex.EncodeToString(theirFingerprint[:]))

			if theirFingerprint == ourFingerprint {
				skip = true
			} else {
				doSkip()
				n.SplitRange(step, lower, upper, currBound, partialOutput)
			}

		case IdListMode:
			fmt.Print(n.Name(), " ", step, " <~  idlist")
			numIds, err := decodeVarInt(reader)
			if err != nil {
				return nil, err
			}
			fmt.Printf(" (%d)", numIds)

			theirElems := make(map[string]struct{})
			var idb [32]byte

			firstid := "()"
			lastid := "()"
			for i := 0; i < numIds; i++ {
				_, err := reader.Read(idb[:])
				if err != nil {
					return nil, err
				}
				// fmt.Println(n.Name(), step, "<~    id", hex.EncodeToString(idb))
				id := hex.EncodeToString(idb[:])
				if firstid == "()" {
					firstid = id
				}
				theirElems[id] = struct{}{}
				lastid = id
			}
			fmt.Println("", firstid, "---", lastid)

			n.storage.Iterate(lower, upper, func(item Item, _ int) bool {
				id := item.ID
				if _, exists := theirElems[id]; !exists {
					if n.isInitiator {
						n.haveIds = append(n.haveIds, id)
						// fmt.Println(n.Name(), step, "<~      have", id)
					}
				} else {
					delete(theirElems, id)
				}
				return true
			})

			if n.isInitiator {
				skip = true
				fmt.Println(n.Name(), step, "~>        will be skipped")

				for id := range theirElems {
					n.needIds = append(n.needIds, id)
					// fmt.Println(n.Name(), step, "<~      need", id)
				}
			} else {
				doSkip()

				responseIds := make([]byte, 0, 32*n.storage.Size())
				endBound := currBound
				fmt.Print(n.Name(), " ", step, " ~>      idlist")

				firstid := "()"
				lastid := "()"
				n.storage.Iterate(lower, upper, func(item Item, index int) bool {
					if firstid == "()" {
						firstid = item.ID
					}

					if n.frameSizeLimit-200 < fullOutput.Len()+len(responseIds) {
						fmt.Println(" ###")
						endBound = Bound{item}
						upper = index
						return false
					}

					lastid = item.ID
					id, _ := hex.DecodeString(item.ID)
					// fmt.Println(n.Name(), step, "~>      id", item.ID)
					responseIds = append(responseIds, id...)
					return true
				})
				fmt.Println(endBound, firstid, "---", lastid)

				encodedBound := n.encodeBound(endBound)

				partialOutput.Write(encodedBound)
				partialOutput.WriteByte(IdListMode)
				partialOutput.Write(encodeVarInt(len(responseIds) / 32))
				partialOutput.Write(responseIds)

				partialOutput.WriteTo(fullOutput)
				partialOutput.Reset()
			}

		default:
			return nil, fmt.Errorf("unexpected mode %d", mode)
		}

		if n.frameSizeLimit-200 < fullOutput.Len()+partialOutput.Len() {
			fmt.Println(" #####")
			// frame size limit exceeded, handle by encoding a boundary and fingerprint for the remaining range
			remainingFingerprint, err := n.storage.Fingerprint(upper, n.storage.Size())
			if err != nil {
				panic(err)
			}

			fullOutput.Write(n.encodeBound(infiniteBound))
			fullOutput.WriteByte(FingerprintMode)
			fullOutput.Write(remainingFingerprint[:])
			fmt.Println(n.Name(), step, "~>   last fingerprint", infiniteBound)

			break // stop processing further
		} else {
			// append the constructed output for this iteration
			partialOutput.WriteTo(fullOutput)
		}

		prevIndex = upper
		prevBound = currBound
	}

	return fullOutput.Bytes(), nil
}

func (n *Negentropy) SplitRange(step int, lower, upper int, upperBound Bound, output *bytes.Buffer) {
	numElems := upper - lower
	const buckets = 16

	fmt.Println(n.Name(), step, "~> splitting range", lower, n.storage.GetBound(lower), "---", n.storage.GetBound(upper), upper)

	if numElems < buckets*2 {
		// we just send the full ids here
		boundEncoded := n.encodeBound(upperBound)
		output.Write(boundEncoded)
		output.WriteByte(IdListMode)
		output.Write(encodeVarInt(numElems))

		fmt.Print(n.Name(), " ", step, " ~>   idlist ", upperBound)

		firstid := "()"
		lastid := "()"
		n.storage.Iterate(lower, upper, func(item Item, _ int) bool {
			if firstid == "()" {
				firstid = item.ID
			}
			lastid = item.ID
			// fmt.Println(n.Name(), step, "~>    ", item.ID)
			id, _ := hex.DecodeString(item.ID)
			output.Write(id)
			return true
		})
		fmt.Println("", firstid, "---", lastid)
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

				minBound := getMinimalBound(prevItem, currItem)
				nextBound = minBound
			}

			fmt.Println(n.Name(), step, "~>   bound and fingerprint", nextBound, hex.EncodeToString(ourFingerprint[:]))
			boundEncoded := n.encodeBound(nextBound)
			output.Write(boundEncoded)
			output.WriteByte(FingerprintMode)
			output.Write(ourFingerprint[:])
		}
	}
}

func (n *Negentropy) Name() string {
	p := unsafe.Pointer(n)
	return fmt.Sprintf("%d", uintptr(p)&127)
}
