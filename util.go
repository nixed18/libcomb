package libcomb

import (
	"crypto/sha256"
	"hash"
)

var whitepaper = [32]byte{106, 251, 172, 89, 92, 29, 7, 163, 212, 197, 23, 151, 88, 245, 188, 228, 70, 42, 108, 38, 63, 110, 109, 252, 217, 66, 1, 20, 51, 173, 170, 231}
var testnet_whitepaper = [32]byte{46, 56, 65, 182, 231, 94, 151, 23, 171, 125, 42, 139, 87, 36, 139, 127, 97, 26, 84, 115, 56, 27, 94, 67, 42, 175, 143, 232, 136, 116, 251, 254}

func Hash256(data []byte) (out [32]byte) {
	if !testnet {
		return sha256.Sum256(data)
	} else {
		var h hash.Hash = sha256.New()
		h.Write(testnet_whitepaper[:])
		h.Write(testnet_whitepaper[:])
		h.Write(data)
		h.Sum(out[0:0])
		return out
	}
}

func Hash256Concat32(data [][32]byte) (out [32]byte) {
	var c []byte = make([]byte, 32*len(data))
	for i, d := range data {
		copy(c[i*32:(i+1)*32], d[:])
	}
	return Hash256(c[:])
}

func Hash256Adjacent(a [32]byte, b [32]byte) (out [32]byte) {
	var c [64]byte
	copy(c[0:], a[:])
	copy(c[32:], b[:])
	return Hash256(c[:])
}

func commit(hash [32]byte) [32]byte {
	var data = [2][32]byte{whitepaper, hash}
	return Hash256Concat32(data[:])
}
