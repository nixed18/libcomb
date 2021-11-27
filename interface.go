package main

type WalletKey struct {
	public  [32]byte
	private [21][32]byte
	balance uint64
}

type Stack struct {
	destination [32]byte
	sum         uint64
	change      [32]byte
}

type Transaction struct {
	source      [32]byte
	destination [32]byte
}

type Commit struct {
	commit	[32]byte
	tag		utxotag
}

type Decider struct {
	private [2][32]byte
}

type ShortDecider struct {
	public [2][32]byte
}

type LongDecider struct {
	signature [2][32]byte
}

type MerkleSegment struct {
	short [2][32]byte
	long [2][32]byte
	branches [16][32]byte
	leaf [32]byte
	next [32]byte
}

type Contract struct {
	short [2][32]byte
	next [32]byte
	root [32]byte
}

func ComputeWalletKey(key [21][32]byte) (w WalletKey) {
	w.private = key
	w.public = wallet_compute_public_key(key)
	return w
}

func GenerateWalletKey() (w WalletKey) {
	w.public, w.private = wallet_generate_key()
	return w
}

func GetAddressBalance(address [32]byte) uint64 {
	return balance_read(address)
}

func SignTransaction(tx Transaction) [21][32]byte {
	var signature = wallet_sign_transaction(tx.source, tx.destination)
	return signature
}

func LoadTransaction(tx Transaction, signature [21][32]byte ) {
	transaction_load(tx.source, tx.destination, signature)
}

func LoadWalletKey(k WalletKey) {
	wallet_load_key(k.private)
	return
}

func LoadStack(s Stack) {
	stack_load_data(s.destination, s.change, s.sum)
	return
}

func GetStackAddress(s Stack) [32]byte {
	return stack_address(s.destination, s.change, s.sum)
}

func LoadCommit(commit Commit) {
	miner_mine_commit(commit.commit, commit.tag)
}

func UnloadCommit(commit Commit) {
	miner_unmine_commit(commit.commit, commit.tag)
}

func FinishBlock() {
	miner_finish_block()
}

func BlockLoadCommits(commits []Commit) {
	for _, c := range commits {
		LoadCommit(c)
	}
	FinishBlock()
}

func CommitAddress(a [32]byte) [32]byte {
	return commit(a[:])
}

func GenerateDecider() (d Decider) {
	d.private = purse_generate_decider()
	return d
}

func ComputeShortDecider(d Decider) (s ShortDecider) {
	s.public = purse_compute_short_decider(d.private)
	return s
}

func SignDecider(d Decider, number uint16) (l LongDecider) {
	l.signature = purse_sign_decider(d.private, number)
	return l
}

func ConstructContract(tree [65536][32]byte, s ShortDecider) (c Contract) {
	c.short = s.public
	c.root = merkle_compute_root(tree)
	return c
}

func ComputeContractAddress(c Contract) (contract_address [32]byte) {
	var short_address [32]byte = purse_compute_short_address(c.short, c.next)
	contract_address = contract_compute_address(short_address, c.root)
	return contract_address
}

func DecideContract(c Contract, l LongDecider, tree [65536][32]byte) (m MerkleSegment) {
	var number uint16
	var ok bool
	if number, ok = purse_recover_signed_number(c.short, l.signature); !ok {
		log("error long decider does not decide this contract")
		return m
	}

	m.short = c.short
	m.next = c.next
	m.long = l.signature
	_, m.branches, m.leaf = merkle_traverse_tree(tree, number)
	return m
}

func LoadMerkleSegment(m MerkleSegment) {
	var short_address [32]byte = purse_compute_short_address(m.short, m.next)
	notify_transaction(m.next, short_address, m.short[0], m.short[1], m.long[0], m.long[1], m.branches, m.leaf)
}

func ResetCOMB() {
	reset_all()
}