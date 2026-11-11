package blockchain

import (
	"Blockchain/gui/state"
	"Blockchain/gui/ui"
	"Blockchain/resources"
	"Blockchain/resources/icons"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const genesisQuote = "September 30th, 1998. It's a day I'll never forget. The cop inside me died that day."

type BlockchainUI struct {
	window fyne.Window
	state  *state.AppState
	snap   *state.Snapshot

	setup    fyne.CanvasObject
	explorer fyne.CanvasObject
	content  *fyne.Container

	genesis       *widget.Select
	subsidySlider *widget.Slider
	bitsSlider    *widget.Slider
	create        *widget.Button
	activity      *widget.Activity
	working       *ui.Text

	height     *ui.Stat
	difficulty *ui.Stat
	txs        *ui.Stat
	age        *ui.Stat
	strip      *fyne.Container
	scroll     *ui.HStrip
	inspector  *fyne.Container
	cards      []*ui.Surface
	shownTip   []byte
}

func New(st *state.AppState, window fyne.Window) *BlockchainUI {
	b := &BlockchainUI{window: window, state: st, snap: st.Snapshot()}
	b.setup = b.buildSetup()
	b.explorer = b.buildExplorer()
	b.content = container.NewStack(b.setup, b.explorer)
	st.Subscribe(b.update)
	return b
}

func (b *BlockchainUI) Content() fyne.CanvasObject {
	return b.content
}

func (b *BlockchainUI) SetActive(active bool) {
	if active {
		b.syncParams()
	}
}

func (b *BlockchainUI) syncParams() {
	b.subsidySlider.SetValue(float64(b.state.Subsidy))
	b.bitsSlider.SetValue(float64(b.state.TargetBits))
}

func (b *BlockchainUI) Focus(address string) {
	if !b.snap.HasChain {
		b.genesis.SetSelected(address)
	}
}

func (b *BlockchainUI) buildSetup() fyne.CanvasObject {
	b.genesis = widget.NewSelect(nil, func(string) { b.updateCreate() })
	b.genesis.PlaceHolder = "Choose the wallet that receives the genesis reward"

	subsidyLabel := ui.NewText("", 13, theme.ColorNameForeground)
	subsidy := widget.NewSlider(1, 100)
	subsidy.Step = 1
	subsidy.OnChanged = func(v float64) {
		b.state.Subsidy = int(v)
		subsidyLabel.SetText(fmt.Sprintf("Block reward (subsidy): %d", b.state.Subsidy))
	}
	subsidy.SetValue(float64(b.state.Subsidy))
	b.subsidySlider = subsidy

	bitsLabel := ui.NewText("", 13, theme.ColorNameForeground)
	bitsHint := ui.NewText("", 11, resources.ColorNameMuted)
	bits := widget.NewSlider(8, 28)
	bits.Step = 1
	bits.OnChanged = func(v float64) {
		b.state.TargetBits = int(v)
		bitsLabel.SetText(fmt.Sprintf("Mining difficulty (bits): %d", b.state.TargetBits))
		if b.state.TargetBits > 20 {
			bitsHint.SetColor(theme.ColorNameWarning)
			bitsHint.SetText("Above 20 bits mining can take a very long time, 16 to 18 is recommended")
		} else {
			bitsHint.SetColor(resources.ColorNameMuted)
			bitsHint.SetText(fmt.Sprintf("About %s hashes on average", ui.Attempts(b.state.TargetBits)))
		}
	}
	bits.SetValue(float64(b.state.TargetBits))
	b.bitsSlider = bits

	b.activity = widget.NewActivity()
	b.activity.Hide()
	b.working = ui.Secondary("")
	b.create = ui.Primary("Mine genesis block", theme.ContentAddIcon(), b.onCreate)

	newWallet := ui.Flat("New wallet", theme.ContentAddIcon(), func() {
		address, err := b.state.CreateWallet()
		if err != nil {
			b.state.Toast("Wallet error", err.Error(), state.LevelError)
			return
		}
		b.state.Toast("Wallet created", address, state.LevelSuccess)
		b.genesis.SetOptions(append(b.genesis.Options, address))
		b.genesis.SetSelected(address)
		b.state.Refresh()
	})

	quote := ui.Muted("\"" + genesisQuote + "\"")

	panel := ui.NewPanel(container.New(layout.NewCustomPaddedVBoxLayout(8),
		container.NewBorder(nil, nil, nil, ui.Icon(icons.ResourceBlockchainPng, 56),
			container.NewVBox(layout.NewSpacer(), ui.Title("Create the blockchain", 24), layout.NewSpacer()),
		),
		ui.Muted("The genesis block is the first block of the chain. It has no previous hash, and its coinbase "+
			"transaction pays the block reward to the address you choose."),
		ui.Divider(),
		ui.Caption("Genesis address"),
		container.NewBorder(nil, nil, nil, newWallet, b.genesis),
		subsidyLabel, subsidy,
		bitsLabel, bits, bitsHint,
		ui.Caption("Coinbase data"),
		quote,
		ui.Divider(),
		container.NewBorder(nil, nil, container.NewHBox(b.activity, container.NewCenter(b.working)), b.create),
	))
	return container.NewCenter(ui.FixedWidth(640, panel))
}

func (b *BlockchainUI) updateCreate() {
	if b.genesis.Selected == "" || b.state.IsMining() || b.activity.Visible() {
		b.create.Disable()
	} else {
		b.create.Enable()
	}
}

func (b *BlockchainUI) onCreate() {
	address := b.genesis.Selected
	if address == "" {
		return
	}

	b.activity.Show()
	b.activity.Start()
	b.working.SetText(fmt.Sprintf("Mining the genesis block at %d bits", b.state.TargetBits))
	b.updateCreate()

	go func() {
		err := b.state.CreateBlockchain(address)
		fyne.Do(func() {
			b.activity.Stop()
			b.activity.Hide()
			b.working.SetText("")
			b.updateCreate()

			if err != nil {
				b.state.Toast("Genesis failed", err.Error(), state.LevelError)
				return
			}
			b.state.Toast("Blockchain created", fmt.Sprintf("Difficulty %d bits, subsidy %d", b.state.TargetBits, b.state.Subsidy), state.LevelSuccess)
		})
		b.state.Refresh()
	}()
}

func (b *BlockchainUI) buildExplorer() fyne.CanvasObject {
	b.height = ui.NewStat("Blocks")
	b.difficulty = ui.NewStat("Tip difficulty")
	b.txs = ui.NewStat("Transactions")
	b.age = ui.NewStat("Last block")

	b.strip = container.New(layout.NewCustomPaddedHBoxLayout(12))
	b.scroll = ui.NewHStrip(ui.Padded(0, 12, 0, 0, b.strip))

	reset := ui.Button("Reset", theme.DeleteIcon(), b.onReset)
	mine := ui.Primary("Mine next block", theme.MediaPlayIcon(), func() { b.state.Go("mining", "") })

	chain := ui.NewSurface(container.NewVBox(
		container.NewBorder(nil, nil, container.NewCenter(ui.Section("Chain", "")), container.NewHBox(reset, mine)),
		b.scroll,
	))

	b.inspector = container.NewStack()

	inspector := ui.NewSurface(b.inspector)
	inspector.Padding = 20

	return container.NewBorder(
		container.New(layout.NewCustomPaddedVBoxLayout(16),
			container.NewGridWithColumns(4, b.height.Box, b.difficulty.Box, b.txs.Box, b.age.Box),
			chain,
			layout.NewSpacer(),
		),
		nil, nil, nil,
		inspector,
	)
}

func (b *BlockchainUI) update(snap *state.Snapshot) {
	b.snap = snap

	if !snap.HasChain {
		b.explorer.Hide()
		b.setup.Show()
		b.syncParams()
		current := b.genesis.Selected
		b.genesis.SetOptions(snap.Addresses())
		if current != "" {
			b.genesis.SetSelected(current)
		} else if len(snap.Wallets) > 0 {
			b.genesis.SetSelected(snap.Wallets[0].Address)
		}
		b.updateCreate()
		b.shownTip = nil
		return
	}

	tip := snap.Tip()
	if tip == nil {
		return
	}

	b.setup.Hide()
	b.explorer.Show()

	b.height.Set(ui.Number(int64(snap.Height())), fmt.Sprintf("tip at height %d", snap.Height()-1))
	b.difficulty.Set(fmt.Sprintf("%d bits", tip.Bits), "about "+ui.Attempts(tip.Bits)+" hashes")
	b.txs.Set(ui.Number(int64(snap.TxCount)), "including coinbase")
	b.age.Set(ui.Ago(tip.Timestamp), ui.FormatTime(tip.Timestamp))

	if string(b.shownTip) == string(tip.Hash) && len(b.cards) == snap.Height() {
		return
	}
	b.shownTip = tip.Hash
	b.renderStrip()
	b.inspect(0)
}

func (b *BlockchainUI) onReset() {
	confirm := dialog.NewConfirm("Reset blockchain",
		"This deletes blockchain.db and mempool.dat. Wallets in wallet.dat are kept.",
		func(ok bool) {
			if !ok {
				return
			}
			if err := b.state.ResetBlockchain(); err != nil {
				b.state.Toast("Reset failed", err.Error(), state.LevelError)
				return
			}
			b.cards = nil
			b.state.Toast("Blockchain removed", "Create a new genesis block to start again", state.LevelWarning)
			b.state.Refresh()
		}, b.window)
	confirm.SetConfirmText("Delete chain")
	confirm.SetConfirmImportance(widget.DangerImportance)
	confirm.Show()
}
