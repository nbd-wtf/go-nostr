package libsecp256k1

/*
#cgo LDFLAGS: -lsecp256k1
#include <secp256k1.h>
#include <secp256k1_schnorrsig.h>
#include <secp256k1_extrakeys.h>
*/
import "C"

import (
	"crypto/rand"
	"errors"
	"unsafe"
)

type Context struct {
	ctx *C.secp256k1_context
}

func NewContext() (*Context, error) {
	ctx := C.secp256k1_context_create(C.SECP256K1_CONTEXT_SIGN | C.SECP256K1_CONTEXT_VERIFY)
	if ctx == nil {
		return nil, errors.New("failed to create secp256k1 context")
	}
	return &Context{ctx: ctx}, nil
}

func (c *Context) Destroy() {
	C.secp256k1_context_destroy(c.ctx)
}

func (c *Context) Sign(msg [32]byte, sk [32]byte) ([64]byte, error) {
	var sig [64]byte

	var keypair C.secp256k1_keypair
	if C.secp256k1_keypair_create(c.ctx, &keypair, (*C.uchar)(unsafe.Pointer(&sk[0]))) != 1 {
		return sig, errors.New("failed to parse private key")
	}

	var random [32]byte
	rand.Read(random[:])

	if C.secp256k1_schnorrsig_sign32(c.ctx, (*C.uchar)(unsafe.Pointer(&sig[0])), (*C.uchar)(unsafe.Pointer(&msg[0])), &keypair, (*C.uchar)(unsafe.Pointer(&random[0]))) != 1 {
		return sig, errors.New("failed to sign message")
	}

	return sig, nil
}

func (c *Context) Verify(msg [32]byte, sig [64]byte, pk [32]byte) bool {
	var xonly C.secp256k1_xonly_pubkey
	if C.secp256k1_xonly_pubkey_parse(c.ctx, &xonly, (*C.uchar)(unsafe.Pointer(&pk[0]))) != 1 {
		return false
	}

	return C.secp256k1_schnorrsig_verify(c.ctx, (*C.uchar)(unsafe.Pointer(&sig[0])), (*C.uchar)(unsafe.Pointer(&msg[0])), 32, &xonly) == 1
}
