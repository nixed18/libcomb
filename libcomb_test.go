package main

import (
	"testing"
)

func TestMiner(t *testing.T) {
	ResetCOMB()
	var myKey WalletKey = GenerateWalletKey()
	var myOtherKey WalletKey = GenerateWalletKey()
	var yourKey WalletKey = GenerateWalletKey()

	//Mine some COMB into myKey
	var c Commit
	c.commit = CommitAddress(myKey.public) //Might want to use steath address's
	c.tag.height = 1 //First block
	LoadCommit(c)
	FinishBlock()

	LoadWalletKey(myKey)
	logf("\tmyKey\t\t%d\n", GetAddressBalance(myKey.public))
	logf("\tmyOtherKey\t%d\n", GetAddressBalance(myOtherKey.public))
	logf("\tyourKey\t\t%d\n",GetAddressBalance(yourKey.public))

	if GetAddressBalance(myKey.public) != 210000000 {
		t.Error("coinbase error")
	}

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
	logf("Committing Signature...\n")
	c.tag.height = 2
	for i, leg := range signature {
		c.commit = CommitAddress(leg)
		c.tag.commitnum = uint16(i)
		LoadCommit(c)
	}
	FinishBlock()

	logf("\tmyKey\t\t%d\n", GetAddressBalance(myKey.public))
	logf("\tmyOtherKey\t%d\n", GetAddressBalance(myOtherKey.public))
	logf("\tyourKey\t\t%d\n",GetAddressBalance(yourKey.public))

	if GetAddressBalance(yourKey.public) != 100000000 || GetAddressBalance(myOtherKey.public) != 110000000 {
		t.Error("transaction error")
	}

	//Rollback the first signature commit
	c.commit = CommitAddress(signature[0])
	c.tag.commitnum = 0

	logf("Rolling back Signature...\n")
	UnloadCommit(c)
	FinishBlock()

	logf("\tmyKey\t\t%d\n", GetAddressBalance(myKey.public))
	logf("\tmyOtherKey\t%d\n", GetAddressBalance(myOtherKey.public))
	logf("\tyourKey\t\t%d\n",GetAddressBalance(yourKey.public))

	if GetAddressBalance(myKey.public) != 210000000 {
		t.Error("rollback error")
	}

	logf("Whoops lets add that commit back...\n")
	c.tag.direction = false
	LoadCommit(c)
	FinishBlock()

	logf("\tmyKey\t\t%d\n", GetAddressBalance(myKey.public))
	logf("\tmyOtherKey\t%d\n", GetAddressBalance(myOtherKey.public))
	logf("\tyourKey\t\t%d\n",GetAddressBalance(yourKey.public))
	logf("HASH COUNT: %d\n", hash_count)
	logf("CHAIN HASH COUNT: %d\n", chain_hash_count)

	if GetAddressBalance(yourKey.public) != 100000000 || GetAddressBalance(myOtherKey.public) != 110000000 {
		t.Error("redo error")
	}
}

func TestMerkle(t *testing.T) {
	ResetCOMB()
	var myKey WalletKey = GenerateWalletKey()
	var myOtherKey WalletKey = GenerateWalletKey()
	var yourKey WalletKey = GenerateWalletKey()
	LoadWalletKey(myKey)

	var c Commit
	c.commit = CommitAddress(myKey.public)
	c.tag.height = 1
	LoadCommit(c)
	FinishBlock()

	logf("\tmyKey\t\t%d\n", GetAddressBalance(myKey.public))

	var myDecider = GenerateDecider()
	var myShortDecider = ComputeShortDecider(myDecider)

	var contractTree [65536][32]byte
	//every other address is zero, dont sign the wrong branch :)
	contractTree[0] = myOtherKey.public
	contractTree[1] = yourKey.public

	var contract = ConstructContract(contractTree, myShortDecider)
	var contractAddress = ComputeContractAddress(contract)

	var tx Transaction
	tx.source = myKey.public
	tx.destination = contractAddress
	var signature = SignTransaction(tx)

	LoadTransaction(tx, signature)
	logf("Committing Tx Signature...\n")
	c.tag.height = 2
	for i, leg := range signature {
		c.commit = CommitAddress(leg)
		c.tag.commitnum = uint16(i)
		LoadCommit(c)
	}
	FinishBlock()

	logf("\tmyKey\t\t%d\n", GetAddressBalance(myKey.public))
	logf("\tcontract\t%d\n", GetAddressBalance(contractAddress))

	var myLongDecider = SignDecider(myDecider, 1)
	var merkleSegment = DecideContract(contract, myLongDecider, contractTree)
	
	logf("Committing Long Decider...\n")
	c.tag.height = 3
	for i, leg := range myLongDecider.signature {
		c.commit = CommitAddress(leg)
		c.tag.commitnum = uint16(i)
		LoadCommit(c)
	}
	FinishBlock()
	LoadMerkleSegment(merkleSegment)

	logf("\tcontract\t%d\n", GetAddressBalance(contractAddress))
	logf("\tmyOtherKey\t%d\n", GetAddressBalance(myOtherKey.public))
	logf("\tyourKey\t\t%d\n", GetAddressBalance(yourKey.public))	
}