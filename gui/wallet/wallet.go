package wallet

import (
	"Blockchain/core"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type WalletUI struct {
	window     fyne.Window
	list       *widget.List
	walletData *core.Wallets
}

func NewWalletTab(window fyne.Window) *container.TabItem {
	ui := &WalletUI{window: window}
	ui.walletData, _ = core.NewWallets()
	return ui.createTab()
}

func (w *WalletUI) createTab() *container.TabItem {
	createWalletBtn := widget.NewButtonWithIcon("Create New Wallet", theme.ContentAddIcon(), w.onCreateWallet)
	w.list = w.createAddressesList()
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), w.refreshList)

	buttonBar := container.NewHBox(
		layout.NewSpacer(),
		createWalletBtn,
		refreshBtn,
		layout.NewSpacer(),
	)

	scrollContainer := container.NewScroll(w.list)
	scrollContainer.SetMinSize(fyne.NewSize(400, 300))

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Wallet Management",
				fyne.TextAlignCenter,
				fyne.TextStyle{Bold: true}),
			buttonBar,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		scrollContainer,
	)

	return container.NewTabItem("Wallet", content)
}

func (w *WalletUI) onCreateWallet() {
	address := w.walletData.CreateWallet()
	w.walletData.SaveToFile()

	dialog.ShowInformation(
		"New Wallet Created",
		fmt.Sprintf("Address:\n%s", address),
		w.window,
	)
	w.refreshList()
}

func (w *WalletUI) refreshList() {
	var err error
	w.walletData, err = core.NewWallets()
	if err != nil {
		dialog.ShowError(err, w.window)
		return
	}
	w.list.Refresh()
}

func (w *WalletUI) createAddressesList() *widget.List {
	return widget.NewList(
		w.getAddressCount,
		w.createAddressItem,
		w.updateAddressItem,
	)
}

func (w *WalletUI) getAddressCount() int {
	if w.walletData == nil {
		return 0
	}
	return len(w.walletData.GetAddresses())
}

func (w *WalletUI) createAddressItem() fyne.CanvasObject {
	addressLabel := widget.NewLabel("")
	balanceLabel := widget.NewLabel("")

	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), nil)
	checkBtn := widget.NewButtonWithIcon("", theme.InfoIcon(), nil)

	copyBtn.Importance = widget.LowImportance
	checkBtn.Importance = widget.LowImportance

	rightBox := container.NewHBox(
		balanceLabel,
		widget.NewSeparator(),
		checkBtn,
		copyBtn,
	)

	return container.NewHBox(addressLabel, layout.NewSpacer(), rightBox)
}

func (w *WalletUI) updateAddressItem(i int, item fyne.CanvasObject) {
	if w.walletData == nil {
		return
	}

	addresses := w.walletData.GetAddresses()
	if i >= len(addresses) {
		return
	}

	address := addresses[i]
	row := item.(*fyne.Container)

	addressLabel := row.Objects[0].(*widget.Label)
	rightBox := row.Objects[2].(*fyne.Container)

	balanceLabel := rightBox.Objects[0].(*widget.Label)
	checkBtn := rightBox.Objects[2].(*widget.Button)
	copyBtn := rightBox.Objects[3].(*widget.Button)

	addressLabel.SetText(address)
	addressLabel.Refresh()

	balanceLabel.SetText("")
	balanceLabel.Refresh()

	copyBtn.OnTapped = func() {
		w.window.Clipboard().SetContent(address)
		dialog.ShowInformation(
			"Address Copied",
			"Wallet address copied to clipboard",
			w.window,
		)
	}

	checkBtn.OnTapped = func() {
		balance, err := w.getBalance(address)
		if err != nil {
			dialog.ShowError(err, w.window)
			return
		}
		balanceLabel.SetText(fmt.Sprintf("%d coins", balance))
		balanceLabel.Refresh()
	}
}

func (w *WalletUI) getBalance(address string) (int, error) {
	if !core.ValidateAddress(address) {
		return 0, fmt.Errorf("Invalid address")
	}

	bc, err := core.NewBlockchain()
	if err != nil {
		return 0, err
	}
	defer bc.Db.Close()

	UTXOSet := core.UTXOSet{bc}
	pubKeyHash := core.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	balance := 0
	for _, out := range UTXOs {
		balance += out.Value
	}

	return balance, nil
}
