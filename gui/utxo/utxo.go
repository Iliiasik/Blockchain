package utxo

import (
	"Blockchain/core"
	"Blockchain/resources/icons"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"strconv"
)

type UTXOTab struct {
	window        fyne.Window
	utxoContainer *fyne.Container
	addressEntry  *widget.Entry
}

func NewUTXOTab(window fyne.Window) *container.TabItem {
	utxo := &UTXOTab{
		window:        window,
		utxoContainer: container.NewVBox(),
	}
	return utxo.createTab()
}

func (u *UTXOTab) createTab() *container.TabItem {
	u.addressEntry = widget.NewEntry()
	u.addressEntry.SetPlaceHolder("Enter address to view UTXOs")

	controls := container.NewVBox(
		widget.NewLabelWithStyle("UTXO Set", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			widget.NewButtonWithIcon("View UTXOs", theme.SearchIcon(), u.onViewUTXOs),
			widget.NewButtonWithIcon("Reindex UTXO", theme.ViewRefreshIcon(), u.onReindexUTXO),
		),
		u.addressEntry,
	)

	scroll := container.NewScroll(u.utxoContainer)
	scroll.SetMinSize(fyne.NewSize(800, 400))

	return container.NewTabItemWithIcon("UTXO",
		icons.ResourceUtxoPng,
		container.NewBorder(
			controls,
			nil,
			nil,
			nil,
			scroll,
		),
	)
}

func (u *UTXOTab) onViewUTXOs() {
	address := u.addressEntry.Text
	if address == "" {
		dialog.ShowError(fmt.Errorf("please enter an address"), u.window)
		return
	}

	if !core.ValidateAddress(address) {
		dialog.ShowError(fmt.Errorf("invalid address format"), u.window)
		return
	}

	bc, err := core.NewBlockchain()
	if err != nil {
		dialog.ShowError(err, u.window)
		return
	}
	defer bc.Db.Close()

	pubKeyHash := core.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOSet := core.UTXOSet{bc}
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	u.utxoContainer.RemoveAll()
	if len(UTXOs) == 0 {
		u.utxoContainer.Add(widget.NewLabel("No UTXOs found for this address"))
		return
	}

	for _, out := range UTXOs {
		u.addUTXOToContainer(out)
	}
	u.utxoContainer.Refresh()
}

func (u *UTXOTab) addUTXOToContainer(out core.TXOutput) {
	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Value: %d", out.Value)),
		widget.NewLabel(fmt.Sprintf("Locked with: %x", out.PubKeyHash)),
		widget.NewLabel(fmt.Sprintf("Script: %x", out.PubKeyHash)),
	)

	card := widget.NewCard(
		fmt.Sprintf("UTXO Output"),
		"",
		content,
	)
	u.utxoContainer.Add(card)
	u.utxoContainer.Add(layout.NewSpacer())
}

func (u *UTXOTab) onReindexUTXO() {
	bc, err := core.NewBlockchain()
	if err != nil {
		dialog.ShowError(err, u.window)
		return
	}
	defer bc.Db.Close()

	UTXOSet := core.UTXOSet{bc}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	dialog.ShowInformation("UTXO Reindex",
		"Reindex complete!\nTransactions: "+strconv.Itoa(count),
		u.window)
}
