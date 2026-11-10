package blockchain

import (
	"Blockchain/gui/state"
	"Blockchain/gui/ui"
	"Blockchain/resources"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

func (b *BlockchainUI) renderStrip() {
	blocks := b.snap.Blocks
	b.strip.RemoveAll()
	b.cards = make([]*ui.Surface, len(blocks))

	for pos := len(blocks) - 1; pos >= 0; pos-- {
		index := pos
		block := blocks[index]
		height := len(blocks) - 1 - index

		label := fmt.Sprintf("Block #%d", height)
		if len(block.PrevBlockHash) == 0 {
			label = "Genesis"
		}

		card := ui.NewTapSurface(container.NewVBox(
			ui.NewText(label, 14, theme.ColorNameForeground),
			ui.Mono(ui.ShortString(fmt.Sprintf("%x", block.Hash), 6, 6), resources.ColorNameSecondary),
			ui.Caption(fmt.Sprintf("%d tx, %s", len(block.Transactions), ui.Ago(block.Timestamp))),
		), func() { b.inspect(index) })
		b.cards[index] = card

		b.strip.Add(ui.FixedWidth(160, card))
	}

	b.strip.Refresh()
	b.scroll.ScrollToOffset(fyne.NewPos(b.strip.MinSize().Width, 0))
}

func (b *BlockchainUI) inspect(index int) {
	if index < 0 || index >= len(b.snap.Blocks) {
		return
	}

	for i, card := range b.cards {
		card.SetSelected(i == index)
	}

	block := b.snap.Blocks[index]
	height := len(b.snap.Blocks) - 1 - index
	notify := func(string) {
		b.state.Toast("Copied", "Value copied to clipboard", state.LevelInfo)
	}

	b.inspector.Objects = []fyne.CanvasObject{ui.NewNestedScroll(ui.BlockView(block, height, notify))}
	b.inspector.Refresh()
}
