package p2p_demo

import (
	"Blockchain/core"
	"Blockchain/resources/icons"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"sync"
)

type P2PDemoUI struct {
	window             fyne.Window
	nodes              map[string]*P2PNode
	nodeVisualizations *fyne.Container
	controls           *fyne.Container
	content            *fyne.Container
	loadingSpinner     *widget.ProgressBarInfinite
	mu                 sync.Mutex
	started            bool
	localTargetBits    int
}

type P2PNode struct {
	Name     string
	Blocks   []*core.Block
	mu       sync.Mutex
	selected bool
}

func NewP2PDemoTab(window fyne.Window) *container.TabItem {
	demo := &P2PDemoUI{
		window:          window,
		nodes:           make(map[string]*P2PNode),
		localTargetBits: 16,
	}

	demo.loadingSpinner = widget.NewProgressBarInfinite()
	demo.loadingSpinner.Hide()

	startBtn := widget.NewButtonWithIcon("Start simulation", theme.MediaPlayIcon(), func() {
		demo.startDemo()
	})
	startBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(
			container.NewVBox(
				container.NewCenter(startBtn),
				container.NewCenter(demo.loadingSpinner),
			),
		),
		layout.NewSpacer(),
	)

	demo.content = content

	return container.NewTabItemWithIcon("P2P simulation", icons.ResourceP2pNetworkPng, demo.content)
}

func (p *P2PDemoUI) startDemo() {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return
	}
	p.started = true
	p.mu.Unlock()

	fyne.Do(func() {
		p.loadingSpinner.Show()
	})

	go func() {
		genesisAddress := "1P2PDemoGenesisAddress"
		genesisTx := core.NewCoinbaseTX(genesisAddress, "Genesis block for P2P Demo", 16)
		genesisBlock := core.NewGenesisBlock(genesisTx, p.localTargetBits)

		fyne.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()

			p.nodes["A"] = &P2PNode{
				Name:   "Node A",
				Blocks: []*core.Block{genesisBlock},
			}
			p.nodes["B"] = &P2PNode{
				Name:   "Node B",
				Blocks: []*core.Block{genesisBlock},
			}
			p.nodes["C"] = &P2PNode{
				Name:   "Node C",
				Blocks: []*core.Block{genesisBlock},
			}

			bitsLabel := widget.NewLabel(fmt.Sprintf("Mining difficulty (bits): %d", p.localTargetBits))
			bitsSlider := widget.NewSlider(8, 28)
			bitsSlider.SetValue(float64(p.localTargetBits))
			bitsSlider.Step = 1

			bitsSlider.OnChanged = func(val float64) {
				intVal := int(val)
				p.localTargetBits = intVal
				bitsLabel.SetText(fmt.Sprintf("Mining difficulty (bits): %d", intVal))
			}

			bitsSlider.OnChangeEnded = func(val float64) {
				intVal := int(val)
				if intVal > 20 {
					dialog.ShowInformation("Warning",
						"Mining with difficulty > 20 may take very long time!\nRecommended: 16-18",
						p.window)
				}
			}

			buttonsRow1 := container.NewGridWithColumns(3,
				widget.NewButtonWithIcon("Mine A", theme.ContentAddIcon(), func() { p.mineBlock("A") }),
				widget.NewButtonWithIcon("Mine B", theme.ContentAddIcon(), func() { p.mineBlock("B") }),
				widget.NewButtonWithIcon("Mine C", theme.ContentAddIcon(), func() { p.mineBlock("C") }),
			)

			buttonsRow2 := container.NewGridWithColumns(3,
				widget.NewButtonWithIcon("Sync B ← A", theme.MailForwardIcon(), func() { p.syncNodes("A", "B") }),
				widget.NewButtonWithIcon("Sync C ← A", theme.MailForwardIcon(), func() { p.syncNodes("A", "C") }),
				widget.NewButtonWithIcon("Sync C ← B", theme.MailForwardIcon(), func() { p.syncNodes("B", "C") }),
			)

			buttonsRow3 := container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Fork C", theme.WarningIcon(), func() { p.forkNode("C") }),
				widget.NewButtonWithIcon("Auto-sync", theme.ViewRefreshIcon(), func() { p.autoSync() }),
			)

			buttonsRow4 := container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Reset", theme.DeleteIcon(), func() { p.resetAll() }),
				widget.NewButtonWithIcon("Refresh", theme.VisibilityIcon(), func() { p.updateVisualization() }),
			)

			p.controls = container.NewVBox(
				widget.NewLabelWithStyle("P2P network demo", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewSeparator(),
				container.NewVBox(
					bitsLabel,
					bitsSlider,
				),
				widget.NewSeparator(),
				buttonsRow1,
				buttonsRow2,
				buttonsRow3,
				buttonsRow4,
			)

			p.nodeVisualizations = container.NewVBox()
			for _, nodeName := range []string{"A", "B", "C"} {
				nodeViz := p.createNodeVisualization(p.nodes[nodeName])
				p.nodeVisualizations.Add(nodeViz)
				p.nodeVisualizations.Add(widget.NewSeparator())
			}

			scroll := container.NewScroll(p.nodeVisualizations)
			scroll.SetMinSize(fyne.NewSize(800, 400))

			mainContent := container.NewBorder(
				container.NewVBox(
					p.loadingSpinner,
					widget.NewSeparator(),
					p.controls,
				),
				nil,
				nil,
				nil,
				scroll,
			)

			p.content.RemoveAll()
			p.content.Add(mainContent)

			p.loadingSpinner.Hide()
			p.showNotification("P2P simulation initialized successfully.\n\n" +
				"This is an isolated P2P simulation environment.\n\n" +
				"• Completely separate from main blockchain\n" +
				"• Demonstrates node synchronization\n" +
				"• Shows fork scenarios\n" +
				"• Stores data locally (no DB persistence)\n")
		})
	}()
}
