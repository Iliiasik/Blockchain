package transaction

import (
	"Blockchain/core"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"strconv"
)

type TransactionUI struct {
	window fyne.Window
}

func NewTransactionTab(window fyne.Window) *container.TabItem {
	ui := &TransactionUI{window: window}
	return ui.createTab()
}

func (t *TransactionUI) createTab() *container.TabItem {
	fromBinding := binding.NewString()
	toBinding := binding.NewString()
	amountBinding := binding.NewString()

	fromEntry := widget.NewEntryWithData(fromBinding)
	fromEntry.SetPlaceHolder("Sender address")

	toEntry := widget.NewEntryWithData(toBinding)
	toEntry.SetPlaceHolder("Recipient address")

	amountEntry := widget.NewEntryWithData(amountBinding)
	amountEntry.SetPlaceHolder("Amount")

	amountEntry.OnChanged = func(s string) {
		filtered := ""
		for _, r := range s {
			if r >= '0' && r <= '9' {
				filtered += string(r)
			}
		}
		if s != filtered {
			amountEntry.SetText(filtered)
		}
	}

	sendBtn := widget.NewButton("Send transaction", func() {
		t.onSendTransaction(fromEntry.Text, toEntry.Text, amountEntry.Text)
	})
	sendBtn.Disable()

	checkInputs := func() {
		from, _ := fromBinding.Get()
		to, _ := toBinding.Get()
		amount, _ := amountBinding.Get()

		shouldEnable := from != "" && to != "" && amount != ""
		if shouldEnable {
			sendBtn.Enable()
		} else {
			sendBtn.Disable()
		}
	}

	fromBinding.AddListener(binding.NewDataListener(func() { checkInputs() }))
	toBinding.AddListener(binding.NewDataListener(func() { checkInputs() }))
	amountBinding.AddListener(binding.NewDataListener(func() { checkInputs() }))

	statusLabel := widget.NewLabel("")

	return container.NewTabItem("Transactions",
		container.NewVBox(
			widget.NewLabelWithStyle("Send transaction",
				fyne.TextAlignCenter,
				fyne.TextStyle{Bold: true}),

			widget.NewLabel("From address"),
			fromEntry,

			widget.NewLabel("To address"),
			toEntry,

			widget.NewLabel("Amount (integer)"),
			amountEntry,

			sendBtn,
			statusLabel,
		),
	)
}

func (t *TransactionUI) onSendTransaction(from, to, amount string) {
	defer func() {
		if r := recover(); r != nil {
			dialog.ShowError(fmt.Errorf("Transaction failed: %v", r), t.window)
		}
	}()

	if !core.ValidateAddress(from) {
		dialog.ShowError(fmt.Errorf("Invalid sender address"), t.window)
		return
	}
	if !core.ValidateAddress(to) {
		dialog.ShowError(fmt.Errorf("Invalid recipient address"), t.window)
		return
	}

	amountInt, err := strconv.Atoi(amount)
	if err != nil || amountInt <= 0 {
		dialog.ShowError(fmt.Errorf("Invalid amount"), t.window)
		return
	}

	bc, err := core.NewBlockchain()
	if err != nil {
		dialog.ShowError(err, t.window)
		return
	}
	defer bc.Db.Close()

	UTXOSet := core.UTXOSet{bc}
	tx, err := core.NewUTXOTransaction(from, to, amountInt, &UTXOSet)
	if err != nil {
		dialog.ShowError(err, t.window)
		return
	}

	cbTx := core.NewCoinbaseTX(from, "")
	newBlock := bc.MineBlock([]*core.Transaction{cbTx, tx})
	UTXOSet.Update(newBlock)

	dialog.ShowInformation("Success",
		fmt.Sprintf("Transaction sent!\nNew block mined: %x", newBlock.Hash),
		t.window)
}
