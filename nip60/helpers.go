package nip60

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut01"
	"github.com/elnosh/gonuts/cashu/nuts/nut02"
	"github.com/elnosh/gonuts/cashu/nuts/nut10"
	"github.com/elnosh/gonuts/cashu/nuts/nut11"
	"github.com/elnosh/gonuts/cashu/nuts/nut12"
	"github.com/elnosh/gonuts/crypto"
)

func calculateFee(inputs cashu.Proofs, keysets []nut02.Keyset) uint64 {
	var n uint = 0
next:
	for _, proof := range inputs {
		for _, ks := range keysets {
			if ks.Id == proof.Id {
				n += ks.InputFeePpk
				continue next
			}
		}

		panic(fmt.Errorf("spending a proof we don't have the keyset for? %v // %v", proof, keysets))
	}
	return uint64((n + 999) / 1000)
}

// returns blinded messages, secrets - [][]byte, and list of r
func createBlindedMessages(
	splitAmounts []uint64,
	keysetId string,
	spendingCondition *nut10.SpendingCondition,
) (cashu.BlindedMessages, []string, []*secp256k1.PrivateKey, error) {
	splitLen := len(splitAmounts)
	blindedMessages := make(cashu.BlindedMessages, splitLen)
	secrets := make([]string, splitLen)
	rs := make([]*secp256k1.PrivateKey, splitLen)

	for i, amt := range splitAmounts {
		r, err := secp256k1.GeneratePrivateKey()
		if err != nil {
			return nil, nil, nil, err
		}

		var secret string
		if spendingCondition != nil {
			secret, err = nut10.NewSecretFromSpendingCondition(*spendingCondition)
			if err != nil {
				return nil, nil, nil, err
			}
		} else {
			secretBytes := make([]byte, 32)
			if _, err := rand.Read(secretBytes); err != nil {
				return nil, nil, nil, err
			}
			secret = hex.EncodeToString(secretBytes)
		}

		B_, r, err := crypto.BlindMessage(secret, r)
		if err != nil {
			return nil, nil, nil, err
		}

		blindedMessages[i] = cashu.NewBlindedMessage(keysetId, amt, B_)
		secrets[i] = secret
		rs[i] = r
	}

	return blindedMessages, secrets, rs, nil
}

func signInput(
	privateKey *btcec.PrivateKey,
	proof cashu.Proof,
) (string, error) {
	hash := sha256.Sum256([]byte(proof.Secret))
	signature, err := schnorr.Sign(privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}
	witness, _ := json.Marshal(nut11.P2PKWitness{
		Signatures: []string{hex.EncodeToString(signature.Serialize())},
	})
	return string(witness), nil
}

func signOutput(
	privateKey *btcec.PrivateKey,
	output cashu.BlindedMessage,
) (string, error) {
	msg, _ := hex.DecodeString(output.B_)
	hash := sha256.Sum256(msg)
	signature, err := schnorr.Sign(privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}
	witness, _ := json.Marshal(nut11.P2PKWitness{
		Signatures: []string{hex.EncodeToString(signature.Serialize())},
	})
	return string(witness), nil
}

// constructProofs unblinds the blindedSignatures and returns the proofs
func constructProofs(
	prep preparedOutputs,
	blindedSignatures cashu.BlindedSignatures,
	keys map[uint64]*btcec.PublicKey,
) (cashu.Proofs, error) {
	// blinded sigs might be less than slices in prep, but that is fine, we just ignore the last
	// items in prep. it happens when we are building proofs from change sent by a mint after melt.

	proofs := make(cashu.Proofs, len(blindedSignatures))
	for i, blindedSignature := range blindedSignatures {
		pubkey, ok := keys[blindedSignature.Amount]
		if !ok {
			return nil, errors.New("key not found")
		}

		var dleq *cashu.DLEQProof
		// verify DLEQ if present
		if blindedSignature.DLEQ != nil {
			if !nut12.VerifyBlindSignatureDLEQ(
				*blindedSignature.DLEQ,
				pubkey,
				prep.bm[i].B_,
				blindedSignature.C_,
			) {
				return nil, errors.New("got blinded signature with invalid DLEQ proof")
			} else {
				dleq = &cashu.DLEQProof{
					E: blindedSignature.DLEQ.E,
					S: blindedSignature.DLEQ.S,
					R: hex.EncodeToString(prep.rs[i].Serialize()),
				}
			}
		}

		C, err := unblindSignature(blindedSignature.C_, prep.rs[i], pubkey)
		if err != nil {
			return nil, err
		}

		proof := cashu.Proof{
			Amount: blindedSignature.Amount,
			Secret: prep.secrets[i],
			C:      C,
			Id:     blindedSignature.Id,
			DLEQ:   dleq,
		}
		proofs[i] = proof
	}

	return proofs, nil
}

func unblindSignature(C_str string, r *secp256k1.PrivateKey, key *secp256k1.PublicKey) (
	string,
	error,
) {
	C_bytes, err := hex.DecodeString(C_str)
	if err != nil {
		return "", err
	}
	C_, err := secp256k1.ParsePubKey(C_bytes)
	if err != nil {
		return "", err
	}

	C := crypto.UnblindSignature(C_, r, key)
	Cstr := hex.EncodeToString(C.SerializeCompressed())
	return Cstr, nil
}

func ParseKeysetKeys(keys nut01.KeysMap) (map[uint64]*btcec.PublicKey, error) {
	parsedKeys := make(map[uint64]*btcec.PublicKey)
	for amount, pkh := range keys {
		pkb, err := hex.DecodeString(pkh)
		if err != nil {
			return nil, err
		}
		pubkey, err := btcec.ParsePubKey(pkb)
		if err != nil {
			return nil, err
		}
		parsedKeys[amount] = pubkey
	}
	return parsedKeys, nil
}

func GetSatoshisAmountFromBolt11(bolt11 string) (uint64, error) {
	if len(bolt11) < 50 {
		return 0, fmt.Errorf("invalid invoice, too short")
	}
	bolt11 = bolt11[0:50]
	idx := strings.LastIndex(bolt11, "1")
	if idx == -1 {
		return 0, fmt.Errorf("invalid invoice")
	}
	hrp := bolt11[0:idx]
	amount, ok := strings.CutPrefix(hrp, "lnbc")
	if !ok {
		return 0, fmt.Errorf("invalid invoice")
	}
	if len(amount) < 1 {
		return 0, nil
	}

	// if last character is a digit, then the amount can just be interpreted as BTC
	char := amount[len(amount)-1]
	digit := char - '0'
	isDigit := digit >= 0 && digit <= 9

	cutPoint := len(amount) - 1
	if isDigit {
		cutPoint++
	}

	// if not a digit, it must be part of the known units
	num := amount[:cutPoint]
	if len(num) < 1 {
		return 0, nil
	}

	am, err := strconv.ParseUint(num, 10, 64)
	if err != nil {
		return 0, err
	}

	switch char {
	case 'm':
		return am * 100000, nil
	case 'u':
		return am * 100, nil
	case 'n':
		return am / 10, nil
	case 'p':
		return am / 10000, nil
	default:
		// is BTC
		return am * 100000000, nil
	}
}
