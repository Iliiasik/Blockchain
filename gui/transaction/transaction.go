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
	"strings"
)

type TransactionUI struct {
	window fyne.Window
	state  *state.AppState
}

func NewTransactionTab(window fyne.Window, state *state.AppState) *container.TabItem {
	ui := &TransactionUI{
		window: window,
		state:  state,
	}
	return ui.createTab()
}

func (t *TransactionUI) createTab() *container.TabItem {
	fromBinding := binding.NewString()
	toBinding := binding.NewString()
	amountBinding := binding.NewString()
	feeBinding := binding.NewString()
	feeBinding.Set("0")

	fromEntry := t.createAddressEntry(fromBinding, "Sender address")
	toEntry := t.createAddressEntry(toBinding, "Recipient address")
	amountEntry := t.createAmountEntry(amountBinding)
	feeEntry := t.createFeeEntry(feeBinding)

	sendBtn := t.createSendButton(fromEntry, toEntry, amountEntry, feeBinding)

	t.setupInputValidation(fromBinding, toBinding, amountBinding, feeBinding,
		fromEntry, toEntry, amountEntry, feeEntry, sendBtn)

	form := t.createForm(fromEntry, toEntry, amountEntry, feeEntry)
	content := t.createContent(form, sendBtn)

	return container.NewTabItemWithIcon("Transactions", icons.ResourceTransactionsPng,
		container.NewPadded(content))
}

func (t *TransactionUI) createAddressEntry(binding binding.String, placeholder string) *widget.Entry {
	entry := widget.NewEntryWithData(binding)
	entry.SetPlaceHolder(placeholder)
	entry.Validator = func(s string) error {
		if !core.ValidateAddress(s) && s != "" {
			return fmt.Errorf("invalid address")
		}
		return nil
	}
	return entry
}

func (t *TransactionUI) createAmountEntry(binding binding.String) *widget.Entry {
	entry := widget.NewEntryWithData(binding)
	entry.SetPlaceHolder("Amount (integer)")
	entry.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		_, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("must be a positive integer")
		}
		return nil
	}
	t.attachNumericFilter(entry)
	return entry
}

func (t *TransactionUI) createFeeEntry(binding binding.String) *widget.Entry {
	entry := widget.NewEntryWithData(binding)
	entry.SetPlaceHolder("Fee (integer)")
	entry.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		_, err := strconv.Atoi(s)
		if err != nil || s[0] == '-' {
			return fmt.Errorf("must be a non-negative integer")
		}
		return nil
	}
	t.attachNumericFilter(entry)
	return entry
}

func (t *TransactionUI) attachNumericFilter(entry *widget.Entry) {
	entry.OnChanged = func(s string) {
		filtered := ""
		for _, r := range s {
			if r >= '0' && r <= '9' {
				filtered += string(r)
			}
		}
		if s != filtered {
			entry.SetText(filtered)
		}
	}
}

func (t *TransactionUI) createSendButton(fromEntry, toEntry, amountEntry *widget.Entry, feeBinding binding.String) *widget.Button {
	sendBtn := widget.NewButtonWithIcon("Send transaction", theme.MailSendIcon(), func() {
		fee, _ := feeBinding.Get()
		t.onSendTransaction(fromEntry.Text, toEntry.Text, amountEntry.Text, fee)
	})
	sendBtn.Disable()
	return sendBtn
}

func (t *TransactionUI) setupInputValidation(fromBinding, toBinding, amountBinding, feeBinding binding.String,
	fromEntry, toEntry, amountEntry, feeEntry *widget.Entry, sendBtn *widget.Button) {

	checkInputs := func() {
		from, _ := fromBinding.Get()
		to, _ := toBinding.Get()
		amount, _ := amountBinding.Get()
		fee, _ := feeBinding.Get()

		shouldEnable := from != "" && to != "" && amount != "" && fee != "" &&
			fromEntry.Validate() == nil &&
			toEntry.Validate() == nil &&
			amountEntry.Validate() == nil &&
			feeEntry.Validate() == nil

		if shouldEnable {
			sendBtn.Enable()
		} else {
			sendBtn.Disable()
		}
	}

	fromBinding.AddListener(binding.NewDataListener(checkInputs))
	toBinding.AddListener(binding.NewDataListener(checkInputs))
	amountBinding.AddListener(binding.NewDataListener(checkInputs))
	feeBinding.AddListener(binding.NewDataListener(checkInputs))
}

func (t *TransactionUI) createForm(fromEntry, toEntry, amountEntry, feeEntry *widget.Entry) *widget.Form {
	return &widget.Form{
		Items: []*widget.FormItem{
			{Widget: fromEntry, HintText: "Sender's wallet address"},
			{Widget: toEntry, HintText: "Recipient's wallet address"},
			{Widget: amountEntry, HintText: "Amount to send (integer)"},
			{Widget: feeEntry, HintText: "Transaction fee (integer, min 0)"},
		},
		SubmitText: "",
		CancelText: "",
	}
}

func (t *TransactionUI) createContent(form *widget.Form, sendBtn *widget.Button) *fyne.Container {
	return container.NewVBox(
		container.NewCenter(
			widget.NewLabelWithStyle("Send transaction", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		),
		container.NewPadded(
			container.NewVBox(
				container.NewPadded(form),
				container.NewCenter(sendBtn),
			),
		),
		layout.NewSpacer(),
	)
}

func (t *TransactionUI) onSendTransaction(from, to, amount, fee string) {
	defer func() {
		if r := recover(); r != nil {
			dialog.ShowError(fmt.Errorf("Transaction failed: %v", r), t.window)
		}
	}()

	amountInt, err := strconv.Atoi(amount)
	if err != nil || amountInt <= 0 {
		dialog.ShowError(fmt.Errorf("Amount must be a positive integer"), t.window)
		return
	}

	feeInt, err := strconv.Atoi(fee)
	if err != nil || feeInt < 0 {
		dialog.ShowError(fmt.Errorf("Fee must be a non-negative integer"), t.window)
		return
	}

	go func() {
		bc, err := core.NewBlockchain()
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("Failed to access blockchain: %v", err), t.window)
			})
			return
		}
		defer bc.Db.Close()

		UTXOSet := core.UTXOSet{bc}
		_, err = core.NewUTXOTransaction(from, to, amountInt, feeInt, &UTXOSet)
		if err != nil {
			var errorMsg string
			switch {
			case strings.Contains(err.Error(), "wallet already has pending transaction"):
				errorMsg = "Your wallet already has a pending transaction.\nPlease mine a block to confirm it before sending another."
			case strings.Contains(err.Error(), "mempool is full"):
				errorMsg = "The mempool is full. Please wait until some transactions are mined."
			default:
				errorMsg = fmt.Sprintf("Transaction creation failed: %v", err)
			}

			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf(errorMsg), t.window)
			})
			return
		}

		fyne.Do(func() {
			dialog.ShowInformation(
				"Transaction Sent",
				fmt.Sprintf("Amount: %d\nFee: %d", amountInt, feeInt),
				t.window,
			)
		})
	}()
}
