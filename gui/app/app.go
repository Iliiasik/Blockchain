package app

import (
	"Blockchain/gui/about"
	"Blockchain/gui/blockchain"
	"Blockchain/gui/p2p_demo"
	"Blockchain/gui/state"
	"Blockchain/gui/transaction"
	"Blockchain/gui/wallet"
	"Blockchain/resources"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"net/url"
)

type BlockchainApp struct {
	app    fyne.App
	window fyne.Window
	state  *state.AppState
}

func NewBlockchainApp() *BlockchainApp {
	a := app.NewWithID("blockchain.demo")
	a.Settings().SetTheme(&resources.CustomTheme{})
	w := a.NewWindow("Blockchain Demo")
	w.SetIcon(resources.ResourceIconPng)
	w.Resize(fyne.NewSize(800, 600))

	return &BlockchainApp{
		app:    a,
		window: w,
		state:  state.NewAppState(),
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
		fyne.NewMenu("Code",
			fyne.NewMenuItem("GitHub", func() {
				u, _ := url.Parse("https://github.com/Iliiasik/Blockchain")
				_ = fyne.CurrentApp().OpenURL(u)
			}),
		),
	)
}

func (b *BlockchainApp) createContent() fyne.CanvasObject {
	tabs := container.NewAppTabs(
		about.NewAboutTab(),
		wallet.NewWalletTab(b.window, b.state),
		blockchain.NewBlockchainTab(b.window, b.state),
		transaction.NewTransactionTab(b.window, b.state),
		p2p_demo.NewP2PDemoTab(b.window, b.state),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	return tabs
}
