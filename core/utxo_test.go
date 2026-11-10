package core

import (
	"encoding/hex"
	"errors"
	"testing"
)

func TestTransferFlow(t *testing.T) {
	f := newFixture(t)

	if got := f.balance(t, f.alice); got != 10 {
		t.Fatalf("genesis balance = %d, want 10", got)
	}

	tx := f.send(t, f.alice, f.bob, 4, 1)
	if len(tx.Vout) != 2 || tx.Vout[0].Value != 4 || tx.Vout[1].Value != 5 {
		t.Fatalf("outputs = %+v, want 4 to bob and 5 change", tx.Vout)
	}
	if f.bc.Mempool.Size() != 1 {
		t.Fatal("transaction did not reach the mempool")
	}

	pending, err := UTXOSet{f.bc}.FindUTXO(AddressToPubKeyHash(f.alice))
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || !pending[0].Pending {
		t.Fatal("the spent genesis output must be flagged as pending")
	}
	if got := f.balance(t, f.bob); got != 0 {
		t.Fatalf("bob has %d before mining", got)
	}

	f.mine(t, tx)

	if f.bc.Mempool.Size() != 0 {
		t.Fatal("mined transaction stayed in the mempool")
	}
	want := map[string]int{f.alice: 5, f.bob: 4, f.miner: 11}
	for address, value := range want {
		if got := f.balance(t, address); got != value {
			t.Errorf("balance of %s = %d, want %d", address[:6], got, value)
		}
	}

	f.mine(t, f.send(t, f.bob, f.alice, 4, 0))
	if f.balance(t, f.bob) != 0 || f.balance(t, f.alice) != 9 {
		t.Fatal("spending the whole balance must leave no change output")
	}
}

func TestNewUTXOTransactionErrors(t *testing.T) {
	f := newFixture(t)
	set := &UTXOSet{f.bc}
	stranger := string(NewWallet().GetAddress())

	cases := []struct {
		name             string
		from, to         string
		amount, fee      int
		want             error
		wantAnyErrorOnly bool
	}{
		{name: "zero amount", from: f.alice, to: f.bob, amount: 0, wantAnyErrorOnly: true},
		{name: "negative fee", from: f.alice, to: f.bob, amount: 1, fee: -1, wantAnyErrorOnly: true},
		{name: "bad recipient", from: f.alice, to: "nope", amount: 1, wantAnyErrorOnly: true},
		{name: "foreign sender", from: stranger, to: f.bob, amount: 1, want: ErrUnknownWallet},
		{name: "not enough", from: f.alice, to: f.bob, amount: 10, fee: 1, want: ErrNotEnoughFunds},
		{name: "empty wallet", from: f.bob, to: f.alice, amount: 1, want: ErrNotEnoughFunds},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewUTXOTransaction(c.from, c.to, c.amount, c.fee, set)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !c.wantAnyErrorOnly && !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}
	if f.bc.Mempool.Size() != 0 {
		t.Fatal("rejected transactions reached the mempool")
	}

	f.send(t, f.alice, f.bob, 1, 0)
	if _, err := NewUTXOTransaction(f.alice, f.bob, 1, 0, set); !errors.Is(err, ErrPendingTransaction) {
		t.Fatalf("second pending transfer err = %v", err)
	}

	f.bc.Mempool.SetMaxSize(1)
	f.mine(t)
	if _, err := NewUTXOTransaction(f.miner, f.bob, 1, 0, set); !errors.Is(err, ErrMempoolFull) {
		t.Fatalf("full mempool err = %v", err)
	}
}

func TestUpdateMatchesReindex(t *testing.T) {
	f := newFixture(t)
	f.mine(t, f.send(t, f.alice, f.bob, 3, 1))
	f.mine(t, f.send(t, f.bob, f.miner, 2, 0), f.send(t, f.alice, f.miner, 1, 1))

	set := UTXOSet{f.bc}
	incremental := snapshotUTXO(t, set)
	if err := set.Reindex(); err != nil {
		t.Fatal(err)
	}
	rebuilt := snapshotUTXO(t, set)

	if len(incremental) != len(rebuilt) {
		t.Fatalf("incremental %d outputs, rebuilt %d", len(incremental), len(rebuilt))
	}
	for key, value := range rebuilt {
		if incremental[key] != value {
			t.Fatalf("output %s: incremental %d, rebuilt %d", key, incremental[key], value)
		}
	}

	count, err := set.CountTransactions()
	if err != nil || count == 0 {
		t.Fatalf("CountTransactions = %d, %v", count, err)
	}
}

func snapshotUTXO(t *testing.T, set UTXOSet) map[string]int {
	t.Helper()
	outputs, err := set.FindUTXO(nil)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]int{}
	for _, out := range outputs {
		result[hex.EncodeToString(out.TxID)+":"+string(rune('0'+out.Index))] = out.Value
	}
	return result
}

func TestFindSpendableOutputsSkipsReserved(t *testing.T) {
	f := newFixture(t)
	f.mine(t)
	set := UTXOSet{f.bc}
	minerHash := AddressToPubKeyHash(f.miner)

	total, outputs, err := set.FindSpendableOutputs(minerHash, 5)
	if err != nil || total != 10 || len(outputs) != 1 {
		t.Fatalf("spendable = %d %v %v", total, outputs, err)
	}

	f.send(t, f.miner, f.bob, 2, 0)
	total, _, err = set.FindSpendableOutputs(minerHash, 5)
	if err != nil || total != 0 {
		t.Fatalf("reserved output still spendable: %d %v", total, err)
	}
}

func TestPruneMempoolDropsStaleTransactions(t *testing.T) {
	f := newFixture(t)
	tx := f.send(t, f.alice, f.bob, 2, 0)

	set := UTXOSet{f.bc}
	removed, err := set.PruneMempool()
	if err != nil || removed != 0 {
		t.Fatalf("valid transaction pruned: %d %v", removed, err)
	}

	f.mine(t, tx)
	if err := f.bc.Mempool.AddTransaction(tx); err != nil {
		t.Fatal(err)
	}
	removed, err = set.PruneMempool()
	if err != nil || removed != 1 || f.bc.Mempool.Size() != 0 {
		t.Fatalf("already mined transaction kept: %d %v", removed, err)
	}

	fresh := f.send(t, f.alice, f.bob, 1, 0)
	fresh.Vout[0].Value = 7
	if err := f.bc.Mempool.RemoveTransactions(hex.EncodeToString(fresh.ID)); err != nil {
		t.Fatal(err)
	}
	if err := f.bc.Mempool.AddTransaction(fresh); err != nil {
		t.Fatal(err)
	}
	removed, err = set.PruneMempool()
	if err != nil || removed != 1 {
		t.Fatalf("transaction with a broken signature kept: %d %v", removed, err)
	}
}
