package core

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"testing"
)

func TestCreateBlockchainValidation(t *testing.T) {
	useTempDir(t)
	if BlockchainExists() {
		t.Fatal("fresh directory reports a blockchain")
	}
	if _, err := NewBlockchain(); !errors.Is(err, ErrNoBlockchain) {
		t.Fatalf("NewBlockchain err = %v", err)
	}
	if _, err := CreateBlockchain("not an address", 10, testBits); err == nil {
		t.Fatal("invalid genesis address accepted")
	}

	address := newWallets(t, 1)[0]
	bc, err := CreateBlockchain(address, 10, testBits)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateBlockchain(address, 10, testBits); err == nil {
		t.Fatal("second CreateBlockchain must fail")
	}
	if err := bc.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewBlockchain()
	if err != nil {
		t.Fatal(err)
	}
	blocks := reopened.Blocks()
	if len(blocks) != 1 || len(blocks[0].PrevBlockHash) != 0 {
		t.Fatalf("reopened chain has %d blocks", len(blocks))
	}
	if blocks[0].Transactions[0].GetRecipientAddress() != address {
		t.Fatal("genesis reward went to the wrong address")
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}

	if err := RemoveBlockchain(); err != nil {
		t.Fatal(err)
	}
	if BlockchainExists() {
		t.Fatal("RemoveBlockchain left the database")
	}
	if err := RemoveBlockchain(); err != nil {
		t.Fatalf("removing twice must be harmless: %v", err)
	}
}

func TestChainLinksAndIteration(t *testing.T) {
	f := newFixture(t)
	first := f.mine(t)
	second := f.mine(t)

	blocks := f.bc.Blocks()
	if len(blocks) != 3 {
		t.Fatalf("height = %d, want 3", len(blocks))
	}
	if !bytes.Equal(blocks[0].Hash, second.Hash) || !bytes.Equal(blocks[1].Hash, first.Hash) {
		t.Fatal("Blocks must start at the tip")
	}
	for i := 0; i < len(blocks)-1; i++ {
		if !bytes.Equal(blocks[i].PrevBlockHash, blocks[i+1].Hash) {
			t.Fatalf("block %d does not point to its parent", i)
		}
		if !NewProofOfWork(blocks[i]).Validate() {
			t.Fatalf("block %d has invalid proof of work", i)
		}
	}

	found, err := f.bc.FindTransaction(first.Transactions[0].ID)
	if err != nil || !bytes.Equal(found.ID, first.Transactions[0].ID) {
		t.Fatalf("FindTransaction: %v", err)
	}
	if _, err := f.bc.FindTransaction([]byte{1, 2, 3}); err == nil {
		t.Fatal("unknown transaction found")
	}
}

func TestMineBlockValidation(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	if _, err := f.bc.MineBlock(ctx, nil, testBits, nil); err == nil {
		t.Fatal("empty block accepted")
	}
	tx := f.send(t, f.alice, f.bob, 3, 0)
	if _, err := f.bc.MineBlock(ctx, []*Transaction{tx}, testBits, nil); err == nil {
		t.Fatal("block without coinbase accepted")
	}

	forged := *tx
	forged.Vout = append([]TXOutput{}, tx.Vout...)
	forged.Vout[0].Value = 1000
	coinbase := NewCoinbaseTX(f.miner, "", 10)
	if _, err := f.bc.MineBlock(ctx, []*Transaction{coinbase, &forged}, testBits, nil); err == nil {
		t.Fatal("block with a forged transaction accepted")
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := f.bc.MineBlock(cancelled, []*Transaction{coinbase}, 60, nil); !errors.Is(err, ErrMiningCancelled) {
		t.Fatalf("cancelled mining err = %v", err)
	}
	if len(f.bc.Blocks()) != 1 {
		t.Fatal("failed mining changed the chain")
	}

	block, err := f.bc.MineBlock(ctx, []*Transaction{coinbase}, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if block.Bits != testBits {
		t.Fatalf("zero bits must reuse the previous difficulty, got %d", block.Bits)
	}
}

func TestFindUTXOMatchesUTXOSet(t *testing.T) {
	f := newFixture(t)
	f.mine(t, f.send(t, f.alice, f.bob, 4, 1))
	f.mine(t, f.send(t, f.bob, f.alice, 2, 0))

	fromChain := f.bc.FindUTXO()
	total := 0
	for _, outs := range fromChain {
		for _, out := range outs.Outputs {
			total += out.Value
		}
	}
	balances, err := UTXOSet{f.bc}.Balances()
	if err != nil {
		t.Fatal(err)
	}
	indexed := 0
	for _, value := range balances {
		indexed += value
	}
	if total != indexed || total != 30 {
		t.Fatalf("chain scan = %d, index = %d, want 30", total, indexed)
	}

	for id, outs := range fromChain {
		raw, err := hex.DecodeString(id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.bc.FindTransaction(raw); err != nil {
			t.Fatalf("UTXO points to unknown transaction %s", id)
		}
		if len(outs.Outputs) == 0 {
			t.Fatal("UTXO entry without outputs")
		}
	}
}
