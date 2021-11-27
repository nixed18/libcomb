package main

import (
		"sync"
)

var commits_mutex sync.RWMutex

var commits map[[32]byte]utxotag
var combbases map[[32]byte]struct{}

var commit_cache_mutex sync.Mutex

var commit_current_height uint32
var commit_cache [][32]byte
var commit_tag_cache []utxotag
var commit_rollback [][32]byte
var commit_rollback_tags []utxotag

func mine_reset() {
	commits_mutex.Lock()
	commit_cache_mutex.Lock()

	commits = make(map[[32]byte]utxotag)
	combbases = make(map[[32]byte]struct{})
	commit_current_height = 0
	commit_cache = nil
	commit_tag_cache = nil
	commit_rollback = nil
	commit_rollback_tags = nil

	commit_cache_mutex.Unlock()
	commits_mutex.Unlock()
}

func init() {
	mine_reset()
}

func height_view() (h uint32) {
	commit_cache_mutex.Lock()
	h = commit_current_height
	commit_cache_mutex.Unlock()
	return h
}

//trickles a tx if its confirmed by commit_cache[iter]
func miner_trickle_cache_leg(iter int, tx *[32]byte) bool {
	var key = commit_cache[iter]
	var tagval = commit_tag_cache[iter]

	var oldactivity = tx_legs_activity[*tx]
	var newactivity = oldactivity
	var tags [21]utxotag
	var iterations [21]uint16
	var input [21][32]byte

	//setup input for hash_chains
	for i := 0; i < 21; i++ { 
		input[i] = segments_transaction_data[*tx][i]
		ok := (key == commit(input[i][:]))

		if !ok {
			iterations[i] = 0
		} else {
			iterations[i] = 65535
			tags[i] = tagval
		}
	}
	
	//compute leg activity
	var activities = hash_chains_compare(input, iterations, tags)

	for i := 0; i < 21; i++ { 
		if oldactivity&(1<<i) != 0 {
			continue
		}
		if key == commit(input[i][0:]) {
			if !activities[i] {
				newactivity |= (1 << i)
			}
		}
	}

	//something changed, could be forward or rollback
	if oldactivity != newactivity {
		tx_legs_activity[*tx] = newactivity

		//transaction fully confirmed and valid
		if newactivity == 2097151 {

			logf("block confirms transaction %X \n", *tx)

			segments_transaction_mutex.RLock()

			var actuallyfrom = segments_transaction_data[*tx][21]

			segments_transaction_mutex.RUnlock()

			//we dont know the desintation from the txid, however all loaded transactions are stored in txdoublespends
			//so look for our txid there
			txdoublespends_each_doublespend_target(actuallyfrom, func(txidto *[2][32]byte) bool {
				if *tx == (*txidto)[0] {
					//definitively set the transaction as the target for the source address
					segments_transaction_next[actuallyfrom] = *txidto
					return false
				}
				return true
			})

			//some additional trickling if the source address is a coinbase
			var maybecoinbase = commit(actuallyfrom[0:])
			if _, ok1 := combbases[maybecoinbase]; ok1 {
				segments_coinbase_trickle_auto(maybecoinbase, actuallyfrom)
			}

			//finally trickle the source to the destination (and further trickle depending on the destination construct)
			segments_transaction_trickle(make(map[[32]byte]struct{}), actuallyfrom)
		}
	}

	return true
}

func miner_mine_block() {
	//give coinbase to the first unseen commit
	for i := range commit_cache {
		if _, ok := commits[commit_cache[i]]; !ok {
			var basetag = commit_tag_cache[i]
			var btag = basetag
			var bheight = uint64(btag.height)

			segments_coinbase_mine(commit_cache[i], bheight)
			combbases[commit_cache[i]] = struct{}{}

			break
		}
	}
	
	//load the cache into main memory
	for key, val := range commit_cache {
		if _, ok := commits[val]; ok {
		} else {
			commits[val] = commit_tag_cache[key]
		}
	}

	//trickle constructs that are confirmed by these commits (transaction, decider, etc)
	for iter, key := range commit_cache {
		merkle_mine(key)

		txleg_mutex.RLock()
		txlegs_each_leg_target(key, func(tx *[32]byte) bool { return miner_trickle_cache_leg(iter, tx) })
		txleg_mutex.RUnlock()
	}

	commit_cache = nil
	commit_tag_cache = nil
}

func miner_unmine_block() {
	var unwritten bool = false
	var reorg_height uint64

	//rollback the coinbase
	for i := range commit_rollback {
		if tagcommit, ok5 := commits[commit_rollback[i]]; ok5 {

			var basetag = commit_rollback_tags[i]
			var ctag = tagcommit
			var btag = basetag

			if utag_cmp(&ctag, &btag) != 0 {
				continue
			}

			var bheight = uint64(btag.height)

			if _, ok6 := combbases[commit_rollback[i]]; ok6 {

				segments_coinbase_unmine(commit_rollback[i], bheight)
				delete(combbases, commit_rollback[i])

			}

			break
		}
	}

	//delete the commits from main memory + free used keys
	for i := len(commit_rollback) - 1; i >= 0; i-- {
		key := commit_rollback[i]

		if tagcommit, ok5 := commits[key]; !ok5 {
		} else {

			taggy := commit_rollback_tags[i]

			var ctag = tagcommit
			var btag = taggy

			if utag_cmp(&ctag, &btag) == 0 {
				//CommitDbUnWrite(key)
				delete(commits, key)
				unwritten = true

				if enable_used_key_feature {
					log("reorg commit height", ctag.height)

					reorg_height = uint64(ctag.height)

					used_key_commit_reorg(key, reorg_height)
				}
			}
		}
	}
	
	//finally rollback any constructs previously confirmed by these commits (transaction, decider, etc)
	for _, key := range commit_rollback {

		if _, ok5 := commits[key]; ok5 {
			continue
		}

		merkle_unmine(key)

		txleg_mutex.RLock()

		txlegs_each_leg_target(key, func(tx *[32]byte) bool {
			var oldactivity = tx_legs_activity[*tx]
			var newactivity = oldactivity

			for i := uint(0); i < 21; i++ {
				if oldactivity&(1<<i) == 0 {
					continue
				}

				var val = segments_transaction_data[*tx][i]

				if key == commit(val[0:]) {
					newactivity &= 2097151 ^ (1 << i)
				}
			}

			if oldactivity != newactivity {
				tx_legs_activity[*tx] = newactivity

				if oldactivity == 2097151 {
					logf("block rollbacks transaction %X \n", *tx)

					var actuallyfrom = segments_transaction_data[*tx][21]

					segments_transaction_untrickle(nil, actuallyfrom, 0xffffffffffffffff)

					delete(segments_transaction_next, actuallyfrom)
				}
			}

			return true
		})

		txleg_mutex.RUnlock()
	}

	commit_rollback = nil
	commit_rollback_tags = nil

	if unwritten && enable_used_key_feature {
		log("reorg block height", reorg_height)
		used_key_height_reorg(reorg_height)
	}

	//assume we just unmined every commit in the block
	commit_current_height--
}

func miner_finish_block() {
	commit_cache_mutex.Lock()
	commits_mutex.Lock()
	if len(commit_rollback) > 0 && len(commit_cache) > 0 {
		//protect from this in the interface!
		commits_mutex.Unlock()
		commit_cache_mutex.Unlock()
		return
	} else if len(commit_cache) > 0 {
		miner_mine_block()
	} else if len(commit_rollback) > 0 {
		miner_unmine_block()
	} else {
		//nothing to do!
	}

	resetgraph()

	commits_mutex.Unlock()
	commit_cache_mutex.Unlock()
	return
}

func miner_mine_commit(rawcommit [32]byte, tag utxotag) {
	var is_first bool

	commit_cache_mutex.Lock()

	is_first = len(commit_cache)+len(commit_rollback) == 0

	//can only mine a commit thats higher than the current block
	if is_first && commit_current_height >= tag.height {
		commit_cache_mutex.Unlock()
		logf("error: mined first commitment must be on greater height\n")
		return
	}
	//every batch of commits is in the same block
	if !is_first && commit_current_height != tag.height {
		commit_cache_mutex.Unlock()
		logf("error: commitment must be on same height as first commitment\n")
		return
	}

	commit_cache = append(commit_cache, rawcommit)
	commit_tag_cache = append(commit_tag_cache, tag)

	commits_mutex.Lock()
	if is_first {
		commit_current_height = tag.height
	}
	commits_mutex.Unlock()
	commit_cache_mutex.Unlock()
}

func miner_unmine_commit(rawcommit [32]byte, tag utxotag) {
	var is_first bool

	commit_cache_mutex.Lock()

	is_first = len(commit_cache)+len(commit_rollback) == 0

	//cant unmine a commit thats higher than the current block
	if is_first && commit_current_height < tag.height {
		commit_cache_mutex.Unlock()
		logf("error: unmined first commitment must be on smaller height\n")
		return
	}
	//every batch of commits is in the same block
	if !is_first && commit_current_height != tag.height {
		commit_cache_mutex.Unlock()
		logf("error: commitment must be on same height as first commitment\n")
		return
	}

	commit_rollback = append(commit_rollback, rawcommit)
	commit_rollback_tags = append(commit_rollback_tags, tag)
	commits_mutex.Lock()
	if is_first {
		commit_current_height = tag.height
	}
	commits_mutex.Unlock()
	commit_cache_mutex.Unlock()
}