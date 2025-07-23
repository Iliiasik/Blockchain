package blockchain

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"strconv"
	"strings"
)

type BlockchainUI struct {
	window         fyne.Window
	state          *state.AppState
	visualization  *fyne.Container
	blocks         []*core.Block
	loadingSpinner *widget.ProgressBarInfinite
}

func NewBlockchainTab(window fyne.Window, state *state.AppState) *container.TabItem {
	ui := &BlockchainUI{
		window:        window,
		state:         state,
		visualization: container.NewHBox(),
	}
	return ui.createTab()
}

func (b *BlockchainUI) createTab() *container.TabItem {
	createAddress := widget.NewEntry()
	createAddress.SetPlaceHolder("Genesis address")

	subsidyLabel := widget.NewLabel(fmt.Sprintf("Block reward (subsidy): %d", b.state.Subsidy))
	subsidySlider := widget.NewSlider(1, 100)
	subsidySlider.SetValue(float64(b.state.Subsidy))
	subsidySlider.OnChanged = func(val float64) {
		b.state.Subsidy = int(val)
		subsidyLabel.SetText(fmt.Sprintf("Block reward (subsidy): %d", b.state.Subsidy))
	}

	bitsLabel := widget.NewLabel(fmt.Sprintf("Mining difficulty (bits): %d", b.state.TargetBits))
	bitsSlider := widget.NewSlider(8, 28)
	bitsSlider.SetValue(float64(b.state.TargetBits))
	bitsSlider.Step = 1

	bitsSlider.OnChanged = func(val float64) {
		intVal := int(val)
		b.state.TargetBits = intVal
		bitsLabel.SetText(fmt.Sprintf("Mining difficulty (bits): %d", intVal))
	}

	bitsSlider.OnChangeEnded = func(val float64) {
		intVal := int(val)
		if intVal > 20 {
			dialog.ShowInformation("Warning",
				"Mining with difficulty > 20 may take very long time!\nRecommended: 16-18",
				b.window)
		}
	}
	b.loadingSpinner = widget.NewProgressBarInfinite()
	b.loadingSpinner.Hide()

	controls := container.NewVBox(
		widget.NewLabelWithStyle("Blockchain controls", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			widget.NewButtonWithIcon("Create blockchain", theme.ContentAddIcon(), func() {
				b.createBlockchain(createAddress.Text)
			}),
			widget.NewButtonWithIcon("Reindex UTXO", theme.ViewRefreshIcon(), b.onReindexUTXO),
		),
		createAddress,
		subsidyLabel,
		subsidySlider,
		bitsLabel,
		bitsSlider,
		b.loadingSpinner,
		widget.NewButtonWithIcon("View blockchain", theme.VisibilityIcon(), b.updateVisualization),
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

func (b *BlockchainUI) createBlockchain(address string) {
	if address == "" {
		dialog.ShowError(fmt.Errorf("please enter a genesis address"), b.window)
		return
	}

	if !core.ValidateAddress(address) {
		dialog.ShowError(fmt.Errorf("invalid address format"), b.window)
		return
	}

	if b.state.TargetBits > 20 {
		dialog.ShowConfirm("High Difficulty Warning",
			fmt.Sprintf("Mining with difficulty %d may take VERY LONG TIME!\nAre you sure?", b.state.TargetBits),
			func(confirm bool) {
				if confirm {
					b.createBlockchainConfirmed(address)
				}
			}, b.window)
	} else {
		b.createBlockchainConfirmed(address)
	}
}

func (b *BlockchainUI) createBlockchainConfirmed(address string) {
	b.loadingSpinner.Show()

	go func() {
		bc, err := core.CreateBlockchain(address, b.state.Subsidy, b.state.TargetBits)

		fyne.Do(func() {
			b.loadingSpinner.Hide()

			if err != nil {
				dialog.ShowError(err, b.window)
				return
			}

			UTXOSet := core.UTXOSet{bc}
			UTXOSet.Reindex()

			bc.Db.Close()

			dialog.ShowInformation("Blockchain created",
				fmt.Sprintf("New blockchain created!\nDifficulty: %d\nSubsidy: %d",
					b.state.TargetBits, b.state.Subsidy),
				b.window)
		})
	}()
}

func (b *BlockchainUI) showBlockDetails(block *core.Block) {
	header := canvas.NewText(fmt.Sprintf("Block: %x", block.Hash), color.NRGBA{R: 0, G: 102, B: 204, A: 255})
	header.TextSize = 18
	header.TextStyle = fyne.TextStyle{Bold: true}
	header.Alignment = fyne.TextAlignCenter

	blockInfo := container.NewVBox(
		widget.NewLabelWithStyle("Previous hash:", fyne.TextAlignLeading, fyne.TextStyle{Bold: false}),
		widget.NewLabel(fmt.Sprintf("%x", block.PrevBlockHash)),
		widget.NewLabelWithStyle("Transactions count:", fyne.TextAlignLeading, fyne.TextStyle{Bold: false}),
		widget.NewLabel(fmt.Sprintf("%d", len(block.Transactions))),
		widget.NewLabelWithStyle("PoW valid:", fyne.TextAlignLeading, fyne.TextStyle{Bold: false}),
		widget.NewLabel(fmt.Sprintf("%t", core.NewProofOfWork(block, block.Bits).Validate())),
		widget.NewLabelWithStyle("Difficulty (bits):", fyne.TextAlignLeading, fyne.TextStyle{Bold: false}),
		widget.NewLabel(fmt.Sprintf("%d", block.Bits)),
	)

	txAccordion := widget.NewAccordion()
	for i, tx := range block.Transactions {
		txType := "Regular"
		if tx.IsCoinbase() {
			txType = "Coinbase"
		}

		basicInfo := container.NewVBox(
			widget.NewLabelWithStyle("Type:", fyne.TextAlignLeading, fyne.TextStyle{Bold: false}),
			widget.NewLabel(txType),
			widget.NewLabelWithStyle("Inputs count:", fyne.TextAlignLeading, fyne.TextStyle{Bold: false}),
			widget.NewLabel(fmt.Sprintf("%d", len(tx.Vin))),
			widget.NewLabelWithStyle("Outputs count:", fyne.TextAlignLeading, fyne.TextStyle{Bold: false}),
			widget.NewLabel(fmt.Sprintf("%d", len(tx.Vout))),
		)

		var inputsList, outputsList []string

		for _, input := range tx.Vin {
			inputsList = append(inputsList, fmt.Sprintf("TxID: %x\nOutIdx: %d\nSignature: %x\nPubKey: %x\n",
				input.Txid, input.Vout, input.Signature, input.PubKey))
		}

		for _, output := range tx.Vout {
			outputsList = append(outputsList, fmt.Sprintf("Value: %d\nTo: %x\n", output.Value, output.PubKeyHash))
		}

		inputsLabel := widget.NewLabel(strings.Join(inputsList, "\n"))
		inputsLabel.Wrapping = fyne.TextWrapWord

		outputsLabel := widget.NewLabel(strings.Join(outputsList, "\n"))
		outputsLabel.Wrapping = fyne.TextWrapWord

		tabs := container.NewAppTabs(
			container.NewTabItem("Basic", basicInfo),
			container.NewTabItem("Inputs", container.NewScroll(inputsLabel)),
			container.NewTabItem("Outputs", container.NewScroll(outputsLabel)),
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
		blockInfo,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Transactions:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		txAccordion,
	)

	contentWithMaxWidth := container.NewMax(
		container.NewVBox(content),
	)
	contentWithMaxWidth.Resize(fyne.NewSize(800, 450))

	scroll := container.NewScroll(contentWithMaxWidth)
	scroll.SetMinSize(fyne.NewSize(800, 450))

	dialog.ShowCustom(
		"Details",
		"Close",
		scroll,
		b.window,
	)
}

func (b *BlockchainUI) createBlockUI(block *core.Block) fyne.CanvasObject {
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

	card := widget.NewCard("", "", blockContainer)
	card.Resize(fyne.NewSize(120, 80))

	rect := canvas.NewRectangle(baseColor)
	rect.SetMinSize(fyne.NewSize(120, 80))
	rect.CornerRadius = 5

	infoBtn := widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
		b.showBlockDetails(block)
	})
	infoBtn.Importance = widget.LowImportance
	infoBtn.Resize(fyne.NewSize(24, 24))
	infoBtn.Move(fyne.NewPos(85, 18))

	return container.NewStack(
		rect,
		container.NewPadded(card),
		container.NewWithoutLayout(infoBtn),
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

		blockUI := b.createBlockUI(block)
		b.visualization.Add(blockUI)

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
