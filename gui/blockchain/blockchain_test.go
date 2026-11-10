package blockchain

import (
	"Blockchain/gui/testkit"
	"testing"
)

func TestBlockchainScreenBeforeGenesis(t *testing.T) {
	env := testkit.New(t, 2, false)
	b := New(env.State, env.Window)
	env.Show(b.Content())
	snap := testkit.Sync(t, env.State)

	if b.setup.Visible() == false || b.explorer.Visible() {
		t.Fatal("setup must be shown before the genesis block exists")
	}
	if len(b.genesis.Options) != 2 || b.genesis.Selected != snap.Wallets[0].Address {
		t.Fatalf("genesis options %v, selected %q", b.genesis.Options, b.genesis.Selected)
	}
	b.Focus(snap.Wallets[1].Address)
	if b.genesis.Selected != snap.Wallets[1].Address {
		t.Fatal("Focus must preselect the genesis wallet")
	}
	if b.create.Disabled() {
		t.Fatal("creating the chain must be possible with a wallet selected")
	}
}

func TestBlockchainScreenExplorer(t *testing.T) {
	env := testkit.New(t, 1, true)
	b := New(env.State, env.Window)
	env.Show(b.Content())
	testkit.Sync(t, env.State)

	if !b.explorer.Visible() || b.setup.Visible() {
		t.Fatal("explorer must be shown once the chain exists")
	}
	if b.height.Value.Text != "1" || b.difficulty.Value.Text != "8 bits" || b.txs.Value.Text != "1" {
		t.Fatalf("stats = %s, %s, %s", b.height.Value.Text, b.difficulty.Value.Text, b.txs.Value.Text)
	}
	if len(b.cards) != 1 {
		t.Fatalf("strip shows %d cards", len(b.cards))
	}
}
