package libcomb

import (
	"sync"
)

var commits_mutex sync.RWMutex

var commits map[[32]byte]UTXOtag
var combbases map[[32]byte]struct{}
var combbase_height map[uint64][32]byte

var commit_cache_mutex sync.Mutex

var commit_current_height uint64
var commit_cache [][32]byte
var commit_cache_tags []UTXOtag

var commit_diff []Commit

func mine_reset() {
	commits_mutex.Lock()
	commit_cache_mutex.Lock()

	commits = make(map[[32]byte]UTXOtag)
	combbases = make(map[[32]byte]struct{})
	combbase_height = make(map[uint64][32]byte)
	commit_current_height = 0
	commit_cache = nil
	commit_cache_tags = nil
	commit_diff = nil

	commit_cache_mutex.Unlock()
	commits_mutex.Unlock()
}

func init() {
	mine_reset()
}

func height_view() (h uint64) {
	commit_cache_mutex.Lock()
	h = commit_current_height
	commit_cache_mutex.Unlock()
	return h
}

//trickles a tx if its confirmed by commit_cache[iter]
func miner_trickle_cache_leg(iter int, tx *[32]byte) bool {
	var key = commit_cache[iter]
	var tagval = commit_cache_tags[iter]

	var oldactivity = segments_transaction_activity[*tx]
	var newactivity = oldactivity
	var tags [21]UTXOtag
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
		segments_transaction_activity[*tx] = newactivity

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

func miner_mine_block(height uint64) {
	commit_cache_mutex.Lock()
	commits_mutex.Lock()
	commit_diff = nil

	//give coinbase to the first unseen commit
	for i, c := range commit_cache {
		if _, ok := commits[c]; !ok {
			segments_coinbase_mine(c, commit_cache_tags[i].Height)
			combbases[c] = struct{}{}
			combbase_height[commit_cache_tags[i].Height] = c
			break
		}
	}

	//load the cache into main memory
	for i, c := range commit_cache {
		if _, ok := commits[c]; !ok {
			commit_diff = append(commit_diff, Commit{c, commit_cache_tags[i]})
			commits[c] = commit_cache_tags[i]
		}
	}

	//trickle constructs that are confirmed by these commits (transaction, decider, etc)
	for i, c := range commit_cache {
		merkle_mine(c)

		segments_transaction_mutex.RLock()
		txlegs_each_leg_target(c, func(tx *[32]byte) bool { return miner_trickle_cache_leg(i, tx) })
		segments_transaction_mutex.RUnlock()
	}

	//update the current height
	commit_current_height = height

	commit_cache = nil
	commit_cache_tags = nil

	resetgraph()

	commits_mutex.Unlock()
	commit_cache_mutex.Unlock()
	return
}

func miner_unmine_block() {
	commit_cache_mutex.Lock()
	commits_mutex.Lock()
	commit_diff = nil
	//rollback the coinbase
	for i := range commit_cache {
		if tagcommit, ok := commits[commit_cache[i]]; ok {

			var basetag = commit_cache_tags[i]
			var ctag = tagcommit
			var btag = basetag

			if utag_cmp(&ctag, &btag) != 0 {
				continue
			}

			var bheight = uint64(btag.Height)

			if _, ok = combbases[commit_cache[i]]; ok {

				segments_coinbase_unmine(commit_cache[i], bheight)
				delete(combbase_height, bheight)
				delete(combbases, commit_cache[i])

			}

			break
		}
	}

	//delete the commits from main memory
	for i := len(commit_cache) - 1; i >= 0; i-- {
		key := commit_cache[i]

		if tagcommit, ok := commits[key]; ok {
			taggy := commit_cache_tags[i]

			var ctag = tagcommit
			var btag = taggy

			if utag_cmp(&ctag, &btag) == 0 {
				commit_diff = append(commit_diff, Commit{key, commit_cache_tags[i]})
				delete(commits, key)
			}
		}
	}

	//finally rollback any constructs previously confirmed by these commits (transaction, decider, etc)
	for _, key := range commit_cache {

		if _, ok5 := commits[key]; ok5 {
			continue
		}

		merkle_unmine(key)

		segments_transaction_mutex.RLock()

		txlegs_each_leg_target(key, func(tx *[32]byte) bool {
			var oldactivity = segments_transaction_activity[*tx]
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
				segments_transaction_activity[*tx] = newactivity

				if oldactivity == 2097151 {
					logf("block rollbacks transaction %X \n", *tx)

					var actuallyfrom = segments_transaction_data[*tx][21]

					segments_transaction_untrickle(nil, actuallyfrom, 0xffffffffffffffff)

					delete(segments_transaction_next, actuallyfrom)
				}
			}

			return true
		})

		segments_transaction_mutex.RUnlock()
	}

	commit_cache = nil
	commit_cache_tags = nil

	commit_current_height--

	resetgraph()

	commits_mutex.Unlock()
	commit_cache_mutex.Unlock()
}

func miner_mine_commit(rawcommit [32]byte, tag UTXOtag) {
	var is_first bool

	commit_cache_mutex.Lock()

	is_first = len(commit_cache) == 0

	//can only mine a commit thats higher than the current block
	if is_first && commit_current_height >= tag.Height {
		commit_cache_mutex.Unlock()
		logf("error mined first commitment must be on greater height, %d >= %d\n", commit_current_height, tag.Height)
		return
	}
	//every batch of commits is in the same block
	if !is_first && tag.Height != commit_cache_tags[0].Height {
		commit_cache_mutex.Unlock()
		logf("error commitment must be on same height as first commitment, %d != %d\n", commit_cache_tags[0].Height, tag.Height)
		return
	}

	commit_cache = append(commit_cache, rawcommit)
	commit_cache_tags = append(commit_cache_tags, tag)
	commit_cache_mutex.Unlock()
}

func miner_unmine_commit(rawcommit [32]byte, tag UTXOtag) {
	var is_first bool

	commit_cache_mutex.Lock()

	is_first = len(commit_cache) == 0

	//cant unmine a commit thats higher than the current block
	if is_first && commit_current_height < tag.Height {
		commit_cache_mutex.Unlock()
		logf("error unmined first commitment must be on smaller height, %d >= %d\n", commit_current_height, tag.Height)
		return
	}
	//every batch of commits is in the same block
	if !is_first && tag.Height != commit_cache_tags[0].Height {
		commit_cache_mutex.Unlock()
		logf("error commitment must be on same height as first commitment, %d != %d\n", commit_cache_tags[0], tag.Height)
		return
	}

	commit_cache = append(commit_cache, rawcommit)
	commit_cache_tags = append(commit_cache_tags, tag)
	commit_cache_mutex.Unlock()
}
