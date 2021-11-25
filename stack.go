package main

import (
	"crypto/sha256"
)

func stack_encode(destination, change [32]byte, sum uint64) (raw [72]byte) {
	var sum_bytes [8]byte = uint64_to_bytes(sum)
	copy(raw[0:], change[:])
	copy(raw[32:], destination[:])
	copy(raw[64:], sum_bytes[:])
	return raw
}

func stack_address(destination, change [32]byte, sum uint64) [32]byte {
	var rawdata = stack_encode(destination, change, sum)
	var hash = sha256.Sum256(rawdata[0:])
	return hash
}

func stack_load_data(destination, change [32]byte, sum uint64) {
	var rawdata = stack_encode(destination, change, sum)
	var hash = sha256.Sum256(rawdata[0:])
	var maybecoinbase = commit(hash[0:])

	segments_stack_mutex.Lock()
	segments_stack[hash] = rawdata
	segments_stack_uncommit[maybecoinbase] = hash
	segments_stack_mutex.Unlock()

	commits_mutex.RLock()
	if _, ok1 := combbases[maybecoinbase]; ok1 {
		segments_coinbase_trickle_auto(maybecoinbase, hash)
	}
	commits_mutex.RUnlock()

	segments_transaction_mutex.RLock()
	segments_merkle_mutex.RLock()
	segments_stack_trickle(make(map[[32]byte]struct{}), hash)
	segments_merkle_mutex.RUnlock()
	segments_transaction_mutex.RUnlock()
}

func stack_decode(b []byte) (changeto [32]byte, sumto [32]byte, sum uint64) {
	copy(changeto[0:], b[0:32])
	copy(sumto[0:], b[32:64])

	sum = uint64(b[64])
	sum <<= 8
	sum += uint64(b[65])
	sum <<= 8
	sum += uint64(b[66])
	sum <<= 8
	sum += uint64(b[67])
	sum <<= 8
	sum += uint64(b[68])
	sum <<= 8
	sum += uint64(b[69])
	sum <<= 8
	sum += uint64(b[70])
	sum <<= 8
	sum += uint64(b[71])

	return changeto, sumto, sum
}