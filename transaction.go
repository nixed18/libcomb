package libcomb

import (
	"fmt"
)

type Transaction struct {
	Source      [32]byte
	Destination [32]byte
	Signature   [21][32]byte
}

func (tx Transaction) ID() [32]byte {
	return Hash256Adjacent(tx.Source, tx.Destination)
}

func (tx Transaction) Active() bool {
	if destination, ok := balance_edge[tx.Source]; ok && destination == tx.Destination {
		return true
	}
	return false
}

func (tx Transaction) trigger() (err error) {
	if tx.Active() {
		return //already triggered
	}
	var id [32]byte = tx.ID()
	var cuts = cut(id[:])

	var signature_commits [21][32]byte
	for i, s := range tx.Signature {
		signature_commits[i] = commit(s)
	}

	//check signature is committed
	if missing, ok := query_commits_exist(signature_commits[:]); !ok {
		return fmt.Errorf("missing %d signature commits", len(missing))
	}

	//check for double spends
	for i, s := range tx.Signature {
		var leg_commits [][32]byte = make([][32]byte, 0)
		for k := uint16(0); k < cuts[i]; k++ {
			s = Hash256(s[:])
			leg_commits = append(leg_commits, commit(s))
		}
		if query_commits_any_older_than(leg_commits, signature_commits[i]) {
			return fmt.Errorf("older signature on leg %d", i)
		}
	}

	//create the balance edge then propagate
	balance_redirect(tx.Source, tx.Destination)
	fmt.Println("tx activated")
	return nil
}

func (tx Transaction) triggers() (t [][32]byte) {
	for _, s := range tx.Signature {
		t = append(t, commit(s))
	}
	return t
}

func tx_recover(tx Transaction) error {
	var id [32]byte = tx.ID()
	cuts := cut(id[:])
	for i := range tx.Signature {
		for x := uint16(0); x < cuts[i]; x++ {
			tx.Signature[i] = Hash256(tx.Signature[i][:])
		}
	}

	var source [32]byte = Hash256Concat32(tx.Signature[:])

	if source != tx.Source {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

func tx_sign(tx *Transaction) error {
	if c, ok := constructs[tx.Source]; ok {
		if k, ok := c.(Key); ok {
			if !k.Active() {
				tx.Signature = key_sign(&k, tx.ID())
			} else {
				return fmt.Errorf("source already spent (%X)", tx.Source)
			}
		} else {
			return fmt.Errorf("source is not a key (%X)", tx.Source)
		}
	} else {
		return fmt.Errorf("source is not loaded (%X)", tx.Source)
	}
	return nil
}
