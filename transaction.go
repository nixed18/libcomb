package main

import (
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
	var txid = hash256(txraw[:])

	//create our data structures
	for j := 0; j < 21; j++ {
		txcommitsandfrom[j] = signature[j]
	}
	txcommitsandfrom[21] = source

	txidandto[0] = txid
	txidandto[1] = destination

	//verify signature
	var teeth_lengths = CutCombWhere(txid[0:])
	var data [672]byte

	var tip = hash_chains(signature, teeth_lengths)
	for i := range tip {
		copy(data[i*32:i*32+32], tip[i][0:])
	}

	var actuallyfrom = hash256(data[:])
	
	if actuallyfrom != source {
		log("error signature invalid")
		return
	}
	
	commits_mutex.RLock()
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
	var oldactivity = segments_transaction_activity[txid]
	var newactivity = tx_scan_leg_activity(txid)

	//logf("old %021b\nnew %021b\n", oldactivity, newactivity)
	segments_transaction_activity[txid] = newactivity
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
	commits_mutex.RUnlock()
}

func hash_seq_next(h *[32]byte) {
	//treat h as a big 256bit integer and increment it
	for i := range *h {
		if (*h)[i] != 255 {
			(*h)[i]++
			break
		}
		(*h)[i] = 0
	}
}

func txlegs_store_leg(leg [32]byte, totx [32]byte) bool {
	//attempt to store (leg -> totx) in segments_transaction_target
	//if the leg is already mapped then increment leg until it finds free spot (return true),
	//or finds a spot thats already mapped to totx (return false)

	//the other store functions in this file work the same way

	var iter = leg
	for {
		hash_seq_next(&iter)

		var maybetx, ok = segments_transaction_target[iter]

		if !ok {
			segments_transaction_target[iter] = totx
			return true
		}
		if ok && maybetx == totx {
			return false
		}
	}
}

func txlegs_each_leg_target(leg [32]byte, eacher func(*[32]byte) bool) {
	//execute eacher on all the entries including and after leg in segments_transaction_target
	//terminates if eacher return false or there are no more entries
	//essentially executes eather on every txid that leg has been mapped to (every double spend + valid spend)

	//other target functions in this file work the same way

	var iter = leg

	for {
		hash_seq_next(&iter)
		var maybetx, ok = segments_transaction_target[iter]

		if !ok {
			return
		}

		if !eacher(&maybetx) {
			return
		}
	}
}

func txdoublespends_store_doublespend(source [32]byte, to [2][32]byte) bool {
	var iter = source

	for {
		hash_seq_next(&iter)

		var maybetx, ok = segments_transaction_doublespends[iter]

		if !ok {
			segments_transaction_doublespends[iter] = to
			return true
		}
		if ok && maybetx == to {
			return false
		}
	}
}

func txdoublespends_each_doublespend_target(source [32]byte, eacher func(*[2][32]byte) bool) {
	var iter = source

	for {
		hash_seq_next(&iter)
		var maybetx, ok = segments_transaction_doublespends[iter]

		if !ok {
			return
		}

		if !eacher(&maybetx) {
			return
		}
	}
}

func merkledata_store_epsilonzeroes(source [32]byte, to [32]byte) bool {
	var iter = source

	for {
		hash_seq_next(&iter)

		var maybedata, ok = epsilonzeroes[iter]

		if !ok {
			epsilonzeroes[iter] = to
			return true
		}
		if ok && maybedata == to {
			return false
		}
	}
}

func tx_scan_leg_activity(tx [32]byte) (activity uint32) {

	var data, ok1 = segments_transaction_data[tx]
	if !ok1 {
		return 0
	}

	var tags [21]utxotag
	var iterations [21]uint16
	var input [21][32]byte
	for i := 0; i < 21; i++ { 
		input[i] = data[i]
		var roottag, ok2 = commits[commit(data[i][0:])]

		if !ok2 {
			iterations[i] = 0
		} else {
			iterations[i] = 65535
			tags[i] = roottag
		}
	}

	var activities = hash_chains_compare(input, iterations, tags)

	for i := 0; i < 21; i++ { 
		if activities[i] {
			activity |= 1 << uint(i)
		}
	}
	return activity
}