package libcomb

import (
	"fmt"
	"sync"
	"testing"
)

func init() {
	logging_enabled = true
}

var block_mutex sync.RWMutex //mining seperate blocks concurrently is not supported
var height uint64 = 1

func TestOutput(t *testing.T) {
	var myKey [32]byte = hex2byte32([]byte("5b87914dd5dafe0915da195f0eedc3ddb636e5075163b8f9ffecbce6e62600e2"))
	var yourKey [32]byte = hex2byte32([]byte("0232c6089bcc9af41fb82c89262996da170aa1686dcb8228c1fc17bf93151c95"))
	var emptyKey [32]byte

	var myDecider = GenerateDecider()
	fmt.Printf("decider: %s\n", myDecider.Export(emptyKey))

	var myShortDecider = ComputeShortDecider(myDecider)

	var contractTree [65536][32]byte
	contractTree[0] = myKey
	contractTree[1] = yourKey

	var contract = ConstructContract(contractTree, myShortDecider)
	var contractAddress = ComputeContractAddress(contract)
	fmt.Printf("contract: %X\n", contractAddress)

	var myLongDecider = SignDecider(myDecider, 1)
	var merkleSegment = DecideContract(contract, myLongDecider, contractTree)
	fmt.Printf("segment: %s\n", merkleSegment.Export())
}
func TestMining(t *testing.T) {
	var err error
	fmt.Println("MINING TEST START")
	var myKey WalletKey = GenerateWalletKey()
	var myOtherKey WalletKey = GenerateWalletKey()
	var yourKey WalletKey = GenerateWalletKey()

	block_mutex.Lock()
	var c Commit
	var commits []Commit
	c.Commit = CommitAddress(myKey.Public)
	c.Tag.Height = height
	commits = append(commits, c)
	if err = LoadBlock(height, commits); err != nil {
		t.Error(err.Error())
	}
	height++
	block_mutex.Unlock()

	LoadWalletKey(myKey)

	if GetAddressBalance(myKey.Public) == 0 {
		t.Error("coinbase error")
	}

	var s Stack
	s.Destination = yourKey.Public
	s.Change = myOtherKey.Public
	s.Sum = 100000000 //1 COMB

	var tx Transaction
	tx.Source = myKey.Public
	tx.Destination = GetStackAddress(s)
	var signature = SignTransaction(tx)

	LoadStack(s)
	LoadTransaction(tx, signature)

	block_mutex.Lock()
	c.Tag.Height = height
	commits = nil
	for i, leg := range signature {
		c.Commit = CommitAddress(leg)
		c.Tag.Commitnum = uint32(i)
		commits = append(commits, c)
	}
	if err = LoadBlock(height, commits); err != nil {
		t.Error(err.Error())
	}

	if GetAddressBalance(yourKey.Public) == 0 || GetAddressBalance(myOtherKey.Public) == 0 {
		t.Error("transaction error")
	}

	UnloadBlock()

	//fmt.Printf("myKey\t%d\nmyOtherKey\t%d\nyourKey\t%d\n", GetAddressBalance(myKey.Public), GetAddressBalance(myOtherKey.Public), GetAddressBalance(yourKey.Public))

	if GetAddressBalance(myKey.Public) == 0 {
		t.Error("rollback error")
	}

	LoadBlock(height, commits)

	if GetAddressBalance(yourKey.Public) == 0 || GetAddressBalance(myOtherKey.Public) == 0 {
		t.Error("redo error")
	}
	height++
	block_mutex.Unlock()
	fmt.Println("MINING TEST FINISH")
}

func TestMerkle(t *testing.T) {
	fmt.Println("MERKLE TEST START")
	var myKey WalletKey = GenerateWalletKey()
	var myOtherKey WalletKey = GenerateWalletKey()
	var yourKey WalletKey = GenerateWalletKey()
	LoadWalletKey(myKey)

	block_mutex.Lock()
	var c Commit
	var commits []Commit
	c.Commit = CommitAddress(myKey.Public)
	c.Tag.Height = height

	commits = append(commits, c)
	LoadBlock(height, commits)

	height++
	block_mutex.Unlock()

	if GetAddressBalance(myKey.Public) == 0 {
		t.Error("coinbase error")
	}

	var myDecider = GenerateDecider()
	var myShortDecider = ComputeShortDecider(myDecider)

	var contractTree [65536][32]byte
	contractTree[0] = myOtherKey.Public
	contractTree[1] = yourKey.Public

	var contract = ConstructContract(contractTree, myShortDecider)
	var contractAddress = ComputeContractAddress(contract)

	var tx Transaction
	tx.Source = myKey.Public
	tx.Destination = contractAddress
	var signature = SignTransaction(tx)

	LoadTransaction(tx, signature)
	block_mutex.Lock()
	commits = nil
	c.Tag.Height = height
	for i, leg := range signature {
		c.Commit = CommitAddress(leg)
		c.Tag.Commitnum = uint32(i)
		commits = append(commits, c)
	}
	LoadBlock(height, commits)
	height++
	block_mutex.Unlock()

	if GetAddressBalance(contractAddress) == 0 {
		t.Error("transaction error")
	}

	var myLongDecider = SignDecider(myDecider, 1)
	var merkleSegment = DecideContract(contract, myLongDecider, contractTree)

	block_mutex.Lock()
	commits = nil
	c.Tag.Height = height
	for i, leg := range myLongDecider.Signature {
		c.Commit = CommitAddress(leg)
		c.Tag.Commitnum = uint32(i)
		commits = append(commits, c)
	}
	LoadBlock(height, commits)
	height++
	block_mutex.Unlock()

	LoadMerkleSegment(merkleSegment)

	if GetAddressBalance(yourKey.Public) == 0 {
		t.Error("merkle error")
	}

	fmt.Println("MERKLE TEST FINISH")
}
func TestHashing(t *testing.T) {
	var key [21][32]byte
	var tip [21][32]byte
	var buf [672]byte
	var depths = [21]uint16{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200, 210}

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
	t.Run("Merkle", func(t *testing.T) {
		t.Parallel()
		TestMerkle(t)
	})
}
