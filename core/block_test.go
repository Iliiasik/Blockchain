package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"
)

func sampleBlock(t *testing.T) *Block {
	t.Helper()
	coinbase := NewCoinbaseTX(string(NewWallet().GetAddress()), "sample", 10)
	block, err := NewBlock([]*Transaction{coinbase}, []byte{1, 2, 3}, testBits)
	if err != nil {
		t.Fatalf("NewBlock: %v", err)
	}
	return block
}

func TestNewBlockIsMinedAndValid(t *testing.T) {
	block := sampleBlock(t)

	if len(block.Hash) != sha256.Size {
		t.Fatalf("hash length = %d", len(block.Hash))
	}
	if block.Bits != testBits {
		t.Fatalf("bits = %d", block.Bits)
	}
	if !hasLeadingZeroBits([32]byte(block.Hash), testBits) {
		t.Fatalf("hash %x does not meet %d bits", block.Hash, testBits)
	}
	if !NewProofOfWork(block).Validate() {
		t.Fatal("mined block does not validate")
	}
}

func TestGenesisBlockHasNoParent(t *testing.T) {
	coinbase := NewCoinbaseTX(string(NewWallet().GetAddress()), "genesis", 10)
	genesis, err := NewGenesisBlock(coinbase, testBits)
	if err != nil {
		t.Fatal(err)
	}
	if len(genesis.PrevBlockHash) != 0 {
		t.Fatalf("genesis parent = %x", genesis.PrevBlockHash)
	}
	if len(genesis.Transactions) != 1 || !genesis.Transactions[0].IsCoinbase() {
		t.Fatal("genesis must hold exactly one coinbase")
	}
}

func TestBlockSerializationRoundTrip(t *testing.T) {
	block := sampleBlock(t)
	decoded := DeserializeBlock(block.Serialize())

	if decoded.Timestamp != block.Timestamp || decoded.Nonce != block.Nonce || decoded.Bits != block.Bits {
		t.Fatalf("header fields differ: %+v vs %+v", decoded, block)
	}
	if !bytes.Equal(decoded.Hash, block.Hash) || !bytes.Equal(decoded.PrevBlockHash, block.PrevBlockHash) {
		t.Fatal("hashes differ after round trip")
	}
	if len(decoded.Transactions) != 1 || !bytes.Equal(decoded.Transactions[0].Serialize(), block.Transactions[0].Serialize()) {
		t.Fatal("transactions differ after round trip")
	}
	if !bytes.Equal(decoded.Serialize(), block.Serialize()) {
		t.Fatal("serialization is not stable")
	}
	if !NewProofOfWork(decoded).Validate() {
		t.Fatal("decoded block no longer validates")
	}
}

func TestDeserializeBlockSurvivesGarbage(t *testing.T) {
	data := sampleBlock(t).Serialize()
	for _, cut := range []int{0, 1, 7, 9, len(data) / 2, len(data) - 1} {
		block := DeserializeBlock(data[:cut])
		if block == nil {
			t.Fatalf("cut %d returned nil", cut)
		}
	}
	DeserializeBlock(bytes.Repeat([]byte{0xff}, 64))
}

func TestHashTransactionsDependsOnEveryTransaction(t *testing.T) {
	block := sampleBlock(t)
	extra := NewCoinbaseTX(string(NewWallet().GetAddress()), "other", 5)
	block.Transactions = append(block.Transactions, extra)
	root := block.HashTransactions()

	extra.Vout[0].Value = 6
	if bytes.Equal(root, block.HashTransactions()) {
		t.Fatal("changing a transaction did not change the Merkle root")
	}
	if (&Block{}).HashTransactions() != nil {
		t.Fatal("empty block must have no Merkle root")
	}
}

func TestNewBlockContextStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	coinbase := NewCoinbaseTX(string(NewWallet().GetAddress()), "cancel", 10)
	start := time.Now()
	_, err := NewBlockContext(ctx, []*Transaction{coinbase}, nil, 60, nil)
	if !errors.Is(err, ErrMiningCancelled) {
		t.Fatalf("err = %v, want ErrMiningCancelled", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("cancellation took too long")
	}
}

func TestProofOfWorkRejectsTampering(t *testing.T) {
	coinbase := NewCoinbaseTX(string(NewWallet().GetAddress()), "honest", 10)
	block, err := NewBlock([]*Transaction{coinbase}, nil, 16)
	if err != nil {
		t.Fatal(err)
	}

	forged := *block
	forged.Transactions = []*Transaction{NewCoinbaseTX(string(NewWallet().GetAddress()), "forged", 1000)}
	if NewProofOfWork(&forged).Validate() {
		t.Fatal("a block with replaced transactions still validates")
	}

	moved := *block
	moved.Nonce++
	if NewProofOfWork(&moved).Validate() {
		t.Fatal("a block with a different nonce still validates")
	}

	harder := *block
	harder.Bits = 64
	if NewProofOfWork(&harder).Validate() {
		t.Fatal("a block claiming more work still validates")
	}
}

func TestHasLeadingZeroBits(t *testing.T) {
	var hash [32]byte
	hash[2] = 0x10
	cases := map[int]bool{0: true, 8: true, 16: true, 19: true, 20: false, 24: false}
	for bits, want := range cases {
		if got := hasLeadingZeroBits(hash, bits); got != want {
			t.Errorf("bits %d = %v, want %v", bits, got, want)
		}
	}
}

func TestMiningProgressReportsLastNonce(t *testing.T) {
	var progress MiningProgress
	coinbase := NewCoinbaseTX(string(NewWallet().GetAddress()), "progress", 10)
	block, err := NewBlockContext(context.Background(), []*Transaction{coinbase}, nil, testBits, &progress)
	if err != nil {
		t.Fatal(err)
	}
	nonce, hash, elapsed := progress.Snapshot()
	if int(nonce) != block.Nonce || !bytes.Equal(hash[:], block.Hash) {
		t.Fatalf("progress = %d %x, block = %d %x", nonce, hash, block.Nonce, block.Hash)
	}
	if elapsed < 0 {
		t.Fatal("negative elapsed time")
	}

	var empty MiningProgress
	if n, _, d := empty.Snapshot(); n != 0 || d != 0 {
		t.Fatalf("fresh progress = %d %v", n, d)
	}
}

func TestMerkleTree(t *testing.T) {
	if NewMerkleTree(nil).RootNode != nil {
		t.Fatal("empty tree must have no root")
	}

	leaf := sha256.Sum256([]byte("a"))
	single := NewMerkleTree([][]byte{[]byte("a")})
	if !bytes.Equal(single.RootNode.Data, leaf[:]) {
		t.Fatal("single leaf root must be the leaf hash")
	}

	hashA, hashB := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	pair := sha256.Sum256(append(hashA[:], hashB[:]...))
	two := NewMerkleTree([][]byte{[]byte("a"), []byte("b")})
	if !bytes.Equal(two.RootNode.Data, pair[:]) {
		t.Fatal("two leaf root is wrong")
	}

	odd := NewMerkleTree([][]byte{[]byte("a"), []byte("b"), []byte("c")})
	even := NewMerkleTree([][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("c")})
	if !bytes.Equal(odd.RootNode.Data, even.RootNode.Data) {
		t.Fatal("an odd level must duplicate its last node")
	}

	swapped := NewMerkleTree([][]byte{[]byte("b"), []byte("a")})
	if bytes.Equal(two.RootNode.Data, swapped.RootNode.Data) {
		t.Fatal("order of leaves must matter")
	}
}
