package libcomb

func coinbase_give_reward(c [32]byte, t Tag) {
	balance[c] += coinbase_reward(t.Height)
	coinbase_check_commit(c)
}

func coinbase_check_commit(c [32]byte) {
	//if an address is mapped to c (commit(a) == c), then redirect funds to a

	//check if c is a coinbase
	if tag, ok := commits[c]; !ok || tag.Order != 0 {
		return //not a coinbase
	}
	//check if c is a committed address
	if a, ok := construct_uncommits[c]; ok {
		//redirect funds to the address
		balance_redirect(c, a)
	}
}

func coinbase_check_address(a [32]byte) {
	//if a coinbase exists on commit(a) then redirect funds to a
	var c [32]byte = commit(a)
	if tag, ok := commits[c]; !ok || tag.Order != 0 {
		return //not a coinbase
	}
	balance_redirect(c, a)
}

const precision = 50

func mult128to128(o1hi uint64, o1lo uint64, o2hi uint64, o2lo uint64, hi *uint64, lo *uint64) {
	mult64to128(o1lo, o2lo, hi, lo)

	*hi += o1hi * o2lo
	*hi += o2hi * o1lo
}

func mult64to128(op1 uint64, op2 uint64, hi *uint64, lo *uint64) {
	var u1 = (op1 & 0xffffffff)
	var v1 = (op2 & 0xffffffff)
	var t = (u1 * v1)
	var w3 = (t & 0xffffffff)
	var k = (t >> 32)

	op1 >>= 32
	t = (op1 * v1) + k
	k = (t & 0xffffffff)
	var w1 = (t >> 32)

	op2 >>= 32
	t = (u1 * op2) + k
	k = (t >> 32)

	*hi = (op1 * op2) + w1 + k
	*lo = (t << 32) + w3
}

func log2(xx uint64) (uint64, uint64) {
	var b uint64 = 1 << (precision - 1)
	var yhi uint64 = 0
	var ylo uint64 = 0
	var zhi uint64 = xx >> (64 - precision)
	var zlo uint64 = xx << precision

	for (zhi > 0) || (zlo >= 2<<precision) {
		zlo = (zhi << (64 - 1)) | (zlo >> 1)
		zhi = zhi >> 1
		if ylo+(1<<precision) < ylo {
			yhi++
		}

		ylo += 1 << precision
	}

	for i := 0; i < precision; i++ {

		mult128to128(zhi, zlo, zhi, zlo, &zhi, &zlo)

		zlo = (zhi << (64 - precision)) | (zlo >> precision)
		zhi = zhi >> precision

		if (zhi > 0) || (zlo >= 2<<precision) {

			zlo = (zhi << (64 - 1)) | (zlo >> 1)
			zhi = zhi >> 1

			if ylo+b < ylo {
				yhi++
			}

			ylo += b
		}
		b >>= 1
	}

	return yhi, ylo
}

func coin_supply(height uint64) (uint64, uint64) {
	var loghi, loglo = log2(height)

	var hi, lo uint64

	mult128to128(loghi, loglo, loghi, loglo, &hi, &lo)

	lo = lo>>(precision) | hi<<(64-precision)
	hi = hi >> (precision)

	var hi2, lo2 uint64

	mult128to128(hi, lo, hi, lo, &hi2, &lo2)

	lo2 = lo2>>(precision) | hi2<<(64-precision)
	hi2 = hi2 >> (precision)

	var hi3, lo3 uint64

	mult128to128(hi, lo, hi2, lo2, &hi3, &lo3)

	lo3 = lo3>>(precision) | hi3<<(64-precision)
	hi3 = hi3 >> (precision)

	lo3 = lo3>>(precision) | hi3<<(64-precision)
	hi3 = hi3 >> (precision)

	return lo3, loglo
}

func coinbase_reward(height uint64) uint64 {
	if height >= 21835313 {
		return 0
	}

	var decrease, _ = coin_supply(height)

	return 210000000 - decrease
}
