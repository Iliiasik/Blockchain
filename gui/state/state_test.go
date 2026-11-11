package state

import (
	"Blockchain/core"
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
)

func newState(t *testing.T) *AppState {
	t.Helper()
	test.NewTempApp(t)
	t.Chdir(t.TempDir())

	s := NewAppState()
	s.TargetBits = 8
	t.Cleanup(func() {
		waitIdle(t, s)
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return s
}

func waitIdle(t *testing.T, s *AppState) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for s.Busy() {
		if time.Now().After(deadline) {
			t.Fatal("background work did not finish")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func mine(t *testing.T, s *AppState, miner string, txs []*core.Transaction) *core.Block {
	t.Helper()
	type result struct {
		block *core.Block
		err   error
	}
	done := make(chan result, 1)
	s.Mine(miner, txs, func(b *core.Block, err error) { done <- result{b, err} })
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("Mine: %v", r.err)
		}
		waitIdle(t, s)
		return r.block
	case <-time.After(10 * time.Second):
		t.Fatal("mining timed out")
	}
	return nil
}

func TestSnapshotHelpers(t *testing.T) {
	empty := &Snapshot{}
	if empty.Height() != 0 || empty.Tip() != nil || len(empty.Addresses()) != 0 {
		t.Fatal("empty snapshot helpers must return zero values")
	}

	snap := &Snapshot{
		Blocks:  []*core.Block{{Nonce: 2}, {Nonce: 1}},
		Wallets: []WalletInfo{{Address: "a", Balance: 3}, {Address: "b"}},
	}
	if tip := snap.Tip(); snap.Height() != 2 || tip == nil || tip.Nonce != 2 {
		t.Fatal("tip must be the first block")
	}
	if w, ok := snap.Wallet("a"); !ok || w.Balance != 3 {
		t.Fatal("wallet lookup failed")
	}
	if _, ok := snap.Wallet("c"); ok {
		t.Fatal("unknown wallet found")
	}
	if got := snap.Addresses(); len(got) != 2 || got[1] != "b" {
		t.Fatalf("addresses = %v", got)
	}
}

func TestFullLifecycle(t *testing.T) {
	s := newState(t)

	if err := s.Open(); err != nil || s.Blockchain() != nil {
		t.Fatalf("Open without a database: %v", err)
	}
	if _, err := s.Send("a", "b", 1, 0); !errors.Is(err, core.ErrNoBlockchain) {
		t.Fatalf("Send without chain err = %v", err)
	}
	if _, err := s.Reindex(); !errors.Is(err, core.ErrNoBlockchain) {
		t.Fatalf("Reindex without chain err = %v", err)
	}

	alice, err := s.CreateWallet()
	if err != nil {
		t.Fatal(err)
	}
	bob, err := s.CreateWallet()
	if err != nil {
		t.Fatal(err)
	}

	snap := s.load()
	if snap.HasChain || len(snap.Wallets) != 2 || snap.Err != nil {
		t.Fatalf("before genesis: %+v", snap)
	}

	if err := s.CreateBlockchain(alice); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBlockchain(alice); err == nil {
		t.Fatal("second genesis accepted")
	}

	snap = s.load()
	if !snap.HasChain || snap.Height() != 1 || snap.Supply != 10 || snap.TxCount != 1 || snap.UTXOCount != 1 {
		t.Fatalf("after genesis: height %d supply %d txs %d utxo %d", snap.Height(), snap.Supply, snap.TxCount, snap.UTXOCount)
	}
	if w, _ := snap.Wallet(alice); w.Balance != 10 || w.Available != 10 || w.Outputs != 1 {
		t.Fatalf("alice after genesis: %+v", w)
	}

	tx, err := s.Send(alice, bob, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	snap = s.load()
	if len(snap.Mempool) != 1 {
		t.Fatal("transfer did not reach the mempool")
	}
	if w, _ := snap.Wallet(alice); w.Balance != 10 || w.Available != 0 {
		t.Fatalf("pending coins must be locked: %+v", w)
	}

	var events []bool
	s.OnMiningChanged(func(active bool) { events = append(events, active) })
	block := mine(t, s, bob, snap.Mempool)
	if len(block.Transactions) != 2 || string(block.Transactions[1].ID) != string(tx.ID) {
		t.Fatal("mined block does not contain the transfer")
	}
	if len(events) != 2 || !events[0] || events[1] {
		t.Fatalf("mining events = %v, want [true false]", events)
	}

	snap = s.load()
	if snap.Height() != 2 || len(snap.Mempool) != 0 || snap.Supply != 20 {
		t.Fatalf("after mining: height %d mempool %d supply %d", snap.Height(), len(snap.Mempool), snap.Supply)
	}
	balances := map[string]int{}
	for _, w := range snap.Wallets {
		balances[w.Address] = w.Balance
	}
	if balances[alice] != 5 || balances[bob] != 4+11 {
		t.Fatalf("balances = %v", balances)
	}

	count, err := s.Reindex()
	if err != nil || count != snap.UTXOCount {
		t.Fatalf("Reindex = %d, %v, want %d", count, err, snap.UTXOCount)
	}

	s.SetMempoolSize(3)
	if s.Blockchain().Mempool.MaxSize() != 3 {
		t.Fatal("SetMempoolSize did not reach the mempool")
	}

	s.Subsidy = 42
	s.TargetBits = 12
	if err := s.ResetBlockchain(); err != nil {
		t.Fatal(err)
	}
	if s.Blockchain() != nil || core.BlockchainExists() {
		t.Fatal("reset left the chain behind")
	}
	if s.Subsidy != defaultSubsidy || s.TargetBits != defaultTargetBits {
		t.Fatalf("reset kept subsidy %d and bits %d", s.Subsidy, s.TargetBits)
	}
	if snap = s.load(); snap.HasChain || len(snap.Wallets) != 2 {
		t.Fatal("reset must keep wallets and drop the chain")
	}
}

func TestReopenExistingChain(t *testing.T) {
	s := newState(t)
	alice, err := s.CreateWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBlockchain(alice); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("closing twice must be harmless: %v", err)
	}

	if err := s.Open(); err != nil || s.Blockchain() == nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Open(); err != nil {
		t.Fatalf("opening twice must be harmless: %v", err)
	}
	if s.load().Height() != 1 {
		t.Fatal("reopened chain lost its genesis block")
	}
}

func TestMiningCanBeCancelled(t *testing.T) {
	s := newState(t)
	alice, err := s.CreateWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBlockchain(alice); err != nil {
		t.Fatal(err)
	}

	s.TargetBits = 60
	done := make(chan error, 1)
	s.Mine(alice, nil, func(_ *core.Block, err error) { done <- err })
	if !s.IsMining() {
		t.Fatal("mining did not start")
	}

	second := make(chan error, 1)
	s.Mine(alice, nil, func(_ *core.Block, err error) { second <- err })
	if err := <-second; err == nil {
		t.Fatal("a second miner started")
	}
	if err := s.ResetBlockchain(); err == nil {
		t.Fatal("reset allowed while mining")
	}

	s.CancelMining()
	select {
	case err := <-done:
		if !errors.Is(err, core.ErrMiningCancelled) {
			t.Fatalf("err = %v, want ErrMiningCancelled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("cancel had no effect")
	}
	waitIdle(t, s)
	if s.load().Height() != 1 {
		t.Fatal("cancelled mining changed the chain")
	}
}

func TestMineWithoutChain(t *testing.T) {
	s := newState(t)
	done := make(chan error, 1)
	s.Mine("x", nil, func(_ *core.Block, err error) { done <- err })
	if err := <-done; !errors.Is(err, core.ErrNoBlockchain) {
		t.Fatalf("err = %v", err)
	}
}

func TestSubscribeAndRefresh(t *testing.T) {
	s := newState(t)
	updates := make(chan *Snapshot, 8)
	s.Subscribe(func(snap *Snapshot) { updates <- snap })
	if first := <-updates; first != s.Snapshot() {
		t.Fatal("Subscribe must deliver the current snapshot right away")
	}

	if _, err := s.CreateWallet(); err != nil {
		t.Fatal(err)
	}
	s.Refresh()
	s.Refresh()

	select {
	case snap := <-updates:
		if len(snap.Wallets) != 1 {
			t.Fatalf("refreshed snapshot has %d wallets", len(snap.Wallets))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Refresh did not notify subscribers")
	}
	waitIdle(t, s)
	if len(s.Snapshot().Wallets) != 1 {
		t.Fatal("Snapshot was not replaced")
	}
}

func TestNavigationAndToastHooks(t *testing.T) {
	s := NewAppState()
	s.Go("wallets", "x")
	s.Toast("title", "message", LevelInfo)

	var screen, arg, title string
	var level Level
	s.Navigate = func(sc, a string) { screen, arg = sc, a }
	s.Notify = func(ti, _ string, l Level) { title, level = ti, l }

	s.Go("mining", "addr")
	s.Toast("Done", "", LevelSuccess)
	if screen != "mining" || arg != "addr" || title != "Done" || level != LevelSuccess {
		t.Fatalf("hooks got %q %q %q %v", screen, arg, title, level)
	}
}
