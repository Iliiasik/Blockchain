package transaction

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"Blockchain/resources/icons"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"strconv"
)

type TransactionUI struct {
	window         fyne.Window
	state          *state.AppState
	blocks         []*core.Block
	loadingSpinner *widget.ProgressBarInfinite
}

func NewTransactionTab(window fyne.Window, state *state.AppState) *container.TabItem {
	ui := &TransactionUI{
		window: window,
		state:  state,
	}
	ui.loadingSpinner = widget.NewProgressBarInfinite()
	ui.loadingSpinner.Hide()
	return ui.createTab()
}

func (t *TransactionUI) createTab() *container.TabItem {
	fromBinding := binding.NewString()
	toBinding := binding.NewString()
	amountBinding := binding.NewString()

	fromEntry := widget.NewEntryWithData(fromBinding)
	fromEntry.SetPlaceHolder("Sender address")
	fromEntry.Validator = func(s string) error {
		if !core.ValidateAddress(s) && s != "" {
			return fmt.Errorf("invalid sender address")
		}
		return nil
	}

	toEntry := widget.NewEntryWithData(toBinding)
	toEntry.SetPlaceHolder("Recipient address")
	toEntry.Validator = func(s string) error {
		if !core.ValidateAddress(s) && s != "" {
			return fmt.Errorf("invalid recipient address")
		}
		return nil
	}

	amountEntry := widget.NewEntryWithData(amountBinding)
	amountEntry.SetPlaceHolder("Amount (integer)")
	amountEntry.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		_, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("must be a positive integer")
		}
		return nil
	}

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

	sendBtn := widget.NewButtonWithIcon("Send transaction", theme.MailSendIcon(), func() {
		t.onSendTransaction(fromEntry.Text, toEntry.Text, amountEntry.Text)
	})
	sendBtn.Disable()

	checkInputs := func() {
		from, _ := fromBinding.Get()
		to, _ := toBinding.Get()
		amount, _ := amountBinding.Get()

		shouldEnable := from != "" && to != "" && amount != "" &&
			fromEntry.Validate() == nil &&
			toEntry.Validate() == nil &&
			amountEntry.Validate() == nil

		if shouldEnable {
			sendBtn.Enable()
		} else {
			sendBtn.Disable()
		}
	}

	fromBinding.AddListener(binding.NewDataListener(checkInputs))
	toBinding.AddListener(binding.NewDataListener(checkInputs))
	amountBinding.AddListener(binding.NewDataListener(checkInputs))

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Widget: fromEntry, HintText: "Sender's wallet address"},
			{Widget: toEntry, HintText: "Recipient's wallet address"},
			{Widget: amountEntry, HintText: "Amount to send (integer)"},
		},
		SubmitText: "",
		CancelText: "",
	}

	content := container.NewVBox(
		container.NewCenter(
			widget.NewLabelWithStyle("Send transaction", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		),
		container.NewPadded(
			container.NewVBox(
				container.NewPadded(form),
				container.NewMax(t.loadingSpinner),
				container.NewCenter(sendBtn),
			),
		),
		layout.NewSpacer(),
	)

	return container.NewTabItemWithIcon("Transactions", icons.ResourceTransactionsPng,
		container.NewPadded(content))
}

func (t *TransactionUI) onSendTransaction(from, to, amount string) {
	defer func() {
		if r := recover(); r != nil {
			t.loadingSpinner.Hide()
			dialog.ShowError(fmt.Errorf("transaction failed: %v", r), t.window)
		}
	}()

	amountInt, err := strconv.Atoi(amount)
	if err != nil || amountInt <= 0 {
		dialog.ShowError(fmt.Errorf("invalid amount"), t.window)
		return
	}

	subsidy := t.state.Subsidy
	if subsidy <= 0 {
		dialog.ShowError(fmt.Errorf("invalid subsidy"), t.window)
		return
	}

	t.loadingSpinner.Show()

	go func() {
		bc, err := core.NewBlockchain()
		if err != nil {
			fyne.Do(func() {
				t.loadingSpinner.Hide()
				dialog.ShowError(err, t.window)
			})
			return
		}
		defer bc.Db.Close()

		UTXOSet := core.UTXOSet{bc}
		tx, err := core.NewUTXOTransaction(from, to, amountInt, &UTXOSet)
		if err != nil {
			fyne.Do(func() {
				t.loadingSpinner.Hide()
				dialog.ShowError(err, t.window)
			})
			return
		}

		cbTx := core.NewCoinbaseTX(from, "", subsidy)
		newBlock := bc.MineBlock([]*core.Transaction{cbTx, tx}, t.state.TargetBits)
		UTXOSet.Update(newBlock)

		fyne.Do(func() {
			t.loadingSpinner.Hide()
			dialog.ShowInformation("Success",
				fmt.Sprintf("Transaction sent!\nNew block mined: %x", newBlock.Hash),
				t.window)
		})
	}()
}
