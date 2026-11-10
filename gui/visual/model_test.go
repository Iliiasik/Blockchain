package visual

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"context"
	"encoding/hex"
	"strings"
	"testing"
)

func TestDemoModelIsConsistent(t *testing.T) {
	for _, mined := range []bool{false, true} {
		m := demoModel(mined)

		for h, b := range m.blocks {
			if b.height != h || !strings.HasPrefix(b.hash, "0000") {
				t.Fatalf("block %d: height %d hash %s", h, b.height, b.hash)
			}
			fees := 0
			for i, tx := range b.txs {
				if tx.coinbase != (i == 0) {
					t.Fatalf("block %d: coinbase must be the first transaction", h)
				}
				if tx.block != h {
					t.Fatalf("tx %s says block %d, sits in %d", tx.id[:6], tx.block, h)
				}
				fees += tx.fee
			}
			if reward := sumOutputs(b.txs[0]); reward != 10+fees {
				t.Fatalf("block %d coinbase pays %d, want subsidy 10 plus fees %d", h, reward, fees)
			}
		}

		for _, tx := range m.txs {
			if tx.coinbase {
				continue
			}
			in := 0
			for _, i := range tx.inputs {
				source, ok := m.txs[i.txid]
				if !ok || i.vout >= len(source.outputs) {
					t.Fatalf("tx %s spends an unknown output", tx.id[:6])
				}
				if source.outputs[i.vout].spentBy != tx.id {
					t.Fatalf("output spent by %s is not linked back", tx.id[:6])
				}
				if source.outputs[i.vout].owner != tx.sender() {
					t.Fatalf("tx %s spends coins of another wallet", tx.id[:6])
				}
				in += i.value
			}
			if in-sumOutputs(tx) != tx.fee {
				t.Fatalf("tx %s: inputs %d outputs %d fee %d", tx.id[:6], in, sumOutputs(tx), tx.fee)
			}
		}

		for _, w := range m.wallets {
			owned := 0
			for _, tx := range m.txs {
				if tx.pending() {
					continue
				}
				for _, out := range tx.outputs {
					spender, spent := m.txs[out.spentBy]
					if out.owner == w.address && (!spent || spender.pending()) {
						owned += out.value
					}
				}
			}
			if owned != w.balance {
				t.Fatalf("mined=%v wallet %s shows %d but owns %d", mined, w.name, w.balance, owned)
			}
		}
	}
}

func sumOutputs(t *txn) int {
	total := 0
	for _, out := range t.outputs {
		total += out.value
	}
	return total
}

func TestTransferExcludesChange(t *testing.T) {
	m := demoModel(false)
	amount, to := m.txs[demoAlice].transfer()
	if amount != 4 || to != "bob" {
		t.Fatalf("transfer = %d to %s, want 4 to bob", amount, to)
	}
	amount, to = m.txs[demoCoinbase[1]].transfer()
	if amount != 10 || to != "miner" {
		t.Fatalf("coinbase = %d to %s", amount, to)
	}

	self := &txn{inputs: []input{{owner: "a"}}, outputs: []output{{value: 3, owner: "a"}}}
	if amount, to := self.transfer(); amount != 3 || to != "a" {
		t.Fatalf("self transfer = %d to %s", amount, to)
	}
}

func TestRelated(t *testing.T) {
	m := demoModel(false)

	block := m.related(blockRef(1))
	for _, r := range []ref{blockRef(0), blockRef(2), txRef(demoCoinbase[1]), txRef(demoAlice), walletRef("bob"), txRef(demoBob), txRef(demoPending)} {
		if !block[r] {
			t.Errorf("block #1 is not related to %+v", r)
		}
	}

	tx := m.related(txRef(demoBob))
	for _, r := range []ref{blockRef(2), txRef(demoAlice), walletRef("bob"), walletRef("alice")} {
		if !tx[r] {
			t.Errorf("Bob's transfer is not related to %+v", r)
		}
	}
	if tx[blockRef(0)] {
		t.Error("Bob's transfer must not pull in unrelated blocks")
	}

	wallet := m.related(walletRef("alice"))
	for _, r := range []ref{txRef(demoCoinbase[0]), txRef(demoAlice), txRef(demoBob), txRef(demoPending)} {
		if !wallet[r] {
			t.Errorf("Alice is not related to %+v", r)
		}
	}

	if m.related(ref{}) != nil {
		t.Error("an empty selection must relate to nothing")
	}
}

func TestExplainCoversEveryKind(t *testing.T) {
	m := demoModel(false)
	refs := []ref{blockRef(0), blockRef(2), txRef(demoCoinbase[1]), txRef(demoAlice), txRef(demoPending), walletRef("alice"), walletRef("bob")}
	for _, r := range refs {
		title, text := m.explain(r)
		if title == "" || len(text) == 0 {
			t.Errorf("no explanation for %+v", r)
		}
	}

	_, alice := m.explain(walletRef("alice"))
	if !strings.Contains(strings.Join(alice, " "), "Another 6 coins") {
		t.Errorf("Alice's reserved change is not explained: %v", alice)
	}
	_, pending := m.explain(txRef(demoPending))
	if !strings.Contains(strings.Join(pending, " "), "6 - 5 = 1") {
		t.Errorf("fee arithmetic missing: %v", pending)
	}

	if title, text := m.explain(ref{}); title != "" || text != nil {
		t.Error("empty selection must have no explanation")
	}
	if title, _ := m.explain(blockRef(99)); title != "" {
		t.Error("unknown block explained")
	}
	if title, _ := m.explain(txRef("missing")); title != "" {
		t.Error("unknown transaction explained")
	}
}

func TestFromSnapshot(t *testing.T) {
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
	transfer, err := core.NewUTXOTransaction(alice, bob, 3, 1, set)
	if err != nil {
		t.Fatal(err)
	}
	block, err := bc.MineBlock(context.Background(), []*core.Transaction{core.NewCoinbaseTX(bob, "", 11), transfer}, 8, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := set.Update(block); err != nil {
		t.Fatal(err)
	}
	pending, err := core.NewUTXOTransaction(bob, alice, 2, 0, set)
	if err != nil {
		t.Fatal(err)
	}

	snap := &state.Snapshot{
		HasChain:   true,
		Blocks:     bc.Blocks(),
		Mempool:    bc.Mempool.GetTransactions(),
		MempoolMax: 10,
		Wallets:    []state.WalletInfo{{Address: alice, Balance: 6}, {Address: bob, Balance: 14}},
	}
	m := fromSnapshot(snap)

	if len(m.blocks) != 2 || m.blocks[0].height != 0 || len(m.blocks[1].txs) != 2 {
		t.Fatal("blocks must be ordered from genesis to tip")
	}
	if len(m.mempool) != 1 || !m.mempool[0].pending() || m.maxPool != 10 {
		t.Fatal("mempool not converted")
	}
	mined := m.txs[m.blocks[1].txs[1].id]
	if mined.sender() != alice || mined.inputs[0].value != 10 || mined.fee != 1 {
		t.Fatalf("transfer converted wrong: %+v", mined)
	}
	if mined.outputs[0].owner != bob || mined.outputs[1].owner != alice {
		t.Fatal("output owners were not decoded from key hashes")
	}
	genesis := m.blocks[0].txs[0]
	if genesis.outputs[0].spentBy != mined.id {
		t.Fatal("genesis output is not linked to the transfer that spent it")
	}
	queued := m.mempool[0]
	if queued.id != hex.EncodeToString(pending.ID) {
		t.Fatal("pending transaction ID changed")
	}
	if source := m.txs[queued.inputs[0].txid]; source == nil || source.outputs[queued.inputs[0].vout].spentBy != queued.id {
		t.Fatal("the pending transfer does not reserve the output it spends")
	}
	if fromSnapshot(nil) == nil || len(fromSnapshot(nil).blocks) != 0 {
		t.Fatal("nil snapshot must give an empty model")
	}
}
