package wallet

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"Blockchain/resources/icons"
	"encoding/hex"
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
	addressContainer := container.NewHBox()

	addressLabel := widget.NewLabel("")
	addressContainer.Add(addressLabel)

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

	return container.NewHBox(
		addressContainer,
		layout.NewSpacer(),
		rightBox,
	)
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
	addressContainer := row.Objects[0].(*fyne.Container)
	addressLabel := addressContainer.Objects[0].(*widget.Label)
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
		coinIcon.SetMinSize(fyne.NewSize(35, 35))
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
		bc, err := core.NewBlockchain()
		if err != nil {
			dialog.ShowError(err, w.window)
			return
		}
		defer bc.Db.Close()

		options := []string{"Mine empty block (only coinbase)", "Mine with selected transactions"}
		radio := widget.NewRadioGroup(options, nil)
		radio.SetSelected(options[0])

		var miningTypeDialog dialog.Dialog

		continueBtn := widget.NewButton("Continue", func() {
			miningTypeDialog.Hide()

			if radio.Selected == options[0] {
				cbTx := core.NewCoinbaseTX(address, "", w.state.Subsidy)
				transactions := []*core.Transaction{cbTx}
				miningBC, err := core.NewBlockchain()
				if err != nil {
					dialog.ShowError(err, w.window)
					return
				}
				w.startMiningProcess(addressContainer, transactions, miningBC)
			} else {
				miningBC, err := core.NewBlockchain()
				if err != nil {
					dialog.ShowError(err, w.window)
					return
				}
				defer miningBC.Db.Close()

				mempoolTxs := miningBC.Mempool.GetTransactions()
				if len(mempoolTxs) == 0 {
					dialog.ShowInformation("Mempool Empty", "No transactions in mempool", w.window)
					return
				}

				selectedTxs := make(map[string]*core.Transaction)
				totalFees := 0
				infoLabel := widget.NewLabel("")
				updateInfo := func() {
					infoLabel.SetText(fmt.Sprintf("Selected: %d | Fees: %d | Reward: %d",
						len(selectedTxs), totalFees, totalFees+w.state.Subsidy))
				}
				updateInfo()

				txList := widget.NewList(
					func() int { return len(mempoolTxs) },
					func() fyne.CanvasObject {
						return container.NewHBox(
							widget.NewCheck("", nil),
							widget.NewLabel("TXID:"),
							widget.NewLabel(""),
							widget.NewLabel("Fee:"),
							widget.NewLabel(""),
						)
					},
					func(i int, item fyne.CanvasObject) {
						tx := mempoolTxs[i]
						c := item.(*fyne.Container)
						check := c.Objects[0].(*widget.Check)
						txIDLabel := c.Objects[2].(*widget.Label)
						feeLabel := c.Objects[4].(*widget.Label)
						txID := hex.EncodeToString(tx.ID)
						txIDLabel.SetText(txID[:30] + "...")
						feeLabel.SetText(fmt.Sprintf("%d", tx.Fee))
						check.OnChanged = func(checked bool) {
							if checked {
								selectedTxs[txID] = tx
								totalFees += tx.Fee
							} else {
								delete(selectedTxs, txID)
								totalFees -= tx.Fee
							}
							updateInfo()
						}
					},
				)

				scrollContainer := container.NewScroll(txList)
				scrollContainer.SetMinSize(fyne.NewSize(0, 300))
				content := container.NewBorder(nil, infoLabel, nil, nil, scrollContainer)

				var txSelectDialog dialog.Dialog

				startMiningBtn := widget.NewButton("Start Mining", func() {
					txSelectDialog.Hide()
					var transactions []*core.Transaction
					for _, tx := range selectedTxs {
						transactions = append(transactions, tx)
					}
					cbTx := core.NewCoinbaseTX(address, "", w.state.Subsidy+totalFees)
					transactions = append([]*core.Transaction{cbTx}, transactions...)
					miningBC, err := core.NewBlockchain()
					if err != nil {
						dialog.ShowError(err, w.window)
						return
					}
					w.startMiningProcess(addressContainer, transactions, miningBC)
				})

				closeBtn := widget.NewButton("Close", func() {
					txSelectDialog.Hide()
				})

				buttons := container.NewHBox(layout.NewSpacer(), closeBtn, startMiningBtn)
				dialogContent := container.NewVBox(
					widget.NewLabel("Select transactions to include:"),
					content,
					buttons,
				)

				txSelectDialog = dialog.NewCustomWithoutButtons(
					"Select transactions",
					dialogContent,
					w.window,
				)
				txSelectDialog.Resize(fyne.NewSize(600, 400))
				txSelectDialog.Show()
			}
		})

		cancelBtn := widget.NewButton("Cancel", func() {
			miningTypeDialog.Hide()
		})

		miningTypeContent := container.NewVBox(
			widget.NewLabel("Select mining type:"),
			radio,
			container.NewHBox(layout.NewSpacer(), cancelBtn, continueBtn),
		)

		miningTypeDialog = dialog.NewCustomWithoutButtons(
			"Mining options",
			miningTypeContent,
			w.window,
		)
		miningTypeDialog.Resize(fyne.NewSize(400, 180))
		miningTypeDialog.Show()
	}

}

func (w *WalletUI) startMiningProcess(addressContainer *fyne.Container, transactions []*core.Transaction, bc *core.Blockchain) {
	stopAnimation := make(chan struct{})
	miningIcon := canvas.NewImageFromResource(icons.ResourceMiningPng)
	miningIcon.SetMinSize(fyne.NewSize(35, 35))

	go func() {
		showIcon := false
		for {
			select {
			case <-stopAnimation:
				fyne.Do(func() {
					if len(addressContainer.Objects) > 1 {
						addressContainer.Objects = addressContainer.Objects[:1]
						addressContainer.Refresh()
					}
				})
				return
			default:
				fyne.Do(func() {
					if showIcon {
						if len(addressContainer.Objects) == 1 {
							addressContainer.Add(miningIcon)
						}
					} else {
						if len(addressContainer.Objects) > 1 {
							addressContainer.Objects = addressContainer.Objects[:1]
						}
					}
					addressContainer.Refresh()
					showIcon = !showIcon
				})
				time.Sleep(400 * time.Millisecond)
			}
		}
	}()

	go func() {
		defer bc.Db.Close()
		block := bc.MineBlock(transactions, w.state.TargetBits)
		close(stopAnimation)
		if block == nil {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("mining failed"), w.window)
			})
			return
		}
		utxo := core.UTXOSet{bc}
		utxo.Update(block)
		reward := w.state.Subsidy
		if len(transactions) > 1 {
			fees := 0
			for _, tx := range transactions[1:] {
				fees += tx.Fee
			}
			reward += fees
			fyne.Do(func() {
				dialog.ShowInformation(
					"Block Mined",
					fmt.Sprintf("Successfully mined block!\nHash: %x\nReward: %d",
						block.Hash, reward),
					w.window,
				)
			})
		} else {
			fyne.Do(func() {
				dialog.ShowInformation(
					"Block Mined",
					fmt.Sprintf("Successfully mined empty block!\nHash: %x\nReward: %d",
						block.Hash, reward),
					w.window,
				)
			})
		}
		fyne.Do(func() {
			w.refreshList()
		})
	}()
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
