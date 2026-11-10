package app

import (
	"Blockchain/gui/about"
	"Blockchain/gui/blockchain"
	"Blockchain/gui/mining"
	"Blockchain/gui/p2p_demo"
	"Blockchain/gui/state"
	"Blockchain/gui/transaction"
	"Blockchain/gui/ui"
	"Blockchain/gui/utxo"
	"Blockchain/gui/visual"
	"Blockchain/gui/wallet"
	"Blockchain/gui/wiki"
	"Blockchain/resources"
	"Blockchain/resources/icons"
	"fmt"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const appVersion = "2.0.0"

var repoURL = &url.URL{Scheme: "https", Host: "github.com", Path: "/Iliiasik/Blockchain"}

var navGroups = map[int]string{1: "Ledger", 7: "Learn"}

type content interface {
	Content() fyne.CanvasObject
}

type focuser interface {
	Focus(arg string)
}

type activator interface {
	SetActive(active bool)
}

type screen struct {
	id       string
	title    string
	subtitle string
	icon     fyne.Resource
	build    func() content
	view     content
	object   fyne.CanvasObject
	nav      *ui.NavItem
}

type BlockchainApp struct {
	app     fyne.App
	window  fyne.Window
	state   *state.AppState
	theme   *resources.Theme
	screens []*screen
	current *screen
	body    *fyne.Container
	loading fyne.CanvasObject
	title   *ui.Text
	sub     *ui.Text
	led     *ui.LED
	status  *ui.Text
	height  *ui.Text
	mempool *ui.Text
	toaster *ui.Toaster
	mode    *widget.Button
}

func NewBlockchainApp() *BlockchainApp {
	resources.UseEnglish()
	a := app.NewWithID("com.iliiasik.blockchain")
	t := &resources.Theme{Dark: a.Preferences().BoolWithFallback("DarkMode", true)}
	a.Settings().SetTheme(t)

	w := a.NewWindow("Blockchain")
	w.SetIcon(icons.ResourceIconPng)
	w.Resize(fyne.NewSize(1320, 840))

	b := &BlockchainApp{
		app:     a,
		window:  w,
		state:   state.NewAppState(),
		theme:   t,
		toaster: ui.NewToaster(),
	}

	b.state.Navigate = b.navigate
	b.state.Notify = b.notify
	b.state.OnMiningChanged(func(bool) { b.updateStatus(b.state.Snapshot()) })

	return b
}

func (b *BlockchainApp) Run() {
	b.registerScreens()
	b.window.SetContent(b.createContent())
	b.registerShortcuts()
	b.window.SetOnClosed(func() {
		if err := b.state.Close(); err != nil {
			fyne.LogError("Could not close the blockchain database", err)
		}
	})

	b.state.Subscribe(b.updateStatus)
	b.show(b.screens[0])

	go func() {
		if err := b.state.Open(); err != nil {
			fyne.Do(func() {
				b.notify("Database unavailable", err.Error(), state.LevelError)
			})
		}
		b.state.Refresh()
	}()

	b.window.ShowAndRun()
}

func (b *BlockchainApp) registerScreens() {
	st, w := b.state, b.window
	add := func(id, title, subtitle string, icon fyne.Resource, build func() content) {
		b.screens = append(b.screens, &screen{id: id, title: title, subtitle: subtitle, icon: icon, build: build})
	}

	add("overview", "Overview", "System status and a guided path through the demo", icons.ResourceInfoPng,
		func() content { return about.New(st) })
	add("wallets", "Wallets", "ECDSA key pairs, Base58Check addresses and balances", icons.ResourceWalletsPng,
		func() content { return wallet.New(st, w) })
	add("blockchain", "Blockchain", "Create the genesis block and explore the chain", icons.ResourceBlockchainPng,
		func() content { return blockchain.New(st, w) })
	add("visual", "Visualizer", "The whole chain in 3D, follow every coin from block to wallet", icons.ResourceIconPng,
		func() content { return visual.New(st, w) })
	add("mining", "Mining", "Assemble a block and search for a valid nonce", icons.ResourceMiningPng,
		func() content { return mining.New(st) })
	add("transactions", "Transactions", "Sign transfers and watch the mempool", icons.ResourceTransactionsPng,
		func() content { return transaction.New(st) })
	add("utxo", "UTXO set", "Unspent outputs indexed from the chain", icons.ResourceUtxoPng,
		func() content { return utxo.New(st) })
	add("network", "P2P network", "Three isolated nodes, forks and longest chain sync", icons.ResourceP2pNetworkPng,
		func() content { return p2p_demo.New(w) })
	add("wiki", "Wiki", "How every part of this blockchain works, with code", icons.ResourceWikiPng,
		func() content { return wiki.New() })
}

func (b *BlockchainApp) createContent() fyne.CanvasObject {
	b.title = ui.Title("", 26)
	b.sub = ui.Secondary("")
	b.led = ui.NewLED(16)
	b.status = ui.Secondary("")
	b.height = ui.Mono("", resources.ColorNameSecondary)
	b.mempool = ui.Mono("", resources.ColorNameSecondary)

	header := container.NewBorder(nil, nil,
		container.NewVBox(b.title, b.sub),
		container.NewHBox(
			container.NewCenter(b.height),
			ui.Padded(0, 0, 16, 16, container.NewCenter(b.mempool)),
			container.NewCenter(b.led),
			container.NewCenter(b.status),
		),
	)

	b.loading = ui.Padded(0, 0, 0, 0, ui.SkeletonPage())
	b.body = container.NewStack(b.loading)

	main := container.NewBorder(
		ui.Padded(24, 20, 32, 32, header),
		nil, nil, nil,
		ui.Padded(0, 24, 32, 32, b.body),
	)

	return container.NewStack(
		container.NewBorder(nil, nil, b.createSidebar(), nil, main),
		b.toaster.Layer(),
	)
}

func (b *BlockchainApp) createSidebar() fyne.CanvasObject {
	brand := container.NewHBox(
		ui.Icon(icons.ResourceIconPng, 32),
		container.NewCenter(ui.Title("Blockchain", 20)),
	)

	nav := container.New(layout.NewCustomPaddedVBoxLayout(2))
	for i, s := range b.screens {
		s := s
		if label, ok := navGroups[i]; ok {
			nav.Add(ui.Padded(16, 4, 12, 0, ui.Caption(label)))
		}
		s.nav = ui.NewNavItem(s.title, s.icon, func() { b.show(s) })
		nav.Add(s.nav)
	}

	b.mode = ui.Flat("", theme.ColorPaletteIcon(), b.toggleTheme)
	b.updateModeLabel()

	repo := ui.Flat("Source code", theme.ComputerIcon(), func() {
		if err := b.app.OpenURL(repoURL); err != nil {
			b.state.Toast("Could not open the browser", err.Error(), state.LevelError)
		}
	})
	repo.Alignment = widget.ButtonAlignLeading
	b.mode.Alignment = widget.ButtonAlignLeading

	footer := container.NewVBox(
		b.mode,
		repo,
		ui.Padded(8, 0, 12, 0, ui.Caption("Version "+appVersion)),
	)

	bg := canvas.NewRectangle(ui.Color(theme.ColorNameHeaderBackground))
	edge := canvas.NewRectangle(ui.Color(resources.ColorNameBorder))
	edge.SetMinSize(fyne.NewSize(1, 0))

	content := container.NewBorder(
		ui.Padded(24, 16, 16, 16, brand),
		ui.Padded(0, 16, 12, 12, footer),
		nil, nil,
		ui.Padded(0, 0, 12, 12, container.NewVScroll(nav)),
	)

	side := &sidebar{bg: bg, edge: edge}
	side.ExtendBaseWidget(side)
	side.content = content
	return ui.FixedWidth(248, side)
}

type sidebar struct {
	widget.BaseWidget
	bg, edge *canvas.Rectangle
	content  fyne.CanvasObject
}

func (s *sidebar) CreateRenderer() fyne.WidgetRenderer {
	return &sidebarRenderer{s: s}
}

type sidebarRenderer struct {
	s *sidebar
}

func (r *sidebarRenderer) Destroy() {}

func (r *sidebarRenderer) Layout(size fyne.Size) {
	r.s.bg.Resize(size)
	r.s.edge.Move(fyne.NewPos(size.Width-1, 0))
	r.s.edge.Resize(fyne.NewSize(1, size.Height))
	r.s.content.Resize(size)
}

func (r *sidebarRenderer) MinSize() fyne.Size {
	return r.s.content.MinSize()
}

func (r *sidebarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.s.bg, r.s.edge, r.s.content}
}

func (r *sidebarRenderer) Refresh() {
	r.s.bg.FillColor = ui.Color(theme.ColorNameHeaderBackground)
	r.s.edge.FillColor = ui.Color(resources.ColorNameBorder)
	r.s.bg.Refresh()
	r.s.edge.Refresh()
}

func (b *BlockchainApp) show(s *screen) {
	if b.current == s {
		return
	}

	if s.object == nil {
		s.view = s.build()
		s.object = container.NewScroll(s.view.Content())
		b.body.Add(s.object)
	}
	if page, ok := s.object.(fyne.Scrollable); ok {
		ui.SetPage(page)
	}

	for _, other := range b.screens {
		if other.object != nil && other != s {
			other.object.Hide()
		}
		if a, ok := other.view.(activator); ok {
			a.SetActive(other == s)
		}
		if other.nav != nil {
			other.nav.SetSelected(other == s)
		}
	}

	if b.state.Snapshot().Loaded {
		s.object.Show()
	} else {
		s.object.Hide()
	}
	b.current = s
	b.title.SetText(s.title)
	b.sub.SetText(s.subtitle)
}

func (b *BlockchainApp) navigate(id, arg string) {
	for _, s := range b.screens {
		if s.id != id {
			continue
		}
		b.show(s)
		if f, ok := s.view.(focuser); ok && arg != "" {
			f.Focus(arg)
		}
		return
	}
}

func (b *BlockchainApp) notify(title, message string, level state.Level) {
	var color fyne.ThemeColorName
	switch level {
	case state.LevelInfo:
		color = resources.ColorNameAccent
	case state.LevelSuccess:
		color = theme.ColorNameSuccess
	case state.LevelWarning:
		color = theme.ColorNameWarning
	case state.LevelError:
		color = theme.ColorNameError
	}
	b.toaster.Show(title, message, color)
}

func (b *BlockchainApp) updateStatus(snap *state.Snapshot) {
	if b.led == nil {
		return
	}
	if snap.Loaded && b.loading.Visible() {
		b.loading.Hide()
		if b.current != nil {
			b.current.object.Show()
		}
	}

	b.height.SetText(fmt.Sprintf("%d blocks", snap.Height()))
	b.mempool.SetText(fmt.Sprintf("%d/%d pending", len(snap.Mempool), snap.MempoolMax))

	switch {
	case b.state.IsMining():
		b.led.SetState(ui.LEDBusy)
		b.status.SetText("Mining")
	case !snap.HasChain:
		b.led.SetState(ui.LEDOff)
		b.status.SetText("No chain")
	default:
		b.led.SetState(ui.LEDIdle)
		b.status.SetText("Ready")
	}
}

func (b *BlockchainApp) toggleTheme() {
	b.theme = &resources.Theme{Dark: !b.theme.Dark}
	b.app.Settings().SetTheme(b.theme)
	b.app.Preferences().SetBool("DarkMode", b.theme.Dark)
	b.updateModeLabel()
}

func (b *BlockchainApp) updateModeLabel() {
	if b.theme.Dark {
		b.mode.SetText("Light mode")
	} else {
		b.mode.SetText("Dark mode")
	}
}

func (b *BlockchainApp) registerShortcuts() {
	keys := []fyne.KeyName{fyne.Key1, fyne.Key2, fyne.Key3, fyne.Key4, fyne.Key5, fyne.Key6, fyne.Key7, fyne.Key8, fyne.Key9}
	for i, key := range keys {
		if i >= len(b.screens) {
			break
		}
		s := b.screens[i]
		b.window.Canvas().AddShortcut(
			&desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierShortcutDefault},
			func(fyne.Shortcut) { b.show(s) },
		)
	}
}
