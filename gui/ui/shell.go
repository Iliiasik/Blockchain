package ui

import (
	"Blockchain/resources"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type LEDState int

const (
	LEDIdle LEDState = iota
	LEDBusy
	LEDAlert
	LEDOff
)

type LED struct {
	widget.BaseWidget

	State    LEDState
	Diameter float32

	glow, ring, core *canvas.Circle
	anim             *fyne.Animation
	pulse            float32
}

func NewLED(size float32) *LED {
	l := &LED{Diameter: size}
	l.ExtendBaseWidget(l)
	return l
}

func (l *LED) SetState(state LEDState) {
	if l.State == state {
		return
	}
	l.State = state
	l.Refresh()
}

func (l *LED) colorName() fyne.ThemeColorName {
	switch l.State {
	case LEDBusy:
		return theme.ColorNameWarning
	case LEDAlert:
		return theme.ColorNameError
	case LEDOff:
		return theme.ColorNameDisabled
	case LEDIdle:
		return theme.ColorNameSuccess
	}
	return theme.ColorNameSuccess
}

func (l *LED) CreateRenderer() fyne.WidgetRenderer {
	l.glow = canvas.NewCircle(color.Transparent)
	l.ring = canvas.NewCircle(color.Transparent)
	l.core = canvas.NewCircle(color.Transparent)
	l.anim = fyne.NewAnimation(1100*time.Millisecond, func(v float32) {
		l.pulse = v
		l.glow.FillColor = Tint(l.colorName(), uint8(0x40*v))
		l.glow.Refresh()
	})
	l.anim.AutoReverse = true
	l.anim.RepeatCount = fyne.AnimationRepeatForever
	l.anim.Curve = fyne.AnimationEaseInOut

	r := &ledRenderer{l: l}
	r.Refresh()
	return r
}

type ledRenderer struct {
	l       *LED
	running bool
}

func (r *ledRenderer) Destroy() {
	r.l.anim.Stop()
}

func (r *ledRenderer) Layout(size fyne.Size) {
	d := fyne.Min(size.Width, size.Height)
	off := fyne.NewPos((size.Width-d)/2, (size.Height-d)/2)
	r.l.glow.Move(off)
	r.l.glow.Resize(fyne.NewSquareSize(d))

	ring := d * 0.72
	r.l.ring.Move(off.AddXY((d-ring)/2, (d-ring)/2))
	r.l.ring.Resize(fyne.NewSquareSize(ring))

	core := d * 0.34
	r.l.core.Move(off.AddXY((d-core)/2, (d-core)/2))
	r.l.core.Resize(fyne.NewSquareSize(core))
}

func (r *ledRenderer) MinSize() fyne.Size {
	return fyne.NewSquareSize(r.l.Diameter)
}

func (r *ledRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.l.glow, r.l.ring, r.l.core}
}

func (r *ledRenderer) Refresh() {
	l := r.l
	name := l.colorName()
	l.ring.FillColor = color.Transparent
	l.ring.StrokeColor = Color(name)
	l.ring.StrokeWidth = fyne.Max(1.5, l.Diameter/9)
	l.core.FillColor = Color(name)
	l.glow.FillColor = color.Transparent

	active := l.State == LEDBusy
	if active && !r.running {
		l.anim.Start()
		r.running = true
	} else if !active && r.running {
		l.anim.Stop()
		r.running = false
	}

	l.glow.Refresh()
	l.ring.Refresh()
	l.core.Refresh()
}

type NavItem struct {
	widget.BaseWidget

	Label    string
	Icon     fyne.Resource
	Selected bool
	OnTapped func()

	hovered bool
}

func NewNavItem(label string, icon fyne.Resource, tapped func()) *NavItem {
	n := &NavItem{Label: label, Icon: icon, OnTapped: tapped}
	n.ExtendBaseWidget(n)
	return n
}

func (n *NavItem) SetSelected(selected bool) {
	if n.Selected == selected {
		return
	}
	n.Selected = selected
	n.Refresh()
}

func (n *NavItem) Tapped(*fyne.PointEvent) {
	if n.OnTapped != nil {
		n.OnTapped()
	}
}

func (n *NavItem) MouseIn(*desktop.MouseEvent) {
	n.hovered = true
	n.Refresh()
}

func (n *NavItem) MouseMoved(*desktop.MouseEvent) {}

func (n *NavItem) MouseOut() {
	n.hovered = false
	n.Refresh()
}

func (n *NavItem) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (n *NavItem) CreateRenderer() fyne.WidgetRenderer {
	r := &navRenderer{
		n:     n,
		bg:    canvas.NewRectangle(color.Transparent),
		icon:  Icon(n.Icon, 20),
		label: canvas.NewText(n.Label, nil),
	}
	r.bg.CornerRadius = 8
	r.label.TextSize = 14
	r.label.FontSource = resources.SoraMedium
	r.Refresh()
	return r
}

type navRenderer struct {
	n     *NavItem
	bg    *canvas.Rectangle
	icon  *canvas.Image
	label *canvas.Text
}

func (r *navRenderer) Destroy() {}

func (r *navRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.icon.Move(fyne.NewPos(12, (size.Height-20)/2))
	r.icon.Resize(fyne.NewSquareSize(20))
	label := r.label.MinSize()
	r.label.Move(fyne.NewPos(44, (size.Height-label.Height)/2))
	r.label.Resize(label)
}

func (r *navRenderer) MinSize() fyne.Size {
	return fyne.NewSize(56+r.label.MinSize().Width, 36)
}

func (r *navRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.icon, r.label}
}

func (r *navRenderer) Refresh() {
	n := r.n
	r.label.Text = n.Label
	r.label.Color = Color(resources.ColorNameSecondary)
	r.bg.FillColor = color.Transparent
	r.bg.StrokeColor = color.Transparent
	r.bg.StrokeWidth = 1

	switch {
	case n.Selected:
		r.label.Color = Color(theme.ColorNameForeground)
		r.bg.FillColor = Color(resources.ColorNameRaised)
		r.bg.StrokeColor = Color(resources.ColorNameBorder)
	case n.hovered:
		r.label.Color = Color(theme.ColorNameForeground)
		r.bg.FillColor = Color(theme.ColorNameHover)
	}

	r.bg.Refresh()
	r.label.Refresh()
}

type Toaster struct {
	box *fyne.Container
}

func NewToaster() *Toaster {
	return &Toaster{box: container.NewVBox()}
}

func (t *Toaster) Layer() fyne.CanvasObject {
	return container.NewBorder(nil,
		container.NewHBox(layout.NewSpacer(), Padded(0, 24, 0, 24, FixedWidth(320, t.box))),
		nil, nil,
	)
}

func (t *Toaster) Show(title, message string, colorName fyne.ThemeColorName) {
	body := container.NewVBox(NewBadge(title, colorName))
	if message != "" {
		body.Add(Muted(message))
	}

	plate := canvas.NewRectangle(Color(resources.ColorNameRaised))
	plate.StrokeColor = Color(resources.ColorNameBorder)
	plate.StrokeWidth = 1
	plate.CornerRadius = theme.Size(theme.SizeNameCardRadius)
	card := container.NewStack(plate, Padded(12, 12, 12, 12, body))

	if len(t.box.Objects) >= 3 {
		t.box.Remove(t.box.Objects[0])
	}
	t.box.Add(card)
	t.box.Refresh()

	time.AfterFunc(2500*time.Millisecond, func() {
		fyne.Do(func() {
			t.box.Remove(card)
			t.box.Refresh()
		})
	})
}

type fixedWidth struct {
	width float32
}

func FixedWidth(width float32, obj fyne.CanvasObject) *fyne.Container {
	return container.New(&fixedWidth{width}, obj)
}

func (f *fixedWidth) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}

func (f *fixedWidth) MinSize(objects []fyne.CanvasObject) fyne.Size {
	h := float32(0)
	for _, o := range objects {
		h = fyne.Max(h, o.MinSize().Height)
	}
	return fyne.NewSize(f.width, h)
}
