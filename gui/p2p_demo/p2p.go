package p2p_demo

import (
	"Blockchain/core"
	"Blockchain/gui/ui"
	"Blockchain/resources"
	"Blockchain/resources/icons"
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type P2PNode struct {
	Name   string
	Blocks []*core.Block
	Miner  string
	busy   bool
	lane   *lane
}

type P2PDemoUI struct {
	window          fyne.Window
	mu              sync.Mutex
	nodes           []*P2PNode
	started         bool
	localTargetBits int

	content *fyne.Container
	intro   fyne.CanvasObject
	log     *fyne.Container
}

func New(window fyne.Window) *P2PDemoUI {
	p := &P2PDemoUI{window: window, localTargetBits: 16}
	p.log = container.NewVBox()
	p.intro = p.buildIntro()
	p.content = container.NewStack(p.intro)
	return p
}

func (p *P2PDemoUI) Content() fyne.CanvasObject {
	return p.content
}

func (p *P2PDemoUI) buildIntro() fyne.CanvasObject {
	start := ui.Primary("Start simulation", theme.MediaPlayIcon(), p.startDemo)

	points := container.NewVBox()
	for _, line := range []string{
		"Completely separate from main blockchain",
		"Demonstrates node synchronization",
		"Shows fork scenarios",
		"Stores data locally (no DB persistence)",
	} {
		points.Add(ui.NewBadge(line, resources.ColorNameMuted))
	}

	panel := ui.NewPanel(container.New(layout.NewCustomPaddedVBoxLayout(8),
		container.NewBorder(nil, nil, nil, ui.Icon(icons.ResourceP2pNetworkPng, 64),
			container.NewVBox(layout.NewSpacer(), ui.Title("P2P network simulation", 24), layout.NewSpacer()),
		),
		ui.Muted("Three nodes keep their own copy of a chain in memory. Mine on one node, create forks and "+
			"let the nodes agree on the longest valid chain."),
		ui.Padded(4, 8, 0, 0, points),
		ui.Divider(),
		container.NewBorder(nil, nil, nil, start),
	))
	return container.NewCenter(ui.FixedWidth(640, panel))
}

func (p *P2PDemoUI) startDemo() {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return
	}
	p.started = true
	p.mu.Unlock()

	intro := p.content.Objects
	p.content.Objects = []fyne.CanvasObject{container.NewBorder(nil, ui.Padded(16, 0, 0, 0, ui.NewSkeleton(0, 180)), nil, nil, ui.SkeletonRows(3, 112))}
	p.content.Refresh()

	go func() {
		genesis, err := p.newGenesis()
		if err != nil {
			fyne.Do(func() {
				p.mu.Lock()
				p.started = false
				p.mu.Unlock()
				p.content.Objects = intro
				p.content.Refresh()
				dialog.ShowError(fmt.Errorf("could not mine the shared genesis block: %w", err), p.window)
			})
			return
		}

		p.mu.Lock()
		for _, name := range []string{"A", "B", "C"} {
			p.nodes = append(p.nodes, &P2PNode{
				Name:   "Node " + name,
				Blocks: []*core.Block{genesis},
				Miner:  string(core.NewWallet().GetAddress()),
			})
		}
		p.mu.Unlock()

		fyne.Do(func() {
			p.content.Objects = []fyne.CanvasObject{p.buildNetwork()}
			p.content.Refresh()
			p.logf("P2P simulation initialized, all nodes share the same genesis block")
		})
	}()
}

func (p *P2PDemoUI) newGenesis() (*core.Block, error) {
	genesisTx := core.NewCoinbaseTX(string(core.NewWallet().GetAddress()), "Genesis block for P2P Demo", 10)
	return core.NewGenesisBlock(genesisTx, p.localTargetBits)
}

func (p *P2PDemoUI) buildNetwork() fyne.CanvasObject {
	bitsLabel := ui.Secondary("")
	bitsSlider := widget.NewSlider(8, 24)
	bitsSlider.Step = 1
	bitsSlider.OnChanged = func(val float64) {
		p.localTargetBits = int(val)
		bitsLabel.SetText(fmt.Sprintf("Mining difficulty (bits): %d", p.localTargetBits))
	}
	bitsSlider.SetValue(float64(p.localTargetBits))

	controls := ui.NewSurface(container.NewBorder(nil, nil,
		ui.Padded(0, 0, 0, 16, ui.FixedWidth(260, container.NewVBox(bitsLabel, bitsSlider))),
		nil,
		container.NewGridWithColumns(3,
			widget.NewButtonWithIcon("Sync B from A", theme.MailForwardIcon(), func() { p.syncNodes(0, 1) }),
			widget.NewButtonWithIcon("Sync C from A", theme.MailForwardIcon(), func() { p.syncNodes(0, 2) }),
			widget.NewButtonWithIcon("Sync C from B", theme.MailForwardIcon(), func() { p.syncNodes(1, 2) }),
			ui.Primary("Auto-sync", theme.ViewRefreshIcon(), p.autoSync),
			widget.NewButtonWithIcon("Sync A from C", theme.MailReplyIcon(), func() { p.syncNodes(2, 0) }),
			ui.Button("Reset", theme.DeleteIcon(), p.resetAll),
		),
	))

	lanes := container.New(layout.NewCustomPaddedVBoxLayout(12))
	for _, node := range p.nodes {
		node.lane = p.newLane(node)
		lanes.Add(node.lane.box)
	}
	p.render()

	console := ui.NewSurface(container.NewBorder(ui.Section("Network log", ""), nil, nil, nil, ui.Padded(8, 0, 0, 0, ui.NewNestedScroll(p.log))))

	page := ui.NewNestedScroll(lanes)
	for _, node := range p.nodes {
		node.lane.scroll.Parent = page
	}

	split := container.NewVSplit(page, console)
	split.Offset = 0.72

	return container.NewBorder(ui.Padded(0, 16, 0, 0, controls), nil, nil, nil, split)
}

func (p *P2PDemoUI) logf(format string, args ...any) {
	line := ui.Mono(time.Now().Format("15:04:05")+"  "+fmt.Sprintf(format, args...), resources.ColorNameMuted)
	p.log.Objects = append([]fyne.CanvasObject{line}, p.log.Objects...)
	if len(p.log.Objects) > 80 {
		p.log.Objects = p.log.Objects[:80]
	}
	p.log.Refresh()
}
