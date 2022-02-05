package nip04

// random -> IV
// crpyto -> aes-cbc
// secpk256k1 -> for computing shared (symmetric) key

func computeSharedKey() (key []byte) {
	return make([]byte, 32)
}
