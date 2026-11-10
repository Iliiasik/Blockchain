package ui

import (
	"Blockchain/resources"
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type Skeleton struct {
	widget.BaseWidget
	Width, Height float32
}

func NewSkeleton(width, height float32) *Skeleton {
	s := &Skeleton{Width: width, Height: height}
	s.ExtendBaseWidget(s)
	return s
}

func (s *Skeleton) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(Color(resources.ColorNameRaised))
	rect.CornerRadius = 6
	r := &skeletonRenderer{s: s, rect: rect}
	pulse.add(r)
	return r
}

type skeletonRenderer struct {
	s    *Skeleton
	rect *canvas.Rectangle
}

func (r *skeletonRenderer) Layout(size fyne.Size) {
	r.rect.Resize(size)
}

func (r *skeletonRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.s.Width, r.s.Height)
}

func (r *skeletonRenderer) Refresh() {
	r.paint(pulse.level)
}

func (r *skeletonRenderer) paint(level float64) {
	base := Color(resources.ColorNameRaised)
	edge := Color(resources.ColorNameBorder)
	r.rect.FillColor = mix(base, edge, level)
	r.rect.Refresh()
}

func (r *skeletonRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.rect}
}

func (r *skeletonRenderer) Destroy() {
	pulse.remove(r)
}

type pulser struct {
	renderers map[*skeletonRenderer]bool
	anim      *fyne.Animation
	start     time.Time
	level     float64
}

var pulse = &pulser{renderers: map[*skeletonRenderer]bool{}}

func (p *pulser) add(r *skeletonRenderer) {
	p.renderers[r] = true
	if p.anim != nil {
		return
	}
	p.start = time.Now()
	p.anim = fyne.NewAnimation(time.Second, func(float32) { p.tick() })
	p.anim.Curve = fyne.AnimationLinear
	p.anim.RepeatCount = fyne.AnimationRepeatForever
	p.anim.Start()
}

func (p *pulser) remove(r *skeletonRenderer) {
	delete(p.renderers, r)
}

func (p *pulser) tick() {
	p.level = (1 - math.Cos(time.Since(p.start).Seconds()*math.Pi)) / 2
	driver := fyne.CurrentApp().Driver()
	for r := range p.renderers {
		if driver.CanvasForObject(r.s) == nil {
			delete(p.renderers, r)
			continue
		}
		if r.s.Visible() {
			r.paint(p.level)
		}
	}
	if len(p.renderers) == 0 && p.anim != nil {
		p.anim.Stop()
		p.anim = nil
	}
}

func mix(a, b color.Color, t float64) color.Color {
	x := color.NRGBAModel.Convert(a).(color.NRGBA)
	y := color.NRGBAModel.Convert(b).(color.NRGBA)
	lerp := func(p, q uint8) uint8 {
		return uint8(math.Round(float64(p) + (float64(q)-float64(p))*t))
	}
	return color.NRGBA{R: lerp(x.R, y.R), G: lerp(x.G, y.G), B: lerp(x.B, y.B), A: lerp(x.A, y.A)}
}

func SkeletonRows(count int, height float32) fyne.CanvasObject {
	rows := container.New(layout.NewCustomPaddedVBoxLayout(8))
	for i := 0; i < count; i++ {
		rows.Add(NewSkeleton(0, height))
	}
	return rows
}

func SkeletonPage() fyne.CanvasObject {
	stats := container.NewGridWithColumns(3, NewSkeleton(0, 96), NewSkeleton(0, 96), NewSkeleton(0, 96))
	return container.New(layout.NewCustomPaddedVBoxLayout(16),
		NewSkeleton(0, 72),
		stats,
		SkeletonRows(4, 56),
	)
}
