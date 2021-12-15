package libcomb

import (
	"errors"
	"fmt"
	"sync"
)

type WalletKey struct {
	Public  [32]byte
	Private [21][32]byte
}

func (w WalletKey) Export() (out string) {
	out = "/wallet/data/"
	for _, k := range w.Private {
		out += fmt.Sprintf("%X", k)
	}
	return out
}

type Stack struct {
	Destination [32]byte
	Sum         uint64
	Change      [32]byte
}

func (s Stack) Export() (out string) {
	var raw = stack_encode(s.Destination, s.Change, s.Sum)
	out = fmt.Sprintf("/stack/data/%X", raw)
	return out
}

type Transaction struct {
	Source      [32]byte
	Destination [32]byte
}

func (tx Transaction) Export(signature [21][32]byte) (out string) {
	out = fmt.Sprintf("/tx/recv/%X%X", tx.Source, tx.Destination)
	for _, k := range signature {
		out += fmt.Sprintf("%X", k)
	}
	return out
}

type Commit struct {
	Commit [32]byte
	Tag    UTXOtag
}

type Decider struct {
	Private [2][32]byte
}

func (d Decider) Export(next [32]byte) (out string) {
	out = fmt.Sprintf("/purse/data/%X%X%X", next, d.Private[0], d.Private[1])
	return out
}

type ShortDecider struct {
	Public [2][32]byte
}

type LongDecider struct {
	Signature [2][32]byte
}

type MerkleSegment struct {
	Short    [2][32]byte
	Long     [2][32]byte
	Branches [16][32]byte
	Leaf     [32]byte
	Next     [32]byte
}

func (m MerkleSegment) Export() (out string) {
	out = fmt.Sprintf("/merkle/data/%X%X%X%X", m.Short[0], m.Short[1], m.Long[0], m.Long[1])
	for b, _ := range m.Branches {
		out += fmt.Sprintf("%X", b)
	}
	out += fmt.Sprintf("%X%X", m.Leaf, m.Next)
	return out
}

type Contract struct {
	Short [2][32]byte
	Next  [32]byte
	Root  [32]byte
}

type Block struct {
	Height  uint64
	Commits []Commit
}

func ComputeWalletKey(key [21][32]byte) (w WalletKey) {
	w.Private = key
	w.Public = wallet_compute_public_key(key)
	return w
}

func GenerateWalletKey() (w WalletKey) {
	w.Public, w.Private = wallet_generate_key()
	return w
}

func GetAddressBalance(address [32]byte) uint64 {
	return balance_read(address)
}

func SignTransaction(tx Transaction) [21][32]byte {
	var signature = wallet_sign_transaction(tx.Source, tx.Destination)
	return signature
}

func LoadTransaction(tx Transaction, signature [21][32]byte) ([32]byte, error) {
	return transaction_load(tx.Source, tx.Destination, signature)
}

func LoadWalletKey(k WalletKey) [32]byte {
	return wallet_load_key(k.Private)
}

func LoadStack(s Stack) [32]byte {
	return stack_load_data(s.Destination, s.Change, s.Sum)
}

func GetStackAddress(s Stack) [32]byte {
	return stack_address(s.Destination, s.Change, s.Sum)
}

func GetCOMBBase(height uint64) (commit [32]byte, err error) {
	commits_mutex.Lock()
	defer commits_mutex.Unlock()
	if commit, ok := combbase_height[height]; ok {
		return commit, nil
	} else {
		return commit, fmt.Errorf("no combbase at height %d", height)
	}
}

func HaveCommit(commit [32]byte) bool {
	commits_mutex.Lock()
	_, ok := commits[commit]
	commits_mutex.Unlock()
	return ok
}

func GetCommitCount() (l uint64) {
	commits_mutex.Lock()
	l = uint64(len(commits))
	commits_mutex.Unlock()
	return l
}

func GetHeight() uint64 {
	return height_view()
}

var modify_mutex sync.Mutex

func LoadBlock(block Block) (err error) {
	modify_mutex.Lock()
	defer modify_mutex.Unlock()
	if GetHeight() != 0 && block.Height != GetHeight()+1 {
		return fmt.Errorf("error blocks must be sequential %d != %d", block.Height, GetHeight())
	}
	var commitnum int64 = -1
	var thiscommitnum int64
	for _, c := range block.Commits {
		if c.Tag.Height != block.Height {
			return errors.New("error commit height different to block height")
		}
		thiscommitnum = int64(c.Tag.Commitnum)
		if thiscommitnum != commitnum+1 {
			return errors.New("error commits are not sequential")
		}
		commitnum = thiscommitnum
	}
	for _, c := range block.Commits {
		miner_mine_commit(c.Commit, c.Tag)
	}
	miner_mine_block(block.Height)
	return nil
}

func UnloadBlock() (err error) {
	modify_mutex.Lock()
	defer modify_mutex.Unlock()
	for commit, tag := range commits {
		if tag.Height == commit_current_height {
			miner_unmine_commit(commit, tag)
		}
	}
	miner_unmine_block()
	return nil
}

func GetCommitDifference() []Commit {
	return commit_diff
}

func CommitAddress(a [32]byte) [32]byte {
	return commit(a[:])
}

func GenerateDecider() (d Decider) {
	d.Private = purse_generate_decider()
	return d
}

func LoadDecider(d Decider) [32]byte {
	return purse_load_decider(d.Private)
}

func ComputeShortDecider(d Decider) (s ShortDecider) {
	s.Public = purse_compute_short_decider(d.Private)
	return s
}

func SignDecider(d Decider, number uint16) (l LongDecider) {
	l.Signature = purse_sign_decider(d.Private, number)
	return l
}

func ConstructContract(tree [65536][32]byte, s ShortDecider) (c Contract) {
	c.Short = s.Public
	c.Root = merkle_compute_root(tree)
	return c
}

func ComputeContractAddress(c Contract) (contract_address [32]byte) {
	var short_address [32]byte = purse_compute_short_address(c.Short, c.Next)
	contract_address = contract_compute_address(short_address, c.Root)
	return contract_address
}

func DecideContract(c Contract, l LongDecider, tree [65536][32]byte) (m MerkleSegment) {
	var number uint16
	var ok bool
	if number, ok = purse_recover_signed_number(c.Short, l.Signature); !ok {
		log("error long decider does not decide this contract")
		return m
	}

	m.Short = c.Short
	m.Next = c.Next
	m.Long = l.Signature
	_, m.Branches, m.Leaf = merkle_traverse_tree(tree, number)
	return m
}

func LoadMerkleSegment(m MerkleSegment) [32]byte {
	var short_address [32]byte = purse_compute_short_address(m.Short, m.Next)
	_, address := notify_transaction(m.Next, short_address, m.Short[0], m.Short[1], m.Long[0], m.Long[1], m.Branches, m.Leaf)
	return address
}

func ResetCOMB() {
	//should reset all state... hopefully
	balance_reset()
	mine_reset()
	segmentmerkle_reset()
	segmentstack_reset()
	segmenttx_reset()
	wallet_reset()
	resetgraph_reset()
}
