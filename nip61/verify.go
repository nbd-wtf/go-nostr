package nip61

import (
	"encoding/hex"
	"encoding/json"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/crypto"
	"github.com/nbd-wtf/go-nostr"
)

func VerifyNutzap(
	keyset map[uint64]*btcec.PublicKey,
	evt *nostr.Event,
) (sats uint64, ok bool) {
	for _, tag := range evt.Tags {
		if len(tag) < 2 {
			continue
		}
		if tag[0] != "proof" {
			continue
		}

		var proof cashu.Proof
		if err := json.Unmarshal([]byte(tag[1]), &proof); err != nil {
			continue
		}

		if !verifyProofDLEQ(proof, keyset[proof.Amount]) {
			return 0, false
		}

		sats += proof.Amount
	}

	return sats, true
}

func verifyProofDLEQ(
	proof cashu.Proof,
	A *btcec.PublicKey,
) bool {
	e, s, r, err := parseDLEQ(*proof.DLEQ)
	if err != nil || r == nil {
		return false
	}

	B_, _, err := crypto.BlindMessage(proof.Secret, r)
	if err != nil {
		return false
	}

	CBytes, err := hex.DecodeString(proof.C)
	if err != nil {
		return false
	}

	C, err := btcec.ParsePubKey(CBytes)
	if err != nil {
		return false
	}

	var CPoint, APoint btcec.JacobianPoint
	C.AsJacobian(&CPoint)
	A.AsJacobian(&APoint)

	// C' = C + r*A
	var C_Point, rAPoint btcec.JacobianPoint
	btcec.ScalarMultNonConst(&r.Key, &APoint, &rAPoint)
	rAPoint.ToAffine()
	btcec.AddNonConst(&CPoint, &rAPoint, &C_Point)
	C_Point.ToAffine()
	C_ := btcec.NewPublicKey(&C_Point.X, &C_Point.Y)

	return crypto.VerifyDLEQ(e, s, A, B_, C_)
}

func VerifyBlindSignatureDLEQ(
	dleq cashu.DLEQProof,
	A *btcec.PublicKey,
	B_str string,
	C_str string,
) bool {
	e, s, _, err := parseDLEQ(dleq)
	if err != nil {
		return false
	}

	B_bytes, err := hex.DecodeString(B_str)
	if err != nil {
		return false
	}
	B_, err := btcec.ParsePubKey(B_bytes)
	if err != nil {
		return false
	}

	C_bytes, err := hex.DecodeString(C_str)
	if err != nil {
		return false
	}
	C_, err := btcec.ParsePubKey(C_bytes)
	if err != nil {
		return false
	}

	return crypto.VerifyDLEQ(e, s, A, B_, C_)
}

func parseDLEQ(dleq cashu.DLEQProof) (
	*btcec.PrivateKey,
	*btcec.PrivateKey,
	*btcec.PrivateKey,
	error,
) {
	ebytes, err := hex.DecodeString(dleq.E)
	if err != nil {
		return nil, nil, nil, err
	}
	e := secp256k1.PrivKeyFromBytes(ebytes)

	sbytes, err := hex.DecodeString(dleq.S)
	if err != nil {
		return nil, nil, nil, err
	}
	s := secp256k1.PrivKeyFromBytes(sbytes)

	if dleq.R == "" {
		return e, s, nil, nil
	}

	rbytes, err := hex.DecodeString(dleq.R)
	if err != nil {
		return nil, nil, nil, err
	}
	r := secp256k1.PrivKeyFromBytes(rbytes)

	return e, s, r, nil
}
