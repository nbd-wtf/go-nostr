package nip26

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/nbd-wtf/go-nostr"
)

type DelegationToken struct {
	delegator [32]byte
	token     [64]byte
	kinds     []int
	since     *time.Time
	until     *time.Time
	tag       [4]string
}

// Tag() returns the nostr formatted delegation tag for the DelegationToken d.
func (d *DelegationToken) Tag() nostr.Tag {
	return nostr.Tag(d.tag[:])
}

// Conditions() is the delegation conditions string as in NIP-26.
func (d *DelegationToken) Conditions() (conditions string) {
	for _, k := range d.kinds {
		conditions += fmt.Sprintf("kind=%d&", k)
	}
	if d.since != nil {
		conditions += fmt.Sprintf("created_at>%d&", d.since.Unix())
	}
	if d.until != nil {
		conditions += fmt.Sprintf("created_at<%d&", d.until.Unix())
	}
	return strings.TrimSuffix(conditions, "&")
}

// This error is returned by d.Parse(ev) if the event ev does not have a delegation token.
var NoDelegationTag error = fmt.Errorf("No Delegation Tag")

// This error is returned by Import(t,delegatee_pk) if the token signature verification fails.
var VerificationFailed error = fmt.Errorf("VerificationFailed")

// CheckDelegation reads the event and reports whether or not it is correctly delegated.
// If there is a delegation tag, the delegation token signature will be checked according to NIP-26.
// If there is no delegation tag, ok will be true and err will be nil.
// For checking many events, it is advisable to use Parse to reduce the number of memory allocations.
func CheckDelegation(ev *nostr.Event) (ok bool, err error) {
	d := new(DelegationToken)
	if ok, err := d.Parse(ev); ok == true && (err == nil || err == NoDelegationTag) {
		return true, nil
	}
	return false, err
}

// Import verifies that t is NIP-26 delegation token for the given delegatee.
// The returned DelegationToken object can be used in DelegatedSign.
// If the token signature verification fails, the error VerificationFailed will be returned.
func Import(t nostr.Tag, delegateePk string) (d *DelegationToken, e error) {
	d = new(DelegationToken)
	if len(t) == 4 && t[0] == "delegation" {
		copy(d.tag[:], t)
	} else {
		return nil, fmt.Errorf("not a delegation tag")
	}
	if n, e := hex.Decode(d.delegator[:], []byte(d.tag[1])); n != 32 || e != nil {
		return nil, fmt.Errorf("invalid delegation tag")
	}
	if n, e := hex.Decode(d.token[:], []byte(d.tag[3])); n != 64 || e != nil {
		return nil, fmt.Errorf("invalid delegation tag")
	}
	if d.kinds, d.since, d.until, e = parseConditions(d.tag[2]); e != nil {
		return nil, fmt.Errorf("invalid conditions string")
	}

	// compute the digest
	h := sha256.Sum256([]byte(fmt.Sprintf("nostr:delegation:%s:%s", delegateePk, d.tag[2])))

	sig, err := schnorr.ParseSignature(d.token[:])
	if err != nil {
		return nil, fmt.Errorf("error: %s", err.Error())
	}

	pubkey, err := schnorr.ParsePubKey(d.delegator[:])
	if err != nil {
		return nil, fmt.Errorf("error: %s", err.Error())
	}
	if !sig.Verify(h[:], pubkey) {
		return nil, VerificationFailed
	}
	return d, nil
}

// Parse reads the event ev and stores the delegation token into d.
// The ok value verifies the event is correctly delegated.
// If there is no delegation token, then d will not be changed. In this case the error value will be `NoDelegationTag`, and ok will be set to true.
// Parse does NOT verify the event was correctly signed. Use ev.CheckSignature() for this.
// More efficient memory allocations versus CheckDelegation(ev) if many events need to be checked.
func (d *DelegationToken) Parse(ev *nostr.Event) (ok bool, err error) {
	for _, t := range ev.Tags {
		if t[0] == "delegation" && len(t) == 4 {
			copy(d.tag[:], t)
			goto jump1
		}
	}
	// event has no delegation. set the token to nil and return.
	return true, NoDelegationTag

jump1:
	if n, e := hex.Decode(d.delegator[:], []byte(d.tag[1])); n != 32 || e != nil {
		return false, fmt.Errorf("invalid delegation tag")
	}
	if n, e := hex.Decode(d.token[:], []byte(d.tag[3])); n != 64 || e != nil {
		return false, fmt.Errorf("invalid delegation tag")
	}

	if d.kinds, d.since, d.until, err = parseConditions(d.tag[2]); err != nil {
		return false, fmt.Errorf("invalid conditions string")
	}

	if len(d.kinds) > 0 {
		for _, k := range d.kinds {
			if ev.Kind == k {
				goto jump2
			}
		}
		return false, fmt.Errorf("event kind %d is not allowed in delegation condition", ev.Kind)
	}

jump2:
	if d.since != nil && ev.CreatedAt.Time().Before(*d.since) {
		return false, fmt.Errorf("event is created before delegation conditions allow")
	}
	if d.until != nil && ev.CreatedAt.Time().After(*d.until) {
		return false, fmt.Errorf("event is created after delegation conditions allow")
	}

	// compute the digest
	h := sha256.Sum256([]byte(fmt.Sprintf("nostr:delegation:%s:%s", ev.PubKey, d.tag[2])))

	sig, err := schnorr.ParseSignature(d.token[:])
	if err != nil {
		return false, fmt.Errorf("error: %s", err.Error())
	}

	pubkey, err := schnorr.ParsePubKey(d.delegator[:])
	if err != nil {
		return false, fmt.Errorf("error: %s", err.Error())
	}

	return sig.Verify(h[:], pubkey), nil
}

// DelegatedSign performs a delegated signature on the event ev.
// The delegation signature is not verified. If desired, the caller can ensure the delegation signature is correct by calling d.Parse(ev) or CheckDelegation(ev) afterwards.
func DelegatedSign(ev *nostr.Event,
	d *DelegationToken, delegateeSk string,
) error {
	for _, t := range ev.Tags {
		if t[0] == "delegation" {
			return fmt.Errorf("event already has delegation token")
		}
	}
	if d.since != nil && ev.CreatedAt.Time().Before(*d.since) || d.until != nil && ev.CreatedAt.Time().After(*d.until) {
		return fmt.Errorf("event created_at field is not compatible with delegation token time conditions")
	}
	if len(d.kinds) > 0 {
		for _, k := range d.kinds {
			if ev.Kind == k {
				goto jump
			}
		}
		return fmt.Errorf("event kind %d is not compatible with delegation token conditions", ev.Kind)
	}
jump:
	if pk, e := nostr.GetPublicKey(delegateeSk); e != nil {
		return fmt.Errorf("invalid delegatee secret key")
	} else {
		ev.PubKey = pk
	}
	ev.Tags = append(ev.Tags, d.Tag())
	return ev.Sign(delegateeSk)
}

// CreateToken creates a DelegationToken according to NIP-26.
func CreateToken(delegator_sk string, delegatee_pk string, kinds []int,
	since *time.Time, until *time.Time,
) (d *DelegationToken, e error) {
	d = new(DelegationToken)
	s, e := hex.DecodeString(delegator_sk)
	if e != nil {
		return nil, fmt.Errorf("invalid delegator secret key")
	}

	teePk, e := hex.DecodeString(delegatee_pk)
	if len(teePk) != 32 || e != nil {
		return nil, fmt.Errorf("invalid delegatee pubkey")
	}

	// set delegator
	sk, torPk := btcec.PrivKeyFromBytes(s)
	copy(d.delegator[:], schnorr.SerializePubKey(torPk))

	d.kinds = kinds
	d.since = since
	d.until = until

	// generate tag
	d.tag[0] = "delegation"
	d.tag[1] = fmt.Sprintf("%x", d.delegator)
	d.tag[2] = d.Conditions()

	h := sha256.Sum256([]byte(fmt.Sprintf("nostr:delegation:%x:%s", teePk, d.tag[2])))

	if sig, err := schnorr.Sign(sk, h[:]); err != nil {
		panic(err)
	} else {
		copy(d.token[:], sig.Serialize())
	}

	d.tag[3] = fmt.Sprintf("%x", d.token)

	return d, nil
}

func parseConditions(conditions string) (kinds []int, since *time.Time, until *time.Time, err error) {
	kinds = make([]int, 0)
	for _, v := range strings.Split(conditions, "&") {
		switch {
		case strings.HasPrefix(v, "kind="):
			if i, e := strconv.ParseInt(strings.TrimPrefix(v, "kind="), 10, 64); e == nil {
				kinds = append(kinds, int(i))
			} else {
				return nil, nil, nil, fmt.Errorf("Invalid: %s!", v)
			}
		case strings.HasPrefix(v, "created_at>") && since == nil:
			if i, e := strconv.ParseInt(strings.TrimPrefix(v, "created_at>"), 10, 64); e == nil {
				t := time.Unix(i, 0)
				since = &t
			} else {
				return nil, nil, nil, fmt.Errorf("Invalid: %s", v)
			}
		case strings.HasPrefix(v, "created_at<") && until == nil:
			if i, e := strconv.ParseInt(strings.TrimPrefix(v, "created_at<"), 10, 64); e == nil {
				t := time.Unix(i, 0)
				until = &t
			} else {
				return nil, nil, nil, fmt.Errorf("Invalid: %s", v)
			}
		default:
			return nil, nil, nil, fmt.Errorf("Invalid: %s", v)
		}
	}
	return
}
