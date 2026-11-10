package transaction

import (
	"Blockchain/core"
	"Blockchain/gui/testkit"
	"errors"
	"strings"
	"testing"
)

func TestTransactionValidation(t *testing.T) {
	env := testkit.New(t, 2, true)
	tx := New(env.State)
	env.Show(tx.Content())
	snap := testkit.Sync(t, env.State)

	if tx.from.Selected != snap.Wallets[0].Address {
		t.Fatal("the first wallet must be preselected as sender")
	}
	tx.from.SetSelected(env.Wallets[0])

	cases := []struct {
		to, amount, fee string
		problem         string
	}{
		{"", "1", "0", "Enter a valid recipient address"},
		{"not-an-address", "1", "0", "Enter a valid recipient address"},
		{env.Wallets[1], "", "0", "Amount must be a positive integer"},
		{env.Wallets[1], "0", "0", "Amount must be a positive integer"},
		{env.Wallets[1], "2", "x", "Fee must be a non-negative integer"},
		{env.Wallets[1], "10", "1", "Not enough spendable funds"},
	}
	for _, c := range cases {
		tx.to.SetText(c.to)
		tx.amount.SetText(c.amount)
		tx.fee.SetText(c.fee)
		tx.validate()
		if !tx.send.Disabled() || !strings.Contains(tx.feedback.Text, c.problem) {
			t.Errorf("%+v: send enabled=%v feedback %q", c, !tx.send.Disabled(), tx.feedback.Text)
		}
	}

	tx.amount.SetText("-12a3456789012")
	if tx.amount.Text != "123456789" {
		t.Fatalf("amount entry kept %q, want digits only and at most nine", tx.amount.Text)
	}

	tx.to.SetText(env.Wallets[1])
	tx.amount.SetText("4")
	tx.fee.SetText("1")
	tx.validate()
	if tx.send.Disabled() {
		t.Fatalf("a valid transfer is blocked: %q", tx.feedback.Text)
	}

	tx.Focus(env.Wallets[1])
	if tx.from.Selected != env.Wallets[1] {
		t.Fatal("Focus must choose the sender")
	}
}

func TestExplain(t *testing.T) {
	cases := map[error]string{
		core.ErrPendingTransaction: "pending transaction",
		core.ErrMempoolFull:        "mempool is full",
		core.ErrOutputReserved:     "already being spent",
		errors.New("other"):        "other",
	}
	for err, want := range cases {
		if got := explain(err); !strings.Contains(got, want) {
			t.Errorf("explain(%v) = %q", err, got)
		}
	}
}
