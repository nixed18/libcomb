package libcomb

import (
	"crypto/rand"
	"fmt"
	"log"
)

type Decider struct {
	Private [2][32]byte
	Tips    [2][32]byte
	id      [32]byte
}

func (d Decider) ID() [32]byte {
	if d.id == [32]byte{} {
		log.Panicf("decider has not been recovered")
	}
	return d.id
}

func (d Decider) trigger() error {
	return nil
}

func (d Decider) triggers() [][32]byte {
	return nil
}

func decider_lookup(id [32]byte) (Decider, error) {
	var decider Decider
	if c, ok := constructs[id]; ok {
		if decider, ok = c.(Decider); !ok {
			return decider, fmt.Errorf("not a decider")
		}
	} else {
		return decider, fmt.Errorf("not a construct")
	}

	return decider, nil
}

func decider_create() (Decider, error) {
	var d Decider
	for i := range d.Private {
		if _, err := rand.Read(d.Private[i][0:]); err != nil {
			return d, fmt.Errorf("cannot get random value: %s", err)
		}
	}

	decider_recover(&d)
	return d, nil
}

func decider_sign(d Decider, number uint16) (signature [2][32]byte) {
	signature[0] = d.Private[0]
	signature[1] = d.Private[1]

	for i := uint16(0); i < number; i++ {
		signature[0] = Hash256(signature[0][:])
	}

	for i := uint16(0); i < uint16(65535-number); i++ {
		signature[1] = Hash256(signature[1][:])
	}

	return signature
}

func decider_tips(d Decider) (tips [2][32]byte) {
	tips[0] = d.Private[0]
	tips[1] = d.Private[1]

	for i := uint16(0); i < uint16(65535); i++ {
		tips[0] = Hash256(tips[0][:])
		tips[1] = Hash256(tips[1][:])
	}

	return tips
}

func decider_recover(d *Decider) {
	d.Tips = decider_tips(*d)
	d.id = Hash256Adjacent(d.Tips[0], d.Tips[1])
}
