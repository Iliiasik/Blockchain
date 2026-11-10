package ui

import (
	"Blockchain/core"
	"Blockchain/resources"
	"Blockchain/resources/icons"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func themedApp(t *testing.T) fyne.App {
	t.Helper()
	a := test.NewTempApp(t)
	a.Settings().SetTheme(&resources.Theme{Dark: true})
	return a
}

func TestNumber(t *testing.T) {
	cases := map[int64]string{0: "0", 7: "7", 999: "999", 1000: "1 000", 65536: "65 536", 1234567: "1 234 567", -4200: "-4 200"}
	for in, want := range cases {
		if got := Number(in); got != want {
			t.Errorf("Number(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestShortening(t *testing.T) {
	if got := ShortString("0123456789abcdef", 4, 2); got != "0123...ef" {
		t.Fatalf("ShortString = %q", got)
	}
	if got := ShortString("short", 4, 2); got != "short" {
		t.Fatalf("short strings must stay intact, got %q", got)
	}
	if got := ShortAddress("1A2B3C4D5E6F7G8H9J"); got != "1A2B3C4D...7G8H9J" {
		t.Fatalf("ShortAddress = %q", got)
	}
	if got := Short([]byte{0xde, 0xad, 0xbe, 0xef, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9}); !strings.HasPrefix(got, "deadbeef00") || !strings.HasSuffix(got, "070809") {
		t.Fatalf("Short = %q", got)
	}
}

func TestDifficultyHelpers(t *testing.T) {
	if got := Target(16); len(got) != 64 || !strings.HasPrefix(got, "00010000") {
		t.Fatalf("Target(16) = %s", got)
	}
	if got := Target(8); !strings.HasPrefix(got, "01") {
		t.Fatalf("Target(8) = %s", got)
	}
	if Attempts(16) != "65 536" || Attempts(0) != "1" {
		t.Fatal("Attempts must be 2 to the power of bits")
	}
}

func TestTimeHelpers(t *testing.T) {
	now := time.Now()
	cases := map[time.Duration]string{
		10 * time.Second: "just now",
		5 * time.Minute:  "5 min ago",
		3 * time.Hour:    "3 h ago",
		50 * time.Hour:   "2 d ago",
	}
	for ago, want := range cases {
		if got := Ago(now.Add(-ago).Unix()); got != want {
			t.Errorf("Ago(-%v) = %q, want %q", ago, got, want)
		}
	}
	stamp := time.Date(2026, 3, 4, 5, 6, 7, 0, time.Local).Unix()
	if got := FormatTime(stamp); got != "2026-03-04 05:06:07" {
		t.Fatalf("FormatTime = %q", got)
	}
}

func TestTextWidget(t *testing.T) {
	themedApp(t)
	text := NewText("hello", 14, theme.ColorNameForeground)
	test.WidgetRenderer(text)
	small := text.MinSize()

	text.SetText("hello world")
	if text.MinSize().Width <= small.Width {
		t.Fatal("longer text must be wider")
	}
	text.SetColor(resources.ColorNameMuted)
	if text.ColorName != resources.ColorNameMuted {
		t.Fatal("color was not applied")
	}
	if Title("x", 20).Display != true || !Strong("x", 12).Bold || !Mono("x", theme.ColorNameForeground).Mono {
		t.Fatal("text presets lost their style")
	}
}

func TestSurfaceAndNavInteraction(t *testing.T) {
	themedApp(t)
	taps := 0
	surface := NewTapSurface(widget.NewLabel("x"), func() { taps++ })
	test.Tap(surface)
	surface.MouseIn(nil)
	if !surface.hovered {
		t.Fatal("tappable surface must react to hover")
	}
	surface.MouseOut()
	surface.SetSelected(true)
	if taps != 1 || !surface.Selected || surface.hovered {
		t.Fatal("surface state is wrong")
	}

	plain := NewSurface(widget.NewLabel("x"))
	plain.MouseIn(nil)
	test.Tap(plain)
	if plain.hovered {
		t.Fatal("a plain surface must not look clickable")
	}
	if NewPanel(nil).Padding != 24 || NewRaised(nil).Padding != 12 || !NewRaised(nil).Raised {
		t.Fatal("surface presets changed")
	}

	navTaps := 0
	nav := NewNavItem("Wallets", icons.ResourceWalletsPng, func() { navTaps++ })
	test.WidgetRenderer(nav)
	test.Tap(nav)
	nav.SetSelected(true)
	if navTaps != 1 || !nav.Selected {
		t.Fatal("nav item did not react")
	}
	if nav.MinSize().Height < 36 {
		t.Fatal("nav item is too small to hit")
	}
}

func TestLEDColors(t *testing.T) {
	themedApp(t)
	led := NewLED(16)
	want := map[LEDState]fyne.ThemeColorName{
		LEDIdle:  theme.ColorNameSuccess,
		LEDBusy:  theme.ColorNameWarning,
		LEDAlert: theme.ColorNameError,
		LEDOff:   theme.ColorNameDisabled,
	}
	test.WidgetRenderer(led)
	for state, color := range want {
		led.SetState(state)
		if led.colorName() != color {
			t.Errorf("state %d shows %s", state, led.colorName())
		}
	}
}

func TestBadgeAndStat(t *testing.T) {
	themedApp(t)
	badge := NewBadge("Coinbase", theme.ColorNameSuccess)
	test.WidgetRenderer(badge)
	width := badge.MinSize().Width
	badge.Set("Transfer pending", theme.ColorNameWarning)
	if badge.MinSize().Width <= width || badge.ColorName != theme.ColorNameWarning {
		t.Fatal("badge did not update")
	}

	stat := NewStat("Balance")
	stat.Set("10", "coins")
	if stat.Value.Text != "10" || stat.Sub.Text != "coins" || stat.Box == nil {
		t.Fatal("stat did not update")
	}
}

func TestToasterKeepsThreeNewest(t *testing.T) {
	themedApp(t)
	toaster := NewToaster()
	if toaster.Layer() == nil {
		t.Fatal("toaster has no layer")
	}
	for i := 0; i < 5; i++ {
		toaster.Show("Title", "message", theme.ColorNamePrimary)
	}
	if len(toaster.box.Objects) != 3 {
		t.Fatalf("toaster shows %d cards, want 3", len(toaster.box.Objects))
	}
}

func TestLayoutsAndIcons(t *testing.T) {
	themedApp(t)
	fixed := FixedWidth(320, widget.NewLabel("x"))
	if fixed.MinSize().Width != 320 {
		t.Fatalf("fixed width = %v", fixed.MinSize().Width)
	}

	img := Icon(icons.ResourceCoinPng, 16)
	if img.MinSize() != fyne.NewSquareSize(16) || img.Image.Bounds().Dx() != 32 {
		t.Fatal("icons must be pre-scaled to twice their size")
	}
	if Icon(icons.ResourceCoinPng, 16).Image != img.Image {
		t.Fatal("scaled icons must be cached")
	}
	if bad := downscale(fyne.NewStaticResource("bad.png", []byte("nope")), 8); bad.Bounds().Dx() != 1 {
		t.Fatal("a broken image must fall back to a blank pixel")
	}
}

func TestHStripPassesVerticalScrollToParent(t *testing.T) {
	themedApp(t)
	content := container.NewHBox()
	for i := 0; i < 40; i++ {
		content.Add(widget.NewLabel("wide column of text"))
	}
	tall := container.NewVBox(widget.NewLabel("x"))
	for i := 0; i < 60; i++ {
		tall.Add(widget.NewLabel("row"))
	}

	parent := container.NewVScroll(tall)
	parent.Resize(fyne.NewSize(300, 200))
	strip := NewHStrip(content)
	strip.Parent = parent
	strip.Resize(fyne.NewSize(300, 80))

	strip.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -40)})
	if parent.Offset.Y == 0 || strip.Offset.X != 0 {
		t.Fatalf("vertical wheel moved strip %v and page %v", strip.Offset, parent.Offset)
	}

	strip.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(-40, 0)})
	if strip.Offset.X == 0 {
		t.Fatal("horizontal wheel must scroll the strip")
	}
}

func TestBlockAndTransactionViews(t *testing.T) {
	themedApp(t)
	wallet := core.NewWallet()
	coinbase := core.NewCoinbaseTX(string(wallet.GetAddress()), "view", 10)
	block, err := core.NewBlock([]*core.Transaction{coinbase}, nil, 8)
	if err != nil {
		t.Fatal(err)
	}

	view := BlockView(block, 0, nil)
	w := test.NewWindow(view)
	defer w.Close()
	w.Resize(fyne.NewSize(800, 600))
	if view.MinSize().Height < 200 {
		t.Fatal("block view is empty")
	}

	transfer := &core.Transaction{
		Vin:  []core.TXInput{{Txid: coinbase.ID, Vout: 0, PubKey: wallet.PublicKey}},
		Vout: []core.TXOutput{*core.NewTXOutput(10, string(core.NewWallet().GetAddress()))},
	}
	if TransactionView(transfer, 1, nil).MinSize().Height <= 0 {
		t.Fatal("transaction view is empty")
	}
	if Section("Title", "subtitle") == nil || Paragraph("text") == nil || Muted("text") == nil || Empty(icons.ResourceCoinPng, "a", "b") == nil {
		t.Fatal("helpers returned nothing")
	}
}

func TestCopyable(t *testing.T) {
	a := themedApp(t)
	copied := ""
	row := Copyable("abc123", func(s string) { copied = s }).(*fyne.Container)
	for _, o := range row.Objects {
		if b, ok := o.(*widget.Button); ok {
			test.Tap(b)
		}
	}
	if copied != "abc123" || a.Clipboard().Content() != "abc123" {
		t.Fatalf("copied %q, clipboard %q", copied, a.Clipboard().Content())
	}
}

type recorder struct {
	widget.BaseWidget
	events int
}

func (r *recorder) Scrolled(*fyne.ScrollEvent) {
	r.events++
}

func (r *recorder) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(widget.NewLabel(""))
}

func tallColumn(rows int) *fyne.Container {
	column := container.NewVBox()
	for i := 0; i < rows; i++ {
		column.Add(widget.NewLabel("row"))
	}
	return column
}

func TestNestedScrollChainsToParent(t *testing.T) {
	themedApp(t)
	parent := &recorder{}
	parent.ExtendBaseWidget(parent)

	short := NewNestedScroll(tallColumn(1))
	short.Parent = parent
	short.Resize(fyne.NewSize(200, 300))
	short.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -30)})
	if parent.events != 1 {
		t.Fatal("content that fits must pass the wheel to the parent")
	}

	long := NewNestedScroll(tallColumn(80))
	long.Parent = parent
	long.Resize(fyne.NewSize(200, 300))
	long.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 30)})
	if parent.events != 2 {
		t.Fatal("scrolling up at the top must reach the parent")
	}
	long.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -30)})
	if parent.events != 2 || long.Offset.Y == 0 {
		t.Fatal("content with room to scroll must scroll itself")
	}
	long.ScrollToBottom()
	long.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -30)})
	if parent.events != 3 {
		t.Fatal("scrolling down at the bottom must reach the parent")
	}

	SetPage(parent)
	defer SetPage(nil)
	orphan := NewNestedScroll(tallColumn(1))
	orphan.Resize(fyne.NewSize(200, 300))
	orphan.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -30)})
	if parent.events != 4 {
		t.Fatal("without an explicit parent the page must receive the wheel")
	}
}

func TestNestedScrollMovesItsBar(t *testing.T) {
	themedApp(t)
	scroll := NewNestedScroll(tallColumn(80))
	w := test.NewWindow(scroll)
	defer w.Close()
	w.Resize(fyne.NewSize(200, 300))

	bar := func() fyne.Position {
		objects := test.WidgetRenderer(scroll).Objects()
		area := objects[len(objects)-2].(fyne.Widget)
		return test.WidgetRenderer(area).Objects()[1].Position()
	}
	before := bar()
	scroll.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -120)})
	if scroll.Offset.Y == 0 || bar().Y <= before.Y {
		t.Fatalf("scroll bar stayed at %v after the content moved to %v", bar(), scroll.Offset)
	}
}

func TestSkeleton(t *testing.T) {
	themedApp(t)
	sk := NewSkeleton(0, 40)
	w := test.NewWindow(container.NewVBox(sk, SkeletonRows(3, 20), SkeletonPage()))
	defer w.Close()
	w.Resize(fyne.NewSize(600, 600))

	if sk.MinSize().Height != 40 {
		t.Fatalf("skeleton height = %v", sk.MinSize().Height)
	}
	if len(pulse.renderers) == 0 {
		t.Fatal("visible skeletons must pulse")
	}
	pulse.tick()
	if pulse.level < 0 || pulse.level > 1 {
		t.Fatalf("pulse level %v out of range", pulse.level)
	}

	w.SetContent(widget.NewLabel("done"))
	before := len(pulse.renderers)
	test.WidgetRenderer(sk).Destroy()
	if len(pulse.renderers) != before-1 {
		t.Fatal("a destroyed skeleton must stop pulsing")
	}
}
