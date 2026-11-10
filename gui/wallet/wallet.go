package wallet

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"Blockchain/gui/ui"
	"Blockchain/resources"
	"Blockchain/resources/icons"
	"encoding/hex"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type WalletUI struct {
	window   fyne.Window
	state    *state.AppState
	snap     *state.Snapshot
	selected string

	list    *widget.List
	count   *ui.Text
	detail  *fyne.Container
	content fyne.CanvasObject
}

func New(st *state.AppState, window fyne.Window) *WalletUI {
	w := &WalletUI{window: window, state: st, snap: st.Snapshot()}
	w.content = w.build()
	st.Subscribe(w.update)
	return w
}

func (w *WalletUI) Content() fyne.CanvasObject {
	return w.content
}

func (w *WalletUI) Focus(address string) {
	for i, info := range w.snap.Wallets {
		if info.Address == address {
			w.list.Select(i)
			w.list.ScrollTo(i)
			return
		}
	}
}

func (w *WalletUI) build() fyne.CanvasObject {
	w.count = ui.Caption("")

	w.list = widget.NewList(
		func() int { return len(w.snap.Wallets) },
		func() fyne.CanvasObject {
			return ui.Padded(4, 4, 4, 4, container.NewBorder(nil, nil, nil,
				ui.Padded(0, 0, 12, 0, container.NewHBox(container.NewCenter(ui.Strong("", 16)), container.NewCenter(ui.Icon(icons.ResourceCoinPng, 16)))),
				container.NewVBox(ui.Flush(ui.MonoValue("")), ui.Caption("")),
			))
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(w.snap.Wallets) {
				return
			}
			info := w.snap.Wallets[id]
			row := item.(*fyne.Container).Objects[0].(*fyne.Container)
			texts := row.Objects[0].(*fyne.Container)
			texts.Objects[0].(*fyne.Container).Objects[0].(*widget.Label).SetText(info.Address)

			sub := fmt.Sprintf("%d unspent outputs", info.Outputs)
			if !w.snap.HasChain {
				sub = "no blockchain yet"
			} else if info.Available != info.Balance {
				sub = fmt.Sprintf("%d locked in mempool", info.Balance-info.Available)
			}
			texts.Objects[1].(*ui.Text).SetText(sub)

			balance := row.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*ui.Text)
			balance.SetText(ui.Number(int64(info.Balance)))
		},
	)
	w.list.OnSelected = func(id widget.ListItemID) {
		if id < len(w.snap.Wallets) && w.snap.Wallets[id].Address != w.selected {
			w.selected = w.snap.Wallets[id].Address
			w.showDetail()
		}
	}

	create := ui.Primary("New wallet", theme.ContentAddIcon(), w.onCreateWallet)

	left := ui.NewSurface(container.NewBorder(
		container.NewVBox(
			container.NewBorder(nil, nil, container.NewCenter(ui.Section("Key store", "")), create),
			w.count,
			ui.Padded(8, 4, 0, 0, ui.Divider()),
		),
		nil, nil, nil,
		w.list,
	))

	w.detail = container.NewStack()
	w.showDetail()

	split := container.NewHSplit(left, w.detail)
	split.Offset = 0.4
	return split
}

func (w *WalletUI) update(snap *state.Snapshot) {
	w.snap = snap
	w.count.SetText(fmt.Sprintf("%d wallets in wallet.dat", len(snap.Wallets)))
	w.list.Refresh()

	if w.selected != "" {
		for i, info := range snap.Wallets {
			if info.Address == w.selected {
				w.list.Select(i)
				break
			}
		}
	}
	w.showDetail()
}

func (w *WalletUI) onCreateWallet() {
	address, err := w.state.CreateWallet()
	if err != nil {
		w.state.Toast("Wallet error", err.Error(), state.LevelError)
		return
	}

	w.selected = address
	w.state.Toast("Wallet created", address, state.LevelSuccess)
	w.state.Refresh()
}

func (w *WalletUI) copied(string) {
	w.state.Toast("Copied", "Value copied to clipboard", state.LevelInfo)
}

func (w *WalletUI) showDetail() {
	info, ok := w.snap.Wallet(w.selected)
	if !ok {
		w.detail.Objects = []fyne.CanvasObject{ui.NewSurface(ui.Empty(icons.ResourceWalletsPng,
			"No wallet selected", "Create a wallet or pick one from the key store"))}
		w.detail.Refresh()
		return
	}

	raw := core.Base58Decode([]byte(info.Address))

	balance := ui.NewStat("Balance")
	balance.Set(ui.Number(int64(info.Balance)), "confirmed coins")
	available := ui.NewStat("Spendable")
	available.Set(ui.Number(int64(info.Available)), "not locked by pending tx")
	outputs := ui.NewStat("Outputs")
	outputs.Set(ui.Number(int64(info.Outputs)), "unspent outputs")

	actions := container.NewHBox(
		ui.Primary("Send", theme.MailSendIcon(), func() { w.state.Go("transactions", info.Address) }),
		ui.Button("Mine to wallet", theme.MediaPlayIcon(), func() { w.state.Go("mining", info.Address) }),
		ui.Button("Outputs", theme.SearchIcon(), func() { w.state.Go("utxo", info.Address) }),
	)
	if !w.snap.HasChain {
		actions.Add(ui.Button("Use as genesis", theme.ContentAddIcon(), func() { w.state.Go("blockchain", info.Address) }))
	}

	header := ui.NewPanel(container.NewVBox(
		ui.Caption("Address"),
		ui.Copyable(info.Address, w.copied),
		ui.KV(
			ui.Key("Version"), ui.MonoValue(fmt.Sprintf("0x%02x", raw[0])),
			ui.Key("Public key hash"), ui.Copyable(hex.EncodeToString(info.PubKeyHash), w.copied),
			ui.Key("Checksum"), ui.MonoValue(hex.EncodeToString(raw[len(raw)-4:])),
		),
		ui.Padded(8, 0, 0, 0, actions),
	))

	entries := container.New(layout.NewCustomPaddedVBoxLayout(8), ui.SkeletonRows(3, 56))
	historyPanel := ui.NewSurface(container.NewVBox(ui.Section("History", ""), entries))

	w.detail.Objects = []fyne.CanvasObject{ui.NewNestedScroll(container.New(layout.NewCustomPaddedVBoxLayout(16),
		header,
		container.NewGridWithColumns(3, balance.Box, available.Box, outputs.Box),
		historyPanel,
	))}
	w.detail.Refresh()

	snap := w.snap
	go func() {
		list := history(snap.Blocks, snap.Mempool, info.PubKeyHash)
		fyne.Do(func() {
			if w.selected != info.Address {
				return
			}
			entries.RemoveAll()
			if len(list) == 0 {
				entries.Add(ui.Muted("No transactions yet. Mine a block to this wallet or receive coins."))
				entries.Refresh()
				return
			}
			for _, e := range list {
				entries.Add(w.entryRow(e))
			}
			entries.Refresh()
		})
	}()
}

func (w *WalletUI) entryRow(e entry) fyne.CanvasObject {
	kind := resources.ColorNameAccent
	amountColor := theme.ColorNameSuccess
	amount := fmt.Sprintf("+%s", ui.Number(int64(e.Amount)))
	label := map[string]string{"mined": "Mined", "sent": "Sent", "received": "Received"}[e.Kind]
	switch e.Kind {
	case "mined":
		kind = theme.ColorNameSuccess
	case "sent":
		kind = theme.ColorNameError
		amountColor = theme.ColorNameForeground
		amount = ui.Number(int64(e.Amount))
	}

	when := fmt.Sprintf("Block #%d, %s", e.Height, ui.FormatTime(e.Time))
	if e.Pending {
		when = "Waiting in the mempool"
		kind = theme.ColorNameWarning
		label += ", pending"
	}
	badges := container.NewHBox(ui.NewBadge(label, kind))

	counterparty := e.Counterparty
	if len(counterparty) > 20 {
		counterparty = ui.ShortAddress(counterparty)
	}

	fee := ""
	if e.Fee > 0 && e.Kind == "sent" {
		fee = fmt.Sprintf("Fee %d", e.Fee)
	}

	return ui.NewRaised(container.NewBorder(nil, nil,
		ui.FixedWidth(140, container.NewCenter(container.NewHBox(badges))),
		container.NewVBox(ui.NewText(amount, 16, amountColor), ui.Caption(fee)),
		ui.Padded(0, 0, 12, 12, container.NewVBox(
			ui.Mono(ui.Short(e.TxID)+"   "+counterparty, theme.ColorNameForeground),
			ui.Caption(when),
		)),
	))
}
