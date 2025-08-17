package app

import (
	"Blockchain/gui/about"
	"Blockchain/gui/blockchain"
	"Blockchain/gui/p2p_demo"
	"Blockchain/gui/state"
	"Blockchain/gui/transaction"
	"Blockchain/gui/utxo"
	"Blockchain/gui/wallet"
	"Blockchain/gui/wiki"
	"Blockchain/resources"
	"Blockchain/resources/icons"
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

	if a.Preferences().BoolWithFallback("DarkMode", false) {
		a.Settings().SetTheme(&resources.DarkTheme{})
	} else {
		a.Settings().SetTheme(&resources.LightTheme{})
	}

	w := a.NewWindow("Blockchain Demo")
	w.SetIcon(icons.ResourceIconPng)
	w.Resize(fyne.NewSize(800, 650))

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
		fyne.NewMenu("Theme",
			fyne.NewMenuItem("Toggle", func() {
				current := b.app.Settings().Theme()
				switch current.(type) {
				case *resources.DarkTheme:
					b.app.Settings().SetTheme(&resources.LightTheme{})
					b.app.Preferences().SetBool("DarkMode", false)
				default:
					b.app.Settings().SetTheme(&resources.DarkTheme{})
					b.app.Preferences().SetBool("DarkMode", true)
				}
			}),
		),
		fyne.NewMenu("Code",
			fyne.NewMenuItem("GitHub", func() {
				u, _ := url.Parse("https://github.com/Iliiasik/Blockchain")
				_ = b.app.OpenURL(u)
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
		utxo.NewUTXOTab(b.window),
		p2p_demo.NewP2PDemoTab(b.window),
		wiki.NewWikiTab(),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	return tabs
}
