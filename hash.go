package main

import ( 
	"github.com/minio/sha256-simd"
	"sync"
)

var hash_count uint64 //single threaded hashing (try to minimize)
var chain_hash_count uint64 //optimized threaded hashing

/*
extracted from the rest of libcomb for optimization
sha256-simd should be faster on CPU's with SIMD or dedicated SHA instructions
basic multithreading added for chain hashing, an opencl kernel may perform better
processing structures with chains (keys, transactions, deciders) can easily result in hashing millions of times
*/

func hash_chains(in [21][32]byte, iterations [21]uint16) [21][32]byte {
	var out [21][32]byte
	var wg sync.WaitGroup
	for i := uint16(0); i < 21; i++ {
		wg.Add(1)
		go func(idx uint16) {
			defer wg.Done()
			out[idx] = hash_chain(in[idx], iterations[idx])
		}(i)
	}
	wg.Wait()
	return out
}

func hash_chains_fixed(in [21][32]byte, iterations uint16) [21][32]byte {
	var arr [21]uint16
	for i := range in {
		arr[i] = iterations
	}
	return hash_chains(in, arr)
}

func hash_chains_compare(in [21][32]byte, iterations [21]uint16, tags [21]utxotag) [21]bool {
	var out [21]bool
	var wg sync.WaitGroup
	for i := uint16(0); i < 21; i++ {
		wg.Add(1)
		go func(idx uint16) {
			defer wg.Done()
			out[idx] = hash_chain_compare(in[idx], iterations[idx], tags[idx])
		}(i)
	}
	wg.Wait()
	return out
}

func hash_chain_compare(in [32]byte, iterations uint16, tag utxotag) bool {
	var buf [64]byte
	var hash [32]byte = in
	copy(buf[0:], whitepaper[:])
	copy(buf[32:], hash[:])
	for j := uint16(0); j < iterations; j++ {
		hash = sha256.Sum256(buf[32:])
		copy(buf[32:], hash[:])
		var candidate, ok = commits[sha256.Sum256(buf[:])]
		//hash_count+=2
		if !ok {
			continue
		}
		if utag_cmp(&tag, &candidate) >= 0 {
			return true
		}
	}
	return false
}

func hash_chain(in [32]byte, iterations uint16) [32]byte {
	for i := uint16(0); i < iterations; i++ {
		in = sha256.Sum256(in[:])
		chain_hash_count++
	}
	return in
}

func hash256(in []byte) [32]byte {
	//processing keys or transactions can easily result in hashing millions of times
	//should be faster on CPU's with SIMD or dedicated SHA instructions
	hash_count++
	return sha256.Sum256(in)
}