package wallet

import (
	"Blockchain/core"
	"Blockchain/gui/testkit"
	"context"
	"testing"
)

func TestHistory(t *testing.T) {
	t.Chdir(t.TempDir())
	wallets, err := core.NewWallets()
	if err != nil {
		t.Fatal(err)
	}
	alice, bob := wallets.CreateWallet(), wallets.CreateWallet()
	if err := wallets.SaveToFile(); err != nil {
		t.Fatal(err)
	}
	bc, err := core.CreateBlockchain(alice, 10, 8)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := bc.Close(); err != nil {
			t.Error(err)
		}
	}()
	set := &core.UTXOSet{Blockchain: bc}
	if err := set.Reindex(); err != nil {
		t.Fatal(err)
	}

	sent, err := core.NewUTXOTransaction(alice, bob, 4, 1, set)
	if err != nil {
		t.Fatal(err)
	}
	block, err := bc.MineBlock(context.Background(), []*core.Transaction{core.NewCoinbaseTX(bob, "", 11), sent}, 8, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := set.Update(block); err != nil {
		t.Fatal(err)
	}
	pending, err := core.NewUTXOTransaction(alice, bob, 2, 0, set)
	if err != nil {
		t.Fatal(err)
	}

	aliceHistory := history(bc.Blocks(), []*core.Transaction{pending}, core.AddressToPubKeyHash(alice))
	if len(aliceHistory) != 3 {
		t.Fatalf("alice has %d entries, want pending, sent and mined", len(aliceHistory))
	}
	if e := aliceHistory[0]; !e.Pending || e.Kind != "sent" || e.Amount != -2 {
		t.Fatalf("pending entry = %+v", e)
	}
	if e := aliceHistory[1]; e.Kind != "sent" || e.Amount != -5 || e.Fee != 1 || e.Counterparty != bob || e.Height != 1 {
		t.Fatalf("sent entry = %+v", e)
	}
	if e := aliceHistory[2]; e.Kind != "mined" || e.Amount != 10 || e.Height != 0 {
		t.Fatalf("genesis entry = %+v", e)
	}

	bobHistory := history(bc.Blocks(), nil, core.AddressToPubKeyHash(bob))
	kinds := map[string]int{}
	for _, e := range bobHistory {
		kinds[e.Kind] += e.Amount
	}
	if kinds["mined"] != 11 || kinds["received"] != 4 {
		t.Fatalf("bob history = %+v", bobHistory)
	}

	if len(history(bc.Blocks(), nil, core.AddressToPubKeyHash(string(core.NewWallet().GetAddress())))) != 0 {
		t.Fatal("a stranger has no history")
	}
}

func TestWalletScreen(t *testing.T) {
	env := testkit.New(t, 2, true)
	w := New(env.State, env.Window)
	env.Show(w.Content())

	testkit.Sync(t, env.State)
	if w.list.Length() != 2 {
		t.Fatalf("list shows %d wallets", w.list.Length())
	}
	if w.count.Text != "2 wallets in wallet.dat" {
		t.Fatalf("count = %q", w.count.Text)
	}
	if len(w.detail.Objects) != 1 {
		t.Fatal("the empty state must be shown before a wallet is picked")
	}

	for _, kind := range []string{"mined", "sent", "received"} {
		row := w.entryRow(entry{Kind: kind, Amount: 3, Fee: 1, Counterparty: env.Wallets[1], TxID: []byte{1, 2, 3}})
		if row.MinSize().Height <= 0 {
			t.Fatalf("%s row is empty", kind)
		}
	}
	if w.entryRow(entry{Kind: "sent", Pending: true, TxID: []byte{1}}) == nil {
		t.Fatal("pending row missing")
	}
}
