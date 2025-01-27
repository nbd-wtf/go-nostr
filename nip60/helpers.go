package nip60

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"

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

func calculateFee(inputs cashu.Proofs, keysets []nut02.Keyset) uint {
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
	return (n + 999) / 1000
}

// returns blinded messages, secrets - [][]byte, and list of r
func createBlindedMessages(
	splitAmounts []uint64,
	keysetId string,
) (cashu.BlindedMessages, []string, []*secp256k1.PrivateKey, error) {
	splitLen := len(splitAmounts)
	blindedMessages := make(cashu.BlindedMessages, splitLen)
	secrets := make([]string, splitLen)
	rs := make([]*secp256k1.PrivateKey, splitLen)

	for i, amt := range splitAmounts {
		var secret string
		var r *secp256k1.PrivateKey
		secret, r, err := generateRandomSecret()
		if err != nil {
			return nil, nil, nil, err
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

func generateRandomSecret() (string, *secp256k1.PrivateKey, error) {
	r, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return "", nil, err
	}

	secretBytes := make([]byte, 32)
	_, err = rand.Read(secretBytes)
	if err != nil {
		return "", nil, err
	}
	secret := hex.EncodeToString(secretBytes)

	return secret, r, nil
}

func splitWalletTarget(proofs cashu.Proofs, amountToSplit uint64, mint string) []uint64 {
	target := 3

	// amounts that are in wallet
	amountsInWallet := make([]uint64, len(proofs))
	for i, proof := range proofs {
		amountsInWallet[i] = proof.Amount
	}
	slices.Sort(amountsInWallet)

	allPossibleAmounts := make([]uint64, crypto.MAX_ORDER)
	for i := 0; i < crypto.MAX_ORDER; i++ {
		amount := uint64(math.Pow(2, float64(i)))
		allPossibleAmounts[i] = amount
	}

	// based on amounts that are already in the wallet
	// define what amounts wanted to reach target
	var neededAmounts []uint64
	for _, amount := range allPossibleAmounts {
		count := cashu.Count(amountsInWallet, amount)
		timesToAdd := cashu.Max(0, uint64(target)-uint64(count))
		for i := 0; i < int(timesToAdd); i++ {
			neededAmounts = append(neededAmounts, amount)
		}
	}
	slices.Sort(neededAmounts)

	// fill in based on the needed amounts
	// that are below the amount passed (amountToSplit)
	var amounts []uint64
	var amountsSum uint64 = 0
	for amountsSum < amountToSplit {
		if len(neededAmounts) > 0 {
			if amountsSum+neededAmounts[0] > amountToSplit {
				break
			}
			amounts = append(amounts, neededAmounts[0])
			amountsSum += neededAmounts[0]
			neededAmounts = slices.Delete(neededAmounts, 0, 1)
		} else {
			break
		}
	}

	remainingAmount := amountToSplit - amountsSum
	if remainingAmount > 0 {
		amounts = append(amounts, cashu.AmountSplit(remainingAmount)...)
	}
	slices.Sort(amounts)

	return amounts
}

func signInput(
	privateKey *btcec.PrivateKey,
	publicKey *btcec.PublicKey,
	proof cashu.Proof,
	nut10Secret nut10.WellKnownSecret,
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
	blindedSignatures cashu.BlindedSignatures,
	blindedMessages cashu.BlindedMessages,
	secrets []string,
	rs []*secp256k1.PrivateKey,
	keys map[uint64]*btcec.PublicKey,
) (cashu.Proofs, error) {
	sigsLenght := len(blindedSignatures)
	if sigsLenght != len(secrets) || sigsLenght != len(rs) {
		return nil, errors.New("lengths do not match")
	}

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
				blindedMessages[i].B_,
				blindedSignature.C_,
			) {
				return nil, errors.New("got blinded signature with invalid DLEQ proof")
			} else {
				dleq = &cashu.DLEQProof{
					E: blindedSignature.DLEQ.E,
					S: blindedSignature.DLEQ.S,
					R: hex.EncodeToString(rs[i].Serialize()),
				}
			}
		}

		C, err := unblindSignature(blindedSignature.C_, rs[i], pubkey)
		if err != nil {
			return nil, err
		}

		proof := cashu.Proof{
			Amount: blindedSignature.Amount,
			Secret: secrets[i],
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

func parseKeysetKeys(keys nut01.KeysMap) (map[uint64]*btcec.PublicKey, error) {
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
