package libcomb

type utxotag struct {
	direction bool
	height    uint32
	commitnum uint16
}

const UTAG_UNMINE int = -1
const UTAG_MINE int = 1

func utag_mining_sign(t utxotag) int {
	if t.direction {
		return UTAG_UNMINE
	}
	return UTAG_MINE
}

func utag_cmp(l *utxotag, r *utxotag) int {
	if l.height != r.height {
		return int(l.height) - int(r.height)
	}
	return int(l.commitnum) - int(r.commitnum)
}