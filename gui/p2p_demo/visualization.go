package p2p_demo

import (
	"Blockchain/core"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"time"
)

func (p *P2PDemoUI) createNodeVisualization(node *P2PNode) *fyne.Container {
	title := widget.NewLabelWithStyle(node.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	if node.selected {
		title.TextStyle.Bold = true
		title.TextStyle.Italic = true
	}

	blocksContainer := container.NewHBox()
	for _, block := range node.Blocks {
		blocksContainer.Add(p.createBlockUI(block, node.Name))
	}

	chainInfo := widget.NewLabel(fmt.Sprintf("Chain length: %d blocks", len(node.Blocks)))
	chainInfo.Alignment = fyne.TextAlignTrailing

	return container.NewVBox(
		container.NewHBox(
			title,
			layout.NewSpacer(),
			chainInfo,
		),
		widget.NewSeparator(),
		container.NewHScroll(blocksContainer),
	)
}

func (p *P2PDemoUI) createBlockUI(block *core.Block, nodeName string) fyne.CanvasObject {
	baseColor := color.NRGBA{R: 70, G: 130, B: 180, A: 220}
	if len(block.PrevBlockHash) == 0 {
		baseColor = color.NRGBA{R: 34, G: 139, B: 34, A: 220}
	}

	blockHash := canvas.NewText(fmt.Sprintf("%x", block.Hash[:8]), color.White)
	blockHash.Alignment = fyne.TextAlignCenter
	blockHash.TextSize = 10

	blockContainer := container.NewVBox(
		widget.NewLabel("Block"),
		blockHash,
	)

	rect := canvas.NewRectangle(baseColor)
	rect.SetMinSize(fyne.NewSize(120, 100))
	rect.CornerRadius = 5

	infoBtn := widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
		p.showBlockDetails(block, nodeName)
	})
	infoBtn.Importance = widget.LowImportance
	infoBtn.Resize(fyne.NewSize(24, 24))
	infoBtn.Move(fyne.NewPos(85, 11))

	return container.NewStack(
		rect,
		container.NewPadded(blockContainer),
		container.NewWithoutLayout(infoBtn),
	)
}

func (p *P2PDemoUI) showBlockDetails(block *core.Block, nodeName string) {
	headerText := canvas.NewText(fmt.Sprintf("%s: Block %x", nodeName, block.Hash), color.NRGBA{R: 0, G: 102, B: 204, A: 255})
	headerText.TextSize = 18
	headerText.TextStyle.Bold = true
	headerText.Alignment = fyne.TextAlignCenter
	header := container.NewHBox(
		layout.NewSpacer(),
		headerText,
		layout.NewSpacer(),
	)

	dataColor := color.NRGBA{R: 0, G: 102, B: 204, A: 255}

	createRow := func(label, value string) *fyne.Container {
		return container.NewHBox(
			widget.NewLabel(label),
			canvas.NewText(value, dataColor),
		)
	}

	blockInfo := container.NewVBox(
		createRow("Date: ", time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05")),
		createRow("Previous hash: ", fmt.Sprintf("%x", block.PrevBlockHash)),
		createRow("Transactions count: ", fmt.Sprintf("%d", len(block.Transactions))),
		container.NewHBox(
			widget.NewLabel("PoW valid: "),
			func() fyne.CanvasObject {
				if core.NewProofOfWork(block, block.Bits).Validate() {
					return container.NewHBox(
						widget.NewIcon(theme.ConfirmIcon()),
						canvas.NewText("true", color.NRGBA{R: 0, G: 200, B: 0, A: 255}),
					)
				}
				return container.NewHBox(
					widget.NewIcon(theme.ErrorIcon()),
					canvas.NewText("false", color.NRGBA{R: 200, G: 0, B: 0, A: 255}),
				)
			}(),
		),
		createRow("Difficulty (bits): ", fmt.Sprintf("%d", block.Bits)),
	)

	txAccordion := widget.NewAccordion()
	for i, tx := range block.Transactions {
		txType := "Regular"
		txIcon := theme.FileIcon()
		if tx.IsCoinbase() {
			txType = "Coinbase"
			txIcon = theme.MediaPlayIcon()
		}

		basicInfo := container.NewVBox(
			container.NewHBox(
				widget.NewIcon(txIcon),
				createRow("Type: ", txType),
			),
			createRow("Inputs count: ", fmt.Sprintf("%d", len(tx.Vin))),
			createRow("Outputs count: ", fmt.Sprintf("%d", len(tx.Vout))),
		)

		inputsContainer := container.NewVBox()
		for _, input := range tx.Vin {
			inputsContainer.Add(createRow("TxID: ", fmt.Sprintf("%x", input.Txid)))
			inputsContainer.Add(createRow("OutIdx: ", fmt.Sprintf("%d", input.Vout)))
			inputsContainer.Add(widget.NewSeparator())
		}

		outputsContainer := container.NewVBox()
		for _, output := range tx.Vout {
			outputsContainer.Add(createRow("Value: ", fmt.Sprintf("%d", output.Value)))
			outputsContainer.Add(createRow("To: ", fmt.Sprintf("%x", output.PubKeyHash)))
			outputsContainer.Add(widget.NewSeparator())
		}

		tabs := container.NewAppTabs(
			container.NewTabItemWithIcon("Basic", theme.DocumentIcon(), basicInfo),
			container.NewTabItemWithIcon("Inputs", theme.ListIcon(), container.NewScroll(inputsContainer)),
			container.NewTabItemWithIcon("Outputs", theme.ListIcon(), container.NewScroll(outputsContainer)),
		)
		tabs.SetTabLocation(container.TabLocationTop)

		txAccordion.Append(widget.NewAccordionItem(
			fmt.Sprintf("Transaction %d: %x...", i+1, tx.ID[:18]),
			tabs,
		))
	}

	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		container.NewPadded(blockInfo),
		widget.NewSeparator(),
		container.NewHBox(
			widget.NewIcon(theme.ListIcon()),
			widget.NewLabel("Transactions:"),
		),
		container.NewPadded(txAccordion),
	)

	scroll := container.NewScroll(content)
	scroll.SetMinSize(fyne.NewSize(800, 450))

	d := dialog.NewCustom(
		"Block details",
		"Close",
		scroll,
		p.window,
	)
	d.Resize(fyne.NewSize(940, 450))
	d.Show()
}

func (p *P2PDemoUI) updateVisualization() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	fyne.Do(func() {
		p.nodeVisualizations.Objects = nil
		for _, nodeName := range []string{"A", "B", "C"} {
			nodeViz := p.createNodeVisualization(p.nodes[nodeName])
			p.nodeVisualizations.Add(nodeViz)
			p.nodeVisualizations.Add(widget.NewSeparator())
		}
	})
}
