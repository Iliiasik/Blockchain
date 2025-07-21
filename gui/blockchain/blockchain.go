package blockchain

import (
	"Blockchain/core"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"strconv"
)

type BlockchainUI struct {
	window        fyne.Window
	visualization *fyne.Container
	blocks        []*core.Block
}

func NewBlockchainTab(window fyne.Window) *container.TabItem {
	ui := &BlockchainUI{
		window:        window,
		visualization: container.NewHBox(),
	}
	return ui.createTab()
}

func (b *BlockchainUI) createTab() *container.TabItem {
	createAddress := widget.NewEntry()
	createAddress.SetPlaceHolder("Genesis address")

	controls := container.NewVBox(
		widget.NewLabelWithStyle("Blockchain controls", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			widget.NewButtonWithIcon("Create blockchain", theme.ContentAddIcon(), func() {
				b.onCreateBlockchain(createAddress.Text)
			}),
			widget.NewButtonWithIcon("Reindex UTXO", theme.ViewRefreshIcon(), b.onReindexUTXO),
		),
		createAddress,
		widget.NewButton("View blockchain", b.updateVisualization),
	)

	visualizationTitle := widget.NewLabelWithStyle("Blockchain visualization", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	scroll := container.NewScroll(b.visualization)
	scroll.SetMinSize(fyne.NewSize(800, 120))

	return container.NewTabItem("Blockchain",
		container.NewBorder(
			controls,
			nil,
			nil,
			nil,
			container.NewVBox(
				visualizationTitle,
				widget.NewSeparator(),
				scroll,
			),
		),
	)
}

func (b *BlockchainUI) createBlockUI(block *core.Block) fyne.CanvasObject {
	blockColor := color.NRGBA{R: 70, G: 130, B: 180, A: 255}
	if len(block.PrevBlockHash) == 0 {
		blockColor = color.NRGBA{R: 34, G: 139, B: 34, A: 255}
	}

	label := canvas.NewText(fmt.Sprintf("%x", block.Hash[:6]), color.White)
	label.TextSize = 12
	label.Alignment = fyne.TextAlignCenter

	rect := canvas.NewRectangle(blockColor)
	rect.SetMinSize(fyne.NewSize(80, 80))

	blockSquare := container.NewMax(
		rect,
		container.NewCenter(label),
	)

	infoIcon := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		b.showBlockDetails(block)
	})
	infoIcon.Importance = widget.LowImportance

	return container.NewHBox(
		blockSquare,
		container.NewCenter(infoIcon),
	)
}

func (b *BlockchainUI) showBlockDetails(block *core.Block) {
	var details string

	details += fmt.Sprintf("=== Block %x ===\n\n", block.Hash)
	details += fmt.Sprintf("• Previous hash: %x\n", block.PrevBlockHash)
	details += fmt.Sprintf("• Transactions: %d\n", len(block.Transactions))
	pow := core.NewProofOfWork(block)
	details += fmt.Sprintf("• PoW valid: %t\n\n", pow.Validate())

	for i, tx := range block.Transactions {
		details += fmt.Sprintf("Transaction %d: %x\n", i+1, tx.ID[:8])
		if tx.IsCoinbase() {
			details += "  Type: Coinbase\n"
		} else {
			details += "  Type: Regular\n"
		}
		details += fmt.Sprintf("  Inputs: %d, Outputs: %d\n\n", len(tx.Vin), len(tx.Vout))
	}

	content := widget.NewLabel(details)
	content.TextStyle = fyne.TextStyle{Monospace: true}

	scroll := container.NewScroll(content)
	scroll.SetMinSize(fyne.NewSize(600, 400))

	dialog.ShowCustom(
		fmt.Sprintf("Block %x Details", block.Hash[:8]),
		"Close",
		scroll,
		b.window,
	)
}

func (b *BlockchainUI) updateVisualization() {
	bc, err := core.NewBlockchain()
	if err != nil {
		dialog.ShowError(err, b.window)
		return
	}
	defer bc.Db.Close()

	b.visualization.Objects = nil
	b.blocks = nil

	bci := bc.Iterator()
	for {
		block := bci.Next()
		b.blocks = append(b.blocks, block)
		b.visualization.Add(b.createBlockUI(block))

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	b.visualization.Refresh()
}

func (b *BlockchainUI) onReindexUTXO() {
	bc, err := core.NewBlockchain()
	if err != nil {
		dialog.ShowError(err, b.window)
		return
	}
	defer bc.Db.Close()

	UTXOSet := core.UTXOSet{bc}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	dialog.ShowInformation("UTXO Reindex",
		"Reindex complete!\nTransactions: "+strconv.Itoa(count),
		b.window)
}

func (b *BlockchainUI) onCreateBlockchain(address string) {
	if address == "" {
		dialog.ShowError(fmt.Errorf("Please enter a genesis address"), b.window)
		return
	}

	if !core.ValidateAddress(address) {
		dialog.ShowError(fmt.Errorf("Invalid address format"), b.window)
		return
	}

	bc, err := core.CreateBlockchain(address)
	if err != nil {
		dialog.ShowError(err, b.window)
		return
	}
	defer bc.Db.Close()

	UTXOSet := core.UTXOSet{bc}
	UTXOSet.Reindex()

	dialog.ShowInformation("Blockchain created",
		"New blockchain with genesis block created!",
		b.window)
}
