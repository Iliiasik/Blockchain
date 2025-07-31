package blockchain

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"Blockchain/resources/icons"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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
		widget.NewButtonWithIcon("Create blockchain", theme.ContentAddIcon(), func() {
			b.createBlockchain(createAddress.Text)
		}),

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

	return container.NewTabItemWithIcon("Blockchain",
		icons.ResourceBlockchainPng,
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
