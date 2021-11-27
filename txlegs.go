package main

import "sync"

var txleg_mutex sync.RWMutex
var txleg_to_tx map[[32]byte][32]byte

func txlegs_reset() {
	txleg_mutex.Lock()
	txleg_to_tx = make(map[[32]byte][32]byte)
	txleg_mutex.Unlock()
}

func init() {
	txlegs_reset()
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
	//attempt to store (leg -> totx) in txleg_to_tx
	//if the leg is already mapped then increment leg until it finds free spot (return true),
	//or finds a spot thats already mapped to totx (return false)

	//the other store functions in this file work the same way

	var iter = leg
	for {
		hash_seq_next(&iter)

		var maybetx, ok = txleg_to_tx[iter]

		if !ok {
			txleg_to_tx[iter] = totx
			return true
		}
		if ok && maybetx == totx {
			return false
		}
	}
}

func txlegs_each_leg_target(leg [32]byte, eacher func(*[32]byte) bool) {
	//execute eacher on all the entries including and after leg in txleg_to_tx
	//terminates if eacher return false or there are no more entries
	//essentially executes eather on every txid that leg has been mapped to (every double spend + valid spend)

	//other target functions in this file work the same way

	var iter = leg

	for {
		hash_seq_next(&iter)
		var maybetx, ok = txleg_to_tx[iter]

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

func merkledata_each_epsilonzeroes(source [32]byte, eacher func(*[32]byte) bool) {
	var iter = source

	for {
		hash_seq_next(&iter)
		var maybedata, ok = epsilonzeroes[iter]

		if !ok {
			return
		}

		if !eacher(&maybedata) {
			return
		}
	}
}
