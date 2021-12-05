package libcomb

import "fmt"

const VersionMajor byte = 0
const VersionMinor byte = 4
const VersionPatch byte = 0
const Version string = "0.4.0"

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
