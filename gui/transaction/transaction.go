package transaction

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"Blockchain/gui/ui"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type TransactionUI struct {
	state *state.AppState
	snap  *state.Snapshot

	content  fyne.CanvasObject
	from     *widget.Select
	to       *widget.SelectEntry
	amount   *widget.Entry
	fee      *widget.Entry
	summary  *fyne.Container
	feedback *ui.Text
	send     *widget.Button
	pool     *fyne.Container
	capacity *ui.Meter
	size     *ui.Text
}

func New(st *state.AppState) *TransactionUI {
	t := &TransactionUI{state: st, snap: st.Snapshot()}
	t.content = t.build()
	st.Subscribe(t.update)
	return t
}

func (t *TransactionUI) Content() fyne.CanvasObject {
	return t.content
}

func (t *TransactionUI) Focus(address string) {
	t.from.SetSelected(address)
}

func (t *TransactionUI) build() fyne.CanvasObject {
	t.from = widget.NewSelect(nil, func(string) { t.validate() })
	t.from.PlaceHolder = "Sender wallet"

	t.to = widget.NewSelectEntry(nil)
	t.to.SetPlaceHolder("Recipient address, type or pick")
	t.to.Validator = func(s string) error {
		if s != "" && !core.ValidateAddress(s) {
			return errors.New("invalid address")
		}
		return nil
	}
	t.to.OnChanged = func(string) { t.validate() }

	t.amount = numericEntry("Amount", t.validate)
	t.fee = numericEntry("Fee", t.validate)
	t.fee.SetText("0")

	t.summary = container.NewVBox()
	t.feedback = ui.Secondary("")
	t.send = ui.Primary("Sign and send", theme.MailSendIcon(), t.onSend)

	form := &widget.Form{Items: []*widget.FormItem{
		{Text: "From", Widget: t.from, HintText: "Sender's wallet address"},
		{Text: "To", Widget: t.to, HintText: "Recipient's wallet address"},
		{Text: "Amount", Widget: t.amount, HintText: "Amount to send (integer)"},
		{Text: "Fee", Widget: t.fee, HintText: "Transaction fee (integer, min 0), goes to the miner"},
	}}

	composer := ui.NewPanel(container.NewVBox(
		ui.Section("Compose transaction", "Inputs are collected from your unspent outputs, signed with the private key and placed in the mempool until a miner includes them in a block."),
		form,
		ui.Divider(),
		t.summary,
		container.NewBorder(nil, nil, nil, t.send, t.feedback),
	))

	t.pool = container.New(layout.NewCustomPaddedVBoxLayout(8))
	t.capacity = ui.NewMeter()
	t.size = ui.Secondary("")

	sizeSlider := widget.NewSlider(1, 50)
	sizeSlider.Step = 1
	sizeSlider.SetValue(float64(t.state.MempoolSize))
	sizeSlider.OnChangeEnded = func(v float64) {
		t.state.SetMempoolSize(int(v))
		t.state.Refresh()
	}
	sizeSlider.OnChanged = func(v float64) {
		t.size.SetText(fmt.Sprintf("Mempool capacity: %d", int(v)))
	}
	t.size.SetText(fmt.Sprintf("Mempool capacity: %d", t.state.MempoolSize))

	mempool := ui.NewSurface(container.NewBorder(
		container.NewVBox(
			container.NewBorder(nil, nil, container.NewCenter(ui.Section("Mempool", "")),
				ui.Flat("Mine pending", theme.MediaPlayIcon(), func() { t.state.Go("mining", "") })),
			ui.Padded(8, 4, 0, 0, t.capacity),
			t.size,
			sizeSlider,
			ui.Padded(4, 8, 0, 0, ui.Divider()),
		),
		nil, nil, nil,
		ui.NewNestedScroll(t.pool),
	))
	mempool.Padding = 20

	return container.NewBorder(nil, nil, nil, ui.Padded(0, 0, 16, 0, ui.FixedWidth(420, mempool)), ui.NewNestedScroll(composer))
}

func numericEntry(placeholder string, changed func()) *widget.Entry {
	entry := widget.NewEntry()
	entry.SetPlaceHolder(placeholder)
	entry.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		if _, err := strconv.Atoi(s); err != nil {
			return errors.New("must be a non-negative integer")
		}
		return nil
	}
	entry.OnChanged = func(s string) {
		filtered := make([]rune, 0, len(s))
		for _, r := range s {
			if r >= '0' && r <= '9' {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) > 9 {
			filtered = filtered[:9]
		}
		if string(filtered) != s {
			entry.SetText(string(filtered))
			return
		}
		changed()
	}
	return entry
}

func (t *TransactionUI) update(snap *state.Snapshot) {
	t.snap = snap

	options := make([]string, 0, len(snap.Wallets))
	for _, w := range snap.Wallets {
		options = append(options, w.Address)
	}
	current := t.from.Selected
	t.from.SetOptions(options)
	if current != "" {
		t.from.SetSelected(current)
	} else if len(options) > 0 {
		t.from.SetSelected(options[0])
	}
	t.to.SetOptions(options)

	if snap.MempoolMax > 0 {
		t.capacity.SetValue(float64(len(snap.Mempool)) / float64(snap.MempoolMax))
	}

	t.pool.RemoveAll()
	if len(snap.Mempool) == 0 {
		t.pool.Add(ui.Muted("No pending transactions."))
	}
	for _, tx := range snap.Mempool {
		t.pool.Add(t.poolRow(tx))
	}
	t.pool.Refresh()

	t.validate()
}

func (t *TransactionUI) poolRow(tx *core.Transaction) fyne.CanvasObject {
	return ui.NewRaised(container.NewVBox(
		container.NewBorder(nil, nil,
			ui.Mono(ui.ShortString(hex.EncodeToString(tx.ID), 12, 6), theme.ColorNameForeground),
			ui.Secondary(fmt.Sprintf("Fee %d", tx.Fee)),
		),
		ui.Caption(fmt.Sprintf("%s to %s", ui.ShortAddress(tx.GetSenderAddress()), ui.ShortAddress(tx.GetRecipientAddress()))),
		ui.Caption(fmt.Sprintf("Amount %d, %d inputs, %d outputs", tx.GetAmount(), len(tx.Vin), len(tx.Vout))),
	))
}

func (t *TransactionUI) values() (amount, fee int, problem string) {
	amount, err := strconv.Atoi(t.amount.Text)
	if err != nil || amount <= 0 {
		return 0, 0, "Amount must be a positive integer"
	}
	fee, err = strconv.Atoi(t.fee.Text)
	if err != nil || fee < 0 {
		return amount, 0, "Fee must be a non-negative integer"
	}
	return amount, fee, ""
}

func (t *TransactionUI) validate() {
	if t.send == nil {
		return
	}

	amount, fee, inputProblem := t.values()
	info, hasSender := t.snap.Wallet(t.from.Selected)
	problem := ""

	switch {
	case !t.snap.HasChain:
		problem = "Create the blockchain first"
	case len(t.snap.Wallets) == 0:
		problem = "Create a wallet first"
	case !hasSender:
		problem = "Choose the sender wallet"
	case t.to.Text == "" || t.to.Validate() != nil:
		problem = "Enter a valid recipient address"
	case inputProblem != "":
		problem = inputProblem
	case amount+fee > info.Available:
		problem = fmt.Sprintf("Not enough spendable funds, %d available", info.Available)
	case len(t.snap.Mempool) >= t.snap.MempoolMax:
		problem = "The mempool is full. Mine a block first."
	}

	for _, tx := range t.snap.Mempool {
		if hasSender && tx.GetSenderAddress() == info.Address {
			problem = "This wallet already has a pending transaction. Mine a block to confirm it."
		}
	}

	t.summary.RemoveAll()
	if hasSender {
		change := info.Available - amount - fee
		t.summary.Add(ui.KV(
			ui.Key("Spendable"), ui.Value(ui.Number(int64(info.Available))),
			ui.Key("Total cost"), ui.Value(fmt.Sprintf("%d (amount %d + fee %d)", amount+fee, amount, fee)),
			ui.Key("Left after send"), ui.Value(ui.Number(int64(max(change, 0)))),
		))
	}

	t.summary.Refresh()

	t.feedback.SetText(problem)
	if problem == "" {
		t.send.Enable()
	} else {
		t.send.Disable()
	}
}

func (t *TransactionUI) onSend() {
	from, to := t.from.Selected, t.to.Text
	amount, fee, problem := t.values()
	if problem != "" {
		t.state.Toast("Check the form", problem, state.LevelWarning)
		return
	}
	t.send.Disable()
	placeholder := ui.NewSkeleton(0, 72)
	t.pool.Objects = append([]fyne.CanvasObject{placeholder}, t.pool.Objects...)
	t.pool.Refresh()

	go func() {
		tx, err := t.state.Send(from, to, amount, fee)
		fyne.Do(func() {
			if err != nil {
				t.pool.Remove(placeholder)
				t.pool.Refresh()
				t.state.Toast("Transaction rejected", explain(err), state.LevelError)
				t.validate()
				return
			}
			t.amount.SetText("")
			t.state.Toast("Transaction signed", fmt.Sprintf("%d sent to the mempool, id %s", amount, ui.Short(tx.ID)), state.LevelSuccess)
		})
		t.state.Refresh()
	}()
}

func explain(err error) string {
	switch {
	case errors.Is(err, core.ErrPendingTransaction):
		return "Your wallet already has a pending transaction.\nPlease mine a block to confirm it before sending another."
	case errors.Is(err, core.ErrMempoolFull):
		return "The mempool is full. Please wait until some transactions are mined."
	case errors.Is(err, core.ErrOutputReserved):
		return "One of the coins is already being spent by a pending transaction."
	}
	return err.Error()
}
