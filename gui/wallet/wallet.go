package wallet

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"Blockchain/resources/icons"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"time"
)

type WalletUI struct {
	window     fyne.Window
	list       *widget.List
	walletData *core.Wallets
	state      *state.AppState
}

func NewWalletTab(window fyne.Window, state *state.AppState) *container.TabItem {
	ui := &WalletUI{window: window, state: state}
	ui.walletData, _ = core.NewWallets()
	return ui.createTab()
}

func (w *WalletUI) createTab() *container.TabItem {
	createWalletBtn := widget.NewButtonWithIcon("Create new wallet", theme.ContentAddIcon(), w.onCreateWallet)
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
			widget.NewLabelWithStyle("Wallets management",
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

	return container.NewTabItemWithIcon("Wallets", icons.ResourceWalletsPng, content)
}

func (w *WalletUI) onCreateWallet() {
	address := w.walletData.CreateWallet()
	w.walletData.SaveToFile()

	dialog.ShowInformation(
		"New wallet created",
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
	mineBtn := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), nil)

	copyBtn.Importance = widget.LowImportance
	checkBtn.Importance = widget.LowImportance
	mineBtn.Importance = widget.LowImportance

	rightBox := container.NewHBox(
		balanceLabel,
		widget.NewSeparator(),
		checkBtn,
		copyBtn,
		mineBtn,
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

	var balanceContainer *fyne.Container
	if len(rightBox.Objects) > 0 {
		switch obj := rightBox.Objects[0].(type) {
		case *widget.Label:
			balanceContainer = container.NewHBox()
			rightBox.Objects[0] = balanceContainer
			rightBox.Refresh()
		case *fyne.Container:
			balanceContainer = obj
		default:
			balanceContainer = container.NewHBox()
			rightBox.Objects[0] = balanceContainer
			rightBox.Refresh()
		}
	}

	checkBtn := rightBox.Objects[2].(*widget.Button)
	copyBtn := rightBox.Objects[3].(*widget.Button)
	mineBtn := rightBox.Objects[4].(*widget.Button)

	addressLabel.SetText(address)

	balance, err := w.getBalance(address)
	if err == nil {
		coinIcon := canvas.NewImageFromResource(icons.ResourceCoinPng)
		coinIcon.SetMinSize(fyne.NewSize(40, 40))

		balanceContainer.Objects = []fyne.CanvasObject{
			widget.NewLabel(fmt.Sprintf("%d", balance)),
			coinIcon,
		}
		balanceContainer.Refresh()
	} else {
		balanceContainer.Objects = []fyne.CanvasObject{
			widget.NewLabel("Create blockchain first"),
		}
		balanceContainer.Refresh()
	}

	copyBtn.OnTapped = func() {
		w.window.Clipboard().SetContent(address)
		dialog.ShowInformation(
			"Address copied",
			"Wallet address copied to clipboard",
			w.window,
		)
	}

	checkBtn.OnTapped = func() {
		w.showTransactionHistory(address)
	}
	mineBtn.OnTapped = func() {
		stopAnimation := make(chan struct{})
		go func() {
			labels := []string{"Mining 💎 ⛏️", "Mining 💥 ⛏️"}
			i := 0

			for {
				select {
				case <-stopAnimation:
					return
				default:
					fyne.Do(func() {
						addressLabel.SetText(labels[i%len(labels)])
						addressLabel.Refresh()
					})
					i++
					time.Sleep(400 * time.Millisecond)
				}
			}
		}()

		go func() {
			bc, err := core.NewBlockchain()
			if err != nil {
				fyne.Do(func() {
					close(stopAnimation)
					addressLabel.SetText(address)
					addressLabel.Refresh()

					dialog.ShowError(err, w.window)
				})
				return
			}
			defer bc.Db.Close()

			cbTx := core.NewCoinbaseTX(address, "", w.state.Subsidy)
			block := bc.MineBlock([]*core.Transaction{cbTx}, w.state.TargetBits)

			utxo := core.UTXOSet{bc}
			utxo.Update(block)

			fyne.Do(func() {
				close(stopAnimation)
				addressLabel.SetText(address)
				addressLabel.Refresh()

				dialog.ShowInformation(
					"Block Mined",
					fmt.Sprintf("Successfully mined a new block!\nHash:\n%x", block.Hash),
					w.window,
				)
				w.refreshList()
			})
		}()
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
