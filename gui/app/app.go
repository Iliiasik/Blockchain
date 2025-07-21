package app

import (
	"Blockchain/gui/about"
	"Blockchain/gui/blockchain"
	"Blockchain/gui/transaction"
	"Blockchain/gui/wallet"
	"Blockchain/resources"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"net/url"
)

type BlockchainApp struct {
	app    fyne.App
	window fyne.Window
}

func NewBlockchainApp() *BlockchainApp {
	a := app.NewWithID("blockchain.demo")
	w := a.NewWindow("Blockchain Demo")
	w.SetIcon(resources.ResourceIconPng)
	w.Resize(fyne.NewSize(800, 600))

	return &BlockchainApp{
		app:    a,
		window: w,
	}
}

func (b *BlockchainApp) Run() {
	b.window.SetMainMenu(b.createMainMenu())
	b.window.SetContent(b.createContent())
	b.window.ShowAndRun()
}

func (b *BlockchainApp) createMainMenu() *fyne.MainMenu {
	return fyne.NewMainMenu(
		fyne.NewMenu("File"),
		fyne.NewMenu("Theme",
			fyne.NewMenuItem("Light", func() {
				b.app.Settings().SetTheme(theme.LightTheme())
			}),
			fyne.NewMenuItem("Dark", func() {
				b.app.Settings().SetTheme(theme.DarkTheme())
			}),
		),
		fyne.NewMenu("Code",
			fyne.NewMenuItem("GitHub", func() {
				err := fyne.CurrentApp().OpenURL(parseURL("https://github.com/Iliiasik/Blockchain"))
				if err != nil {
					dialog.ShowError(err, b.window)
				}
			}),
		),
	)
}

func (b *BlockchainApp) createContent() fyne.CanvasObject {
	tabs := container.NewAppTabs(
		about.NewAboutTab(),
		wallet.NewWalletTab(b.window),
		blockchain.NewBlockchainTab(b.window),
		transaction.NewTransactionTab(b.window),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	return tabs
}

func (b *BlockchainApp) showError(message string) {
	dialog.ShowError(fmt.Errorf(message), b.window)
}

func parseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	return u
}
