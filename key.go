package libcomb

import (
	"crypto/rand"
	"fmt"
	"log"
)

type Key struct {
	Public  [32]byte
	Private [21][32]byte
}

func (k Key) ID() [32]byte {
	if k.Public == [32]byte{} {
		log.Panicf("key has not been recovered")
	}
	return k.Public
}

func (k Key) trigger() (err error) {
	return nil //optionally check for old spends here
}

func (k Key) triggers() [][32]byte {
	return nil
}

func (k Key) Active() bool {
	if _, ok := balance_edge[k.Public]; ok {
		return true
	}
	return false
}

func key_lookup(id [32]byte) (Key, error) {
	var key Key
	if c, ok := constructs[id]; ok {
		if key, ok = c.(Key); !ok {
			return key, fmt.Errorf("not a key")
		}
	} else {
		return key, fmt.Errorf("not a construct")
	}

	return key, nil
}

func key_create() (Key, error) {
	var k Key
	for i := range k.Private {
		if _, err := rand.Read(k.Private[i][0:]); err != nil {
			return k, fmt.Errorf("cannot get random value: %s", err)
		}
	}
	key_recover(&k)
	return k, nil
}

func key_recover(k *Key) {
	var tips [21][32]byte = k.Private
	for i := range tips {
		for x := uint16(0); x < uint16(LEVELS); x++ {
			tips[i] = Hash256(tips[i][:])
		}
	}
	k.Public = Hash256Concat32(tips[:])
}

func key_sign(k *Key, value [32]byte) (signature [21][32]byte) {
	cuts := cut(value[:])
	for i, k := range k.Private {
		signature[i] = k
		for x := uint16(0); x < uint16(LEVELS)-cuts[i]; x++ {
			signature[i] = Hash256(signature[i][:])
		}
	}
	return signature
}
