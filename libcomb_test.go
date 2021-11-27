package main

import (
	"testing"
	"fmt"
	"sync"
)

func init() {
	logging_enabled = false
}

var block_mutex sync.RWMutex //mining seperate blocks concurrently is not supported
var height uint32 = 1

func TestMining(t *testing.T) {
	fmt.Println("MINING TEST START")
	var myKey WalletKey = GenerateWalletKey()
	var myOtherKey WalletKey = GenerateWalletKey()
	var yourKey WalletKey = GenerateWalletKey()

	block_mutex.Lock()
	var c Commit
	c.commit = CommitAddress(myKey.public)
	c.tag.height = height
	height++
	LoadCommit(c)
	FinishBlock()
	block_mutex.Unlock()

	LoadWalletKey(myKey)

	if GetAddressBalance(myKey.public) == 0 {
		t.Error("coinbase error")
	}

	var s Stack
	s.destination = yourKey.public
	s.change = myOtherKey.public
	s.sum = 100000000 //1 COMB

	var tx Transaction
	tx.source = myKey.public
	tx.destination = GetStackAddress(s)
	var signature = SignTransaction(tx)

	LoadStack(s)
	LoadTransaction(tx, signature)

	block_mutex.Lock()
	c.tag.height = height
	height++
	for i, leg := range signature {
		c.commit = CommitAddress(leg)
		c.tag.commitnum = uint16(i)
		LoadCommit(c)
	}
	FinishBlock()

	if GetAddressBalance(yourKey.public) == 0 || GetAddressBalance(myOtherKey.public) == 0 {
		t.Error("transaction error")
	}

	c.commit = CommitAddress(signature[0])
	c.tag.commitnum = 0

	UnloadCommit(c)
	FinishBlock()

	if GetAddressBalance(myKey.public) == 0 {
		t.Error("rollback error")
	}

	c.tag.direction = false
	LoadCommit(c)
	FinishBlock()

	if GetAddressBalance(yourKey.public) == 0 || GetAddressBalance(myOtherKey.public) == 0 {
		t.Error("redo error")
	}
	block_mutex.Unlock()
	fmt.Println("MINING TEST FINISH")
}

func TestMiningOrder(t *testing.T) {
	fmt.Println("ORDERED MINING TEST START")
	var myKey WalletKey = GenerateWalletKey()
	var myOtherKey WalletKey = GenerateWalletKey()
	var yourKey WalletKey = GenerateWalletKey()

	var s Stack
	s.destination = yourKey.public
	s.change = myOtherKey.public
	s.sum = 100000000

	var tx Transaction
	tx.source = myKey.public
	tx.destination = GetStackAddress(s)

	LoadStack(s)
	LoadWalletKey(myKey)
	var signature = SignTransaction(tx)
	LoadTransaction(tx, signature)

	block_mutex.Lock()
	var c Commit
	c.commit = CommitAddress(myKey.public)
	c.tag.height = height
	height++
	LoadCommit(c)
	for i, leg := range signature {
		c.commit = CommitAddress(leg)
		c.tag.commitnum = uint16(i+1)
		LoadCommit(c)
	}
	FinishBlock()
	block_mutex.Unlock()

	if GetAddressBalance(yourKey.public) == 0 || GetAddressBalance(myOtherKey.public) == 0 {
		t.Error("tx order error")
	}
	fmt.Println("ORDERED MINING TEST FINISH")
}

func TestMerkle(t *testing.T) {
	fmt.Println("MERKLE TEST START")
	var myKey WalletKey = GenerateWalletKey()
	var myOtherKey WalletKey = GenerateWalletKey()
	var yourKey WalletKey = GenerateWalletKey()
	LoadWalletKey(myKey)

	block_mutex.Lock()
	var c Commit
	c.commit = CommitAddress(myKey.public)
	c.tag.height = height
	height++
	LoadCommit(c)
	FinishBlock()
	block_mutex.Unlock()

	if GetAddressBalance(myKey.public) == 0 {
		t.Error("coinbase error")
	}

	var myDecider = GenerateDecider()
	var myShortDecider = ComputeShortDecider(myDecider)

	var contractTree [65536][32]byte
	contractTree[0] = myOtherKey.public
	contractTree[1] = yourKey.public

	var contract = ConstructContract(contractTree, myShortDecider)
	var contractAddress = ComputeContractAddress(contract)

	var tx Transaction
	tx.source = myKey.public
	tx.destination = contractAddress
	var signature = SignTransaction(tx)

	LoadTransaction(tx, signature)
	block_mutex.Lock()
	c.tag.height = height
	height++
	for i, leg := range signature {
		c.commit = CommitAddress(leg)
		c.tag.commitnum = uint16(i)
		LoadCommit(c)
	}
	FinishBlock()
	block_mutex.Unlock()

	if GetAddressBalance(contractAddress) == 0 {
		t.Error("transaction error")
	}

	var myLongDecider = SignDecider(myDecider, 1)
	var merkleSegment = DecideContract(contract, myLongDecider, contractTree)
	
	block_mutex.Lock()
	c.tag.height = height
	height++
	for i, leg := range myLongDecider.signature {
		c.commit = CommitAddress(leg)
		c.tag.commitnum = uint16(i)
		LoadCommit(c)
	}
	FinishBlock()
	block_mutex.Unlock()

	LoadMerkleSegment(merkleSegment)

	if GetAddressBalance(yourKey.public) == 0 {
		t.Error("merkle error")
	}

	fmt.Println("MERKLE TEST FINISH")
}


func TestMerkleOrder(t *testing.T) {
	fmt.Println("ORDERED MERKLE TEST START")
	var myKey WalletKey = GenerateWalletKey()
	var myOtherKey WalletKey = GenerateWalletKey()
	var yourKey WalletKey = GenerateWalletKey()

	var myDecider = GenerateDecider()
	var myShortDecider = ComputeShortDecider(myDecider)

	var contractTree [65536][32]byte
	contractTree[0] = myOtherKey.public
	contractTree[1] = yourKey.public

	var contract = ConstructContract(contractTree, myShortDecider)
	var contractAddress = ComputeContractAddress(contract)

	var tx Transaction
	tx.source = myKey.public
	tx.destination = contractAddress
	LoadWalletKey(myKey)
	var signature = SignTransaction(tx)

	var myLongDecider = SignDecider(myDecider, 1)
	var merkleSegment = DecideContract(contract, myLongDecider, contractTree)

	LoadTransaction(tx, signature)
	LoadMerkleSegment(merkleSegment)

	block_mutex.Lock()
	var c Commit
	var i uint16 = 1
	c.commit = CommitAddress(myKey.public)
	c.tag.height = height
	height++
	LoadCommit(c)
	for _, leg := range signature {
		c.commit = CommitAddress(leg)
		c.tag.commitnum = i
		i++
		LoadCommit(c)
	}
	for _, leg := range myLongDecider.signature {
		c.commit = CommitAddress(leg)
		c.tag.commitnum = i
		i++
		LoadCommit(c)
	}
	FinishBlock()
	block_mutex.Unlock()

	if GetAddressBalance(yourKey.public) == 0 {
		t.Error("merkle order error")
	}

	fmt.Println("ORDERED MERKLE TEST FINISH")
}

func TestHashing(t *testing.T) {
	var key [21][32]byte
	var tip [21][32]byte
	var buf [672]byte
	var depths = [21]uint16{10,20,30,40,50,60,70,80,90,100,110,120,130,140,150,160,170,180,190,200,210}

	tip = hash_chains(key, depths)
	for i := range tip {
		copy(buf[i*32:i*32+32], tip[i][:])
	}
	pubA := hash256(buf[:])

	for i := range tip {
		tip[i] = hash_chain(key[i], depths[i])
		copy(buf[i*32:i*32+32], tip[i][:])
	}
	pubB := hash256(buf[:])

	for i := range tip {
		tip[i] = key[i]
		for j := uint16(0); j < depths[i]; j++ {
			tip[i] = hash256(tip[i][:])
		}
		copy(buf[i*32:i*32+32], tip[i][:])
	}
	pubC := hash256(buf[:])
	
	if pubA != pubB || pubA != pubC {
		t.Error("hash chain mismatch")
	}
}

func TestParallel(t *testing.T) {
	fmt.Println("PARALLEL TEST")
	t.Run("Mining", func(t *testing.T) {
            t.Parallel()
			TestMining(t)
	})
	t.Run("OrderedMining", func(t *testing.T) {
            t.Parallel()
			TestMiningOrder(t)
	})
	t.Run("Merkle", func(t *testing.T) {
            t.Parallel()
			TestMerkle(t)
	})
	t.Run("OrderedMerkle", func(t *testing.T) {
            t.Parallel()
			TestMerkleOrder(t)
	})
}