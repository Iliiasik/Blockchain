package core

import (
	"bytes"
	"encoding/hex"
	"os"
	"runtime"
	"testing"
)

func TestNewWalletAddress(t *testing.T) {
	wallet := NewWallet()
	address := string(wallet.GetAddress())

	if len(wallet.PublicKey) != 2*coordinateSize {
		t.Fatalf("public key length = %d", len(wallet.PublicKey))
	}
	if address[0] != '1' {
		t.Fatalf("address %s must start with 1 for version 0x00", address)
	}
	if !ValidateAddress(address) {
		t.Fatalf("own address %s is invalid", address)
	}
	if len(HashPubKey(wallet.PublicKey)) != 20 {
		t.Fatal("public key hash must be 20 bytes")
	}
	if !bytes.Equal(AddressToPubKeyHash(address), HashPubKey(wallet.PublicKey)) {
		t.Fatal("address does not decode to the key hash")
	}
	if PubKeyToAddress(wallet.PublicKey) != address {
		t.Fatal("PubKeyToAddress disagrees with GetAddress")
	}
}

func TestValidateAddressRejectsBrokenInput(t *testing.T) {
	address := string(NewWallet().GetAddress())
	raw := Base58Decode([]byte(address))

	badChecksum := append([]byte{}, raw...)
	badChecksum[len(badChecksum)-1] ^= 1

	badVersion := append([]byte{}, raw...)
	badVersion[0] = 5

	cases := map[string]string{
		"empty":          "",
		"checksum":       string(Base58Encode(badChecksum)),
		"version":        string(Base58Encode(badVersion)),
		"too short":      address[:len(address)-3],
		"invalid symbol": "0" + address[1:],
	}
	for name, input := range cases {
		if ValidateAddress(input) {
			t.Errorf("%s: %q accepted", name, input)
		}
		if AddressToPubKeyHash(input) != nil {
			t.Errorf("%s: %q decoded to a key hash", name, input)
		}
	}
}

func TestWalletsPersistence(t *testing.T) {
	useTempDir(t)

	empty, err := NewWallets()
	if err != nil {
		t.Fatalf("missing wallet.dat must not be an error: %v", err)
	}
	if len(empty.GetAddresses()) != 0 {
		t.Fatal("new store must be empty")
	}

	first := empty.CreateWallet()
	second := empty.CreateWallet()
	if err := empty.SaveToFile(); err != nil {
		t.Fatal(err)
	}

	loaded, err := NewWallets()
	if err != nil {
		t.Fatal(err)
	}
	addresses := loaded.GetAddresses()
	if len(addresses) != 2 || addresses[0] > addresses[1] {
		t.Fatalf("addresses = %v, want two sorted entries", addresses)
	}

	for _, address := range []string{first, second} {
		original, _ := empty.GetWallet(address)
		restored, ok := loaded.GetWallet(address)
		if !ok {
			t.Fatalf("wallet %s was not restored", address)
		}
		want, _ := original.PrivateKey.Bytes()
		got, err := restored.PrivateKey.Bytes()
		if err != nil || !bytes.Equal(original.PublicKey, restored.PublicKey) || !bytes.Equal(want, got) {
			t.Fatal("restored keys differ")
		}
		if string(restored.GetAddress()) != address {
			t.Fatal("restored wallet has another address")
		}
	}

	if _, ok := loaded.GetWallet("nope"); ok {
		t.Fatal("unknown address returned a wallet")
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(walletFile)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("wallet.dat mode = %v, want 0600", info.Mode().Perm())
		}
	}
}

func TestRestoredWalletCanSign(t *testing.T) {
	useTempDir(t)
	store, err := NewWallets()
	if err != nil {
		t.Fatal(err)
	}
	address := store.CreateWallet()
	if err := store.SaveToFile(); err != nil {
		t.Fatal(err)
	}

	loaded, err := NewWallets()
	if err != nil {
		t.Fatal(err)
	}
	wallet, _ := loaded.GetWallet(address)

	funding := NewCoinbaseTX(address, "funding", 10)
	tx := &Transaction{
		Vin:  []TXInput{{Txid: funding.ID, Vout: 0, PubKey: wallet.PublicKey}},
		Vout: []TXOutput{*NewTXOutput(10, string(NewWallet().GetAddress()))},
	}
	tx.ID = tx.Hash()
	prev := map[string]Transaction{hex.EncodeToString(funding.ID): *funding}

	if err := tx.Sign(wallet.PrivateKey, prev); err != nil {
		t.Fatal(err)
	}
	if !tx.Verify(prev) {
		t.Fatal("a wallet loaded from disk produced an invalid signature")
	}
}

func TestCorruptWalletFile(t *testing.T) {
	useTempDir(t)
	if err := os.WriteFile(walletFile, []byte("not a gob"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewWallets(); err == nil {
		t.Fatal("a corrupt wallet.dat must be reported")
	}
}
