package visual

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"Blockchain/gui/ui"
	"Blockchain/resources"
	"fmt"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Visualizer struct {
	state  *state.AppState
	window fyne.Window
	snap   *state.Snapshot
	model  *model

	content  fyne.CanvasObject
	scene    *Scene
	card     *fyne.Container
	transfer *widget.Button
	mine     *widget.Button
	spin     *widget.Button
	stop     chan struct{}
}

func New(st *state.AppState, window fyne.Window) *Visualizer {
	v := &Visualizer{state: st, window: window, snap: st.Snapshot()}
	v.content = v.build()
	st.Subscribe(v.update)
	st.OnMiningChanged(v.miningChanged)
	return v
}

func (v *Visualizer) Content() fyne.CanvasObject {
	return v.content
}

func (v *Visualizer) SetActive(active bool) {
	v.scene.SetPaused(!active)
}

func (v *Visualizer) build() fyne.CanvasObject {
	v.scene = NewScene()
	v.scene.OnSelect = func(ref) { v.showCard() }

	v.transfer = ui.Button("Random transfer", theme.MailSendIcon(), v.onTransfer)
	v.mine = ui.Primary("Mine block", theme.MediaPlayIcon(), v.onMine)
	reset := ui.Flat("Reset view", theme.ZoomFitIcon(), v.scene.ResetView)
	v.spin = ui.Flat("Auto rotate", theme.ViewRefreshIcon(), v.toggleSpin)

	legend := container.NewHBox(
		ui.NewBadge("Block", resources.ColorNameMuted),
		ui.NewBadge("Tip", theme.ColorNamePrimary),
		ui.NewBadge("Coinbase", theme.ColorNameSuccess),
		ui.NewBadge("Transfer", resources.ColorNameAccent),
		ui.NewBadge("Pending", theme.ColorNameWarning),
	)

	toolbar := ui.NewSurface(container.NewBorder(nil, nil,
		container.NewCenter(legend),
		container.NewHBox(reset, v.spin, v.transfer, v.mine),
	))

	view := container.NewScroll(v.scene)
	view.Direction = container.ScrollNone

	v.card = container.NewStack()
	overlay := ui.Padded(12, 12, 12, 12, container.NewVBox(container.NewHBox(ui.FixedWidth(360, v.card), layout.NewSpacer())))

	stage := ui.NewSurface(container.NewStack(view, overlay))
	stage.Padding = 1

	check := func(label string, kind linkKind) *widget.Check {
		c := widget.NewCheck(label, func(on bool) { v.scene.SetLinkVisible(kind, on) })
		c.SetChecked(true)
		return c
	}
	links := container.NewHBox(
		container.NewCenter(ui.Caption("Links")),
		check("Hash", linkHash),
		check("Spending", linkSpend),
		check("Ownership", linkOwn),
	)

	hint := widget.NewLabel("Drag to rotate, Shift drag to pan, scroll to zoom")
	hint.SizeName = theme.SizeNameCaptionText
	hint.Importance = widget.LowImportance
	hint.Truncation = fyne.TextTruncateEllipsis

	how := ui.Button("How to read?", theme.QuestionIcon(), func() { showGuide(v.window) })

	bottom := ui.NewSurface(container.NewBorder(nil, nil,
		ui.Padded(0, 0, 0, 24, links),
		ui.Padded(0, 0, 32, 0, how),
		hint,
	))

	v.showCard()
	return container.NewBorder(ui.Padded(0, 12, 0, 0, toolbar), ui.Padded(12, 0, 0, 0, bottom), nil, nil, stage)
}

func (v *Visualizer) update(snap *state.Snapshot) {
	v.snap = snap
	v.model = fromSnapshot(snap)
	v.scene.SetModel(v.model, true)
	v.showCard()
	v.updateControls()
}

func (v *Visualizer) showCard() {
	if v.card == nil {
		return
	}
	selected := v.scene.selection()
	var title string
	var text []string
	if v.model != nil && selected.kind != refNone {
		title, text = v.model.explain(selected)
	}

	if title == "" {
		v.card.Objects = nil
		v.card.Refresh()
		return
	}

	body := container.NewVBox(container.NewBorder(nil, nil, nil, ui.Flat("", theme.CancelIcon(), func() { v.scene.Select(ref{}) }), container.NewVBox(layout.NewSpacer(), ui.Title(title, 16), layout.NewSpacer())))
	for _, line := range text {
		body.Add(ui.Paragraph(line))
	}
	if open := v.openAction(selected); open != nil {
		body.Add(ui.Padded(4, 0, 0, 0, container.NewHBox(open)))
	}

	v.card.Objects = []fyne.CanvasObject{ui.NewRaised(body)}
	v.card.Refresh()
}

func (v *Visualizer) openAction(r ref) fyne.CanvasObject {
	switch r.kind {
	case refNone:
		return nil
	case refBlock:
		return ui.Button("Open block", theme.SearchIcon(), func() { v.showBlock(r.height) })
	case refWallet:
		return ui.Button("Open wallet", theme.AccountIcon(), func() { v.state.Go("wallets", r.id) })
	case refTx:
		t, ok := v.model.txs[r.id]
		if !ok {
			return nil
		}
		if t.pending() {
			return ui.Button("Open mempool", theme.ListIcon(), func() { v.state.Go("transactions", "") })
		}
		return ui.Button("Open its block", theme.SearchIcon(), func() { v.showBlock(t.block) })
	}
	return nil
}

func (v *Visualizer) toggleSpin() {
	on := !v.scene.spin
	v.scene.SetSpin(on)
	if on {
		v.spin.SetText("Stop rotating")
		v.spin.SetIcon(theme.MediaPauseIcon())
	} else {
		v.spin.SetText("Auto rotate")
		v.spin.SetIcon(theme.ViewRefreshIcon())
	}
}

func (v *Visualizer) updateControls() {
	mining := v.state.IsMining()
	if v.snap.HasChain && len(v.snap.Wallets) > 0 && !mining {
		v.mine.Enable()
	} else {
		v.mine.Disable()
	}
	if v.snap.HasChain && len(v.snap.Wallets) > 1 {
		v.transfer.Enable()
	} else {
		v.transfer.Disable()
	}
}

func (v *Visualizer) miningChanged(active bool) {
	v.scene.SetMining(active)
	v.updateControls()

	if v.stop != nil {
		close(v.stop)
		v.stop = nil
	}
	if !active {
		return
	}

	v.stop = make(chan struct{})
	go func(stop chan struct{}) {
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				nonce, _, _ := v.state.Progress.Snapshot()
				fyne.Do(func() {
					if v.state.IsMining() {
						v.scene.SetNonce("nonce " + ui.Number(nonce))
					}
				})
			}
		}
	}(v.stop)
}

func (v *Visualizer) onMine() {
	miner := v.snap.Wallets[rand.Intn(len(v.snap.Wallets))].Address
	v.state.Mine(miner, v.snap.Mempool, func(block *core.Block, err error) {
		if err != nil {
			v.state.Toast("Mining failed", err.Error(), state.LevelError)
			return
		}
		v.state.Toast("Block mined", fmt.Sprintf("%d transactions, reward to %s", len(block.Transactions), ui.ShortAddress(miner)), state.LevelSuccess)
	})
}

func (v *Visualizer) onTransfer() {
	busy := make(map[string]bool)
	for _, tx := range v.snap.Mempool {
		busy[tx.GetSenderAddress()] = true
	}

	var senders []state.WalletInfo
	for _, w := range v.snap.Wallets {
		if w.Available > 1 && !busy[w.Address] {
			senders = append(senders, w)
		}
	}
	if len(senders) == 0 {
		v.state.Toast("No sender available", "Every wallet is empty or already has a pending transaction, mine a block first", state.LevelWarning)
		return
	}

	from := senders[rand.Intn(len(senders))]
	to := from.Address
	for to == from.Address {
		to = v.snap.Wallets[rand.Intn(len(v.snap.Wallets))].Address
	}
	fee := rand.Intn(min(3, from.Available-1))
	amount := 1 + rand.Intn(min(10, from.Available-fee))

	go func() {
		_, err := v.state.Send(from.Address, to, amount, fee)
		fyne.Do(func() {
			if err != nil {
				v.state.Toast("Transfer rejected", err.Error(), state.LevelError)
			}
		})
		v.state.Refresh()
	}()
}

func (v *Visualizer) showBlock(height int) {
	index := len(v.snap.Blocks) - 1 - height
	if index < 0 || index >= len(v.snap.Blocks) {
		return
	}

	scroll := container.NewVScroll(ui.BlockView(v.snap.Blocks[index], height, nil))
	scroll.SetMinSize(fyne.NewSize(820, 520))
	dialog.NewCustom("Block details", "Close", scroll, v.window).Show()
}
