package main

import (
	)

var tx_legs_activity map[[32]byte]uint32

func init() {
	tx_legs_activity = make(map[[32]byte]uint32)
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

