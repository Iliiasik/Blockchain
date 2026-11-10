package core

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

func signedTransfer(t *testing.T) (*Transaction, map[string]Transaction, *Wallet, *Wallet) {
	t.Helper()
	alice, bob := NewWallet(), NewWallet()

	funding := NewCoinbaseTX(string(alice.GetAddress()), "funding", 10)
	tx := &Transaction{
		Vin:  []TXInput{{Txid: funding.ID, Vout: 0, PubKey: alice.PublicKey}},
		Vout: []TXOutput{*NewTXOutput(7, string(bob.GetAddress())), *NewTXOutput(2, string(alice.GetAddress()))},
		Fee:  1,
	}
	tx.ID = tx.Hash()

	prev := map[string]Transaction{hex.EncodeToString(funding.ID): *funding}
	if err := tx.Sign(alice.PrivateKey, prev); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return tx, prev, alice, bob
}

func TestCoinbaseTransaction(t *testing.T) {
	address := string(NewWallet().GetAddress())
	tx := NewCoinbaseTX(address, "", 12)

	if !tx.IsCoinbase() {
		t.Fatal("coinbase not detected")
	}
	if tx.GetSenderAddress() != "" {
		t.Fatal("coinbase must have no sender")
	}
	if tx.GetRecipientAddress() != address || tx.GetAmount() != 12 {
		t.Fatalf("recipient %s amount %d", tx.GetRecipientAddress(), tx.GetAmount())
	}
	if len(tx.Vin[0].PubKey) == 0 {
		t.Fatal("an empty data field must be filled with random bytes")
	}
	if !bytes.Equal(tx.ID, tx.Hash()) {
		t.Fatal("ID must be the hash of the transaction")
	}

	other := NewCoinbaseTX(address, "", 12)
	if bytes.Equal(tx.ID, other.ID) {
		t.Fatal("two coinbases to the same address must still differ")
	}
	if !tx.Verify(nil) {
		t.Fatal("coinbase always verifies")
	}
}

func TestTransactionSerializationRoundTrip(t *testing.T) {
	tx, _, _, _ := signedTransfer(t)
	decoded := DeserializeTransaction(tx.Serialize())

	if !bytes.Equal(decoded.Serialize(), tx.Serialize()) {
		t.Fatal("round trip changed the transaction")
	}
	if decoded.Fee != tx.Fee || len(decoded.Vin) != 1 || len(decoded.Vout) != 2 {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestHashIgnoresID(t *testing.T) {
	tx, _, _, _ := signedTransfer(t)
	before := tx.Hash()
	tx.ID = []byte("anything")
	if !bytes.Equal(before, tx.Hash()) {
		t.Fatal("hash must not depend on the ID field")
	}
}

func TestSignAndVerify(t *testing.T) {
	tx, prev, _, _ := signedTransfer(t)

	if len(tx.Vin[0].Signature) != 2*coordinateSize {
		t.Fatalf("signature length = %d", len(tx.Vin[0].Signature))
	}
	if !tx.Verify(prev) {
		t.Fatal("valid signature rejected")
	}
}

func TestVerifyRejectsTampering(t *testing.T) {
	cases := map[string]func(tx *Transaction, prev map[string]Transaction){
		"changed amount": func(tx *Transaction, _ map[string]Transaction) { tx.Vout[0].Value = 9 },
		"changed recipient": func(tx *Transaction, _ map[string]Transaction) {
			tx.Vout[0].PubKeyHash = HashPubKey(NewWallet().PublicKey)
		},
		"changed fee":       func(tx *Transaction, _ map[string]Transaction) { tx.Fee = 0 },
		"missing signature": func(tx *Transaction, _ map[string]Transaction) { tx.Vin[0].Signature = nil },
		"broken signature":  func(tx *Transaction, _ map[string]Transaction) { tx.Vin[0].Signature[5] ^= 0xff },
		"unknown previous":  func(_ *Transaction, prev map[string]Transaction) { clear(prev) },
		"output out of range": func(tx *Transaction, _ map[string]Transaction) {
			tx.Vin[0].Vout = 3
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			tx, prev, _, _ := signedTransfer(t)
			mutate(tx, prev)
			if tx.Verify(prev) {
				t.Fatal("tampered transaction verified")
			}
		})
	}
}

func TestVerifyRejectsSomeoneElsesKey(t *testing.T) {
	tx, prev, _, _ := signedTransfer(t)
	thief := NewWallet()

	tx.Vin[0].PubKey = thief.PublicKey
	if err := tx.Sign(thief.PrivateKey, prev); err != nil {
		t.Fatal(err)
	}
	if tx.Verify(prev) {
		t.Fatal("an input signed by a key that does not own the output verified")
	}
}

func TestSignFailsWithoutPreviousTransaction(t *testing.T) {
	tx, _, alice, _ := signedTransfer(t)
	if err := tx.Sign(alice.PrivateKey, map[string]Transaction{}); err == nil {
		t.Fatal("Sign must fail when the spent transaction is unknown")
	}
}

func TestTrimmedCopyDropsSignatures(t *testing.T) {
	tx, _, _, _ := signedTransfer(t)
	trimmed := tx.TrimmedCopy()
	if trimmed.Vin[0].Signature != nil || trimmed.Vin[0].PubKey != nil {
		t.Fatal("trimmed copy must not carry signatures or keys")
	}
	trimmed.Vout[0].Value = 100
	if tx.Vout[0].Value == 100 {
		t.Fatal("trimmed copy shares outputs with the original")
	}
}

func TestTransactionHelpersAndString(t *testing.T) {
	tx, _, alice, bob := signedTransfer(t)
	if tx.GetSenderAddress() != string(alice.GetAddress()) {
		t.Fatal("wrong sender")
	}
	if tx.GetRecipientAddress() != string(bob.GetAddress()) || tx.GetAmount() != 7 {
		t.Fatal("wrong recipient or amount")
	}

	text := tx.String()
	for _, want := range []string{"Transaction", "Input 0", "Output 1", "Fee: 1"} {
		if !strings.Contains(text, want) {
			t.Errorf("String() is missing %q", want)
		}
	}

	empty := &Transaction{}
	if empty.GetRecipientAddress() != "" || empty.GetAmount() != 0 || empty.GetSenderAddress() != "" {
		t.Fatal("empty transaction helpers must return zero values")
	}
}

func TestOutputLocking(t *testing.T) {
	wallet := NewWallet()
	out := NewTXOutput(5, string(wallet.GetAddress()))
	if !out.IsLockedWithKey(HashPubKey(wallet.PublicKey)) {
		t.Fatal("output is not locked to its owner")
	}
	if out.IsLockedWithKey(HashPubKey(NewWallet().PublicKey)) {
		t.Fatal("output is locked to a stranger")
	}

	in := TXInput{PubKey: wallet.PublicKey}
	if !in.UsesKey(HashPubKey(wallet.PublicKey)) {
		t.Fatal("input does not use its own key")
	}

	outs := TXOutputs{Outputs: []TXOutput{*out, {}, *NewTXOutput(9, string(wallet.GetAddress()))}}
	decoded := DeserializeOutputs(outs.Serialize())
	if len(decoded.Outputs) != 3 || decoded.Outputs[2].Value != 9 || len(decoded.Outputs[1].PubKeyHash) != 0 {
		t.Fatalf("outputs round trip = %+v", decoded)
	}
}
