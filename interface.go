package main

var initial_writeback_over = true

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

func ComputeWalletKey(key [21][32]byte) (w WalletKey) {
	w.private = key
	w.public = wallet_compute_public_key(key)
	return w
}

func CreateWalletKey() (w WalletKey) {
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
	miner_add_commit(commit.commit, commit.tag)
}

func ProcessCommits() {
	miner_process()
}

func BatchLoadCommit(commits []Commit) {
	for _, c := range commits {
		LoadCommit(c)
	}
	ProcessCommits()
}

func CommitAddress(a [32]byte) [32]byte {
	return commit(a[:])
}