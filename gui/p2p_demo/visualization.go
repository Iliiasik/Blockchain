package p2p_demo

import (
	"Blockchain/core"
	"Blockchain/gui/ui"
	"Blockchain/resources"
	"bytes"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type lane struct {
	box    *ui.Surface
	led    *ui.LED
	info   *ui.Text
	chips  *fyne.Container
	scroll *ui.HStrip
	mine   *widget.Button
	fork   *widget.Button
}

func (p *P2PDemoUI) newLane(node *P2PNode) *lane {
	l := &lane{
		led:  ui.NewLED(16),
		info: ui.Caption(""),
	}
	l.mine = ui.Button("Mine", theme.ContentAddIcon(), func() { p.mineBlock(node) })
	l.fork = ui.Button("Fork", theme.WarningIcon(), func() { p.forkNode(node) })
	l.chips = container.New(layout.NewCustomPaddedHBoxLayout(8))
	l.scroll = ui.NewHStrip(l.chips)

	l.box = ui.NewSurface(container.NewVBox(
		container.NewBorder(nil, nil,
			container.NewHBox(container.NewCenter(l.led), container.NewCenter(ui.NewText(node.Name, 16, theme.ColorNameForeground)), ui.Padded(0, 0, 8, 0, container.NewCenter(l.info))),
			container.NewHBox(l.fork, l.mine),
		),
		ui.Padded(8, 0, 0, 0, l.scroll),
	))
	return l
}

func (p *P2PDemoUI) render() {
	p.mu.Lock()
	type view struct {
		node   *P2PNode
		blocks []*core.Block
		busy   bool
	}
	views := make([]view, len(p.nodes))
	var longest []*core.Block
	for i, node := range p.nodes {
		views[i] = view{node, append([]*core.Block(nil), node.Blocks...), node.busy}
		if len(node.Blocks) > len(longest) {
			longest = node.Blocks
		}
	}
	p.mu.Unlock()

	for _, v := range views {
		l := v.node.lane
		if l == nil {
			continue
		}

		if v.busy {
			l.led.SetState(ui.LEDBusy)
			l.mine.Disable()
			l.fork.Disable()
		} else {
			l.led.SetState(ui.LEDIdle)
			l.mine.Enable()
			l.fork.Enable()
		}

		tip := v.blocks[len(v.blocks)-1]
		status := "on the longest chain"
		if len(v.blocks) < len(longest) {
			status = fmt.Sprintf("%d blocks behind", len(longest)-len(v.blocks))
		}
		l.info.SetText(fmt.Sprintf("Length %d, tip %x, %s", len(v.blocks), tip.Hash[:4], status))

		l.chips.RemoveAll()
		for height, block := range v.blocks {
			l.chips.Add(p.chip(v.node.Name, block, height, longest))
		}
		l.chips.Refresh()
		l.scroll.ScrollToOffset(fyne.NewPos(l.chips.MinSize().Width, 0))
	}
}

func (p *P2PDemoUI) chip(nodeName string, block *core.Block, height int, longest []*core.Block) fyne.CanvasObject {
	label := fmt.Sprintf("Block #%d", height)
	if height == 0 {
		label = "Genesis"
	}

	var status fyne.CanvasObject = ui.Caption(fmt.Sprintf("%d bits", block.Bits))
	if height >= len(longest) || !bytes.Equal(longest[height].Hash, block.Hash) {
		status = ui.NewBadge("Fork", theme.ColorNameWarning)
	}

	content := container.NewVBox(
		ui.NewText(label, 14, theme.ColorNameForeground),
		ui.Mono(fmt.Sprintf("%x", block.Hash[:5]), resources.ColorNameSecondary),
		status,
	)

	card := ui.NewTapSurface(content, func() { p.showBlockDetails(block, nodeName, height) })
	return ui.FixedWidth(128, card)
}

func (p *P2PDemoUI) showBlockDetails(block *core.Block, nodeName string, height int) {
	scroll := container.NewVScroll(ui.BlockView(block, height, nil))
	scroll.SetMinSize(fyne.NewSize(820, 520))

	d := dialog.NewCustom(fmt.Sprintf("%s, block #%d", nodeName, height), "Close", scroll, p.window)
	d.Show()
}
