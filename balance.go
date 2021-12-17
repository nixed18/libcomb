package libcomb

import (
	"fmt"
	"sync"
)

type balance = uint64

var balance_mutex sync.RWMutex

var balance_edge map[[32]byte]balance
var balance_node map[[32]byte]balance
var balance_loop map[[32]byte]balance

func nats(b balance) uint32 {
	return uint32(b % 100000000)
}

func combs(b balance) uint32 {
	return uint32(b / 100000000)
}

func balance_reset() {
	balance_mutex.Lock()

	balance_edge = make(map[[32]byte]balance)
	balance_node = make(map[[32]byte]balance)
	balance_loop = make(map[[32]byte]balance)

	balance_mutex.Unlock()
}

func init() {
	balance_reset()
}

func balance_try_increase_loop(where_node [32]byte) bool {
	balance_mutex.Lock()

	if _, ok := balance_loop[where_node]; ok {
		balance_loop[where_node] += balance_node[where_node]
		balance_node[where_node] = 0
		balance_mutex.Unlock()
		return true
	}

	balance_mutex.Unlock()
	return false
}

func balance_create_loop(where_node [32]byte) {
	balance_mutex.Lock()

	balance_loop[where_node] += balance_node[where_node]
	balance_node[where_node] = 0

	balance_mutex.Unlock()
}

func balance_create_coinbase(where_node [32]byte, sum uint64) {
	balance_mutex.Lock()

	if fmt.Sprintf("%X", where_node) == "875860200A9EB83562FBC1F474F45F0EF80B61054BAE608DAB84F7CEAD2BE6A4" {
		fmt.Printf("haha\n")
	}
	balance_node[where_node] += sum

	balance_mutex.Unlock()
}

func balance_destroy_coinbase(where_node [32]byte, sum uint64) {
	balance_mutex.Lock()

	var amt = balance_node[where_node]

	if sum == amt {
		delete(balance_node, where_node)
	} else if sum < amt {
		balance_node[where_node] -= sum
	} else {

		logf("%X % 21d \n", where_node, amt)
		log(sum)
		println("balance_destroy_coinbase: insufficient balance at to be destroyed coinbase")
		panic("")
	}

	balance_mutex.Unlock()
}

func balance_do(from [32]byte, to [32]byte, bal balance) {

	if bal == 0 {
		return
	}

	if various_debug_prints_and_self_checking {
		logf("%X -> %X % 21d\n", from, to, bal)
	}

	balance_mutex.Lock()

	if bal != 0xffffffffffffffff {
		if balance_node[from] < bal {
			logf("%X % 21d \n", from, balance_node[from])
			println("balance_do: insufficient balance at source")
			panic("")
		}
		balance_node[from] -= bal
		balance_node[to] += bal
	} else {
		bal = balance_node[from]
		balance_node[to] += bal
		balance_node[from] -= bal
	}

	balance_edge[merkle(from[0:], to[0:])] += bal
	balance_mutex.Unlock()
}

func balance_check(from [32]byte, to [32]byte) balance {
	var m = merkle(from[0:], to[0:])

	balance_mutex.RLock()
	var bal = balance_edge[m]

	balance_mutex.RUnlock()

	if various_debug_prints_and_self_checking {
		logf("%X ~~ %X % 21d\n", from, to, bal)
	}

	return bal
}

func balance_split_if_enough(from [32]byte, to [32]byte, tobal [32]byte, bal balance) (out byte) {
	var m = merkle(from[0:], tobal[0:])
	if various_debug_prints_and_self_checking {
		logf("%X => %X %X, % 21d\n", from, to, tobal, bal)
	}

	balance_mutex.Lock()

	_, ok := balance_edge[m]

	if !ok {
		if various_debug_prints_and_self_checking {
			println("nothing")
		}
		if balance_node[from] < bal {
			balance_mutex.Unlock()
			return 0
		}
		balance_mutex.Unlock()

		balance_do(from, tobal, bal)
		out = 2
	} else {
		if various_debug_prints_and_self_checking {
			println("something already")
		}
		out = 1
		balance_mutex.Unlock()
	}

	balance_do(from, to, 0xffffffffffffffff)
	return out
}

func balance_read(key [32]byte) (b balance) {
	balance_mutex.RLock()
	b = 0
	if bal, ok := balance_node[key]; ok {
		b = bal
	}
	balance_mutex.RUnlock()

	return b
}
