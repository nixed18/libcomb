package libcomb

func get_parents(address [32]byte) [][32]byte {
	var parents [][32]byte = make([][32]byte, 0)
	for s, d := range balance_edge {
		if d == address {
			parents = append(parents, s)
		}
	}
	for s, d := range balance_one_off {
		if d == address {
			parents = append(parents, s)
		}
	}
	return parents
}

func get_history(address [32]byte) (history map[[32]byte]struct{}) {
	history = make(map[[32]byte]struct{}, 0)
	var processing [][32]byte = make([][32]byte, 1)
	history[address] = struct{}{}
	processing[0] = address
	for len(processing) > 0 {
		var parents [][32]byte = make([][32]byte, 0)
		for _, a := range processing {
			parents = append(parents, get_parents(a)...)
		}
		processing = make([][32]byte, 0)
		for _, p := range parents {
			if _, ok := history[p]; !ok {
				history[p] = struct{}{}
				processing = append(processing, p)
			}
		}
	}
	return history
}
