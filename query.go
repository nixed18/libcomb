package libcomb

func query_commits_exist(commits_to_check [][32]byte) ([][32]byte, bool) {
	var all_committed bool = true
	var missing [][32]byte = make([][32]byte, 0)

	for _, c := range commits_to_check {
		if _, ok := commits[c]; !ok {
			all_committed = false
			missing = append(missing, c)
		}
	}

	return missing, all_committed
}

func query_commits_any_older_than(commits_to_check [][32]byte, commit [32]byte) (ok bool) {
	var tag Tag
	if tag, ok = commits[commit]; !ok {
		return true //any commit is older than a nonexistant commit
	}

	for _, c := range commits_to_check {
		if t, ok := commits[c]; ok {
			if t.OlderThan(tag) {
				return true
			}
		}
	}

	return false //commit is older than every commit in commits_to_check
}

func query_get_coinbase(c [32]byte) uint64 {
	var tag Tag
	var ok bool

	if tag, ok = commits[c]; !ok || tag.Order != 0 {
		return 0
	}

	return coinbase_reward(tag.Height)
}
