package main

import (
	)

var whitepaper = [32]byte{106,251,172,89,92,29,7,163,212,197,23,151,88,245,188,228,70,42,108,38,63,110,109,252,217,66,1,20,51,173,170,231}

func commit(hash []byte) [32]byte {
	var buf [64]byte
	var sli []byte
	sli = buf[0:0]
	
	sli = append(sli, whitepaper[0:]...)
	sli = append(sli, hash[0:]...)

	return hash256(sli)
}

func merkle(a []byte, b []byte) [32]byte {
	var buf [64]byte
	var sli []byte
	sli = buf[0:0]

	sli = append(sli, a[0:]...)
	sli = append(sli, b[0:]...)

	return hash256(sli)
}
