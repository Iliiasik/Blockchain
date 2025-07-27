package blockchain

import (
	"Blockchain/core"
	"fyne.io/fyne/v2/dialog"
	"strconv"
)

func (b *BlockchainUI) onReindexUTXO() {
	bc, err := core.NewBlockchain()
	if err != nil {
		dialog.ShowError(err, b.window)
		return
	}
	defer bc.Db.Close()

	UTXOSet := core.UTXOSet{bc}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	dialog.ShowInformation("UTXO Reindex",
		"Reindex complete!\nTransactions: "+strconv.Itoa(count),
		b.window)
}
