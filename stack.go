package libcomb

import (
	"encoding/binary"
	"fmt"
)

type Stack struct {
	Destination [32]byte
	Sum         uint64
	Change      [32]byte
}

func (s Stack) ID() [32]byte {
	//stack address/id is hash({change, destination, sum})
	var data [72]byte
	copy(data[0:32], s.Change[:])
	copy(data[32:64], s.Destination[:])
	binary.BigEndian.PutUint64(data[64:], s.Sum)
	return Hash256(data[:])
}

func (s Stack) trigger() (err error) {
	var id = s.ID()

	if s.Active() {
		return nil //already triggered
	}

	if balance[id] < s.Sum {
		return fmt.Errorf("critical balance not reached")
	}

	//do a one time transfer to destination then redirect all funds to change
	balance_stack_redirect(id, s.Destination, s.Change, s.Sum)

	fmt.Println("stack activated")
	return nil
}

func (s Stack) triggers() [][32]byte {
	return nil //triggered when receiving funds
}

func (s Stack) Active() bool {
	if _, ok := balance_edge[s.ID()]; ok {
		return true
	}
	return false
}
