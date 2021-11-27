package main

import (
	"fmt"
)

func amount_decided_later_exact(min_amount, max_amount uint64, branch_id int64) uint64 {
	var amt uint64

	if (min_amount < (1 << 47)) && (max_amount < (1 << 47)) {

		amt = uint64(int64(min_amount<<16)+(int64(branch_id)*int64((max_amount-min_amount)))) >> 16

	} else {
		amt = uint64(int64(min_amount<<12)+(int64(branch_id)*int64((max_amount-min_amount)>>4))) >> 12
	}
	return amt
}

func main() {

	a := amount_decided_later_exact(0, 1000000, 65536)
	fmt.Printf("%d\n", a)
}