package main

import "fmt"

func main() {
	var myKey WalletKey = CreateWalletKey()
	var myOtherKey WalletKey = CreateWalletKey()
	var yourKey WalletKey = CreateWalletKey()

	//Mine some COMB into myKey
	var c Commit
	c.commit = CommitAddress(myKey.public) //Might want to use steath address's
	c.tag.height = 1 //First block
	LoadCommit(c)
	ProcessCommits()

	LoadWalletKey(myKey)
	fmt.Printf("myKey %d\n", GetAddressBalance(myKey.public))

	//Now lets transfer 1 COMB to yourkey

	//Contruct Stack
	var s Stack
	s.destination = yourKey.public
	s.change = myOtherKey.public
	s.sum = 100000000 //1 COMB

	//Construct Transaction
	var tx Transaction
	tx.source = myKey.public
	tx.destination = GetStackAddress(s)
	
	//Sign Transaction
	var signature = SignTransaction(tx)

	//Load Structures
	LoadStack(s)
	LoadTransaction(tx, signature)
	fmt.Printf("Transaction Loaded!\n")

	//Commit the signature
	c.tag.height = 2
	for i, leg := range signature {
		c.commit = CommitAddress(leg)
		c.tag.txnum = uint16(i+1)
		LoadCommit(c)
	}
	ProcessCommits()

	fmt.Printf("myOtherKey %d\n", GetAddressBalance(myOtherKey.public))
	fmt.Printf("yourKey %d\n",GetAddressBalance(yourKey.public))
}