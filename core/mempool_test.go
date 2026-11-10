package core

import (
	"encoding/hex"
	"errors"
	"os"
	"testing"
)

func pendingTx(fee int, spends ...TXInput) *Transaction {
	tx := &Transaction{Vin: spends, Vout: []TXOutput{*NewTXOutput(1, string(NewWallet().GetAddress()))}, Fee: fee}
	tx.ID = tx.Hash()
	return tx
}

func input(seed byte, vout int) TXInput {
	return TXInput{Txid: []byte{seed, seed, seed}, Vout: vout}
}

func TestMempoolAddGetRemove(t *testing.T) {
	useTempDir(t)
	mp := NewMempool(3)
	tx := pendingTx(1, input(1, 0))

	if err := mp.AddTransaction(tx); err != nil {
		t.Fatal(err)
	}
	id := hex.EncodeToString(tx.ID)
	if got, ok := mp.GetTransaction(id); !ok || got != tx {
		t.Fatal("added transaction not found")
	}
	if !mp.IsOutputSpent(hex.EncodeToString([]byte{1, 1, 1}), 0) {
		t.Fatal("spent output not reserved")
	}
	if mp.IsOutputSpent(hex.EncodeToString([]byte{1, 1, 1}), 1) {
		t.Fatal("unrelated output reserved")
	}

	if err := mp.RemoveTransactions(id, "missing"); err != nil {
		t.Fatal(err)
	}
	if _, ok := mp.GetTransaction(id); ok || mp.Size() != 0 {
		t.Fatal("transaction still present after removal")
	}
	if mp.IsOutputSpent(hex.EncodeToString([]byte{1, 1, 1}), 0) {
		t.Fatal("reservation kept after removal")
	}
	if err := mp.RemoveTransactions("missing"); err != nil {
		t.Fatal(err)
	}
}

func TestMempoolRejectsDoubleSpendAndOverflow(t *testing.T) {
	useTempDir(t)
	mp := NewMempool(2)

	first := pendingTx(1, input(1, 0))
	if err := mp.AddTransaction(first); err != nil {
		t.Fatal(err)
	}
	if err := mp.AddTransaction(first); err != nil {
		t.Fatalf("adding the same transaction twice must be harmless: %v", err)
	}
	if err := mp.AddTransaction(pendingTx(2, input(1, 0))); !errors.Is(err, ErrOutputReserved) {
		t.Fatalf("double spend err = %v", err)
	}
	if err := mp.AddTransaction(pendingTx(2, input(2, 0))); err != nil {
		t.Fatal(err)
	}
	if !mp.IsFull() {
		t.Fatal("mempool should be full")
	}
	if err := mp.AddTransaction(pendingTx(3, input(3, 0))); !errors.Is(err, ErrMempoolFull) {
		t.Fatalf("overflow err = %v", err)
	}

	mp.SetMaxSize(3)
	if mp.MaxSize() != 3 || mp.IsFull() {
		t.Fatal("SetMaxSize did not apply")
	}
}

func TestMempoolOrdersByFee(t *testing.T) {
	useTempDir(t)
	mp := NewMempool(10)
	for i, fee := range []int{1, 5, 3, 5} {
		if err := mp.AddTransaction(pendingTx(fee, input(byte(i+1), 0))); err != nil {
			t.Fatal(err)
		}
	}

	txs := mp.GetTransactions()
	for i := 1; i < len(txs); i++ {
		if txs[i-1].Fee < txs[i].Fee {
			t.Fatalf("fees out of order: %d before %d", txs[i-1].Fee, txs[i].Fee)
		}
		if txs[i-1].Fee == txs[i].Fee && hex.EncodeToString(txs[i-1].ID) > hex.EncodeToString(txs[i].ID) {
			t.Fatal("equal fees must be ordered by ID")
		}
	}
}

func TestMempoolPersistence(t *testing.T) {
	useTempDir(t)
	mp := NewMempool(5)
	a, b := pendingTx(1, input(1, 0)), pendingTx(2, input(2, 1))
	for _, tx := range []*Transaction{a, b} {
		if err := mp.AddTransaction(tx); err != nil {
			t.Fatal(err)
		}
	}

	reloaded := NewMempool(5)
	if reloaded.Size() != 2 {
		t.Fatalf("reloaded size = %d", reloaded.Size())
	}
	if !reloaded.IsOutputSpent(hex.EncodeToString([]byte{2, 2, 2}), 1) {
		t.Fatal("reservations were not rebuilt from the file")
	}

	if err := reloaded.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mempoolFile); !os.IsNotExist(err) {
		t.Fatal("an empty mempool must remove its file")
	}
	if NewMempool(5).Size() != 0 {
		t.Fatal("cleared mempool came back")
	}
}

func TestMempoolSkipsCorruptLines(t *testing.T) {
	useTempDir(t)
	good := pendingTx(1, input(1, 0))
	content := "zz-not-hex\n" + hex.EncodeToString(good.Serialize()) + "\n\n"
	if err := os.WriteFile(mempoolFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if size := NewMempool(5).Size(); size != 1 {
		t.Fatalf("size = %d, want 1", size)
	}
}
