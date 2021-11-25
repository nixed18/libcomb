package main

import (
	"crypto/sha256"
)

func transaction_raw_data(source, destination [32]byte) (raw [64]byte) {
	copy(raw[0:], source[0:])
	copy(raw[32:], destination[0:])
	return raw
}

func transaction_load(source, destination [32]byte, signature [21][32]byte) {
	var txcommitsandfrom [22][32]byte
	var txidandto [2][32]byte
	var txraw = transaction_raw_data(source, destination)
	var txid = sha256.Sum256(txraw[:])

	//create our data structures
	for j := 0; j < 21; j++ {
		txcommitsandfrom[j] = signature[j]
	}
	txcommitsandfrom[21] = source

	txidandto[0] = txid
	txidandto[1] = destination

	//verify signature
	var teeth_lengths = CutCombWhere(txid[0:])
    var teeth_tips [21][32]byte

	var hash = sha256.New()
	for i := 0; i < 21; i++ {
		var hashchain = signature[i]
		for j := uint16(0); j < teeth_lengths[i]; j++ {
			hashchain = sha256.Sum256(hashchain[0:])
		}
		hash.Write(hashchain[:])
		teeth_tips[i] = hashchain

	}
	var actuallyfrom [32]byte
	copy(actuallyfrom[:], hash.Sum(nil))

	logf("%X %X\n", hash.Sum(nil), source)
	if actuallyfrom != source {
		log("error signature invalid")
		return
	}
	
	commits_mutex.RLock()
	txleg_mutex.Lock()
	segments_transaction_mutex.Lock()
	segments_merkle_mutex.Lock()

	//point the source commit to the source (since its usually not reversible)
	//so we can trickle coinbases on source to the destination (or untrickle)
	segments_transaction_uncommit[commit(source[0:])] = source

	//point each signature commit to our transaction
	//used to trickle funds when all commits are seen, or untrickle if commits are rolled back
	for i := 0; i < 21; i++ {  
		txlegs_store_leg(commit(signature[i][0:]), txid)
	}

	//point the source to our transaction (and the destination)
	//used to decide which destination to trickle funds, in the case of a double spends
	if _, ok := segments_transaction_data[txid]; !ok {
		txdoublespends_store_doublespend(source, txidandto)
	}

	segments_transaction_data[txid] = txcommitsandfrom

	//check if transaction is valid (commited + not a double spend), and trickle/untrickle accordingly
	var oldactivity = tx_legs_activity[txid]
	var newactivity = tx_scan_leg_activity(txid)
	tx_legs_activity[txid] = newactivity
	if oldactivity != newactivity {
		//there is an older transaction thats valid
		if oldactivity == 2097151 /*0b111111111111111111111 (21 1's)*/ {
			segments_transaction_untrickle(nil, source, 0xffffffffffffffff)
			delete(segments_transaction_next, source)
		}
		//our transaction is valid
		if newactivity == 2097151 {
			segments_transaction_next[source] = txidandto
			var maybecoinbase = commit(source[0:])
			if _, ok1 := combbases[maybecoinbase]; ok1 {
				segments_coinbase_trickle_auto(maybecoinbase, source)
			}
			segments_transaction_trickle(make(map[[32]byte]struct{}), source)
		}
	}

	segments_merkle_mutex.Unlock()
	segments_transaction_mutex.Unlock()
	txleg_mutex.Unlock()
	commits_mutex.RUnlock()
}
