package cli_controllers

import (
	"Blockchain/core"
	"fmt"
)

func (cli *CLI) createWallet(nodeID string) {
	wallets, _ := core.NewWallets(nodeID)
	address := wallets.CreateWallet()
	wallets.SaveToFile(nodeID)

	fmt.Printf("Your new address: %s\n", address)
}
