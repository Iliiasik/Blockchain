package core

import (
	"context"
	"testing"
)

const testBits = 8

type fixture struct {
	bc    *Blockchain
	alice string
	bob   string
	miner string
}

func useTempDir(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
}

func newWallets(t *testing.T, count int) []string {
	t.Helper()
	wallets, err := NewWallets()
	if err != nil {
		t.Fatalf("NewWallets: %v", err)
	}
	addresses := make([]string, count)
	for i := range addresses {
		addresses[i] = wallets.CreateWallet()
	}
	if err := wallets.SaveToFile(); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	return addresses
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	useTempDir(t)
	addresses := newWallets(t, 3)

	bc, err := CreateBlockchain(addresses[0], 10, testBits)
	if err != nil {
		t.Fatalf("CreateBlockchain: %v", err)
	}
	t.Cleanup(func() {
		if err := bc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	if err := (UTXOSet{bc}).Reindex(); err != nil {
		t.Fatalf("Reindex: %v", err)
	}

	return fixture{bc: bc, alice: addresses[0], bob: addresses[1], miner: addresses[2]}
}

func (f fixture) mine(t *testing.T, txs ...*Transaction) *Block {
	t.Helper()
	fees := 0
	for _, tx := range txs {
		fees += tx.Fee
	}
	coinbase := NewCoinbaseTX(f.miner, "", 10+fees)
	block, err := f.bc.MineBlock(context.Background(), append([]*Transaction{coinbase}, txs...), testBits, nil)
	if err != nil {
		t.Fatalf("MineBlock: %v", err)
	}
	if err := (UTXOSet{f.bc}).Update(block); err != nil {
		t.Fatalf("Update: %v", err)
	}
	return block
}

func (f fixture) send(t *testing.T, from, to string, amount, fee int) *Transaction {
	t.Helper()
	tx, err := NewUTXOTransaction(from, to, amount, fee, &UTXOSet{f.bc})
	if err != nil {
		t.Fatalf("NewUTXOTransaction: %v", err)
	}
	return tx
}

func (f fixture) balance(t *testing.T, address string) int {
	t.Helper()
	outputs, err := UTXOSet{f.bc}.FindUTXO(AddressToPubKeyHash(address))
	if err != nil {
		t.Fatalf("FindUTXO: %v", err)
	}
	total := 0
	for _, out := range outputs {
		total += out.Value
	}
	return total
}
