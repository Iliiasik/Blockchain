package visual

import (
	"Blockchain/gui/ui"
	"Blockchain/resources"
	"fmt"
	"image/color"
	"math"
	"sort"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	blockW        = 2.0
	blockH        = 0.5
	blockPitch    = 4.0
	txSize        = 0.46
	txStep        = 0.62
	txCols        = 3
	maxTx         = 9
	visibleBlocks = 30
	framedBlocks  = 7
	poolSide      = 3
	poolPlate     = 2.0
	slotSize      = txSize
	slotStep      = txStep
	walletPitch   = 3.0
	walletPlate   = 2.0
	zoneMargin    = 1.0
	zoneGap       = 2.0
	walletCol     = 0.9
	nearPlane     = 0.25
	fieldOfView   = 50 * math.Pi / 180
	arcSteps      = 32
	flightTime    = 1100 * time.Millisecond
)

var lightDir = vec3{-0.35, 1, 0.55}.unit()

type vec3 struct {
	x, y, z float64
}

func (a vec3) add(b vec3) vec3 {
	return vec3{a.x + b.x, a.y + b.y, a.z + b.z}
}

func (a vec3) sub(b vec3) vec3 {
	return vec3{a.x - b.x, a.y - b.y, a.z - b.z}
}

func (a vec3) mul(k float64) vec3 {
	return vec3{a.x * k, a.y * k, a.z * k}
}

func (a vec3) dot(b vec3) float64 {
	return a.x*b.x + a.y*b.y + a.z*b.z
}

func (a vec3) cross(b vec3) vec3 {
	return vec3{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x}
}

func (a vec3) length() float64 {
	return math.Sqrt(a.dot(a))
}

func (a vec3) unit() vec3 {
	if l := a.length(); l > 0 {
		return a.mul(1 / l)
	}
	return a
}

func lerp3(a, b vec3, t float64) vec3 {
	return a.add(b.sub(a).mul(t))
}

func bezier(a, c, b vec3, t float64) vec3 {
	u := 1 - t
	return a.mul(u * u).add(c.mul(2 * u * t)).add(b.mul(t * t))
}

func arc(a, b vec3, lift float64) []vec3 {
	c := lerp3(a, b, 0.5)
	c.y += lift
	pts := make([]vec3, arcSteps+1)
	for i := range pts {
		pts[i] = bezier(a, c, b, float64(i)/arcSteps)
	}
	return pts
}

func clearLift(solids []solid, a, b vec3, lift, margin float64) float64 {
	const samples = 24
	blocked := func(lift float64) bool {
		c := lerp3(a, b, 0.5)
		c.y += lift
		for i := 1; i < samples; i++ {
			pt := bezier(a, c, b, float64(i)/samples)
			if pt.sub(a).length() < 0.55 || pt.sub(b).length() < 0.55 {
				continue
			}
			for _, sd := range solids {
				if sd.flat || sd.ghost {
					continue
				}
				if pt.x > sd.min.x-margin && pt.x < sd.max.x+margin &&
					pt.y > sd.min.y-margin && pt.y < sd.max.y+margin &&
					pt.z > sd.min.z-margin && pt.z < sd.max.z+margin {
					return true
				}
			}
		}
		return false
	}
	for range 16 {
		if !blocked(lift) {
			break
		}
		lift += 0.45
	}
	return lift
}

func rayHitsBox(from, to, lo, hi vec3) bool {
	d := to.sub(from)
	enter, exit := 0.0, 1.0
	axes := [3][4]float64{
		{from.x, d.x, lo.x, hi.x},
		{from.y, d.y, lo.y, hi.y},
		{from.z, d.z, lo.z, hi.z},
	}
	for _, a := range axes {
		origin, dir, low, high := a[0], a[1], a[2], a[3]
		if math.Abs(dir) < 1e-12 {
			if origin < low || origin > high {
				return false
			}
			continue
		}
		t0, t1 := (low-origin)/dir, (high-origin)/dir
		if t0 > t1 {
			t0, t1 = t1, t0
		}
		enter = math.Max(enter, t0)
		exit = math.Min(exit, t1)
		if enter > exit {
			return false
		}
	}
	return enter < 1-1e-4
}

func straight(a, b vec3, steps int) []vec3 {
	pts := make([]vec3, steps+1)
	for i := range pts {
		pts[i] = lerp3(a, b, float64(i)/float64(steps))
	}
	return pts
}

type camera struct {
	yaw, pitch, dist float64
	target           vec3
}

func (c camera) eye() vec3 {
	cp := math.Cos(c.pitch)
	return c.target.add(vec3{cp * math.Sin(c.yaw), math.Sin(c.pitch), cp * math.Cos(c.yaw)}.mul(c.dist))
}

func blendCamera(a, b camera, t float64) camera {
	turn := math.Remainder(b.yaw-a.yaw, 2*math.Pi)
	return camera{
		yaw:    a.yaw + turn*t,
		pitch:  a.pitch + (b.pitch-a.pitch)*t,
		dist:   a.dist + (b.dist-a.dist)*t,
		target: lerp3(a.target, b.target, t),
	}
}

type view struct {
	eye, right, up, fwd vec3
	cx, cy, focal       float64
}

func (v view) toCam(p vec3) vec3 {
	d := p.sub(v.eye)
	return vec3{d.dot(v.right), d.dot(v.up), d.dot(v.fwd)}
}

func (v view) screen(c vec3) fyne.Position {
	return fyne.NewPos(float32(v.cx+v.focal*c.x/c.z), float32(v.cy-v.focal*c.y/c.z))
}

func (v view) segment(a, b vec3) (fyne.Position, fyne.Position, float64, bool) {
	ca, cb := v.toCam(a), v.toCam(b)
	if ca.z < nearPlane && cb.z < nearPlane {
		return fyne.Position{}, fyne.Position{}, 0, false
	}
	if ca.z < nearPlane {
		ca = lerp3(ca, cb, (nearPlane-ca.z)/(cb.z-ca.z))
	} else if cb.z < nearPlane {
		cb = lerp3(cb, ca, (nearPlane-cb.z)/(ca.z-cb.z))
	}
	return v.screen(ca), v.screen(cb), (ca.z + cb.z) / 2, true
}

func (v view) polygon(pts []vec3) ([]fyne.Position, bool) {
	out := make([]fyne.Position, len(pts))
	for i, p := range pts {
		c := v.toCam(p)
		if c.z < nearPlane {
			return nil, false
		}
		out[i] = v.screen(c)
	}
	return out, true
}

type solid struct {
	min, max vec3
	color    fyne.ThemeColorName
	top      fyne.ThemeColorName
	owner    ref
	group    int
	child    bool
	ghost    bool
	flat     bool
	vanish   float64
}

type face struct {
	pts    [4]vec3
	normal vec3
}

func (s solid) faces() [6]face {
	a, b := s.min, s.max
	return [6]face{
		{[4]vec3{{a.x, b.y, a.z}, {b.x, b.y, a.z}, {b.x, b.y, b.z}, {a.x, b.y, b.z}}, vec3{0, 1, 0}},
		{[4]vec3{{a.x, a.y, a.z}, {a.x, a.y, b.z}, {b.x, a.y, b.z}, {b.x, a.y, a.z}}, vec3{0, -1, 0}},
		{[4]vec3{{a.x, a.y, b.z}, {a.x, b.y, b.z}, {b.x, b.y, b.z}, {b.x, a.y, b.z}}, vec3{0, 0, 1}},
		{[4]vec3{{a.x, a.y, a.z}, {b.x, a.y, a.z}, {b.x, b.y, a.z}, {a.x, b.y, a.z}}, vec3{0, 0, -1}},
		{[4]vec3{{b.x, a.y, a.z}, {b.x, a.y, b.z}, {b.x, b.y, b.z}, {b.x, b.y, a.z}}, vec3{1, 0, 0}},
		{[4]vec3{{a.x, a.y, a.z}, {a.x, b.y, a.z}, {a.x, b.y, b.z}, {a.x, a.y, b.z}}, vec3{-1, 0, 0}},
	}
}

func (f face) center() vec3 {
	return f.pts[0].add(f.pts[2]).mul(0.5)
}

type linkKind int

const (
	linkHash linkKind = iota
	linkSpend
	linkOwn
	linkPending
	linkMine
	linkKinds
)

type link struct {
	kind   linkKind
	pts    []vec3
	color  fyne.ThemeColorName
	width  float32
	dashed bool
	arrow  bool
	a, b   ref
}

type tag struct {
	at    vec3
	text  string
	color fyne.ThemeColorName
	size  float32
	mono  bool
	bold  bool
	align int
	owner ref
	fixed bool
}

type plan struct {
	solids  []solid
	groups  []vec3
	tops    []float64
	members [][]int
	ground  []solid
	grid    [][2]vec3
	links   []link
	tags    []tag
	anchors map[ref]vec3
	ghost   vec3
	nonce   int
	pool    [2]float64
	lo, hi  vec3
	extent  []vec3
}

type wire struct {
	link int
	ends [2]vec3
	lift float64
}

func faceOf(r ref) float64 {
	switch r.kind {
	case refTx:
		return txSize
	case refWallet:
		return walletCol
	case refBlock, refNone:
		return blockW
	}
	return 0
}

func spread(wires []wire, links []link) {
	type end struct{ wire, side int }
	groups := map[ref][]end{}
	for i, w := range wires {
		l := links[w.link]
		groups[l.a] = append(groups[l.a], end{i, 0})
		groups[l.b] = append(groups[l.b], end{i, 1})
	}
	shifts := make([][2]vec3, len(wires))
	for r, list := range groups {
		if len(list) < 2 {
			continue
		}
		other := func(e end) vec3 {
			d := wires[e.wire].ends[1-e.side].sub(wires[e.wire].ends[e.side])
			d.y = 0
			return d
		}
		var mean vec3
		for _, e := range list {
			if d := other(e); d.length() > 1e-9 {
				mean = mean.add(d.unit())
			}
		}
		axis := vec3{1, 0, 0}
		if mean.length() > 1e-9 {
			axis = vec3{-mean.z, 0, mean.x}.unit()
		}
		sort.SliceStable(list, func(i, j int) bool {
			return other(list[i]).dot(axis) < other(list[j]).dot(axis)
		})
		step := math.Min(0.16, faceOf(r)*0.7/float64(len(list)-1))
		for j, e := range list {
			shifts[e.wire][e.side] = axis.mul((float64(j) - float64(len(list)-1)/2) * step)
		}
	}
	for i := range wires {
		wires[i].ends[0] = wires[i].ends[0].add(shifts[i][0])
		wires[i].ends[1] = wires[i].ends[1].add(shifts[i][1])
	}
}

func poolSlot(slots map[string]int, i int, id string) int {
	if slot, ok := slots[id]; ok {
		return slot
	}
	return i
}

func arrange(m *model, mining bool, slots map[string]int) plan {
	p := plan{anchors: map[ref]vec3{}, nonce: -1}
	p.lo = vec3{math.Inf(1), math.Inf(1), math.Inf(1)}
	p.hi = vec3{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	include := func(a, b vec3) {
		for _, x := range []float64{a.x, b.x} {
			for _, y := range []float64{a.y, b.y} {
				for _, z := range []float64{a.z, b.z} {
					p.extent = append(p.extent, vec3{x, y, z})
				}
			}
		}
		p.lo = vec3{math.Min(p.lo.x, a.x), math.Min(p.lo.y, a.y), math.Min(p.lo.z, a.z)}
		p.hi = vec3{math.Max(p.hi.x, b.x), math.Max(p.hi.y, b.y), math.Max(p.hi.z, b.z)}
	}
	group := func(center vec3, top float64) int {
		p.groups = append(p.groups, center)
		p.tops = append(p.tops, top)
		p.members = append(p.members, nil)
		return len(p.groups) - 1
	}
	add := func(s solid) {
		p.members[s.group] = append(p.members[s.group], len(p.solids))
		p.solids = append(p.solids, s)
	}

	count := len(m.blocks)
	first := max(0, count-visibleBlocks)
	xOf := func(h int) float64 {
		return float64(h-first) * blockPitch
	}
	tipX := -blockPitch
	if count > 0 {
		tipX = xOf(count - 1)
	}
	nextX := tipX + blockPitch
	startX := xOf(first)
	p.ghost = vec3{nextX, blockH, 0}

	for h := first; h < count; h++ {
		b := m.blocks[h]
		cx := xOf(h)
		g := group(vec3{cx, blockH / 2, 0}, blockH)
		top := fyne.ThemeColorName("")
		if h == count-1 {
			top = theme.ColorNamePrimary
		}
		add(solid{min: vec3{cx - blockW/2, 0, -blockW / 2}, max: vec3{cx + blockW/2, blockH, blockW / 2}, color: resources.ColorNameRaised, top: top, owner: blockRef(h), group: g})
		p.anchors[blockRef(h)] = vec3{cx, blockH, 0}

		shown := b.txs
		if len(shown) > maxTx {
			shown = shown[:maxTx]
		}
		cols := min(max(len(shown), 1), txCols)
		rows := (len(shown) + cols - 1) / cols
		for i, t := range shown {
			x := cx + (float64(i%cols)-float64(cols-1)/2)*txStep
			z := (float64(i/cols) - float64(rows-1)/2) * txStep
			c := resources.ColorNameAccent
			if t.coinbase {
				c = theme.ColorNameSuccess
			}
			add(solid{min: vec3{x - txSize/2, blockH, z - txSize/2}, max: vec3{x + txSize/2, blockH + txSize, z + txSize/2}, color: c, owner: txRef(t.id), group: g, child: true})
			p.anchors[txRef(t.id)] = vec3{x, blockH + txSize, z}
		}

		label := "#" + strconv.Itoa(h)
		if extra := len(b.txs) - len(shown); extra > 0 {
			label += fmt.Sprintf("  +%d tx", extra)
		}
		p.tags = append(p.tags,
			tag{at: vec3{cx, blockH + txSize + 0.6, 0}, text: label, color: theme.ColorNameForeground, size: 13, bold: true, owner: blockRef(h)},
			tag{at: vec3{cx, 0, blockW/2 + 0.5}, text: ui.ShortString(b.hash, 10, 0), color: resources.ColorNameSecondary, size: 11, mono: true, owner: blockRef(h)},
		)

		if h > first {
			from := vec3{cx - blockW/2, blockH / 2, 0}
			to := vec3{xOf(h-1) + blockW/2, blockH / 2, 0}
			p.links = append(p.links, link{kind: linkHash, pts: []vec3{from, to}, color: resources.ColorNameSecondary, width: 2.5, arrow: true, a: blockRef(h), b: blockRef(h - 1)})
		}
		if h >= count-framedBlocks {
			include(vec3{cx - blockW/2, 0, -blockW / 2}, vec3{cx + blockW/2, blockH + txSize*2, blockW / 2})
		}
	}

	if first > 0 {
		from := vec3{-blockW / 2, blockH / 2, 0}
		to := vec3{-blockPitch + blockW/2, blockH / 2, 0}
		p.links = append(p.links, link{kind: linkHash, pts: straight(from, to, 8), color: resources.ColorNameSecondary, width: 2.5, dashed: true, a: blockRef(first), b: blockRef(first - 1)})
		p.tags = append(p.tags, tag{at: vec3{-blockPitch, blockH / 2, 0}, text: fmt.Sprintf("%d earlier", first), color: resources.ColorNameMuted, size: 11, fixed: true})
	}

	if mining {
		g := group(vec3{nextX, blockH / 2, 0}, blockH)
		add(solid{min: vec3{nextX - blockW/2, 0, -blockW / 2}, max: vec3{nextX + blockW/2, blockH, blockW / 2}, color: theme.ColorNamePrimary, ghost: true, group: g})
		p.nonce = len(p.tags)
		p.tags = append(p.tags, tag{at: vec3{nextX, blockH + 0.9, 0}, text: "Mining", color: theme.ColorNamePrimary, size: 12, mono: true, fixed: true})
		if count > 0 {
			from := vec3{nextX - blockW/2, blockH / 2, 0}
			to := vec3{tipX + blockW/2, blockH / 2, 0}
			p.links = append(p.links, link{kind: linkMine, pts: straight(from, to, 8), color: theme.ColorNamePrimary, width: 2, dashed: true, arrow: true})
		}
		include(vec3{nextX - blockW/2, 0, -blockW / 2}, vec3{nextX + blockW/2, blockH, blockW / 2})
	}

	poolX := startX
	capacity := m.maxPool
	if capacity <= 0 {
		capacity = poolSide * poolSide
	}
	chainEdge := blockW/2 + zoneMargin
	poolW := poolPlate
	poolZ := chainEdge + zoneGap + zoneMargin + poolW/2
	walletZ := poolZ + poolW/2 + 2*zoneMargin + zoneGap + walletPlate/2
	cell := func(slot int) vec3 {
		col, row, layer := slot%poolSide, (slot/poolSide)%poolSide, slot/(poolSide*poolSide)
		return vec3{
			poolX + (float64(col)-float64(poolSide-1)/2)*slotStep,
			0.12 + float64(layer)*slotSize,
			poolZ + (float64(row)-float64(poolSide-1)/2)*slotStep,
		}
	}

	gp := group(vec3{poolX, 0.06, poolZ}, 0.12)
	add(solid{min: vec3{poolX - poolW/2, 0, poolZ - poolW/2}, max: vec3{poolX + poolW/2, 0.12, poolZ + poolW/2}, color: resources.ColorNameRaised, group: gp})
	taken := map[int]bool{}
	for i, t := range m.mempool {
		taken[poolSlot(slots, i, t.id)] = true
	}
	for slot := 0; slot < min(capacity, poolSide*poolSide); slot++ {
		if taken[slot] {
			continue
		}
		c := cell(slot)
		add(solid{min: vec3{c.x - slotSize/2, 0.12, c.z - slotSize/2}, max: vec3{c.x + slotSize/2, 0.135, c.z + slotSize/2}, color: resources.ColorNameBorder, group: gp, child: true})
	}
	for i, t := range m.mempool {
		c := cell(poolSlot(slots, i, t.id))
		x, y, z := c.x, c.y, c.z
		add(solid{min: vec3{x - slotSize/2, y, z - slotSize/2}, max: vec3{x + slotSize/2, y + slotSize, z + slotSize/2}, color: theme.ColorNameWarning, owner: txRef(t.id), group: gp, child: true})
		p.anchors[txRef(t.id)] = vec3{x, y + slotSize, z}
	}
	poolLabel := fmt.Sprintf("Mempool  %d", len(m.mempool))
	if m.maxPool > 0 {
		poolLabel = fmt.Sprintf("Mempool  %d/%d", len(m.mempool), m.maxPool)
	}
	include(vec3{poolX - poolW/2, 0, poolZ - poolW/2}, vec3{poolX + poolW/2 + 3.2, 1, poolZ + poolW/2})
	p.pool = [2]float64{poolW, poolZ}

	maxBalance := 1
	for _, w := range m.wallets {
		maxBalance = max(maxBalance, w.balance)
	}
	n := len(m.wallets)
	walletFrom := startX
	walletTo := walletFrom + float64(max(n-1, 0))*walletPitch
	for i, w := range m.wallets {
		x := walletFrom + float64(i)*walletPitch
		height := 0.15 + 2.4*float64(w.balance)/float64(maxBalance)
		g := group(vec3{x, 0.06, walletZ}, 0.12)
		add(solid{min: vec3{x - walletPlate/2, 0, walletZ - walletPlate/2}, max: vec3{x + walletPlate/2, 0.12, walletZ + walletPlate/2}, color: resources.ColorNameRaised, owner: walletRef(w.address), group: g})
		add(solid{min: vec3{x - walletCol/2, 0.12, walletZ - walletCol/2}, max: vec3{x + walletCol/2, 0.12 + height, walletZ + walletCol/2}, color: resources.ColorNameRaised, top: theme.ColorNameSuccess, owner: walletRef(w.address), group: g, child: true})
		p.anchors[walletRef(w.address)] = vec3{x, 0.12 + height, walletZ}
		p.tags = append(p.tags,
			tag{at: vec3{x, 0.12 + height + 0.45, walletZ}, text: ui.Number(int64(w.balance)), color: theme.ColorNameForeground, size: 13, bold: true, owner: walletRef(w.address)},
			tag{at: vec3{x, 0, walletZ + walletPlate/2 + 0.5}, text: w.name, color: resources.ColorNameSecondary, size: 11, mono: !w.named, owner: walletRef(w.address)},
		)
		include(vec3{x - walletPlate/2, 0, walletZ - walletPlate/2}, vec3{x + walletPlate/2, 0.12 + height, walletZ + walletPlate/2 + 0.6})
	}

	var wires []wire
	route := func(l link, from, to vec3, lift float64) {
		wires = append(wires, wire{link: len(p.links), ends: [2]vec3{from, to}, lift: lift})
		p.links = append(p.links, l)
	}
	if mining {
		for _, t := range m.mempool {
			from := p.anchors[txRef(t.id)]
			route(link{kind: linkMine, color: theme.ColorNamePrimary, width: 1.5, dashed: true, a: txRef(t.id)}, from, p.ghost, 1.2)
		}
	}

	spends := func(t *txn) {
		to, ok := p.anchors[txRef(t.id)]
		if !ok {
			return
		}
		for _, in := range t.inputs {
			from, ok := p.anchors[txRef(in.txid)]
			if !ok {
				continue
			}
			l := link{kind: linkSpend, color: resources.ColorNameAccent, width: 2, arrow: true, a: txRef(in.txid), b: txRef(t.id)}
			if t.pending() {
				l.kind, l.color, l.dashed = linkPending, theme.ColorNameWarning, true
			}
			route(l, from, to, 0.7+0.14*from.sub(to).length())
		}
	}
	for h := first; h < count; h++ {
		for _, t := range m.blocks[h].txs {
			spends(t)
			from, ok := p.anchors[txRef(t.id)]
			if !ok {
				continue
			}
			for _, out := range t.outputs {
				to, ok := p.anchors[walletRef(out.owner)]
				if !ok || out.spentBy != "" {
					continue
				}
				route(link{kind: linkOwn, color: theme.ColorNameSuccess, width: 1.4, a: txRef(t.id), b: walletRef(out.owner)}, from, to, 0.4+0.06*from.sub(to).length())
			}
		}
	}
	for _, t := range m.mempool {
		spends(t)
		from := p.anchors[txRef(t.id)]
		for _, out := range t.outputs {
			to, ok := p.anchors[walletRef(out.owner)]
			if !ok || out.owner == t.sender() {
				continue
			}
			route(link{kind: linkPending, color: theme.ColorNameWarning, width: 1.4, dashed: true, arrow: true, a: txRef(t.id), b: walletRef(out.owner)}, from, to, 0.6)
		}
	}
	spread(wires, p.links)
	for _, w := range wires {
		a, b := w.ends[0], w.ends[1]
		p.links[w.link].pts = arc(a, b, clearLift(p.solids, a, b, w.lift, 0.1))
	}

	chainFrom := xOf(first) - blockW/2 - zoneMargin
	lastX := tipX
	if mining || count == 0 {
		lastX = nextX
	}
	chainTo := lastX + blockW/2 + zoneMargin
	if first > 0 {
		chainFrom -= blockPitch
	}
	zone := func(x0, z0, x1, z1 float64, title string, c fyne.ThemeColorName, right bool) {
		p.ground = append(p.ground, solid{min: vec3{x0, 0, z0}, max: vec3{x1, 0, z1}, color: resources.ColorNameBorder, flat: true})
		t := tag{at: vec3{x0 - 0.4, 0, (z0 + z1) / 2}, text: title, color: c, size: 12, bold: true, align: 2, fixed: true}
		if right {
			t.at.x, t.align = x1+0.4, 1
		}
		p.tags = append(p.tags, t)
	}
	pad := poolW/2 + zoneMargin
	zone(chainFrom, -chainEdge, chainTo, chainEdge, "Blockchain", resources.ColorNameSecondary, false)
	zone(poolX-pad, poolZ-pad, poolX+pad, poolZ+pad, poolLabel, theme.ColorNameWarning, true)
	pad = walletPlate/2 + zoneMargin
	if n > 0 {
		zone(walletFrom-pad, walletZ-pad, walletTo+pad, walletZ+pad, "Wallets", resources.ColorNameSecondary, false)
	}

	gx0 := math.Floor(math.Min(chainFrom, walletFrom-pad)) - 6
	gx1 := math.Ceil(math.Max(chainTo, walletTo+pad)) + 6
	gz0, gz1 := -chainEdge-6, walletZ+pad+6
	for x := gx0; x <= gx1; x++ {
		p.grid = append(p.grid, [2]vec3{{x, 0, gz0}, {x, 0, gz1}})
	}
	for z := gz0; z <= gz1; z++ {
		p.grid = append(p.grid, [2]vec3{{gx0, 0, z}, {gx1, 0, z}})
	}

	return p
}

type opKind int

const (
	opPoly opKind = iota
	opLine
)

type op struct {
	kind   opKind
	depth  float64
	seq    int
	pts    []fyne.Position
	fill   color.Color
	stroke color.Color
	width  float32
	owner  ref
	grow   bool
}

type hitArea struct {
	pts   []fyne.Position
	owner ref
}

type flight struct {
	from, to vec3
	color    fyne.ThemeColorName
	start    time.Time
	lift     float64
	size     float64
	land     ref
}

type tween struct {
	from, to camera
	start    time.Time
	dur      time.Duration
}

type Scene struct {
	widget.BaseWidget

	OnSelect func(ref)

	model    *model
	plan     plan
	cam      camera
	framed   bool
	manual   bool
	size     fyne.Size
	world    *fyne.Container
	mining   bool
	nonce    string
	selected ref
	focus    map[ref]bool
	hover    ref
	hoverAt  fyne.Position
	show     [linkKinds]bool
	spin     bool
	paused   bool

	flights  []flight
	slots    map[string]int
	tween    *tween
	anim     *fyne.Animation
	lastTick time.Time

	dragging  bool
	panning   bool
	rightDown bool
	lastMouse fyne.Position

	ops     []op
	hits    []hitArea
	objects []fyne.CanvasObject
	polys   []*canvas.ArbitraryPolygon
	lines   []*canvas.Line
	texts   []*canvas.Text
	tipBg   *canvas.Rectangle
	tipText *canvas.Text
	sizes   map[string]fyne.Size
	colors  map[fyne.ThemeColorName]color.Color
}

func NewScene() *Scene {
	s := &Scene{
		model:   newModel(),
		cam:     camera{yaw: 0.42, pitch: 0.5, dist: 24},
		world:   container.NewWithoutLayout(),
		tipBg:   canvas.NewRectangle(color.Transparent),
		tipText: canvas.NewText("", color.Black),
		sizes:   map[string]fyne.Size{},
		colors:  map[fyne.ThemeColorName]color.Color{},
	}
	for i := range s.show {
		s.show[i] = true
	}
	s.tipBg.CornerRadius = 6
	s.tipBg.StrokeWidth = 1
	s.tipText.TextSize = 12
	s.plan = arrange(s.model, false, s.slots)
	s.ExtendBaseWidget(s)
	return s
}

func (s *Scene) CreateRenderer() fyne.WidgetRenderer {
	return &sceneRenderer{s: s}
}

func (s *Scene) SetModel(m *model, animate bool) {
	prevModel, prevPlan := s.model, s.plan
	s.model = m
	s.assignSlots()
	s.plan = arrange(m, s.mining, s.slots)

	if s.selected.kind != refNone {
		if s.known(s.selected) {
			s.focus = m.related(s.selected)
		} else {
			s.selected, s.focus = ref{}, nil
			if s.OnSelect != nil {
				s.OnSelect(ref{})
			}
		}
	}
	if !s.framed && s.size.Width > 0 && len(m.blocks) > 0 {
		s.cam, s.framed = s.frame(), true
	}
	if animate && prevModel != nil {
		s.launch(prevModel, prevPlan)
	}
	s.render()
}

func (s *Scene) SetMining(active bool) {
	s.mining = active
	s.plan = arrange(s.model, active, s.slots)
	if !active {
		s.nonce = ""
	}
	s.render()
}

func (s *Scene) SetNonce(text string) {
	s.nonce = text
	s.render()
}

func (s *Scene) Select(r ref) {
	if r == s.selected {
		return
	}
	s.selected = r
	s.focus = nil
	if r.kind != refNone {
		s.focus = s.model.related(r)
	}
	if s.OnSelect != nil {
		s.OnSelect(r)
	}
	s.render()
}

func (s *Scene) selection() ref {
	return s.selected
}

func (s *Scene) SetFocus(refs ...ref) {
	s.selected = ref{}
	s.focus = nil
	if len(refs) > 0 {
		s.focus = map[ref]bool{}
		for _, r := range refs {
			s.focus[r] = true
		}
	}
	s.render()
}

func (s *Scene) SetLinkVisible(kind linkKind, visible bool) {
	s.show[kind] = visible
	s.render()
}

func (s *Scene) SetSpin(on bool) {
	s.spin = on
	if on {
		s.wake()
	}
}

func (s *Scene) SetPaused(paused bool) {
	s.paused = paused
	if paused {
		s.halt()
		return
	}
	if s.spin || len(s.flights) > 0 || s.tween != nil {
		s.wake()
	}
	s.render()
}

func (s *Scene) Stop() {
	s.flights = nil
	s.tween = nil
	s.spin = false
	s.halt()
}

func (s *Scene) frame() camera {
	c := camera{yaw: 0.42, pitch: 0.5, dist: 24}
	if math.IsInf(s.plan.lo.x, 0) {
		return c
	}
	c.target = lerp3(s.plan.lo, s.plan.hi, 0.5)
	c.target.y = 0.6
	radius := s.plan.hi.sub(s.plan.lo).length() / 2
	fov := fieldOfView / 2
	if s.size.Height > 0 && s.size.Width < s.size.Height {
		fov = math.Atan(math.Tan(fov) * float64(s.size.Width/s.size.Height))
	}
	c.dist = math.Max(5, math.Min(140, radius/math.Tan(fov)))
	if s.size.Width <= 0 || s.size.Height <= 0 {
		return c
	}
	corners := s.plan.extent
	saved := s.cam
	defer func() { s.cam = saved }()
	for range 4 {
		s.cam = c
		v := s.makeView()
		extent := 0.0
		for _, p := range corners {
			q := v.toCam(p)
			if q.z < nearPlane {
				extent = 2
				break
			}
			pt := v.screen(q)
			extent = math.Max(extent, math.Abs(float64(pt.X)-v.cx)/(v.cx*0.86))
			extent = math.Max(extent, math.Abs(float64(pt.Y)-v.cy)/(v.cy*0.8))
		}
		if extent <= 0 {
			break
		}
		c.dist = math.Max(5, math.Min(140, c.dist*math.Pow(extent, 0.85)))
	}
	return c
}

func (s *Scene) ResetView() {
	s.LookAt(s.frame(), true)
	s.manual = false
}

func (s *Scene) LookAt(c camera, animated bool) {
	s.framed = true
	s.manual = true
	if !animated {
		s.tween = nil
		s.cam = c
		s.render()
		return
	}
	s.tween = &tween{from: s.cam, to: c, start: time.Now(), dur: 850 * time.Millisecond}
	s.wake()
}

func (s *Scene) anchor(r ref) (vec3, bool) {
	v, ok := s.plan.anchors[r]
	return v, ok
}

func (s *Scene) known(r ref) bool {
	switch r.kind {
	case refNone:
		return false
	case refBlock:
		return r.height >= 0 && r.height < len(s.model.blocks)
	case refTx:
		_, ok := s.model.txs[r.id]
		return ok
	case refWallet:
		return s.model.wallet(r.id) != nil
	}
	return false
}

func (s *Scene) assignSlots() {
	if s.slots == nil {
		s.slots = map[string]int{}
	}
	pending := map[string]bool{}
	for _, t := range s.model.mempool {
		pending[t.id] = true
	}
	used := map[int]bool{}
	for id, slot := range s.slots {
		if !pending[id] {
			delete(s.slots, id)
			continue
		}
		used[slot] = true
	}
	next := 0
	for _, t := range s.model.mempool {
		if _, ok := s.slots[t.id]; ok {
			continue
		}
		for used[next] {
			next++
		}
		s.slots[t.id] = next
		used[next] = true
	}
}

func (s *Scene) launch(prev *model, prevPlan plan) {
	m := s.model
	if len(m.blocks) < len(prev.blocks) {
		return
	}
	center := vec3{0, slotSize / 2, 0}
	above := vec3{0, 0.75, 0}
	for _, t := range m.mempool {
		if _, seen := prev.txs[t.id]; seen {
			continue
		}
		from, ok := s.plan.anchors[walletRef(t.sender())]
		to, ok2 := s.plan.anchors[txRef(t.id)]
		if ok && ok2 {
			end := to.sub(center)
			lift := clearLift(s.plan.solids, from, end, 0.8+0.15*from.sub(end).length(), slotSize/2)
			s.flights = append(s.flights, flight{from: from, to: end, color: theme.ColorNameWarning, start: time.Now(), lift: lift, size: slotSize, land: txRef(t.id)})
			s.wake()
		}
	}
	for h := len(prev.blocks); h < len(m.blocks) && len(prev.blocks) > 0; h++ {
		for i, t := range m.blocks[h].txs {
			to, ok := s.plan.anchors[txRef(t.id)]
			if !ok {
				continue
			}
			delay := time.Duration(i) * 90 * time.Millisecond
			if from, ok := prevPlan.anchors[txRef(t.id)]; ok && !t.coinbase {
				s.fly(from.sub(center), to.add(above), theme.ColorNameWarning, delay)
			}
			c := resources.ColorNameAccent
			if t.coinbase {
				c = theme.ColorNameSuccess
			}
			for _, out := range t.outputs {
				if w, ok := s.plan.anchors[walletRef(out.owner)]; ok && out.owner != t.sender() {
					s.fly(to, w.add(above), c, flightTime+delay+150*time.Millisecond)
				}
			}
		}
	}
}

func (s *Scene) fly(from, to vec3, c fyne.ThemeColorName, delay time.Duration) {
	lift := clearLift(s.plan.solids, from, to, 0.8+0.15*from.sub(to).length(), 0.17)
	s.flights = append(s.flights, flight{from: from, to: to, color: c, start: time.Now().Add(delay), lift: lift, size: 0.34})
	s.wake()
}

func (s *Scene) wake() {
	if s.anim != nil || s.paused {
		return
	}
	s.lastTick = time.Now()
	s.anim = fyne.NewAnimation(time.Second, func(float32) { s.tick() })
	s.anim.Curve = fyne.AnimationLinear
	s.anim.RepeatCount = fyne.AnimationRepeatForever
	s.anim.Start()
}

func (s *Scene) halt() {
	if s.anim != nil {
		s.anim.Stop()
		s.anim = nil
	}
}

func (s *Scene) tick() {
	if s.anim == nil {
		return
	}
	now := time.Now()
	dt := math.Min(now.Sub(s.lastTick).Seconds(), 0.1)
	s.lastTick = now
	busy := false

	if s.tween != nil {
		t := now.Sub(s.tween.start).Seconds() / s.tween.dur.Seconds()
		if t >= 1 {
			s.cam, s.tween = s.tween.to, nil
		} else {
			s.cam = blendCamera(s.tween.from, s.tween.to, easeInOut(t))
			busy = true
		}
	}
	if s.spin {
		busy = true
		if !s.dragging {
			s.cam.yaw += dt * 0.16
		}
	}

	live := s.flights[:0]
	for _, f := range s.flights {
		if now.Sub(f.start) < flightTime {
			live = append(live, f)
		}
	}
	s.flights = live
	busy = busy || len(live) > 0

	s.render()
	if !busy {
		s.halt()
	}
}

func easeInOut(t float64) float64 {
	return t * t * (3 - 2*t)
}

func (s *Scene) color(name fyne.ThemeColorName) color.Color {
	if c, ok := s.colors[name]; ok {
		return c
	}
	c := ui.Color(name)
	s.colors[name] = c
	return c
}

func (s *Scene) makeView() view {
	eye := s.cam.eye()
	fwd := s.cam.target.sub(eye).unit()
	right := fwd.cross(vec3{0, 1, 0}).unit()
	up := right.cross(fwd)
	h := float64(s.size.Height)
	return view{eye: eye, right: right, up: up, fwd: fwd, cx: float64(s.size.Width) / 2, cy: h / 2, focal: h / 2 / math.Tan(fieldOfView/2)}
}

func (s *Scene) emphasis(owner ref) float64 {
	if s.focus == nil || owner.kind == refNone || s.focus[owner] {
		return 1
	}
	return 0.16
}

func (s *Scene) linkAlpha(l link) float64 {
	related := s.focus == nil || (s.focus[l.a] && s.focus[l.b])
	if l.kind == linkMine {
		return 1
	}
	if !s.show[l.kind] && !(l.kind == linkPending && s.show[linkSpend]) {
		if s.focus != nil && related {
			return 1
		}
		return 0
	}
	if !related {
		return 0.07
	}
	if l.kind == linkOwn && s.focus == nil {
		return 0.5
	}
	return 1
}

func (s *Scene) render() {
	if s.size.Width < 2 || s.size.Height < 2 || s.paused {
		return
	}
	clear(s.colors)
	v := s.makeView()
	s.ops = s.ops[:0]
	seq := 0
	push := func(o op) {
		o.seq = seq
		seq++
		s.ops = append(s.ops, o)
	}

	grid := withAlpha(s.color(resources.ColorNameBorder), 0.5)
	for _, g := range s.plan.grid {
		a, b, _, ok := v.segment(g[0], g[1])
		if ok {
			push(op{kind: opLine, depth: math.MaxFloat64, pts: []fyne.Position{a, b}, stroke: grid, width: 1})
		}
	}
	for _, z := range s.plan.ground {
		s.pushSolid(v, z, math.MaxFloat64, push)
	}

	hidden := map[ref]bool{}
	for _, f := range s.flights {
		if f.land.kind != refNone && time.Since(f.start) < flightTime {
			hidden[f.land] = true
		}
	}

	for gi, center := range s.plan.groups {
		depth := v.toCam(center).z
		members := append([]int(nil), s.plan.members[gi]...)
		above := v.eye.y > s.plan.tops[gi]
		sort.SliceStable(members, func(i, j int) bool {
			a, b := s.plan.solids[members[i]], s.plan.solids[members[j]]
			if a.child != b.child {
				return a.child != above
			}
			return v.toCam(lerp3(a.min, a.max, 0.5)).z > v.toCam(lerp3(b.min, b.max, 0.5)).z
		})
		for _, idx := range members {
			if hidden[s.plan.solids[idx].owner] {
				continue
			}
			s.pushSolid(v, s.plan.solids[idx], depth, push)
		}
	}

	blockers := s.occluders(v, hidden)
	const overlay = -1.0
	for _, l := range s.plan.links {
		alpha := s.linkAlpha(l)
		if alpha <= 0 {
			continue
		}
		c := withAlpha(s.color(l.color), alpha)
		width := l.width
		if s.focus != nil && alpha == 1 {
			width *= 1.3
		}
		path := s.trace(v, blockers, l.pts)
		for _, seg := range path.stroke(l.dashed, l.arrow, width) {
			push(op{kind: opLine, depth: overlay, pts: []fyne.Position{seg[0], seg[1]}, stroke: c, width: width})
		}
		if l.arrow {
			if head := path.head(width); head != nil {
				push(op{kind: opPoly, depth: overlay, pts: head, fill: c})
			}
		}
	}

	now := time.Now()
	for _, f := range s.flights {
		t := now.Sub(f.start).Seconds() / flightTime.Seconds()
		if t < 0 || t > 1 {
			continue
		}
		mid := lerp3(f.from, f.to, 0.5)
		mid.y += f.lift
		pos := bezier(f.from, mid, f.to, easeInOut(t))
		size, vanish := f.size, 0.0
		if f.land.kind == refNone && t > 0.65 {
			vanish = (t - 0.65) / 0.35
			size *= 1 - 0.5*vanish
			pos.y += 0.3 * vanish
		}
		half := vec3{size / 2, size / 2, size / 2}
		s.pushSolid(v, solid{min: pos.sub(half), max: pos.add(half), color: f.color, vanish: vanish}, v.toCam(pos).z, push)
	}

	sort.Slice(s.ops, func(i, j int) bool {
		if s.ops[i].depth != s.ops[j].depth {
			return s.ops[i].depth > s.ops[j].depth
		}
		return s.ops[i].seq < s.ops[j].seq
	})

	s.objects = s.objects[:0]
	s.hits = s.hits[:0]
	polys, lines, texts := 0, 0, 0
	w, h := s.size.Width, s.size.Height
	for i := range s.ops {
		o := &s.ops[i]
		switch o.kind {
		case opPoly:
			lo, hi := bounds(o.pts)
			if hi.X < 0 || hi.Y < 0 || lo.X > w || lo.Y > h {
				continue
			}
			if o.grow {
				o.pts = grown(o.pts, 0.6)
				lo, hi = bounds(o.pts)
			}
			poly := s.poly(polys)
			polys++
			poly.Points = poly.Points[:0]
			for _, p := range o.pts {
				poly.Points = append(poly.Points, p.Subtract(lo))
			}
			poly.FillColor = o.fill
			poly.StrokeColor = o.stroke
			poly.StrokeWidth = o.width
			poly.Move(lo)
			poly.Resize(fyne.NewSize(hi.X-lo.X+1, hi.Y-lo.Y+1))
			s.objects = append(s.objects, poly)
			if o.owner.kind != refNone {
				s.hits = append(s.hits, hitArea{pts: o.pts, owner: o.owner})
			}
		case opLine:
			line := s.line(lines)
			lines++
			line.Position1, line.Position2 = o.pts[0], o.pts[1]
			line.StrokeColor = o.stroke
			line.StrokeWidth = o.width
			s.objects = append(s.objects, line)
		}
	}

	type placed struct {
		t     tag
		depth float64
		at    fyne.Position
	}
	var labels []placed
	for i, t := range s.plan.tags {
		c := v.toCam(t.at)
		if c.z < nearPlane {
			continue
		}
		if i == s.plan.nonce && s.nonce != "" {
			t.text = s.nonce
		}
		labels = append(labels, placed{t: t, depth: c.z, at: v.screen(c)})
	}
	sort.SliceStable(labels, func(i, j int) bool { return labels[i].depth > labels[j].depth })
	for _, l := range labels {
		alpha := 1.0
		if !l.t.fixed {
			alpha = s.emphasis(l.t.owner)
			if alpha < 1 {
				alpha = 0.3
			}
		}
		alpha *= math.Max(0.25, math.Min(1, 1.6-l.depth/(s.cam.dist*2.4)))
		text := s.text(texts, l.t.text, withAlpha(s.color(l.t.color), alpha), l.t.size, fyne.TextStyle{Monospace: l.t.mono, Bold: l.t.bold})
		texts++
		size := s.measure(l.t.text, l.t.size, text.TextStyle)
		x := l.at.X - size.Width/2
		switch l.t.align {
		case 1:
			x = l.at.X
		case 2:
			x = l.at.X - size.Width
		}
		text.Move(fyne.NewPos(x, l.at.Y-size.Height/2))
		text.Resize(size)
		s.objects = append(s.objects, text)
	}

	if s.hover.kind != refNone {
		label := s.describe(s.hover)
		s.tipText.TextStyle = fyne.TextStyle{}
		if s.tipText.Text != label {
			s.tipText.Text = label
			s.tipText.Color = s.color(theme.ColorNameForeground)
			s.tipText.Refresh()
		}
		size := s.measure(label, 12, fyne.TextStyle{})
		pos := s.hoverAt.AddXY(16, 18)
		pos.X = fyne.Min(pos.X, w-size.Width-20)
		pos.Y = fyne.Min(pos.Y, h-size.Height-14)
		s.tipBg.FillColor = s.color(theme.ColorNameOverlayBackground)
		s.tipBg.StrokeColor = s.color(resources.ColorNameBorder)
		s.tipBg.Move(pos)
		s.tipBg.Resize(size.AddWidthHeight(16, 10))
		s.tipText.Move(pos.AddXY(8, 5))
		s.tipText.Resize(size)
		s.objects = append(s.objects, s.tipBg, s.tipText)
	}

	s.world.Objects = s.objects
	canvas.Refresh(s.world)
}

type occluder struct {
	lo, hi   vec3
	min, max fyne.Position
}

type occluders []occluder

func (s *Scene) occluders(v view, hidden map[ref]bool) occluders {
	var list occluders
	const inset = 0.004
	for _, sd := range s.plan.solids {
		if sd.flat || sd.ghost || hidden[sd.owner] || s.emphasis(sd.owner) < 1 {
			continue
		}
		o := occluder{lo: sd.min.add(vec3{inset, inset, inset}), hi: sd.max.sub(vec3{inset, inset, inset})}
		first := true
		visible := true
		for _, x := range []float64{sd.min.x, sd.max.x} {
			for _, y := range []float64{sd.min.y, sd.max.y} {
				for _, z := range []float64{sd.min.z, sd.max.z} {
					c := v.toCam(vec3{x, y, z})
					if c.z < nearPlane {
						visible = false
						continue
					}
					p := v.screen(c)
					if first {
						o.min, o.max, first = p, p, false
						continue
					}
					o.min = fyne.NewPos(fyne.Min(o.min.X, p.X), fyne.Min(o.min.Y, p.Y))
					o.max = fyne.NewPos(fyne.Max(o.max.X, p.X), fyne.Max(o.max.Y, p.Y))
				}
			}
		}
		if !visible {
			o.min, o.max = fyne.NewPos(-1e9, -1e9), fyne.NewPos(1e9, 1e9)
		}
		list = append(list, o)
	}
	return list
}

func (list occluders) hide(v view, point vec3) bool {
	c := v.toCam(point)
	if c.z < nearPlane {
		return false
	}
	p := v.screen(c)
	for _, o := range list {
		if p.X < o.min.X || p.X > o.max.X || p.Y < o.min.Y || p.Y > o.max.Y {
			continue
		}
		if rayHitsBox(v.eye, point, o.lo, o.hi) {
			return true
		}
	}
	return false
}

func (s *Scene) edgeShade() float64 {
	r, g, b, _ := s.color(theme.ColorNameBackground).RGBA()
	if (r+g+b)/3 < 0x8000 {
		return 1.35
	}
	return 0.8
}

func (s *Scene) pushSolid(v view, sd solid, depth float64, push func(op)) {
	alpha := s.emphasis(sd.owner)
	alpha *= 1 - sd.vanish
	base := s.color(sd.color)
	var stroke color.Color
	width := float32(1)
	switch {
	case sd.owner.kind != refNone && sd.owner == s.selected:
		stroke, width = s.color(theme.ColorNamePrimary), 2
	case sd.owner.kind != refNone && sd.owner == s.hover:
		stroke, width = withAlpha(s.color(theme.ColorNamePrimary), 0.8), 1.5
	}

	for _, f := range sd.faces() {
		if sd.flat && f.normal.y < 0.5 {
			continue
		}
		visible := f.normal.dot(v.eye.sub(f.center())) > 0
		if sd.ghost {
			pts, ok := v.polygon(f.pts[:])
			if !ok {
				continue
			}
			a := 0.35
			if visible {
				a = 0.9
			}
			fill := color.Color(color.Transparent)
			if visible && f.normal.y > 0.5 {
				fill = withAlpha(base, 0.12)
			}
			push(op{kind: opPoly, depth: depth, pts: pts, fill: fill, stroke: withAlpha(base, a), width: 1.5})
			continue
		}
		if !visible {
			continue
		}
		pts, ok := v.polygon(f.pts[:])
		if !ok {
			continue
		}
		if sd.flat {
			push(op{kind: opPoly, depth: depth, pts: pts, fill: withAlpha(base, 0.3)})
			continue
		}
		fill := base
		if f.normal.y > 0.5 && sd.top != "" {
			fill = s.color(sd.top)
		}
		light := 0.78 + 0.5*math.Max(0, f.normal.dot(lightDir))
		face := shade(fill, light)
		edge := stroke
		if edge == nil {
			edge = withAlpha(shade(face, s.edgeShade()), alpha)
		}
		push(op{kind: opPoly, depth: depth, pts: pts, fill: withAlpha(face, alpha), stroke: edge, width: width, owner: sd.owner, grow: true})
	}
}

func (s *Scene) poly(i int) *canvas.ArbitraryPolygon {
	if i < len(s.polys) {
		return s.polys[i]
	}
	p := canvas.NewArbitraryPolygon(nil, color.Transparent)
	s.polys = append(s.polys, p)
	return p
}

func (s *Scene) line(i int) *canvas.Line {
	if i < len(s.lines) {
		return s.lines[i]
	}
	l := canvas.NewLine(color.Transparent)
	s.lines = append(s.lines, l)
	return l
}

func (s *Scene) text(i int, value string, c color.Color, size float32, style fyne.TextStyle) *canvas.Text {
	if i >= len(s.texts) {
		s.texts = append(s.texts, canvas.NewText("", c))
	}
	t := s.texts[i]
	if t.Text != value || t.Color != c || t.TextSize != size || t.TextStyle != style {
		t.Text, t.Color, t.TextSize, t.TextStyle = value, c, size, style
		t.Refresh()
	}
	return t
}

func (s *Scene) measure(text string, size float32, style fyne.TextStyle) fyne.Size {
	key := fmt.Sprintf("%s|%v|%v|%v", text, size, style.Monospace, style.Bold)
	if v, ok := s.sizes[key]; ok {
		return v
	}
	v := fyne.MeasureText(text, size, style)
	if len(s.sizes) > 4000 {
		clear(s.sizes)
	}
	s.sizes[key] = v
	return v
}

func (s *Scene) describe(r ref) string {
	m := s.model
	switch r.kind {
	case refNone:
		return ""
	case refBlock:
		if b := m.block(r.height); b != nil {
			return fmt.Sprintf("Block #%d, %s", r.height, plural(len(b.txs), "transaction"))
		}
	case refTx:
		t, ok := m.txs[r.id]
		if !ok {
			break
		}
		amount, to := t.transfer()
		switch {
		case t.coinbase:
			return fmt.Sprintf("Coinbase of block #%d: %d new coins for %s", t.block, amount, m.walletName(to))
		case t.pending():
			return fmt.Sprintf("Pending: %d coins from %s to %s", amount, m.walletName(t.sender()), m.walletName(to))
		default:
			return fmt.Sprintf("Transfer in block #%d: %d coins from %s to %s", t.block, amount, m.walletName(t.sender()), m.walletName(to))
		}
	case refWallet:
		if w := m.wallet(r.id); w != nil {
			return fmt.Sprintf("Wallet %s: %d coins", w.name, w.balance)
		}
	}
	return ""
}

func (s *Scene) pick(pos fyne.Position) ref {
	for i := len(s.hits) - 1; i >= 0; i-- {
		if inside(pos, s.hits[i].pts) {
			return s.hits[i].owner
		}
	}
	return ref{}
}

func (s *Scene) Tapped(ev *fyne.PointEvent) {
	r := s.pick(ev.Position)
	if r == s.selected {
		r = ref{}
	}
	s.Select(r)
}

func (s *Scene) MouseDown(ev *desktop.MouseEvent) {
	s.panning = ev.Button == desktop.MouseButtonTertiary || ev.Modifier&fyne.KeyModifierShift != 0
	if ev.Button == desktop.MouseButtonSecondary {
		s.rightDown = true
		s.lastMouse = ev.Position
	}
}

func (s *Scene) MouseUp(ev *desktop.MouseEvent) {
	if ev.Button == desktop.MouseButtonSecondary {
		s.rightDown = false
	}
}

func (s *Scene) MouseIn(ev *desktop.MouseEvent) {
	s.MouseMoved(ev)
}

func (s *Scene) MouseMoved(ev *desktop.MouseEvent) {
	if s.rightDown && ev.Button == desktop.MouseButtonSecondary {
		d := ev.Position.Subtract(s.lastMouse)
		s.lastMouse = ev.Position
		s.panBy(float64(d.X), float64(d.Y))
		s.render()
		return
	}
	s.rightDown = false

	r := s.pick(ev.Position)
	if r == s.hover && (r.kind == refNone || ev.Position == s.hoverAt) {
		return
	}
	s.hover, s.hoverAt = r, ev.Position
	s.render()
}

func (s *Scene) MouseOut() {
	s.rightDown = false
	if s.hover.kind != refNone {
		s.hover = ref{}
		s.render()
	}
}

func (s *Scene) Dragged(ev *fyne.DragEvent) {
	s.tween = nil
	s.manual = true
	s.dragging = true
	if s.hover.kind != refNone {
		s.hover = ref{}
	}
	if s.panning {
		s.panBy(float64(ev.Dragged.DX), float64(ev.Dragged.DY))
	} else {
		s.cam.yaw -= float64(ev.Dragged.DX) * 0.008
		s.cam.pitch = math.Max(0.04, math.Min(1.5, s.cam.pitch+float64(ev.Dragged.DY)*0.008))
	}
	s.render()
}

func (s *Scene) DragEnd() {
	s.dragging = false
	s.panning = false
}

func (s *Scene) panBy(dx, dy float64) {
	s.tween = nil
	s.manual = true
	v := s.makeView()
	scale := s.cam.dist / v.focal
	right := vec3{v.right.x, 0, v.right.z}.unit()
	ahead := vec3{v.fwd.x, 0, v.fwd.z}.unit()
	s.cam.target = s.cam.target.sub(right.mul(dx * scale)).add(ahead.mul(dy * scale))
}

func (s *Scene) Scrolled(ev *fyne.ScrollEvent) {
	s.tween = nil
	s.manual = true
	s.cam.dist = math.Max(3, math.Min(160, s.cam.dist/math.Pow(1.0018, float64(ev.Scrolled.DY))))
	s.render()
}

func (s *Scene) Cursor() desktop.Cursor {
	if s.hover.kind != refNone {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

type sceneRenderer struct {
	s *Scene
}

func (r *sceneRenderer) Destroy() {
	r.s.halt()
}

func (r *sceneRenderer) Layout(size fyne.Size) {
	if size == r.s.size {
		return
	}
	r.s.size = size
	r.s.world.Resize(size)
	if !r.s.manual && len(r.s.model.blocks) > 0 && r.s.tween == nil {
		r.s.cam, r.s.framed = r.s.frame(), true
	}
	r.s.render()
}

func (r *sceneRenderer) MinSize() fyne.Size {
	return fyne.NewSize(240, 240)
}

func (r *sceneRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.s.world}
}

func (r *sceneRenderer) Refresh() {
	r.s.render()
}

type piece struct {
	a, b   fyne.Position
	from   float32
	length float32
	seen   bool
}

type trail struct {
	pieces []piece
	length float32
}

const (
	linkGapStart = 7
	linkGapEnd   = 5
	dashOn       = 7
	dashOff      = 5
)

func (s *Scene) trace(v view, blockers occluders, pts []vec3) trail {
	var t trail
	for i := 0; i+1 < len(pts); i++ {
		start, end := pts[i], pts[i+1]
		steps := max(1, int(math.Ceil(end.sub(start).length()/0.25)))
		for k := 0; k < steps; k++ {
			pa := lerp3(start, end, float64(k)/float64(steps))
			pb := lerp3(start, end, float64(k+1)/float64(steps))
			a, b, _, ok := v.segment(pa, pb)
			if !ok {
				continue
			}
			n := distance(a, b)
			t.pieces = append(t.pieces, piece{a: a, b: b, from: t.length, length: n, seen: !blockers.hide(v, lerp3(pa, pb, 0.5))})
			t.length += n
		}
	}
	return t
}

func distance(a, b fyne.Position) float32 {
	return float32(math.Hypot(float64(b.X-a.X), float64(b.Y-a.Y)))
}

func along(a, b fyne.Position, f float32) fyne.Position {
	return fyne.NewPos(a.X+(b.X-a.X)*f, a.Y+(b.Y-a.Y)*f)
}

func headSize(width float32) float32 {
	return 6 + width*2.2
}

func (t trail) at(d float32) (fyne.Position, bool) {
	for _, p := range t.pieces {
		if d <= p.from+p.length && p.length > 0 {
			return along(p.a, p.b, (d-p.from)/p.length), p.seen
		}
	}
	return fyne.Position{}, false
}

func (t trail) span(arrow bool, width float32) (float32, float32, bool) {
	from, to := float32(linkGapStart), t.length-linkGapEnd
	if arrow {
		to -= headSize(width) * 0.7
	}
	if to-from < 2 {
		from, to = t.length*0.25, t.length*0.75
		if arrow {
			to = t.length - headSize(width)*0.7
		}
	}
	return from, to, to > from
}

func (t trail) stroke(dashed, arrow bool, width float32) [][2]fyne.Position {
	from, to, ok := t.span(arrow, width)
	if !ok {
		return nil
	}
	var out [][2]fyne.Position
	emit := func(p piece, lo, hi float32) {
		if hi-lo < 0.01 || p.length <= 0 {
			return
		}
		out = append(out, [2]fyne.Position{along(p.a, p.b, (lo-p.from)/p.length), along(p.a, p.b, (hi-p.from)/p.length)})
	}
	for _, p := range t.pieces {
		a, b := max(p.from, from), min(p.from+p.length, to)
		if !p.seen || a >= b {
			continue
		}
		if dashed {
			period := float32(dashOn + dashOff)
			for k := float32(math.Floor(float64((a - from) / period))); from+k*period < b; k++ {
				da, db := max(a, from+k*period), min(b, from+k*period+dashOn)
				if da < db {
					emit(p, da, db)
				}
			}
			continue
		}
		emit(p, a, b)
	}
	return merge(out)
}

func merge(segs [][2]fyne.Position) [][2]fyne.Position {
	if len(segs) < 2 {
		return segs
	}
	out := segs[:1]
	for _, s := range segs[1:] {
		last := &out[len(out)-1]
		if distance(last[1], s[0]) < 0.01 && collinear(last[0], last[1], s[1]) {
			last[1] = s[1]
			continue
		}
		out = append(out, s)
	}
	return out
}

func collinear(a, b, c fyne.Position) bool {
	cross := (b.X-a.X)*(c.Y-b.Y) - (b.Y-a.Y)*(c.X-b.X)
	return math.Abs(float64(cross)) < 0.02*float64(distance(a, b)+distance(b, c))
}

func (t trail) head(width float32) []fyne.Position {
	size := headSize(width)
	end := t.length - linkGapEnd
	if end-size < 1 {
		return nil
	}
	tip, seen := t.at(end)
	if !seen {
		return nil
	}
	back, _ := t.at(end - size)
	return arrowHead(back, tip, size)
}

func arrowHead(a, b fyne.Position, size float32) []fyne.Position {
	dx, dy := b.X-a.X, b.Y-a.Y
	l := float32(math.Hypot(float64(dx), float64(dy)))
	if l < 0.5 {
		return nil
	}
	dx, dy = dx/l, dy/l
	back := fyne.NewPos(b.X-dx*size, b.Y-dy*size)
	notch := fyne.NewPos(b.X-dx*size*0.72, b.Y-dy*size*0.72)
	wing := size * 0.48
	return []fyne.Position{
		b,
		fyne.NewPos(back.X-dy*wing, back.Y+dx*wing),
		notch,
		fyne.NewPos(back.X+dy*wing, back.Y-dx*wing),
	}
}

func grown(pts []fyne.Position, by float32) []fyne.Position {
	var cx, cy float32
	for _, p := range pts {
		cx += p.X
		cy += p.Y
	}
	cx /= float32(len(pts))
	cy /= float32(len(pts))
	out := make([]fyne.Position, len(pts))
	for i, p := range pts {
		dx, dy := p.X-cx, p.Y-cy
		l := float32(math.Hypot(float64(dx), float64(dy)))
		if l < 0.001 {
			out[i] = p
			continue
		}
		out[i] = fyne.NewPos(p.X+dx/l*by, p.Y+dy/l*by)
	}
	return out
}

func bounds(pts []fyne.Position) (fyne.Position, fyne.Position) {
	lo, hi := pts[0], pts[0]
	for _, p := range pts[1:] {
		lo = fyne.NewPos(fyne.Min(lo.X, p.X), fyne.Min(lo.Y, p.Y))
		hi = fyne.NewPos(fyne.Max(hi.X, p.X), fyne.Max(hi.Y, p.Y))
	}
	return lo, hi
}

func inside(p fyne.Position, pts []fyne.Position) bool {
	in := false
	for i, j := 0, len(pts)-1; i < len(pts); j, i = i, i+1 {
		a, b := pts[i], pts[j]
		if (a.Y > p.Y) != (b.Y > p.Y) && p.X < (b.X-a.X)*(p.Y-a.Y)/(b.Y-a.Y)+a.X {
			in = !in
		}
	}
	return in
}

func withAlpha(c color.Color, factor float64) color.Color {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	n.A = uint8(math.Max(0, math.Min(255, float64(n.A)*factor)))
	return n
}

func shade(c color.Color, factor float64) color.Color {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	scale := func(v uint8) uint8 {
		return uint8(math.Max(0, math.Min(255, float64(v)*factor)))
	}
	return color.NRGBA{R: scale(n.R), G: scale(n.G), B: scale(n.B), A: n.A}
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
