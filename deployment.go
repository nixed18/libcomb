package libcomb

import "fmt"

const version_major byte = 0
const version_minor byte = 4
const version_patch byte = 0

const various_debug_prints_and_self_checking bool = false

var logging_enabled = true

func logf(f string, a ...interface{}) {
	if logging_enabled {
    	fmt.Printf(f, a...)
	}
}

func log(a ...interface{}) {
	if logging_enabled {
		fmt.Println(a...)
	}
}