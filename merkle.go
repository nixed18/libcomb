package main

import (
	)


func merkle_construct_internal(addr0, addr1, addr2, secret0, secret1, chaining [32]byte, siggy uint16) {

	var tube [3][17][32]byte

	tube[0][0] = addr0
	tube[1][0] = addr1
	tube[2][0] = addr2

	for y := 0; y < 16; y++ {
		for x := 0; x < 3; x++ {
			tube[x][y+1] = merkle(tube[x][y][:], tube[(x+1)%3][y][:])
		}
	}

	var sig = int(siggy)

	var bitsum byte = 0
	for i := sig; i > 0; i >>= 1 {
		bitsum += byte(i & 1)
	}

	var b0 = tube[(bitsum+2)%3][16]

	logf("merkle=%X\n", b0)

	var chainz [2][32]byte
	var chtipz [2][32]byte
	chainz[0] = secret0
	chainz[1] = secret1

	chainz[0] = hash_chain(chainz[0], uint16(65535-sig))
	chainz[1] = hash_chain(chainz[1], uint16(sig))
	chtipz = chainz

	chtipz[0] = hash_chain(chtipz[0], uint16(sig))
	chtipz[1] = hash_chain(chtipz[0], uint16(65535-sig))

	logf("commit0=%X\n", commit(chainz[0][0:]))
	logf("commit1=%X\n", commit(chainz[1][0:]))

	var a0buf [96]byte

	copy(a0buf[32:64], chtipz[0][0:])
	copy(a0buf[64:96], chtipz[1][0:])

	var a0 = hash256(a0buf[0:])
	var e0 = merkle(a0[0:], b0[0:])

	logf("nextchainer=%X\n", a0)
	logf("pay-to-root=%X\n", e0)

	logf("fullbranch=")

	logf("%X", chtipz[0])
	logf("%X", chtipz[1])
	logf("%X", chainz[0])
	logf("%X", chainz[1])

	var x = 0
	for y := uint(0); y < 16; y++ {
		if ((sig >> y) & 1) == 1 {
			x++
			x %= 3
		} else {
			x += 2
			x %= 3

		}
		logf("%X", tube[x][y])
		if ((sig >> y) & 1) == 1 {
			x += 2
			x %= 3
		}
	}

	logf("%X", tube[0][0])

	var chainer [32]byte = chaining
	logf("%X", chainer)
	logf("\n")
}

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

func notify_transaction(a1, a0, u1, u2, q1, q2 [32]byte, z [16][32]byte, b1 [32]byte) (bool, [32]byte) {

	var e [2][32]byte

	var a1_is_zero = a1 == e[0]

	var sig int

	var hash = q1

	for i := 0; i < 65536; i++ {
		if hash == u1 {
			sig = i
			break
		}

		hash = hash256(hash[0:])
	}
	if hash != u1 {
		logf("error merkle solution sig hash 1 does not match")
		return false, [32]byte{}
	}
	
	hash = hash_chain(q2, uint16(65535-sig))

	if hash != u2 {
		logf("error merkle solution sig hash 2 does not match")
		return false, [32]byte{}
	}

	var b0 = b1

	for i := byte(0); i < 16; i++ {
		if ((sig >> i) & 1) == 1 {
			b0 = merkle(b0[0:], z[i][0:])
		} else {
			b0 = merkle(z[i][0:], b0[0:])
		}
	}

	e[0] = merkle(a0[0:], b0[0:])

	var cq1 = commit(q1[0:])
	var cq2 = commit(q2[0:])

	segments_merkle_mutex.Lock()

	segments_merkle_uncommit[commit(e[0][0:])] = e[0]

	merkledata_store_epsilonzeroes(cq1, e[0])
	merkledata_store_epsilonzeroes(cq2, e[0])

	//logf("e0 = %X\n", e[0])

	if a1_is_zero {
		e[1] = b1

	} else {
		e[1] = merkle(a1[0:], b1[0:])

	}
	var tx = merkle(e[0][0:], e[1][0:])
	if a1_is_zero {
		segments_merkle_whiteheart[tx] = [4][32]byte{q1, q2, u1, u2}
	} else {
		segments_merkle_blackheart[tx] = [4][32]byte{q1, q2, u1, u2}
	}

	var e0q1 = merkle(e[0][0:], cq1[0:])
	var e0q2 = merkle(e[0][0:], cq2[0:])
	segments_merkle_lever[e0q1] = e[1]
	segments_merkle_lever[e0q2] = e[1]

	segments_merkle_mutex.Unlock()

	//var ahash = merkle(a0[0:], a1[0:])

	//segments_merkle_heart[ahash] = [2]commitment_t{q1,q2}
	commits_mutex.Lock()
	//segments_merkle_mutex.Lock()
	reactivate(tx, e)
	//segments_merkle_mutex.Unlock()
	commits_mutex.Unlock()
	return true, e[0]
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
func merkle_load_data_internal(rawdata [704]byte) {
	var arraydata [22][32]byte

	for i := range arraydata {
		copy(arraydata[i][0:], rawdata[32*i:32+32*i])
	}

	var z [16][32]byte
	for i := range z {
		z[i] = arraydata[MERKLE_DATA_Z0+i]
	}

	var buf3_a0 [96]byte

	copy(buf3_a0[0:32], arraydata[MERKLE_INPUT_A1][0:32])
	copy(buf3_a0[32:64], arraydata[MERKLE_DATA_U1][0:32])
	copy(buf3_a0[64:96], arraydata[MERKLE_DATA_U2][0:32])

	var a0 = hash256(buf3_a0[0:])

	//logf("a0=%X\n", a0)

	var notified, e0 = notify_transaction(arraydata[MERKLE_INPUT_A1], a0, arraydata[MERKLE_DATA_U1],
		arraydata[MERKLE_DATA_U2], arraydata[MERKLE_DATA_Q1], arraydata[MERKLE_DATA_Q2], z, arraydata[MERKLE_DATA_B1])

	if notified {

		segments_merkle_mutex.Lock()

		segmets_merkle_userinput[arraydata] = e0

		segments_merkle_mutex.Unlock()
	}
}
