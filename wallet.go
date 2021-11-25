package main

import (
	"crypto/rand"
	"crypto/sha256"
	"sync"
)

var wallet_mutex sync.RWMutex
var wallet map[[32]byte][21][32]byte
var wallet_commitments map[[32]byte][32]byte
var wallet_saved int

func init() {
	wallet = make(map[[32]byte][21][32]byte)
	wallet_commitments = make(map[[32]byte][32]byte)
}

func wallet_compute_public_key(key [21][32]byte) (pub [32]byte) {
	var tip [21][32]byte
	var buf [672]byte
	var sli []byte
	sli = buf[0:0]
	for i := range key {
		tip[i] = key[i]
		for j := 0; j < 59213; j++ {
			tip[i] = sha256.Sum256(tip[i][:])
		}
		sli = append(sli, tip[i][:]...)
	}
	pub = sha256.Sum256(sli)
	return pub
}

func wallet_generate_key() (public [32]byte, private [21][32]byte) {
	for i := range private {
		_, err := rand.Read(private[i][0:])
		if err != nil {
			logf("error generating true random key: %s", err)
			return
		}
	}
	public = wallet_compute_public_key(private)
	return public, private
}

func wallet_load_key(key [21][32]byte) {

	public := wallet_compute_public_key(key)
	pub_commit := commit(public[0:])

	wallet_mutex.Lock()
	wallet[public] = key
	wallet_commitments[pub_commit] = public
	wallet_mutex.Unlock()

	//does this really access the cache??
	commit_cache_mutex.Lock()
	commits_mutex.Lock()
	if _, ok1 := combbases[pub_commit]; ok1 {
		segments_coinbase_trickle_auto(pub_commit, public)
	}
	commits_mutex.Unlock()
	commit_cache_mutex.Unlock()
}

func wallet_sign_transaction(source [32]byte, destination [32]byte) (signature [21][32]byte) {
	var key [21][32]byte
	var ok bool

	wallet_mutex.RLock()
	if key, ok = wallet[source]; !ok {
		wallet_mutex.RUnlock()
		logf("error signing, no such key in wallet")
		return
	}
	wallet_mutex.RUnlock()

	var buffer [736]byte
	var slice []byte
	slice = buffer[0:0]

	slice = append(slice, source[0:]...)
	slice = append(slice, destination[0:]...)

	var id = sha256.Sum256(slice)
	depths := CutCombWhere(id[0:])

	for i := range depths {
		for j := uint16(0); j < uint16(LEVELS)-uint16(depths[i]); j++ {
			key[i] = sha256.Sum256(key[i][0:])
		}
		signature[i] = key[i]//commit(key[i][0:])
	}
	return signature
}