package ui

import (
	"Blockchain/resources"
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/draw"
)

func Color(name fyne.ThemeColorName) color.Color {
	return theme.Color(name)
}

func Tint(name fyne.ThemeColorName, alpha uint8) color.Color {
	r, g, b, _ := theme.Color(name).RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: alpha}
}

type Text struct {
	widget.BaseWidget

	Text      string
	TextSize  float32
	ColorName fyne.ThemeColorName
	Mono      bool
	Bold      bool
	Display   bool
	Alignment fyne.TextAlign

	text *canvas.Text
}

func NewText(text string, size float32, colorName fyne.ThemeColorName) *Text {
	t := &Text{Text: text, TextSize: size, ColorName: colorName}
	t.ExtendBaseWidget(t)
	return t
}

func Title(text string, size float32) *Text {
	t := NewText(text, size, theme.ColorNameForeground)
	t.Display = true
	return t
}

func Strong(text string, size float32) *Text {
	t := NewText(text, size, theme.ColorNameForeground)
	t.Bold = true
	return t
}

func Caption(text string) *Text {
	return NewText(text, 12, resources.ColorNameMuted)
}

func Secondary(text string) *Text {
	return NewText(text, 13, resources.ColorNameSecondary)
}

func Mono(text string, colorName fyne.ThemeColorName) *Text {
	t := NewText(text, 12, colorName)
	t.Mono = true
	return t
}

func (t *Text) SetText(text string) {
	if t.Text == text {
		return
	}
	t.Text = text
	t.Refresh()
}

func (t *Text) SetColor(name fyne.ThemeColorName) {
	t.ColorName = name
	t.Refresh()
}

func (t *Text) CreateRenderer() fyne.WidgetRenderer {
	t.text = canvas.NewText("", nil)
	r := &textRenderer{t: t}
	r.Refresh()
	return r
}

type textRenderer struct {
	t *Text
}

func (r *textRenderer) Destroy() {}

func (r *textRenderer) Layout(size fyne.Size) {
	r.t.text.Resize(size)
}

func (r *textRenderer) MinSize() fyne.Size {
	return r.t.text.MinSize()
}

func (r *textRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.t.text}
}

func (r *textRenderer) Refresh() {
	txt := r.t.text
	txt.Text = r.t.Text
	txt.Color = Color(r.t.ColorName)
	txt.TextSize = r.t.TextSize
	if txt.TextSize == 0 {
		txt.TextSize = theme.TextSize()
	}
	txt.TextStyle = fyne.TextStyle{Monospace: r.t.Mono, Bold: r.t.Bold}
	txt.FontSource = nil
	if r.t.Display {
		txt.FontSource = resources.SoraSemiBold
	}
	txt.Alignment = r.t.Alignment
	txt.Refresh()
}

type Surface struct {
	widget.BaseWidget

	Content  fyne.CanvasObject
	Raised   bool
	Selected bool
	Padding  float32
	OnTapped func()

	hovered bool
}

func NewSurface(content fyne.CanvasObject) *Surface {
	s := &Surface{Content: content, Padding: 16}
	s.ExtendBaseWidget(s)
	return s
}

func NewPanel(content fyne.CanvasObject) *Surface {
	s := NewSurface(content)
	s.Padding = 24
	return s
}

func NewRaised(content fyne.CanvasObject) *Surface {
	s := NewSurface(content)
	s.Raised = true
	s.Padding = 12
	return s
}

func NewTapSurface(content fyne.CanvasObject, tapped func()) *Surface {
	s := NewRaised(content)
	s.OnTapped = tapped
	return s
}

func (s *Surface) SetSelected(selected bool) {
	if s.Selected == selected {
		return
	}
	s.Selected = selected
	s.Refresh()
}

func (s *Surface) Tapped(*fyne.PointEvent) {
	if s.OnTapped != nil {
		s.OnTapped()
	}
}

func (s *Surface) MouseIn(*desktop.MouseEvent) {
	if s.OnTapped == nil {
		return
	}
	s.hovered = true
	s.Refresh()
}

func (s *Surface) MouseMoved(*desktop.MouseEvent) {}

func (s *Surface) MouseOut() {
	if !s.hovered {
		return
	}
	s.hovered = false
	s.Refresh()
}

func (s *Surface) Cursor() desktop.Cursor {
	if s.OnTapped != nil {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

func (s *Surface) CreateRenderer() fyne.WidgetRenderer {
	r := &surfaceRenderer{s: s, bg: canvas.NewRectangle(color.Transparent)}
	r.Refresh()
	return r
}

type surfaceRenderer struct {
	s  *Surface
	bg *canvas.Rectangle
}

func (r *surfaceRenderer) Destroy() {}

func (r *surfaceRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)

	p := r.s.Padding
	if r.s.Content != nil {
		r.s.Content.Move(fyne.NewPos(p, p))
		r.s.Content.Resize(fyne.NewSize(size.Width-2*p, size.Height-2*p))
	}
}

func (r *surfaceRenderer) MinSize() fyne.Size {
	p := 2 * r.s.Padding
	if r.s.Content == nil {
		return fyne.NewSize(p, p)
	}
	return r.s.Content.MinSize().Add(fyne.NewSize(p, p))
}

func (r *surfaceRenderer) Objects() []fyne.CanvasObject {
	if r.s.Content == nil {
		return []fyne.CanvasObject{r.bg}
	}
	return []fyne.CanvasObject{r.bg, r.s.Content}
}

func (r *surfaceRenderer) Refresh() {
	s := r.s
	r.bg.FillColor = Color(resources.ColorNameSurface)
	r.bg.StrokeColor = Color(resources.ColorNameBorder)
	r.bg.StrokeWidth = 1
	r.bg.CornerRadius = theme.Size(theme.SizeNameCardRadius)

	if s.Raised {
		r.bg.FillColor = Color(resources.ColorNameRaised)
		r.bg.CornerRadius = theme.Size(theme.SizeNameInputRadius)
	}

	switch {
	case s.Selected:
		r.bg.StrokeColor = Color(theme.ColorNamePrimary)
	case s.hovered:
		r.bg.StrokeColor = Tint(theme.ColorNameForeground, 0x33)
	}

	r.bg.Refresh()
}

type Badge struct {
	widget.BaseWidget

	Text      string
	ColorName fyne.ThemeColorName
}

func NewBadge(text string, colorName fyne.ThemeColorName) *Badge {
	b := &Badge{Text: text, ColorName: colorName}
	b.ExtendBaseWidget(b)
	return b
}

func (b *Badge) Set(text string, colorName fyne.ThemeColorName) {
	b.Text = text
	b.ColorName = colorName
	b.Refresh()
}

func (b *Badge) CreateRenderer() fyne.WidgetRenderer {
	label := canvas.NewText("", nil)
	label.TextSize = 12
	r := &badgeRenderer{b: b, dot: canvas.NewCircle(color.Transparent), label: label}
	r.Refresh()
	return r
}

type badgeRenderer struct {
	b     *Badge
	dot   *canvas.Circle
	label *canvas.Text
}

func (r *badgeRenderer) Destroy() {}

func (r *badgeRenderer) Layout(size fyne.Size) {
	label := r.label.MinSize()
	r.dot.Resize(fyne.NewSquareSize(8))
	r.dot.Move(fyne.NewPos(0, (size.Height-8)/2))
	r.label.Move(fyne.NewPos(16, (size.Height-label.Height)/2))
	r.label.Resize(label)
}

func (r *badgeRenderer) MinSize() fyne.Size {
	return r.label.MinSize().Add(fyne.NewSize(16, 0))
}

func (r *badgeRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.dot, r.label}
}

func (r *badgeRenderer) Refresh() {
	r.label.Text = r.b.Text
	r.label.Color = Color(resources.ColorNameSecondary)
	r.dot.FillColor = Color(r.b.ColorName)
	r.label.Refresh()
	r.dot.Refresh()
}

type Stat struct {
	Value *Text
	Sub   *Text
	Box   *Surface
}

func NewStat(label string) *Stat {
	s := &Stat{
		Value: Strong("0", 28),
		Sub:   NewText("", 12, resources.ColorNameMuted),
	}
	s.Box = NewSurface(container.NewVBox(Caption(label), s.Value, s.Sub))
	return s
}

func (s *Stat) Set(value, sub string) {
	s.Value.SetText(value)
	s.Sub.SetText(sub)
}

func Section(title, subtitle string) fyne.CanvasObject {
	head := Title(title, 17)
	if subtitle == "" {
		return head
	}
	return container.NewVBox(head, Muted(subtitle))
}

func Paragraph(text string) fyne.CanvasObject {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return Flush(label)
}

func Muted(text string) fyne.CanvasObject {
	rt := widget.NewRichText(&widget.TextSegment{
		Text:  text,
		Style: widget.RichTextStyle{ColorName: resources.ColorNameSecondary},
	})
	rt.Wrapping = fyne.TextWrapWord
	return Flush(rt)
}

func Flush(obj fyne.CanvasObject) fyne.CanvasObject {
	inset := -theme.Size(theme.SizeNameInnerPadding)
	return Padded(inset, inset, inset, inset, obj)
}

func KV(rows ...fyne.CanvasObject) *fyne.Container {
	return container.New(layout.NewFormLayout(), rows...)
}

func Key(text string) fyne.CanvasObject {
	return container.NewVBox(layout.NewSpacer(), NewText(text, 13, resources.ColorNameMuted), layout.NewSpacer())
}

func Value(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Truncation = fyne.TextTruncateEllipsis
	return label
}

func MonoValue(text string) *widget.Label {
	label := Value(text)
	label.TextStyle = fyne.TextStyle{Monospace: true}
	return label
}

func Copyable(text string, notify func(string)) fyne.CanvasObject {
	label := MonoValue(text)
	btn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		fyne.CurrentApp().Clipboard().SetContent(label.Text)
		if notify != nil {
			notify(label.Text)
		}
	})
	btn.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, nil, btn, label)
}

func Icon(res fyne.Resource, size float32) *canvas.Image {
	img := canvas.NewImageFromImage(downscale(res, int(size*2)))
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleSmooth
	img.SetMinSize(fyne.NewSquareSize(size))
	return img
}

var (
	iconMu    sync.Mutex
	iconCache = map[string]image.Image{}
)

func downscale(res fyne.Resource, px int) image.Image {
	key := fmt.Sprintf("%s@%d", res.Name(), px)

	iconMu.Lock()
	defer iconMu.Unlock()

	if img, ok := iconCache[key]; ok {
		return img
	}

	src, _, err := image.Decode(bytes.NewReader(res.Content()))
	if err != nil {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	dst := image.NewRGBA(image.Rect(0, 0, px, px))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	iconCache[key] = dst

	return dst
}

func Divider() fyne.CanvasObject {
	return widget.NewSeparator()
}

func Button(label string, icon fyne.Resource, tapped func()) *widget.Button {
	return widget.NewButtonWithIcon(label, icon, tapped)
}

func Primary(label string, icon fyne.Resource, tapped func()) *widget.Button {
	btn := widget.NewButtonWithIcon(label, icon, tapped)
	btn.Importance = widget.HighImportance
	return btn
}

func Flat(label string, icon fyne.Resource, tapped func()) *widget.Button {
	btn := widget.NewButtonWithIcon(label, icon, tapped)
	btn.Importance = widget.LowImportance
	return btn
}

func Padded(top, bottom, left, right float32, obj fyne.CanvasObject) fyne.CanvasObject {
	return container.New(layout.NewCustomPaddedLayout(top, bottom, left, right), obj)
}

func Empty(icon fyne.Resource, title, text string) fyne.CanvasObject {
	return container.NewCenter(container.NewVBox(
		container.NewCenter(Icon(icon, 64)),
		Padded(8, 0, 0, 0, container.NewCenter(Title(title, 17))),
		container.NewCenter(NewText(text, 13, resources.ColorNameSecondary)),
	))
}

type Meter struct {
	widget.BaseWidget

	Value     float64
	ColorName fyne.ThemeColorName
}

func NewMeter() *Meter {
	m := &Meter{ColorName: resources.ColorNameAccent}
	m.ExtendBaseWidget(m)
	return m
}

func (m *Meter) SetValue(value float64) {
	m.Value = max(0, min(1, value))
	m.Refresh()
}

func (m *Meter) CreateRenderer() fyne.WidgetRenderer {
	r := &meterRenderer{m: m, track: canvas.NewRectangle(color.Transparent), bar: canvas.NewRectangle(color.Transparent)}
	r.track.CornerRadius = 3
	r.bar.CornerRadius = 3
	r.Refresh()
	return r
}

type meterRenderer struct {
	m          *Meter
	track, bar *canvas.Rectangle
	size       fyne.Size
}

func (r *meterRenderer) Destroy() {}

func (r *meterRenderer) Layout(size fyne.Size) {
	r.size = size
	r.track.Resize(size)
	r.bar.Resize(fyne.NewSize(size.Width*float32(r.m.Value), size.Height))
}

func (r *meterRenderer) MinSize() fyne.Size {
	return fyne.NewSize(40, 6)
}

func (r *meterRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.track, r.bar}
}

func (r *meterRenderer) Refresh() {
	r.track.FillColor = Color(resources.ColorNameBorder)
	r.bar.FillColor = Color(r.m.ColorName)
	r.Layout(r.size)
	r.track.Refresh()
	r.bar.Refresh()
}

type HStrip struct {
	*container.Scroll
	Parent fyne.Scrollable
}

func NewHStrip(content fyne.CanvasObject) *HStrip {
	h := &HStrip{Scroll: &container.Scroll{Direction: container.ScrollHorizontalOnly, Content: content}}
	h.ExtendBaseWidget(h)
	return h
}

func (h *HStrip) Scrolled(ev *fyne.ScrollEvent) {
	vertical := ev.Scrolled.DX == 0 || abs(ev.Scrolled.DY) > abs(ev.Scrolled.DX)
	if parent := parentOf(h.Parent); vertical && parent != nil {
		parent.Scrolled(ev)
		return
	}
	h.Scroll.Scrolled(ev)
}

var page fyne.Scrollable

func SetPage(scroll fyne.Scrollable) {
	page = scroll
}

func parentOf(explicit fyne.Scrollable) fyne.Scrollable {
	if explicit != nil {
		return explicit
	}
	return page
}

type NestedScroll struct {
	*container.Scroll
	Parent fyne.Scrollable
}

func NewNestedScroll(content fyne.CanvasObject) *NestedScroll {
	n := &NestedScroll{Scroll: &container.Scroll{Direction: container.ScrollVerticalOnly, Content: content}}
	n.ExtendBaseWidget(n)
	return n
}

func (n *NestedScroll) Scrolled(ev *fyne.ScrollEvent) {
	parent := parentOf(n.Parent)
	if parent == nil || n.Content == nil {
		n.Scroll.Scrolled(ev)
		return
	}

	overflow := n.Content.MinSize().Height - n.Size().Height
	up := ev.Scrolled.DY > 0
	atTop := n.Offset.Y <= 0
	atBottom := n.Offset.Y >= overflow-0.5
	if overflow <= 0 || (up && atTop) || (!up && atBottom) {
		parent.Scrolled(ev)
		return
	}
	n.Scroll.Scrolled(ev)
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
