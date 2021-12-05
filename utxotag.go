package libcomb

type UTXOtag struct {
	Height    uint64
	Commitnum uint32
}

func utag_cmp(l *UTXOtag, r *UTXOtag) int {
	if l.Height != r.Height {
		return int(l.Height) - int(r.Height)
	}
	return int(l.Commitnum) - int(r.Commitnum)
}
