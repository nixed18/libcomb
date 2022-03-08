package libcomb

import (
	"fmt"
	"log"
)

const Version string = "0.0.0"

var testnet bool = false

type Block struct {
	Commits [][32]byte
}

type Tag struct {
	Height uint64
	Order  uint32
}

func (a Tag) OlderThan(b Tag) bool {
	if a.Height != b.Height {
		return a.Height < b.Height
	}
	return a.Order < b.Order
}

func Commit(address [32]byte) [32]byte {
	return commit(address)
}

func HaveCommit(commit [32]byte) bool {
	var have bool
	commits_guard.RLock()
	defer commits_guard.RUnlock()
	_, have = commits[commit]
	return have
}

func GetCommitCount() uint64 {
	commits_guard.RLock()
	defer commits_guard.RUnlock()
	return uint64(len(commits))
}

func GetCommits() map[[32]byte]Tag {
	out := make(map[[32]byte]Tag)
	commits_guard.RLock()
	defer commits_guard.RUnlock()
	for key, val := range commits {
		out[key] = val
	}
	return out
}

func GetBlockCommits(h uint64) map[[32]byte]Tag {
	out := make(map[[32]byte]Tag)
	commits_guard.RLock()
	defer commits_guard.RUnlock()
	for key, val := range commits {
		if val.Height == h {
			out[key] = val
		}
	}
	return out
}

func GetLock() {
	commits_guard.Lock()
	balance_guard.Lock()
}

func ReleaseLock() {
	balance_guard.Unlock()
	commits_guard.Unlock()
}

func LoadBlock(b Block) {
	load_block(b)
}

func UnloadBlock() uint64 {
	return unload_block()
}

func FinishReorg() {
	balance_rebuild()
}

func GetHeight() uint64 {
	commits_guard.RLock()
	defer commits_guard.RUnlock()
	return height
}

func SetHeight(h uint64) error {
	commits_guard.Lock()
	defer commits_guard.Unlock()

	if len(commits) > 0 {
		return fmt.Errorf("commit set is not empty")
	}
	height = h
	return nil
}

func GetBalance(address [32]byte) uint64 {
	balance_guard.RLock()
	defer balance_guard.RUnlock()
	return balance[address]
}

func Reset() {
	constructs_guard.Lock()
	commits_guard.Lock()
	balance_guard.Lock()
	defer balance_guard.Unlock()
	defer commits_guard.Unlock()
	defer constructs_guard.Unlock()

	constructs_initialize()
	balance_initialize()
	commits_initialize()
	height = 0
	testnet = false
}

func SwitchToTestnet() error {
	commits_guard.Lock()
	defer commits_guard.Unlock()

	if len(commits) > 0 {
		return fmt.Errorf("commit set is not empty")
	}
	testnet = true
	return nil
}

func LoadConstruct(c Construct) [32]byte {
	constructs_guard.Lock()
	defer constructs_guard.Unlock()
	return constructs_load(c)
}

func GetConstruct(id [32]byte) Construct {
	constructs_guard.Lock()
	defer constructs_guard.Unlock()
	return constructs[id]
}

func LoadKey(k Key) [32]byte {
	key_recover(&k)
	return LoadConstruct(k)
}

func NewKey() (k Key, err error) {
	return key_create()
}

func LoadTransaction(tx Transaction) (id [32]byte, err error) {
	if err = tx_recover(tx); err != nil {
		return id, err
	}

	return LoadConstruct(tx), nil
}

func SignTransaction(tx *Transaction) error {
	constructs_guard.RLock()
	defer constructs_guard.RUnlock()
	return tx_sign(tx)
}

func LoadStack(s Stack) [32]byte {
	return LoadConstruct(s)
}

func LoadDecider(d Decider) [32]byte {
	decider_recover(&d)
	return LoadConstruct(d)
}

func RecoverDecider(d Decider) Decider {
	decider_recover(&d)
	return d
}

func NewDecider() (d Decider, err error) {
	d, err = decider_create()
	return d, err
}

func SignDecider(d Decider, number uint16) (signature [2][32]byte, err error) {
	return decider_sign(d, number), nil
}

func LoadUnsignedMerkleSegment(m UnsignedMerkleSegment) (id [32]byte, err error) {
	if c, ok := constructs[m.ID()]; ok && c.triggers() != nil {
		return m.ID(), fmt.Errorf("cannot overwrite a signed merkle segment with an unsigned one")
	}

	return LoadConstruct(m), nil
}

func LoadMerkleSegment(m MerkleSegment) (id [32]byte, err error) {
	if err = merkle_recover(&m); err != nil {
		return id, err
	}

	//special consideration is needed since you can have different outputs while having the same ID
	if c, ok := constructs[m.ID()]; ok && c.triggers() != nil { //check if a signed merkle segment is loaded with our ID
		if s, ok := c.(MerkleSegment); ok && !merkle_compare(s, m) { //check if its different to our merkle segment
			log.Printf("tried to load a conflicting merkle segment!")
			return m.ID(), fmt.Errorf("cannot load conflicting merkle segment")
		}
	}

	return LoadConstruct(m), nil
}

func RecoverMerkleSegment(m *MerkleSegment) error {
	return merkle_recover(m)
}

func ComputeProof(tree [65536][32]byte, destination uint16) (root [32]byte, branches [16][32]byte, leaf [32]byte) {
	return merkle_traverse_tree(tree, destination)
}

func GetKeys() []Key {
	var keys []Key = make([]Key, 0)
	for _, c := range construct_load_order {
		if key, ok := constructs[c].(Key); ok {
			keys = append(keys, key)
		}
	}
	return keys
}

func GetStacks() []Stack {
	var stacks []Stack = make([]Stack, 0)
	for _, c := range construct_load_order {
		if stack, ok := constructs[c].(Stack); ok {
			stacks = append(stacks, stack)
		}
	}
	return stacks
}

func GetTransactions() []Transaction {
	var transactions []Transaction = make([]Transaction, 0)
	for _, c := range construct_load_order {
		if transaction, ok := constructs[c].(Transaction); ok {
			transactions = append(transactions, transaction)
		}
	}
	return transactions
}

func GetDeciders() []Decider {
	var deciders []Decider = make([]Decider, 0)
	for _, c := range construct_load_order {
		if decider, ok := constructs[c].(Decider); ok {
			deciders = append(deciders, decider)
		}
	}
	return deciders
}

func GetMerkleSegments() []MerkleSegment {
	var segments []MerkleSegment = make([]MerkleSegment, 0)
	for _, c := range construct_load_order {
		if segment, ok := constructs[c].(MerkleSegment); ok {
			segments = append(segments, segment)
		}
	}
	return segments
}

func GetUnsignedMerkleSegments() []UnsignedMerkleSegment {
	var segments []UnsignedMerkleSegment = make([]UnsignedMerkleSegment, 0)
	for _, c := range construct_load_order {
		if segment, ok := constructs[c].(UnsignedMerkleSegment); ok {
			segments = append(segments, segment)
		}
	}
	return segments
}

func LookupDecider(id [32]byte) (Decider, error) {
	return decider_lookup(id)
}

func LookupUnsignedMerkleSegment(id [32]byte) (UnsignedMerkleSegment, error) {
	return unsigned_merkle_segment_lookup(id)
}

func GetCOMBBase(height uint64) (commit [32]byte, err error) {
	commits_guard.RLock()
	defer commits_guard.RUnlock()
	for c, t := range commits {
		if t.Height == height && t.Order == 0 {
			return c, nil
		}
	}
	return commit, fmt.Errorf("no combbase at height %d", height)
}

func GetCommitTag(commit [32]byte) (t Tag, err error) {
	commits_guard.RLock()
	defer commits_guard.RUnlock()
	var b bool
	if t, b = commits[commit]; !b {
		return t, fmt.Errorf("no such commit")
	}
	return t, nil
}

func GetCoinHistory(address [32]byte) map[[32]byte]struct{} {
	balance_guard.RLock()
	defer balance_guard.RUnlock()
	return get_history(address)
}

func DEBUGAddCommits(arr [][32]byte) {
	var t Tag
	t.Height = GetHeight()
	t.Order = 0
	for _, tag := range commits {
		if tag.Height == t.Height && tag.Order > t.Order {
			t.Order = tag.Order
		}
	}
	for _, c := range arr {
		t.Order++
		commits[c] = t
		constructs_check_commit(c)
	}
}
