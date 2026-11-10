package about

import (
	"Blockchain/gui/state"
	"Blockchain/gui/ui"
	"Blockchain/resources"
	"Blockchain/resources/icons"
	"fmt"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type objective struct {
	title  string
	text   string
	screen string
	status *ui.Badge
	label  *ui.Text
}

type Overview struct {
	state      *state.AppState
	content    fyne.CanvasObject
	blocks     *ui.Stat
	supply     *ui.Stat
	mempool    *ui.Stat
	difficulty *ui.Stat
	objectives []*objective
	recent     *fyne.Container
}

func New(st *state.AppState) *Overview {
	o := &Overview{state: st}
	o.content = o.build()
	st.Subscribe(o.update)
	return o
}

func (o *Overview) Content() fyne.CanvasObject {
	return o.content
}

func (o *Overview) build() fyne.CanvasObject {
	u := &url.URL{Scheme: "https", Host: "github.com", Path: "/Iliiasik/Blockchain"}

	hero := ui.NewPanel(container.NewBorder(nil, nil, nil,
		ui.Padded(0, 0, 24, 0, container.NewCenter(ui.Icon(icons.ResourceAboutPng, 96))),
		container.New(layout.NewCustomPaddedVBoxLayout(12),
			ui.Title("Blockchain", 30),
			ui.Muted("A graphical blockchain simulator built with Go and Fyne. It visualizes core concepts like "+
				"Proof-of-Work, UTXO-based transactions, mining, mempool, wallets and block structure in an "+
				"interactive and educational way."),
			ui.Padded(8, 0, 0, 0, container.NewHBox(
				ui.Flush(widget.NewHyperlink("If you found this project helpful, give it a star on GitHub", u)),
				layout.NewSpacer(),
			)),
		),
	))

	o.blocks = ui.NewStat("Blocks")
	o.supply = ui.NewStat("Coin supply")
	o.mempool = ui.NewStat("Mempool")
	o.difficulty = ui.NewStat("Difficulty")

	stats := container.NewGridWithColumns(4, o.blocks.Box, o.supply.Box, o.mempool.Box, o.difficulty.Box)

	o.objectives = []*objective{
		{title: "Create a wallet", text: "Generate an ECDSA key pair and its Base58Check address", screen: "wallets"},
		{title: "Create the blockchain", text: "Mine the genesis block and pay its reward to your wallet", screen: "blockchain"},
		{title: "Mine a block", text: "Search for a nonce that satisfies the target", screen: "mining"},
		{title: "Send a transaction", text: "Sign inputs, pay a fee and wait in the mempool", screen: "transactions"},
		{title: "Confirm it", text: "Mine the pending transaction into the next block", screen: "mining"},
	}

	steps := container.New(layout.NewCustomPaddedVBoxLayout(12))
	for i, obj := range o.objectives {
		obj := obj
		obj.status = ui.NewBadge("", resources.ColorNameMuted)
		obj.label = ui.NewText(fmt.Sprintf("%d. %s", i+1, obj.title), 14, theme.ColorNameForeground)
		open := ui.Flat("Open", theme.NavigateNextIcon(), func() { o.state.Go(obj.screen, "") })
		open.IconPlacement = widget.ButtonIconTrailingText
		steps.Add(container.NewBorder(nil, nil,
			container.NewCenter(obj.status),
			container.NewCenter(open),
			container.NewVBox(obj.label, ui.Caption(obj.text)),
		))
	}

	guide := ui.NewSurface(container.NewVBox(
		ui.Section("Getting started", ""),
		ui.Padded(8, 0, 0, 0, steps),
	))
	guide.Padding = 20

	o.recent = container.New(layout.NewCustomPaddedVBoxLayout(8))
	recent := ui.NewSurface(container.NewBorder(
		container.NewBorder(nil, nil, container.NewCenter(ui.Section("Latest blocks", "")),
			ui.Flat("Explorer", theme.NavigateNextIcon(), func() { o.state.Go("blockchain", "") })),
		nil, nil, nil,
		ui.Padded(8, 0, 0, 0, o.recent),
	))
	recent.Padding = 20

	features := []struct{ title, desc string }{
		{"Wallet generation", "ECDSA keys with secure storage"},
		{"Transaction creation", "UTXO model implementation"},
		{"Block mining", "Proof-of-Work simulation"},
		{"Chain visualization", "Interactive blockchain explorer"},
		{"Custom parameters", "Adjustable subsidy and targetBits"},
		{"Transaction history", "Per-wallet TX log with UTXO tracking"},
		{"P2P simulation", "Basic peer-to-peer block propagation"},
	}
	featureList := container.NewGridWithColumns(3)
	for _, f := range features {
		featureList.Add(ui.Padded(0, 8, 0, 8, container.NewVBox(ui.NewText(f.title, 14, theme.ColorNameForeground), ui.Caption(f.desc))))
	}

	about := ui.NewSurface(container.NewVBox(
		ui.Section("Key features", ""),
		ui.Padded(8, 8, 0, 0, featureList),
		ui.Divider(),
		ui.Padded(8, 0, 0, 0, ui.Section("Project purpose", "This is an open-source learning resource. Feel free to study the code, report issues and contribute improvements. "+
			"You are welcome to use this project and enhance it for your own implementations. "+
			"You can find theoretical information in the Wiki tab and the technical implementation on GitHub.")),
	))
	about.Padding = 20

	return ui.NewNestedScroll(container.New(layout.NewCustomPaddedVBoxLayout(16),
		hero,
		stats,
		container.NewBorder(nil, nil, nil, ui.FixedWidth(440, recent), guide),
		about,
	))
}

func (o *Overview) update(snap *state.Snapshot) {
	o.blocks.Set(ui.Number(int64(snap.Height())), "in the local chain")
	o.supply.Set(ui.Number(int64(snap.Supply)), "coins in unspent outputs")
	o.mempool.Set(fmt.Sprintf("%d/%d", len(snap.Mempool), snap.MempoolMax), "pending transactions")

	if tip := snap.Tip(); tip != nil {
		o.difficulty.Set(fmt.Sprintf("%d bits", tip.Bits), "about "+ui.Attempts(tip.Bits)+" hashes per block")
	} else {
		o.difficulty.Set("None", "create the blockchain first")
	}

	transfers := len(snap.Mempool) > 0
	confirmed := false
	for _, block := range snap.Blocks {
		if len(block.Transactions) > 1 {
			confirmed = true
			transfers = true
		}
	}

	done := []bool{len(snap.Wallets) > 0, snap.HasChain, snap.Height() > 1, transfers, confirmed}
	current := true
	for i, obj := range o.objectives {
		switch {
		case done[i]:
			obj.status.Set("", theme.ColorNameSuccess)
			obj.label.SetColor(resources.ColorNameSecondary)
		case current:
			obj.status.Set("", theme.ColorNamePrimary)
			obj.label.SetColor(theme.ColorNameForeground)
			current = false
		default:
			obj.status.Set("", theme.ColorNameDisabled)
			obj.label.SetColor(resources.ColorNameMuted)
		}
	}

	o.recent.RemoveAll()
	if len(snap.Blocks) == 0 {
		o.recent.Add(ui.Muted("The chain is empty. Create the genesis block to start."))
		o.recent.Refresh()
		return
	}
	for i, block := range snap.Blocks {
		if i == 6 {
			break
		}
		o.recent.Add(container.NewBorder(nil, nil,
			ui.FixedWidth(40, ui.Mono(fmt.Sprintf("#%d", snap.Height()-1-i), resources.ColorNameMuted)),
			ui.Caption(fmt.Sprintf("%d tx, %s", len(block.Transactions), ui.Ago(block.Timestamp))),
			ui.Mono(ui.Short(block.Hash), theme.ColorNameForeground),
		))
	}
	o.recent.Refresh()
}
