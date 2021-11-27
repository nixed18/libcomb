package main

import "sync"

var segments_merkle_mutex sync.RWMutex

//heart is the id of a merkle segment = hash(address, destination), exactly like a txid
var segments_merkle_uncommit map[[32]byte][32]byte //commit(address) -> address
var segments_merkle_lever map[[32]byte][32]byte //hash(address, signature) -> destination
var segments_merkle_blackheart map[[32]byte][4][32]byte //heart -> long + short
var segments_merkle_whiteheart map[[32]byte][4][32]byte //heart -> long + short (final segment in tree)
var epsilonzeroes map[[32]byte][32]byte //commit(signature) -> address
var segments_merkle_activity map[[32]byte]byte //heart -> activity (last seen activity)
var e0_to_e1 map[[32]byte][32]byte //address -> destination

func segmentmerkle_reset() {
	segments_merkle_mutex.Lock()
	segments_merkle_uncommit = make(map[[32]byte][32]byte)
	epsilonzeroes = make(map[[32]byte][32]byte)
	segments_merkle_blackheart = make(map[[32]byte][4][32]byte)
	segments_merkle_whiteheart = make(map[[32]byte][4][32]byte)
	segments_merkle_lever = make(map[[32]byte][32]byte)
	segments_merkle_activity = make(map[[32]byte]byte)
	e0_to_e1 = make(map[[32]byte][32]byte)
	segments_merkle_mutex.Unlock()
}

func init() {
	segmentmerkle_reset()
}

const SEGMENT_MERKLE_TRICKLED byte = 16

func segments_merkle_trickle(loopkiller map[[32]byte]struct{}, commitment [32]byte) {
	
	if balance_try_increase_loop(commitment) {
		return
	}

	if _, ok2 := loopkiller[commitment]; ok2 {

		balance_create_loop(commitment)
		return
	}
	loopkiller[commitment] = struct{}{}

	segments_merkle_mutex.RLock()
	var txidandto, ok = e0_to_e1[commitment]
	segments_merkle_mutex.RUnlock()
	var to = txidandto

	balance_do(commitment, to, 0xffffffffffffffff)

	if !ok {
		println("trickle non existent tx")
	}

	var type3 = segments_stack_type(to)
	if type3 == SEGMENT_STACK_TRICKLED {
		segments_stack_trickle(loopkiller, to)
	}

	var type2 = segments_merkle_type(to)
	if type2 == SEGMENT_MERKLE_TRICKLED {
		segments_merkle_trickle(loopkiller, to)
	}

	var type1 = segments_transaction_type(to)
	if type1 == SEGMENT_TX_TRICKLED {
		segments_transaction_trickle(loopkiller, to)
	} else if type1 == SEGMENT_ANY_UNTRICKLED {
	} else if type1 == SEGMENT_UNKNOWN {
	}

}
func segments_merkle_untrickle(loopkiller *[32]byte, commitment [32]byte, bal balance) {
	graph_dirty = true
}

func segments_merkle_type(commit [32]byte) segment_type {
	segments_merkle_mutex.RLock()
	_, ok1 := e0_to_e1[commit]
	segments_merkle_mutex.RUnlock()

	if ok1 {
		return SEGMENT_MERKLE_TRICKLED
	}

	return SEGMENT_UNKNOWN
}

func segments_merkle_loopdetect(norecursion, loopkiller map[[32]byte]struct{}, commitment [32]byte) bool {
	if _, ok2 := loopkiller[commitment]; ok2 {

		return true
	}
	loopkiller[commitment] = struct{}{}
	segments_merkle_mutex.RLock()
	var txidandto, ok = e0_to_e1[commitment]
	segments_merkle_mutex.RUnlock()
	var to = txidandto

	if !ok {
		return false
	}
	if _, ok2 := loopkiller[to]; ok2 {

		return true
	}
	var type3 = segments_stack_type(to)
	if type3 == SEGMENT_STACK_TRICKLED {
		return segments_stack_loopdetect(norecursion, loopkiller, to)
	}
	var type2 = segments_merkle_type(to)
	if type2 == SEGMENT_MERKLE_TRICKLED {
		return segments_merkle_loopdetect(norecursion, loopkiller, to)
	}
	var type1 = segments_transaction_type(to)
	if type1 == SEGMENT_TX_TRICKLED {
		return segments_transaction_loopdetect(norecursion, loopkiller, to)
	} else if type1 == SEGMENT_ANY_UNTRICKLED {
	} else if type1 == SEGMENT_UNKNOWN {
	}

	return false
}

func segments_merkle_backgraph(backgraph map[[32]byte][][32]byte, norecursion map[[32]byte]struct{}, target, commitment [32]byte) {

	_, is_stack_recursion := norecursion[commitment]

	if is_stack_recursion {
		return
	}

	norecursion[commitment] = struct{}{}

	segments_merkle_mutex.RLock()
	var txidandto, ok = e0_to_e1[commitment]
	segments_merkle_mutex.RUnlock()
	var to = txidandto

	if !ok {
		return
	}

	add_to_backgraph(backgraph, to, commitment)

	var type3 = segments_stack_type(to)
	if type3 == SEGMENT_STACK_TRICKLED {
		segments_stack_backgraph(backgraph, norecursion, target, to)
	}
	var type2 = segments_merkle_type(to)
	if type2 == SEGMENT_MERKLE_TRICKLED {
		segments_merkle_backgraph(backgraph, norecursion, target, to)
	}
	var type1 = segments_transaction_type(to)
	if type1 == SEGMENT_TX_TRICKLED {
		segments_transaction_backgraph(backgraph, norecursion, target, to)
	} else if type1 == SEGMENT_ANY_UNTRICKLED {
	} else if type1 == SEGMENT_UNKNOWN {
	}

	return
}
