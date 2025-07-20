package cli_controllers

import (
	"Blockchain/core"
	"fmt"
)

func (cli *CLI) reindexUTXO(nodeID string) {
	bc := core.NewBlockchain(nodeID)
	UTXOSet := core.UTXOSet{bc}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}
