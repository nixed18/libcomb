package main

import (
	)

func merkle_mine(c [32]byte) {
	segments_merkle_mutex.Lock()

	merkledata_each_epsilonzeroes(c, func(e0 *[32]byte) bool {
		var e [2][32]byte
		e[0] = *e0

		var e0q = merkle(e[0][0:], c[0:])

		//logf("e0q=%X\n", e0q)

		e[1] = segments_merkle_lever[e0q]

		var tx = merkle(e[0][0:], e[1][0:])

		//logf("mine tx=%X\n", tx)

		segments_merkle_mutex.Unlock()
		reactivate(tx, e)
		segments_merkle_mutex.Lock()

		return true
	})
	segments_merkle_mutex.Unlock()
}

func merkle_unmine(c [32]byte) {
	segments_merkle_mutex.Lock()
	merkledata_each_epsilonzeroes(c, func(e0 *[32]byte) bool {

		var e [2][32]byte
		e[0] = *e0
		e[1] = e0_to_e1[e[0]]
		var tx = merkle(e[0][0:], e[1][0:])

		//logf("unmine tx=%X\n", tx)

		segments_merkle_mutex.Unlock()
		reactivate(tx, e)
		segments_merkle_mutex.Lock()
		return true
	})
	segments_merkle_mutex.Unlock()
}

func merkle_scan_leg_activity(tx [32]byte) (activity uint8) {

	var data [4][32]byte

	segments_merkle_mutex.RLock()

	if data1, ok1 := segments_merkle_blackheart[tx]; ok1 {
		data = data1
	} else if data2, ok2 := segments_merkle_whiteheart[tx]; ok2 {
		data = data2
	} else {
		segments_merkle_mutex.RUnlock()

		//println("no heart")
		return 0
	}

	segments_merkle_mutex.RUnlock()

	var j = 0
outer:
	for i := 0; i < 2; i++ {

		var rawroottag, ok2 = commits[commit(data[i][0:])]

		if !ok2 {
			continue
		}

		var roottag = rawroottag

		var hash = data[i]

		for ; j < sigvariability; j++ {
			hash = hash256(hash[0:])
			if hash == data[i+2] {
				j++
				break
			}

			var candidaterawtag, ok3 = commits[commit(hash[0:])]

			if !ok3 {
				continue
			}
			var candidatetag = candidaterawtag

			if utag_cmp(&roottag, &candidatetag) > 0 {

				//log("miscompared hash=", hash)

				//panic("")

				continue outer
			}

		}
		//log("solved activity", hash)
		activity |= 1 << uint(i)
	}
	//log("activity, j", activity, j)
	return activity
}

func notify_transaction(next, short_decider, left_tip, right_tip, left_sig, right_sig [32]byte, branches [16][32]byte, leaf [32]byte) (bool, [32]byte) {
	var address [32]byte
	var destination [32]byte

	//are we at the bottom of our tree?
	var next_is_zero = next == address //zeros

	var sig int

	//recover signature from right_sig and right_tip
	var hash = right_sig
	for i := 0; i < 65536; i++ {
		if hash == right_tip {
			sig = i
			break
		}
		hash = hash256(hash[0:])
	}

	//no signature was found!
	if hash != right_tip {
		logf("error merkle signature invalid")
		return false, [32]byte{}
	}
	
	//verify right side matches signature
	hash = hash_chain(left_sig, uint16(65535-sig))
	if hash != left_tip {
		logf("error signature is inconsistent")
		return false, [32]byte{}
	}

	//recover merkle root from signature, leaf, branches (branches are proof)
	var root = leaf
	for i := byte(0); i < 16; i++ {
		if ((sig >> i) & 1) == 0 {
			root = merkle(root[0:], branches[i][0:])
		} else {
			root = merkle(branches[i][0:], root[0:])
		}
	}

	//recover contract address (or more generically, the merkle segment address)
	address = merkle(short_decider[0:], root[0:])	

	//map: commit -> address (for later confirmation etc)
	var left_commit = commit(left_sig[0:])
	var right_commit = commit(right_sig[0:])
	segments_merkle_mutex.Lock()
	segments_merkle_uncommit[commit(address[0:])] = address
	merkledata_store_epsilonzeroes(left_commit, address)
	merkledata_store_epsilonzeroes(right_commit, address)


	//for multi-level trees the destination is another segment, otherwise the leaf
	if next_is_zero {
		destination = leaf
	} else {
		destination = merkle(next[0:], leaf[0:]) //actually hash(short_decider, merkle_root) = address
	}
	
	//construct the txid (also called the heart) (because this is a transaction!)
	var tx = merkle(address[0:], destination[0:])

	//map the transaction to the signature
	if next_is_zero {
		//different maps depending on if this is the bottom? idk why
		segments_merkle_whiteheart[tx] = [4][32]byte{left_sig, right_sig, left_tip, right_tip}
	} else {
		segments_merkle_blackheart[tx] = [4][32]byte{left_sig, right_sig, left_tip, right_tip}
	}

	//now map: commit + address -> desination
	//used to correlate new commits to this segment like so:
	//	commit -> address, then commit + address -> desitnation, then heart/txid = hash(address, destination)
	//why two steps? because one decider can decide many segments!
	var address_left_sig = merkle(address[0:], left_commit[0:])
	var address_right_sig = merkle(address[0:], right_commit[0:])
	segments_merkle_lever[address_left_sig] = destination
	segments_merkle_lever[address_right_sig] = destination
	segments_merkle_mutex.Unlock()

	//finally trickle the merkle segment if its confirmed already
	commits_mutex.Lock()
	reactivate(tx, [2][32]byte{address, destination})
	commits_mutex.Unlock()
	return true, address
}

func reactivate(tx [32]byte, e [2][32]byte) {
	var oldactivity = segments_merkle_activity[tx]
	var newactivity = merkle_scan_leg_activity(tx)
	segments_merkle_activity[tx] = newactivity
	
	if oldactivity != newactivity {
		if oldactivity == 3 {
			//var maybecoinbase = commit(e[0][0:])

			segments_merkle_untrickle(nil, e[0], 0xffffffffffffffff)
			//segments_coinbase_untrickle_auto(maybecoinbase, e[0])

			segments_merkle_mutex.Lock()
			delete(e0_to_e1, e[0])
			segments_merkle_mutex.Unlock()
		}
		if newactivity == 3 {
			segments_merkle_mutex.Lock()
			if _, ok1 := e0_to_e1[e[0]]; ok1 {
				log("Panic: e0 to e1 already have live path")
				panic("")
			}

			e0_to_e1[e[0]] = e[1]
			segments_merkle_mutex.Unlock()
			var maybecoinbase = commit(e[0][0:])
			if _, ok1 := combbases[maybecoinbase]; ok1 {
				segments_coinbase_trickle_auto(maybecoinbase, e[0])
			}

			segments_merkle_trickle(make(map[[32]byte]struct{}), e[0])
		}
	}
}

func merkle_compute_root(tree [65536][32]byte) [32]byte {
	for j := 0; j < 16; j++ {
		for i := 0; i < 1<<uint(15-j); i++ {
			tree[i] = merkle(tree[2*i][0:], tree[2*i+1][0:])
		}
	}
	return tree[0]
}

func merkle_traverse_tree(tree [65536][32]byte, number uint16) (root [32]byte, branches [16][32]byte, leaf [32]byte) {
	leaf = tree[number]
	for j := 0; j < 16; j++ {
		branches[j] = tree[number^1]
		for i := 0; i < 1<<uint(15-j); i++ {
			tree[i] = merkle(tree[2*i][0:], tree[2*i+1][0:])
		}
		number >>= 1
	}
	root = tree[0]
	return root, branches, leaf
}