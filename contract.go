package libcomb

import (
	"crypto/rand"
	"sync"
)

var purse_mutex sync.Mutex
var purse map[[32]byte][2][32]byte //short->decider

func init() {
	purse = make(map[[32]byte][2][32]byte)
}

func purse_load_decider(decider [2][32]byte) (short_address [32]byte) {
	var next [32]byte
	var short [2][32]byte = purse_compute_short_decider(decider)
	short_address = purse_compute_short_address(short, next)
	purse_mutex.Lock()
	purse[short_address] = decider
	purse_mutex.Unlock()
	return short_address
}

func purse_generate_decider() (private [2][32]byte) {
	for i := range private {
		_, err := rand.Read(private[i][0:])
		if err != nil {
			logf("error generating true random key (%s)", err.Error())
			return
		}
	}
	return private
}

func purse_compute_short_decider(decider [2][32]byte) (short [2][32]byte) {
	short[0] = hash_chain(decider[0], 65535)
	short[1] = hash_chain(decider[1], 65535)
	return short
}

func purse_compute_short_address(short [2][32]byte, next [32]byte) (short_address [32]byte) {
	var buf [96]byte

	copy(buf[0:], next[:])
	copy(buf[32:], short[0][:])
	copy(buf[64:], short[1][:])

	short_address = hash256(buf[:])
	return short_address
}

func purse_sign_decider(decider [2][32]byte, number uint16) (signature [2][32]byte) {
	signature[0] = hash_chain(decider[0], number)
	signature[1] = hash_chain(decider[1], 65535-number)
	return signature
}

func purse_recover_signed_number(short [2][32]byte, signature [2][32]byte) (number uint16, ok bool) {
	var hash [32]byte
	hash = signature[1]

	for i := 0; i < 65536; i++ {
		if hash == short[1] {
			number = uint16(i)
			break
		}
		hash = hash256(hash[0:])
	}

	if hash != short[1] {
		return 0, false
	}

	hash = hash_chain(signature[0], uint16(65535-number))
	if hash != short[0] {
		return 0, false
	}

	return number, true
}
func contract_compute_address(short_address [32]byte, root [32]byte) (address [32]byte) {
	address = merkle(short_address[:], root[:])
	return address
}
