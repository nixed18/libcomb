package main

import "fmt"

var logging_enabled = false

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