package utxo

import (
	"Blockchain/core"
	"Blockchain/gui/testkit"
	"testing"
)

func TestUTXOScreenWithoutChain(t *testing.T) {
	env := testkit.New(t, 1, false)
	u := New(env.State)
	env.Show(u.Content())
	testkit.Sync(t, env.State)

	if len(u.rows.Objects) != 0 {
		t.Fatal("no outputs without a chain")
	}
	if u.entries.Value.Text != "0" {
		t.Fatalf("entries = %s", u.entries.Value.Text)
	}
}

func TestUTXORender(t *testing.T) {
	env := testkit.New(t, 1, false)
	u := New(env.State)
	env.Show(u.Content())

	u.outputs = []core.UnspentOutput{
		{TxID: []byte{1}, Index: 0, TXOutput: core.TXOutput{Value: 7}},
		{TxID: []byte{2}, Index: 1, TXOutput: core.TXOutput{Value: 3}, Pending: true},
	}
	u.render(5)
	if u.entries.Value.Text != "5" || u.shown.Value.Text != "2" || u.value.Value.Text != "10" || u.pending.Value.Text != "3" {
		t.Fatalf("stats = %s %s %s %s", u.entries.Value.Text, u.shown.Value.Text, u.value.Value.Text, u.pending.Value.Text)
	}
	if len(u.rows.Objects) != 2 {
		t.Fatal("list does not show the outputs")
	}

	u.outputs = make([]core.UnspentOutput, maxRows+5)
	u.render(0)
	if len(u.rows.Objects) != maxRows+1 {
		t.Fatalf("long lists must be capped, got %d rows", len(u.rows.Objects))
	}
}
