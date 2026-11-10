package core

import (
	"bytes"
	"encoding/gob"
	"os"
	"sort"
)

type Wallets struct {
	Wallets map[string]*Wallet
}

func NewWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFromFile()

	return &wallets, err
}

func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := string(wallet.GetAddress())

	ws.Wallets[address] = wallet

	return address
}

func (ws *Wallets) GetAddresses() []string {
	addresses := make([]string, 0, len(ws.Wallets))

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	sort.Strings(addresses)

	return addresses
}

func (ws Wallets) GetWallet(address string) (*Wallet, bool) {
	wallet, ok := ws.Wallets[address]

	return wallet, ok
}

func (ws *Wallets) LoadFromFile() error {
	fileContent, err := os.ReadFile(walletFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var wallets Wallets
	if err := gob.NewDecoder(bytes.NewReader(fileContent)).Decode(&wallets); err != nil {
		return err
	}

	if wallets.Wallets != nil {
		ws.Wallets = wallets.Wallets
	}

	return nil
}

func (ws Wallets) SaveToFile() error {
	var content bytes.Buffer

	if err := gob.NewEncoder(&content).Encode(ws); err != nil {
		return err
	}

	return os.WriteFile(walletFile, content.Bytes(), 0600)
}
