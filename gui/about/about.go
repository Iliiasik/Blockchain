package about

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

func NewAboutTab() *container.TabItem {
	padded := func(obj fyne.CanvasObject) *fyne.Container {
		return container.New(layout.NewPaddedLayout(), obj)
	}

	title := canvas.NewText("Blockchain demonstration", color.NRGBA{R: 0, G: 120, B: 215, A: 255})
	title.TextSize = 24
	title.TextStyle = fyne.TextStyle{Bold: true}

	header := container.NewHBox(
		title,
	)
	headerSpacer := container.NewVBox(
		padded(header),
		widget.NewSeparator(),
	)

	featuresTitle := widget.NewLabelWithStyle("Key features", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	features := []struct {
		icon  fyne.Resource
		title string
		desc  string
	}{
		{theme.MailSendIcon(), "Wallet generation", "ECDSA keys with secure storage"},
		{theme.StorageIcon(), "Transaction creation", "UTXO model implementation"},
		{theme.ComputerIcon(), "Block mining", "Proof-of-Work simulation"},
		{theme.ListIcon(), "Chain visualization", "Interactive blockchain explorer"},
	}

	featureItems := container.NewVBox()
	for _, f := range features {
		item := container.NewHBox(
			widget.NewIcon(f.icon),
			container.NewVBox(
				widget.NewLabelWithStyle(f.title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(f.desc),
			),
		)
		featureItems.Add(padded(item))
		featureItems.Add(widget.NewSeparator())
	}

	techTitle := widget.NewLabelWithStyle("Technology stack", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	techItems := container.NewVBox(
		padded(container.NewHBox(
			widget.NewIcon(theme.DownloadIcon()),
			widget.NewLabel("Backend: Go (crypto, blockchain logic)"),
		)),
		padded(container.NewHBox(
			widget.NewIcon(theme.ComputerIcon()),
			widget.NewLabel("Frontend: Fyne (cross-platform GUI)"),
		)),
	)

	purposeTitle := widget.NewLabelWithStyle("Project purpose", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	purposeItems := container.NewVBox(
		padded(widget.NewLabel("This is an open-source learning resource.")),
		padded(widget.NewLabel("Feel free to:")),
		padded(container.NewHBox(
			widget.NewIcon(theme.DocumentIcon()),
			widget.NewLabel("Study the code"),
		)),
		padded(container.NewHBox(
			widget.NewIcon(theme.WarningIcon()),
			widget.NewLabel("Report issues"),
		)),
		padded(container.NewHBox(
			widget.NewIcon(theme.MailReplyIcon()),
			widget.NewLabel("Contribute improvements"),
		)),
	)

	content := container.NewVBox(
		padded(headerSpacer),
		padded(featuresTitle),
		padded(featureItems),
		padded(techTitle),
		padded(techItems),
		padded(purposeTitle),
		padded(purposeItems),
	)

	return container.NewTabItemWithIcon(
		"About",
		theme.InfoIcon(),
		container.NewVScroll(
			container.New(layout.NewPaddedLayout(), content),
		),
	)
}
