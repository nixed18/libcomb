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
	fmt.Printf("\tmyKey\t\t%d\n", GetAddressBalance(myKey.public))
	fmt.Printf("\tmyOtherKey\t%d\n", GetAddressBalance(myOtherKey.public))
	fmt.Printf("\tyourKey\t\t%d\n",GetAddressBalance(yourKey.public))

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

	//Commit the signature
	fmt.Printf("Committing Siganture...\n")
	c.tag.height = 2
	for i, leg := range signature {
		c.commit = CommitAddress(leg)
		c.tag.txnum = uint16(i+1)
		LoadCommit(c)
	}
	ProcessCommits()

	fmt.Printf("\tmyKey\t\t%d\n", GetAddressBalance(myKey.public))
	fmt.Printf("\tmyOtherKey\t%d\n", GetAddressBalance(myOtherKey.public))
	fmt.Printf("\tyourKey\t\t%d\n",GetAddressBalance(yourKey.public))

	//Rollback the first signature commit
	c.commit = CommitAddress(signature[0])
	c.tag.txnum = 1
	c.tag.direction = true

	fmt.Printf("Rolling back Siganture...\n")
	LoadCommit(c)
	ProcessCommits()

	fmt.Printf("\tmyKey\t\t%d\n", GetAddressBalance(myKey.public))
	fmt.Printf("\tmyOtherKey\t%d\n", GetAddressBalance(myOtherKey.public))
	fmt.Printf("\tyourKey\t\t%d\n",GetAddressBalance(yourKey.public))

	fmt.Printf("Whoops lets add commit back...\n")
	c.tag.direction = false
	LoadCommit(c)
	ProcessCommits()

	fmt.Printf("\tmyKey\t\t%d\n", GetAddressBalance(myKey.public))
	fmt.Printf("\tmyOtherKey\t%d\n", GetAddressBalance(myOtherKey.public))
	fmt.Printf("\tyourKey\t\t%d\n",GetAddressBalance(yourKey.public))
}